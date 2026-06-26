#!/usr/bin/env bash
#
# Cross-compile GoTypeMyAdmin for every Go-supported OS/CPU target.
#
# Produces self-contained single-file binaries (the frontend is embedded), one
# archive per platform, plus a SHA256SUMS manifest, in ./dist-bin.
#
# Env overrides:
#   VERSION   release label (default: git describe, else "dev")
#   TARGETS   space/newline list of GOOS/GOARCH (default: all of `go tool dist
#             list` except the wasm targets)
#   SKIP_FRONTEND=1  reuse an existing frontend/dist instead of rebuilding it
#
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/dist-bin"
VERSION="${VERSION:-$(git -C "$ROOT" describe --tags --always --dirty 2>/dev/null || echo dev)}"
LDFLAGS="-s -w -X main.version=${VERSION}"

echo "==> GoTypeMyAdmin release ${VERSION}"
rm -rf "$OUT"; mkdir -p "$OUT"

# 1. Build the frontend and embed it into the backend module.
if [ "${SKIP_FRONTEND:-0}" != "1" ]; then
  echo "==> building frontend"
  ( cd "$ROOT/frontend" && (npm ci || npm install) && npm run build )
fi
echo "==> embedding frontend into backend/web/dist"
rm -rf "$ROOT/backend/web/dist"
mkdir -p "$ROOT/backend/web/dist"
cp -r "$ROOT/frontend/dist/." "$ROOT/backend/web/dist/"

# 2. Resolve the target matrix.
if [ -z "${TARGETS:-}" ]; then
  TARGETS="$(cd "$ROOT/backend" && go tool dist list | grep -vE '/wasm$')"
fi
total="$(echo "$TARGETS" | wc -w | tr -d ' ')"
echo "==> cross-compiling ${total} targets (CGO disabled)"

ok=0; skip=0
for target in $TARGETS; do
  GOOS="${target%/*}"; GOARCH="${target#*/}"
  base="gotypemyadmin_${VERSION}_${GOOS}_${GOARCH}"
  bin="gotypemyadmin"; [ "$GOOS" = "windows" ] && bin="gotypemyadmin.exe"
  stage="$OUT/$base"; mkdir -p "$stage"

  if ! ( cd "$ROOT/backend" && CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
         go build -trimpath -ldflags "$LDFLAGS" -o "$stage/$bin" . ) 2>"$OUT/.err"; then
    printf '  skip  %-18s %s\n' "$target" "$(head -1 "$OUT/.err")"
    rm -rf "$stage"; skip=$((skip+1)); continue
  fi
  cp "$ROOT/README.md" "$stage/" 2>/dev/null || true

  if [ "$GOOS" = "windows" ]; then
    archive="$base.zip"
    if command -v zip >/dev/null 2>&1; then
      ( cd "$OUT" && zip -qr "$archive" "$base" )
    else
      ( cd "$OUT" && python3 -c "import shutil; shutil.make_archive('$base','zip','.','$base')" )
    fi
  else
    archive="$base.tar.gz"
    ( cd "$OUT" && tar czf "$archive" "$base" )
  fi
  rm -rf "$stage"

  if [ ! -s "$OUT/$archive" ]; then
    printf '  FAIL  %-18s (archiving failed)\n' "$target"; skip=$((skip+1)); continue
  fi
  ( cd "$OUT" && sha256sum "$archive" >> SHA256SUMS )
  printf '  ok    %-18s -> %s\n' "$target" "$archive"
  ok=$((ok+1))
done

rm -f "$OUT/.err"
echo "==> done: ${ok} built, ${skip} skipped — artifacts in dist-bin/"
