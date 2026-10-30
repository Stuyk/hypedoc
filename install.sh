#!/bin/sh
# Install the latest (or a pinned) hypedoc release.
#   curl -fsSL https://raw.githubusercontent.com/stuyk/hypedoc/main/install.sh | sh
# Env: HYPEDOC_VERSION=0.1.0 (default: latest), HYPEDOC_INSTALL_DIR (default: ~/.local/bin)
set -eu

repo="stuyk/hypedoc"
dir="${HYPEDOC_INSTALL_DIR:-$HOME/.local/bin}"

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) echo "hypedoc: unsupported OS $(uname -s); download from https://github.com/$repo/releases" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "hypedoc: unsupported arch $(uname -m)" >&2; exit 1 ;;
esac

if [ -n "${HYPEDOC_VERSION:-}" ]; then
  version="${HYPEDOC_VERSION#v}"
else
  version="$(curl -fsSL "https://api.github.com/repos/$repo/releases/latest" | sed -n 's/.*"tag_name": *"v\{0,1\}\([^"]*\)".*/\1/p' | head -n1)"
  [ -n "$version" ] || { echo "hypedoc: could not determine latest release" >&2; exit 1; }
fi

archive="hypedoc_${version}_${os}_${arch}.tar.gz"
base="https://github.com/$repo/releases/download/v$version"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "downloading hypedoc v$version ($os/$arch)..."
curl -fsSL "$base/$archive" -o "$tmp/$archive"
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt"

expected="$(grep " $archive\$" "$tmp/checksums.txt" | cut -d' ' -f1)"
if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp/$archive" | cut -d' ' -f1)"
else
  actual="$(shasum -a 256 "$tmp/$archive" | cut -d' ' -f1)"
fi
if [ -z "$expected" ] || [ "$expected" != "$actual" ]; then
  echo "hypedoc: checksum mismatch for $archive" >&2; exit 1
fi

tar -xzf "$tmp/$archive" -C "$tmp" hypedoc
mkdir -p "$dir"
install -m 0755 "$tmp/hypedoc" "$dir/hypedoc"
echo "installed $dir/hypedoc ($("$dir/hypedoc" version))"
case ":$PATH:" in *":$dir:"*) ;; *) echo "note: add $dir to your PATH" ;; esac
