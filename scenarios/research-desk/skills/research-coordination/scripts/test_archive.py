#!/usr/bin/env python3
"""Tests for archive.py. Run: python3 -I test_archive.py (from this directory)."""

from __future__ import annotations

import contextlib
import io
import json
import os
import sys
import tempfile
import time
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import archive  # noqa: E402

OLD = time.time() - 3 * 86400


def write(root: Path, rel: str, body: str = "x") -> Path:
    path = root / rel
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(body)
    return path


def age(root: Path) -> None:
    """Make every file and folder under root three days old."""
    for d, dirs, files in os.walk(root):
        for name in dirs + files:
            os.utime(Path(d) / name, (OLD, OLD))
    os.utime(root, (OLD, OLD))


class ArchiveTest(unittest.TestCase):
    def setUp(self) -> None:
        tmp = Path(tempfile.mkdtemp())
        self.home, self.archive = tmp / "research", tmp / "archive"
        run = "topics/t/runs/r1"
        write(self.home, run + "/README.md", "# r1\n\nResults in [proof](cf2-a/result.json).\n")
        write(self.home, run + "/cf2-a/result.json", "{}")
        write(self.home, run + "/cf2-a/lpbs-workspace/VISION.md")
        write(self.home, run + "/cf2-a/lpbs-workspace/scenarios/x/api/main.go")
        write(self.home, run + "/app/node_modules/pkg/index.js")
        write(self.home, run + "/cf2-b/workspace/AGENTS.md")
        write(self.home, run + "/cf2-b/workspace/scenarios/y/README.md")
        write(self.home, run + "/cf2-b/NOTE.md", "Replay from workspace/scenarios/y.\n")
        age(self.home)

    def run_cli(self, *args: str) -> tuple[int, dict]:
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            code = archive.main([*args, "--home", str(self.home), "--archive", str(self.archive), "--no-desk", "--json"])
        return code, json.loads(out.getvalue())

    def test_plan_finds_rehearsal_copies_and_rebuildable_trees_and_keeps_linked_ones(self) -> None:
        code, res = self.run_cli("plan", "--topic", "t")
        self.assertEqual(code, 0)
        planned = {(i["kind"], i["path"].split("/r1/")[1]) for i in res["items"]}
        self.assertEqual(planned, {("rehearsal", "cf2-a/lpbs-workspace"), ("rebuildable", "app/node_modules")})
        kept = {i["path"].split("/r1/")[1]: i["why"] for i in res["refused"]}
        self.assertIn("cf2-b/workspace", kept, "a note beside the copy links into it, so it stays")
        self.assertIn("linked from", kept["cf2-b/workspace"])
        self.assertTrue((self.home / "topics/t/runs/r1/cf2-a/lpbs-workspace").is_dir(), "plan moves nothing")

    def test_apply_moves_by_rename_and_leaves_a_pointer(self) -> None:
        src = self.home / "topics/t/runs/r1/cf2-a/lpbs-workspace"
        inode = os.stat(src / "VISION.md").st_ino
        code, res = self.run_cli("apply", "--topic", "t")
        self.assertEqual(code, 0)
        dest = self.archive / "rehearsal/t/runs/r1/cf2-a/lpbs-workspace"
        self.assertFalse(src.exists())
        self.assertEqual(os.stat(dest / "VISION.md").st_ino, inode, "moved, not copied")
        pointer = (src.parent / "lpbs-workspace.ARCHIVED.md").read_text()
        self.assertIn(str(dest), pointer)
        self.assertIn(f"mv '{dest}' '{src}'", pointer)
        self.assertTrue((self.archive / "rebuildable/t/runs/r1/app/node_modules/pkg/index.js").exists())
        for kind in ("rehearsal", "rebuildable"):
            index = [json.loads(line) for line in (self.archive / kind / "index.jsonl").read_text().splitlines()]
            self.assertEqual([i["kind"] for i in index], [kind])
        self.assertTrue((self.home / "topics/t/runs/r1/cf2-a/result.json").exists(), "evidence beside the copy stays")
        code, res = self.run_cli("plan", "--topic", "t")
        self.assertEqual([i["path"] for i in res["items"]], [], "nothing left to move")

    def test_a_recently_written_run_is_not_closed(self) -> None:
        write(self.home, "topics/t/runs/r1/CHECKPOINT.md", "still going")
        _, res = self.run_cli("plan", "--topic", "t")
        self.assertEqual(res["items"], [])
        self.assertTrue(all("ago" in i["why"] for i in res["refused"]))

    def test_a_live_topic_is_never_touched(self) -> None:
        res = archive.plan(self.home, "t", None, 24, {"t"})
        self.assertEqual(res["items"], [])
        self.assertTrue(all("live" in i["why"] for i in res["refused"]))

    def test_live_topics_reads_the_desk_commission_shape(self) -> None:
        shape = {"commissions": [{"id": "c1", "topics": [{"topic": "t", "basis": "packet"}]}]}
        real = archive.subprocess.run
        archive.subprocess.run = lambda *a, **k: type("R", (), {"stdout": json.dumps(shape)})()
        try:
            live, note = archive.live_topics()
        finally:
            archive.subprocess.run = real
        self.assertEqual((live, note), ({"t"}, ""))

    def test_apply_refuses_when_the_desk_cannot_say_what_is_live(self) -> None:
        real = archive.live_topics
        archive.live_topics = lambda: (set(), "Research Desk unavailable (OSError)")
        try:
            err = io.StringIO()
            with contextlib.redirect_stderr(err):
                code = archive.main(["apply", "--topic", "t", "--home", str(self.home), "--archive", str(self.archive)])
        finally:
            archive.live_topics = real
        self.assertEqual(code, 1)
        self.assertIn("refused", err.getvalue())
        self.assertTrue((self.home / "topics/t/runs/r1/cf2-a/lpbs-workspace").is_dir())

    def test_an_existing_archive_entry_is_never_overwritten(self) -> None:
        write(self.archive, "rehearsal/t/runs/r1/cf2-a/lpbs-workspace/OLD.md")
        code, res = self.run_cli("apply", "--topic", "t")
        self.assertEqual(code, 1)
        self.assertTrue((self.home / "topics/t/runs/r1/cf2-a/lpbs-workspace").is_dir())
        self.assertIn("already exists", " ".join(i["why"] for i in res["refused"]))


if __name__ == "__main__":
    unittest.main()
