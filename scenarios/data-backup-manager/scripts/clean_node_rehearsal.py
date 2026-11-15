#!/usr/bin/env python3
"""Restore an explicit DBM snapshot set on a replacement node and retain a receipt.

The manifest contains identifiers and destination paths only. Credential values
are imported by the credential authority on the replacement node before this
runner starts; this process never reads or exports secret material.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import tempfile
from datetime import datetime, timezone


def fail(message: str) -> None:
    raise ValueError(message)


def validate_restore_location(raw: str) -> None:
    path = Path(raw)
    if not path.is_absolute():
        fail(f"restore location must be absolute: {raw}")
    try:
        info = path.lstat()
    except FileNotFoundError:
        return
    if stat.S_ISLNK(info.st_mode) or not stat.S_ISDIR(info.st_mode):
        fail(f"restore location exists and is not an empty directory: {raw}")
    try:
        with os.scandir(path) as entries:
            if next(entries, None) is not None:
                fail(f"restore location is not empty: {raw}")
    except OSError as exc:
        fail(f"cannot inspect restore location {raw}: {exc}")


def load_manifest(path: Path) -> dict:
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        fail(f"cannot read rehearsal manifest: {exc}")
    if not isinstance(data, dict) or data.get("schema_version") != 1:
        fail("manifest schema_version must be 1")
    unknown = set(data) - {"schema_version", "rehearsal_id", "source_generation", "credential_recovery_receipt", "destinations", "targets", "scenarios"}
    if unknown:
        fail(f"manifest contains unsupported fields: {', '.join(sorted(unknown))}")
    for key in ("rehearsal_id", "source_generation"):
        if not isinstance(data.get(key), str) or not data[key].strip():
            fail(f"manifest requires {key}")
    receipt = data.get("credential_recovery_receipt")
    if not isinstance(receipt, dict) or not isinstance(receipt.get("path"), str) or not isinstance(receipt.get("sha256"), str):
        fail("manifest requires credential_recovery_receipt.path and sha256")
    if set(receipt) != {"path", "sha256"}:
        fail("credential_recovery_receipt accepts only path and sha256")
    if len(receipt["sha256"]) != 64 or any(c not in "0123456789abcdef" for c in receipt["sha256"].lower()):
        fail("credential recovery receipt sha256 must be a 64-character hex digest")
    targets = data.get("targets")
    if not isinstance(targets, list) or not targets:
        fail("manifest targets must be a non-empty list")
    for index, target in enumerate(targets):
        if not isinstance(target, dict):
            fail(f"targets[{index}] must be an object")
        if set(target) != {"owner", "name", "kind", "locator", "destination_id", "snapshot_id", "restore_location"}:
            fail(f"targets[{index}] contains unsupported fields")
        for key in ("owner", "name", "kind", "locator", "destination_id", "snapshot_id", "restore_location"):
            if not isinstance(target.get(key), str) or not target[key].strip():
                fail(f"targets[{index}] requires {key}")
        if "," in target["snapshot_id"]:
            fail(f"targets[{index}].snapshot_id cannot contain a comma")
        location = Path(target["restore_location"])
        if not location.is_absolute():
            fail(f"targets[{index}].restore_location must be absolute")
        validate_restore_location(target["restore_location"])
    destinations = data.get("destinations")
    if not isinstance(destinations, list) or not destinations:
        fail("manifest destinations must be a non-empty list")
    destination_ids = set()
    for index, destination in enumerate(destinations):
        if not isinstance(destination, dict) or set(destination) != {"destination_id", "name", "backend", "location", "created_at", "cap_bytes", "cap_policy"}:
            fail(f"destinations[{index}] must contain non-secret destination catalog metadata")
        for key in ("destination_id", "name", "backend", "location", "created_at", "cap_policy"):
            if not isinstance(destination.get(key), str) or not destination[key].strip():
                fail(f"destinations[{index}] requires {key}")
        if destination["backend"] not in ("filesystem", "sftp", "s3"):
            fail(f"destinations[{index}].backend must be filesystem, sftp, or s3")
        if not isinstance(destination.get("cap_bytes"), int) or destination["cap_bytes"] < 0:
            fail(f"destinations[{index}].cap_bytes must be a non-negative integer")
        if destination["cap_policy"] not in ("alert-block", "alert-only"):
            fail(f"destinations[{index}].cap_policy must be alert-block or alert-only")
        if destination["backend"] == "filesystem" and not Path(destination["location"]).is_absolute():
            fail(f"destinations[{index}].location must be absolute")
        if destination["destination_id"] in destination_ids:
            fail(f"destinations[{index}] repeats destination_id")
        destination_ids.add(destination["destination_id"])
    referenced_destinations = {target["destination_id"] for target in targets}
    if not referenced_destinations.issubset(destination_ids):
        fail("every target destination_id must match an imported destination")
    if referenced_destinations != destination_ids:
        fail("each imported destination must contain at least one selected recovery snapshot")
    restore_paths = [Path(target["restore_location"]).resolve(strict=False) for target in targets]
    for i, left in enumerate(restore_paths):
        for right in restore_paths[i + 1:]:
            if left == right or left in right.parents or right in left.parents:
                fail("target restore locations must be separate non-overlapping paths")
    scenarios = data.get("scenarios", [])
    if not isinstance(scenarios, list) or not scenarios or any(not isinstance(s, str) or not s.strip() for s in scenarios):
        fail("scenarios must be a non-empty list of scenario names to start and health-check")
    return data


def run(argv: list[str], *, capture: bool = True) -> subprocess.CompletedProcess[str]:
    return subprocess.run(argv, check=False, text=True, stdout=subprocess.PIPE if capture else None,
                          stderr=subprocess.PIPE if capture else None, env=os.environ.copy())


def atomic_json(path: Path, value: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    fd, tmp_name = tempfile.mkstemp(prefix=f".{path.name}.", dir=path.parent)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as stream:
            json.dump(value, stream, indent=2, sort_keys=True)
            stream.write("\n")
            stream.flush()
            os.fsync(stream.fileno())
        os.chmod(tmp_name, 0o600)
        os.replace(tmp_name, path)
    finally:
        if os.path.exists(tmp_name):
            os.unlink(tmp_name)


def parse_restore_outcome(proc: subprocess.CompletedProcess[str], *, expected_status: str, target_id: str,
                         destination_id: str, snapshot_id: str, verify: bool) -> dict:
    if proc.returncode:
        fail(f"DBM restore command exited {proc.returncode}")
    try:
        payload = json.loads(proc.stdout)
    except (TypeError, json.JSONDecodeError) as exc:
        fail(f"DBM restore command returned invalid JSON: {exc}")
    restore = payload.get("restore") if isinstance(payload, dict) else None
    if not isinstance(restore, dict):
        fail("DBM restore response is missing its restore record")
    for field, expected in (("target_id", target_id), ("destination_id", destination_id), ("snapshot_id", snapshot_id)):
        if restore.get(field) != expected:
            fail(f"DBM restore response {field} does not match the requested recovery item")
    if restore.get("status") != expected_status or not isinstance(restore.get("id"), str) or not restore["id"]:
        fail(f"DBM restore did not reach {expected_status}")
    outcome = {"restore_id": restore["id"], "status": restore["status"]}
    if verify:
        checksum = restore.get("checksum")
        if not isinstance(checksum, str) or len(checksum) != 64 or any(c not in "0123456789abcdef" for c in checksum.lower()):
            fail("DBM scratch verification did not return a SHA-256 checksum")
        outcome["checksum_sha256"] = checksum.lower()
    return outcome


def execute(manifest_path: Path, receipt_path: Path) -> dict:
    manifest = load_manifest(manifest_path)
    if not receipt_path.is_absolute():
        fail("rehearsal receipt path must be absolute")
    if receipt_path.exists() or receipt_path.is_symlink():
        fail("rehearsal receipt path already exists; choose a new durable receipt path")
    resolved_receipt = receipt_path.resolve(strict=False)
    resolved_manifest = manifest_path.resolve(strict=True)
    if resolved_receipt == resolved_manifest:
        fail("rehearsal receipt cannot overwrite its manifest")
    for target in manifest["targets"]:
        validate_restore_location(target["restore_location"])
        restore_path = Path(target["restore_location"]).resolve(strict=False)
        if resolved_receipt == restore_path or resolved_receipt in restore_path.parents or restore_path in resolved_receipt.parents:
            fail("rehearsal receipt path must be separate from restore targets")
        for destination in manifest["destinations"]:
            if destination["backend"] == "filesystem":
                bundle_path = Path(destination["location"]).resolve(strict=False)
                if restore_path == bundle_path or restore_path in bundle_path.parents or bundle_path in restore_path.parents:
                    fail("restore targets must be separate from filesystem destination bundles")
    credential_receipt_path = Path(manifest["credential_recovery_receipt"]["path"])
    receipt_bytes = credential_receipt_path.read_bytes()
    receipt_sha = hashlib.sha256(receipt_bytes).hexdigest()
    if receipt_sha.lower() != manifest["credential_recovery_receipt"]["sha256"].lower():
        fail("credential recovery receipt digest does not match the manifest")
    try:
        credential_receipt = json.loads(receipt_bytes)
    except json.JSONDecodeError as exc:
        fail(f"credential recovery receipt is not JSON: {exc}")
    if not isinstance(credential_receipt, dict) or credential_receipt.get("status") != "verified":
        fail("credential recovery receipt must report status=verified")
    if credential_receipt.get("source_generation") != manifest["source_generation"]:
        fail("credential recovery receipt source_generation does not match manifest")

    result = {
        "schema_version": 1,
        "rehearsal_id": manifest["rehearsal_id"],
        "source_generation": manifest["source_generation"],
        "started_at": datetime.now(timezone.utc).isoformat(),
        "credential_recovery_receipt_sha256": receipt_sha,
        "credential_authority": "replacement-node receipt only; no old-host secret read",
        "targets": [],
        "destinations": [],
        "scenarios": [],
        "status": "running",
    }
    atomic_json(receipt_path, result)
    try:
        snapshots_by_destination = {}
        for target in manifest["targets"]:
            ids = snapshots_by_destination.setdefault(target["destination_id"], [])
            if target["snapshot_id"] not in ids:
                ids.append(target["snapshot_id"])
        for destination in manifest["destinations"]:
            expected_snapshot_ids = snapshots_by_destination.get(destination["destination_id"], [])
            proc = run([
                "data-backup-manager", "destinations", "import-existing",
                "--id", destination["destination_id"], "--name", destination["name"],
                "--backend", destination["backend"], "--location", destination["location"],
                "--created-at", destination["created_at"], "--snapshot-ids", ",".join(expected_snapshot_ids),
                "--cap-bytes", str(destination["cap_bytes"]), "--cap-policy", destination["cap_policy"], "--json",
            ])
            try:
                imported = json.loads(proc.stdout).get("destination", {})
            except (TypeError, json.JSONDecodeError):
                imported = {}
            row = {"destination_id": destination["destination_id"], "exit_code": proc.returncode}
            if proc.returncode or imported.get("id") != destination["destination_id"]:
                row["status"] = "failed"
                result["destinations"].append(row)
                fail(f"existing destination import failed or identity did not match ({destination['destination_id']})")
            row["status"] = "attached"
            result["destinations"].append(row)
            atomic_json(receipt_path, result)

        target_ids = {}
        for target in manifest["targets"]:
            proc = run([
                "data-backup-manager", "targets", "register", "--owner", target["owner"],
                "--name", target["name"], "--kind", target["kind"], "--locator", target["locator"], "--json",
            ])
            try:
                registered = json.loads(proc.stdout).get("target", {})
            except (TypeError, json.JSONDecodeError):
                registered = {}
            target_id = registered.get("id")
            if proc.returncode or not isinstance(target_id, str) or not target_id:
                fail(f"target registration failed for {target['owner']}/{target['name']}")
            target_ids[(target["owner"], target["name"])] = target_id

        for target in manifest["targets"]:
            target_id = target_ids[(target["owner"], target["name"])]
            row = {"target_id": target_id, "destination_id": target["destination_id"], "snapshot_id": target["snapshot_id"]}
            row["restore_location"] = target["restore_location"]
            verified = run([
                "data-backup-manager", "restores", "verify",
                "--target", target_id, "--destination", target["destination_id"],
                "--snapshot", target["snapshot_id"], "--json",
            ])
            row["verify_exit_code"] = verified.returncode
            try:
                row["verification"] = parse_restore_outcome(
                    verified, expected_status="RESTORE_STATUS_VERIFIED", target_id=target_id,
                    destination_id=target["destination_id"], snapshot_id=target["snapshot_id"], verify=True,
                )
            except ValueError as exc:
                row["status"] = "snapshot_verification_failed"
                result["targets"].append(row)
                atomic_json(receipt_path, result)
                fail(f"scratch verification failed for target {target_id}: {exc}")
            proc = run([
                "data-backup-manager", "restores", "restore",
                "--target", target_id, "--destination", target["destination_id"],
                "--snapshot", target["snapshot_id"], "--location", target["restore_location"], "--json",
            ])
            row["exit_code"] = proc.returncode
            try:
                row["restore"] = parse_restore_outcome(
                    proc, expected_status="RESTORE_STATUS_RESTORED", target_id=target_id,
                    destination_id=target["destination_id"], snapshot_id=target["snapshot_id"], verify=False,
                )
            except ValueError as exc:
                row["status"] = "failed"
                result["targets"].append(row)
                atomic_json(receipt_path, result)
                fail(f"restore failed for target {target_id}: {exc}")
            row["status"] = "restored"
            result["targets"].append(row)
            atomic_json(receipt_path, result)

        for name in manifest.get("scenarios", []):
            row = {"scenario": name}
            started = run(["vrooli", "scenario", "start", name, "--json"])
            row["start_exit_code"] = started.returncode
            if started.returncode:
                row["status"] = "failed_to_start"
                result["scenarios"].append(row)
                fail(f"scenario {name} failed to start")
            waited = run(["vrooli", "scenario", "wait", name, "--json"])
            row["wait_exit_code"] = waited.returncode
            if waited.returncode:
                row["status"] = "failed_to_become_ready"
                result["scenarios"].append(row)
                fail(f"scenario {name} did not become ready")
            status = run(["vrooli", "scenario", "status", name, "--json"])
            row["status_exit_code"] = status.returncode
            if status.returncode:
                row["status"] = "health_check_failed"
                result["scenarios"].append(row)
                fail(f"scenario {name} health status failed")
            try:
                status_payload = json.loads(status.stdout)
            except (TypeError, json.JSONDecodeError):
                row["status"] = "invalid_health_response"
                result["scenarios"].append(row)
                fail(f"scenario {name} returned invalid JSON health status")
            scenario_status = status_payload.get("scenario", {}) if isinstance(status_payload, dict) else {}
            if not isinstance(scenario_status, dict) or scenario_status.get("status") != "running" or scenario_status.get("health_status") != "healthy":
                row["status"] = "unhealthy"
                result["scenarios"].append(row)
                fail(f"scenario {name} is not running and healthy")
            row["health_status"] = "healthy"
            row["status"] = "healthy"
            result["scenarios"].append(row)
            atomic_json(receipt_path, result)
        result["status"] = "verified"
    except (OSError, ValueError) as exc:
        result["status"] = "failed"
        result["failure"] = str(exc)
        result["finished_at"] = datetime.now(timezone.utc).isoformat()
        atomic_json(receipt_path, result)
        raise
    result["finished_at"] = datetime.now(timezone.utc).isoformat()
    result["manifest_sha256"] = hashlib.sha256(manifest_path.read_bytes()).hexdigest()
    atomic_json(receipt_path, result)
    return result


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--manifest", required=True, type=Path)
    parser.add_argument("--receipt", required=True, type=Path, help="durable JSON rehearsal receipt")
    args = parser.parse_args()
    try:
        result = execute(args.manifest, args.receipt)
    except (OSError, ValueError) as exc:
        print(f"clean-node rehearsal failed: {exc}", file=sys.stderr)
        return 1
    print(json.dumps({"status": result["status"], "receipt": str(args.receipt)}, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
