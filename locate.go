package ndt7client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	locatev2 "github.com/m-lab/locate/api/v2"
)

const locateURL = "https://locate.measurementlab.net/v2/nearest/ndt/ndt7"

type locateResponse struct {
	Results []json.RawMessage `json:"results"`
}

// discover asks the locate service once, keeps the Date header as ServerTime
// in epoch milliseconds, and returns the first result both as raw JSON (for
// OnServerChosen) and as the targets the ndt7 client dials.
func discover(ctx context.Context, clientName, clientVersion string) (*int64, json.RawMessage, []locatev2.Target, error) {
	u, err := url.Parse(locateURL)
	if err != nil {
		return nil, nil, nil, err
	}
	q := u.Query()
	q.Set("client_name", clientName)
	q.Set("client_version", clientVersion)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, nil, err
	}
	req.Header.Set("User-Agent", clientName+"/"+clientVersion)

	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, nil, nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, nil, fmt.Errorf("locate returned %s", resp.Status)
	}

	var parsed locateResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, nil, nil, err
	}
	if len(parsed.Results) == 0 {
		return nil, nil, nil, fmt.Errorf("locate response has no results")
	}

	var targets []locatev2.Target
	if err := json.Unmarshal(body, &struct {
		Results *[]locatev2.Target `json:"results"`
	}{Results: &targets}); err != nil {
		return nil, nil, nil, err
	}
	if len(targets) == 0 {
		return nil, nil, nil, fmt.Errorf("locate response has no results")
	}
	return serverTimeFromDate(resp.Header.Get("Date")), parsed.Results[0], targets, nil
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

// fixedLocator returns a locate result captured earlier, so download and
// upload share one discovery and one ServerTime.
type fixedLocator struct {
	targets []locatev2.Target
}

func (f fixedLocator) Nearest(context.Context, string) ([]locatev2.Target, error) {
	if len(f.targets) == 0 {
		return nil, fmt.Errorf("locate response has no results")
	}
	return f.targets, nil
}
