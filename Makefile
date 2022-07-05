docker-build-m1:
	docker build --platform=linux/amd64 . -t localhost:5001/grpc-example

docker-build:
	docker build . -t localhost:5001/grpc-example

docker-push:
	docker push localhost:5001/grpc-example:latest

build-linux:
	GO111MODULE=on GOARCH="amd64" GOOS=linux go build -o grpc-example main.go