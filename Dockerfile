FROM golang:alpine AS build-env
RUN apk update && apk add git gcc libc-dev curl ca-certificates
RUN mkdir -p /go/src/app
WORKDIR /go/src/app
COPY Gopkg.toml Gopkg.lock ./
RUN go get -v -u github.com/golang/dep/... && dep ensure -vendor-only
ADD . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main main.go handler.go

RUN mkdir /TMP

RUN curl -sL https://github.com/openfaas/faas/releases/download/0.9.0/fwatchdog > /usr/bin/fwatchdog \
    && chmod +x /usr/bin/fwatchdog

FROM busybox
COPY --from=build-env /go/src/app/main .
COPY --from=build-env /usr/bin/fwatchdog .
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /TMP /tmp
ENV fprocess="./main"
CMD ["./fwatchdog"]
