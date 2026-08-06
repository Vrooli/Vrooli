"""Manifest-shaped namespace adapter used by the session kernel."""

from __future__ import annotations

from pathlib import Path
import json
import sys


def namespace_from_manifest(path: str | Path) -> dict[str, dict[str, str]]:
    manifest = json.loads(Path(path).read_text())
    return {
        group["name"]: {command["name"]: command["name"] for command in group.get("commands", [])}
        for group in manifest.get("groups", [])
    }


if __name__ == "__main__":
    print(json.dumps(namespace_from_manifest(sys.argv[1])))
