#!/usr/bin/env python3
"""Read issue-level age anchors from canonical owner PROBLEMS.md documents.

Open issue headings must carry an explicit stable ID and opened date, for
example ``## Open — title <!-- pt5-id: GCT-RETENTION; opened: 2026-11-21 -->``.
Canonical ``## Open Issues`` lists can instead annotate each active bullet with
``- <!-- pt5-id: OWNER-ITEM; opened: 2026-11-21 --> ...``. No file modification
timestamp is used as an issue-age substitute.
"""
from __future__ import annotations

from dataclasses import dataclass
from datetime import date
import os
import re
from pathlib import Path

OPEN_HEADING = re.compile(r"^##\s+Open\s+[—–-]\s*(?P<title>.+?)\s*$")
ANCHOR = re.compile(
    r"^(?P<title>.+?)\s*<!--\s*pt5-id:\s*(?P<id>[A-Za-z0-9][A-Za-z0-9._-]*)"
    r";\s*opened:\s*(?P<opened>\d{4}-\d{2}-\d{2})\s*-->\s*$"
)
OPEN_ISSUES_HEADING = "## open issues"
BULLET_ANCHOR = re.compile(
    r"^-\s*<!--\s*pt5-id:\s*(?P<id>[A-Za-z0-9][A-Za-z0-9._-]*)"
    r";\s*opened:\s*(?P<opened>\d{4}-\d{2}-\d{2})\s*-->\s*(?P<body>.*)$"
)
RESOLVED_BULLET = re.compile(r"^-\s*(?:\*\*)?(?:resolved|closed)\b", re.IGNORECASE)


@dataclass(frozen=True)
class OpenIssue:
    item_id: str
    title: str
    opened: date
    age_days: int
    basis: str


def _stable_read(path: Path) -> bytes:
    """Read one consistent file version or fail on a concurrent write."""
    with path.open("rb") as stream:
        descriptor = stream.fileno()
        first = os.fstat(descriptor)
        data = stream.read()
        last = os.fstat(descriptor)
    identity_first = (first.st_dev, first.st_ino, first.st_size, first.st_mtime_ns)
    identity_last = (last.st_dev, last.st_ino, last.st_size, last.st_mtime_ns)
    if identity_first != identity_last or len(data) != last.st_size:
        raise ValueError("PROBLEMS.md changed while issue ages were being read")
    return data


def parse_open_issues(text: str, as_of: date | None = None) -> list[OpenIssue]:
    """Parse only explicitly annotated top-level Open issue headings.

    Historical Corrected/Resolved sections and deferred ideas are not current
    open work. Every active Open heading must have an unambiguous anchor.
    """
    day = as_of or date.today()
    issues: list[OpenIssue] = []
    ids: set[str] = set()
    in_open = False
    in_open_issues = False
    open_sections = 0

    def add(item_id: str, title: str, opened_text: str, line_number: int, basis: str) -> None:
        if item_id in ids:
            raise ValueError(f"duplicate open issue ID {item_id!r}")
        try:
            opened = date.fromisoformat(opened_text)
        except ValueError as error:
            raise ValueError(f"invalid opened date at line {line_number}") from error
        if opened > day:
            raise ValueError(f"open issue {item_id!r} has a future opened date")
        ids.add(item_id)
        issues.append(OpenIssue(item_id, title.strip(), opened, (day - opened).days, basis))

    for line_number, line in enumerate(text.splitlines(), 1):
        if line.startswith("## "):
            match = OPEN_HEADING.fullmatch(line)
            in_open = bool(match)
            in_open_issues = line.lower() == OPEN_ISSUES_HEADING
            if match:
                open_sections += 1
                anchored = ANCHOR.fullmatch(match.group("title"))
                if not anchored:
                    raise ValueError(f"open issue heading at line {line_number} lacks a valid PT5 ID/opened date anchor")
                add(anchored.group("id"), anchored.group("title"), anchored.group("opened"), line_number,
                    "explicit stable item ID and opened date on canonical Open heading")
            elif in_open_issues:
                open_sections += 1
        elif in_open and line.startswith("### "):
            raise ValueError(f"nested issue heading under Open at line {line_number} is ambiguous")
        elif in_open_issues and line.startswith("- "):
            if RESOLVED_BULLET.match(line):
                continue
            anchored = BULLET_ANCHOR.fullmatch(line)
            if not anchored:
                raise ValueError(f"open issue bullet at line {line_number} lacks a valid PT5 ID/opened date anchor")
            add(anchored.group("id"), anchored.group("body"), anchored.group("opened"), line_number,
                "explicit stable item ID and opened date on canonical Open Issues bullet")
    if open_sections == 0:
        raise ValueError("no explicit top-level Open issue sections; cannot infer an empty backlog from dated/history headings")
    return issues


def measure(path: Path, as_of: date | None = None) -> dict:
    """Return issue counts and explicit age anchors for a stable source read."""
    source_ref = path.as_posix()
    return item_age_evidence(_stable_read(path).decode("utf-8"), source_ref, as_of=as_of)


def item_age_evidence(text: str, source_ref: str, as_of: date | None = None) -> dict:
    """Return neutral owner evidence; unrecognized/open-unanchored sources stay unknown."""
    try:
        issues = parse_open_issues(text, as_of=as_of)
    except ValueError as error:
        return {
            "status": "unavailable",
            "eligible_count": None,
            "aged_count": 0,
            "items": [],
            "reason": str(error),
        }
    return {
        "status": "available",
        "eligible_count": len(issues),
        "aged_count": len(issues),
        "items": [
            {"id": issue.item_id, "opened_at": issue.opened.isoformat(),
             "age_days": issue.age_days,
             "source_ref": f"{source_ref}#pt5-id={issue.item_id}",
             "basis": issue.basis}
            for issue in issues
        ],
    }
