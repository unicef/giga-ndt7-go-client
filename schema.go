package ndt7client

import (
	"encoding/json"
	"math"

	"github.com/m-lab/ndt7-client-go/spec"
)

// clientMeasurement is the progress and LastClientMeasurement object emitted
// by the desktop ndt7 workers: elapsed time in seconds, not microseconds.
type clientMeasurement struct {
	ElapsedTime    float64 `json:"ElapsedTime"`
	NumBytes       int64   `json:"NumBytes"`
	MeanClientMbps float64 `json:"MeanClientMbps"`
}

// completeSummary is the downloadComplete / uploadComplete object, including
// ServerTime from the vendored desktop client.
type completeSummary struct {
	LastClientMeasurement *clientMeasurement `json:"LastClientMeasurement,omitempty"`
	LastServerMeasurement json.RawMessage    `json:"LastServerMeasurement,omitempty"`
	ServerTime            *int64             `json:"ServerTime"`
}

func clientFromMeasurement(m spec.Measurement) (clientMeasurement, bool) {
	if m.Origin != spec.OriginClient || m.AppInfo == nil {
		return clientMeasurement{}, false
	}
	elapsedSec := float64(m.AppInfo.ElapsedTime) / 1e6
	var mbps float64
	if elapsedSec > 0 {
		mbps = (float64(m.AppInfo.NumBytes) * 8) / 1e6 / elapsedSec
	}
	if math.IsNaN(mbps) || math.IsInf(mbps, 0) {
		mbps = 0
	}
	return clientMeasurement{
		ElapsedTime:    elapsedSec,
		NumBytes:       m.AppInfo.NumBytes,
		MeanClientMbps: mbps,
	}, true
}

// summarize turns one direction's measurement stream into the desktop
// complete JSON. Client byte counts come only from client-origin AppInfo.
// A server measurement frame is not added to NumBytes. The server object is
// the last server-origin message, serialized whole.
func summarize(measurements []spec.Measurement, serverTime *int64) ([]byte, clientMeasurement, bool, error) {
	var lastClient *clientMeasurement
	var lastServer *spec.Measurement
	for i := range measurements {
		m := measurements[i]
		if cm, ok := clientFromMeasurement(m); ok {
			copyCM := cm
			lastClient = &copyCM
			continue
		}
		if m.Origin == spec.OriginServer {
			copyM := m
			lastServer = &copyM
		}
	}
	summary := completeSummary{ServerTime: serverTime}
	if lastClient != nil {
		summary.LastClientMeasurement = lastClient
	}
	if lastServer != nil {
		raw, err := json.Marshal(lastServer)
		if err != nil {
			return nil, clientMeasurement{}, false, err
		}
		summary.LastServerMeasurement = raw
	}
	encoded, err := json.Marshal(summary)
	if err != nil {
		return nil, clientMeasurement{}, false, err
	}
	if lastClient == nil {
		return encoded, clientMeasurement{}, false, nil
	}
	return encoded, *lastClient, true, nil
}
