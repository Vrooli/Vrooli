#!/usr/bin/env python3
"""Discover and count operator-caught UX defects against completed UI reviews.

The script reads native local transcripts through harvest-operator-corrections.py.
It never sends transcript text to a service. Discovery outputs belong in the private
EX1 run directory; publication counts only explicitly adjudicated, review-linked
defects. Repeated source messages count once when they share a defect_key.
"""
from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import os
import re
import sys
from collections import Counter, defaultdict
from datetime import datetime
from pathlib import Path

HOME = Path.home()
PROJECT = HOME / "Vrooli"
DEFAULT_PRIVATE = HOME / ".vrooli/research/topics/experience-engineering/runs/steward-raise-ux-tooling-20261112/private/forward-measure"
HARVESTER = PROJECT / "scenarios/ui-health/scripts/harvest-operator-corrections.py"

spec = importlib.util.spec_from_file_location("operator_corrections", HARVESTER)
if spec is None or spec.loader is None:
    raise RuntimeError(f"cannot load local transcript adapters: {HARVESTER}")
harvester = importlib.util.module_from_spec(spec)
spec.loader.exec_module(harvester)


def read_json(path: Path) -> dict:
    with path.open(encoding="utf-8") as stream:
        value = json.load(stream)
    if not isinstance(value, dict):
        raise ValueError(f"expected a JSON object in {path}")
    return value


def parse_time(value: str, label: str) -> datetime:
    try:
        parsed = datetime.fromisoformat(value.replace("Z", "+00:00"))
    except (TypeError, ValueError) as exc:
        raise ValueError(f"{label} must be an ISO-8601 timestamp with timezone") from exc
    if parsed.tzinfo is None:
        raise ValueError(f"{label} must include a timezone")
    return parsed


def source_inventory() -> dict:
    """Report discoverable source coverage; counts never imply unavailable=empty."""
    paths = {
        "claude": HOME / ".claude/projects/-home-matthalloran8-Vrooli",
        "codex": HOME / ".codex/sessions/2026",
        "web-console": HOME / ".vrooli/data/vrooli/web-console/web-console.db",
        "agent-manager": HOME / ".vrooli/data/vrooli/agent-manager/legacy-history/agent-manager.db",
    }
    inventory = {}
    for name, path in paths.items():
        if name == "claude":
            files = sorted(path.glob("*.jsonl")) if path.is_dir() else []
            state = "readable" if files else ("empty" if path.is_dir() else "unavailable")
            inventory[name] = {"state": state, "files_seen": len(files)}
        elif name == "codex":
            files = sorted(path.glob("**/rollout-*.jsonl")) if path.is_dir() else []
            inventory[name] = {"state": "readable" if files else ("empty" if path.is_dir() else "unavailable"), "files_seen": len(files)}
        else:
            if not path.is_file():
                inventory[name] = {"state": "unavailable", "files_seen": 0}
                continue
            try:
                import sqlite3
                db = sqlite3.connect(f"file:{path}?mode=ro", uri=True)
                expected = "conversation_events" if name == "web-console" else "run_events"
                present = db.execute("SELECT 1 FROM sqlite_master WHERE type='table' AND name=?", (expected,)).fetchone()
                db.close()
                inventory[name] = {"state": "readable" if present else "schema_missing", "files_seen": 1}
            except Exception:
                inventory[name] = {"state": "unreadable", "files_seen": 1}
    return inventory


def candidate_tier(text: str, scenario: str) -> str | None:
    if not isinstance(text, str) or len(text) > 10000:
        return None
    if harvester.SENSITIVE.search(text) or harvester.QUOTE.search(text) or harvester.SYNTHETIC.search(text):
        return None
    if not re.search(r"\b" + re.escape(scenario).replace(r"\-", r"[\s-]+") + r"\b", text, re.I):
        return None
    has_ux = bool(harvester.TARGET_UI.search(text))
    has_defect = bool(harvester.OBSERVED_DEFECT.search(text))
    has_correction = bool(harvester.CORRECTION.search(text))
    if has_ux and (has_defect or has_correction):
        return "strong-candidate" if has_defect and has_correction else "ambiguous-candidate"
    return None


def evidence_id(row: dict) -> str:
    fields = (row.get("source", ""), row.get("file", ""), row.get("message_id") or row.get("message", ""), row.get("timestamp", ""))
    return hashlib.sha256("|".join(map(str, fields)).encode()).hexdigest()[:24]


def direct_rows():
    # Adapters read user-role messages from Claude, Codex, Web Console and the
    # optional local Agent Manager import archive. include_all allows this tool to
    # retain correction-like but uncertain mentions as explicit ambiguous candidates.
    for reader in (harvester.read_claude, harvester.read_codex, harvester.read_web_console, harvester.read_agent_manager):
        yield from reader(include_all=True)


def safe_anchor(anchor: dict, label: str) -> dict:
    if not isinstance(anchor, dict) or not anchor.get("uri"):
        raise ValueError(f"{label} requires an object with a non-empty uri")
    uri = str(anchor["uri"])
    if re.search(r"https?://|[?&](?:token|key|secret)=", uri, re.I):
        raise ValueError(f"{label} must be a local or stable evidence reference, not a URL or credential-bearing URI")
    return {"label": str(anchor.get("label", "evidence")), "uri": uri}


def load_review(path: Path) -> dict:
    review = read_json(path)
    for field in ("review_id", "scenario", "status", "review_started_at", "review_ended_at"):
        if not review.get(field):
            raise ValueError(f"review manifest missing {field}")
    if review["status"] != "completed":
        raise ValueError("only completed reviews can have a forward count")
    start, end = parse_time(review["review_started_at"], "review_started_at"), parse_time(review["review_ended_at"], "review_ended_at")
    if end < start:
        raise ValueError("review_ended_at must be at or after review_started_at")
    anchors = review.get("review_evidence", [])
    if not isinstance(anchors, list) or not anchors:
        raise ValueError("review manifest needs at least one review_evidence anchor")
    review["review_evidence"] = [safe_anchor(anchor, "review_evidence") for anchor in anchors]
    review["_start"], review["_end"] = start, end
    return review


def discover(review: dict) -> tuple[list[dict], dict]:
    matches, by_text = [], {}
    total = Counter()
    for row in direct_rows():
        source = row.get("source", "unknown")
        total[source] += 1
        tier = candidate_tier(row.get("text", ""), review["scenario"])
        if not tier:
            continue
        try:
            stamp = parse_time(row.get("timestamp", ""), "transcript timestamp")
        except ValueError:
            continue
        if stamp < review["_start"] or stamp > review["_end"]:
            continue
        normalized = re.sub(r"\W+", " ", row.get("text", "").lower()).strip()
        if not normalized:
            continue
        ident = hashlib.sha256((review["review_id"] + "|" + normalized).encode()).hexdigest()[:24]
        source_anchor = {"source": source, "file": row.get("file", ""), "message": str(row.get("message", "")),
                         "message_id": row.get("message_id", ""), "timestamp": row.get("timestamp", ""), "evidence_id": ident}
        if normalized in by_text:
            by_text[normalized]["source_anchors"].append(source_anchor)
            if tier == "strong-candidate":
                by_text[normalized]["tier"] = tier
            continue
        item = {"evidence_id": ident, "tier": tier, "source_anchors": [source_anchor],
                "text": row.get("text", ""), "initial_status": "unknown"}
        by_text[normalized] = item
        matches.append(item)
    matches.sort(key=lambda item: (item["source_anchors"][0].get("timestamp", ""), item["evidence_id"]))
    sources = Counter(anchor["source"] for item in matches for anchor in item["source_anchors"])
    return matches, {"user_messages_seen_by_adapter": dict(total), "candidate_messages": len(matches), "candidate_sources": dict(sources)}


def write_private(path: Path, value, jsonl=False):
    ensure_private_path(path)
    path.parent.mkdir(mode=0o700, parents=True, exist_ok=True)
    os.chmod(path.parent, 0o700)
    if jsonl:
        payload = "".join(json.dumps(item, ensure_ascii=False) + "\n" for item in value)
    else:
        payload = json.dumps(value, ensure_ascii=False, indent=2) + "\n"
    path.write_text(payload, encoding="utf-8")
    os.chmod(path, 0o600)


def ensure_private_path(path: Path):
    resolved = path.expanduser().resolve()
    try:
        resolved.relative_to(PROJECT.resolve())
    except ValueError:
        return
    raise ValueError(f"private transcript/count data cannot be read from or written inside the repository: {resolved}")


def read_jsonl(path: Path) -> list[dict]:
    rows = []
    with path.open(encoding="utf-8") as stream:
        for line_no, line in enumerate(stream, 1):
            if not line.strip():
                continue
            try:
                item = json.loads(line)
            except ValueError as exc:
                raise ValueError(f"invalid JSONL at {path}:{line_no}") from exc
            if not isinstance(item, dict):
                raise ValueError(f"expected object at {path}:{line_no}")
            rows.append(item)
    return rows


def publish(review: dict, candidates: list[dict], decisions: list[dict], discovery_coverage: dict | None = None) -> dict:
    by_id = {item["evidence_id"]: item for item in candidates}
    seen_decisions = set()
    confirmed, ambiguous, excluded = [], [], []
    for decision in decisions:
        ident = decision.get("evidence_id")
        if ident not in by_id:
            raise ValueError(f"adjudication references unknown evidence_id {ident!r}")
        if decision.get("review_id") != review["review_id"]:
            raise ValueError(f"adjudication for {ident} is linked to a different review")
        if ident in seen_decisions:
            raise ValueError(f"duplicate adjudication for evidence_id {ident}")
        seen_decisions.add(ident)
        outcome = decision.get("decision")
        if outcome not in ("confirmed", "ambiguous", "exclude"):
            raise ValueError(f"invalid decision for {ident}: use confirmed, ambiguous, or exclude")
        anchor = safe_anchor({"label": decision.get("anchor_label", "review item"), "uri": decision.get("review_anchor", "")}, "review_anchor") if outcome != "exclude" else None
        if outcome == "confirmed":
            if not decision.get("defect_key") or not decision.get("defect_class"):
                raise ValueError(f"confirmed item {ident} needs defect_key and defect_class")
            confirmed.append((decision, by_id[ident], anchor))
        elif outcome == "ambiguous":
            ambiguous.append((decision, by_id[ident], anchor))
        else:
            excluded.append((decision, by_id[ident]))
    # Count each defect instance once, even when the operator repeated it or said
    # the same thing in multiple local surfaces. Adjudicators explicitly assign a
    # stable defect_key and can retain all transcript anchors on the counted row.
    grouped = defaultdict(list)
    for decision, item, anchor in confirmed:
        grouped[str(decision["defect_key"])].append((decision, item, anchor))
    counted = []
    for key, group in sorted(grouped.items()):
        first_decision, _, first_anchor = group[0]
        classes = sorted({str(decision["defect_class"]) for decision, _, _ in group})
        counted.append({"defect_key": key, "defect_class": classes[0] if len(classes) == 1 else "needs-class-reconciliation",
                        "review_id": review["review_id"], "review_evidence": review["review_evidence"],
                        "review_anchor": first_anchor,
                        "transcript_anchors": [anchor for _, item, _ in group for anchor in item.get("source_anchors", [])],
                        "deduplicated_source_mentions": len(group)})
    unknown = len(by_id) - len(seen_decisions)
    source_counts = Counter(anchor["source"] for item in candidates for anchor in item.get("source_anchors", []))
    source_state = source_inventory()
    required_sources = ("claude", "codex", "web-console")
    degraded_sources = [name for name in required_sources if source_state[name]["state"] != "readable"]
    if source_state["agent-manager"]["state"] not in ("readable", "unavailable", "empty"):
        degraded_sources.append("agent-manager")
    class_conflicts = any(len({str(decision["defect_class"]) for decision, _, _ in group}) > 1 for group in grouped.values())
    complete = unknown == 0 and not ambiguous and not degraded_sources and not class_conflicts
    return {
        "schema_version": 1,
        "metric": "operator_caught_defects_per_ui_review",
        "review_id": review["review_id"], "scenario": review["scenario"],
        "review_status": review["status"], "review_started_at": review["review_started_at"],
        "review_ended_at": review["review_ended_at"],
        "count": len(counted), "count_status": "complete" if complete else "provisional",
        "counted_defects": counted,
        "coverage": {"candidate_messages": len(candidates), "confirmed_mentions": len(confirmed),
                     "unique_confirmed_defects": len(counted), "ambiguous_mentions": len(ambiguous),
                     "unreviewed_unknown_mentions": unknown, "excluded_mentions": len(excluded),
                     "candidate_messages_by_source": dict(source_counts), "source_availability": source_state,
                     "degraded_sources": degraded_sources, "defect_class_conflicts": class_conflicts,
                     "adjudicated_candidate_fraction": round(len(seen_decisions) / len(by_id), 4) if by_id else 1.0},
        "discovery_coverage": discovery_coverage or {"source_availability_at_discovery": "not-retained"},
        "limitations": [
            "Only explicit adjudications count; discovery phrases are candidate retrieval, not a semantic verdict.",
            "Review-window matching may miss feedback delivered later; link delayed feedback with a separate completed-review manifest/window.",
            "Defect deduplication depends on adjudicators assigning the same defect_key to repeated mentions of one issue.",
            "Unavailable or unreadable transcript sources reduce coverage and are reported, never treated as zero mentions.",
        ],
    }


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    discover_parser = sub.add_parser("discover", help="scan local transcripts for review-window candidates")
    discover_parser.add_argument("--review", type=Path, required=True, help="completed review manifest JSON")
    discover_parser.add_argument("--output", type=Path, default=DEFAULT_PRIVATE / "candidates.jsonl")
    publish_parser = sub.add_parser("publish", help="publish adjudicated count with review and transcript anchors")
    publish_parser.add_argument("--review", type=Path, required=True)
    publish_parser.add_argument("--candidates", type=Path, required=True)
    publish_parser.add_argument("--adjudications", type=Path, required=True, help="JSONL: evidence_id, review_id, decision; confirmed also needs defect_key/class/review_anchor")
    publish_parser.add_argument("--output", type=Path, default=DEFAULT_PRIVATE / "published-count.json")
    args = parser.parse_args()
    try:
        for path_arg in (args.review, getattr(args, "candidates", None), getattr(args, "adjudications", None), args.output):
            if path_arg is not None:
                ensure_private_path(path_arg)
        review = load_review(args.review)
        if args.command == "discover":
            candidates, coverage = discover(review)
            coverage["source_availability"] = source_inventory()
            write_private(args.output, candidates, jsonl=True)
            coverage_path = args.output.with_suffix(".coverage.json")
            write_private(coverage_path, coverage)
            print(json.dumps({"review_id": review["review_id"], "candidate_messages": len(candidates),
                              "coverage": coverage, "source_availability": source_inventory(),
                              "private_output": str(args.output), "coverage_output": str(coverage_path)}))
        else:
            coverage_path = args.candidates.with_suffix(".coverage.json")
            discovery_coverage = read_json(coverage_path) if coverage_path.exists() else None
            report = publish(review, read_jsonl(args.candidates), read_jsonl(args.adjudications), discovery_coverage)
            write_private(args.output, report)
            print(json.dumps({"review_id": report["review_id"], "count": report["count"],
                              "count_status": report["count_status"], "coverage": report["coverage"],
                              "private_output": str(args.output)}))
        return 0
    except (OSError, ValueError) as exc:
        print(f"forward-count: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
