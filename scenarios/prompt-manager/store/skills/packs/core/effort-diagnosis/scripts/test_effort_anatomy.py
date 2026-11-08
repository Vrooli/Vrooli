#!/usr/bin/env python3
"""Regression tests for effort_anatomy.py using synthetic transcripts.

Run: python3 -m unittest discover -s <skillDir>/scripts -p 'test_*.py'
"""
import contextlib
import io
import json
import os
import sqlite3
import sys
import tempfile
import unittest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import effort_anatomy as ea  # noqa: E402


def line(ts, kind, payload, ordinal=None):
    record = {"timestamp": ts}
    if ordinal is not None:
        record["ordinal"] = ordinal
    record.update({"type": kind, "payload": payload})
    return json.dumps(record, separators=(",", ":"))


def user_message(text):
    return {"type": "message", "role": "user", "content": [{"type": "input_text", "text": text}]}


def file_change(path, diff):
    return {"type": "item_completed", "item": {"type": "FileChange", "changes": {path: {"type": "update", "unified_diff": diff}}}}


def spawn(task, fork=None):
    args = {"task_name": task, "message": "gAAAA-encrypted"}
    if fork:
        args["fork_turns"] = fork
    return {"type": "function_call", "name": "spawn_agent", "namespace": "collaboration", "arguments": json.dumps(args)}


GOAL_TEXT = '<codex_internal_context source="goal">\nContinue.\n<objective>\nWork until ten rounds are done.\n</objective>'


def write_rollout(directory, name, lines):
    path = os.path.join(directory, name)
    with open(path, "w", encoding="utf-8") as handle:
        handle.write("\n".join(lines) + "\n")
    return path


class RolloutFixtures(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        day = os.path.join(self.tmp.name, "sessions", "2026", "11", "05")
        os.makedirs(day)
        self.root = write_rollout(day, "rollout-2026-11-05T00-00-00-root.jsonl", [
            line("2026-11-05T05:00:00Z", "session_meta", {"id": "root-1", "cli_version": "0.166.0", "source": "cli"}),
            line("2026-11-05T05:00:01Z", "turn_context", {"model": "gpt-6-luna", "effort": "high"}),
            line("2026-11-05T05:00:02Z", "event_msg", {"type": "task_started", "turn_trigger": "goal"}),
            line("2026-11-05T05:00:02Z", "response_item", user_message(GOAL_TEXT)),
            line("2026-11-05T05:00:03Z", "response_item", user_message("Please be bold.")),
            line("2026-11-05T05:00:04Z", "event_msg", {"type": "token_count", "info": {"last_token_usage": {"input_tokens": 21000}, "model_context_window": 258400}}),
            line("2026-11-05T05:10:00Z", "response_item", spawn("hunter_a")),
            line("2026-11-05T05:11:00Z", "response_item", spawn("hunter_b", "none")),
            line("2026-11-05T05:20:00Z", "event_msg", file_change("/repo/a.go", "@@\n-old\n+new\n+more\n")),
            line("2026-11-05T05:30:00Z", "event_msg", {"type": "item_completed", "item": {"type": "AgentMessage", "content": [{"type": "Text", "text": "Round 3 is a no code finding; I'll stop if the signal is too weak."}]}}),
            line("2026-11-05T06:00:00Z", "event_msg", {"type": "item_completed", "item": {"type": "ContextCompaction"}}),
            line("2026-11-05T06:00:00Z", "compacted", {"replacement_history": [{"text": "User set the goal: [objective omitted; exceeds the evidence limit]"}]}),
            line("2026-11-05T07:00:00Z", "event_msg", {"type": "task_complete"}),
        ])
        # A forked child: ordinals below the history start are inherited parent items.
        self.child = write_rollout(day, "rollout-2026-11-05T00-10-00-child.jsonl", [
            line("2026-11-05T05:10:01Z", "session_meta", {"id": "child-1", "parent_thread_id": "root-1", "subagent_history_start_ordinal": 3,
                                                          "source": {"subagent": {"thread_spawn": {"parent_thread_id": "root-1", "depth": 1, "agent_path": "/root/hunter_a"}}}}, 0),
            line("2026-11-05T05:00:02Z", "response_item", user_message(GOAL_TEXT), 1),
            line("2026-11-05T05:00:03Z", "event_msg", file_change("/repo/inherited.go", "+inherited\n"), 2),
            line("2026-11-05T05:10:02Z", "event_msg", {"type": "token_count", "info": {"last_token_usage": {"input_tokens": 190000}}}, 3),
            line("2026-11-05T05:12:00Z", "event_msg", file_change(os.path.join(ea.RESEARCH_PATH, "x/CHECKPOINT.md"), "+a\n+b\n"), 4),
            line("2026-11-05T05:14:00Z", "event_msg", {"type": "task_complete"}, 5),
        ])

    def tearDown(self):
        self.tmp.cleanup()

    def test_root_metrics(self):
        r = ea.scan_rollout(self.root)
        self.assertEqual(r["id"], "root-1")
        self.assertEqual(r["depth"], 0)
        self.assertEqual(r["model"], "gpt-6-luna")
        self.assertEqual(r["goal_injections"], 1)
        self.assertEqual(r["operator_messages"], 1, "goal injection must not count as an operator message")
        self.assertEqual(r["compactions"], 1)
        self.assertEqual(r["objective_omitted"], 1)
        self.assertEqual((r["lines_added"], r["lines_removed"]), (2, 1))
        self.assertEqual((r["spawns"], r["spawn_fork_all"], r["spawn_fork_none"]), (2, 1, 1))
        self.assertEqual(r["context_window"], 258400)
        self.assertAlmostEqual(r["hours"], 2.0, places=2)
        # Gaps of ten minutes or less count (05:00:00→05:30 = 1,800 s); the 30- and 60-minute gaps are idle.
        self.assertAlmostEqual(r["active_hours"] * 3600, 1800, delta=1)

    def test_child_excludes_inherited_history(self):
        c = ea.scan_rollout(self.child)
        self.assertEqual(c["depth"], 1)
        self.assertEqual(c["parent"], "root-1")
        self.assertEqual(c["path"], "/root/hunter_a")
        self.assertEqual(c["goal_injections"], 0, "inherited goal text is the parent's, not the child's")
        self.assertEqual(c["lines_added"], 2, "inherited file changes must not count")
        self.assertEqual(c["research_lines"], 2)
        self.assertEqual(c["first_input_tokens"], 190000)
        self.assertAlmostEqual(c["hours"] * 60, 4.0, places=1)

    def test_scan_summary_and_tree(self):
        scan_path = os.path.join(self.tmp.name, "scan.jsonl")
        root_glob = os.path.join(self.tmp.name, "sessions")
        self.assertEqual(ea.main(["scan", "--root", root_glob, "--out", scan_path]), 0)
        rows = ea.load_scan(scan_path)
        self.assertEqual(len(rows), 2)
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            self.assertEqual(ea.main(["summary", scan_path]), 0)
            self.assertEqual(ea.main(["tree", scan_path, "root-1", "--list"]), 0)
        text = out.getvalue()
        self.assertIn("inherited>100k=100%", text)
        self.assertIn("/root/hunter_a", text)
        self.assertIn("descendants=1", text)

    def test_scan_date_filter(self):
        root_glob = os.path.join(self.tmp.name, "sessions")
        self.assertEqual(ea.discover_rollouts([root_glob], "2026-11-06", None), [])
        self.assertEqual(len(ea.discover_rollouts([root_glob], "2026-11-05", "2026-11-05")), 2)

    def test_timeline(self):
        events, stats = ea.timeline_events(self.root, 2)
        kinds = [k for _t, k, _d in events]
        self.assertEqual(kinds.count("goal"), 1)
        self.assertIn(("2026-11-05T06:00:00", "compaction", "objective omitted"), events)
        self.assertIn("spawn_agent", kinds)
        self.assertTrue(any("fork=none" in d for _t, k, d in events if k == "spawn_agent"))
        self.assertTrue(any("fork=all(default)" in d for _t, k, d in events if k == "spawn_agent"))
        self.assertEqual(stats["hourly"]["2026-11-05T05"], [2, 1])
        self.assertGreaterEqual(stats["families"].get("no_change", 0), 1)
        self.assertGreaterEqual(stats["families"].get("threshold", 0), 1)


class LegacyPatchRollout(unittest.TestCase):
    def test_apply_patch_in_shell_call_counts_when_no_file_change_items(self):
        patch = "apply_patch <<'EOF'\n*** Begin Patch\n*** Update File: a/b.go\n@@\n-old\n+new\n+newer\n*** End Patch\nEOF"
        call = {"type": "function_call", "name": "shell", "arguments": json.dumps({"command": ["bash", "-lc", patch]})}
        with tempfile.TemporaryDirectory() as tmp:
            path = write_rollout(tmp, "rollout-2025-09-29T00-00-00-x.jsonl", [
                line("2025-09-29T00:00:00Z", "session_meta", {"id": "legacy"}),
                line("2025-09-29T00:01:00Z", "response_item", call),
            ])
            r = ea.scan_rollout(path)
            self.assertEqual((r["lines_added"], r["lines_removed"], r["files_touched"]), (2, 1, 1))
            self.assertEqual(r["lines_source"], "apply_patch")

    def test_file_change_items_win_over_patch_text(self):
        call = {"type": "custom_tool_call", "name": "apply_patch", "input": "*** Begin Patch\n*** Update File: x\n+a\n*** End Patch"}
        with tempfile.TemporaryDirectory() as tmp:
            path = write_rollout(tmp, "rollout-2026-11-05T00-00-00-y.jsonl", [
                line("2026-11-05T00:00:00Z", "session_meta", {"id": "new"}),
                line("2026-11-05T00:01:00Z", "response_item", call),
                line("2026-11-05T00:01:01Z", "event_msg", file_change("/repo/x", "+a\n")),
            ])
            r = ea.scan_rollout(path)
            self.assertEqual(r["lines_added"], 1, "the same edit must not be counted twice")
            self.assertEqual(r["lines_source"], "file_change")


class GoalsAndClaude(unittest.TestCase):
    def test_goals_corpus(self):
        with tempfile.TemporaryDirectory() as tmp:
            db = os.path.join(tmp, "goals_1.sqlite")
            conn = sqlite3.connect(db)
            conn.execute("create table thread_goals (thread_id text, goal_id text, objective text, status text, token_budget int, "
                         "tokens_used int, time_used_seconds int, created_at_ms int, updated_at_ms int)")
            conn.execute("insert into thread_goals values ('t1','g1','Work until X.','complete',null,10,3600,1793800000000,1793800000000)")
            conn.execute("insert into thread_goals values ('t2','g2',?, 'paused',null,10,7200,1793900000000,1793900000000)", ("y" * 2000,))
            conn.commit()
            conn.close()
            out = io.StringIO()
            with contextlib.redirect_stdout(out):
                self.assertEqual(ea.main(["goals", "--db", db, "--show", "1"]), 0)
            text = out.getvalue()
            self.assertIn("goals: 2", text)
            self.assertIn(">=1500", text)
            self.assertIn("paused", text)

    def test_claude_session(self):
        with tempfile.TemporaryDirectory() as tmp:
            session = os.path.join(tmp, "abc.jsonl")
            rows = [
                {"type": "user", "timestamp": "2026-11-05T10:00:00Z", "message": {"role": "user", "content": "Go big."}},
                {"type": "assistant", "timestamp": "2026-11-05T10:01:00Z", "message": {"content": [
                    {"type": "tool_use", "name": "Edit", "input": {"old_string": "a\nb", "new_string": "c\nd\ne"}},
                    {"type": "tool_use", "name": "Agent", "input": {"subagent_type": "researcher", "description": "probe"}},
                    {"type": "tool_use", "name": "MultiEdit", "input": {"edits": [{"old_string": "x", "new_string": "y\nz"}]}}]}},
                {"type": "system", "subtype": "compact_boundary", "timestamp": "2026-11-05T11:00:00Z", "compactMetadata": {"preTokens": 200000}},
            ]
            with open(session, "w", encoding="utf-8") as handle:
                handle.write("\n".join(json.dumps(r) for r in rows))
            sub = os.path.join(tmp, "abc", "subagents")
            os.makedirs(sub)
            with open(os.path.join(sub, "agent-x.jsonl"), "w", encoding="utf-8") as handle:
                handle.write(json.dumps({"type": "assistant", "timestamp": "2026-11-05T10:02:00Z", "message": {"content": [
                    {"type": "tool_use", "name": "Write", "input": {"content": "1\n2\n3\n"}}]}}))
            with open(os.path.join(sub, "agent-x.meta.json"), "w", encoding="utf-8") as handle:
                json.dump({"agentType": "implementer", "spawnDepth": 1, "description": "build"}, handle)
            metrics = ea.claude_metrics(session)["metrics"]
            self.assertEqual((metrics["lines_added"], metrics["lines_removed"]), (5, 3))
            self.assertEqual(metrics["compactions"], 1)
            self.assertEqual(metrics["agent_calls"], 1)
            self.assertEqual(metrics["operator_messages"], 1)
            out = io.StringIO()
            with contextlib.redirect_stdout(out):
                self.assertEqual(ea.main(["claude", session, "--events"]), 0)
            self.assertIn("implementer", out.getvalue())
            self.assertIn("+3", out.getvalue())

    def test_tree_handles_missing_start(self):
        with tempfile.TemporaryDirectory() as tmp:
            scan = os.path.join(tmp, "scan.jsonl")
            rows = [{"id": "r", "file": "r.jsonl", "depth": 0, "start": None, "hours": 0},
                    {"id": "c", "parent": "r", "file": "c.jsonl", "depth": 1, "start": None, "hours": 0, "path": "/root/check_callers"}]
            with open(scan, "w", encoding="utf-8") as handle:
                handle.write("\n".join(json.dumps(r) for r in rows))
            out = io.StringIO()
            with contextlib.redirect_stdout(out):
                self.assertEqual(ea.main(["tree", scan, "r", "--list"]), 0)
            self.assertIn("/root/check_callers", out.getvalue())
            self.assertFalse(ea.is_review(rows[1]), "an implementer named check_* is not a reviewer")


if __name__ == "__main__":
    unittest.main()


def minutes(start_hour, count, step=5):
    """Timestamps every `step` minutes, so the gaps count as active time."""
    base = 0
    out = []
    for i in range(count):
        total = base + i * step
        out.append(f"2026-11-05T{start_hour + total // 60:02d}:{total % 60:02d}:00Z")
    return out


class DriftTripwires(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()

    def tearDown(self):
        self.tmp.cleanup()

    def run_drift(self, path, *extra):
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            status = ea.main(["drift", path, *extra])
        return status, out.getvalue()

    def test_decayed_effort_fires_objective_loss_long_turn_and_inheritance(self):
        lines = [line("2026-11-05T01:00:00Z", "session_meta", {"id": "hunt", "source": "cli"}),
                 line("2026-11-05T01:00:00Z", "event_msg", {"type": "task_started", "turn_trigger": "goal"}),
                 line("2026-11-05T01:00:00Z", "response_item", user_message(GOAL_TEXT))]
        stamps = minutes(1, 60)  # five active hours in one turn
        for i, ts in enumerate(stamps):
            if i in (10, 30):
                lines.append(line(ts, "event_msg", {"type": "item_completed", "item": {"type": "ContextCompaction"}}))
                lines.append(line(ts, "compacted", {"replacement_history": [{"text": "[objective omitted; exceeds the evidence limit]"}]}))
            elif i % 10 == 5:
                lines.append(line(ts, "response_item", spawn(f"hunter_{i}")))
            else:
                lines.append(line(ts, "event_msg", file_change("/repo/x.go", "+a\n")))
        lines.append(line("2026-11-05T06:00:00Z", "event_msg", {"type": "task_complete"}))
        path = write_rollout(self.tmp.name, "decayed.jsonl", lines)
        status, out = self.run_drift(path)
        self.assertEqual(status, ea.DRIFT_EXIT)
        for tripwire in ("objective-lost", "unit-per-turn", "inherited-context"):
            self.assertIn(tripwire, out)

    def test_healthy_unit_per_turn_effort_fires_nothing(self):
        lines = [line("2026-11-05T01:00:00Z", "session_meta", {"id": "ok", "source": "cli"})]
        for turn in range(8):
            stamps = minutes(1 + turn, 8)
            lines.append(line(stamps[0], "event_msg", {"type": "task_started", "turn_trigger": "goal"}))
            lines.append(line(stamps[0], "response_item", user_message(GOAL_TEXT)))
            for ts in stamps[1:-1]:
                lines.append(line(ts, "event_msg", file_change("/repo/y.go", "+a\n" * 40)))
            lines.append(line(stamps[-1], "response_item", spawn(f"owner_{turn}", "none")))
            lines.append(line(stamps[-1], "event_msg", {"type": "task_complete"}))
        path = write_rollout(self.tmp.name, "healthy.jsonl", lines)
        status, out = self.run_drift(path, "--turns")
        self.assertEqual(status, 0, out)
        self.assertIn("no tripwire fired", out)

    def test_shrinking_units_and_no_change_streak(self):
        lines = [line("2026-11-05T01:00:00Z", "session_meta", {"id": "shrink", "source": "cli"})]
        sizes = [200, 220, 180, 210, 30, 0, 0, 0, 0]
        for turn, size in enumerate(sizes):
            stamps = minutes(1 + turn, 3, step=4)
            lines.append(line(stamps[0], "event_msg", {"type": "task_started", "turn_trigger": "goal"}))
            if size:
                lines.append(line(stamps[1], "event_msg", file_change("/repo/z.go", "+a\n" * size)))
            lines.append(line(stamps[-1], "event_msg", {"type": "task_complete"}))
        path = write_rollout(self.tmp.name, "shrink.jsonl", lines)
        status, out = self.run_drift(path)
        self.assertEqual(status, ea.DRIFT_EXIT)
        self.assertIn("shrinking-units", out)
        self.assertIn("no-change-streak", out)

    def test_child_lines_are_credited_with_a_scan(self):
        lines = [line("2026-11-05T01:00:00Z", "session_meta", {"id": "sched", "source": "cli"})]
        for turn in range(7):
            stamps = minutes(1 + turn, 3, step=4)
            lines.append(line(stamps[0], "event_msg", {"type": "task_started", "turn_trigger": "goal"}))
            lines.append(line(stamps[1], "response_item", spawn(f"seg_{turn}", "none")))
            lines.append(line(stamps[-1], "event_msg", {"type": "task_complete"}))
        root = write_rollout(self.tmp.name, "sched.jsonl", lines)
        scan = os.path.join(self.tmp.name, "scan.jsonl")
        rows = [ea.scan_rollout(root)]
        for turn in range(7):
            rows.append({"id": f"seg-{turn}", "parent": "sched", "file": f"/x/{turn}", "start": f"2026-11-05T{1 + turn:02d}:04:30Z",
                         "lines_added": 500, "lines_removed": 0, "active_hours": 3.0, "path": f"/root/seg_{turn}"})
        with open(scan, "w", encoding="utf-8") as handle:
            handle.write("\n".join(json.dumps(r) for r in rows) + "\n")
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            status = ea.main(["drift", root, "--scan", scan, "--json"])
        report = json.loads(out.getvalue())
        self.assertEqual(status, 0, report["findings"])
        self.assertEqual(sum(t["child_lines"] for t in report["turns"]), 3500)


def compacted(*user_texts, truncated=()):
    """A compacted record whose retained_context keeps these user messages."""
    kept = [{"text": t, "complete": i not in truncated, "order": i} for i, t in enumerate(user_texts)]
    return {"replacement_history": [], "retained_context": {"user_messages": kept, "assistant_messages": []}}


GOAL_EVENT = '<codex_internal_context source="user_goal">\nUser set the goal: "ship it"\n</codex_internal_context>'


class StanceRetention(unittest.TestCase):
    """A compaction that keeps no operator message leaves the stance only in the charter (M1)."""

    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()

    def tearDown(self):
        self.tmp.cleanup()

    def drift(self, name, meta, record, marker=None):
        stamps = minutes(1, 4)
        path = write_rollout(self.tmp.name, name, [
            line(stamps[0], "session_meta", meta),
            line(stamps[0], "event_msg", {"type": "task_started", "turn_trigger": "goal"}),
            line(stamps[1], "event_msg", file_change("/repo/s.go", "+a\n" * 50)),
            line(stamps[2], "compacted", record),
            line(stamps[3], "event_msg", {"type": "task_complete"}),
        ])
        return [f["id"] for f in ea.drift_findings(ea.turn_units(path, marker), [], 3.0)], path

    def test_goal_events_and_wakes_alone_fire(self):
        ids, path = self.drift("lost.jsonl", {"id": "root", "source": "cli"},
                               compacted(GOAL_EVENT, "[wake] child run ended"))
        self.assertEqual(ids, ["stance-not-retained"])
        events, _ = ea.timeline_events(path, 1)
        self.assertIn("kept 2 user messages (0 operator, 0 truncated)", [d for _t, k, d in events if k == "compaction"][0])

    def test_a_retained_stance_message_fires_nothing(self):
        ids, _ = self.drift("kept.jsonl", {"id": "root", "source": "cli"},
                            compacted("Stance: be bold.", GOAL_EVENT, truncated=(0,)))
        self.assertEqual(ids, [])

    def test_a_marker_catches_a_stance_evicted_behind_later_steering(self):
        record = compacted("continue", GOAL_EVENT)
        self.assertEqual(self.drift("steer.jsonl", {"id": "root", "source": "cli"}, record)[0], [])
        self.assertEqual(self.drift("steer-m.jsonl", {"id": "root", "source": "cli"}, record, "Be bold")[0], ["stance-not-retained"])
        kept = compacted("Stance: Be bold and ambitious.", "continue")
        self.assertEqual(self.drift("kept-m.jsonl", {"id": "root", "source": "cli"}, kept, "Be bold")[0], [])

    def test_children_and_older_records_are_exempt(self):
        child = {"id": "kid", "parent_thread_id": "root", "source": {"subagent": {"thread_spawn": {"depth": 1}}}}
        self.assertEqual(self.drift("child.jsonl", child, compacted())[0], [])
        self.assertEqual(self.drift("old.jsonl", {"id": "root", "source": "cli"}, {"replacement_history": []})[0], [])


class ThreadContinuations(unittest.TestCase):
    def test_rows_and_files_of_one_thread_merge(self):
        with tempfile.TemporaryDirectory() as tmp:
            day1 = os.path.join(tmp, "sessions", "2026", "11", "04")
            day2 = os.path.join(tmp, "sessions", "2026", "11", "05")
            os.makedirs(day1)
            os.makedirs(day2)
            meta = line("2026-11-05T01:00:00Z", "session_meta", {"id": "T1", "source": "cli"})
            first = write_rollout(day1, "rollout-2026-11-04T20-00-00-T1.jsonl", [
                meta, line("2026-11-05T01:00:00Z", "event_msg", {"type": "task_started"}),
                line("2026-11-05T01:05:00Z", "event_msg", {"type": "task_complete"})])
            write_rollout(day2, "rollout-2026-11-05T01-10-00-T1_X.jsonl", [
                meta, line("2026-11-05T01:10:00Z", "event_msg", {"type": "task_started"}),
                line("2026-11-05T01:15:00Z", "event_msg", file_change("/r/a.go", "+a\n+b\n")),
                line("2026-11-05T01:20:00Z", "event_msg", {"type": "task_complete"})])
            write_rollout(day2, "rollout-2026-11-05T02-00-00-OTHER.jsonl", [
                line("2026-11-05T02:00:00Z", "session_meta", {"id": "OTHER", "source": "cli"})])
            self.assertEqual(len(ea.thread_files(first)), 2)
            scan = os.path.join(tmp, "scan.jsonl")
            with contextlib.redirect_stdout(io.StringIO()), contextlib.redirect_stderr(io.StringIO()):
                ea.main(["scan", "--root", os.path.join(tmp, "sessions"), "--out", scan])
            rows = ea.load_scan(scan)
            merged = [r for r in rows if r["id"] == "T1"]
            self.assertEqual(len(merged), 1)
            self.assertEqual(ea.changed(merged[0]), 2)
            self.assertEqual(merged[0]["turns"], 2)
            self.assertEqual(len(merged[0]["files"]), 2)
            out = io.StringIO()
            with contextlib.redirect_stdout(out):
                ea.main(["drift", first])
            self.assertIn("files=2 turns=2", out.getvalue())


class DriftLegacyPatches(unittest.TestCase):
    def test_apply_patch_only_rollout_is_not_a_no_change_streak(self):
        patch = "*** Begin Patch\n*** Update File: a.go\n" + "+x\n" * 100 + "*** End Patch"
        lines = [line("2025-09-29T00:00:00Z", "session_meta", {"id": "legacy"})]
        for turn in range(6):
            stamps = minutes(1 + turn, 3, step=4)
            call = {"type": "function_call", "name": "shell", "arguments": json.dumps({"command": ["bash", "-lc", "apply_patch <<'EOF'\n" + patch + "\nEOF"]})}
            lines.append(line(stamps[0], "event_msg", {"type": "task_started"}))
            lines.append(line(stamps[1], "response_item", call))
            lines.append(line(stamps[-1], "event_msg", {"type": "task_complete"}))
        with tempfile.TemporaryDirectory() as tmp:
            path = write_rollout(tmp, "legacy.jsonl", lines)
            units = ea.turn_units(path)
            self.assertTrue(all(u["lines"] == 100 for u in units), [u["lines"] for u in units])
            self.assertEqual(ea.drift_findings(units, [], 3.0), [])
