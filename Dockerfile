FROM golang:1.18-alpine AS builder
WORKDIR /go/src/app
COPY . .
RUN CGO_ENABLED=0 go build

FROM scratch
COPY --from=builder /go/src/app/mikrotik-exporter /mikrotik-exporter
EXPOSE 9436
ENTRYPOINT ["/mikrotik-exporter"]
