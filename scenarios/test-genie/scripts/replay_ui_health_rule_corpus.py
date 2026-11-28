#!/usr/bin/env python3
"""Build and replay the real ui-health geometry evaluator over frozen JSONL.

This command does not start a scenario or browser service. Its output measures
the `ui-health` BrowserObservation rule evaluator only; it is not a full
Test Genie `experience` phase result.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import subprocess
import tempfile
from datetime import datetime, timezone
from pathlib import Path
from typing import Sequence


def digest_files(paths: Sequence[Path]) -> str:
    digest = hashlib.sha256()
    for path in sorted(paths):
        digest.update(str(path).encode())
        digest.update(b"\0")
        digest.update(path.read_bytes())
        digest.update(b"\0")
    return digest.hexdigest()


def source_files(api_dir: Path) -> list[Path]:
    root = api_dir.parents[2]
    proto_dir = root / "packages/proto/gen/go/ui-health/v1/visualhealth"
    paths = [api_dir / "go.mod", api_dir / "go.sum", api_dir / "cmd/ui-health-trust-replay/main.go"]
    paths.extend(sorted((api_dir / "internal/visualhealth").glob("*.go")))
    paths.extend(sorted(proto_dir.glob("*.go")))
    missing = [path for path in paths if not path.is_file()]
    if missing:
        raise ValueError("evaluator source set is incomplete: " + ", ".join(str(path) for path in missing))
    return paths


def run(argv: list[str], cwd: Path) -> None:
    subprocess.run(argv, cwd=cwd, check=True)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--input", type=Path, required=True, help="fixed evaluator-input JSONL")
    parser.add_argument("--output", type=Path, required=True, help="prediction JSONL output")
    parser.add_argument("--manifest", type=Path, required=True, help="exact evaluator/build/command manifest")
    parser.add_argument("--api-dir", type=Path, help="ui-health/api source tree; defaults to this checkout")
    args = parser.parse_args()
    script = Path(__file__).resolve()
    repo = script.parents[3]
    api_dir = (args.api_dir or (repo / "scenarios/ui-health/api")).resolve()
    input_path = args.input.resolve()
    output_path = args.output.resolve()
    manifest_path = args.manifest.resolve()
    if not input_path.is_file():
        raise ValueError(f"input corpus not found: {input_path}")
    files = source_files(api_dir)
    source_sha = digest_files(files)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.manifest.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="test-genie-ui-replay-") as temporary:
        binary = Path(temporary) / "ui-health-trust-replay"
        # This runner must work on immutable baseline trees as well as dirty
        # source trees. Go's VCS stamping otherwise requires a writable/valid
        # .git checkout at the invocation directory.
        build_argv = ["go", "build", "-buildvcs=false", "-o", str(binary), "./cmd/ui-health-trust-replay"]
        run(build_argv, api_dir)
        binary_sha = hashlib.sha256(binary.read_bytes()).hexdigest()
        execute_argv = [str(binary), "--input", str(input_path), "--output", str(output_path)]
        run(execute_argv, api_dir)
        if digest_files(files) != source_sha:
            output_path.unlink(missing_ok=True)
            raise RuntimeError("evaluator source changed during replay; output was discarded")
    manifest = {
        "schema": "ui-health-rule-replay-evaluator/v1",
        "evaluator": "ui-health.browserObservationFindings/current",
        "build_identity": f"ui-health-trust-replay:sha256:{binary_sha}",
        "command": json.dumps({"build": build_argv, "execute": execute_argv}, ensure_ascii=False),
        "execution_mode": "replay",
        "source_sha256": source_sha,
        "input_sha256": hashlib.sha256(input_path.read_bytes()).hexdigest(),
        "output_sha256": hashlib.sha256(output_path.read_bytes()).hexdigest(),
        "created_at": datetime.now(timezone.utc).isoformat(),
        "source_tree": str(api_dir),
        "input": str(input_path),
        "output": str(output_path),
        "scope": "actual ui-health BrowserObservation rule evaluator replay; not the aggregate Test Genie experience phase",
        "live_services_started": False,
        "bas_capture_used": False,
    }
    manifest_path.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"predictions": str(output_path), "manifest": str(manifest_path), "evaluator": manifest["evaluator"], "build_identity": manifest["build_identity"], "source_sha256": source_sha, "output_sha256": manifest["output_sha256"]}, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
