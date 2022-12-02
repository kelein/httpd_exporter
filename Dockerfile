FROM golang:1.18 AS builder

WORKDIR /go/src/github.com/tanner-bruce/apache_exporter

COPY ./* ./

RUN go mod tidy

RUN env CGO_ENABLED=0 \
        GOOS=linux \
        GOARCH=amd64 \
    go build -o apache_exporter apache_exporter.go && \
    cp ./apache_exporter /apache_exporter

# Multistage
FROM scratch
COPY --from=builder /apache_exporter /apache_exporter

ENTRYPOINT ["/apache_exporter"]
