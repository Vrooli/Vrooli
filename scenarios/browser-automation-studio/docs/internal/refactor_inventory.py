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
import os
import re
import subprocess


ROOT = Path(__file__).resolve().parents[4]
SCENARIO = ROOT / "scenarios/browser-automation-studio"
SUFFIXES = {".go", ".ts", ".tsx", ".js", ".mjs", ".py"}
EXCLUDED = {"node_modules", "dist", "coverage", "vendor", "gen", "generated"}
# Without Git's ignore rules, a filesystem walk also skips runtime output and hidden trees.
WALK_EXCLUDED = EXCLUDED | {"data", "logs", "tmp", "test-results", "playwright-report", "bundle"}


def git(*args):
    return subprocess.check_output(["git", "-C", str(ROOT), *args]).decode()


def walk():
    for directory, subdirectories, names in os.walk(SCENARIO):
        subdirectories[:] = sorted(d for d in subdirectories if d not in WALK_EXCLUDED and not d.startswith("."))
        for name in names:
            yield (Path(directory) / name).relative_to(ROOT).as_posix()


def inventory(include_untracked=False, no_git=False):
    files = []
    digest = hashlib.sha256()
    if no_git:
        tracked, untracked = set(), set()
        candidates = set(walk())
    else:
        tracked = set(filter(None, git("ls-files", "-z", "--cached", "--", str(SCENARIO)).split("\0")))
        untracked = set(filter(None, git("ls-files", "-z", "--others", "--exclude-standard", "--", str(SCENARIO)).split("\0")))
        candidates = tracked | (untracked if include_untracked else set())
    for name in sorted(candidates):
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
        row = {
            "path": relative.as_posix(), "lines": len(text.splitlines()),
            "bytes": len(data), "test": test, "runtime_source": bool(runtime),
            "sha256": content_hash,
        }
        if not no_git:
            row["git_status"] = "tracked" if name in tracked else "untracked"
        files.append(row)
    surfaces = {}
    for surface in sorted({f["path"].split("/")[0] for f in files}):
        rows = [f for f in files if f["path"].split("/")[0] == surface]
        runtime = [f for f in rows if f["runtime_source"]]
        surfaces[surface] = {
            "selected_source_files": len(rows),
            "selected_source_lines": sum(f["lines"] for f in rows),
            "runtime_files": len(runtime),
            "runtime_lines": sum(f["lines"] for f in runtime),
            "runtime_over_500_lines": sum(f["lines"] > 500 for f in runtime),
            "runtime_over_1000_lines": sum(f["lines"] > 1000 for f in runtime),
        }
        if no_git:
            continue
        tracked_rows = [f for f in rows if f["git_status"] == "tracked"]
        untracked_rows = [f for f in rows if f["git_status"] == "untracked"]
        tracked_runtime = [f for f in tracked_rows if f["runtime_source"]]
        untracked_runtime = [f for f in untracked_rows if f["runtime_source"]]
        surfaces[surface].update({
            "tracked_source_files": len(tracked_rows),
            "tracked_source_lines": sum(f["lines"] for f in tracked_rows),
            "untracked_source_files": len(untracked_rows),
            "untracked_source_lines": sum(f["lines"] for f in untracked_rows),
            "tracked_runtime_files": len(tracked_runtime),
            "tracked_runtime_lines": sum(f["lines"] for f in tracked_runtime),
            "untracked_runtime_files": len(untracked_runtime),
            "untracked_runtime_lines": sum(f["lines"] for f in untracked_runtime),
        })
    if no_git:
        population = "Filesystem walk (no Git) of"
        excluded = sorted(WALK_EXCLUDED) + ["hidden directories"]
    else:
        population = "Git-tracked plus nonignored untracked" if include_untracked else "Git-tracked"
        excluded = sorted(EXCLUDED)
    report = {
        "schema_version": 1,
        "observed_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
        "source_digest_sha256": digest.hexdigest(),
        "method": {
            "population": population + " Go/TS/TSX/JS/MJS/Python files; physical lines include blanks and comments. tracked_source_* and untracked_source_* split selected files by Git status (Git mode only); selected_source_* and runtime_* describe the selected population.",
            "include_untracked": include_untracked,
            "no_git": no_git,
            "excluded": excluded + ["filenames containing .generated."],
            "runtime": "api/cli Go and ui/driver src, excluding named tests, testutil/testing directories, testutil_* and test-setup* helpers.",
            "limits": "Not AST complexity, coverage or code-quality score. Runtime recording/testing diagnostics are excluded from main-path count. Ignored generated desktop trees are excluded; untracked source follows include_untracked. Shared worktree can change during scan.",
        },
        "surfaces": surfaces,
        "largest_runtime_files": sorted((f for f in files if f["runtime_source"]), key=lambda f: (-f["lines"], f["path"]))[:30],
        "source_extensions": dict(collections.Counter(Path(f["path"]).suffix for f in files)),
    }
    if not no_git:
        report["head"] = git("rev-parse", "HEAD").strip()
        report["scenario_worktree_status"] = git("status", "--short", "--", str(SCENARIO)).splitlines()
    return report


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--include-untracked", action="store_true", help="Include new source so untracked additions cannot hide growth")
    parser.add_argument("--no-git", action="store_true", help="Walk the filesystem instead of calling Git; omits head, status and the tracked/untracked split")
    args = parser.parse_args()
    print(json.dumps(inventory(args.include_untracked, args.no_git), indent=2))
