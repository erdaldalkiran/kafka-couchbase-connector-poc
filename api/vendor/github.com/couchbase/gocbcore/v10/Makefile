devsetup:
	go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.39.0
	go get github.com/vektra/mockery/.../

test:
	go test ./...
fasttest:
	go test -short ./...

cover:
	go test -coverprofile=cover.out ./...

lint:
	golangci-lint run -v

check: lint
	go test -cover -race ./...

bench:
	go test -run ^$$ -bench . --disable-logger

updatemocks:
	mockery -name dispatcher -output . -testonly -inpkg
	mockery -name configManager -output . -testonly -inpkg
	mockery -name httpComponentInterface -output . -testonly -inpkg

.PHONY: all test devsetup fasttest lint cover checkerrs checkfmt checkvet checkiea checkspell check bench updatemocks
