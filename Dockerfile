FROM golang:alpine AS build-env
RUN apk update && apk add git gcc libc-dev curl ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download -x
ADD . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

RUN mkdir /TMP

RUN curl -sL https://github.com/openfaas/faas/releases/download/0.9.0/fwatchdog > /usr/bin/fwatchdog \
    && chmod +x /usr/bin/fwatchdog

FROM busybox
COPY --from=build-env /app/main .
COPY --from=build-env /usr/bin/fwatchdog .
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /TMP /tmp
ENV fprocess="./main"
CMD ["./fwatchdog"]
