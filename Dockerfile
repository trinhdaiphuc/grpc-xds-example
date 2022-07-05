FROM alpine:3.15.4

WORKDIR /home

COPY ./grpc-example .

RUN chmod +x grpc-example

ENTRYPOINT ["./grpc-example"]