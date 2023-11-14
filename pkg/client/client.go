package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	ecpb "google.golang.org/grpc/examples/features/proto/echo"
	"google.golang.org/grpc/status"
)

var (
	logger *zap.Logger
)

func init() {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, _ = loggerConfig.Build()
}

func ServeHTTP() {
	// Register Prometheus metrics handler.
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	// Start the health check endpoint and make sure not to block
	fmt.Println("Server running on localhost:8080")
	_ = http.ListenAndServe(":8080", nil)
}

func callUnaryEcho(idx int, c ecpb.EchoClient, message string) {
	_, err := c.UnaryEcho(context.Background(), &ecpb.EchoRequest{Message: message})
	if err != nil {
		logger.Fatal("Can not send unary request", zap.Int("worker", idx), zap.Error(err))
	}
}

func startWorker(idx int, cc *grpc.ClientConn, client ecpb.EchoClient, closeChannels <-chan bool) {
	stream, err := client.BidirectionalStreamingEcho(context.Background())
	if err != nil {
		logger.Fatal("could not send request bi direction", zap.Error(err))
	}

	for {
		select {
		case <-closeChannels:
			if stream != nil {
				err := stream.CloseSend()
				if err != nil {
					logger.Error("Close streaming error", zap.Int("worker", idx), zap.Error(err))
				}
			}
			logger.Info("Worker stopped", zap.Int("worker", idx))
			return
		default:
			if stream != nil {
				err = stream.Send(&ecpb.EchoRequest{Message: "streaming message"})
				if err != nil {
					if status.Code(err) == codes.Unavailable || errors.Is(err, io.EOF) {
						stream = reconnect(idx, cc, stream)
					} else {
						logger.Error("Could not send streaming message", zap.Int("worker", idx), zap.Error(err))
					}
				}
			} else {
				stream = reconnect(idx, cc, stream)
			}
			callUnaryEcho(idx, client, "this is examples/load_balancing")
		}
	}
}

func watchState(cc *grpc.ClientConn, ctx context.Context, state connectivity.State) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		default:
			if cc.GetState() == state {
				return true
			}
		}
	}
}

func reconnect(idx int, cc *grpc.ClientConn, stream ecpb.Echo_BidirectionalStreamingEchoClient) ecpb.Echo_BidirectionalStreamingEchoClient {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var err error
	logger.Info("Reconnecting...", zap.Int("worker", idx), zap.Stringer("state", cc.GetState()))
	if watchState(cc, ctx, connectivity.Ready) {
		logger.Info("Reconnected to server.", zap.Int("worker", idx), zap.Stringer("state", cc.GetState()))
		stream, err = ecpb.NewEchoClient(cc).BidirectionalStreamingEcho(context.Background())
		if err != nil {
			logger.Error("New stream failed", zap.Int("worker", idx), zap.Error(err))
		}
		return stream
	}

	logger.Info("Reconnect failed", zap.Int("worker", idx), zap.Stringer("state", cc.GetState()))

	return nil
}

func LoadTest(cc *grpc.ClientConn, duration string, concurrency int) {
	hwc := ecpb.NewEchoClient(cc)
	logger.Info("--- calling helloworld.Greeter/SayHello ---")

	timeDuration, err := time.ParseDuration(duration)
	if err != nil {
		logger.Fatal("could not parse duration", zap.Error(err))
	}
	ticker := time.NewTicker(timeDuration)
	closeChannels := make(chan bool, concurrency)
	for i := 0; i < concurrency; i++ {
		go startWorker(i, cc, hwc, closeChannels)
	}

	<-ticker.C
	for i := 0; i < concurrency; i++ {
		closeChannels <- true
	}
	fmt.Println("done")
}
