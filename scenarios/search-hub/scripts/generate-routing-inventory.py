#!/usr/bin/env python3
"""Generate the provider inventory embedded in the project routing guide."""

import argparse
import json
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
DOC = ROOT / "docs/reference/ai-search-routing.md"
START = "<!-- search-hub-provider-inventory:start -->"
END = "<!-- search-hub-provider-inventory:end -->"


def cell(value):
    return " ".join(str(value or "").split()).replace("|", "\\|")


def table(payload):
    rows = []
    for provider in payload["providers"]:
        lifecycle = provider.get("lifecycle", "").removeprefix("LIFECYCLE_")
        if lifecycle == "FIXTURE":
            continue
        profile = provider.get("routing_profile") or {}
        answers = ", ".join(profile.get("answer_spaces", [])) or provider.get("description", "")
        state = provider.get("state", "").removeprefix("PROVIDER_STATE_")
        bucket = provider.get("bucket", "").removeprefix("BUCKET_")
        kind = provider.get("type", "")
        query = "—" if state == "CAPABILITY_GAP" else f'`search-hub query "<intent>" --type {kind}`'
        rows.append((bucket, kind, provider["provider_id"], state, lifecycle, answers, query))
    rows.sort(key=lambda row: (row[0], row[1], row[2]))
    lines = [
        "| Bucket | Type | Provider | State | Lifecycle | Answers | Query |",
        "|---|---|---|---|---|---|---|",
    ]
    lines.extend("| " + " | ".join(cell(value) for value in row) + " |" for row in rows)
    return "\n".join(lines), len(rows)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    choice = parser.add_mutually_exclusive_group(required=True)
    choice.add_argument("--write", action="store_true", help="refresh the generated table")
    choice.add_argument("--check", action="store_true", help="fail if the table is stale")
    args = parser.parse_args()

    result = subprocess.run(
        ["search-hub", "providers", "list", "--json"],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    )
    generated, count = table(json.loads(result.stdout))
    original = DOC.read_text()
    if original.count(START) != 1 or original.count(END) != 1:
        raise SystemExit(f"{DOC.relative_to(ROOT)} must contain one pair of inventory markers")
    before, rest = original.split(START, 1)
    _, after = rest.split(END, 1)
    updated = before + START + "\n" + generated + "\n" + END + after

    if args.check:
        if original != updated:
            raise SystemExit(f"{DOC.relative_to(ROOT)} has a stale provider inventory")
        print(f"{count} non-fixture providers match the live registry")
    else:
        DOC.write_text(updated)
        print(f"updated {DOC.relative_to(ROOT)} ({count} non-fixture providers)")


if __name__ == "__main__":
    main()
