FROM alpine:3.6

RUN apk --no-cache add ca-certificates && update-ca-certificates

COPY kubernetes-grafana-updater.linux.amd64 /usr/bin/kubernetes-grafana-updater
