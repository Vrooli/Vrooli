#!/usr/bin/env python3
"""Sync the registered research report answers to a topic's files.

Each reporting/history/*/receipt.json or legacy reports/*.json delivery receipt must contain ask_id. Sync reads every receipt,
including old open reports that may have been answered since the previous commission.
The full Hub response is retained; interpretation and applying it to briefs remain the
supervisor's responsibility. A failed read is an error, not proof of no feedback.
"""
from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import re
import subprocess
import sys
from pathlib import Path


def sync(topic: Path) -> list[str]:
    errors = []
    reports = topic / "reports"
    legacy = []
    if reports.exists() and not reports.is_dir():
        return [f"{reports}: legacy report receipt history is not a directory"]
    if reports.is_dir():
        legacy = sorted(path for path in reports.iterdir() if path.suffix == ".json")
    history = topic / "reporting" / "history"
    receipts = sorted(history.glob("*/receipt.json")) if history.is_dir() else []
    # Preserve each old receipt in reporting/history and retain its source path.
    for old in legacy:
        try:
            record = json.loads(old.read_text(encoding="utf-8"))
            ask_id = record.get("ask_id")
            if not isinstance(ask_id, str) or not ask_id:
                raise ValueError("ask_id must be a nonempty string")
            if not any(json.loads(p.read_text(encoding="utf-8")).get("legacy_source_path") == str(old) for p in receipts):
                stamp = dt.datetime.fromtimestamp(old.stat().st_mtime, dt.timezone.utc).strftime("%Y-%m-%dT%H%M%SZ")
                commission = str(record.get("commission") or old.stem)
                commission = re.sub(r"[^A-Za-z0-9._-]+", "-", commission).strip(".-") or "legacy"
                folder = history / f"{stamp}-{commission}"
                if folder.exists():
                    commission = f"{commission}-{hashlib.sha256(str(old.resolve()).encode()).hexdigest()[:8]}"
                    folder = history / f"{stamp}-{commission}"
                folder.mkdir(parents=True, exist_ok=True)
                migrated = {**record, "legacy_source_path": str(old)}
                (folder / "receipt.json").write_text(json.dumps(migrated, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
                prior_report = Path(record.get("report_data_path", ""))
                if prior_report.is_file():
                    report_data = json.loads(prior_report.read_text(encoding="utf-8"))
                    (folder / "report.json").write_text(json.dumps(report_data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
                else:
                    (folder / "report.json").write_text("{}\n", encoding="utf-8")
                (folder / "answer.json").write_text("{}\n", encoding="utf-8")
                receipts.append(folder / "receipt.json")
        except (OSError, ValueError, KeyError, TypeError, json.JSONDecodeError) as exc:
            errors.append(f"{old}: cannot migrate legacy receipt: {exc}")
    # A prior sync may have migrated only the receipt while its Hub report payload was missing;
    # hydrate an empty placeholder once the preserved report_data_path becomes readable.
    for old in legacy:
        try:
            original = json.loads(old.read_text(encoding="utf-8"))
            payload_path = Path(original.get("report_data_path", ""))
            if not payload_path.is_file():
                continue
            migrated_receipt = next((p for p in receipts if json.loads(p.read_text(encoding="utf-8")).get("legacy_source_path") == str(old)), None)
            if migrated_receipt:
                report_path = migrated_receipt.parent / "report.json"
                if json.loads(report_path.read_text(encoding="utf-8")) == {}:
                    payload = json.loads(payload_path.read_text(encoding="utf-8"))
                    report_path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        except (OSError, ValueError, TypeError, json.JSONDecodeError):
            continue
    receipts = sorted(set(receipts))
    history_dirs = sorted(path for path in history.iterdir() if path.is_dir()) if history.is_dir() else []
    if history_dirs:
        latest_receipt = history_dirs[-1] / "receipt.json"
        latest_data = json.loads(latest_receipt.read_text(encoding="utf-8")) if latest_receipt.is_file() else {}
        latest = {"path": str(history_dirs[-1].relative_to(topic)), "commission": latest_data.get("commission", "legacy"),
                  "created_at": history_dirs[-1].name[:18]}
        (topic / "reporting" / "latest.json").write_text(json.dumps(latest, indent=2) + "\n", encoding="utf-8")
    for receipt in receipts:
        try:
            record = json.loads(receipt.read_text(encoding="utf-8"))
            ask_id = record["ask_id"]
            if not isinstance(ask_id, str) or not ask_id.strip():
                raise ValueError("ask_id must be a nonempty string")
            command = ["notification-hub", "conversations", "get"]
            try:
                result = subprocess.run(command + [ask_id, "--json"], capture_output=True, text=True, timeout=30, check=True)
            except subprocess.CalledProcessError:
                key = record.get("subject_key")
                if not isinstance(key, str) or not key:
                    raise ValueError("ask inaccessible by ID and receipt has no subject_key")
                result = subprocess.run(command + ["--subject-key", key, "--json"], capture_output=True,
                                        text=True, timeout=30, check=True)
            response = json.loads(result.stdout)
            ask = response["ask"]
            if ask.get("id") != ask_id:
                raise ValueError("Hub response does not match the receipt's ask_id")
            if not ask.get("answered_at"):
                continue  # open and marked-read reports are not steering answers
            digest = hashlib.sha256(ask_id.encode()).hexdigest()[:24]
            feedback = topic / "feedback" / f"hub-{digest}.json"
            feedback.parent.mkdir(parents=True, exist_ok=True)
            payload = {"source_receipt": str(receipt), "ask_id": ask_id, "hub_response": response}
            feedback.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
            answer_data = ask.get("answer") or {}
            changes = ask.get("answer_changes") or answer_data.get("changes") or {}
            notes = ask.get("notes") or changes.get("notes") or answer_data.get("notes") or {}
            if isinstance(notes, dict):
                items = []
                for key, value in notes.items():
                    note = value if isinstance(value, dict) else {"text": str(value)}
                    items.append({"id": f"{digest}-{hashlib.sha256(str(key).encode()).hexdigest()[:8]}",
                                  "source_ask_id": ask_id, "key": str(key), "text": str(note.get("text", "")),
                                  "attachments": note.get("attachments", []), "status": "planned",
                                  "reason": "Review and route this item in the next commission."})
                if items:
                    (topic / "feedback" / f"hub-{digest}-items.json").write_text(
                        json.dumps({"items": items}, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
            plan = topic / "work" / "README.md"
            if not plan.is_file():
                raise ValueError(f"missing work plan {plan}; answer preserved at {feedback}")
            marker = f"<!-- research-report-feedback:{digest} -->"
            text = plan.read_text(encoding="utf-8")
            if marker not in text:
                with plan.open("a", encoding="utf-8") as out:
                    out.write(f"\n{marker}\n### Operator report feedback · {ask_id}\n\n"
                              f"Source: `../feedback/{feedback.name}`. Application: **pending supervisor review**. "
                              "Interpret every change, note and image; record the specific action and carry it "
                              "into the affected work briefs before staffing.\n")
        except (OSError, ValueError, KeyError, TypeError, subprocess.SubprocessError) as exc:
            errors.append(f"{receipt}: {exc}")
    return errors


def reconcile(topic: Path, item_id: str, status: str, reason: str) -> None:
    if status not in ("applied", "planned", "declined"):
        raise ValueError("status must be applied, planned, or declined")
    if not reason.strip():
        raise ValueError("a reason is required")
    found = False
    for path in sorted((topic / "feedback").glob("hub-*-items.json")):
        data = json.loads(path.read_text(encoding="utf-8"))
        for item in data.get("items", []):
            if item.get("id") == item_id:
                item.update(status=status, reason=reason.strip())
                path.write_text(json.dumps(data, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
                found = True
    if not found:
        raise ValueError(f"feedback item not found: {item_id}")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("topic", type=Path)
    parser.add_argument("--reconcile", nargs=3, metavar=("ITEM_ID", "STATUS", "REASON"))
    args = parser.parse_args()
    if not (args.topic / "MISSION.md").is_file():
        parser.error("topic must contain MISSION.md")
    if args.reconcile:
        try:
            reconcile(args.topic, *args.reconcile)
            print(json.dumps({"ok": True, "reconciled": args.reconcile[0]}))
            return 0
        except (ValueError, OSError, json.JSONDecodeError) as exc:
            print(json.dumps({"ok": False, "errors": [str(exc)]}))
            return 1
    errors = sync(args.topic)
    print(json.dumps({"ok": not errors, "errors": errors}, ensure_ascii=False))
    return 1 if errors else 0


if __name__ == "__main__":
    sys.exit(main())
