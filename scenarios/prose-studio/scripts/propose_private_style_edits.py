#!/usr/bin/env python3
"""Write a private, unapproved style suggestion from one edited candidate.

The operation is disabled until the existing MP-DAY0 voice permission is
affirmative. Input, output and any exact edit text must stay under
~/private/presence/MP3. It never applies a style change or prints text.
"""

from __future__ import annotations

import argparse
import difflib
import hashlib
import json
import os
import re
import stat
import sys
import uuid
from pathlib import Path

from presence_corpus_gate import CORPUS_DIR, GateError, PRIVATE_ROOT, status as corpus_status


MAX_BYTES = 512 * 1024
OPAQUE_ID = re.compile(r"^[A-Za-z0-9_-]{1,100}$")


def _private_text(path: Path) -> str:
    if path.is_symlink():
        raise GateError("input_symlink")
    resolved = path.resolve(strict=True)
    if not resolved.is_relative_to((PRIVATE_ROOT / CORPUS_DIR).resolve(strict=True)):
        raise GateError("input_outside_private_corpus")
    info = resolved.lstat()
    if not stat.S_ISREG(info.st_mode) or info.st_uid != os.getuid():
        raise GateError("input_not_owned_regular_file")
    if stat.S_IMODE(info.st_mode) & 0o077:
        raise GateError("input_permissions_too_open")
    if info.st_size > MAX_BYTES:
        raise GateError("input_too_large")
    try:
        return resolved.read_text(encoding="utf-8")
    except (OSError, UnicodeError) as exc:
        raise GateError("input_unreadable_text") from exc


def _write_private(path: Path, payload: dict) -> None:
    directory = path.parent
    directory.mkdir(mode=0o700, parents=True, exist_ok=True)
    if directory.is_symlink() or not directory.resolve().is_relative_to(PRIVATE_ROOT.resolve(strict=True)):
        raise GateError("output_outside_private_presence")
    os.chmod(directory, 0o700)
    raw = json.dumps(payload, ensure_ascii=False, indent=2).encode("utf-8") + b"\n"
    descriptor = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(descriptor, "wb") as stream:
        stream.write(raw)
        stream.flush()
        os.fsync(stream.fileno())


def propose(candidate_path: Path, edited_path: Path, card_id: str, candidate_id: str) -> dict:
    if not OPAQUE_ID.fullmatch(card_id) or not OPAQUE_ID.fullmatch(candidate_id):
        raise GateError("ids_must_be_opaque")
    consent = corpus_status(PRIVATE_ROOT)
    if not consent["allowed"]:
        raise GateError("operator_voice_permission_required")

    original = _private_text(candidate_path)
    edited = _private_text(edited_path)
    original_lines = original.splitlines()
    edited_lines = edited.splitlines()
    matcher = difflib.SequenceMatcher(a=original_lines, b=edited_lines, autojunk=False)
    additions: list[str] = []
    removals: list[str] = []
    for opcode, i1, i2, j1, j2 in matcher.get_opcodes():
        if opcode in {"replace", "delete"}:
            removals.extend(line for line in original_lines[i1:i2] if line.strip())
        if opcode in {"replace", "insert"}:
            additions.extend(line for line in edited_lines[j1:j2] if line.strip())

    proposal_id = uuid.uuid4().hex
    output_path = PRIVATE_ROOT / "style-proposals" / f"{proposal_id}.json"
    payload = {
        "schema_version": 1,
        "id": proposal_id,
        "card_id": card_id,
        "candidate_id": candidate_id,
        "candidate_sha256": hashlib.sha256(original.encode("utf-8")).hexdigest(),
        "edited_sha256": hashlib.sha256(edited.encode("utf-8")).hexdigest(),
        "suggestions": {
            "possible_exemplars": additions,
            "possible_anti_patterns": removals,
        },
        "operator_confirmation": "pending",
        "apply_to_style": False,
        "storage": "~/private/presence/MP3/style-proposals",
    }
    _write_private(output_path, payload)
    return {
        "status": "proposal_pending_operator_confirmation",
        "proposal_id": proposal_id,
        "suggestion_count": len(additions) + len(removals),
        "operator_confirmation": "pending",
        "applied": False,
        "text_printed": False,
        "path": str(output_path),
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("candidate", type=Path)
    parser.add_argument("edited", type=Path)
    parser.add_argument("--card-id", required=True)
    parser.add_argument("--candidate-id", required=True)
    args = parser.parse_args()
    try:
        result = propose(args.candidate, args.edited, args.card_id, args.candidate_id)
    except (GateError, OSError) as exc:
        reason = str(exc) if isinstance(exc, GateError) else "private_store_unavailable"
        print(json.dumps({"status": "blocked", "reason": reason, "text_printed": False}))
        return 2
    print(json.dumps(result, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main())
