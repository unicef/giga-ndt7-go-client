package ndt7client

import (
	"encoding/json"
	"testing"

	locatev2 "github.com/m-lab/locate/api/v2"
)

func TestChosenTargetJSONUsesTheConnectedHost(t *testing.T) {
	targets := []locatev2.Target{
		{Machine: "mlab1-first.example", URLs: map[string]string{
			wssDownload: "wss://ndt-mlab1-first.example/ndt/v7/download",
		}},
		{Machine: "mlab1-second.example", URLs: map[string]string{
			wssDownload: "wss://ndt-mlab1-second.example/ndt/v7/download",
		}},
	}
	raw, err := chosenTargetJSON(targets, "ndt-mlab1-second.example")
	if err != nil {
		t.Fatal(err)
	}
	var chosen locatev2.Target
	if err := json.Unmarshal([]byte(raw), &chosen); err != nil {
		t.Fatal(err)
	}
	if chosen.Machine != "mlab1-second.example" {
		t.Fatalf("machine = %s", chosen.Machine)
	}
}
