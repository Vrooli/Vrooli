#!/usr/bin/env python3
"""Read-only, stdlib-only source inventory for the BAS refactor assessment.

Run from any directory. Prints JSON; never starts services or writes source.
Physical line counts are size indicators, not cyclomatic-complexity scores.
"""

import collections
import argparse
import datetime
import hashlib
import json
from pathlib import Path
import re
import subprocess


ROOT = Path(__file__).resolve().parents[4]
SCENARIO = ROOT / "scenarios/browser-automation-studio"
SUFFIXES = {".go", ".ts", ".tsx", ".js", ".mjs", ".py"}
EXCLUDED = {"node_modules", "dist", "coverage", "vendor", "gen", "generated"}


def git(*args):
    return subprocess.check_output(["git", "-C", str(ROOT), *args]).decode()


def inventory(include_untracked=False):
    files = []
    digest = hashlib.sha256()
    selection = ["--cached", "--others", "--exclude-standard"] if include_untracked else []
    candidates = sorted(set(filter(None, git("ls-files", "-z", *selection, "--", str(SCENARIO)).split("\0"))))
    for name in candidates:
        path = ROOT / name
        if not path.is_file() or path.suffix not in SUFFIXES:
            continue
        relative = path.relative_to(SCENARIO)
        parts = relative.parts
        if EXCLUDED.intersection(parts) or ".generated." in path.name:
            continue
        data = path.read_bytes()
        text = data.decode(errors="replace")
        test = (
            path.name.endswith("_test.go")
            or bool(re.search(r"\.(test|spec)\.", path.name))
            or bool({"tests", "__tests__", "test-utils"}.intersection(parts))
        )
        runtime = (
            (parts[0] in {"api", "cli"} and path.suffix == ".go")
            or (parts[0] in {"ui", "playwright-driver"} and len(parts) > 1 and parts[1] == "src")
        ) and not test and not {"testutil", "testing"}.intersection(parts) and not path.name.startswith(("testutil_", "test-setup"))
        content_hash = hashlib.sha256(data).hexdigest()
        digest.update(f"{relative.as_posix()}\0{content_hash}\n".encode())
        files.append({
            "path": relative.as_posix(), "lines": len(text.splitlines()),
            "bytes": len(data), "test": test, "runtime_source": bool(runtime),
            "sha256": content_hash,
        })
    surfaces = {}
    for surface in sorted({f["path"].split("/")[0] for f in files}):
        rows = [f for f in files if f["path"].split("/")[0] == surface]
        runtime = [f for f in rows if f["runtime_source"]]
        surfaces[surface] = {
            "tracked_source_files": len(rows),
            "tracked_source_lines": sum(f["lines"] for f in rows),
            "runtime_files": len(runtime),
            "runtime_lines": sum(f["lines"] for f in runtime),
            "runtime_over_500_lines": sum(f["lines"] > 500 for f in runtime),
            "runtime_over_1000_lines": sum(f["lines"] > 1000 for f in runtime),
        }
    return {
        "schema_version": 1,
        "observed_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
        "head": git("rev-parse", "HEAD").strip(),
        "scenario_worktree_status": git("status", "--short", "--", str(SCENARIO)).splitlines(),
        "source_digest_sha256": digest.hexdigest(),
        "method": {
            "population": ("Git-tracked plus nonignored untracked" if include_untracked else "Git-tracked") + " Go/TS/TSX/JS/MJS/Python files; physical lines include blanks and comments. tracked_source_* keys describe this selected population.",
            "include_untracked": include_untracked,
            "excluded": sorted(EXCLUDED) + ["filenames containing .generated."],
            "runtime": "api/cli Go and ui/driver src, excluding named tests, testutil/testing directories, testutil_* and test-setup* helpers.",
            "limits": "Not AST complexity, coverage or code-quality score. Runtime recording/testing diagnostics are excluded from main-path count. Ignored generated desktop trees are excluded; untracked source follows include_untracked. Shared worktree can change during scan.",
        },
        "surfaces": surfaces,
        "largest_runtime_files": sorted((f for f in files if f["runtime_source"]), key=lambda f: (-f["lines"], f["path"]))[:30],
        "source_extensions": dict(collections.Counter(Path(f["path"]).suffix for f in files)),
    }


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--include-untracked", action="store_true", help="Include new source so untracked additions cannot hide growth")
    args = parser.parse_args()
    print(json.dumps(inventory(args.include_untracked), indent=2))
