#!/usr/bin/env python3
"""Tests for history.py. Fixtures copy the shapes found in ~/.vrooli/research and the
Web Console Codex session homes on 2026-11-15.

Run: python3 -I -m unittest discover -s <this folder> -p 'test_history.py'
"""
import contextlib
import importlib.util
import io
import json
import os
import sqlite3
import tempfile
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("history", HERE / "history.py")
history = importlib.util.module_from_spec(spec)
spec.loader.exec_module(history)

SCHED = "01a1ce58-90db-79d1-81d6-1497550061ca"
OWNER = "01a1ce5b-93fd-7cd3-ad4a-6ec8e650eb60"
OTHER = "01a1ce5c-0000-7000-8000-000000000001"
HOME_ID = "aa8d0781-91f7-4063-9027-c6b9238b8dd4"
OLD_THREAD = "01a17ba0-51a4-7c11-ae8b-e221895597ce"
OLD_HOME = "1f40c7b3-2898-4072-bac8-93520417f7c7"

MISSION = """# Mission — {title}

Mode: build
Last architect pass: 2026-11-10
Last ambition review: 2026-11-10 ACCEPT

## Workstreams

| ID | Destination | Dimensions | Magnitude | Showcase | Status |
|---|---|---|---|---|---|
{rows}
"""


def write(path, text):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8") as handle:
        handle.write(text)
    return path


def line(ts, kind, payload, ordinal=None):
    record = {"timestamp": ts}
    if ordinal is not None:
        record["ordinal"] = ordinal
    record.update({"type": kind, "payload": payload})
    return json.dumps(record, separators=(",", ":"))


def meta(ts, thread, parent=None, agent=None, history_start=None):
    payload = {"session_id": SCHED, "id": thread, "cli_version": "0.167.0"}
    if parent:
        payload["parent_thread_id"] = parent
        payload["source"] = {"subagent": {"thread_spawn": {"parent_thread_id": parent, "depth": 1, "agent_path": agent}}}
    if history_start:
        payload["subagent_history_start_ordinal"] = history_start
    return line(ts, "session_meta", payload, 0)


def change(ts, path, diff=None, content=None, ordinal=None):
    body = {"type": "update", "unified_diff": diff} if diff is not None else {"type": "add", "content": content}
    return line(ts, "event_msg", {"type": "item_completed", "item": {"type": "FileChange", "id": "exec-1", "changes": {path: body}}}, ordinal)


def started(ts, ordinal=None):
    return line(ts, "event_msg", {"type": "task_started", "turn_attribution": {"turn_trigger": "goal"}}, ordinal)


def rollout(root, home, thread, lines):
    return write(os.path.join(root, home, "sessions", "2026", "11", "12", f"rollout-2026-11-12T01-01-13-{thread}.jsonl"),
                 "\n".join(lines) + "\n")


class Fixture(unittest.TestCase):
    """A small research home and session root in the real shapes."""

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        base = self.tmp.name
        self.home = os.path.join(base, "research")
        self.sessions = os.path.join(base, "codex")
        self.cache = os.path.join(base, "cache.json")
        h = self.home
        # topics: a landed row inside the commission window, one before it, an alias symlink
        write(f"{h}/topics/debt-reduction/MISSION.md", MISSION.format(title="Debt reduction", rows="\n".join([
            "| DR1 | Delete dead code. | maint | 40–80 h · many | Net lines down. | landed 2026-11-12 |",
            "| DR2 | Convert lifecycles. | maint | 60 h · many | One owner each. | active |",
            "| DR3 | First scenario. | maint | 30 h · one | Fewer lines. | open |"])))
        write(f"{h}/topics/debt-reduction/LEDGER.md", "# Debt reduction — ledger\n\n## Architect pass — capture (2026-11-13)\n\nProposed DR4.\n\n"
                                                       "## Cohort follow-up — DR2 slice (2026-11-12)\n\nDR2 moved its lifecycle dimension.\n")
        write(f"{h}/topics/debt-reduction/runs/debt-crypto-models-20261112/CHECKPOINT.md",
              "# DR checkpoint — debt-crypto-models-20261112\n\n**Last updated:** 2026-11-12T09:00Z\n\n## DR2 follow-up (2026-11-12)\n\nwork\n")
        write(f"{h}/topics/crypto-foundations/MISSION.md", MISSION.format(title="Wallets, accounting and useful agent payments", rows="\n".join([
            "| CF1 | Mandates. | money | 40 h · treasury | Journey. | landed 2026-11-09 |",
            "| CF2 | Accounting. | money | 40 h · ledger | Receipts. | active |",
            "| CF3 | Recovery. | money | 40 h · dbm | Restore. | open |"])))
        write(f"{h}/topics/experience-engineering/MISSION.md", MISSION.format(title="Experience engineering", rows="\n".join([
            "| EX1 | Capture. | ux | 120 h · bas | Bundle. | active |",
            "| EX2 | Contracts. | ux | 60 h · rcl | Story. | open |",
            "| EX3 | Flagships. | ux | 60 h · ui | Phone. | open |"])))
        os.makedirs(f"{h}/topics/experience-engineering/background/validation-quality")
        os.symlink(f"{h}/topics/experience-engineering/background/validation-quality", f"{h}/topics/validation-quality")
        # a desk-launched commission (LAUNCH.json, PACKET.json, bullet ledger, operator file)
        c = f"{h}/cohorts/debt-crypto-models-20261112"
        write(f"{c}/README.md", "# Charter\n\nRead the cards.\n")
        write(f"{c}/PACKET.json", json.dumps({"schema": "research-desk.packet/v1", "commission_id": "debt-crypto-models-20261112",
                                              "built_at": "2026-11-12T06:00:00Z", "spec": {"allowance_hours": 12, "topics": ["crypto-foundations", "debt-reduction"]}}))
        write(f"{c}/LAUNCH.json", json.dumps({
            "schema": "research-desk.launch/v1", "commission_id": "debt-crypto-models-20261112", "launched_by": "desk",
            "web_console_session_id": HOME_ID, "thread_id": SCHED,
            "steps": [{"step": "created", "at": "2026-11-12T06:00:00.080921528Z"}, {"step": "goal_sent", "at": "2026-11-12T06:00:30.6Z"}],
            "failed": None, "discarded": {"at": "2026-11-12T16:00:00Z", "reason": "superseded"},
            "controls": [{"action": "supersede", "at": "2026-11-12T16:00:00.613352462Z", "outcome": "window stopped"}],
            "superseded_by": "next-20261113"}))
        write(f"{c}/LEDGER.md", "# Ledger — debt-crypto-models-20261112\n\n"
                                "- 2026-11-12T06:02Z · Start · Read charter. No scorecard dimension moved yet.\n"
                                "- 2026-11-12T07:02Z · Hourly review 1 · The CF2 joined flow passed and moved five named dimensions.\n"
                                "- Footprint: free 178 GiB (not an entry).\n"
                                "### 2026-11-12 09:02 UTC · hourly review\nOwners steady.\n"
                                "### 2026-11-12 10:?? UTC · footprint admission\nno time\n")
        write(f"{c}/OPERATOR.md", "# Operator — debt-crypto-models-20261112\n\n### 2026-11-12T08:00:00Z · direction · via desk\nGo bigger.\n")
        # an older cohort: id time stamp, COHORT.json studies map with thread ids, closed_at
        o = f"{h}/cohorts/research-renewal-20261027T043008Z"
        write(f"{o}/COHORT.json", json.dumps({"id": "research-renewal-20261027T043008Z", "allowance_hours_each": 8,
                                              "studies": {"crypto-foundations": {"thread": OLD_THREAD}},
                                              "closed_at": "2026-10-27T12:27:07.462590+00:00"}))
        write(f"{o}/STATUS.md", "# Status\n\n- 2026-10-27 05:00 UTC: wallets owner dispatched.\n")
        # a prepared packet that never ran, and a never-started planning record
        write(f"{h}/cohorts/standing-teams-w1-20261110/README.md", "# Standing teams\n\nNo clock has started.\n")
        write(f"{h}/cohorts/research-alignment-20261027T041201Z/COHORT.json", json.dumps({"allowance_started": False}))
        # an operator verdict naming the alias
        write(f"{h}/reviews/backup-validation-console-20261114/VERDICTS.md",
              "# Verdicts — backup-validation-console-20261114\n\n### 2026-11-14T05:09:15Z · verdict validation-quality · via hand\nKeep going.\n")
        # sessions: scheduler, an owner with an inherited fork prefix, a reviewer, an old TUI-era thread
        rollout(self.sessions, HOME_ID, SCHED, [
            meta("2026-11-12T06:00:40Z", SCHED), started("2026-11-12T06:00:41Z"),
            change("2026-11-12T06:05:00Z", f"{h}/cohorts/debt-crypto-models-20261112/LEDGER.md", diff="@@\n+a\n+b\n"),
            line("2026-11-12T06:06:00Z", "event_msg", {"type": "token_count"}),
            line("2026-11-12T11:00:00Z", "event_msg", {"type": "token_count"}),
            line("2026-11-12T15:30:00Z", "event_msg", {"type": "token_count"})])
        rollout(self.sessions, HOME_ID, OWNER, [
            meta("2026-11-12T06:10:00Z", OWNER, SCHED, "/root/debt_reduction_owner", history_start=3),
            line("2026-11-12T06:00:41Z", "event_msg", {"type": "token_count"}, 1),
            change("2026-11-12T06:00:42Z", "/repo/inherited.go", diff="+x\n", ordinal=2),
            started("2026-11-12T06:10:01Z", 3),
            change("2026-11-12T06:20:00Z", "/home/u/Vrooli/scenarios/a/main.go", diff="@@\n-old\n+new\n+more\n", ordinal=4),
            change("2026-11-12T06:25:00Z", "/home/u/Vrooli/scenarios/a/new.go", content="package a\n\nfunc A() {}", ordinal=5),
            line("2026-11-12T06:29:00Z", "event_msg", {"type": "task_complete"}, 6)])
        rollout(self.sessions, HOME_ID, OTHER, [
            meta("2026-11-12T07:00:00Z", OTHER, SCHED, "/root/cf2_challenge"),
            change("2026-11-12T07:01:00Z", "/home/u/Vrooli/scenarios/b/x.go", diff="@@\n+1\n"),
            line("2026-11-12T17:00:00Z", "event_msg", {"type": "token_count"})])  # after the supersede: clipped
        rollout(self.sessions, OLD_HOME, OLD_THREAD, [
            meta("2026-10-27T04:35:00Z", OLD_THREAD), started("2026-10-27T04:35:01Z"),
            change("2026-10-27T05:00:00Z", "/home/u/Vrooli/scenarios/treasury/x.go", diff="@@\n+1\n+2\n-3\n"),
            line("2026-10-27T12:00:00Z", "event_msg", {"type": "token_count"})])
        db = sqlite3.connect(os.path.join(self.sessions, HOME_ID, "goals_1.sqlite"))
        db.execute("create table thread_goals (thread_id text, objective text, status text, tokens_used int, time_used_seconds int, created_at_ms int)")
        db.execute("insert into thread_goals values (?, ?, 'paused', 0, 0, ?)",
                   (SCHED, f"Run {h}/cohorts/debt-crypto-models-20261112/README.md until done", 1794463200000))
        db.commit()
        db.close()
        self.now = "2026-11-15T05:00:00Z"

    def tearDown(self):
        self.tmp.cleanup()

    def desk(self, cache=None):
        sessions = history.Sessions(self.sessions, os.path.join(self.tmp.name, "no-tui"), cache)
        return history.Home(self.home, sessions, history.bound(self.now))


class ParsingTests(unittest.TestCase):
    def test_entries_read_every_dated_shape_and_skip_the_rest(self):
        text = ("# Ledger\n\n### 2026-11-14T11:08:01Z · Commission started\nbody one\n"
                "### 2026-11-14 10:08 UTC · unit start\n- Footprint: 2026-11-14 is not leading\n"
                "## Architect pass — capture (2026-11-13)\n## Needs you\nundated section text\n"
                "- 2026-11-12T06:02Z · Start · Read charter.\n| 2026-11-10 | table row |\n")
        got = history.entries(text, "L")
        self.assertEqual([e["precision"] for e in got], ["minute", "minute", "day", "minute", "day"])
        self.assertEqual(history.iso(got[0]["t"]), "2026-11-14T11:08Z")
        self.assertIn("- Footprint: 2026-11-14 is not leading", got[1]["body"])
        self.assertNotIn("undated section text", " ".join(got[2]["body"]))

    def test_offsets_convert_to_utc(self):
        self.assertEqual(history.iso(history.when("2026-11-04T00:52:20-05:00")[0]), "2026-11-04T05:52Z")
        self.assertEqual(history.iso(history.when("2026-11-12T00:42:22.604567+00:00")[0]), "2026-11-12T00:42Z")

    def test_updated_stamp_counts_as_an_event(self):
        got = history.entries("# Checkpoint\n\n**Last updated:** 2026-11-05T03:35Z  \n", "C")
        self.assertEqual([(history.iso(e["t"]), e["title"]) for e in got], [("2026-11-05T03:35Z", "updated stamp")])

    def test_scorecard_claims_are_read_per_sentence(self):
        claim = lambda text: history.scorecard_claim({"body": [text]})
        self.assertEqual(claim("No scorecard dimension moved yet."), "none")
        self.assertEqual(claim("The flow passed and moved five named dimensions; CF3 is not integrated."), "moved")
        self.assertEqual(claim("Product LOC and authoring scorecard outcomes have not moved."), "none")
        self.assertEqual(claim("Evidence-level dimensions moved in BAS quality. No other change."), "moved")
        self.assertIsNone(claim("Owners steady."))

    def test_gaps_count_window_edges_and_stalls_over_two_hours(self):
        h = 3600.0
        stats = history.gap_stats([(1 * h, 2 * h), (5 * h, 6 * h)], [(0.0, 10 * h)])
        self.assertEqual(stats["max_gap_h"], 4.0)  # 6h to the window end
        self.assertEqual(stats["gaps_over_2h"], 2)  # 2h-5h and 6h-10h; the 1 h lead-in is not a stall
        self.assertEqual(stats["stalled_share"], 0.7)
        self.assertIsNone(history.gap_stats([], []))


class CommissionTests(Fixture):
    def test_launched_commission_window_session_and_delivery(self):
        row = self.desk().commission("debt-crypto-models-20261112")
        self.assertEqual((row["start"], row["end"]), ("2026-11-12T06:00Z", "2026-11-12T16:00Z"))
        self.assertEqual(row["sources"]["start"], "LAUNCH.json steps")
        self.assertEqual(row["state"], "superseded")
        self.assertEqual(row["window_h"], 10.0)
        s = row["session"]
        self.assertEqual(s["threads"], 3)
        # inherited fork history is skipped; new files count every content line; research-home edits are not repo
        self.assertEqual((s["repo_lines_added"], s["repo_lines_removed"]), (6, 1))
        self.assertEqual((s["lines_added"], s["lines_removed"]), (8, 1))
        self.assertEqual(s["repo_paths_changed"], 3)
        self.assertEqual(s["idle"]["max_gap_h"], 4.5)  # 11:00 -> 15:30, the last agent line before the stop
        self.assertEqual(s["no_change"]["max_gap_h"], 8.98)  # last edit 07:01 -> stop 16:00
        self.assertIsNotNone(s["first_repo_edit_h"])  # time to first edit, from the window start
        self.assertGreaterEqual(s["first_repo_edit_h"], 0)
        self.assertLessEqual(s["first_repo_edit_h"], 1.02)  # no repo edit after 07:01
        self.assertEqual(row["landed"], ["debt-reduction:DR1"])
        self.assertEqual(row["record_entries"], 4)  # three timed entries and one day-only heading
        self.assertEqual(row["checkpoint_entries"], 2)
        self.assertEqual(row["operator_entries"], 1)
        self.assertEqual(row["scorecard_self_reported"], {"moved": 1, "none": 1})
        self.assertEqual(row["lines_by_topic"], {"unattributed": 2, "debt-reduction": 6, "crypto-foundations": 1})
        self.assertEqual(row["coverage"], "WRSLC")

    def test_older_cohort_uses_id_stamp_closed_at_and_study_thread_map(self):
        row = self.desk().commission("research-renewal-20261027T043008Z")
        self.assertEqual((row["start"], row["end"]), ("2026-10-27T04:30Z", "2026-10-27T12:27Z"))
        self.assertEqual(row["sources"]["start"], "commission id time stamp")
        self.assertEqual(row["topics"], ["crypto-foundations"])
        self.assertEqual(row["session_links"], [f"thread-ref:{OLD_THREAD}"])
        self.assertEqual(row["session"]["repo_lines_added"], 2)
        self.assertEqual(row["landed"], [])  # CF1 landed 2026-11-09, outside this window

    def test_unknown_stays_unknown(self):
        row = self.desk().commission("standing-teams-w1-20261110")
        self.assertEqual(row["state"], "no-window")
        for key in ("window_h", "session", "landed", "record_entries", "scorecard_self_reported"):
            self.assertIsNone(row[key], key)
        self.assertEqual(row["coverage"], "-----")
        self.assertEqual(self.desk().commission("research-alignment-20261027T041201Z")["state"], "never-started")

    def test_failed_launch_has_no_window_and_unreadable_files_are_recorded(self):
        c = f"{self.home}/cohorts/steward-free-problems-20261114"
        write(f"{c}/LAUNCH.json", json.dumps({"web_console_session_id": HOME_ID, "steps": [{"step": "created", "at": "2026-11-12T06:00:00Z"}],
                                              "failed": {"step": "ready_seen", "reason": "timeout"}, "discarded": None, "controls": []}))
        locked = write(f"{c}/LEDGER.md", "- 2026-11-12T06:02Z · Start\n")
        os.chmod(locked, 0)
        history.UNREADABLE.clear()
        try:
            row = self.desk().commission("steward-free-problems-20261114")
        finally:
            os.chmod(locked, 0o644)
        self.assertEqual((row["state"], row["window_h"], row["session"]), ("failed", None, None))
        if os.geteuid() != 0:
            self.assertEqual(history.UNREADABLE, [f"{locked}: PermissionError"])

    def test_records_written_while_no_rollout_moved_mark_the_session_partial(self):
        ledger = f"{self.home}/cohorts/debt-crypto-models-20261112/LEDGER.md"
        with open(ledger, "a", encoding="utf-8") as handle:
            handle.write("- 2026-11-12T08:00Z · owner return\n- 2026-11-12T08:30Z · review\n- 2026-11-12T10:00Z · review\n")
        row = self.desk().commission("debt-crypto-models-20261112")
        self.assertIn("5 dated record entries", row["session"]["partial"])  # 07:02, 08:00, 08:30, 09:02, 10:00 in 07:01-11:00
        self.assertEqual(row["coverage"], "WRsLC")
        self.assertEqual(row["session"]["idle"]["stalls"][0], ["2026-11-12T11:00Z", 4.5])

    def test_since_clips_the_window_and_its_counts(self):
        desk = self.desk()
        row = desk.commission("debt-crypto-models-20261112", history.bound("2026-11-12T11:00Z"), None)
        self.assertEqual(row["window_h"], 5.0)
        self.assertEqual(row["session"]["lines_added"], 0)
        self.assertEqual(row["landed"], ["debt-reduction:DR1"])  # day-precision date overlaps the clipped window


class TopicAndPortfolioTests(Fixture):
    def test_topic_row_attributes_lines_and_resolves_aliases(self):
        desk = self.desk()
        rows = desk.commissions()
        debt = history.topic_row(desk, "debt-reduction", rows)
        self.assertEqual(debt["landed_all"], ["DR1 2026-11-12"])
        self.assertEqual(debt["session"]["lines_added"], 5)
        self.assertEqual(debt["session"]["unattributed_lines_in_shared_commissions"], 2)
        self.assertEqual(debt["scorecard_self_reported"]["moved"], 1)  # topic ledger "moved its lifecycle dimension"
        ux = history.topic_row(desk, "experience-engineering", rows)
        self.assertEqual((ux["verdicts"], ux["verdict_reviews"]), (1, ["backup-validation-console-20261114"]))
        # a later provenance correction voids the verdicts above it: delegate text is not operator judgment
        with open(f"{self.home}/reviews/backup-validation-console-20261114/VERDICTS.md", "a") as f:
            f.write("\n### 2026-11-15T08:40Z · provenance correction · via audit (not an operator verdict)\nA delegate wrote it.\n")
        fresh = self.desk()
        self.assertEqual(history.topic_row(fresh, "experience-engineering", fresh.commissions())["verdicts"], 0)
        self.assertIsNone(ux["session"])
        later = history.topic_row(desk, "debt-reduction", rows, history.bound("2026-11-13"), None)
        self.assertEqual(later["landed_in_range"], [])

    def test_portfolio_totals_count_each_rollout_once_and_flag_shared_sessions(self):
        desk = self.desk()
        write(f"{self.home}/cohorts/debt-crypto-models-20261112-copy/LAUNCH.json", json.dumps({
            "web_console_session_id": HOME_ID, "steps": [{"step": "created", "at": "2026-11-12T06:00:00Z"}],
            "controls": [{"action": "stop", "at": "2026-11-12T16:00:00Z"}]}))
        desk = self.desk()
        data = history.portfolio(desk)
        by_id = {r["id"]: r for r in data["commissions"]}
        self.assertEqual(by_id["debt-crypto-models-20261112"]["shared_session_with"], ["debt-crypto-models-20261112-copy"])
        self.assertEqual(data["totals"]["research"]["repo_lines_added"], 6 + 2)
        self.assertIn("debt-reduction:DR1", data["totals"]["research"]["landed"])
        # windows 10-27 04:30-12:27 and 11-12 06:00-16:00; span runs to --now 11-15 05:00
        self.assertEqual(data["calendar"]["covered_h"], round((7 * 60 + 57) / 60 + 10, 1))
        self.assertEqual(data["calendar"]["no_commission_gaps"][0], ["2026-10-27T12:27Z", 377.5])  # closed_at 12:27:07 -> 11-12 06:00:00

    def test_cache_reuses_unchanged_rollouts(self):
        first = self.desk(self.cache)
        first.commission("debt-crypto-models-20261112")
        first.sessions.save()
        desk = self.desk(self.cache)
        original = history.rollout_events
        history.rollout_events = lambda path: (_ for _ in ()).throw(AssertionError("rescanned " + path))
        try:
            row = desk.commission("debt-crypto-models-20261112")
        finally:
            history.rollout_events = original
        self.assertEqual(row["session"]["repo_lines_added"], 6)

    def test_cli_markdown_and_json(self):
        args = ["--home", self.home, "--sessions", self.sessions, "--tui-sessions", "/nonexistent", "--cache", "", "--now", self.now]
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            history.main(args + ["portfolio"])
        self.assertIn("| debt-crypto-models-20261112 | research | superseded | 2026-11-12T06:00 | 10.0 |", out.getvalue())
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            history.main(args + ["topic", "validation-quality", "--json"])
        self.assertEqual(json.loads(out.getvalue())["topic"], "experience-engineering")


if __name__ == "__main__":
    unittest.main()
