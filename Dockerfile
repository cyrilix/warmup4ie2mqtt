FROM golang:alpine as builder

WORKDIR /go/src
ADD . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod vendor -tags netgo -o /go/bin/warmup4ie2zwave




FROM scratch

USER 1234
COPY --from=builder /go/bin/warmup4ie2zwave /go/bin/warmup4ie2zwave
ENTRYPOINT ["/go/bin/warmup4ie2zwave"]
