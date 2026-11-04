#!/usr/bin/env bash
# Build hexvpn_bridge (WireGuardKit's Go bridge + libXray in ONE archive) for
# iPhone (arm64) and the arm64 simulator, as c-archives named libwg-go.a —
# the library name WireGuardKit's WireGuardKitGo module map links against.
# Output: hexvpn_bridge/out/<PLATFORM_NAME>/libwg-go.a + out/include/.
#
# Uses a copy of GOROOT with wireguard-apple's boottime-over-monotonic patch
# (Go timers keep counting across device sleep, which WireGuard's keepalive
# and handshake timers rely on), exactly like wireguard-apple's Makefile.
set -euo pipefail
here="$(cd "$(dirname "$0")/.." && pwd)"
bridge="$here/hexvpn_bridge"
out="$bridge/out"
export PATH="$HOME/go/bin:/opt/homebrew/bin:/usr/local/bin:$PATH"

real_goroot="$(go env GOROOT)"
goroot="$bridge/.goroot"
stamp="$(go version)"
if [[ ! -f "$goroot/.prepared" || "$(cat "$goroot/.prepared" 2>/dev/null)" != "$stamp" ]]; then
  echo "==> preparing patched GOROOT ($stamp)"
  rm -rf "$goroot"; mkdir -p "$goroot"
  rsync -a --delete --exclude=pkg/obj/go-build "$real_goroot/" "$goroot/"
  patch -p1 -d "$goroot" < "$bridge/goruntime-boottime-over-monotonic.diff"
  echo "$stamp" > "$goroot/.prepared"
fi
export GOROOT="$goroot"

rm -rf "$out"; mkdir -p "$out/include"
cp "$bridge/libxray_bridge.h" "$bridge/module.modulemap" "$out/include/"

build() {
  local platform="$1" sdk="$2" target="$3"
  local sdkpath; sdkpath="$(xcrun --sdk "$sdk" --show-sdk-path)"
  local flags="-isysroot $sdkpath -arch arm64 $target"
  mkdir -p "$out/$platform"
  echo "==> $platform"
  ( cd "$here" && \
    CGO_ENABLED=1 GOOS=ios GOARCH=arm64 \
    CC="xcrun --sdk $sdk clang" CXX="xcrun --sdk $sdk clang++" \
    CGO_CFLAGS="$flags" CGO_LDFLAGS="$flags" \
    go build -trimpath -ldflags="-w -buildid=" -buildmode=c-archive \
      -o "$out/$platform/libwg-go.a" ./hexvpn_bridge )
  rm -f "$out/$platform/libwg-go.h"
}
build iphoneos iphoneos "-target arm64-apple-ios15.0"
build iphonesimulator iphonesimulator "-target arm64-apple-ios15.0-simulator"
echo "==> done: $(du -sh "$out"/*/libwg-go.a | tr '\n' ' ')"
