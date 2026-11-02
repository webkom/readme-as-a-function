FROM ghcr.io/openfaas/classic-watchdog:0.3.9 AS watchdog

FROM golang:1.25-alpine AS build-env
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download -x
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o main .

RUN mkdir /TMP

FROM busybox
COPY --from=build-env /app/main .
COPY --from=watchdog /fwatchdog .
COPY --from=build-env /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-env /TMP /tmp
ENV fprocess="./main"
CMD ["./fwatchdog"]
