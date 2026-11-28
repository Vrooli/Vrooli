#!/usr/bin/env python3
"""Render the DBM platform qualification table from durable evidence receipts."""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
from datetime import datetime
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RECEIPTS = ROOT / "docs/reference/platform-receipts.json"
VOLUME_RECEIPTS = Path(os.environ.get("VROOLI_HOME", Path.home() / ".vrooli")) / "state/volume-remediation/receipts.jsonl"
DBM_RECEIPTS = Path(os.environ.get("DBM_PLATFORM_RECEIPTS_PATH", Path(os.environ.get("VROOLI_HOME", Path.home() / ".vrooli")) / "state/data-backup-manager/platform-receipts.jsonl"))
MATRIX = Path(os.environ.get("DBM_PLATFORM_MATRIX_PATH", ROOT / "docs/reference/support-matrix.md"))
START = "<!-- BEGIN GENERATED PLATFORM MATRIX -->"
END = "<!-- END GENERATED PLATFORM MATRIX -->"
PLATFORMS = ("linux", "macos", "windows")
CAPABILITIES = (
    ("api_cli_build", "API and CLI native build"),
    ("volume_identity", "Volume identity"),
    ("destination_check", "Read-only destination check"),
    ("filesystem_repair", "Filesystem repair"),
    ("volume_mount", "Mount and remount"),
    ("backup", "Backup to a repository"),
    ("scratch_restore", "Scratch restore and verification"),
)
ID_RE = re.compile(r"^[a-z0-9][a-z0-9-]{2,95}$")
HEX_RE = re.compile(r"^[a-f0-9]{64}$")


def load_receipts() -> list[dict[str, object]]:
    payload = json.loads(RECEIPTS.read_text(encoding="utf-8"))
    if payload.get("schema_version") != 1 or not isinstance(payload.get("receipts"), list):
        raise ValueError("receipt document must use schema_version 1 and a receipts array")
    receipts = list(payload["receipts"])
    if VOLUME_RECEIPTS.exists():
        for line_no, line in enumerate(VOLUME_RECEIPTS.read_text(encoding="utf-8").splitlines(), 1):
            if not line.strip():
                continue
            try:
                row = json.loads(line)
            except json.JSONDecodeError as exc:
                raise ValueError(f"invalid control-plane volume receipt line {line_no}") from exc
            action_to_capability = {
                "inspect": "volume_identity",
                "check": "destination_check",
                "repair": "filesystem_repair",
                "unmount": "volume_mount",
                "mount_read_write": "volume_mount",
            }
            if row.get("action") not in action_to_capability:
                raise ValueError(f"unsupported control-plane volume action on line {line_no}")
            receipts.append({
                "id": row.get("id"), "platform": row.get("platform"),
                "architecture": row.get("architecture"), "host": row.get("host"),
                "observed_at": row.get("observed_at"),
                "capability": action_to_capability[row["action"]],
                "outcome": row.get("outcome"), "evidence_kind": row.get("evidence_kind"),
                "command": row.get("command"), "evidence": row.get("evidence"),
            })
    if DBM_RECEIPTS.exists():
        for line_no, line in enumerate(DBM_RECEIPTS.read_text(encoding="utf-8").splitlines(), 1):
            if not line.strip():
                continue
            try:
                row = json.loads(line)
            except json.JSONDecodeError as exc:
                raise ValueError(f"invalid DBM platform receipt line {line_no}") from exc
            if row.get("capability") not in ("destination_check", "backup", "scratch_restore"):
                raise ValueError(f"unsupported DBM platform capability on line {line_no}")
            receipts.append(row)
    seen: set[str] = set()
    allowed_caps = {key for key, _ in CAPABILITIES}
    for row in receipts:
        if not isinstance(row, dict):
            raise ValueError("each receipt must be an object")
        required = ("id", "platform", "architecture", "host", "observed_at", "capability", "outcome", "evidence_kind", "command", "evidence")
        if any(not isinstance(row.get(key), str) or not row[key].strip() for key in required):
            raise ValueError("receipt is missing a required non-empty string field")
        if not ID_RE.fullmatch(row["id"]) or row["id"] in seen:
            raise ValueError(f"invalid or duplicate receipt id: {row['id']}")
        seen.add(row["id"])
        if row["platform"] not in PLATFORMS or row["capability"] not in allowed_caps:
            raise ValueError(f"unsupported platform or capability in receipt {row['id']}")
        if row["outcome"] not in ("passed", "failed"):
            raise ValueError(f"unsupported outcome in receipt {row['id']}")
        if row["evidence_kind"] not in ("native-build", "native-runtime", "cross-build", "fixture"):
            raise ValueError(f"unsupported evidence kind in receipt {row['id']}")
        try:
            observed_at = datetime.fromisoformat(row["observed_at"].replace("Z", "+00:00"))
        except ValueError as exc:
            raise ValueError(f"invalid observed_at in receipt {row['id']}") from exc
        if observed_at.tzinfo is None:
            raise ValueError(f"observed_at must include a timezone in receipt {row['id']}")
        for digest_key in ("artifact_sha256", "source_sha256"):
            digest = row.get(digest_key)
            if digest is not None and (not isinstance(digest, str) or not HEX_RE.fullmatch(digest)):
                raise ValueError(f"invalid {digest_key} in receipt {row['id']}")
    return receipts


def render_cell(receipts: list[dict[str, object]], platform: str, capability: str) -> str:
    matches = [r for r in receipts if r["platform"] == platform and r["capability"] == capability]
    native = [r for r in matches if r["evidence_kind"] in ("native-build", "native-runtime")]
    if not native:
        return "no native receipt"
    latest = max(native, key=lambda r: datetime.fromisoformat(str(r["observed_at"]).replace("Z", "+00:00")))
    state = "qualified" if latest["outcome"] == "passed" else "failed"
    return f"{state} · `{latest['id']}` ({latest['observed_at']})"


def render_block(receipts: list[dict[str, object]]) -> str:
    rows = [START, "| Capability | Linux | macOS | Windows |", "|---|---|---|---|"]
    for capability, label in CAPABILITIES:
        cells = [render_cell(receipts, platform, capability) for platform in PLATFORMS]
        rows.append("| " + label + " | " + " | ".join(cells) + " |")
    rows.append(END)
    return "\n".join(rows)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true", help="fail if the checked-in matrix differs from receipts")
    args = parser.parse_args()
    try:
        receipts = load_receipts()
        current = MATRIX.read_text(encoding="utf-8")
        block = render_block(receipts)
        if START not in current or END not in current or current.index(START) >= current.index(END):
            raise ValueError("support matrix is missing its generated-section markers")
        updated = current[: current.index(START)] + block + current[current.index(END) + len(END) :]
        if args.check:
            if updated != current:
                print("support matrix is stale; run python3 scripts/generate_support_matrix.py", file=sys.stderr)
                return 1
            return 0
        MATRIX.write_text(updated, encoding="utf-8")
        return 0
    except (OSError, ValueError, json.JSONDecodeError) as exc:
        print(f"support matrix generation failed: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
