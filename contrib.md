# Contributing

This repository is the ndt7 client for the Android daily check app. It does not
reimplement the protocol: download and upload run through M-Lab's
[ndt7-client-go](https://github.com/m-lab/ndt7-client-go). The code is licensed
under the GNU Affero General Public License v3.0. See [LICENSE](./LICENSE).
The M-Lab libraries it calls stay under the Apache License 2.0. See
[NOTICE](./NOTICE).

## Development

- Go 1.26
- `go test ./...`
- `go test -run TestRunAgainstMLab` talks to M-Lab and needs a network
- `./build-aar.sh` builds the Android archive. It needs Go, the Android NDK, and
  `gomobile` on `PATH`

## Pull requests

- Add or update tests for the behavior you change.
- Tests must pass before merge.
- One review is required.
