docker-build-m1:
	docker build --platform=linux/amd64 . -t bigphuc/grpc-example

docker-build:
	docker build . -t bigphuc/grpc-example

docker-push:
	docker push bigphuc/grpc-example

build-linux:
	GO111MODULE=on GOARCH="amd64" GOOS=linux go build -o bin/grpc-example main.go