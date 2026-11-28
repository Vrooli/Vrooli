#!/usr/bin/env python3
"""Permission and residency gate for the Marketing presence voice corpus.

This tool does not collect, import, embed, print, or send corpus text. The
operator alone edits consent.json after responding to the existing MP-DAY0
voice-permission card. Corpus paths are eligible only under ~/private/presence.
"""

from __future__ import annotations

import argparse
import json
import os
import stat
import sys
from datetime import datetime
from pathlib import Path
from typing import Any


PRIVATE_ROOT = Path.home() / "private" / "presence" / "MP3"
REPO_ROOT = Path(__file__).resolve().parents[3]
ALLOWED_SOURCES = {"published_posts", "linkedin_writing", "dictated_messages"}
CONSENT_FILE = "consent.json"
CORPUS_DIR = "corpus"


class GateError(Exception):
    pass


def _inside(child: Path, parent: Path) -> bool:
    try:
        child.relative_to(parent)
        return True
    except ValueError:
        return False


def _check_residency(root: Path) -> None:
    private_parent = Path.home() / "private" / "presence"
    if private_parent.is_symlink():
        raise GateError("private_presence_symlink")
    if root.is_symlink():
        raise GateError("private_root_symlink")
    resolved = root.resolve(strict=False)
    private_parent = private_parent.resolve(strict=False)
    repo = REPO_ROOT.resolve(strict=True)
    if not _inside(resolved, private_parent):
        raise GateError("outside_private_presence")
    if _inside(resolved, repo):
        raise GateError("inside_repository")


def _check_private_directory(path: Path, *, create: bool = False) -> None:
    if create:
        path.mkdir(mode=0o700, parents=True, exist_ok=True)
    info = path.lstat()
    if not stat.S_ISDIR(info.st_mode):
        raise GateError("directory_not_regular")
    if info.st_uid != os.getuid():
        raise GateError("directory_owner_mismatch")
    if stat.S_IMODE(info.st_mode) & 0o077:
        raise GateError("directory_permissions_too_open")


def _read_consent(path: Path) -> dict[str, Any]:
    info = path.lstat()
    if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid():
        raise GateError("consent_file_not_owned_regular_file")
    if stat.S_IMODE(info.st_mode) & 0o077:
        raise GateError("consent_file_permissions_too_open")
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeError, json.JSONDecodeError) as exc:
        raise GateError("consent_file_invalid") from exc
    if not isinstance(value, dict) or value.get("schema_version") != 1:
        raise GateError("consent_schema_invalid")
    if value.get("decision") not in {"pending", "no", "yes"}:
        raise GateError("consent_decision_invalid")
    sources = value.get("allowed_sources")
    if not isinstance(sources, list) or any(not isinstance(item, str) for item in sources):
        raise GateError("consent_sources_invalid")
    if len(set(sources)) != len(sources) or set(sources) - ALLOWED_SOURCES:
        raise GateError("consent_sources_invalid")
    if value["decision"] == "yes":
        decided_at = value.get("decided_at")
        if not isinstance(decided_at, str) or not decided_at.strip() or not sources:
            raise GateError("affirmative_consent_incomplete")
        try:
            timestamp = datetime.fromisoformat(decided_at.replace("Z", "+00:00"))
            if timestamp.tzinfo is None or timestamp.utcoffset() is None:
                raise ValueError("timestamp needs a timezone")
        except ValueError as exc:
            raise GateError("consent_timestamp_invalid") from exc
        if value.get("card_ref") != "MP-DAY0" or value.get("confirmed_by") != "operator":
            raise GateError("consent_provenance_invalid")
    elif value["decision"] == "no":
        decided_at = value.get("decided_at")
        if sources or not isinstance(decided_at, str) or not decided_at.strip():
            raise GateError("negative_consent_incomplete")
        try:
            timestamp = datetime.fromisoformat(decided_at.replace("Z", "+00:00"))
            if timestamp.tzinfo is None or timestamp.utcoffset() is None:
                raise ValueError("timestamp needs a timezone")
        except ValueError as exc:
            raise GateError("consent_timestamp_invalid") from exc
        if value.get("card_ref") != "MP-DAY0" or value.get("confirmed_by") != "operator":
            raise GateError("consent_provenance_invalid")
    elif sources or value.get("decided_at") is not None or value.get("confirmed_by") is not None:
        raise GateError("pending_consent_has_decision_fields")
    return value


def _corpus_inventory(path: Path, allowed: bool) -> tuple[int, int]:
    count = size = 0
    for item in path.iterdir():
        info = item.lstat()
        if not stat.S_ISREG(info.st_mode) or item.is_symlink():
            raise GateError("corpus_contains_nonregular_entry")
        if info.st_uid != os.getuid():
            raise GateError("corpus_file_owner_mismatch")
        if stat.S_IMODE(info.st_mode) & 0o077:
            raise GateError("corpus_file_permissions_too_open")
        count += 1
        size += info.st_size
    if count and not allowed:
        raise GateError("corpus_exists_without_consent")
    return count, size


def initialize(root: Path) -> dict[str, Any]:
    _check_residency(root)
    root.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    _check_private_directory(root.parent)
    _check_private_directory(root, create=True)
    corpus = root / CORPUS_DIR
    _check_private_directory(corpus, create=True)
    consent = root / CONSENT_FILE
    if not consent.exists():
        payload = {
            "schema_version": 1,
            "decision": "pending",
            "allowed_sources": [],
            "decided_at": None,
            "confirmed_by": None,
            "card_ref": "MP-DAY0",
        }
        descriptor = os.open(consent, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        with os.fdopen(descriptor, "w", encoding="utf-8") as stream:
            json.dump(payload, stream, indent=2)
            stream.write("\n")
            stream.flush()
            os.fsync(stream.fileno())
    os.chmod(root, 0o700)
    os.chmod(corpus, 0o700)
    return status(root)


def status(root: Path) -> dict[str, Any]:
    _check_residency(root)
    _check_private_directory(root)
    _check_private_directory(root.parent)
    allowed_entries = {CONSENT_FILE, CORPUS_DIR}
    if any(entry.name not in allowed_entries for entry in root.iterdir()):
        raise GateError("unexpected_private_root_entry")
    _check_private_directory(root / CORPUS_DIR)
    consent = _read_consent(root / CONSENT_FILE)
    allowed = consent["decision"] == "yes"
    count, total_bytes = _corpus_inventory(root / CORPUS_DIR, allowed)
    if consent["decision"] == "no":
        state = "permission_denied"
    else:
        state = "allowed" if allowed else "waiting_for_operator_permission"
    return {
        "status": state,
        "allowed": allowed,
        "sources": consent["allowed_sources"] if allowed else [],
        "corpus_file_count": count,
        "corpus_bytes": total_bytes,
        "text_read_or_emitted": False,
        "repository_storage": False,
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("init", "status"))
    args = parser.parse_args()
    try:
        result = initialize(PRIVATE_ROOT) if args.command == "init" else status(PRIVATE_ROOT)
    except (GateError, OSError) as exc:
        code = str(exc) if isinstance(exc, GateError) else "private_store_unavailable"
        print(json.dumps({"status": "blocked", "reason": code, "text_read_or_emitted": False}))
        return 2
    print(json.dumps(result, sort_keys=True))
    return 0 if result["allowed"] else 2


if __name__ == "__main__":
    sys.exit(main())
