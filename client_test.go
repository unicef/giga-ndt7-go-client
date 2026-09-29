package ndt7client

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/m-lab/ndt7-client-go/spec"
)

type recordingCallbacks struct {
	mu     sync.Mutex
	errors []string
}

func (r *recordingCallbacks) OnServerDiscovery()        {}
func (r *recordingCallbacks) OnServerChosen(string)     {}
func (r *recordingCallbacks) OnDownloadProgress(string) {}
func (r *recordingCallbacks) OnUploadProgress(string)   {}
func (r *recordingCallbacks) OnDownloadComplete(string) {}
func (r *recordingCallbacks) OnUploadComplete(string)   {}
func (r *recordingCallbacks) OnError(direction string, message string) {
	r.mu.Lock()
	r.errors = append(r.errors, direction+": "+message)
	r.mu.Unlock()
}

func TestRunRejectsASecondCall(t *testing.T) {
	runMu.Lock()
	defer runMu.Unlock()

	cb := &recordingCallbacks{}
	done := make(chan struct{})
	go func() {
		Run("test", cb)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("second Run did not return")
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if len(cb.errors) != 1 {
		t.Fatalf("errors = %v, want one rejection", cb.errors)
	}
	if cb.errors[0] != "download: a test is already running" {
		t.Fatalf("error = %q", cb.errors[0])
	}
}

func TestCancelWithoutARunIsANoOp(t *testing.T) {
	Cancel()
}

func TestCancelStopsTheActiveRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	setActiveCancel(cancel)
	defer setActiveCancel(nil)

	Cancel()

	if ctx.Err() == nil {
		t.Fatal("Cancel did not cancel the active run")
	}
}

func TestRunDirectionReportsConnectedBeforeProgress(t *testing.T) {
	var events []string
	start := func(context.Context) (<-chan spec.Measurement, error) {
		ch := make(chan spec.Measurement, 1)
		ch <- spec.Measurement{
			Origin:  spec.OriginClient,
			AppInfo: &spec.AppInfo{ElapsedTime: 1_000_000, NumBytes: 125_000},
		}
		close(ch)
		return ch, nil
	}

	err := runDirection(
		context.Background(),
		start,
		func() { events = append(events, "connected") },
		nil,
		func(string) { events = append(events, "progress") },
		func(string) { events = append(events, "complete") },
	)

	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 2 || events[0] != "connected" || events[1] != "progress" {
		t.Fatalf("events = %v, want connected before progress", events)
	}
}

func TestRunDirectionDoesNotReportConnectedWhenStartFails(t *testing.T) {
	connected := false
	start := func(context.Context) (<-chan spec.Measurement, error) {
		return nil, errNoClientMeasurement
	}

	err := runDirection(context.Background(), start, func() { connected = true }, nil, func(string) {}, func(string) {})

	if err == nil || connected {
		t.Fatalf("err = %v, connected = %t", err, connected)
	}
}
