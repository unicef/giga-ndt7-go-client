#!/bin/sh
# Builds the Android archive consumed by the daily check app.
# Requires a Go toolchain, the Android NDK, and gomobile on PATH.
set -eu
cd "$(dirname "$0")"
gomobile bind -target=android/arm,android/arm64,android/amd64 -androidapi 24 -o ndt7client.aar .
