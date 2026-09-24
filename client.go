// Package ndt7client runs an ndt7 download and upload and reports them with
// the same JSON the desktop client stores for each direction.
package ndt7client

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/m-lab/ndt7-client-go"
	"github.com/m-lab/ndt7-client-go/spec"
)

// Callbacks receives discovery, progress, and completion events. Methods run
// on a Go thread; the caller must not do network or disk work inside them.
type Callbacks interface {
	OnServerDiscovery()
	OnServerChosen(serverJSON string)
	OnDownloadProgress(clientJSON string)
	OnUploadProgress(clientJSON string)
	OnDownloadComplete(summaryJSON string)
	OnUploadComplete(summaryJSON string)
	OnError(direction string, message string)
}

var runMu sync.Mutex

// Run discovers a server, then runs download and upload in that order.
// A second call while one is active reports an error and does not start
// another test. The locate identity is ClientName plus clientVersion.
func Run(clientVersion string, cb Callbacks) {
	if cb == nil {
		return
	}
	if !runMu.TryLock() {
		cb.OnError("download", "a test is already running")
		return
	}
	defer runMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cb.OnServerDiscovery()
	serverTime, targets, err := discover(ctx, clientVersion)
	if err != nil {
		cb.OnError("locate", err.Error())
		return
	}

	client := ndt7.NewClient(ClientName, clientVersion)
	client.Scheme = "wss"
	client.Locate = fixedLocator{targets: targets}

	if err := runDirection(ctx, client.StartDownload, serverTime, cb.OnDownloadProgress, cb.OnDownloadComplete); err != nil {
		cb.OnError("download", err.Error())
		return
	}
	chosen, err := chosenTargetJSON(targets, client.FQDN)
	if err != nil {
		cb.OnError("locate", err.Error())
		return
	}
	cb.OnServerChosen(chosen)
	if err := runDirection(ctx, client.StartUpload, serverTime, cb.OnUploadProgress, cb.OnUploadComplete); err != nil {
		cb.OnError("upload", err.Error())
	}
}

func runDirection(
	ctx context.Context,
	start func(context.Context) (<-chan spec.Measurement, error),
	serverTime *int64,
	onProgress func(string),
	onComplete func(string),
) error {
	ch, err := start(ctx)
	if err != nil {
		return err
	}
	var measurements []spec.Measurement
	for m := range ch {
		measurements = append(measurements, m)
		cm, ok := clientFromMeasurement(m)
		if !ok {
			continue
		}
		encoded, err := json.Marshal(cm)
		if err != nil {
			return err
		}
		onProgress(string(encoded))
	}
	summary, _, ok, err := summarize(measurements, serverTime)
	if err != nil {
		return err
	}
	if !ok {
		return errNoClientMeasurement
	}
	onComplete(string(summary))
	return nil
}

type noClientMeasurementError struct{}

func (noClientMeasurementError) Error() string { return "ndt7 direction produced no client measurement" }

var errNoClientMeasurement = noClientMeasurementError{}
