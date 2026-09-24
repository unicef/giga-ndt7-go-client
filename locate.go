package ndt7client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/m-lab/locate/api/locate"
	locatev2 "github.com/m-lab/locate/api/v2"
)

const (
	// ClientName is the identity sent to M-Lab on locate and on the test URL.
	ClientName = "ndt7-cliente-go-giga"

	wssDownload = "wss:///ndt/v7/download"
	wssUpload   = "wss:///ndt/v7/upload"
)

// dateTransport records the Date header of the locate response.
type dateTransport struct {
	base http.RoundTripper
	mu   sync.Mutex
	date string
}

func (d *dateTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := d.base
	if base == nil {
		base = http.DefaultTransport
	}
	resp, err := base.RoundTrip(req)
	if err != nil || resp == nil {
		return resp, err
	}
	d.mu.Lock()
	if d.date == "" {
		d.date = resp.Header.Get("Date")
	}
	d.mu.Unlock()
	return resp, nil
}

// discover asks locate once for every nearby ndt7 target that has both
// wss URLs. ndt7-client-go then tries them in order if a connection fails.
// ServerTime is the locate Date header, in epoch milliseconds.
func discover(ctx context.Context, clientVersion string) (*int64, []locatev2.Target, error) {
	transport := &dateTransport{}
	loc := locate.NewClient(ClientName + "/" + clientVersion)
	loc.HTTPClient = &http.Client{Timeout: 15 * time.Second, Transport: transport}
	targets, err := loc.Nearest(ctx, "ndt/ndt7")
	if err != nil {
		return nil, nil, err
	}
	usable := make([]locatev2.Target, 0, len(targets))
	for _, target := range targets {
		if target.URLs[wssDownload] == "" || target.URLs[wssUpload] == "" {
			continue
		}
		usable = append(usable, target)
	}
	if len(usable) == 0 {
		return nil, nil, fmt.Errorf("locate response has no usable ndt7 targets")
	}
	return serverTimeFromDate(transport.date), usable, nil
}

func serverTimeFromDate(header string) *int64 {
	if header == "" {
		return nil
	}
	parsed, err := http.ParseTime(header)
	if err != nil {
		return nil
	}
	ms := parsed.UnixMilli()
	return &ms
}

// fixedLocator returns the targets captured by discover, so download and
// upload share one locate response and ndt7-client-go can fail over.
type fixedLocator struct {
	targets []locatev2.Target
}

func (f fixedLocator) Nearest(context.Context, string) ([]locatev2.Target, error) {
	if len(f.targets) == 0 {
		return nil, fmt.Errorf("locate response has no usable ndt7 targets")
	}
	return f.targets, nil
}

// chosenTargetJSON is the locate target whose URL hostname matches the
// machine the test actually connected to.
func chosenTargetJSON(targets []locatev2.Target, fqdn string) (string, error) {
	chosen := targets[0]
	if fqdn != "" {
		for _, target := range targets {
			for _, rawURL := range target.URLs {
				if strings.Contains(rawURL, fqdn) {
					chosen = target
					encoded, err := json.Marshal(chosen)
					if err != nil {
						return "", err
					}
					return string(encoded), nil
				}
			}
		}
	}
	encoded, err := json.Marshal(chosen)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}
