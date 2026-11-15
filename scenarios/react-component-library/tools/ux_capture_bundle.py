"""Consume and remove a GCT EX9 bundle created for an RCL visual artifact."""

from __future__ import annotations

import hashlib
import json
import uuid
import argparse
from pathlib import Path


def copy_artifact(response_path: Path, kind: str, destination: Path) -> None:
    copy_response(json.loads(response_path.read_text()), kind, destination)


def copy_response(response: dict, kind: str, destination: Path) -> None:
    manifest = response.get("manifest")
    bundle_dir_value = response.get("bundle_dir")
    if not isinstance(manifest, dict) or manifest.get("schema_version") != "ux-capture.v1":
        raise ValueError("GCT did not return a versioned EX9 bundle")
    if manifest.get("privacy", {}).get("redaction_status") != "complete":
        raise ValueError("capture did not complete its privacy redaction prepass")
    bundle_id = str(manifest.get("id", ""))
    if str(uuid.UUID(bundle_id)) != bundle_id.lower():
        raise ValueError("bundle id is not a canonical UUID")

    bundle_dir = Path(str(bundle_dir_value))
    if bundle_dir.is_symlink():
        raise ValueError("bundle directory cannot be a symlink")
    resolved_bundle = bundle_dir.resolve(strict=True)
    if resolved_bundle.parent.name != "ux" or resolved_bundle.name != bundle_id:
        raise ValueError("bundle directory is not a BAS EX9 UUID directory")

    matches = [item for item in manifest.get("artifacts", []) if item.get("kind") == kind]
    if len(matches) != 1:
        raise ValueError(f"bundle must contain exactly one {kind} artifact")
    artifact = matches[0]
    relative_text = str(artifact.get("path", ""))
    relative = Path(relative_text)
    if "\\" in relative_text or relative.is_absolute() or ".." in relative.parts:
        raise ValueError("artifact path is unsafe")
    source = resolved_bundle / relative
    if (
        source.is_symlink()
        or any(parent.is_symlink() for parent in source.parents if parent != resolved_bundle.parent)
        or not source.is_file()
    ):
        raise ValueError("artifact file is missing or is a symlink")
    resolved_source = source.resolve(strict=True)
    if resolved_bundle not in resolved_source.parents:
        raise ValueError("artifact path escapes its bundle directory")
    content = resolved_source.read_bytes()
    if len(content) != artifact.get("size_bytes"):
        raise ValueError("artifact size does not match the manifest")
    if hashlib.sha256(content).hexdigest() != artifact.get("sha256"):
        raise ValueError("artifact digest does not match the manifest")

    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.write_bytes(content)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("response", type=Path)
    parser.add_argument("kind")
    parser.add_argument("destination", type=Path)
    args = parser.parse_args()
    copy_artifact(args.response, args.kind, args.destination)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

