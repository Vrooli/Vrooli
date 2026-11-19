"""Tests for new_topic.py: run with python3 -I -m unittest discover -s . -p 'test_new_topic.py'."""
import datetime as dt
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
sys.dont_write_bytecode = True
import mission_check as mc  # noqa: E402
import new_topic  # noqa: E402

DAY = dt.date(2026, 11, 15)


class NewTopicTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.home = Path(self.tmp.name)

    def tearDown(self):
        self.tmp.cleanup()

    def make(self, mode, name="t"):
        return new_topic.create(self.home, name, mode, "Things", '"Make it big."', "chat, 2026-11-15", DAY)

    def test_every_mode_creates_the_small_common_shape_plus_its_knowledge_page(self):
        pages = {"build": "knowledge/MAP.md", "propose-then-build": "knowledge/LANDSCAPE.md",
                 "gated-effect": "knowledge/RUNBOOK.md", "artifact": "knowledge/TASTE.md",
                 "evidence": "knowledge/CLAIMS.md"}
        for mode, page in pages.items():
            t = self.make(mode, name=mode)
            files = sorted(str(p.relative_to(t)) for p in t.rglob("*") if p.is_file())
            expected = sorted(["README.md", "MISSION.md", "LEDGER.md", "TOOLS.md", "INBOX.md", "work/README.md", page, "reporting/report.json"]
                              + (["ARTIFACTS.md"] if mode == "artifact" else []))
            self.assertEqual(files, expected, mode)
            text = "".join(p.read_text() for p in t.rglob("*") if p.is_file())
            self.assertNotIn("{{", text, mode)
            mission = (t / "MISSION.md").read_text()
            self.assertIn(f"Mode: {mode}", mission)
            self.assertIn('> "Make it big." — 2026-11-15, chat, 2026-11-15', mission)

    def test_an_intent_file_carries_every_quote_verbatim(self):
        words = self.home / "WORDS.md"
        words.write_text('# Words\n\n> "First, at length." — 2026-11-01, chat\n\nA note that is not a quote.\n\n'
                         '> "Second:\n>\n> two paragraphs." — 2026-11-02, review\n')
        t = new_topic.create(self.home, "f", "evidence", "Things", None, None, DAY, words)
        mission = (t / "MISSION.md").read_text()
        self.assertIn('> "First, at length." — 2026-11-01, chat\n\n> "Second:\n>\n> two paragraphs." — 2026-11-02, review', mission)
        self.assertNotIn("A note that is not a quote", mission)
        empty = self.home / "EMPTY.md"
        empty.write_text("no quotes here\n")
        with self.assertRaises(SystemExit):
            new_topic.create(self.home, "g", "build", "Things", None, None, DAY, empty)

    def test_a_fresh_template_fails_mission_check_until_filled(self):
        t = self.make("build")
        rep = mc.check_mission("t", t / "MISSION.md")
        self.assertTrue(any("unfilled template markers" in e for e in rep.errors), rep.errors)

    def test_refuses_an_existing_topic_and_a_bad_name(self):
        self.make("build")
        with self.assertRaises(SystemExit):
            self.make("build")
        with self.assertRaises(SystemExit):
            self.make("build", name="Bad Name")


if __name__ == "__main__":
    unittest.main()
