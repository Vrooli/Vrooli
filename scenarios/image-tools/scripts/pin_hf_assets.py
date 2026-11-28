#!/usr/bin/env python3
"""Print the pinned `source.assets[]` rows for files of a Hugging Face repo.

Adding a model to the seed needs every file pinned: a commit-addressed URL, a
sha256 and a size floor. This reads them from the Hub so nobody types a digest:

  * LFS files (weights) carry their sha256 in the Hub tree listing, so nothing is
    downloaded;
  * small plain files (config, tokenizer, dictionary) are fetched once, hashed and
    discarded; they are a few megabytes at most.

Usage:
  python3 scripts/pin_hf_assets.py <org/repo> [--revision <40-char commit>] \
      [--include <glob> ...] [--exclude <glob> ...] [--rename <path>=<filename> ...]

The revision defaults to the repo's current head, which is printed so the row can
carry it. Output is JSON: {"repo", "revision", "license", "assets": [...]}. Paste
`assets` into the model's `source.assets[]` (and `revision` into `rights.revision`).
A file's `kind` is `safetensors`, `onnx`, `gguf` or `bin` by extension, and `config`
for everything else (JSON, txt, tokenizer files), matching the catalog's asset kinds.

This only reads from the Hub to author the seed. It never writes into the managed
model root; the governed install step does the actual download.
"""

from __future__ import annotations

import argparse
import fnmatch
import hashlib
import json
import sys
import urllib.parse
import urllib.request

API = "https://huggingface.co/api/models"
KINDS = {".safetensors": "safetensors", ".onnx": "onnx", ".gguf": "gguf", ".bin": "bin", ".pt": "bin", ".pth": "bin", ".ckpt": "bin"}
MAX_PLAIN_BYTES = 64 * 1024 * 1024  # a plain (non-LFS) file this large is a weight: refuse to hash it blind


def _get(url: str):
    request = urllib.request.Request(url, headers={"User-Agent": "image-tools-pin-assets"})
    return urllib.request.urlopen(request, timeout=60)


def _json(url: str):
    with _get(url) as response:
        return json.load(response)


def _kind(path: str) -> str:
    for suffix, kind in KINDS.items():
        if path.lower().endswith(suffix):
            return kind
    return "config"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("repo", help="org/name")
    parser.add_argument("--revision", default="", help="40-character commit (default: the current head)")
    parser.add_argument("--include", action="append", default=[], help="glob of repo paths to pin (default: all files)")
    parser.add_argument("--exclude", action="append", default=[], help="glob of repo paths to skip")
    parser.add_argument("--rename", action="append", default=[], help="<repo path>=<filename> to store a file under another name")
    args = parser.parse_args()

    info = _json(f"{API}/{args.repo}")
    revision = args.revision or info["sha"]
    renames = dict(item.split("=", 1) for item in args.rename)
    tree = _json(f"{API}/{args.repo}/tree/{revision}?recursive=1")
    assets = []
    for entry in sorted(tree, key=lambda item: item["path"]):
        path = entry["path"]
        if entry.get("type") != "file" or path == ".gitattributes" or path.lower().endswith(".md"):
            continue
        if args.include and not any(fnmatch.fnmatch(path, pattern) for pattern in args.include):
            continue
        if any(fnmatch.fnmatch(path, pattern) for pattern in args.exclude):
            continue
        size = entry["size"]
        lfs = entry.get("lfs") or {}
        if lfs.get("oid"):
            digest = lfs["oid"]
        else:
            if size > MAX_PLAIN_BYTES:
                print(f"refusing to hash {path} ({size} bytes) without an LFS digest", file=sys.stderr)
                return 2
            with _get(f"https://huggingface.co/{args.repo}/resolve/{revision}/{urllib.parse.quote(path)}") as response:
                body = response.read()
            if len(body) != size:
                print(f"{path}: downloaded {len(body)} bytes, tree says {size}", file=sys.stderr)
                return 2
            digest = hashlib.sha256(body).hexdigest()
        assets.append(
            {
                "url": f"https://huggingface.co/{args.repo}/resolve/{revision}/{urllib.parse.quote(path)}",
                "filename": renames.get(path, path),
                "kind": _kind(path),
                "sha256": digest,
                "min_bytes": size,
            }
        )
    if not assets:
        print("no files matched", file=sys.stderr)
        return 1
    license_id = (info.get("cardData") or {}).get("license", "")
    json.dump({"repo": args.repo, "revision": revision, "license": license_id, "assets": assets}, sys.stdout, indent=2)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    sys.exit(main())
