FROM golang:1.20-alpine AS builder

ENV GO111MODULE=on \
    CGO_ENABLED=1

RUN apk update && apk add make git pkgconfig gcc g++ bash

WORKDIR /app
ADD go.mod .
ADD go.sum .
RUN go mod download

COPY . .

RUN go build -tags musl --ldflags "-extldflags -static" -o grpc-example main.go

FROM alpine:3.18.4

WORKDIR /home

COPY --from=builder /app/grpc-example .

ENTRYPOINT ["./grpc-example"]
