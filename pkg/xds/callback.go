package xds

import (
	"context"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"sync"
)

// Callbacks for XD Server
type Callbacks struct {
	Signal         chan struct{}
	Fetches        int
	Requests       int
	DeltaRequests  int
	DeltaResponses int
	mu             sync.Mutex
}

func (cb *Callbacks) OnStreamResponse(ctx context.Context, id int64, req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	logrus.Debug("OnStreamResponse", zap.Int64("id", id), zap.Any("Request", req), zap.Any("Response ", resp))
	cb.Report()
}

func (cb *Callbacks) OnDeltaStreamOpen(ctx context.Context, id int64, typ string) error {
	logrus.Debug("OnDeltaStreamOpen", zap.Int64("id", id), zap.String("type", typ))
	return nil
}

func (cb *Callbacks) OnDeltaStreamClosed(id int64) {

}

func (cb *Callbacks) OnStreamDeltaRequest(id int64, req *discovery.DeltaDiscoveryRequest) error {
	logrus.Debug("OnStreamDeltaRequest", zap.Int64("id", id), zap.Any("Request", req))
	return nil
}

func (cb *Callbacks) OnStreamDeltaResponse(id int64, req *discovery.DeltaDiscoveryRequest, resp *discovery.DeltaDiscoveryResponse) {
	logrus.Debug("OnStreamDeltaResponse", zap.Int64("id", id), zap.Any("Request", req), zap.Any("Response", resp))
}

// Report type
func (cb *Callbacks) Report() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	logrus.Debug("cb.Report()  callbacks", zap.Any("Fetches", cb.Fetches), zap.Any("Requests", cb.Requests))
}

// OnStreamOpen type
func (cb *Callbacks) OnStreamOpen(ctx context.Context, id int64, typ string) error {
	logrus.Debug("OnStreamOpen", zap.Int64("id", id), zap.String("type", typ))
	return nil
}

// OnStreamClosed type
func (cb *Callbacks) OnStreamClosed(id int64) {
	logrus.Debug("OnStreamClosed", zap.Int64("id", id))
}

// OnStreamRequest type
func (cb *Callbacks) OnStreamRequest(id int64, req *discovery.DiscoveryRequest) error {
	logrus.Debug("OnStreamRequest", zap.Int64("id", id), zap.Any("Request", req))
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.Requests++
	if cb.Signal != nil {
		close(cb.Signal)
		cb.Signal = nil
	}
	return nil
}

// OnFetchRequest type
func (cb *Callbacks) OnFetchRequest(ctx context.Context, req *discovery.DiscoveryRequest) error {
	logrus.Debug("OnFetchRequest", zap.Any("Request", req))
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.Fetches++
	if cb.Signal != nil {
		close(cb.Signal)
		cb.Signal = nil
	}
	return nil
}

// OnFetchResponse type
func (cb *Callbacks) OnFetchResponse(req *discovery.DiscoveryRequest, resp *discovery.DiscoveryResponse) {
	logrus.Debug("OnFetchResponse", zap.Any("Request", req), zap.Any("Response", resp))
}
