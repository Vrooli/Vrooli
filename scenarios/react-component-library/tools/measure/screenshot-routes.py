#!/usr/bin/env python3
"""Capture RCL routes through GCT's redacted, versioned EX9 bundle."""

import argparse
import json
import os
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "tools"))
from ux_capture_bundle import copy_response


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("output", nargs="?", default="/tmp/rclshots")
    output = Path(parser.parse_args().output)
    output.mkdir(parents=True, exist_ok=True)
    routes = json.loads(os.environ.get("ROUTES", '["/"]'))
    width = int(os.environ.get("W", "1440"))
    height = int(os.environ.get("H", "900"))
    ready = os.environ.get(
        "RCL_READY_SELECTOR", "#root[data-experience-state=ready][data-rcl-theme]"
    )
    results = []
    for route in routes:
        name = route.strip("/").replace("/", "_") or "root"
        response_path = output / f"{name}.json"
        screenshot_path = output / f"{name}.png"
        args = [
            "git-control-tower", "ux-capture", "--scenario", "react-component-library",
            "--path", route, "--width", str(width), "--height", str(height),
            "--evidence-profile", "componenttests",
        ]
        if ready:
            args.extend(["--wait-selector", ready])
        try:
            completed = subprocess.run(args, check=True, capture_output=True, text=True, timeout=120)
            response = json.loads(completed.stdout)
            response_path.write_text(json.dumps(response, indent=2))
            copy_response(response, "pixels", screenshot_path)
            results.append({"route": route, "status": "captured", "path": str(screenshot_path), "bundle_id": response["manifest"]["id"]})
        except Exception as error:
            results.append({"route": route, "status": "failed", "reason": str(error)[:500]})
    print(json.dumps({"captured": sum(row["status"] == "captured" for row in results), "failed": sum(row["status"] == "failed" for row in results), "results": results}, indent=2))
    return 1 if any(row["status"] == "failed" for row in results) else 0


if __name__ == "__main__":
    raise SystemExit(main())
