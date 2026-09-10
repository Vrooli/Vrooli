#!/usr/bin/env python3
"""Rewrite the generated Today column in program-runtime-improve §2.

The input is the JSON envelope printed by ``program-runtime library run``.
The same envelope can be applied repeatedly, so regeneration is byte-stable.
"""

import argparse
import datetime as dt
import json
import re
from pathlib import Path


def read_envelope(path: Path) -> dict:
    value = json.loads(path.read_text())
    if isinstance(value, dict) and isinstance(value.get("program"), dict):
        value = value["program"]
    if isinstance(value, dict) and isinstance(value.get("stdout"), str):
        value = json.loads(value["stdout"])
    if not isinstance(value, dict) or not isinstance(value.get("signals"), dict):
        raise ValueError("input does not contain a setpoint envelope")
    rows = value["signals"].get("rows")
    if not isinstance(rows, list):
        raise ValueError("setpoint envelope has no signals.rows list")
    return value


def generated_value(row: dict, date: str) -> str:
    if row.get("unavailable"):
        return f"{date}: unavailable ({row.get('reason') or 'unknown'})"
    reading = row.get("reading")
    if isinstance(reading, (dict, list)):
        rendered = json.dumps(reading, sort_keys=True, separators=(",", ":"))
    else:
        rendered = str(reading)
    band = "in band" if row.get("in_band") else "out of band"
    return f"{date}: {rendered} ({band})"


def rewrite(skill: Path, envelope: dict, date: str) -> None:
    text = skill.read_text()
    start_marker = "## 2. Setpoint\n"
    end_marker = "\n### 3. Sensors"
    start = text.index(start_marker) + len(start_marker)
    end = text.index(end_marker, start)
    section = text[start:end]
    values = {str(row.get("row")): generated_value(row, date) for row in envelope["signals"]["rows"]}
    # Agreement is checked by row name in both directions below. A hardcoded row count used to sit
    # here; it had to be bumped by hand whenever setpoint-read gained a row, and when it was not,
    # regeneration refused to run at all while the table and the envelope agreed row for row.

    lines = section.splitlines()
    seen = set()
    listed = set()
    rewritten = []
    for line in lines:
        if line.startswith("|") and line.count("|") >= 5:
            cells = [cell.strip() for cell in line.strip().strip("|").split("|")]
            name = cells[0] if cells else ""
            if re.fullmatch(r"[a-z0-9][a-z0-9-]*", name):
                listed.add(name)
            if name in values and len(cells) >= 4:
                cells[3] = values[name]
                line = "| " + " | ".join(cells) + " |"
                seen.add(name)
        rewritten.append(line)
    missing = sorted(set(values) - seen)
    if missing:
        raise ValueError("skill is missing setpoint rows: " + ", ".join(missing))
    unreported = sorted(listed - set(values))
    if unreported:
        raise ValueError("envelope does not report skill setpoint rows: " + ", ".join(unreported))
    replacement = "\n".join(rewritten)
    skill.write_text(text[:start] + replacement + "\n" + text[end:])


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--envelope", type=Path, required=True, help="setpoint-read JSON output")
    parser.add_argument("--skill", type=Path, required=True, help="canonical improve skill")
    parser.add_argument("--date", default=dt.date.today().isoformat(), help="observation date, default today")
    args = parser.parse_args()
    if not re.fullmatch(r"\d{4}-\d{2}-\d{2}", args.date):
        raise SystemExit("--date must be YYYY-MM-DD")
    rewrite(args.skill, read_envelope(args.envelope), args.date)


if __name__ == "__main__":
    main()
