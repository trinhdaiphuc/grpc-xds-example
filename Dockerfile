FROM golang:1.18.0-alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=1  \
    GOARCH="amd64" \
    GOOS=linux

RUN apk update && apk add make git pkgconfig gcc g++ bash

WORKDIR /app
ADD go.mod .
ADD go.sum .
RUN go mod download

COPY . .

RUN go build -tags musl --ldflags "-extldflags -static" -o grpc-example main.go

FROM alpine:3.15.4

WORKDIR /home

COPY --from=builder /app/grpc-example .

ENTRYPOINT ["./grpc-example"]
