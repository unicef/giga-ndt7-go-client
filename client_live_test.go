//go:build integration

package ndt7client

import (
	"encoding/json"
	"testing"
)

type liveCallbacks struct {
	t              *testing.T
	chosen         string
	download       string
	upload         string
	downloadProg   int
	uploadProg     int
	errors         []string
}

func (l *liveCallbacks) OnServerDiscovery() {}
func (l *liveCallbacks) OnServerChosen(serverJSON string) {
	l.chosen = serverJSON
}
func (l *liveCallbacks) OnDownloadProgress(string) { l.downloadProg++ }
func (l *liveCallbacks) OnUploadProgress(string)   { l.uploadProg++ }
func (l *liveCallbacks) OnDownloadComplete(summaryJSON string) {
	l.download = summaryJSON
}
func (l *liveCallbacks) OnUploadComplete(summaryJSON string) {
	l.upload = summaryJSON
}
func (l *liveCallbacks) OnError(direction string, message string) {
	l.errors = append(l.errors, direction+": "+message)
	l.t.Errorf("%s: %s", direction, message)
}

func TestRunAgainstMLab(t *testing.T) {
	cb := &liveCallbacks{t: t}
	Run("giga-ndt7-go-client", "0.1.0", cb)
	if len(cb.errors) > 0 {
		t.Fatalf("errors: %v", cb.errors)
	}
	if cb.chosen == "" || cb.download == "" || cb.upload == "" {
		t.Fatalf("missing results chosen=%t download=%t upload=%t", cb.chosen != "", cb.download != "", cb.upload != "")
	}
	assertSummary(t, "download", cb.download)
	assertSummary(t, "upload", cb.upload)
	t.Logf("server %s downloadProgress=%d uploadProgress=%d", cb.chosen, cb.downloadProg, cb.uploadProg)
}

func assertSummary(t *testing.T, direction, raw string) {
	t.Helper()
	var summary completeSummary
	if err := json.Unmarshal([]byte(raw), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.LastClientMeasurement == nil {
		t.Fatalf("%s missing LastClientMeasurement", direction)
	}
	cm := summary.LastClientMeasurement
	if cm.ElapsedTime < 1 || cm.ElapsedTime > 30 {
		t.Fatalf("%s ElapsedTime = %v, want seconds", direction, cm.ElapsedTime)
	}
	if cm.NumBytes <= 0 || cm.MeanClientMbps <= 0 {
		t.Fatalf("%s client measurement = %+v", direction, cm)
	}
	var server map[string]any
	if err := json.Unmarshal(summary.LastServerMeasurement, &server); err != nil {
		t.Fatal(err)
	}
	if server["Origin"] != "server" {
		t.Fatalf("%s Origin = %v", direction, server["Origin"])
	}
	if _, ok := server["TCPInfo"]; !ok {
		t.Fatalf("%s missing TCPInfo", direction)
	}
	if summary.ServerTime == nil || *summary.ServerTime < 1_000_000_000_000 {
		t.Fatalf("%s ServerTime = %v", direction, summary.ServerTime)
	}
}
