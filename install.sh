#!/bin/sh
# Install meappy-mc from the latest GitHub release.
#
# Usage:
#   curl -fsSL https://github.com/meappy/mc-fleet/releases/latest/download/install.sh | sh

set -eu

REPO="meappy/mc-fleet"

case "$(uname -s)" in
    Darwin*) OS=darwin ;;
    Linux*)  OS=linux ;;
    *)
        echo "Unsupported OS: $(uname -s)" >&2
        echo "Download manually: https://github.com/${REPO}/releases/latest" >&2
        exit 1
    ;;
esac

case "$(uname -m)" in
    arm64|aarch64) ARCH=arm64 ;;
    x86_64|amd64)  ARCH=amd64 ;;
    *)
        echo "Unsupported architecture: $(uname -m)" >&2
        exit 1
    ;;
esac

DEST="${HOME}/.local/bin/meappy-mc"
ASSET="meappy-mc-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"

mkdir -p "$(dirname "$DEST")"

echo "→ downloading meappy-mc for ${OS}-${ARCH} ..."
if ! curl -fsSL "$URL" -o "${DEST}.partial"; then
    echo "Download failed from $URL" >&2
    rm -f "${DEST}.partial"
    exit 1
fi
mv "${DEST}.partial" "$DEST"
chmod +x "$DEST"

echo "✅ installed → $DEST"
echo
echo "Run it:"
echo "  $DEST"
echo
case ":${PATH:-}:" in
    *":${HOME}/.local/bin:"*) ;;
    *) echo "Tip: add ~/.local/bin to PATH (in ~/.zshrc or ~/.bashrc):"; echo "  export PATH=\"\$HOME/.local/bin:\$PATH\"" ;;
esac
