#!/usr/bin/env python3
"""Fail closed when the BAS rehabilitation campaign is not safe to continue.

This is a campaign-cadence guard, not product qualification. Runtime evidence
inside archive/ is deliberately excluded from the live working-set budget.
"""

import argparse
import json
from pathlib import Path


SCENARIO = Path(__file__).resolve().parents[2]
PROGRESS = SCENARIO / "docs/internal/REFRACTOR_PROGRESS.md"
EVIDENCE = SCENARIO / ".vrooli/runtime/rehabilitation-evidence"
PACKET = (
    SCENARIO / "docs/internal/REFRACTOR_GOAL.md",
    SCENARIO / "docs/internal/TESTING.md",
    PROGRESS,
    SCENARIO / "docs/internal/OPERATOR_FEEDBACK.md",
)
PACKET_LIMIT = 96 * 1024
PROGRESS_LIMIT = 16 * 1024
EVIDENCE_FILE_LIMIT = 128
EVIDENCE_BYTE_LIMIT = 16 * 1024 * 1024
EVIDENCE_LINE_LIMIT = 250_000


def live_evidence_files(root=EVIDENCE):
    if not root.exists():
        return []
    return sorted(
        path for path in root.rglob("*")
        if path.is_file() and "archive" not in path.relative_to(root).parts
    )


def text_line_count(path):
    data = path.read_bytes()
    if b"\0" in data:
        return 0
    try:
        data.decode("utf-8")
    except UnicodeDecodeError:
        return 0
    return data.count(b"\n") + bool(data and not data.endswith(b"\n"))


def inspect(stage="resume"):
    errors = []
    progress = PROGRESS.read_text()
    packet_bytes = sum(path.stat().st_size for path in PACKET)
    active_epochs = progress.count("## Active candidate epoch")
    epoch_summaries = progress.count("## Epoch summary")

    if packet_bytes > PACKET_LIMIT:
        errors.append(f"active packet is {packet_bytes} bytes; limit is {PACKET_LIMIT}")
    if PROGRESS.stat().st_size > PROGRESS_LIMIT:
        errors.append(
            f"active progress is {PROGRESS.stat().st_size} bytes; limit is {PROGRESS_LIMIT}"
        )
    if active_epochs != 1:
        errors.append(f"progress must contain exactly one active candidate epoch; found {active_epochs}")
    if epoch_summaries > 2:
        errors.append(f"progress may retain at most two epoch summaries; found {epoch_summaries}")
    if "## Historical candidate" in progress or "## Historical stopped candidate" in progress:
        errors.append("historical candidate narratives belong in the compressed archive")

    evidence = live_evidence_files()
    evidence_bytes = sum(path.stat().st_size for path in evidence)
    evidence_lines = sum(text_line_count(path) for path in evidence)
    if len(evidence) > EVIDENCE_FILE_LIMIT:
        errors.append(f"live evidence has {len(evidence)} files; limit is {EVIDENCE_FILE_LIMIT}")
    if evidence_bytes > EVIDENCE_BYTE_LIMIT:
        errors.append(f"live evidence is {evidence_bytes} bytes; limit is {EVIDENCE_BYTE_LIMIT}")
    if evidence_lines > EVIDENCE_LINE_LIMIT:
        errors.append(f"live evidence has {evidence_lines} text lines; limit is {EVIDENCE_LINE_LIMIT}")

    candidate_state = "unknown"
    for state in ("implementation", "frozen", "reopened"):
        if f"Candidate state: **{state}**" in progress:
            candidate_state = state
            break
    if stage in ("producer", "qualify") and candidate_state != "frozen":
        errors.append(f"{stage} requires `Candidate state: **frozen**`; found {candidate_state}")
    if stage == "qualify" and "Qualification cycle: **unused**" not in progress:
        errors.append("qualification requires `Qualification cycle: **unused**`")

    return {
        "status": "passed" if not errors else "failed",
        "stage": stage,
        "campaign_only": True,
        "candidate_state": candidate_state,
        "active_packet": {"bytes": packet_bytes, "limit": PACKET_LIMIT},
        "progress": {
            "bytes": PROGRESS.stat().st_size,
            "limit": PROGRESS_LIMIT,
            "active_epochs": active_epochs,
            "epoch_summaries": epoch_summaries,
        },
        "live_evidence": {
            "files": len(evidence),
            "bytes": evidence_bytes,
            "text_lines": evidence_lines,
            "limits": {
                "files": EVIDENCE_FILE_LIMIT,
                "bytes": EVIDENCE_BYTE_LIMIT,
                "text_lines": EVIDENCE_LINE_LIMIT,
            },
        },
        "errors": errors,
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--stage", choices=("resume", "producer", "qualify"), default="resume")
    args = parser.parse_args()
    result = inspect(args.stage)
    print(json.dumps(result, indent=2))
    raise SystemExit(result["status"] != "passed")


if __name__ == "__main__":
    main()
