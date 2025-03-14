ARG go_base_version=1.23
ARG build_version
ARG build_commit

FROM golang:${go_base_version} AS builder
ARG build_version
ARG build_commit
ARG cert_location=/usr/local/share/ca-certificates

RUN apt-get update && apt-get install -y ca-certificates openssl
RUN openssl s_client -showcerts -connect github.com:443 </dev/null 2>/dev/null | openssl x509 -outform PEM > ${cert_location}/github.crt
RUN openssl s_client -showcerts -connect proxy.golang.org:443 </dev/null 2>/dev/null | openssl x509 -outform PEM > ${cert_location}/proxy.golang.crt
RUN openssl s_client -showcerts -connect sum.golang.org:443 </dev/null 2>/dev/null | openssl x509 -outform PEM > ${cert_location}/sum.golang.crt
RUN update-ca-certificates

WORKDIR /build

COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go mod verify

#COPY cmd ./cmd
COPY *.go ./
COPY pkg ./pkg

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags="-X 'main.GitCommit=${build_commit}' -X 'main.Version=${build_version}'" -o echo-server .

FROM golang:${go_base_version}-alpine

RUN apk --no-cache add ca-certificates

WORKDIR /app
EXPOSE 8080

COPY --from=builder /build/echo-server .

CMD ["./echo-server"]
