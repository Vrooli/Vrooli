#!/usr/bin/env python3
"""Tests for find.py. Run: python3 -I test_find.py (from this directory).

The walk-policy case uses the same fixture and expectations as Source Ledger's
TestWalkPolicySummarisesHeavyFoldersAndHonoursFeedsJSON, so the file fallback
and the service agree on what exists.
"""

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
import find  # noqa: E402


def write(root: Path, rel: str, body: str) -> None:
    path = root / rel
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(body)


def run(argv: list[str], env: dict | None = None) -> tuple[int, dict]:
    out = io.StringIO()
    old = dict(os.environ)
    os.environ.update(env or {})
    try:
        with contextlib.redirect_stdout(out):
            code = find.main(argv + ["--json"])
    finally:
        os.environ.clear()
        os.environ.update(old)
    return code, json.loads(out.getvalue())


class FindTest(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def test_edits_and_deletes_are_visible_with_citations(self) -> None:
        write(self.root, "topics/ux/knowledge/MAP.md", "# Map\n\n## Scenarios\n\nThe planner owns week view.\n")
        code, res = run(["planner week", "--home", str(self.root)])
        self.assertEqual(code, 0)
        self.assertEqual(res["hits"][0]["citation"], "topics/ux/knowledge/MAP.md#L3-L5")
        self.assertEqual(res["hits"][0]["heading"], "Map › Scenarios")
        self.assertEqual((res["hits"][0]["topic"], res["hits"][0]["feed"]), ("ux", "knowledge"))

        time.sleep(0.01)
        write(self.root, "topics/ux/knowledge/MAP.md", "# Map\n\n## Scenarios\n\nThe planner owns week view.\n\n## Capture\n\nZebra capture pipeline replaces screenshots.\n")
        _, res = run(["zebra capture", "--home", str(self.root)])
        self.assertEqual(res["hits"][0]["citation"], "topics/ux/knowledge/MAP.md#L7-L9")

        (self.root / "topics/ux/knowledge/MAP.md").unlink()
        code, res = run(["zebra capture", "--home", str(self.root)])
        self.assertEqual((code, res["hits"]), (1, []))

    def test_walk_policy_matches_source_ledger(self) -> None:
        files = {
            "topics/a/README.md": "# A\n",
            "topics/a/runs/r1/README.md": "# run\n",
            "topics/a/runs/r1/notes/x.md": "kept: depth 1\n",
            "topics/a/runs/r1/notes/deep/README.md": "summary kept\n",
            "topics/a/runs/r1/notes/deep/raw.md": "dropped: depth 2\n",
            "topics/a/runs/r1/proto/package.json": "{}\n",
            "topics/a/runs/r1/proto/README.md": "code tree summary\n",
            "topics/a/runs/r1/proto/notes.md": "dropped: code tree\n",
            "topics/a/runs/r1/persist/db.md": "dropped: pruned dir\n",
            "topics/a/corpus/copy/doc.md": "dropped by topic feeds.json\n",
            "topics/a/feedback/2026-11.md": "operator said\n",
            "topics/a/feeds.json": '{"exclude":["corpus/**"],"include":["runs/*/notes/deep/raw.md"],"feeds":{"operator":["feedback/**"]}}',
            "feeds.json": '{"dataset_files": 4}',
        }
        files.update({f"topics/a/runs/r1/data/case{c}.md": "generated\n" for c in "abcde"})
        for rel, body in files.items():
            write(self.root, rel, body)
        walker = find.Walker(self.root)
        walker.walk("")
        got = {e["path"]: e["feed"] for e in walker.entries}
        for kept in ("topics/a/runs/r1/notes/x.md", "topics/a/runs/r1/notes/deep/README.md",
                     "topics/a/runs/r1/notes/deep/raw.md", "topics/a/runs/r1/proto/README.md"):
            self.assertIn(kept, got)
        for dropped in ("topics/a/runs/r1/proto/notes.md", "topics/a/runs/r1/persist/db.md",
                        "topics/a/corpus/copy/doc.md", "topics/a/runs/r1/data/casea.md"):
            self.assertNotIn(dropped, got)
        self.assertEqual(got["topics/a/feedback/2026-11.md"], "operator")
        self.assertEqual(got["topics/a/README.md"], "(root)")
        self.assertEqual(got["topics/a/runs/r1/README.md"], "runs")
        self.assertEqual(walker.problems, [])

    def test_large_logs_keep_their_tail_with_real_line_numbers(self) -> None:
        rows = "when,event\n" + "2026-11-15,routine filler event\n" * 3000 + "2026-11-15,quasar outage detected\n"
        write(self.root, "topics/ops/log.csv", rows)
        write(self.root, "feeds.json", '{"max_file_kb": 8}')
        _, res = run(["quasar outage", "--topic", "ops", "--home", str(self.root)])
        self.assertEqual(res["hits"][0]["lineEnd"], 3002)

    def test_entry_files_outrank_the_evidence_about_them(self) -> None:
        write(self.root, "topics/ux/skills/present-decisions/SKILL.md", "# Usage\n\nTurn a redesign into decisions the operator answers on a phone; present each one with today beside it.\n")
        write(self.root, "topics/ux/skills/present-decisions/VALIDATION.md", "# Validation\n\nChecked the decisions page on a phone. Present and readable.\n")
        _, res = run(["present decisions", "--home", str(self.root)])
        self.assertEqual(res["hits"][0]["path"], "topics/ux/skills/present-decisions/SKILL.md")

    def test_unreachable_source_ledger_falls_back_to_files_and_says_so(self) -> None:
        write(self.root, "topics/ux/knowledge/MAP.md", "# Map\n\nThe planner owns week view.\n")
        env = {"SOURCE_LEDGER_API_BASE": "http://127.0.0.1:9", "VROOLI_RESEARCH_HOME": str(self.root)}
        code, res = run(["planner week"], env)
        self.assertEqual(code, 0)
        self.assertEqual(res["source"], "files")
        self.assertEqual(res["degradations"][0]["reason"], "SOURCE_LEDGER_UNAVAILABLE")
        self.assertEqual(res["hits"][0]["citation"], "topics/ux/knowledge/MAP.md#L1-L3")

    def test_coverage_counts_match_source_ledger(self) -> None:
        # Same fixture and expectations as TestCoverageIsCountedAndSummarisedSubtreesKeepTheirSummaries.
        write(self.root, "topics/a/runs/r1/notes/deep/raw.md", "dropped: depth 2\n")
        write(self.root, "topics/a/runs/r1/notes/deep/deeper/README.md", "summary three levels down\n")
        write(self.root, "topics/a/runs/r1/big.md", "long line of evidence\n" * 100)
        write(self.root, "topics/a/runs/r1/huge.json", '{"x":"' + "y" * 2048 + '"}')
        write(self.root, "topics/a/cohort/OPERATOR.md", "operator conversation\n")
        write(self.root, "topics/a/excluded/x.md", "excluded\n")
        write(self.root, "topics/a/feeds.json", '{"exclude":["excluded/*.md"]}')
        write(self.root, "feeds.json", '{"max_file_kb": 1, "max_json_kb": 1}')
        walker = find.Walker(self.root)
        walker.walk("")
        paths = {e["path"] for e in walker.entries}
        self.assertIn("topics/a/runs/r1/notes/deep/deeper/README.md", paths)
        self.assertNotIn("topics/a/runs/r1/notes/deep/raw.md", paths)
        self.assertEqual(walker.stats, {"private": 1, "excluded": 1, "summarised": 1, "oversizeJson": 1, "truncated": 1})
        _, res = run(["long line evidence", "--home", str(self.root)])
        self.assertTrue(res["hits"][0]["truncated"])

    def test_broken_policy_is_reported_and_never_widens_the_index(self) -> None:
        write(self.root, "topics/a/README.md", "# A\n\nshared marker word\n")
        write(self.root, "topics/a/corpus/copy.md", "shared marker word in an excluded corpus\n")
        write(self.root, "topics/a/feeds.json", '{"exclude": ["corpus/**"],')
        _, res = run(["shared marker", "--home", str(self.root)])
        self.assertEqual([h["path"] for h in res["hits"]], ["topics/a/README.md"])
        self.assertIn("feeds.json", res["degradations"][-1]["detail"])
        write(self.root, "feeds.json", "{nope")
        _, res = run(["shared marker", "--home", str(self.root)])
        self.assertTrue(res["hits"])
        self.assertIn("using defaults", res["degradations"][-1]["detail"])

    def test_private_files_are_never_served_unless_a_topic_opts_in(self) -> None:
        write(self.root, "cohorts/c1/STANCE-MESSAGE.txt", "zircon stance words\n")
        write(self.root, "topics/jobs/runs/r/resume-facts.md", "zircon resume facts\n")
        write(self.root, "topics/open/OPERATOR.md", "zircon operator notes\n")
        write(self.root, "topics/open/feeds.json", '{"private": []}')
        _, res = run(["zircon", "--home", str(self.root)])
        self.assertEqual([h["path"] for h in res["hits"]], ["topics/open/OPERATOR.md"])
        self.assertEqual(res["sync"]["skippedPrivate"], 2)

    def test_bad_usage(self) -> None:
        with contextlib.redirect_stderr(io.StringIO()):
            self.assertEqual(find.main(["!!", "--home", str(self.root)]), 2)
        with self.assertRaises(SystemExit):
            find.search_files(self.root, "x y", "../etc", "", 5)


if __name__ == "__main__":
    unittest.main()
