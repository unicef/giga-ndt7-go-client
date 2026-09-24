# giga-ndt7-go-client

ndt7 client for the Android daily check app. It runs download and upload through
[ndt7-client-go](https://github.com/m-lab/ndt7-client-go) and reports each
direction as the JSON object the desktop client stores: client elapsed time in
seconds, `MeanClientMbps`, the full server measurement, and `ServerTime` from
the locate response `Date` header.

`Run` is safe to call from Kotlin via gomobile. Build the archive with
`./build-aar.sh` (Go, Android NDK, and `gomobile` on `PATH`).
