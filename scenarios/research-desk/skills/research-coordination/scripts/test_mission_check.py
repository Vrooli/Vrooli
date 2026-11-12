#!/usr/bin/env python3
"""Tests for mission_check.py. Run: python3 -m unittest test_mission_check (from this directory)."""

from __future__ import annotations

import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import mission_check as mc  # noqa: E402

WORKSTREAM = "| {id} | Land the thing end to end | Dim A | {hours} h · owner | Demo runs live | {status} |"


def mission(mode="build", end_state="Everything works end to end at full scale.", rows=None, intent='> "Make it big." — 2026-11-07, chat', review="2026-11-07 ACCEPT"):
    rows = rows if rows is not None else [
        WORKSTREAM.format(id="A1", hours="40–80", status="open"),
        WORKSTREAM.format(id="A2", hours="30–60", status="active run-1"),
        WORKSTREAM.format(id="A3", hours="20–40", status="landed ledger"),
        WORKSTREAM.format(id="A4", hours="25", status="approved"),
    ]
    dims = "\n".join(f"| D{i} | today | target |" for i in range(5))
    return f"""# T — mission

Mode: {mode}
Last architect pass: 2026-11-07 · Last ambition review: {review}

## Operator intent (verbatim)

{intent}

## End state

{end_state}

## Scorecard

| Dimension | Today | Target |
|---|---|---|
{dims}

## Workstreams

| ID | Destination | Dimensions | Magnitude | Showcase | Status |
|---|---|---|---|---|---|
{chr(10).join(rows)}

## Approval flow

Build directly.

## Real boundaries

- No money.

## Exemplar

Pinned unit.

## Not this

- Trap one.
- Trap two.
"""


class MissionCheckTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.home = Path(self.tmp.name)
        (self.home / "topics").mkdir()

    def tearDown(self):
        self.tmp.cleanup()

    def topic(self, name, mission_text=None, readme="See [mission](MISSION.md).\n"):
        d = self.home / "topics" / name
        d.mkdir()
        if mission_text is not None:
            (d / "MISSION.md").write_text(mission_text)
        (d / "README.md").write_text(readme)
        return d

    def run_check(self, *extra):
        return mc.main([str(self.home), *extra])

    def report(self, name):
        tdir = self.home / "topics" / name
        rep = mc.check_mission(name, tdir / "MISSION.md")
        mc.check_readme(rep, tdir / "README.md")
        return rep

    def test_a_mission_over_the_owner_reading_ceiling_is_a_warning_not_an_error(self):
        self.topic("big", mission(end_state="Everything works end to end at full scale. " * 1200))
        rep = self.report("big")
        ceiling = mc.mission_ceiling()
        self.assertTrue(ceiling and len((self.home / "topics/big/MISSION.md").read_bytes()) > ceiling)
        self.assertEqual(rep.errors, [])
        self.assertTrue(any(f"over the {ceiling:,} an owner should read at start" in w for w in rep.warnings))
        self.topic("small", mission())
        self.assertFalse(any("owner should read" in w for w in self.report("small").warnings))

    def test_waiting_statuses_are_valid_and_do_not_count_as_staffable(self):
        rows = [WORKSTREAM.format(id="A1", hours="40", status="waiting-operator 2026-11-10"),
                WORKSTREAM.format(id="A2", hours="40", status="waiting-time 2026-11-16"),
                WORKSTREAM.format(id="A3", hours="40", status="open")]
        self.topic("waits", mission(rows=rows))
        rep = self.report("waits")
        self.assertFalse([e for e in rep.errors if "status" in e], rep.errors)
        self.assertEqual((rep.count("waiting-operator"), rep.count("waiting-time")), (1, 1))
        self.assertEqual(rep.open_hours(), 40)
        self.assertTrue(any("at least three" in e for e in rep.errors), rep.errors)
        self.assertIn("| 1 | 0 | 2 | 0 |", mc.render_markdown([rep]))

    def test_valid_mission_passes_and_counts_portfolio(self):
        self.topic("good", mission())
        rep = self.report("good")
        self.assertEqual(rep.errors, [])
        self.assertEqual((rep.count("open"), rep.count("active"), rep.count("landed")), (1, 1, 1))
        self.assertEqual(rep.open_hours(), 40 + 30 + 25)
        self.assertEqual(self.run_check(), 0)

    def test_missing_mission_is_an_error(self):
        self.topic("bare")
        self.assertEqual(self.run_check(), 1)

    def test_symlinked_alias_is_skipped(self):
        self.topic("real", mission())
        (self.home / "topics" / "alias").symlink_to(self.home / "topics" / "real")
        self.assertEqual([p.name for p in mc.topics(self.home, None)], ["real"])

    def test_paraphrased_intent_without_quote_fails(self):
        self.topic("para", mission(intent="The operator wants a good foundation."))
        self.assertTrue(any("quoted operator words" in e for e in self.report("para").errors))

    def test_shrinking_end_state_fails_except_in_evidence_mode(self):
        shrink = "A decision-ready foundation for later work."
        self.topic("shrink", mission(end_state=shrink))
        self.assertTrue(any("decision-ready" in e for e in self.report("shrink").errors))
        self.topic("evid", mission(mode="evidence", end_state=shrink))
        self.assertFalse(any("decision-ready" in e for e in self.report("evid").errors))

    def test_fewer_than_three_live_workstreams_fails(self):
        rows = [WORKSTREAM.format(id="A1", hours="40", status="open"), WORKSTREAM.format(id="A2", hours="40", status="landed")]
        self.topic("thin", mission(rows=rows))
        self.assertTrue(any("at least three" in e for e in self.report("thin").errors))

    def test_task_sized_row_warns_and_missing_hours_fails(self):
        rows = [
            WORKSTREAM.format(id="A1", hours="4", status="open"),
            "| A2 | Thing | Dim | big | Demo | open |",
            WORKSTREAM.format(id="A3", hours="40", status="open"),
        ]
        self.topic("sizes", mission(rows=rows))
        rep = self.report("sizes")
        self.assertTrue(any("A1" in w and "task" in w for w in rep.warnings))
        self.assertTrue(any("A2" in e and "agent-hours" in e for e in rep.errors))

    def test_bad_mode_and_status_fail(self):
        rows = [WORKSTREAM.format(id=f"A{i}", hours="40", status="maybe") for i in range(3)]
        self.topic("bad", mission(mode="study", rows=rows))
        errs = self.report("bad").errors
        self.assertTrue(any("Mode line" in e for e in errs))
        self.assertTrue(any("status 'maybe'" in e for e in errs))

    def test_pending_review_warns(self):
        self.topic("pend", mission(review="pending (next commission)"))
        self.assertTrue(any("ambition review pending" in w for w in self.report("pend").warnings))

    def test_readme_receipts_and_missing_link(self):
        receipts = "".join(f"## Cohort update — 2026-11-0{i}\n\nx\n" for i in range(1, 5))
        self.topic("log", mission(), readme="# Topic\n\n" + receipts)
        rep = self.report("log")
        self.assertTrue(any("MISSION.md" in e for e in rep.errors))
        self.assertTrue(any("receipt headings" in w for w in rep.warnings))

    def test_topic_filter_and_markdown(self):
        self.topic("a", mission())
        self.topic("b")
        self.assertEqual(self.run_check("--topic", "a"), 0)
        table = mc.render_markdown([self.report("a")])
        self.assertIn("[a](topics/a/MISSION.md) | build | 1 | 1 | 0 | 1 | 1 |", table)

    def test_write_readme_replaces_only_the_marked_block(self):
        self.topic("a", mission())
        (self.home / "README.md").write_text(f"intro\n{mc.PORTFOLIO_START}\nstale\n{mc.PORTFOLIO_END}\noutro\n")
        self.assertEqual(self.run_check("--write-readme", "--markdown"), 0)
        text = (self.home / "README.md").read_text()
        self.assertNotIn("stale", text)
        self.assertIn("[a](topics/a/MISSION.md)", text)
        self.assertTrue(text.startswith("intro\n") and text.endswith("outro\n"))

    def test_json_lists_each_workstream_row_in_table_order(self):
        self.topic("rows", mission())
        rows = self.report("rows").as_dict()["workstream_rows"]
        self.assertEqual([(r["id"], r["status"]) for r in rows],
                         [("A1", "open"), ("A2", "active"), ("A3", "landed"), ("A4", "approved")])
        self.assertEqual(rows[0], {"id": "A1", "destination": "Land the thing end to end",
                                   "magnitude": "40–80 h · owner", "showcase": "Demo runs live", "status": "open"})

    def test_write_readme_without_markers_refuses(self):
        self.topic("a", mission())
        (self.home / "README.md").write_text("no markers\n")
        with self.assertRaises(SystemExit):
            self.run_check("--write-readme")
        self.assertEqual((self.home / "README.md").read_text(), "no markers\n")


if __name__ == "__main__":
    unittest.main()
