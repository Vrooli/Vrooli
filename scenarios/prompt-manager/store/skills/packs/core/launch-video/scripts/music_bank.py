#!/usr/bin/env python3
"""Audit and lint exact-hash launch-video music-bank selections.

The portable bank manifest is a provenance/policy overlay. Music Library remains
the source for exact-hash approval and accepted-use history; callers provide its
JSON export so this tool does not depend on a database or live service.
"""
from __future__ import annotations

import argparse
import datetime
import hashlib
import os
import json
import re
import subprocess
from pathlib import Path


VERSION = 1
BPM_RANGE = re.compile(r"(\d{2,3})\s*[–-]\s*(\d{2,3})\s*BPM", re.IGNORECASE)


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for block in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(block)
    return digest.hexdigest()


def read_json(path: Path):
    return json.loads(path.read_text(encoding="utf-8"))


def resolve(base: Path, value: str) -> Path:
    path = Path(value)
    return (path if path.is_absolute() else base / path).resolve()


def probe_audio(path: Path) -> dict:
    result = subprocess.run(
        ["ffprobe", "-v", "error", "-select_streams", "a:0", "-show_entries",
         "format=duration,size:stream=codec_name,sample_rate,channels,bits_per_sample,bits_per_raw_sample",
         "-of", "json", str(path)], capture_output=True, text=True, check=False)
    if result.returncode:
        raise ValueError(f"ffprobe failed for {path}: {result.stderr[-400:]}")
    data = json.loads(result.stdout)
    streams = data.get("streams", [])
    if not streams:
        raise ValueError(f"no audio stream: {path}")
    stream = streams[0]
    return {
        "duration_seconds": float(data.get("format", {}).get("duration", 0)),
        "size_bytes": int(data.get("format", {}).get("size", 0)),
        "codec": stream.get("codec_name"),
        "sample_rate_hz": int(stream["sample_rate"]) if stream.get("sample_rate") else None,
        "channels": int(stream["channels"]) if stream.get("channels") else None,
        "bits_per_sample": int(stream.get("bits_per_raw_sample") or stream.get("bits_per_sample") or 0) or None,
    }


def catalog_tracks(catalog: dict) -> dict:
    tracks = catalog.get("tracks")
    if not isinstance(tracks, list):
        raise ValueError("Music Library export must contain a tracks array")
    return {row.get("sha256"): row for row in tracks if isinstance(row, dict) and row.get("sha256")}


def audit(bank_path: Path, catalog_path: Path) -> dict:
    bank = read_json(bank_path)
    catalog = read_json(catalog_path)
    library = catalog_tracks(catalog)
    errors = []
    audited = []
    for item in bank.get("tracks", []):
        track_errors = []
        digest = item.get("sha256", "")
        record = library.get(digest)
        if not re.fullmatch(r"[0-9a-f]{64}", digest):
            track_errors.append("bank track SHA-256 is malformed")
        if not record:
            track_errors.append("exact hash is absent from the Music Library export")
        elif record.get("approval") != item.get("operator_approval"):
            track_errors.append("bank approval differs from the exact-hash Music Library row")
        source = resolve(bank_path.parent, item.get("source_path", ""))
        receipt = resolve(bank_path.parent, item.get("generation_receipt", ""))
        if not source.is_file():
            track_errors.append(f"source file missing: {source}")
        else:
            source_hash = sha256_file(source)
            if source_hash != digest:
                track_errors.append("source bytes do not match the bank SHA-256")
            if record and Path(record.get("source_path", "")).resolve() != source:
                track_errors.append("Music Library source path differs from the bank source path")
        receipt_hash = None
        receipt_data = {}
        if not receipt.is_file():
            track_errors.append(f"generation receipt missing: {receipt}")
        else:
            receipt_hash = sha256_file(receipt)
            receipt_data = read_json(receipt)
            if receipt_data.get("status") != "completed":
                track_errors.append("generation receipt is not completed")
            if receipt_data.get("output", {}).get("sha256") != digest:
                track_errors.append("generation receipt output hash differs from the bank hash")
            if source.is_file() and receipt_data.get("output", {}).get("bytes") is not None:
                if int(receipt_data["output"]["bytes"]) != source.stat().st_size:
                    track_errors.append("generation receipt output size differs from source file")
            if record and Path(record.get("receipt_path", "")).resolve() != receipt:
                track_errors.append("Music Library receipt path differs from the bank receipt path")
        audio = {}
        if source.is_file():
            try:
                audio = probe_audio(source)
                receipt_duration = receipt_data.get("output", {}).get("duration_seconds")
                if receipt_duration is not None and abs(float(receipt_duration) - audio["duration_seconds"]) > 0.02:
                    track_errors.append("generation receipt output duration differs from ffprobe")
            except (OSError, ValueError, json.JSONDecodeError) as exc:
                track_errors.append(str(exc))
        prompt = receipt_data.get("effective_request", {}).get("prompt", "")
        if not prompt:
            request = item.get("request_path")
            if request:
                prompt = read_json(resolve(bank_path.parent, request)).get("caption", "")
        tempo_match = BPM_RANGE.search(prompt or "")
        tempo_intent = [int(tempo_match.group(1)), int(tempo_match.group(2))] if tempo_match else None
        measurement_path = resolve(bank_path.parent, item.get("energy_measurement", {}).get("path", ""))
        measurement = {}
        if measurement_path.is_file():
            measurement_data = read_json(measurement_path)
            measurement = measurement_data.get(item.get("energy_measurement", {}).get("key", ""), {})
            if measurement.get("source_sha256") != digest:
                track_errors.append("energy measurement source hash differs from bank hash")
        else:
            track_errors.append(f"energy measurement receipt missing: {measurement_path}")
        processing = measurement.get("source_measurement", {})
        usage = (record or {}).get("uses", [])
        accepted_projects = sorted({use.get("project", "") for use in usage if use.get("status") == "accepted"})
        usage_state = "used" if accepted_projects else ("draft" if usage else "available")
        tags = (record or {}).get("tags", [])
        row = {
            "id": item.get("id"), "sha256": digest, "title": item.get("title"),
            "source_path": str(source), "size_bytes": audio.get("size_bytes"),
            "audio": audio, "prompt_tempo_intent_bpm": tempo_intent,
            "measured_tempo_bpm": None,
            "tempo_note": "Prompt tempo is intent only; no measured pulse claim is made.",
            "energy_measurement": {
                "integrated_loudness_lufs": processing.get("input_i"),
                "true_peak_dbfs": processing.get("input_tp"),
                "loudness_range_lu": processing.get("input_lra"),
                "receipt_path": str(measurement_path),
            },
            "music_library": {
                "approval": (record or {}).get("approval"), "feedback": (record or {}).get("feedback"),
                "tags": tags, "take_id": (record or {}).get("take_id"),
                "uses": usage, "accepted_use_projects": accepted_projects,
                "use_state": usage_state, "reservation_state": "not_in_catalog_export",
            },
            "hype_classification": item.get("hype_classification"),
            "rights_provenance": {
                "model_id": receipt_data.get("model", {}).get("model_id"),
                "model_revision": receipt_data.get("model", {}).get("model_revision"),
                "model_license": receipt_data.get("model", {}).get("license"),
                "terms_review": "not_assessed_for_public_or_commercial_use",
                "allowed_bank_scope": item.get("allowed_scope"),
            },
            "generation_receipt": {
                "path": str(receipt), "sha256": receipt_hash,
                "operation_id": receipt_data.get("operation_id"),
                "request_sha256": receipt_data.get("request_sha256"),
            },
            "errors": track_errors,
        }
        if track_errors:
            errors.extend(f"{item.get('id')}: {error}" for error in track_errors)
        audited.append(row)
    return {
        "schema": 1, "bank": str(bank_path.resolve()), "bank_sha256": sha256_file(bank_path),
        "catalog_export": str(catalog_path.resolve()), "catalog_sha256": sha256_file(catalog_path),
        "catalog_snapshot_mtime_utc": datetime.datetime.fromtimestamp(
            catalog_path.stat().st_mtime, tz=datetime.timezone.utc).isoformat(),
        "tracks": audited, "errors": errors, "status": "pass" if not errors else "fail",
        "limitations": [
            "Music Library state is a fixed export; its file timestamp is not live reservation state.",
            "Local format and loudness facts are measurements, not music quality or hype verdicts.",
            "Prompt BPM ranges are not measured playback tempo.",
            "Recorded model terms do not establish rights for public or commercial distribution.",
            "Music Library draft use is distinct from accepted use; preserve both states.",
        ],
    }


def lint(bank_path: Path, catalog_path: Path, selection_path: Path) -> dict:
    receipt = audit(bank_path, catalog_path)
    selection = read_json(selection_path)
    digest = selection.get("track_sha256")
    candidate = next((item for item in receipt["tracks"] if item["sha256"] == digest), None)
    reasons = []
    if selection.get("classification") != "hype":
        reasons.append("selection is explicitly classified as non-hype")
    if not candidate:
        reasons.append("selected exact hash is not in the bank")
    elif candidate["errors"]:
        reasons.append("selected bank record fails source/provenance audit")
    elif candidate["hype_classification"].get("tag") != "hype":
        reasons.append("selected bank record lacks the hype tag")
    elif candidate["music_library"].get("approval") != "liked":
        reasons.append("selected exact hash is not operator-liked in Music Library")
    elif candidate["music_library"]["accepted_use_projects"] and selection.get("project") not in candidate["music_library"]["accepted_use_projects"]:
        reasons.append("selected track already has an accepted use in another project")
    if candidate and candidate["music_library"].get("tags") is not None:
        required_tags = {"instrumental", "launch-video"}
        missing_tags = sorted(required_tags.difference(candidate["music_library"].get("tags", [])))
        if missing_tags:
            reasons.append("Music Library row lacks required tags: " + ", ".join(missing_tags))
    if candidate and selection.get("scope") != candidate["rights_provenance"].get("allowed_bank_scope"):
        reasons.append("selection scope differs from this track's allowed bank scope")
    if selection.get("scope") != "private-review":
        reasons.append("this bank only admits private review; public/commercial rights are unqualified")
    result = {
        "schema": 1, "selection": str(selection_path.resolve()),
        "track_sha256": digest, "project": selection.get("project"),
        "scope": selection.get("scope"), "classification": selection.get("classification"),
        "status": "pass" if not reasons and receipt["status"] == "pass" else "refused",
        "reasons": reasons + (receipt["errors"] if receipt["status"] != "pass" else []),
        "track": candidate, "bank_audit_sha256": hashlib.sha256(
            json.dumps(receipt, sort_keys=True, separators=(",", ":")).encode()).hexdigest(),
        "limitations": ["Hype is an editorial bank tag based on source brief and observed prior operator liking, not a new taste judgment.",
                        "A pass authorizes this private assembly path only; it does not clear model rights for public or commercial use."],
    }
    return result


def query(bank_path: Path, catalog_path: Path, tags=None, approval=None, unused=False, available=False) -> dict:
    """Select exact-hash bank rows using snapshot approval, tags, and use state."""
    receipt = audit(bank_path, catalog_path)
    if receipt["status"] != "pass":
        return {
            "schema": 1, "bank": receipt["bank"], "catalog_export": receipt["catalog_export"],
            "bank_sha256": receipt["bank_sha256"], "catalog_sha256": receipt["catalog_sha256"],
            "status": "fail", "filters": {"tags": sorted(set(tags or [])), "approval": approval,
                                               "unused": unused, "available": available},
            "tracks": [], "errors": receipt["errors"],
            "use_policy": "No query results are exposed when exact-hash audit fails.",
        }
    wanted = set(tags or [])
    selected = []
    for row in receipt["tracks"]:
        library = row["music_library"]
        available_tags = set(library.get("tags") or [])
        overlay_tag = row.get("hype_classification", {}).get("tag")
        if overlay_tag:
            available_tags.add(overlay_tag)
        if approval and library.get("approval") != approval:
            continue
        if wanted and not wanted.issubset(available_tags):
            continue
        # Match Music Library's state model: accepted means used; a draft is
        # still unused, but not available for automatic reservation selection.
        if unused and library.get("accepted_use_projects"):
            continue
        if available and library.get("uses"):
            continue
        selected.append(row)
    return {
        "schema": 1, "bank": receipt["bank"], "catalog_export": receipt["catalog_export"],
        "bank_sha256": receipt["bank_sha256"], "catalog_sha256": receipt["catalog_sha256"],
        "status": receipt["status"], "filters": {"tags": sorted(wanted), "approval": approval, "unused": unused, "available": available},
        "tracks": selected, "errors": receipt["errors"],
        "use_policy": "unused excludes accepted uses per Music Library; available additionally excludes draft uses. Reservation leases are not present in a catalog export.",
    }


def retag(bank_path: Path, track_sha256: str, classification: str, basis: str, output_path: Path) -> dict:
    """Write a new manifest revision with one exact-hash editorial classification."""
    if classification not in ("hype", "non-hype", "unclassified"):
        raise ValueError("classification must be hype, non-hype, or unclassified")
    if not basis.strip():
        raise ValueError("a classification basis is required")
    if bank_path.resolve() == output_path.resolve():
        raise ValueError("write a new manifest revision; do not overwrite the current bank")
    if output_path.exists():
        raise ValueError("tag output already exists; choose a new revision path")
    bank = read_json(bank_path)
    matches = [row for row in bank.get("tracks", []) if row.get("sha256") == track_sha256]
    if len(matches) != 1:
        raise ValueError("classification requires one exact hash present in the bank")
    # Rebase portable references from the old manifest directory to the new one.
    for row in bank.get("tracks", []):
        for key in ("source_path", "generation_receipt", "request_path"):
            row[key] = os.path.relpath(resolve(bank_path.parent, row[key]), output_path.resolve().parent)
        energy = row.get("energy_measurement", {})
        energy["path"] = os.path.relpath(resolve(bank_path.parent, energy["path"]), output_path.resolve().parent)
    matches[0]["hype_classification"] = {"tag": classification, "basis": basis.strip()}
    return bank


def write_json(path: Path, data: dict) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    subparsers = parser.add_subparsers(dest="command", required=True)
    for name in ("audit", "lint", "query"):
        command = subparsers.add_parser(name)
        command.add_argument("--bank", required=True, type=Path)
        command.add_argument("--catalog", required=True, type=Path, help="Music Library catalog list JSON export")
        command.add_argument("--out", required=True, type=Path, help="new or refreshed receipt JSON path")
        if name == "lint":
            command.add_argument("--selection", required=True, type=Path)
        if name == "query":
            command.add_argument("--tag", action="append", default=[])
            command.add_argument("--approval", choices=("liked", "rejected", "unreviewed"))
            command.add_argument("--unused", action="store_true", help="exclude accepted uses, matching Music Library")
            command.add_argument("--available", action="store_true", help="also exclude any draft use; match default no-use reservation candidates")
    tag = subparsers.add_parser("tag", help="write a new manifest revision for one exact-hash editorial tag")
    tag.add_argument("--bank", required=True, type=Path)
    tag.add_argument("--track-sha256", required=True)
    tag.add_argument("--classification", required=True, choices=("hype", "non-hype", "unclassified"))
    tag.add_argument("--basis", required=True)
    tag.add_argument("--out", required=True, type=Path)
    args = parser.parse_args()
    try:
        if args.command == "audit":
            result = audit(args.bank, args.catalog)
        elif args.command == "lint":
            result = lint(args.bank, args.catalog, args.selection)
        elif args.command == "query":
            result = query(args.bank, args.catalog, args.tag, args.approval, args.unused, args.available)
        else:
            result = retag(args.bank, args.track_sha256, args.classification, args.basis, args.out)
        write_json(args.out, result)
        status = "pass" if args.command == "tag" else result.get("status", "pass")
        print(json.dumps({"status": status, "tracks": len(result.get("tracks", [])), "out": str(args.out)}))
        raise SystemExit(0 if status == "pass" else 1)
    except (OSError, UnicodeError, ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
        parser.error(str(exc))


if __name__ == "__main__":
    main()
