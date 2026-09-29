#!/bin/sh
# Builds the Android archive consumed by the daily check app.
# Requires a Go toolchain, the Android NDK, and gomobile on PATH.
#
# Google Play requires native code to support 16 KB memory pages, so the
# shared libraries are linked with a 16 KB maximum page size. NDK r28 and
# later do this by default; the flag makes older NDKs do the same.
set -eu
cd "$(dirname "$0")"
gomobile bind \
  -target=android/arm,android/arm64,android/amd64 \
  -androidapi 24 \
  -ldflags='-extldflags=-Wl,-z,max-page-size=16384' \
  -o ndt7client.aar .

# Record what was built, for the copy checked into the app.
echo "commit: $(git rev-parse HEAD)$(git diff --quiet HEAD -- . || echo ' (dirty)')"
echo "go: $(go version)"
shasum -a 256 ndt7client.aar
