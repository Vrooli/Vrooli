"""Tests for the supervisor's research/plan-health program (research home `programs/`).

Runs the program through Program Runtime's own standalone runner, the path research takes when
Program Runtime is down. Skipped where that home or runner is absent (another machine).
"""
import datetime as dt
import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.dont_write_bytecode = True
REPO = Path(__file__).resolve().parents[5]
RUNNER = REPO / "scenarios" / "program-runtime" / "kernel" / "host"
PROGRAM = Path.home() / ".vrooli" / "research" / "programs" / "plan-health.json"
TODAY = dt.date.today()
OLD = (TODAY - dt.timedelta(days=40)).isoformat()

MISSION = """# T
## Workstreams
| ID | Destination | Dimensions | Magnitude | Showcase | Status |
|---|---|---|---|---|---|
| T1 | a | d | 40 h | s | active |
| T2 | b | d | 40 h | s | open |
| T3 | c | d | 40 h | s | landed |
"""
PLAN = f"""# Plan
| Brief | Workstream | State | Note |
|---|---|---|---|
| `T1-api` | T1 | ready | |
| `T9-ghost` | T9 | active | mentions T2 only in a note |
| `T1-later` | T1 | planned | |

## Risks

| Risk | Likelihood | Impact | Mitigation | Owner | Reviewed |
|---|---|---|---|---|---|
| Capture breaks | high | high | fixtures | owner | {OLD} |
| Scope shrinks | med | high | review | sup | {TODAY.isoformat()} |
"""


@unittest.skipUnless(PROGRAM.is_file() and (RUNNER / "standalone.py").is_file(), "plan-health program or runner absent")
class PlanHealthTest(unittest.TestCase):
    def setUp(self):
        sys.path.insert(0, str(RUNNER))
        import standalone  # noqa: E402
        self.standalone = standalone
        self.tmp = tempfile.TemporaryDirectory()
        self.home = Path(self.tmp.name)
        for name in ("planned", "bare"):
            (self.home / "topics" / name).mkdir(parents=True)
            (self.home / "topics" / name / "MISSION.md").write_text(MISSION)
        work = self.home / "topics" / "planned" / "work"
        (work / "T1-api").mkdir(parents=True)
        (work / "T1-api" / "BRIEF.md").write_text("brief")
        (work / "README.md").write_text(PLAN)
        (self.home / "topics" / "planned" / "INBOX.md").write_text(f"# Inbox\n\n### {OLD}T05:00Z · from supervisor · priority\nDo X first.\n")

    def tearDown(self):
        self.tmp.cleanup()

    def run_program(self, **inputs):
        contract = json.loads(PROGRAM.read_text())
        text, envelope = self.standalone.execute(str(PROGRAM.with_suffix(".py")), {"research_root": str(self.home), **inputs}, contract)
        return envelope or json.loads(text.strip().splitlines()[-1])

    def test_reads_the_plan_against_the_mission(self):
        e = self.run_program()
        self.assertEqual(e["status"], "ok")
        s = e["signals"]
        self.assertEqual((s["topics"], s["with_plan"], s["live_workstreams"]), (2, 1, 4))
        rows = {r["topic"]: r for r in s["attention"]}
        self.assertIn("no plan: 2 live workstreams and no work/README.md", rows["bare"]["findings"])
        found = " | ".join(rows["planned"]["findings"])
        self.assertIn("1 live workstreams not in the plan: T2", found)   # a note is not coverage
        self.assertIn("1 briefs due with no work/<ID>/BRIEF.md: T9-ghost", found)  # planned T1-later may wait
        self.assertEqual(s["planned_unwritten"], 1)
        self.assertIn("1 risks not reviewed in 14 days: Capture breaks", found)
        self.assertIn("inbox holds 1 undrained entries, oldest 40 days", found)
        self.assertEqual(s["undrained_inbox"], 1)

    def test_the_next_brief_must_be_written_even_when_planned(self):
        plan = self.home / "topics" / "planned" / "work" / "README.md"
        plan.write_text(PLAN.replace("| `T1-api` | T1 | ready | |", "| `T0-first` | T1 | planned | |\n| `T1-api` | T1 | ready | |"))
        rows = {r["topic"]: r for r in self.run_program()["signals"]["attention"]}
        self.assertIn("2 briefs due with no work/<ID>/BRIEF.md: T0-first, T9-ghost", " | ".join(rows["planned"]["findings"]))

    def test_a_retired_topic_is_not_read(self):
        (self.home / "topics" / "bare" / "MISSION.md").write_text("# T\nMode: build\nRetired: 2026-11-15 — done\n")
        self.assertEqual(self.run_program()["signals"]["topics"], 1)

    def test_rejects_a_relative_root(self):
        contract = json.loads(PROGRAM.read_text())
        _, e = self.standalone.execute(str(PROGRAM.with_suffix(".py")), {"research_root": "relative"}, contract)
        self.assertEqual(e["status"], "failed")


if __name__ == "__main__":
    unittest.main()
