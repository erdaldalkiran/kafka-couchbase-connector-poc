package gocb

import (
	"context"
	"errors"
	"time"

	gocbcore "github.com/couchbase/gocbcore/v10"
	"github.com/couchbase/gocbcore/v10/memd"
)

type kvOpManager struct {
	parent *Collection
	signal chan struct{}

	err           error
	wasResolved   bool
	mutationToken *MutationToken

	span            RequestSpan
	documentID      string
	transcoder      Transcoder
	timeout         time.Duration
	deadline        time.Time
	bytes           []byte
	flags           uint32
	persistTo       uint
	replicateTo     uint
	durabilityLevel memd.DurabilityLevel
	retryStrategy   *retryStrategyWrapper
	cancelCh        chan struct{}
	impersonate     string

	operationName string
	createdTime   time.Time
	meter         *meterWrapper
	preserveTTL   bool

	ctx context.Context
}

func (m *kvOpManager) getTimeout() time.Duration {
	if m.timeout > 0 {
		if m.durabilityLevel > 0 && m.timeout < durabilityTimeoutFloor {
			m.timeout = durabilityTimeoutFloor
			logWarnf("Durable operation in use so timeout value coerced up to %s", m.timeout.String())
		}
		return m.timeout
	}

	defaultTimeout := m.parent.timeoutsConfig.KVTimeout
	if m.durabilityLevel > memd.DurabilityLevelMajority || m.persistTo > 0 {
		defaultTimeout = m.parent.timeoutsConfig.KVDurableTimeout
	}

	if m.durabilityLevel > 0 && defaultTimeout < durabilityTimeoutFloor {
		defaultTimeout = durabilityTimeoutFloor
		logWarnf("Durable operation in user so timeout value coerced up to %s", defaultTimeout.String())
	}

	return defaultTimeout
}

func (m *kvOpManager) SetDocumentID(id string) {
	m.documentID = id
}

func (m *kvOpManager) SetCancelCh(cancelCh chan struct{}) {
	m.cancelCh = cancelCh
}

func (m *kvOpManager) SetTimeout(timeout time.Duration) {
	m.timeout = timeout
}

func (m *kvOpManager) SetTranscoder(transcoder Transcoder) {
	if transcoder == nil {
		transcoder = m.parent.transcoder
	}
	m.transcoder = transcoder
}

func (m *kvOpManager) SetValue(val interface{}) {
	if m.err != nil {
		return
	}
	if m.transcoder == nil {
		m.err = errors.New("Expected a transcoder to be specified first")
		return
	}

	espan := m.parent.startKvOpTrace("request_encoding", m.span.Context(), true)
	defer espan.End()

	bytes, flags, err := m.transcoder.Encode(val)
	if err != nil {
		m.err = err
		return
	}

	m.bytes = bytes
	m.flags = flags
}

func (m *kvOpManager) SetDuraOptions(persistTo, replicateTo uint, level DurabilityLevel) {
	if persistTo != 0 || replicateTo != 0 {
		if !m.parent.useMutationTokens {
			m.err = makeInvalidArgumentsError("cannot use observe based durability without mutation tokens")
			return
		}

		if level > 0 {
			m.err = makeInvalidArgumentsError("cannot mix observe based durability and synchronous durability")
			return
		}
	}

	if level == DurabilityLevelUnknown {
		level = DurabilityLevelNone
	}

	m.persistTo = persistTo
	m.replicateTo = replicateTo
	m.durabilityLevel, m.err = level.toMemd()

	if level > DurabilityLevelNone {
		levelStr, err := level.toManagementAPI()
		if err != nil {
			logDebugf("Could not convert durability level to string: %v", err)
			return
		}
		m.span.SetAttribute(spanAttribDBDurability, levelStr)
	}
}

func (m *kvOpManager) SetRetryStrategy(retryStrategy RetryStrategy) {
	wrapper := m.parent.retryStrategyWrapper
	if retryStrategy != nil {
		wrapper = newRetryStrategyWrapper(retryStrategy)
	}
	m.retryStrategy = wrapper
}

func (m *kvOpManager) SetImpersonate(user string) {
	m.impersonate = user
}

func (m *kvOpManager) SetContext(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	m.ctx = ctx
}

func (m *kvOpManager) SetPreserveExpiry(preserveTTL bool) {
	m.preserveTTL = preserveTTL
}

func (m *kvOpManager) Finish(noMetrics bool) {
	m.span.End()

	if !noMetrics {
		m.meter.ValueRecord(meterValueServiceKV, m.operationName, m.createdTime)
	}
}

func (m *kvOpManager) TraceSpanContext() RequestSpanContext {
	return m.span.Context()
}

func (m *kvOpManager) TraceSpan() RequestSpan {
	return m.span
}

func (m *kvOpManager) DocumentID() []byte {
	return []byte(m.documentID)
}

func (m *kvOpManager) CollectionName() string {
	return m.parent.name()
}

func (m *kvOpManager) ScopeName() string {
	return m.parent.ScopeName()
}

func (m *kvOpManager) BucketName() string {
	return m.parent.bucketName()
}

func (m *kvOpManager) ValueBytes() []byte {
	return m.bytes
}

func (m *kvOpManager) ValueFlags() uint32 {
	return m.flags
}

func (m *kvOpManager) Transcoder() Transcoder {
	return m.transcoder
}

func (m *kvOpManager) DurabilityLevel() memd.DurabilityLevel {
	return m.durabilityLevel
}

func (m *kvOpManager) DurabilityTimeout() time.Duration {
	if m.durabilityLevel == 0 {
		return 0
	}

	timeout := m.getTimeout()

	duraTimeout := time.Duration(float64(timeout) * 0.9)

	if duraTimeout < durabilityTimeoutFloor {
		duraTimeout = durabilityTimeoutFloor
	}

	return duraTimeout
}

func (m *kvOpManager) Deadline() time.Time {
	if m.deadline.IsZero() {
		timeout := m.getTimeout()
		m.deadline = time.Now().Add(timeout)
	}

	return m.deadline
}

func (m *kvOpManager) RetryStrategy() *retryStrategyWrapper {
	return m.retryStrategy
}

func (m *kvOpManager) Impersonate() string {
	return m.impersonate
}

func (m *kvOpManager) PreserveExpiry() bool {
	return m.preserveTTL
}

func (m *kvOpManager) CheckReadyForOp() error {
	if m.err != nil {
		return m.err
	}

	if m.getTimeout() == 0 {
		return errors.New("op manager had no timeout specified")
	}

	return nil
}

func (m *kvOpManager) NeedsObserve() bool {
	return m.persistTo > 0 || m.replicateTo > 0
}

func (m *kvOpManager) EnhanceErr(err error) error {
	return maybeEnhanceCollKVErr(err, nil, m.parent, m.documentID)
}

func (m *kvOpManager) EnhanceMt(token gocbcore.MutationToken) *MutationToken {
	if token.VbUUID != 0 {
		return &MutationToken{
			token:      token,
			bucketName: m.BucketName(),
		}
	}

	return nil
}

func (m *kvOpManager) Reject() {
	m.signal <- struct{}{}
}

func (m *kvOpManager) Resolve(token *MutationToken) {
	m.wasResolved = true
	m.mutationToken = token
	m.signal <- struct{}{}
}

func (m *kvOpManager) Wait(op gocbcore.PendingOp, err error) error {
	if err != nil {
		return err
	}
	if m.err != nil {
		op.Cancel()
	}

	select {
	case <-m.signal:
		// Good to go
	case <-m.cancelCh:
		op.Cancel()
		<-m.signal
	case <-m.ctx.Done():
		op.Cancel()
		<-m.signal
	}

	if m.wasResolved && (m.persistTo > 0 || m.replicateTo > 0) {
		if m.mutationToken == nil {
			return errors.New("expected a mutation token")
		}

		return m.parent.waitForDurability(
			m.ctx,
			m.span,
			m.documentID,
			m.mutationToken.token,
			m.replicateTo,
			m.persistTo,
			m.Deadline(),
			m.cancelCh,
			m.impersonate,
		)
	}

	return nil
}

func (c *Collection) newKvOpManager(opName string, parentSpan RequestSpan) *kvOpManager {
	var tracectx RequestSpanContext
	if parentSpan != nil {
		tracectx = parentSpan.Context()
	}

	span := c.startKvOpTrace(opName, tracectx, false)

	return &kvOpManager{
		parent:        c,
		signal:        make(chan struct{}, 1),
		span:          span,
		operationName: opName,
		createdTime:   time.Now(),
		meter:         c.meter,
	}
}

func durationToExpiry(dura time.Duration) uint32 {
	// If the duration is 0, that indicates never-expires
	if dura == 0 {
		return 0
	}

	// If the duration is less than one second, we must force the
	// value to 1 to avoid accidentally making it never expire.
	if dura < 1*time.Second {
		return 1
	}

	if dura < 30*24*time.Hour {
		// Translate into a uint32 in seconds.
		return uint32(dura / time.Second)
	}

	// Send the duration as a unix timestamp of now plus duration.
	return uint32(time.Now().Add(dura).Unix())
}
