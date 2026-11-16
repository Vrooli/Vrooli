import datetime as dt
import importlib.util
import json
import socket
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("needs_you.py")
CASES = json.loads(Path(__file__).with_name("needs_you_cases.json").read_text(encoding="utf-8"))["cases"]
SPEC = importlib.util.spec_from_file_location("needs_you", SCRIPT)
assert SPEC and SPEC.loader
needs_you = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(needs_you)


class GrammarCases(unittest.TestCase):
    """The shared cases; research-desk's Go parser runs the same file."""

    def test_every_case(self):
        for case in CASES:
            with self.subTest(case["name"]):
                items, problems = needs_you.parse(case["text"], case.get("section", "Needs you"))
                self.assertEqual(items, case["items"])
                self.assertEqual(len(problems), len(case["problems"]), problems)
                for problem, want in zip(problems, case["problems"]):
                    self.assertIn(want, problem)


class Write(unittest.TestCase):
    def run_write(self, draft: str) -> tuple[subprocess.CompletedProcess, str]:
        with tempfile.TemporaryDirectory() as directory:
            source, target = Path(directory, "draft.md"), Path(directory, "STATE.md")
            source.write_text(draft)
            target.write_text("old state\n")
            result = subprocess.run([sys.executable, "-I", str(SCRIPT), "write", "--file", str(source), "--section", "Needs you", "--target", str(target)],
                                    capture_output=True, text=True, check=False)
            return result, target.read_text()

    def test_a_rejected_draft_leaves_the_target_alone_and_says_how_to_fix_it(self):
        for case in (c for c in CASES if c["problems"] and "section" not in c):
            with self.subTest(case["name"]):
                result, target = self.run_write(case["text"])
                self.assertEqual(result.returncode, 1)
                self.assertIn(case["problems"][0], result.stderr)
                self.assertEqual(target, "old state\n")

    def test_a_valid_draft_replaces_the_target(self):
        draft = CASES[0]["text"]
        result, target = self.run_write(draft)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(target, draft)


def write(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(text, encoding="utf-8")


def item(item_id: str, packet: str = "", extra: str = "") -> str:
    lines = [f"### {item_id} — Pick for {item_id}?", "Plain words.", "Recommended: yes — Yes", "Options: yes — Yes; no — No"]
    return "\n".join(lines + ([f"Packet: {packet}"] if packet else []) + ([extra] if extra else [])) + "\n"


class OpenDecisions(unittest.TestCase):
    """`open` answers "what decisions are open?" across every level from files, with every service down."""

    def setUp(self):
        self.root = Path(tempfile.mkdtemp())
        self.research, self.portfolio = self.root / "research", self.root / "portfolio"
        r = self.research
        write(r / "topics/t1/DECISIONS.md", "# Decisions\n\n## Needs you\n\n" + item("A", "work/A/BRIEF.md") + item("B", "work/B/BRIEF.md")
              + item("OLD-1", "work/OLD-1/BRIEF.md"))
        write(r / "topics/t1/history/INBOX.md", "# Inbox history\n\n### 2026-11-15T09:00:00Z · from operator · answer\nB: no — No\nSource: hub ask x\n")
        # An older desk entry names its item inside the sentence; it still answers it.
        write(r / "topics/t1/INBOX.md", "# Inbox\n\n### 2026-11-15T06:34:18Z · from research-desk · answer\nThe operator answered (item OLD-1 of c, Hub ask y)\n")
        write(r / "topics/t2/STATE.md", "# State\n\n## Needs you\n\n" + item("C", "OPERATOR-INPUTS.md"))
        write(r / "ATTENTION.md", "# Attention\n\n## Actionable now\n\n" + item("S", "work/S/BRIEF.md") + item("S2", "work/S2/BRIEF.md")
              + item("A", "work/A/BRIEF.md"))
        write(r / "INBOX.md", "# Inbox\n\n### 2026-11-15T09:00:00Z · from operator · answer\nS2: yes — Yes\nSource: hub ask z\n")
        future = (dt.datetime.now(dt.timezone.utc) + dt.timedelta(hours=3)).strftime("%Y-%m-%dT%H:%M:%SZ")
        write(r / "cohorts/c1/STATE.md", f"# State\n\n## Needs you\n\n{item('D', '/abs/work/D/BRIEF.md')}{item('E', '/abs/work/E/BRIEF.md')}{item('A', '/abs/A/BRIEF.md')}\n- Allowance ends: {future}\n")
        write(r / "cohorts/c1/OPERATOR.md", "# Operator — c1\n\n### 2026-11-15T10:00:00Z · answer E · via hub\nyes — Yes\n")
        write(r / "cohorts/c1/PACKET.json", json.dumps({"spec": {"topics": ["t1"]}}))
        write(r / "cohorts/c0/STATE.md", "# State\n\n## Needs you\n\n" + item("F") + "\n- Allowance ends: 2026-11-01T00:00:00Z\n")
        write(r / "cohorts/cx/STATE.md", "# State\n\n## Needs you\n\n" + item("G"))
        rows = [{"id": "ask-1", "key": "S-1", "question": "Ship it?", "state": "sent", "recommend": "yes", "explainer": "/p/page.html"},
                {"id": "ask-2", "key": "S-2", "question": "Answered", "state": "sent", "answer": "yes"},
                {"id": "ask-3", "key": "S-3", "question": "Old form", "state": "sent"},
                {"id": "ask-3", "key": "S-3", "question": "Replaced", "state": "replaced"}]
        write(self.portfolio / "steward/state/asks.jsonl", "\n".join(json.dumps(row) for row in rows) + "\n")
        (self.portfolio / "steward/delegations/ask-1").mkdir(parents=True)

    def report(self) -> dict:
        def refuse(*_args, **_kwargs):
            raise OSError("every service is down")
        connect, create = socket.socket.connect, socket.create_connection
        socket.socket.connect, socket.create_connection = refuse, refuse
        try:
            return needs_you.open_decisions(self.research, self.portfolio, dt.datetime.now(dt.timezone.utc))
        finally:
            socket.socket.connect, socket.create_connection = connect, create

    def test_every_level_answers_from_files_with_services_down(self):
        report = self.report()
        got = {level["level"]: [i["id"] for i in level["items"]] for level in report["levels"]}
        self.assertEqual(got, {"ceo": ["S-1"], "supervisor": ["S"], "topic:t1": ["A"], "topic:t2": ["C"], "commission:c1": ["D"]})
        self.assertEqual(report["open"], 5)
        self.assertEqual(report["not_counted"], ["cx"], "a commission whose files do not say whether it is live is named, not guessed")
        a = report["levels"][2]["items"][0]
        self.assertEqual(a["packet"], str(self.research / "topics/t1/work/A/BRIEF.md"))
        self.assertEqual(len(a["also_in"]), 2, "a commission's copy and the supervisor's copy of a topic item are one decision")
        ceo = report["levels"][0]
        self.assertEqual(ceo["items"][0]["packet"], str(self.portfolio / "steward/delegations/ask-1"))
        self.assertEqual(ceo["items"][0]["page"], "/p/page.html")
        c = report["levels"][3]["items"][0]
        self.assertIn("older topic STATE.md", c["legacy"])
        self.assertTrue(any("must name the decision's brief" in p for p in c["problems"]))

    def test_the_command_prints_every_level_and_json(self):
        args = [sys.executable, "-I", str(SCRIPT), "open", "--research", str(self.research), "--portfolio", str(self.portfolio)]
        text = subprocess.run(args, capture_output=True, text=True, check=True).stdout
        for want in ("Open decisions, from files only", "ceo · ", "supervisor · ", "topic:t1 · ", "answers land in: " + str(self.research / "topics/t1/INBOX.md"),
                     "packet: " + str(self.research / "topics/t1/work/A/BRIEF.md"), "research-desk needs-you"):
            self.assertIn(want, text)
        report = json.loads(subprocess.run(args + ["--json"], capture_output=True, text=True, check=True).stdout)
        self.assertEqual(report["open"], 5)


if __name__ == "__main__":
    unittest.main()
