package ndt7client

import (
	"sync"
	"testing"
	"time"
)

type recordingCallbacks struct {
	mu     sync.Mutex
	errors []string
}

func (r *recordingCallbacks) OnServerDiscovery()                         {}
func (r *recordingCallbacks) OnServerChosen(string)                      {}
func (r *recordingCallbacks) OnDownloadProgress(string)                  {}
func (r *recordingCallbacks) OnUploadProgress(string)                    {}
func (r *recordingCallbacks) OnDownloadComplete(string)                  {}
func (r *recordingCallbacks) OnUploadComplete(string)                    {}
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
