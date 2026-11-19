#!/usr/bin/env python3
"""Collect evidence or build a Hub research report from a supervisor review."""
from __future__ import annotations

import argparse
import json
import re
import sys
import hashlib
import datetime as dt
import urllib.parse
from pathlib import Path

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parent))
import mission_check as mc
import evidence_record

HUB_REPORT_SCHEMA = Path(__file__).resolve().parents[4] / "notification-hub/skills/notification-hub/kit/schemas/research-report.schema.json"


def source(path: Path, root: Path) -> dict:
    try:
        label = str(path.relative_to(root))
    except ValueError:
        label = str(path)
    try:
        text = path.read_text(encoding="utf-8")
    except FileNotFoundError:
        return {"path": label, "status": "unavailable", "text": None}
    return {"path": label, "status": "available", "text": text}


def files(directory: Path, root: Path, patterns: tuple[str, ...]) -> dict:
    if not directory.is_dir():
        return {"path": str(directory.relative_to(root)), "status": "unavailable", "items": []}
    paths = sorted({p for pattern in patterns for p in directory.rglob(pattern) if p.is_file()})
    return {"path": str(directory.relative_to(root)), "status": "available", "items": [source(p, root) for p in paths]}


def run_records(directory: Path, root: Path) -> dict:
    """Read top-level run narratives and nested checkpoints, not evidence trees."""
    if not directory.is_dir():
        return {"path": str(directory.relative_to(root)), "status": "unavailable", "items": []}
    paths = {p for p in directory.glob("*.md") if p.is_file()}
    paths.update(p for p in directory.rglob("*.md") if p.is_file() and "checkpoint" in p.name.lower())
    return {"path": str(directory.relative_to(root)), "status": "available", "items": [source(p, root) for p in sorted(paths)]}


def collect(topic: Path, commission: str) -> dict:
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]{0,127}", commission) or commission in (".", ".."):
        raise ValueError("invalid commission identifier")
    topic = topic.resolve()
    if not topic.is_dir():
        raise ValueError("topic directory does not exist")
    mission = source(topic / "MISSION.md", topic)
    sections = mc.split_sections(mission["text"] or "")
    scorecard = [
        {"dimension": row[0], "today": row[1], "target": row[2]}
        for row in mc.table_rows(sections.get("Scorecard", "")) if len(row) >= 3
    ]
    records = []
    evidence_root = topic / "runs" / commission / "evidence"
    if evidence_root.is_dir():
        for record_path in sorted(evidence_root.glob("*/evidence.json")):
            try:
                record = json.loads(record_path.read_text(encoding="utf-8"))
                record["_record_path"] = str(record_path.relative_to(topic))
                records.append(record)
            except (OSError, UnicodeError, json.JSONDecodeError) as exc:
                records.append({"id": record_path.parent.name, "_record_path": str(record_path.relative_to(topic)),
                                "_error": str(exc)})
    config_path = topic / "reporting" / "report.json"
    try:
        config = json.loads(config_path.read_text(encoding="utf-8"))
    except FileNotFoundError:
        config = {}
    return {
        "topic": topic.name,
        "topic_path": str(topic),
        "commission": commission,
        "report_config": config,
        "evidence": records,
        "scorecard": scorecard,
        "sources": {
            "mission": mission,
            "work_plan": source(topic / "work/README.md", topic),
            "ledger": source(topic / "LEDGER.md", topic),
            "inbox": source(topic / "INBOX.md", topic),
            "commission_state": source(topic.parent.parent / "cohorts" / commission / "STATE.md", topic),
            "feedback": files(topic / "feedback", topic, ("*.md", "*.json")),
            "checkpoints": files(topic / "runs" / commission, topic, ("*CHECKPOINT*.md", "*checkpoint*.md")),
            "run_records": run_records(topic / "runs" / commission, topic),
        },
    }


def validate_review(scorecard: list[dict], review: dict) -> None:
    """Require explicit, complete judgments; never estimate progress from counts."""
    judgments = review.get("scorecard") if isinstance(review, dict) else None
    if not scorecard or not isinstance(judgments, list):
        raise ValueError("scorecard and supervisor judgments are required")
    dimensions = [row["dimension"] for row in scorecard]
    names = []
    for row in judgments:
        if not isinstance(row, dict) or not all(isinstance(row.get(k), str) and row[k].strip() for k in ("dimension", "level", "reason")):
            raise ValueError("each scorecard judgment requires dimension, level and reason")
        names.append(row["dimension"])
    if len(names) != len(set(names)) or set(names) != set(dimensions):
        raise ValueError("scorecard judgments must cover each dimension exactly once")


def short(value: str, limit: int) -> str:
    """Normalize a short line and reject prose that needs editorial shortening."""
    value = " ".join(value.split())
    if len(value) <= limit:
        return value
    raise ValueError(f"report line is {len(value)} characters; maximum is {limit}: {value[:120]}")


STATE_LABELS = {
    "landed": ("Landed", "good"),
    "running": ("In progress", "accent"),
    "parked": ("Parked", "warn"),
    "waiting-time": ("Waiting on time", "warn"),
    "waiting-operator": ("Waiting on you", "crit"),
    "waiting-on-you": ("Waiting on you", "crit"),
    "stopped": ("Stopped", "warn"),
}


def state_display(key: str, label: str | None = None) -> tuple[str, str]:
    """Return a human label and tone for known state keys."""
    if key in STATE_LABELS:
        return STATE_LABELS[key]
    dated = re.fullmatch(r"waiting-time-(\d{4})-(\d{2})-(\d{2})", key)
    if dated:
        month = dt.date(int(dated[1]), int(dated[2]), int(dated[3])).strftime("%b")
        return f"Resumes {int(dated[3])} {month}", "warn"
    display = " ".join((label or key).replace("_", "-").split("-"))
    return display[:1].upper() + display[1:], "accent"


def validate_lede_speedups(payload: dict) -> None:
    """Reject numeric speedup claims that disagree with plotted time medians."""
    lede = payload.get("meta", {}).get("lede", "")
    claims = [float(value) for value in re.findall(r"(\d+(?:\.\d+)?)\s*[×x]\s*faster", lede, re.IGNORECASE)]
    if not claims:
        return
    expected = []
    for item in payload.get("measurements", {}).get("items", []):
        values = item.get("values", [])
        unit = item.get("unit", "").lower()
        if len(values) < 2 or unit not in ("ms", "s", "sec", "seconds"):
            continue
        first, last = values[0].get("value"), values[-1].get("value")
        if not isinstance(first, (int, float)) or not isinstance(last, (int, float)) or min(first, last) <= 0:
            continue
        ratio = max(first, last) / min(first, last)
        expected.append(round(ratio, 1))
    if not expected or any(round(claim, 1) not in expected for claim in claims):
        raise ValueError(f"meta.lede speedup claims {claims} do not match plotted time medians {expected}")


def build(data: dict, review: dict) -> dict:
    """Map the topic's scorecard and plan plus supervisor-reviewed sections to the kit."""
    validate_review(data["scorecard"], review)
    meta = review.get("meta")
    if not isinstance(meta, dict) or not isinstance(meta.get("title"), str) or not meta["title"].strip():
        raise ValueError("supervisor review requires meta.title")
    judgments = {row["dimension"]: row for row in review["scorecard"]}
    dimensions = []
    for row in data["scorecard"]:
        judgment = judgments[row["dimension"]]
        today = review.get("todayByDimension", {}).get(row["dimension"], row["today"])
        title = re.sub(r"\s*\(added \d{4}-\d{2}-\d{2}\)$", "", row["dimension"])
        dimensions.append({"key": "d-" + hashlib.sha256(row["dimension"].encode()).hexdigest()[:12],
                           "title": short(title, 80), "today": short(today, 90),
                           "level": short(judgment["level"], 40), "reason": short(judgment["reason"], 90)})
    output = {"template": "research-report", "meta": {**meta, "topic": data["topic"]},
              "commission": {**review.get("commission", {}), "id": data["commission"]},
              "scorecard": {"source": {"status": "partial", "label": "MISSION.md and supervisor review",
                                       "note": "Qualitative supervisor assessment; numerical progress unavailable."},
                            "judgedBy": "Research supervisor", "dimensions": dimensions}}
    plan = data["sources"]["work_plan"]
    if plan["text"] and "workstreams" not in review:
        match = re.search(r"(?m)^\| Brief \|.*(?:\n\|.*)+", plan["text"])
        if match:
            items, states = [], {}
            for row in mc.table_rows(match.group()):
                if len(row) < 4:
                    continue
                ident = re.search(r"\[([^]]+)\]", row[0])
                if not ident or not re.fullmatch(r"[A-Za-z0-9_.-]+", ident.group(1)):
                    continue
                state = row[2].strip(" `")
                state_key = re.sub(r"[^A-Za-z0-9_.-]+", "-", state).strip("-")
                if not state_key:
                    continue
                label, tone = state_display(state_key, state)
                states[state_key] = {"key": state_key, "label": label, "tone": tone}
                items.append({"id": ident.group(1), "title": short(row[1], 80), "state": state_key, "note": short(row[3], 90)})
            if items:
                output["states"] = list(states.values())
                output["workstreams"] = {"source": {"status": "partial", "label": "work/README.md",
                    "note": "States are from the work plan; prior weights are unavailable unless reviewed."}, "items": items}
    # Only the supervisor can synthesize measurements and what feedback changed. These
    # are accepted explicitly, never inferred from free text or manufactured from counts.
    for key in ("states", "summary", "measurements", "workstreams", "feedback", "next", "technical", "asks"):
        if key in review:
            if key in ("summary", "measurements", "workstreams", "feedback", "next"):
                section = review[key]
                provenance = section.get("source") if isinstance(section, dict) else None
                if not isinstance(provenance, dict) or provenance.get("status") not in ("real", "partial", "unavailable"):
                    raise ValueError(f"{key} requires explicit source status")
                if provenance["status"] != "unavailable" and not str(provenance.get("label", "")).strip():
                    raise ValueError(f"{key} requires a source label")
            output[key] = review[key]
    # Normalize reviewer state vocabulary too; raw storage keys are never presentation text.
    if isinstance(output.get("states"), list):
        normalized = []
        for state in output["states"]:
            key = state.get("key", "")
            label, tone = state_display(key, state.get("label"))
            normalized.append({"key": key, "label": label, "tone": tone})
        output["states"] = normalized
    elif isinstance(output.get("workstreams"), dict):
        vocabulary = {}
        for item in output["workstreams"].get("items", []):
            key = item.get("state")
            if key and key not in vocabulary:
                label, tone = state_display(key)
                vocabulary[key] = {"key": key, "label": label, "tone": tone}
        if vocabulary:
            output["states"] = list(vocabulary.values())
    if isinstance(output.get("commission"), dict):
        commission = output["commission"]
        for field in ("startedAt", "endedAt"):
            value = commission.get(field)
            if isinstance(value, str):
                commission[field] = value.replace("Z", "+00:00") if value.endswith("Z") else value
    if isinstance(output.get("asks"), list):
        for ask in output["asks"]:
            if not ask.get("unblocks") and ask.get("workstream"):
                ask["unblocks"] = short(f"Unblocks {ask['workstream']} work", 60)
    for key in ("summary", "measurements", "workstreams", "feedback", "next"):
        if key not in output:
            output[key] = {"source": {"status": "unavailable", "note": "Supervisor review for this section is unavailable."}}
    output["evidence"] = []
    for row in data.get("evidence", []):
        if row.get("_error"):
            raise ValueError(f"invalid evidence record {row.get('_record_path')}: {row['_error']}")
        record = {k: v for k, v in row.items() if not k.startswith("_")}
        workstreams = review.get("evidenceWorkstreams", {})
        if "workstream" not in record and isinstance(workstreams, dict) and record.get("id") in workstreams:
            record["workstream"] = workstreams[record["id"]]
        if not isinstance(record.get("workstream"), str) or not record["workstream"].strip():
            raise ValueError(f"evidence record {record.get('id', '<unknown>')} requires a workstream mapping")
        record = evidence_record.prepare(record)
        evidence_record.validate(record)
        output["evidence"].append(record)
    config = data.get("report_config", {})
    configured_viewers = config.get("viewers") if isinstance(config, dict) else None
    if configured_viewers:
        output["evidence"] = [record for record in output["evidence"] if record.get("format") in configured_viewers]
    selected = set(config.get("sections", [])) if isinstance(config, dict) and config.get("sections") else None
    if isinstance(config, dict) and config.get("asks") and not output.get("asks") and (selected is None or "waiting" in selected):
        output["asks"] = config["asks"]
    receipts = []
    if data.get("topic_path"):
        for source_path in sorted((Path(data["topic_path"]) / "feedback").glob("hub-*-items.json")):
            try:
                items_data = json.loads(source_path.read_text(encoding="utf-8"))
                for item in items_data.get("items", []):
                    if item.get("status") in ("applied", "planned", "declined"):
                        receipts.append({"id": item["id"], "title": item.get("key", "Operator note")[:80],
                                         "status": item["status"], "text": item.get("text", "")[:90],
                                         "reason": item.get("reason", "")[:90]})
            except (OSError, ValueError, KeyError, TypeError):
                continue
    if receipts and (selected is None or "feedback" in selected):
        output["feedbackReceipts"] = receipts
    section_fields = {"waiting": ("asks",), "changes": ("summary",), "evidence": ("evidence",),
                      "scorecard": ("scorecard",), "workstreams": ("workstreams",),
                      "feedback": ("feedback", "feedbackReceipts"), "next": ("next",)}
    if selected is not None:
        retained = {field for section in selected for field in section_fields.get(section, ())}
        for field in tuple(output):
            if field in {item for fields in section_fields.values() for item in fields} and field not in retained:
                output.pop(field, None)
    validate_lede_speedups(output)
    return output


def validate_payload(payload: dict, schema_path: Path = HUB_REPORT_SCHEMA) -> None:
    """Validate the produced Hub payload and report concise JSON-path diagnostics."""
    try:
        import jsonschema
    except ImportError as exc:
        raise ValueError("schema validation requires Python package jsonschema") from exc
    schema = json.loads(schema_path.read_text(encoding="utf-8"))
    # The kit schema currently uses a relative $id; anchor its local #/$defs refs to
    # the actual file so validators do not try to resolve a doubled remote URL.
    if not urllib.parse.urlparse(str(schema.get("$id", ""))).scheme:
        schema["$id"] = schema_path.resolve().as_uri()
    try:
        jsonschema.Draft202012Validator(schema).validate(payload)
    except jsonschema.ValidationError as exc:
        path = "$" + "".join(f"[{part}]" if isinstance(part, int) else f".{part}" for part in exc.absolute_path)
        raise ValueError(f"Hub report schema violation at {path}: {exc.message}") from exc


def write_history(topic: Path, commission: str, report: dict, receipt: dict, answer: dict | None = None,
                  now: dt.datetime | None = None) -> Path:
    """Persist an immutable report event and update its latest pointer."""
    stamp = (now or dt.datetime.now(dt.timezone.utc)).astimezone(dt.timezone.utc).strftime("%Y-%m-%dT%H%M%SZ")
    safe = re.sub(r"[^A-Za-z0-9._-]+", "-", commission).strip(".-") or "commission"
    base = topic / "reporting" / "history" / f"{stamp}-{safe}"
    if base.exists():
        suffix = 2
        while base.with_name(base.name + f"-{suffix}").exists(): suffix += 1
        base = base.with_name(base.name + f"-{suffix}")
    base.mkdir(parents=True)
    for name, payload in (("report.json", report), ("receipt.json", receipt), ("answer.json", answer or {})):
        (base / name).write_text(json.dumps(payload, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    latest = {"path": str(base.relative_to(topic)), "commission": commission, "created_at": stamp}
    (topic / "reporting" / "latest.json").write_text(json.dumps(latest, indent=2) + "\n", encoding="utf-8")
    return base


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("topic", type=Path)
    parser.add_argument("--commission", required=True)
    parser.add_argument("--output", type=Path)
    parser.add_argument("--review", type=Path, help="supervisor review JSON; build Hub payload")
    parser.add_argument("--receipt", type=Path, help="delivery receipt JSON; also write immutable reporting history")
    parser.add_argument("--answer", type=Path, help="Hub answer JSON for reporting history")
    parser.add_argument("--validate-schema", type=Path, default=HUB_REPORT_SCHEMA,
                        help="validate built output against this Hub template schema (default: current repository schema)")
    args = parser.parse_args()
    try:
        result = collect(args.topic, args.commission)
        if args.review:
            review = json.loads(args.review.read_text(encoding="utf-8"))
            result = build(result, review)
            validate_payload(result, args.validate_schema)
        if args.receipt:
            receipt = json.loads(args.receipt.read_text(encoding="utf-8"))
            answer = json.loads(args.answer.read_text(encoding="utf-8")) if args.answer else {}
            write_history(args.topic.resolve(), args.commission, result, receipt, answer)
        payload = json.dumps(result, indent=2, ensure_ascii=False) + "\n"
        if args.output:
            args.output.write_text(payload, encoding="utf-8")
        else:
            print(payload, end="")
    except (ValueError, OSError, UnicodeError) as exc:
        print(f"research report: {exc}", file=sys.stderr)
        return 2
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
