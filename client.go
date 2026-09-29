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

var (
	runMu sync.Mutex

	cancelMu     sync.Mutex
	cancelActive context.CancelFunc
)

// Run discovers a server, then runs download and upload in that order. It
// blocks until the test ends, so call it from a thread the caller can spare.
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
	setActiveCancel(cancel)
	defer setActiveCancel(nil)

	cb.OnServerDiscovery()
	serverTime, targets, err := discover(ctx, clientVersion)
	if err != nil {
		cb.OnError("locate", err.Error())
		return
	}

	client := ndt7.NewClient(ClientName, clientVersion)
	client.Scheme = "wss"
	client.Locate = fixedLocator{targets: targets}

	// The server is known once the download connects, so report it before the
	// first progress update rather than after the download.
	var chosenErr error
	onDownloadConnected := func() {
		chosen, err := chosenTargetJSON(targets, client.FQDN)
		if err != nil {
			chosenErr = err
			return
		}
		cb.OnServerChosen(chosen)
	}
	if err := runDirection(ctx, client.StartDownload, onDownloadConnected, serverTime, cb.OnDownloadProgress, cb.OnDownloadComplete); err != nil {
		cb.OnError("download", err.Error())
		return
	}
	if chosenErr != nil {
		cb.OnError("locate", chosenErr.Error())
		return
	}
	if err := runDirection(ctx, client.StartUpload, nil, serverTime, cb.OnUploadProgress, cb.OnUploadComplete); err != nil {
		cb.OnError("upload", err.Error())
	}
}

// Cancel stops the test that Run is running, if any. Run then reports the
// interrupted direction through OnError and returns.
func Cancel() {
	cancelMu.Lock()
	defer cancelMu.Unlock()
	if cancelActive != nil {
		cancelActive()
	}
}

func setActiveCancel(cancel context.CancelFunc) {
	cancelMu.Lock()
	cancelActive = cancel
	cancelMu.Unlock()
}

func runDirection(
	ctx context.Context,
	start func(context.Context) (<-chan spec.Measurement, error),
	onConnected func(),
	serverTime *int64,
	onProgress func(string),
	onComplete func(string),
) error {
	ch, err := start(ctx)
	if err != nil {
		return err
	}
	if onConnected != nil {
		onConnected()
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

func (noClientMeasurementError) Error() string {
	return "ndt7 direction produced no client measurement"
}

var errNoClientMeasurement = noClientMeasurementError{}
