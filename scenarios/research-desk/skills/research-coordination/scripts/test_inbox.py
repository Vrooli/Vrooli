"""Tests for inbox.py: python3 -I test_inbox.py."""
import contextlib
import datetime as dt
import io
import re
import sys
import tempfile
import unittest
from pathlib import Path

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parent))
import inbox  # noqa: E402


def run(*args):
    out, err = io.StringIO(), io.StringIO()
    with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
        code = inbox.main(list(args))
    return code, out.getvalue(), err.getvalue()


class InboxTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.dir = Path(self.tmp.name) / "topic"
        self.dir.mkdir()

    def tearDown(self):
        self.tmp.cleanup()

    def test_append_stamps_the_real_clock_and_drain_moves_the_entry_with_what_was_done(self):
        before = dt.datetime.now(dt.timezone.utc).replace(microsecond=0)
        code, out, _ = run("append", str(self.dir), "--from", "supervisor", "--kind", "friction", "--body", "Tool X is slow.")
        self.assertEqual(code, 0)
        stamp = re.match(r"### (\S+) ·", out).group(1)
        at = dt.datetime.strptime(stamp, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=dt.timezone.utc)
        self.assertLessEqual(abs((at - before).total_seconds()), 5)  # never a typed time
        run("append", str(self.dir), "--from", "topic:music-quality", "--kind", "feedback", "--body", "Second.",
            "--source", "operator chat (session 6b8b, 2026-11-15T09:05Z)", "--about", "the music owner")
        self.assertIn("Second.\nSource: operator chat (session 6b8b, 2026-11-15T09:05Z)\nAbout: the music owner",
                      (self.dir / "INBOX.md").read_text())
        code, out, _ = run("list", str(self.dir))
        self.assertEqual(len(out.splitlines()), 2)
        code, _, _ = run("drain", str(self.dir), "--entry", stamp, "--did", "Filed as risk R3.")
        self.assertEqual(code, 0)
        left = (self.dir / "INBOX.md").read_text()
        self.assertNotIn("Tool X is slow.", left)
        self.assertIn("Second.", left)
        hist = (self.dir / "history" / "INBOX.md").read_text()
        self.assertIn("Tool X is slow.", hist)
        self.assertRegex(hist, r"Acted: Filed as risk R3\. \(\S+Z\)")

    def test_entries_written_elsewhere_with_one_stamp_drain_by_position(self):
        (self.dir / "INBOX.md").write_text("# Inbox\n\n### 2026-11-15T06:34:18Z · from ceo · answer\nFirst.\n\n"
                                           "### 2026-11-15T06:34:18Z · from ceo · answer\nSecond.\n")
        self.assertEqual(run("drain", str(self.dir), "--entry", "2026-11-15T06:34:18Z", "--did", "x")[0], 1)
        self.assertEqual(run("drain", str(self.dir), "--entry", "2026-11-15T06:34:18Z", "--nth", "2", "--did", "Recorded.")[0], 0)
        self.assertIn("First.", (self.dir / "INBOX.md").read_text())
        self.assertNotIn("Second.", (self.dir / "INBOX.md").read_text())

    def test_a_malformed_heading_lists_as_malformed_and_drains_verbatim_by_its_stamp(self):
        (self.dir / "INBOX.md").write_text("# Inbox\n\n### 2026-11-15T17:21:00Z · EX7 · endpoint-inventory handoff\nBody.\n")
        self.assertIn("MALFORMED ### 2026-11-15T17:21:00Z · EX7", run("list", str(self.dir))[1])
        self.assertEqual(run("drain", str(self.dir), "--entry", "2026-11-15T17:21:00Z", "--did", "Handed to EX7.")[0], 0)
        self.assertNotIn("Body.", (self.dir / "INBOX.md").read_text())
        hist = (self.dir / "history" / "INBOX.md").read_text()
        self.assertIn("### 2026-11-15T17:21:00Z · EX7 · endpoint-inventory handoff\nBody.\nActed: Handed to EX7.", hist)

    def test_bad_level_kind_or_heading_in_body_is_refused(self):
        self.assertEqual(run("append", str(self.dir), "--from", "Bad Level", "--kind", "feedback", "--body", "x")[0], 1)
        self.assertEqual(run("append", str(self.dir), "--from", "topic", "--kind", "feedback", "--body", "### 2026 · fake")[0], 1)
        with contextlib.redirect_stderr(io.StringIO()), self.assertRaises(SystemExit):
            inbox.main(["append", str(self.dir), "--from", "topic", "--kind", "gossip", "--body", "x"])
        self.assertFalse((self.dir / "INBOX.md").exists())

    def test_a_verdict_needs_a_source_and_records_it(self):
        with contextlib.redirect_stderr(io.StringIO()), self.assertRaises(SystemExit):
            inbox.main(["verdict", str(self.dir), "--topic", "music-quality", "--body", "Keep going."])
        self.assertEqual(run("verdict", str(self.dir), "--topic", "music-quality", "--source", "x", "--body", "Keep.")[0], 1)
        code, _, _ = run("verdict", str(self.dir), "--topic", "music-quality",
                         "--source", "Hub answer 7f3c-01 (2026-11-15)", "--body", "Keep going.")
        self.assertEqual(code, 0)
        text = (self.dir / "VERDICTS.md").read_text()
        self.assertRegex(text, r"### \S+Z · verdict music-quality · via hand\nKeep going\.\nSource: Hub answer 7f3c-01")


if __name__ == "__main__":
    unittest.main()
