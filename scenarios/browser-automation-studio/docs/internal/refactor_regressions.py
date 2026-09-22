#!/usr/bin/env python3
"""Run retained isolated behavioral probes as a fail-closed regression pack.

This tests actual modules with synthetic I/O, not browser/platform qualification.
Exit 0 requires every observed expected behavior; exit 1 is a behavioral failure;
exit 2 means a producer failed or returned unusable evidence.
"""

import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile

HERE = Path(__file__).resolve().parent
SCENARIO = HERE.parents[1]
CASES = {
    "recording-driver": "refactor_probes.cjs",
    "recording-api": "refactor_probes.go",
    "profile": "refactor_profile_probes.go",
    "input": "refactor_input_probes.cjs",
    "session-frame": "refactor_session_frame_probes.cjs",
    "retention": "refactor_retention_probes.go",
    "execution-api": "refactor_execution_probes.go",
    "execution-driver": "refactor_execution_probes.cjs",
    "reuse": "refactor_reuse_probes.cjs",
    "stream": "refactor_stream_probes.cjs",
}


def classify(payload):
    rows = payload.get("results")
    if not isinstance(rows, list) or not rows:
        return "unavailable"
    ids = [row.get("id") for row in rows]
    if len(set(ids)) != len(ids) or not all(ids):
        return "unavailable"
    if any(type(row.get("expected_behavior_met")) is not bool or row.get("probe_error") for row in rows):
        return "unavailable"
    return "passed" if all(row["expected_behavior_met"] for row in rows) else "failed"


def run_case(name, directory, cache):
    if name in cache:
        return cache[name]
    path = HERE / CASES[name]
    argv = ["go", "run", str(path)] if path.suffix == ".go" else ["node", str(path)]
    if name == "execution-driver":
        prerequisite = run_case("execution-api", directory, cache)
        if prerequisite["status"] == "unavailable":
            return {"case": name, "status": "unavailable", "reason": "execution-api producer unavailable"}
        argv.append(str(directory / "execution-api.json"))
    try:
        completed = subprocess.run(argv, cwd=SCENARIO / "api", capture_output=True,
                                   text=True, timeout=120,
                                   env={**os.environ, "GOPROXY": "off", "GOTOOLCHAIN": "local", "LOG_LEVEL": "error"})
        payload = json.loads(completed.stdout)
        status = classify(payload) if completed.returncode == 0 else "unavailable"
        (directory / f"{name}.json").write_text(completed.stdout)
        result = {"case": name, "status": status, "producer_sha256": hashlib.sha256(path.read_bytes()).hexdigest(),
                  "observations": payload.get("results", []), "scope": payload.get("scope"),
                  "producer_exit": completed.returncode}
        if completed.returncode:
            result["error"] = completed.stderr[-2000:]
    except (subprocess.SubprocessError, OSError, ValueError) as exc:
        result = {"case": name, "status": "unavailable", "reason": str(exc)[:2000]}
    cache[name] = result
    return result


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--case", action="append", choices=CASES, help="Run only the affected producer; default all")
    args = parser.parse_args()
    with tempfile.TemporaryDirectory(prefix="bas-refactor-probes-") as temporary:
        cache = {}
        results = [run_case(name, Path(temporary), cache) for name in (args.case or CASES)]
    states = [r["status"] for r in results]
    print(json.dumps({"observed_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
                      "scope": "isolated actual-module regressions; not product or native-platform certification",
                      "cases": results, "summary": {state: states.count(state) for state in ("passed", "failed", "unavailable")}}, indent=2))
    raise SystemExit(2 if "unavailable" in states else 1 if "failed" in states else 0)
