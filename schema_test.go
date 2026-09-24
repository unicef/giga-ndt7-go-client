package ndt7client

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/m-lab/ndt7-client-go/spec"
)

func TestSummarizeConvertsMicrosecondsAndIgnoresServerFrameBytes(t *testing.T) {
	serverBytes := int64(999999)
	measurements := []spec.Measurement{
		{
			AppInfo: &spec.AppInfo{ElapsedTime: 10e6, NumBytes: 12500000},
			Origin:  spec.OriginClient,
			Test:    spec.TestDownload,
		},
		{
			AppInfo: &spec.AppInfo{ElapsedTime: 10e6, NumBytes: serverBytes},
			Origin:  spec.OriginServer,
			Test:    spec.TestDownload,
			TCPInfo: &spec.TCPInfo{},
		},
	}
	serverTime := int64(1720000000000)
	raw, client, ok, err := summarize(measurements, &serverTime)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected a client measurement")
	}
	if client.ElapsedTime != 10 {
		t.Fatalf("ElapsedTime = %v, want 10 seconds", client.ElapsedTime)
	}
	if client.NumBytes != 12500000 {
		t.Fatalf("NumBytes = %d, server frame bytes leaked in", client.NumBytes)
	}
	wantMbps := (float64(12500000) * 8) / 1e6 / 10
	if math.Abs(client.MeanClientMbps-wantMbps) > 1e-9 {
		t.Fatalf("MeanClientMbps = %v, want %v", client.MeanClientMbps, wantMbps)
	}

	var summary completeSummary
	if err := json.Unmarshal(raw, &summary); err != nil {
		t.Fatal(err)
	}
	if summary.ServerTime == nil || *summary.ServerTime != serverTime {
		t.Fatalf("ServerTime = %v", summary.ServerTime)
	}
	var server map[string]any
	if err := json.Unmarshal(summary.LastServerMeasurement, &server); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"Origin", "Test", "AppInfo"} {
		if _, exists := server[key]; !exists {
			t.Fatalf("server measurement missing %s: %s", key, summary.LastServerMeasurement)
		}
	}
	if server["Origin"] != "server" {
		t.Fatalf("Origin = %v", server["Origin"])
	}
}

func TestSummarizeServerTimeNullWhenMissing(t *testing.T) {
	raw, _, ok, err := summarize(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected no client measurement")
	}
	var summary map[string]any
	if err := json.Unmarshal(raw, &summary); err != nil {
		t.Fatal(err)
	}
	if summary["ServerTime"] != nil {
		t.Fatalf("ServerTime = %v, want null", summary["ServerTime"])
	}
}
