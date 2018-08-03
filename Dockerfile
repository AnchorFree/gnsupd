FROM golang:1.10-alpine3.7 as builder
LABEL maintainer="v.zorin@anchorfree.com"

RUN apk add --no-cache git bash
COPY cmd /go/src/github.com/anchorfree/gnsupd/cmd
COPY Gopkg.toml /go/src/github.com/anchorfree/gnsupd/
COPY Gopkg.lock /go/src/github.com/anchorfree/gnsupd/

RUN cd /go && go get -u github.com/golang/dep/cmd/dep
RUN cd /go/src/github.com/anchorfree/gnsupd/ && dep ensure
RUN cd /go && go build github.com/anchorfree/gnsupd/cmd/gnsupd

FROM alpine:3.7
LABEL maintainer="v.zorin@anchorfree.com"

COPY --from=builder /go/gnsupd /usr/local/bin/gnsupd

ENTRYPOINT ["/usr/local/bin/gnsupd"]
