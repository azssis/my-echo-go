# syntax=docker/dockerfile:1

ARG go_base_version=1.23
ARG build_version
ARG build_commit

FROM golang:${go_base_version} AS builder
ARG go_base_version
ARG build_version
ARG build_commit

WORKDIR /build

COPY go.mod ./
COPY go.sum ./
RUN go mod download
RUN go mod verify

COPY *.go ./
COPY pkg ./pkg

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags="-s -w" -o my-echo-go .

FROM golang:${go_base_version}-alpine

EXPOSE 8080

WORKDIR /app

COPY --from=builder /build/my-echo-go ./my-echo-go

ENTRYPOINT ["/app/my-echo-go"]
