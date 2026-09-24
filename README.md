# giga-ndt7-go-client

ndt7 client for the Android daily check app. It does not reimplement the ndt7
protocol. Download and upload run through M-Lab's
[ndt7-client-go](https://github.com/m-lab/ndt7-client-go).

`Run` locates nearby servers, then hands that list to the M-Lab client:

```go
client := ndt7.NewClient(ClientName, clientVersion)
client.Scheme = "wss"
client.Locate = fixedLocator{targets: targets}

client.StartDownload(ctx) // then client.StartUpload(ctx)
```

`ndt7.NewClient` is M-Lab's constructor. `ClientName` is `ndt7-cliente-go-giga`.
`client.Locate` does not call locate again: it returns the servers `discover`
already fetched with M-Lab's locate client. `StartDownload` and `StartUpload`
open the WebSocket and return a stream of measurements.

This package only wraps that stream:

- Locate keeps every nearby server that offers both wss URLs. If the first one
  does not connect, ndt7-client-go tries the next. `OnServerChosen` reports the
  server that connected.
- Each direction is reported as the JSON object the desktop client stores:
  `ElapsedTime` in seconds (ndt7-client-go counts microseconds), `MeanClientMbps`
  from the client's own byte count, the full server measurement, and
  `ServerTime` from the locate response `Date` header.

`Run` is safe to call from Kotlin via gomobile. Build the archive with
`./build-aar.sh` (Go, Android NDK, and `gomobile` on `PATH`).
