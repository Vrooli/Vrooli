import datetime as dt
import importlib.util
import json
import socket
import subprocess
import sys
import tempfile
import unittest
from unittest.mock import patch
from pathlib import Path

SCRIPT = Path(__file__).with_name("needs_you.py")
HERE = Path(__file__).parent
CASES = json.loads(Path(__file__).with_name("needs_you_cases.json").read_text(encoding="utf-8"))["cases"]
BRIEFS = json.loads(Path(__file__).with_name("needs_you_cases.json").read_text(encoding="utf-8"))["briefs"]
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


    def test_every_brief_case(self):
        for case in BRIEFS:
            with self.subTest(case["name"]):
                now = dt.datetime.fromisoformat(case["now"].replace("Z", "+00:00")) if case.get("now") else None
                problems = needs_you.brief_problems(case["brief"], case["keys"], now)
                self.assertEqual(len(problems), len(case["problems"]), problems)
                for problem, want in zip(problems, case["problems"]):
                    self.assertIn(want, problem)


# CASES[0] with its decision's folder beside the level file, and that folder made ready.
READY = CASES[0]["text"].replace("/abs/", "")


def ready_packets(directory: str, brief: str | None = None, work_id: str = "AV4") -> None:
    folder = Path(directory, "work", work_id)
    folder.mkdir(parents=True, exist_ok=True)
    today = dt.date.today().isoformat()
    (folder / "BRIEF.md").write_text(brief if brief is not None else
                                     f"Checked: {today} the choice is still open, read in the topic's plan\n\n## On each answer\n"
                                     "- `keep`: keep it.\n- `wait`: nothing yet.\n- `other`: the operator's words are the instruction.\n")
    (folder / "page.html").write_text("<html></html>")


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

    def test_a_valid_draft_into_an_empty_target_becomes_the_file(self):
        draft = READY
        with tempfile.TemporaryDirectory() as directory:
            ready_packets(directory)
            source, target = Path(directory, "draft.md"), Path(directory, "DECISIONS.md")
            source.write_text(draft)
            result = self.write_cli(source, target)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(target.read_text(), draft)

    def test_failed_atomic_replace_keeps_the_previous_file(self):
        with tempfile.TemporaryDirectory() as directory:
            target = Path(directory, "DECISIONS.md")
            target.write_text("previous decisions\n")
            with patch.object(needs_you.os, "replace", side_effect=OSError("replace failed")):
                with self.assertRaisesRegex(OSError, "replace failed"):
                    needs_you.write_atomically(target, "new decisions\n")
            self.assertEqual(target.read_text(), "previous decisions\n")
            self.assertEqual([p.name for p in Path(directory).iterdir()], ["DECISIONS.md"])

    def test_a_decision_without_its_ready_folder_is_refused(self):
        # O7(f): F7's packet was a workstream brief, PF-KEYS asked for keys nobody had checked.
        for brief, want in ((None, "page"), ("# W1 — Build it\n", "no `Checked:"), ("", "")):
            with self.subTest(want or "no folder"), tempfile.TemporaryDirectory() as directory:
                if want:
                    ready_packets(directory, brief)
                    if want == "page":
                        Path(directory, "work/AV4/page.html").unlink()
                source, target = Path(directory, "draft.md"), Path(directory, "DECISIONS.md")
                source.write_text(READY)
                result = self.write_cli(source, target)
                self.assertEqual(result.returncode, 1)
                self.assertIn(want or "BRIEF.md does not exist", result.stderr)
                self.assertFalse(target.exists())

    def test_a_decision_says_what_silence_does_or_none_for_a_bright_line(self):
        for line, ok in (("", False), ("If silent: keep on 2026-12-01\n", True), ("If silent: none\n", True),
                         ("If silent: keep\n", False), ("If silent: on 2026-12-01\n", False)):
            with self.subTest(line), tempfile.TemporaryDirectory() as directory:
                ready_packets(directory)
                draft = "\n".join(l for l in READY.splitlines() if not l.startswith("If silent:")) + "\n"
                draft = draft.replace("Packet:", line + "Packet:", 1)
                source, target = Path(directory, "draft.md"), Path(directory, "DECISIONS.md")
                source.write_text(draft)
                result = self.write_cli(source, target)
                self.assertEqual(result.returncode, 0 if ok else 1, result.stderr)
                if not ok:
                    self.assertIn("If silent:", result.stderr)

    def test_silence_is_asked_only_of_items_being_added(self):
        # A write replaces the section, so an old item travels with the new one: it is not asked again.
        with tempfile.TemporaryDirectory() as directory:
            ready_packets(directory)
            source, target = Path(directory, "draft.md"), Path(directory, "DECISIONS.md")
            old = "\n".join(l for l in READY.splitlines() if not l.startswith("If silent:")) + "\n"
            target.write_text(old)
            source.write_text(old)
            result = self.write_cli(source, target)
            self.assertEqual(result.returncode, 0, result.stderr)
            added = old + "\n### AV9 — Another\nRecommended: keep — Keep.\nOptions: keep — Keep.\nPacket: work/AV9/BRIEF.md\nExplainer: work/AV9/page.html\n"
            source.write_text(added)
            ready_packets(directory, work_id="AV9")
            result = self.write_cli(source, target)
            self.assertEqual(result.returncode, 1)
            self.assertIn("AV9: a decision being added needs `If silent:", result.stderr)
            self.assertNotIn("AV4:", result.stderr)

    def test_an_option_named_other_is_dropped_with_a_warning_and_never_refuses(self):
        with tempfile.TemporaryDirectory() as directory:
            ready_packets(directory)
            source, target = Path(directory, "draft.md"), Path(directory, "DECISIONS.md")
            source.write_text(READY.replace("wait — Wait.", "wait — Wait; other — Something else.", 1))
            result = self.write_cli(source, target)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("warning:", result.stderr)
            self.assertIn("other", result.stderr)

    def test_a_new_item_under_an_answered_id_is_refused(self):
        # The courier never delivers an answered ID again, so a write would leave an invisible
        # question whose silence default still fires.
        for history in (False, True):
            with self.subTest(history=history), tempfile.TemporaryDirectory() as directory:
                ready_packets(directory)
                source, target = Path(directory, "draft.md"), Path(directory, "DECISIONS.md")
                inbox = Path(directory, "history", "INBOX.md") if history else Path(directory, "INBOX.md")
                inbox.parent.mkdir(exist_ok=True)
                inbox.write_text("### 2026-11-15T09:00Z · from operator · answer\nAV4: keep — Keep.\nSource: hub ask a1\n")
                source.write_text(READY)
                result = self.write_cli(source, target)
                self.assertEqual(result.returncode, 1, result.stderr)
                self.assertIn("already holds an answer to that ID", result.stderr)
                self.assertFalse(target.exists())

    def write_cli(self, source: Path, target: Path) -> subprocess.CompletedProcess:
        return subprocess.run([sys.executable, "-I", str(SCRIPT), "write", "--file", str(source), "--section", "Needs you", "--target", str(target)],
                              capture_output=True, text=True, check=False)


# A commission STATE.md as charters render it: the section sits between other sections the
# supervisor owns. A write must change the section and keep every other byte.
STATE = """# State — ee-forcing-functions-20261115

## Needs you

No actionable items are recorded.

## Card 1 — Experience engineering and production UX

- Owner: EX7. Next: usage contracts for AppShell.
- Proof due 2026-11-16.

## Prior review record

- 10:49:58Z review: two items delivered.
"""
NEW_SECTION = "## Needs you\n\n" + READY.split("## Needs you\n", 1)[1].lstrip("\n")


class SectionWrite(unittest.TestCase):
    def write_into(self, target_text: str, draft: str) -> tuple[subprocess.CompletedProcess, str]:
        with tempfile.TemporaryDirectory() as directory:
            source, target = Path(directory, "draft.md"), Path(directory, "STATE.md")
            ready_packets(directory)
            source.write_text(draft)
            target.write_text(target_text)
            result = subprocess.run([sys.executable, "-I", str(SCRIPT), "write", "--file", str(source), "--section", "Needs you", "--target", str(target)],
                                    capture_output=True, text=True, check=False)
            return result, target.read_text()

    def test_only_the_section_changes_in_a_real_state_file(self):
        result, after = self.write_into(STATE, NEW_SECTION)
        self.assertEqual(result.returncode, 0, result.stderr)
        before_head, before_tail = STATE.split("## Card 1", 1)
        after_head, after_tail = after.split("## Card 1", 1)
        self.assertEqual(before_tail, after_tail, "everything after the section is byte-identical")
        self.assertTrue(after_head.startswith("# State — ee-forcing-functions-20261115\n\n## Needs you\n"))
        self.assertIn("### AV4 — Pick one", after_head)
        self.assertNotIn("No actionable items are recorded.", after_head)

    def test_a_commissions_status_lines_survive_an_items_only_draft(self):
        # A copy of the live ee-forcing-functions-20261115 STATE.md (17:30Z): its status lines sit inside Needs you.
        live = (HERE / "testdata/ee-commission-STATE.md").read_text()
        status = [line for line in live.split("## Card 1")[0].splitlines() if line.startswith("- ")]
        self.assertGreaterEqual(len(status), 8)
        result, after = self.write_into(live, NEW_SECTION)
        self.assertEqual(result.returncode, 0, result.stderr)
        section = after.split("## Card 1")[0]
        self.assertEqual([line for line in section.splitlines() if line.startswith("- ")], status, "every status line kept, in order")
        self.assertIn("### AV4 — Pick one", section)
        self.assertNotIn("No unanswered decisions.", section)
        self.assertEqual(after.split("## Card 1")[1], live.split("## Card 1")[1])
        result, again = self.write_into(after, "# State\n\n## Needs you\n\nNo unanswered decisions.\n")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(again, live, "writing the placeholder back restores the original byte for byte")

    def test_a_draft_that_would_drop_a_status_line_is_refused(self):
        live = (HERE / "testdata/ee-commission-STATE.md").read_text()
        draft = "# State\n\n## Needs you\n\nNo unanswered decisions.\n\n- Next review: 2026-11-15T19:00:00Z\n"
        result, after = self.write_into(live, draft)
        self.assertEqual(result.returncode, 1)
        self.assertIn("Allowance ends", result.stderr)
        self.assertEqual(after, live, "nothing written")
        kept = live.split("## Card 1")[0].split("No unanswered decisions.\n", 1)[1]
        updated = kept.replace("- Next review: 2026-11-15T18:04:05Z", "- Next review: 2026-11-15T19:00:00Z")
        result, after = self.write_into(live, "# State\n\n## Needs you\n\nNo unanswered decisions.\n" + updated)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("- Next review: 2026-11-15T19:00:00Z", after)
        self.assertNotIn("18:04:05Z", after, "a status line is updated by its key, not duplicated")

    def test_a_whole_file_draft_changes_only_the_section(self):
        draft = STATE.replace("No actionable items are recorded.\n", NEW_SECTION.split("\n\n", 1)[1]).replace("Proof due", "EDITED OUTSIDE")
        result, after = self.write_into(STATE, draft)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("Proof due 2026-11-16.", after, "edits outside the section are not applied")
        self.assertIn("### AV4 — Pick one", after)

    def test_a_missing_section_is_appended(self):
        target = STATE.replace("## Needs you\n\nNo actionable items are recorded.\n\n", "")
        result, after = self.write_into(target, NEW_SECTION)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertTrue(after.startswith(target.rstrip("\n")))
        self.assertTrue(after.rstrip().endswith(NEW_SECTION.rstrip()))

    def test_a_doubled_section_is_refused_and_the_file_kept(self):
        target = STATE + "\n## Needs you\n\nA second copy.\n"
        result, after = self.write_into(target, NEW_SECTION)
        self.assertEqual(result.returncode, 1)
        self.assertIn("2 times", result.stderr)
        self.assertEqual(after, target)

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
        rows = [{"id": "ask-1", "key": "S-1", "question": "Ship it?", "state": "queued", "packet": "steward/asks/S-1/BRIEF.md"},
                {"id": "ask-1", "key": "S-1", "question": "Ship it?", "state": "sent", "recommend": "yes", "explainer": "/p/page.html"},
                {"id": "ask-2", "key": "S-2", "question": "Answered", "state": "sent", "answer": "yes"},
                {"id": "ask-3", "key": "S-3", "question": "Old form", "state": "sent"},
                {"id": "ask-3", "key": "S-3", "question": "Replaced", "state": "replaced"}]
        write(self.portfolio / "steward/state/asks.jsonl", "\n".join(json.dumps(row) for row in rows) + "\n")

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
        self.assertEqual(ceo["items"][0]["packet"], str(self.portfolio / "steward/asks/S-1/BRIEF.md"),
                         "the ask's own packet field, even when a later row of its history omits it")
        self.assertEqual(ceo["items"][0]["page"], "/p/page.html")
        c = report["levels"][3]["items"][0]
        self.assertIn("older topic STATE.md", c["legacy"])
        self.assertTrue(any("must name the decision's brief" in p for p in c["problems"]))

    def test_an_answer_nobody_has_drained_is_shown_with_its_spawn_line(self):
        t3 = self.research / "topics/t3"
        write(t3 / "INBOX.md", "# Inbox\n\n### 2026-11-15T06:00:00Z · from research-desk · answer\nW1: go — Go ahead\n"
              "Packet: work/W1/BRIEF.md\nSource: hub ask q\n")
        write(t3 / "DECISIONS.md", "# Decisions\n\n## Needs you\n\n" + item("W1", str(t3 / "work/W1/BRIEF.md")))
        level = next(l for l in self.report()["levels"] if l["level"] == "topic:t3")
        self.assertEqual(level["items"], [], "an answered item is no longer open")
        self.assertEqual(len(level["waiting"]), 1)
        self.assertIsNone(level["waiting"][0]["spawn"], "no brief, no message")
        self.assertIn("no brief at work/W1/BRIEF.md", level["waiting"][0]["problems"][0])
        write(t3 / "work/W1/BRIEF.md", f"Checked: {dt.date.today()} it holds, read just now\n\n## On each answer\n- `go`: go.\n- `other`: theirs.\n\n## Stance\n> \"x\"\n")
        level = next(l for l in self.report()["levels"] if l["level"] == "topic:t3")
        self.assertTrue(level["waiting"][0]["spawn"].startswith(f"Execute the approved decision `W1`. Read `{t3}/work/W1/BRIEF.md`; the answer is `go`."))
        self.assertIsNone(level["waiting"][0]["read_by"], "no live commission covers t3")

    def test_an_answer_without_a_packet_has_a_stable_refusal_instead_of_missing_diagnostics(self):
        t3 = self.research / "topics/t3"
        write(t3 / "INBOX.md", "# Inbox\n\n### 2026-11-15T06:00:00Z · from operator · answer\nW1: go — Go ahead\n")
        level = next(l for l in self.report()["levels"] if l["level"] == "topic:t3")
        waiting = level["waiting"][0]
        self.assertIsNone(waiting["spawn"])
        self.assertIn("answer names no packet", waiting["problems"][0])
        result = subprocess.run([sys.executable, "-I", str(SCRIPT), "spawn", "W1", "--research", str(self.research), "--portfolio", str(self.portfolio)],
                                capture_output=True, text=True)
        self.assertEqual(result.returncode, 1)
        self.assertIn("W1 cannot be executed cold:", result.stderr)
        self.assertIn("answer names no packet", result.stderr)
        self.assertNotIn("Traceback", result.stderr)

    def test_the_command_prints_every_level_and_json(self):
        args = [sys.executable, "-I", str(SCRIPT), "open", "--research", str(self.research), "--portfolio", str(self.portfolio)]
        text = subprocess.run(args, capture_output=True, text=True, check=True).stdout
        for want in ("Open decisions, from files only", "ceo · ", "supervisor · ", "topic:t1 · ", "answers land in: " + str(self.research / "topics/t1/INBOX.md"),
                     "packet: " + str(self.research / "topics/t1/work/A/BRIEF.md"), "research-desk needs-you"):
            self.assertIn(want, text)
        report = json.loads(subprocess.run(args + ["--json"], capture_output=True, text=True, check=True).stdout)
        self.assertEqual(report["open"], 5)

class Spawn(unittest.TestCase):
    """O7(f): hand-typed spawns dropped the stance (U77) or ran on a stale packet (U72); the message is printed."""

    def folder(self, root: Path, checked: str, stance: str = "") -> Path:
        brief = root / "topics/t1/work/X1/BRIEF.md"
        write(brief, f"Checked: {checked} the premise holds, read in the plan\n\n## On each answer\n- `go`: do it.\n"
                     f"- `other`: the operator's words.\n{stance}")
        return brief

    def test_the_message_carries_the_stance_and_asks_for_a_recheck_when_stale(self):
        now = dt.datetime(2026, 11, 15, 22, 0, tzinfo=dt.timezone.utc)
        with tempfile.TemporaryDirectory() as d:
            brief = self.folder(Path(d), "2026-11-15T20:00Z")
            message, problems = needs_you.spawn_message(brief, "X1", "go", now)
            self.assertEqual(message, None)
            self.assertIn("no stance", problems[0])
            write(Path(d, "topics/t1/MISSION.md"), "# M\n\n## Operator intent (verbatim)\n\n> \"make it real\"\n\n## End state\nx\n")
            message, problems = needs_you.spawn_message(brief, "X1", "go", now)
            self.assertEqual(problems, [])
            self.assertTrue(message.startswith(f"Execute the approved decision `X1`. Read `{brief}`; the answer is `go`.\n\n> \"make it real\""))
            stale, _ = needs_you.spawn_message(self.folder(Path(d), "2026-11-13T20:00Z"), "X1", "go", now)
            self.assertIn("First re-check the brief's premise live (`Checked:` is 50 h old)", stale)

    def test_no_message_for_a_brief_that_is_not_its_own_or_says_nothing_for_the_answer(self):
        now = dt.datetime(2026, 11, 15, 22, 0, tzinfo=dt.timezone.utc)
        with tempfile.TemporaryDirectory() as d:
            brief = self.folder(Path(d), "2026-11-15T20:00Z", "## Stance\n> \"x\"\n")
            self.assertEqual(needs_you.spawn_message(brief, "W2", "go", now)[0], None)
            message, problems = needs_you.spawn_message(brief, "X1", "hold", now)
            self.assertEqual(message, None)
            self.assertIn("`hold`", problems[0])

    def test_nested_workstream_decision_uses_nearest_topic_mission_intent(self):
        now = dt.datetime(2026, 11, 22, 22, 0, tzinfo=dt.timezone.utc)
        with tempfile.TemporaryDirectory() as d:
            topic = Path(d) / "topics/problem-triage"
            brief = topic / "work/PT2/GCT-WORKTREE-UI-CONCEPT/BRIEF.md"
            write(brief, "Checked: 2026-11-22 the concept holds, read the design\n\n"
                         "## On each answer\n- `contextual-ledger`: approve concept.\n- `other`: exact words.\n")
            write(topic / "MISSION.md", "# Mission\n\n## Operator intent (verbatim)\n\n"
                                       "> \"keep this resilient\"\n\n## End state\nx\n")
            message, problems = needs_you.spawn_message(brief, "GCT-WORKTREE-UI-CONCEPT", "contextual-ledger", now)
            self.assertEqual(problems, [])
            self.assertIn('> "keep this resilient"', message)



if __name__ == "__main__":
    unittest.main()
