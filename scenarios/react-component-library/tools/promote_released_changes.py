#!/usr/bin/env python3
"""Promote source edits without mutating an immutable released version.

The catalog index retains the released source body. For each selected changed
entry this tool restores that body, starts a patch draft, writes the edited
body into the draft, and publishes it through the scenario CLI. The old path
therefore remains byte-for-byte reproducible while the new version carries the
restyle/contract change.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import sqlite3
import subprocess
from pathlib import Path


def bump_patch(version: str) -> str:
    parts = version.split(".")
    if len(parts) != 3 or any(not part.isdigit() for part in parts):
        raise ValueError(f"cannot patch non-semver released version {version!r}")
    return f"{parts[0]}.{parts[1]}.{int(parts[2]) + 1}"


def semver_key(version: str) -> tuple[int, int, int]:
    parts = version.split(".")
    return tuple(int(part) for part in parts[:3])


def run_cli(repo: Path, args: list[str], body: bytes | None = None) -> dict:
    command = ["react-component-library", *args, "--json"]
    result = subprocess.run(command, cwd=repo, input=body, capture_output=True, check=False)
    if result.returncode != 0:
        raise RuntimeError(result.stderr.decode().strip() or result.stdout.decode().strip())
    return json.loads(result.stdout)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--library-id", action="append", help="limit to one or more library ids")
    parser.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()

    repo = Path(__file__).resolve().parents[3]
    db_path = Path(os.environ.get("VROOLI_RCL_DB", Path.home() / ".vrooli/data/vrooli/react-component-library/react-component-library.db"))
    db = sqlite3.connect(f"file:{db_path}?mode=ro", uri=True)
    rows = db.execute(
        "SELECT library_id, version, source_path, content, content_sha256 FROM component_versions WHERE status = 'released' ORDER BY library_id, version"
    ).fetchall()
    latest_rows = {
        library_id: (latest_version, draft_version)
        for library_id, latest_version, draft_version in db.execute(
            "SELECT library_id, latest_version, draft_version FROM components"
        )
    }
    released_versions = {
        library_id: {
            version
            for version, in db.execute(
                "SELECT version FROM component_versions WHERE library_id = ? AND status = 'released'",
                (library_id,),
            )
        }
        for library_id in latest_rows
    }
    previous_versions = {
        library_id: sorted(
            (version for version in versions if "-" not in version),
            key=semver_key,
        )
        for library_id, versions in released_versions.items()
    }
    selected = set(args.library_id or [])
    mismatches: list[tuple[str, str, str, bytes, bytes]] = []
    for library_id, version, source_path, old_content, recorded_hash in rows:
        if selected and library_id not in selected:
            continue
        path = repo / "scenarios/react-component-library/library" / source_path
        if not path.exists():
            continue
        edited = path.read_bytes()
        if hashlib.sha256(edited).hexdigest() == recorded_hash:
            continue
        if isinstance(old_content, str):
            old_content = old_content.encode()
        mismatches.append((library_id, version, source_path, edited, old_content))

    # Restore every released path before creating a new release. This matters
    # when a component has more than one released version: the old versions
    # must all become reproducible, while the newest edited body is promoted
    # once as the next patch release.
    for library_id, version, source_path, edited, old_content in mismatches:
        path = repo / "scenarios/react-component-library/library" / source_path
        print(f"restore {library_id}@{version}")
        if not args.dry_run:
            path.write_bytes(old_content)

    promotions: list[tuple[str, str, str, bytes]] = []
    for library_id, version, source_path, edited, _ in mismatches:
        latest_version, draft_version = latest_rows.get(library_id, ("", ""))
        if version != latest_version or any(item[0] == library_id for item in promotions):
            continue
        if draft_version:
            raise RuntimeError(f"{library_id} has an active draft {draft_version}; publish or discard it first")
        target = bump_patch(latest_version)
        while target in released_versions.get(library_id, set()):
            target = bump_patch(target)
        promotions.append((library_id, target, source_path, edited))

    for library_id, target, source_path, edited in promotions:
        print(f"promote {library_id}@{target}")
        if args.dry_run:
            continue
        # CreateComponentVersion accepts the edited entry through stdin and
        # copies all companion artifacts from the latest release. It performs
        # the release story-coverage checks before making the new version
        # visible, so no mutable draft needs to be edited through the moving
        # component manifest source path.
        released = previous_versions.get(library_id, [])
        latest_version = latest_rows[library_id][0]
        copy_from = latest_version
        older = [version for version in released if semver_key(version) < semver_key(latest_version)]
        if older:
            # A prior failed promotion may have produced a release containing
            # only the entry body. Prefer the preceding release so companion
            # modules are carried into the repaired patch version.
            copy_from = older[-1]
        run_cli(
            repo,
            [
                "components",
                "version-create",
                library_id,
                target,
                "--from-version",
                copy_from,
                "--source-file",
                "-",
                "--release",
                "true",
            ],
            edited,
        )

    print(json.dumps({"restored": len(mismatches), "promoted": len(promotions), "unchanged": len(rows) - len(mismatches)}))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
