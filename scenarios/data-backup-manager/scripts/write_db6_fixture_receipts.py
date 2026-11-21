#!/usr/bin/env python3
"""Materialize typed, explicitly non-native receipts for the simulator phase."""

import hashlib
import json
import os
import platform
import sys
from datetime import datetime, timezone
from pathlib import Path


def platform_name() -> str:
    return {"darwin": "macos"}.get(sys.platform, "windows" if os.name == "nt" else "linux")


def main() -> int:
    volume_path = Path(os.environ["VROOLI_HOME"]) / "state/volume-remediation/receipts.jsonl"
    dbm_path = Path(os.environ["DBM_PLATFORM_RECEIPTS_PATH"])
    observed = datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")
    # The digest binds the phase rows to the exact source inputs being tested;
    # it is intentionally not a claim about a real device or repository.
    root = Path(__file__).resolve().parents[1]
    digest = hashlib.sha256()
    for relative in (
        "api/internal/destinationreadiness/recovery.go",
        "api/handlers/destinations/recovery_decision.go",
        "scripts/db6-fault-injection.sh",
    ):
        digest.update((root / relative).read_bytes())
    source_hash = digest.hexdigest()
    host = "fixture-simulator"
    common = {
        "platform": platform_name(),
        "architecture": platform.machine().lower(),
        "host": host,
        "observed_at": observed,
        "outcome": "passed",
        "evidence_kind": "fixture",
        "command": "DB6 simulated owner fault-injection suite",
        "evidence": f"suite_source_sha256={source_hash}",
    }
    volume_actions = ("inspect", "check", "repair", "unmount", "mount_read_write")
    volume_rows = []
    volume_path.parent.mkdir(parents=True, exist_ok=True)
    with volume_path.open("w", encoding="utf-8") as stream:
        for action in volume_actions:
            row = {**common, "id": f"db6-{platform_name()}-volume-{action.replace('_', '-')}", "action": action}
            volume_rows.append(row)
            stream.write(json.dumps(row, sort_keys=True) + "\n")
    dbm_rows = []
    dbm_path.parent.mkdir(parents=True, exist_ok=True)
    with dbm_path.open("w", encoding="utf-8") as stream:
        for capability in ("destination_check", "backup", "scratch_restore"):
            row = {**common, "id": f"db6-{platform_name()}-dbm-{capability.replace('_', '-')}", "capability": capability}
            dbm_rows.append(row)
            stream.write(json.dumps(row, sort_keys=True) + "\n")
    payload = {
        "evidence_kind": "fixture",
        "native_qualification": False,
        "source_sha256": source_hash,
        "volume_capabilities": ["volume_identity", "destination_check", "filesystem_repair", "volume_mount"],
        "dbm_capabilities": ["destination_check", "backup", "scratch_restore"],
        "volume_receipt_count": len(volume_actions),
        "dbm_receipt_count": 3,
        "receipt_rows": volume_rows + dbm_rows,
    }
    print("DB6_FIXTURE_RECEIPTS=" + json.dumps(payload, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
