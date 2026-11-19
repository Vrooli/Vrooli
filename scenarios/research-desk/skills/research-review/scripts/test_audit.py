"""Tests for audit.py: run with python3 -I -m unittest test_audit.py."""
import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
sys.dont_write_bytecode = True
import audit  # noqa: E402

sys.path.insert(0, str(Path(__file__).resolve().parents[2] / "research-coordination" / "scripts"))
from test_mission_check import mission  # noqa: E402

MISSION = mission()


class AuditTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.home = Path(self.tmp.name)
        t = self.home / "topics" / "t"
        (t / "runs" / "r1").mkdir(parents=True)
        (t / "runs" / "r2").mkdir()
        (t / "runs" / "r2" / "CHECKPOINT.md").write_text("ok")
        (t / "history").mkdir()
        (t / "history" / "old.md").write_text("[gone](nowhere.md)")
        (t / "MISSION.md").write_text(MISSION)
        (t / "README.md").write_text("# T\n\n[mission](MISSION.md) [missing](NOPE.md) [web](https://x.y) "
                                     "[line](MISSION.md:12) [range](MISSION.md:3-9) [anchor](MISSION.md#end-state) [self](#top)\n")
        (t / "BIG.md").write_text("x" * 40_001)
        live = self.home / "cohorts" / "c1"  # a live commission on t, so its `active` row is honest
        live.mkdir(parents=True)
        (live / "PACKET.json").write_text('{"topics": ["t"]}')
        (live / "STATE.md").write_text("live")

    def tearDown(self):
        self.tmp.cleanup()

    def test_measure_counts_read_cost_and_clutter(self):
        m = audit.measure(self.home, "t")
        self.assertEqual(m["entry_bytes"], len(MISSION.encode()) + (self.home / "topics/t/README.md").stat().st_size)
        self.assertEqual(m["broken_links"], ["README.md -> NOPE.md"])  # history/ is not read
        self.assertEqual(m["oversize_pages"], ["BIG.md 40001"])
        self.assertEqual(m["runs"], 2)
        self.assertEqual(m["runs_without_guide"], ["r1"])
        self.assertEqual(m["markdown_outside_runs"]["files"], 3)
        self.assertEqual(m["mission"]["status"], "ok")
        self.assertIn("history/", m["top_level"])

    def test_owner_start_is_mission_plus_next_brief_else_newest_checkpoint(self):
        t = self.home / "topics/t"
        self.assertEqual(audit.measure(self.home, "t")["owner_start"]["files"],
                         {"MISSION.md": len(MISSION.encode()), "runs/r2/CHECKPOINT.md": 2})
        (t / "work" / "T1-api").mkdir(parents=True)
        (t / "work" / "T1-api" / "BRIEF.md").write_text("b" * 10)
        (t / "work" / "README.md").write_text("| Brief | Workstream | State |\n|---|---|---|\n| `T0` | T1 | planned |\n| [T1-api](T1-api/BRIEF.md) | T1 | ready |\n")
        self.assertEqual(audit.measure(self.home, "t")["owner_start"]["bytes"], len(MISSION.encode()) + 10)

    def test_compare_reports_change_and_keeps_throughput_beside_size(self):
        before = audit.measure(self.home, "t")
        (self.home / "topics/t/BIG.md").rename(self.home / "topics/t/history/BIG.md")
        after = audit.measure(self.home, "t")
        table = audit.compare(before, after)
        self.assertIn("| Pages over 40 KB | 1 | 0 | -100% |", table)
        self.assertIn("Landed workstreams", table)
        self.assertIn("Mission check: ok → ok", table)
        self.assertIn("Smaller overall: yes", table)  # BIG.md moved to history/

    def test_an_audit_that_grows_the_folder_outside_its_plan_is_not_smaller(self):
        t = self.home / "topics/t"
        before = first = audit.measure(self.home, "t")
        (t / "work" / "T1").mkdir(parents=True)
        (t / "work" / "T1" / "BRIEF.md").write_text("b" * 500)  # not netted, but bounded by the start-read cut
        self.assertIn("Smaller overall: no: work/ +500 B, more than the start-read cut of 0 B",
                      audit.compare(before, audit.measure(self.home, "t")))
        (t / "runs" / "r2" / "CHECKPOINT.md").write_text("ok" * 400)
        before = audit.measure(self.home, "t")  # an owner used to start from an 800-byte checkpoint
        (t / "runs" / "r2" / "CHECKPOINT.md").write_text("ok")
        table = audit.compare(before, audit.measure(self.home, "t"))
        self.assertIn("Smaller overall: yes", table)
        self.assertIn("| work/ Markdown (plan and briefs), bytes | 500 | 500 |", table)
        before = audit.measure(self.home, "t")
        (t / "NOTES.md").write_text("n" * 300)
        self.assertIn("Smaller overall: no: Markdown outside runs/, history/ and work/ +300 B",
                      audit.compare(before, audit.measure(self.home, "t")))
        old = {k: v for k, v in first.items() if k != "work_markdown"}  # a snapshot from before the field
        self.assertEqual(audit.outside_work(old), first["markdown_outside_runs"]["bytes"])  # it had no work/
        old["optional_parts"] = ["work"]
        self.assertIn("Smaller overall: not measured", audit.compare(old, audit.measure(self.home, "t")))

    def test_cli_writes_the_snapshot(self):
        out = self.home / "snap.json"
        audit.main(["measure", "t", "--home", str(self.home), "--out", str(out)])
        self.assertEqual(json.loads(out.read_text())["topic"], "t")


if __name__ == "__main__":
    unittest.main()
