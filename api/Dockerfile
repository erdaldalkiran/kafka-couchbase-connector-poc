FROM golang:1.18.0-alpine3.15 as build-image
RUN apk update && apk add upx
RUN apk add -U --no-cache ca-certificates
WORKDIR /build-dir
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -v -ldflags "-s -w -extldflags '-static'" -o demo
RUN upx ./demo

FROM scratch
COPY --from=build-image /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-image /build-dir/demo /build-dir/demo
ENTRYPOINT ["/build-dir/demo"]