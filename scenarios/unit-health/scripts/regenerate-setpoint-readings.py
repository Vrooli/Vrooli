#!/usr/bin/env python3
"""Rewrite the generated Today column in unit-health-improve §2 from a setpoint-read envelope.

Usage:
    program-runtime library run unit-health.setpoint-read --provenance operator --json > /tmp/board.json
    python3 scenarios/unit-health/scripts/regenerate-setpoint-readings.py /tmp/board.json

The input is the JSON printed by ``program-runtime library run`` (or ``programs wait``), or the
bare envelope. The same envelope applied twice is byte-stable. The script copies ``reading`` and
``reason`` verbatim; it never computes a reading of its own.
"""

import argparse
import datetime as dt
import json
import re
from pathlib import Path

SKILL = Path(__file__).resolve().parents[1] / "skills" / "unit-health-improve" / "SKILL.md"
START = "### 2. Setpoint\n"
END = "\n### 3. Sensors"
EXPECTED_ROWS = 13


def read_envelope(path: Path) -> dict:
    text = path.read_text()
    value = json.loads(text[text.index("{"):])
    if isinstance(value, dict) and isinstance(value.get("program"), dict):
        value = value["program"]
    if isinstance(value, dict) and isinstance(value.get("stdout"), str):
        stdout = value["stdout"]
        start = stdout.find('{"program": "unit-health.setpoint-read"')
        if start < 0:
            raise ValueError("stdout does not contain a unit-health.setpoint-read envelope")
        value = json.loads(stdout[start:].strip().splitlines()[0])
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
    rendered = json.dumps(reading, sort_keys=True, separators=(",", ":")) if isinstance(reading, (dict, list)) else str(reading)
    if row.get("in_band") is None:
        band = "no band; target undecided"
    else:
        band = "in band" if row.get("in_band") else "out of band"
    return f"{date}: {rendered} ({band})"


def rewrite(skill: Path, envelope: dict, date: str) -> int:
    text = skill.read_text()
    start = text.index(START) + len(START)
    end = text.index(END, start)
    section = text[start:end]
    values = {str(row.get("row")): generated_value(row, date) for row in envelope["signals"]["rows"]}
    if len(values) != EXPECTED_ROWS:
        raise ValueError(f"expected {EXPECTED_ROWS} setpoint rows, got {len(values)}")
    rewritten, seen = [], set()
    for line in section.splitlines():
        if line.startswith("|") and line.count("|") >= 5:
            cells = [cell.strip() for cell in line.strip().strip("|").split("|")]
            name = cells[0]
            if name in values and len(cells) >= 4:
                cells[3] = values[name].replace("|", "\\|")
                seen.add(name)
                line = "| " + " | ".join(cells) + " |"
        rewritten.append(line)
    missing = sorted(set(values) - seen)
    if missing:
        raise ValueError(f"setpoint table has no row for: {', '.join(missing)}")
    new_section = "\n".join(rewritten)
    if section.endswith("\n") and not new_section.endswith("\n"):
        new_section += "\n"
    updated = text[:start] + new_section + text[end:]
    updated = re.sub(r"(updatedAt: \")[^\"]+(\")", rf"\g<1>{date}T00:00:00Z\g<2>", updated, count=1)
    skill.write_text(updated)
    return len(seen)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("envelope", type=Path, help="library run / programs wait JSON, or the bare envelope")
    parser.add_argument("--skill", type=Path, default=SKILL)
    parser.add_argument("--date", default=dt.date.today().isoformat())
    args = parser.parse_args()
    count = rewrite(args.skill, read_envelope(args.envelope), args.date)
    print(f"rewrote {count} Today cells in {args.skill}")


if __name__ == "__main__":
    main()
