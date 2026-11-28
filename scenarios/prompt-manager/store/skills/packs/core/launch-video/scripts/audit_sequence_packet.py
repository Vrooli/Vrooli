#!/usr/bin/env python3
"""Audit a private sequence's exact inputs, blockers, and owned-job cleanup state.

The packet is deliberately not a renderer or quality evaluator. Missing required
media keeps readiness blocked; a no-job cleanup statement is valid only when the
manifest explicitly records that no operation was started.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def audit(manifest_path: Path) -> dict:
    manifest_path = manifest_path.resolve(strict=True)
    manifest = json.loads(manifest_path.read_text())
    if manifest.get("schema") != 1 or not isinstance(manifest.get("artifacts"), list):
        raise ValueError("manifest must use schema 1 and provide artifacts[]")
    root = (manifest_path.parent / manifest.get("root", ".")).resolve(strict=True)
    records, errors, missing = [], [], []
    for item in manifest["artifacts"]:
        rel = Path(item["path"])
        if rel.is_absolute() or ".." in rel.parts:
            raise ValueError(f"artifact path must stay within packet root: {rel}")
        path = root / rel
        expected = item.get("sha256")
        required = bool(item.get("required", True))
        record = {"path": rel.as_posix(), "role": item.get("role", "input"), "required": required,
                  "expected_sha256": expected}
        if not path.exists():
            record["status"] = "missing"
            missing.append(record)
            records.append(record)
            continue
        resolved = path.resolve(strict=True)
        if root not in resolved.parents:
            raise ValueError(f"artifact resolves outside packet root: {rel}")
        if not resolved.is_file():
            raise ValueError(f"artifact is not a regular file: {rel}")
        actual = sha256(resolved)
        record.update(status="verified", actual_sha256=actual, size_bytes=resolved.stat().st_size)
        if expected and actual != expected:
            errors.append(f"{rel}: expected sha256 {expected}, got {actual}")
            record["status"] = "hash-mismatch"
        records.append(record)

    operation = manifest.get("operation", {})
    state = operation.get("state")
    if state == "not-started":
        cleanup = {"status": "not-applicable-no-owned-operation", "owned_resources": [],
                   "basis": operation.get("basis"),
                   "limits": "This records that this packet started no job; it is not a host-wide process or claim inventory."}
    elif state == "completed":
        cleanup_receipt = operation.get("cleanup_receipt")
        if not cleanup_receipt:
            cleanup = {"status": "missing-receipt", "owned_resources": operation.get("owned_resources", [])}
            errors.append("completed operation requires a cleanup_receipt")
        else:
            rel = Path(cleanup_receipt)
            if rel.is_absolute() or ".." in rel.parts or not (root / rel).is_file():
                cleanup = {"status": "missing-receipt", "owned_resources": operation.get("owned_resources", [])}
                errors.append(f"cleanup receipt unavailable: {cleanup_receipt}")
            else:
                cleanup = {"status": "receipt-present-owner-verification-required",
                           "receipt_path": rel.as_posix(), "receipt_sha256": sha256(root / rel),
                           "owned_resources": operation.get("owned_resources", [])}
    else:
        cleanup = {"status": "unresolved-operation-state", "owned_resources": operation.get("owned_resources", [])}
        errors.append("operation state must be not-started or completed")

    required_missing = [item for item in missing if item["required"]]
    return {"schema": 1, "manifest_path": str(manifest_path), "manifest_sha256": sha256(manifest_path),
            "packet_id": manifest.get("packet_id"),
            "readiness": "blocked" if errors or required_missing else "inputs-present",
            "artifacts": records, "missing_required": required_missing, "errors": errors,
            "operation_cleanup": cleanup,
            "limits": ["Input hashes and declared operation state only; no rendering, playback, taste, or claim review."]}


def write_new(path: Path, data: bytes) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    try:
        with os.fdopen(fd, "wb") as stream:
            stream.write(data)
            stream.flush()
            os.fsync(stream.fileno())
    except Exception:
        path.unlink(missing_ok=True)
        raise


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("manifest", type=Path)
    parser.add_argument("--out", required=True, type=Path)
    args = parser.parse_args()
    try:
        result = audit(args.manifest)
        write_new(args.out, (json.dumps(result, indent=2) + "\n").encode())
        print(json.dumps({"readiness": result["readiness"], "verified": sum(x["status"] == "verified" for x in result["artifacts"]),
                          "missing_required": len(result["missing_required"]), "errors": len(result["errors"]),
                          "cleanup": result["operation_cleanup"]["status"], "receipt": str(args.out.resolve())}))
    except (OSError, UnicodeError, ValueError, json.JSONDecodeError) as exc:
        parser.error(str(exc))


if __name__ == "__main__":
    main()
