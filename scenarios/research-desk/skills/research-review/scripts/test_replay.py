#!/usr/bin/env python3
"""Tests for replay.py: parsing, brief rendering, metrics, grading inputs and the sandbox. No model is called.

Run: python3 -I test_replay.py   (from this folder)
"""

from __future__ import annotations

import io
import json
import os
import re
import shutil
import socket
import ssl
import subprocess
import sys
import tempfile
import time
import unittest
from unittest import mock
from pathlib import Path

sys.dont_write_bytecode = True
sys.path.insert(0, str(Path(__file__).resolve().parent))
import replay as rp  # noqa: E402

CHARTER = """# Research commission demo-20261110 — Alpha · Beta

## Operator stance (verbatim; carry all of it into every brief)

> "Be ambitious and exercise creative thinking." — 2026-11-05

> "things that need my attention
> do actually make it to me"

## Current operator steering (2026-11-10)

> "make sure the research agents stay focused"

## Steering

Read entries of OPERATOR.md beside this file.

## Shared machine

Leave 120 GB free.

## Cards

### Card 1 — Alpha · mode `build`

- **Folder:** `/r/topics/alpha/` — `MISSION.md`; list files `CAPABILITIES.md`, `WANTS.md`; run folder `runs/demo/`.
- **Read first:** `/r/topics/alpha/NOTE.md`
- **Workstreams** (active, approved, then open; not a priority order: choose by scorecard movement):
  1. **AL1** (60–120 h · image-tools, ai-gateway): every capability served
     end to end.
  2. **AL2** (20–40 h · image-tools): profile every model.

Operator's notes for this card, verbatim:

Rank capabilities first.

### Card 2 — Beta · mode `propose-then-build`

- **Mission:** [beta/MISSION.md](../../topics/beta/MISSION.md).
- **Order:**
  1. BE3, the rehearsal scenario.

## Scheduler procedure

1. Start.
"""

MISSION = """# Alpha — mission

Mode: build

## Operator intent (verbatim)

> "Make it real."

## Approval flow

Mode `build`. Owners change anything.

Second paragraph.

## Real boundaries

- No paid APIs.

## Not this

- A report.
"""

TEMPLATE = """# Owner brief template

```text
You own [card topic] for this commission: land [the card's workstreams, in its order if it gives one]. [The card's Read first line and operator's notes, verbatim.] Stance: [the stance quotes, then any direction for this topic]. Charter [abs MISSION.md path]. Skill [the adaptive-research path in the charter]. Mode [mode]: [the mission's approval flow line]. [the mission's sections, unchanged] [The charter's Shared machine section.] Checkpoint in [run folder]/CHECKPOINT.md. Run `python3 -I [F] diff --baseline [B] --keep-file [K]`. [If the card names list files: re-read them at each checkpoint.] [an unknown bracket]
```
"""


def ev(t, turn, e):
    return {"t": t, "turn": turn, "e": e}


def cmd_item(i, command, output, code=0):
    return {"type": "item.completed", "item": {"id": f"item_{i}", "type": "command_execution", "command": command,
                                               "aggregated_output": output, "exit_code": code, "status": "completed"}}


class Parsing(unittest.TestCase):
    def test_topics_cards_and_stance(self):
        topics = ["alpha", "beta"]
        self.assertEqual(rp.charter_topics(CHARTER, {"spec": {"topics": topics}}), topics)
        cs = rp.cards(CHARTER, topics)
        self.assertEqual([(c["n"], c["topic"], c["mode"]) for c in cs],
                         [(1, "alpha", "build"), (2, "beta", "propose-then-build")])
        quotes = rp.stance_quotes(CHARTER)
        self.assertEqual(len(quotes), 2)
        self.assertEqual(quotes[1], '"things that need my attention\ndo actually make it to me"')
        self.assertEqual(rp.strip_attribution(quotes[0]), '"Be ambitious and exercise creative thinking."')

    def test_charter_topics_from_paths_when_no_packet(self):
        text = "see topics/alpha/MISSION.md and topics/beta/MISSION.md and topics/alpha/MISSION.md"
        self.assertEqual(rp.charter_topics(text, None), ["alpha", "beta"])

    def test_single_topic_cards_take_the_only_topic(self):
        text = "## Cards\n\n### Card 1 — Guards\n\n- **Order:**\n  1. SW23: snapshot.\n"
        self.assertEqual(rp.cards(text, ["teams"])[0]["topic"], "teams")

    def test_numbered_list_joins_continuations(self):
        body = rp.cards(CHARTER, ["alpha", "beta"])[0]["body"]
        items = rp.numbered_list(body)
        self.assertEqual(items[0], "**AL1** (60–120 h · image-tools, ai-gateway): every capability served end to end.")
        self.assertEqual(len(items), 2)

    def test_reconstruct_spec_without_packet(self):
        texts = {"README.md": CHARTER, "KERNEL.txt": "Work until done; 24h allowance.", "STANCE-MESSAGE.txt": "x"}
        spec = rp.reconstruct_spec("demo-20261110", texts, None, ["alpha", "beta"])
        self.assertEqual(spec["allowance_hours"], 24)
        self.assertEqual(spec["order"], {"alpha": ["AL1", "AL2"], "beta": ["BE3"]})
        self.assertEqual(spec["stance_quotes"][0], '"Be ambitious and exercise creative thinking."')
        self.assertRegex(spec["slug"], r"^[a-z0-9][a-z0-9-]*$")


class Rendering(unittest.TestCase):
    def test_fill_brief(self):
        card = rp.cards(CHARTER, ["alpha", "beta"])[0]
        brief, unfilled = rp.fill_brief(TEMPLATE, charter=CHARTER, card=card, topic="alpha", home=Path("/h"),
                                        cid="demo", skills=Path("/s"), mission=MISSION)
        self.assertIn("You own Alpha for this commission: land **AL1**", brief)
        self.assertIn("**Read first:** `/r/topics/alpha/NOTE.md`", brief)
        self.assertIn("Rank capabilities first.", brief)
        self.assertIn('"Be ambitious', brief)
        self.assertIn("make sure the research agents stay focused", brief)  # operator steering carried
        self.assertNotIn("Read entries of OPERATOR.md", brief)               # the procedural Steering section is not
        self.assertIn("Charter /h/topics/alpha/MISSION.md.", brief)
        self.assertIn("Skill /s/adaptive-research/SKILL.md.", brief)
        self.assertIn("Mode build: Mode `build`. Owners change anything.", brief)
        self.assertNotIn("Second paragraph", brief)
        self.assertIn("Real boundaries:\n- No paid APIs.", brief)
        self.assertIn("Leave 120 GB free.", brief)
        self.assertIn("/h/topics/alpha/runs/demo/CHECKPOINT.md", brief)
        self.assertIn("python3 -I /s/research-coordination/scripts/footprint.py diff --baseline /h/cohorts/demo/BASELINE.json "
                      "--keep-file /h/cohorts/demo/FOOTPRINT-KEEP.txt", brief)
        self.assertIn("List files: CAPABILITIES.md, WANTS.md; re-read them", brief)
        self.assertEqual(unfilled, ["an unknown bracket"])

    def test_fill_brief_condensed_template(self):
        template = ("```text\nYou own [topic] for this commission. Land [the card's workstreams, in its order]. "
                    "[The card's Read first line and the operator's notes for this card, verbatim. If the topic has "
                    "work/<ID>/BRIEF.md for a workstream, name its path: start from it.]\n\nYour charter is "
                    "[abs MISSION.md path]. Read the adaptive-research skill first ([its path]).\n```\n")
        with tempfile.TemporaryDirectory() as d:
            home = Path(d)
            (home / "topics/alpha/work/AL1").mkdir(parents=True)
            (home / "topics/alpha/work/AL1/BRIEF.md").write_text("x")
            card = rp.cards(CHARTER, ["alpha", "beta"])[0]
            brief, unfilled = rp.fill_brief(template, charter=CHARTER, card=card, topic="alpha", home=home,
                                            cid="demo", skills=Path("/s"), mission=MISSION)
        self.assertEqual(unfilled, [])
        self.assertIn("You own Alpha for this commission. Land **AL1**", brief)
        self.assertIn("Read the adaptive-research skill first (/s/adaptive-research/SKILL.md).", brief)
        self.assertIn(f"Start from {home}/topics/alpha/work/AL1/BRIEF.md", brief)
        self.assertIn(f"Your charter is {home}/topics/alpha/MISSION.md.", brief)

    def test_list_file_sentence_dropped_without_list_files(self):
        card = rp.cards(CHARTER, ["alpha", "beta"])[1]
        brief, _ = rp.fill_brief(TEMPLATE, charter=CHARTER, card=card, topic="beta", home=Path("/h"), cid="d",
                                 skills=Path("/s"), mission=MISSION)
        self.assertNotIn("re-read them", brief)
        self.assertIn("land BE3, the rehearsal scenario.", brief)
    def test_snapshot_copy_keeps_text_and_leaves_out_bulk(self):
        with tempfile.TemporaryDirectory() as d:
            src, dest = Path(d) / "alpha", Path(d) / "out"
            for rel, text in (("MISSION.md", "m"), ("corpus/a.json", "x" * 900), ("runs/old/CHECKPOINT.md", "c"),
                              ("runs/old/capture.json", "{}"), ("runs/demo-20261110/CHECKPOINT.md", "own"),
                              ("tools/x.bin", "b"), ("runs/old/a/b/c/deep.md", "deep")):
                (src / rel).parent.mkdir(parents=True, exist_ok=True)
                (src / rel).write_text(text)
            info = rp.copy_text_tree(src, dest, cap_file=1000, cap_dir=500, skip_prefixes=("demo-20261110",))
            kept = sorted(str(p.relative_to(dest)) for p in dest.rglob("*") if p.is_file())
            self.assertEqual(kept, ["MISSION.md", "runs/old/CHECKPOINT.md"])
            self.assertEqual(info["left_out_dirs"], {"corpus": 900})

    def test_variant_skills_never_carry_the_harness(self):
        with tempfile.TemporaryDirectory() as d:
            dest = Path(d) / "skills"
            rp.copy_skills(rp.SKILLS, dest)
            self.assertTrue((dest / "research-coordination/SKILL.md").is_file())
            for f in rp.REPLAY_FILES:
                self.assertFalse((dest / "research-review/scripts" / f).exists(), f)


SUPERVISOR_EVENTS = [
    ev(0.5, 1, {"type": "thread.started", "thread_id": "th-1"}),
    ev(0.6, 1, {"type": "item.completed", "item": {"type": "agent_message", "text": "ready"}}),
    ev(1.0, 1, {"type": "turn.completed", "usage": {"input_tokens": 1000, "cached_input_tokens": 400, "output_tokens": 5}}),
    ev(2.0, 2, cmd_item(1, "/bin/bash -lc 'cat /etc/hostname'", "host\n")),
    ev(3.0, 2, cmd_item(2, "/bin/bash -lc 'prompt-manager skill read adaptive-research research-coordination --lines 1-160 | head -50'",
                        "x" * 100)),
    ev(4.0, 2, cmd_item(3, "/bin/bash -lc 'search-hub query \"x\" --limit 5'",
                        "Error: query: unavailable: dial tcp 127.0.0.1:19157: connection refused\n", 1)),
    ev(5.0, 2, cmd_item(4, "/bin/bash -lc \"mkdir -p /r/topics/a/runs/c && python3 -I f.py snapshot --out /r/B.json\"",
                        "mkdir: cannot create directory: Read-only file system\n", 1)),
    ev(6.0, 2, cmd_item(5, "/bin/bash -lc 'cat /etc/os-release'", "y" * 50)),
    ev(7.0, 2, {"type": "item.completed", "item": {"type": "agent_message", "text": 'Plan:\n```json\n{"role": "supervisor", "owners": ['
                '{"topic": "alpha", "workstream": "AL1", "changes": ["image-tools", "ai-gateway API"], "brief": "go {big}"},'
                '{"topic": "beta", "workstream": "BE3", "changes": ["client-rehearsal"]}], "operator_questions": ["q?"]}\n```'}}),
    ev(7.5, 2, {"type": "turn.completed", "usage": {"input_tokens": 5000, "cached_input_tokens": 3000, "output_tokens": 700,
                                                     "reasoning_output_tokens": 300}}),
]


class OwnerBriefs(unittest.TestCase):
    def test_a_split_card_gets_one_brief_per_owner(self):
        charter = ("## Cards\n\n### Card 1 — alpha\n\n1. **A1** (10 h): first\n2. **A2** (10 h): second\n3. **A3** (5 h): third\n\n"
                   "## Scheduler\n\nStart 2 owners (card 1: 2, one per listed workstream in card order) from the owner brief.\n")
        self.assertEqual(rp.owner_split(charter), {1: 2})
        self.assertEqual(rp.owner_split("Start 3 owners (one per card) from the owner brief."), {})
        with tempfile.TemporaryDirectory() as d:
            ws = fake_workspace(Path(d))
            pdir = Path(d) / "p"
            owners, unfilled = rp.owner_briefs(ws, pdir, "```text\nYou own [topic]. Land [the card's workstreams].\n```",
                                               charter, ["alpha"], "c1", rp.RESEARCH)
            self.assertEqual([o["file"] for o in owners], ["owner-alpha-card1.txt", "owner-alpha-card1-owner2.txt"])
            self.assertEqual([o["workstreams"] for o in owners], [["**A1** (10 h): first"], ["**A2** (10 h): second"]])
            self.assertEqual(unfilled, {})
            self.assertEqual(rp.owner_brief_file(pdir, "alpha", None).name, "owner-alpha-card1.txt")
            self.assertNotIn("A2", (pdir / "owner-alpha-card1.txt").read_text())


class Metrics(unittest.TestCase):
    def test_supervisor_metrics(self):
        m = rp.metrics(SUPERVISOR_EVENTS, Path("/"), scenarios={"image-tools", "ai-gateway", "notification-hub"})
        self.assertEqual(m["first_action"]["kind"], "write")
        self.assertEqual(m["commands_before_first_action"], 3)
        self.assertEqual(m["bytes_before_first_action"], len("host\n") + 100 +
                         len("Error: query: unavailable: dial tcp 127.0.0.1:19157: connection refused\n"))
        self.assertEqual(m["paths_before_first_action"],
                         ["/etc/hostname", "skill:adaptive-research", "skill:research-coordination", "cli:search-hub query"])
        self.assertEqual(m["files_before_first_action"], ["/etc/hostname"])
        self.assertEqual(m["tokens"], {"input": 6000, "cached_input": 3400, "output": 705, "reasoning": 300})
        self.assertEqual([r["route"] for r in m["route_failures"]], ["search-hub"])
        self.assertTrue(m["plan_ok"])
        self.assertEqual(m["units"], 2)
        self.assertEqual(m["scenarios_named"], ["ai-gateway", "image-tools"])
        self.assertEqual(m["operator_questions"], 1)
        self.assertFalse(m["spawned"])

    def test_agent_that_stops_reads_everything(self):
        events = [ev(1, 1, cmd_item(1, "/bin/bash -lc 'cat /etc/hostname'", "abc")),
                  ev(2, 1, {"type": "item.completed", "item": {"type": "agent_message", "text": '{"role": "owner", "workstream": "AL1"}'}})]
        m = rp.metrics(events, Path("/"), scenarios=set())
        self.assertIsNone(m["first_action"])
        self.assertEqual(m["bytes_before_first_action"], 3)
        self.assertEqual(m["units"], 1)

    def test_a_transport_fallback_error_item_is_not_an_action(self):
        events = [ev(1, 1, {"type": "item.completed", "item": {"type": "error", "message": "Falling back from WebSockets"}}),
                  ev(2, 1, cmd_item(1, "/bin/bash -lc 'cat /etc/hostname'", "abc"))]
        m = rp.metrics(events, Path("/"), scenarios=set())
        self.assertIsNone(m["first_action"])
        self.assertEqual(m["bytes_before_first_action"], 3)

    def test_spawn_and_unknown_items_are_actions(self):
        events = [ev(1, 1, cmd_item(1, "/bin/bash -lc 'cat /etc/hostname'", "abc")),
                  ev(2, 1, {"type": "item.completed", "item": {"type": "collab_tool_call", "tool": "spawn_agent"}}),
                  ev(3, 1, cmd_item(2, "/bin/bash -lc 'cat /etc/hostname'", "abc"))]
        m = rp.metrics(events, Path("/"), scenarios=set())
        self.assertEqual(m["first_action"]["kind"], "spawn")
        self.assertTrue(m["spawned"])
        self.assertEqual(m["bytes_before_first_action"], 3)

    def test_degradation_notes_are_not_route_failures(self):
        events = [ev(1, 1, cmd_item(1, "/bin/bash -lc 'search-hub query x --limit 5'",
                                    "Summary:\n  Searched 9 corpora (degraded: provider unavailable)\n", 0))]
        self.assertEqual(rp.metrics(events, Path("/"), scenarios=set())["route_failures"], [])

    def test_write_detection(self):
        for cmd in ("mkdir -p x", "echo a > f.txt", "cat a | tee b", "sed -i s/a/b/ f", "research-desk say c 'x'",
                    "python3 -I footprint.py snapshot --out B.json", "cd x && touch y"):
            self.assertTrue(rp.is_write(cmd), cmd)
        for cmd in ("cat f.txt", "rg -n 'a>b' f", "ls 2>/dev/null", "sed -n 1,20p f", "prompt-manager skill read x",
                    "research-desk commission c", "git status", "echo hi 2>&1 | head"):
            self.assertFalse(rp.is_write(cmd), cmd)
        self.assertTrue(rp.is_write("cat x", "touch: cannot touch 'y': Read-only file system"))

    def test_extract_json_takes_last_object(self):
        text = 'first {"a": 1} then ```json\n{"b": {"c": "}"}}\n``` end'
        self.assertEqual(rp.extract_json(text), {"b": {"c": "}"}})
        self.assertIsNone(rp.extract_json("no json here {broken"))
    def test_a_failed_plan_is_not_zero_units(self):
        events = [ev(1, 1, cmd_item(1, "/bin/bash -lc 'cat /etc/hostname'", "abc")),
                  ev(2, 1, {"type": "item.completed", "item": {"type": "agent_message", "text": "I would start owners."}})]
        m = rp.metrics(events, Path("/"), scenarios=set())
        self.assertFalse(m["plan_ok"])
        self.assertIsNone(m["units"])
        self.assertIsNone(rp.MEASURES["units"](m))

    def test_python_writes_are_writes(self):
        self.assertTrue(rp.is_write("python3 - <<'PY'\nfrom pathlib import Path\nPath('x').write_text('y')\nPY"))
        self.assertTrue(rp.is_write("python3 -c \"open('f', 'w').write('1')\""))
        self.assertTrue(rp.is_write("python3 - <<P\nimport os; os.makedirs('x')\nP"))
        self.assertFalse(rp.is_write("python3 -c \"print(open('a').read())\""))
        self.assertFalse(rp.is_write("python3 -c \"import json; json.load(open('a.json'))\""))


class Rollout(unittest.TestCase):
    def test_bytes_seen_stop_at_the_first_action(self):
        def call(i, cmd):
            return {"payload": {"type": "custom_tool_call", "name": "exec", "call_id": f"c{i}",
                                "input": f"const r = await tools.exec_command({{cmd:{json.dumps(cmd)}}});\ntext(r.output);"}}
        def out(i, text):
            return {"payload": {"type": "custom_tool_call_output", "call_id": f"c{i}",
                                "output": [{"type": "input_text", "text": text}]}}
        recs = [call(1, "cat MISSION.md"), out(1, "x" * 40), call(2, "rg -n 'a=>b' big"), out(2, "y" * 10),
                {"payload": {"type": "function_call", "name": "spawn_agent", "call_id": "c3", "arguments": "{}"}},
                call(4, "cat more"), out(4, "z" * 99)]
        calls = rp.rollout_calls(recs)
        self.assertEqual([c["cmds"] for c in calls[:2]], [["cat MISSION.md"], ["rg -n 'a=>b' big"]])
        seen = rp.seen_before_action(calls, None)
        self.assertEqual(seen["bytes_seen_before_first_action"], 50)
        self.assertEqual(seen["first_action_call"], "spawn_agent")
        recs[4] = call(3, "mkdir -p /r/runs/x")
        seen = rp.seen_before_action(rp.rollout_calls(recs), None)
        self.assertEqual((seen["bytes_seen_before_first_action"], seen["calls_before_first_action"]), (50, 2))
        self.assertEqual(rp.seen_before_action(rp.rollout_calls(recs[:4]), None)["first_action_call"], None)

    def test_commands_in_js_arrays_and_patches_inside_exec(self):
        js = 'const cmds = ["cat MISSION.md", "rg -n foo docs"];\nfor (const c of cmds) await tools.exec_command({cmd: c});'
        recs = [{"payload": {"type": "custom_tool_call", "name": "exec", "call_id": "a", "input": js}},
                {"payload": {"type": "custom_tool_call_output", "call_id": "a", "output": "x" * 7}},
                {"payload": {"type": "custom_tool_call", "name": "exec", "call_id": "b",
                             "input": 'await tools.apply_patch("*** Begin Patch\\n...")'}}]
        calls = rp.rollout_calls(recs)
        self.assertEqual(calls[0]["cmds"], ["cat MISSION.md", "rg -n foo docs"])
        seen = rp.seen_before_action(calls, None)
        self.assertEqual((seen["bytes_seen_before_first_action"], seen["calls_before_first_action"]), (7, 1))

    def test_relative_reads_resolve_against_the_commands_working_folder(self):
        with tempfile.TemporaryDirectory() as tmp:
            cohort, repo = Path(tmp, "cohort"), Path(tmp, "repo")
            for d in (cohort, repo):
                d.mkdir()
                (d / "README.md").write_text("x")
            (cohort / "STATE.md").write_text("y")
            recs = [{"payload": {"type": "custom_tool_call", "name": "exec", "call_id": "a", "input":
                                 f'const r = await tools.exec_command({{cmd:"cat README.md && cat \\"STATE.md\\"",'
                                 f'workdir:{json.dumps(str(cohort))}}});'}},
                    {"payload": {"type": "function_call", "name": "exec_command", "call_id": "b",
                                 "arguments": json.dumps({"cmd": "cat README.md", "workdir": "../repo"})}}]
            workdirs = rp.command_workdirs(recs)
            self.assertEqual(workdirs, {'cat README.md && cat "STATE.md"': str(cohort), "cat README.md": "../repo"})
            run = lambda cmd: {"type": "item.completed", "item": {"type": "command_execution", "command": cmd,
                                                                  "aggregated_output": "x", "exit_code": 0}}
            events = [run("/bin/bash -lc 'cat README.md && cat \"STATE.md\"'"), run("/bin/bash -lc 'cat README.md'")]
            m = rp.metrics(events, cohort, scenarios=set(), workdirs=workdirs)   # the second's workdir is relative to cwd
            self.assertEqual(m["files_before_first_action"], [str(cohort / "README.md"), str(cohort / "STATE.md"),
                                                              str(repo / "README.md")])
            self.assertEqual(rp.metrics(events[:1], repo, scenarios=set())["files_before_first_action"],
                             [str(repo / "README.md")])                         # no workdir known: the run's cwd

    def test_a_read_counts_where_the_sandbox_put_the_file_not_where_the_host_has_one(self):
        with tempfile.TemporaryDirectory() as tmp:
            mounted = Path(tmp, "variant/cohorts/c-new")                       # the run's commission, on the host
            mounted.mkdir(parents=True)
            (mounted / "README.md").write_text("x")
            inside = Path(tmp, "research/cohorts/c-new")                       # where the run saw it
            view = [(str(inside), mounted), (str(Path(tmp, "research/cohorts")), Path("/nonexistent"))]
            self.assertEqual(rp.command_paths(f"cat {inside}/README.md README.md", inside, view),
                             [str(inside / "README.md")])                      # one file, named as the run saw it
            Path(tmp, "research/cohorts/c-old").mkdir(parents=True)
            Path(tmp, "research/cohorts/c-old/README.md").write_text("live")   # on the host, but not mounted
            self.assertEqual(rp.command_paths(f"cat {tmp}/research/cohorts/c-old/README.md", inside, view), [])


class Grading(unittest.TestCase):
    def test_grade_prompt_is_blind(self):
        units = [{"topic": "alpha", "brief": f"read {rp.RESEARCH}/topics/alpha/MISSION.md and {rp.SKILLS}/adaptive-research "
                                             "for demo-20261110 (demo-20261110-20261115)"}]
        clean = rp.sanitize(json.dumps(units), rp.RESEARCH)
        for leak in (str(rp.RESEARCH), str(rp.SKILLS), "demo-20261110"):
            self.assertNotIn(leak, clean)
        prompt = rp.grade_prompt("MISSION", rp.RUBRIC.read_text(), json.loads(clean), [])
        self.assertIn("<plan-a>", prompt)
        self.assertNotIn("old", prompt.split("<plan-a>")[1].lower().replace("threshold", ""))

    def test_topic_units_filters_supervisor_plans(self):
        with tempfile.TemporaryDirectory() as d:
            run = Path(d)
            rp.write_json(run / "meta.json", {"role": "supervisor"})
            rp.write_json(run / "plan.json", {"plan": {"owners": [{"topic": "alpha"}, {"topic": "beta"}]}})
            self.assertEqual(rp.topic_units(run, "beta"), [{"topic": "beta"}])
            rp.write_json(run / "plan.json", {"plan": None})
            self.assertEqual(rp.topic_units(run, "beta"), [rp.NO_PLAN])  # the grader sees the failure, not an empty plan

    def test_swapped_orders_must_agree(self):
        with tempfile.TemporaryDirectory() as d:
            ws = rp.WS(Path(d))
            for name, picks in (("g1", ("r1", "r1")), ("g2", ("r1", "r2"))):
                orders = [{"A": "r1", "B": "r2", "better_run": picks[0], "scores": {"r1": 6, "r2": 4},
                           "verdict": {"A": {"units": [{"scope_level": "whole-api"}]}, "B": {"units": []}}},
                          {"A": "r2", "B": "r1", "better_run": picks[1], "scores": {"r2": 5, "r1": 4},
                           "verdict": {"A": {"units": []}, "B": {"units": [{"scope_level": "module"}]}}}]
                rp.write_json(ws.grades() / name / "mapping.json", {"topic": "alpha", "orders": orders})
                agree = picks[0] == picks[1]
                rp.write_json(ws.grades() / name / "grade.json", {"orders": orders, "agree": agree,
                                                                  "better_run": picks[0] if agree else None})
            g1, g2 = rp.grade_records(ws)
            self.assertEqual((g1["better_run"], g1["agree"]), ("r1", True))
            self.assertEqual((g2["better_run"], g2["agree"]), (None, False))
            self.assertEqual(g1["score"], {"r1": 5.0, "r2": 4.5})
            self.assertEqual(g1["levels"]["r1"], ["whole-api", "module"])

    def test_rubric_names_every_criterion(self):
        text = rp.RUBRIC.read_text()
        for word in ("whole-scenario", "whole-api", "several-scenarios", "lists_affected_scenarios", "proportionate",
                     "keeps_operator_intent", "needless_operator_questions", '"better"'):
            self.assertIn(word, text)



class Harness(unittest.TestCase):
    """DRIVER and stream with fake agents in place of Codex (no model, no sandbox)."""

    @staticmethod
    def fake(lines: list[dict], sleep: int, *, rollout: str | None = None) -> list[str]:
        """An agent that prints events (ARG becomes its last argument, the resumed thread) and sleeps."""
        code = ("import json, os, sys, time\n"
                f"for e in json.loads({json.dumps(json.dumps(lines))}):\n"
                "    print(json.dumps(e).replace('ARG', sys.argv[-1]), flush=True)\n"
                + (f"d = os.path.join(os.environ['CODEX_HOME'], 'sessions', '2026')\nos.makedirs(d, exist_ok=True)\n"
                   f"open(os.path.join(d, {rollout!r}), 'w').write('{{\"a\": 1}}\\n{{\"b\": 2}}\\n')\n" if rollout else "")
                + f"time.sleep({sleep})\n")
        return [sys.executable, "-I", "-c", code, "-"]

    def drive(self, turns: list[list[str]], rollouts: Path | None = None):
        with tempfile.TemporaryDirectory() as home:
            (Path(home) / "driver.py").write_text(rp.DRIVER)
            t0, buf = time.time(), io.StringIO()
            res = rp.stream([sys.executable, "-I", str(Path(home) / "driver.py")],
                            {"turns": turns, "files": {"auth.json": "{}"}, "dirs": ["sessions"]},
                            {"CODEX_HOME": home, "PATH": os.environ["PATH"]}, t0 + 60, buf, t0, rollouts=rollouts)
            mode = (Path(home) / "auth.json").stat().st_mode & 0o777
        return res, time.time() - t0, [json.loads(x) for x in buf.getvalue().splitlines()], mode

    def test_the_stance_turn_ends_on_the_reply_and_the_plan_turn_resumes_its_thread(self):
        stance = self.fake([{"type": "thread.started", "thread_id": "th-1"},
                            {"type": "item.completed", "item": {"type": "agent_message", "text": "Ready."}}], sleep=50)
        plan = self.fake([{"type": "item.completed", "item": {"type": "agent_message", "text": '{"role": "owner", "x": "ARG"}'}}],
                         sleep=50)[:-1] + ["{thread}"]
        res, took, events, _ = self.drive([stance, plan])
        self.assertEqual([(t["turn"], t["killed"], t["thread"]) for t in res["turns"]],
                         [(1, "answered", "th-1"), (2, "answered", None)])
        self.assertIn('"x": "th-1"', events[-1]["e"]["item"]["text"])       # the plan turn resumed thread th-1
        self.assertEqual(events[-1]["turn"], 2)
        self.assertLess(took, 40)

    def test_the_plan_turn_waits_for_the_json(self):
        res, took, _, _ = self.drive([self.fake([
            {"type": "item.completed", "item": {"type": "agent_message", "text": "Reading the charter."}},
            {"type": "item.completed", "item": {"type": "agent_message", "text": '{"role": "owner"}'}}], sleep=50)])
        self.assertEqual(res["turns"][0]["killed"], "answered")
        self.assertLess(took, 25)

    def test_a_turn_that_ends_itself_is_not_killed_and_a_resume_without_a_thread_never_runs(self):
        done = self.fake([{"type": "item.completed", "item": {"type": "agent_message", "text": '{"role": "owner"}'}},
                          {"type": "turn.completed", "usage": {}}], sleep=0)
        res, _, _, mode = self.drive([done, done[:-1] + ["{thread}"]])
        self.assertEqual(res["turns"], [{"turn": 1, "exit": 0, "killed": None, "thread": None}])
        self.assertEqual(mode, 0o600)                                           # the login copy is private

    def test_a_deeply_nested_line_is_kept_raw(self):
        code = "print('[' * 200000, flush=True)\nprint('{\"type\": \"turn.completed\", \"usage\": {}}')"
        res, _, events, _ = self.drive([[sys.executable, "-I", "-c", code]])
        self.assertEqual(res["turns"][0]["exit"], 0)
        self.assertEqual(len(events[0]["e"]["raw"]), 200000)

    def test_the_sandbox_login_is_unusable_and_server_side_tools_are_off(self):
        tokens = {"access_token": "h.eyJleHAiOjF9.realsig", "id_token": "h.p.idsig", "refresh_token": "r", "account_id": "a"}
        with tempfile.TemporaryDirectory() as d, mock.patch.object(rp, "HOME", Path(d)):
            rp.write_json(Path(d) / ".codex/auth.json", {"tokens": tokens})
            header = rp.codex_header([["codex"]])
            self.assertEqual(rp.host_access_token(), tokens["access_token"])
        got = json.loads(header["files"]["auth.json"])["tokens"]
        self.assertEqual(got, {"access_token": "h.eyJleHAiOjF9." + rp.DUMMY_SIGNATURE, "id_token": "h.p." + rp.DUMMY_SIGNATURE,
                               "refresh_token": "", "account_id": "a"})
        argv = rp.codex_turn("p", model="m", effort="e", cwd=Path("/"), resume=None, dev=None)
        for flag in ('web_search="disabled"', "features.image_generation=false"):
            self.assertEqual(argv[argv.index(flag) - 1], "-c")

    def test_every_replay_turn_runs_over_https_only(self):
        for resume in (None, "{thread}"):
            argv = rp.codex_turn("p", model="m", effort="e", cwd=Path("/"), resume=resume, dev=None)
            provider = next(a for a in argv if a.startswith("model_providers.openai_https="))
            self.assertEqual(argv[argv.index(provider) - 1], "-c")
            self.assertIn("supports_websockets=false", provider)
            self.assertIn('base_url="https://chatgpt.com/backend-api/codex"', provider)
            self.assertEqual(argv[argv.index('model_provider="openai_https"') - 1], "-c")

    def test_a_websocket_attempt_is_seen_in_the_gate_log_stderr_or_events(self):
        https = {"denied": ["GET /backend-api/ps/plugins/list (route)", "POST /backend-api/codex/responses (model turns off)"]}
        self.assertEqual(rp.websocket_attempts(https, "ok", [{"type": "turn.completed"}]), [])
        self.assertEqual(rp.websocket_attempts({"denied": ["GET /backend-api/codex/responses (route)"]}),
                         ["GET /backend-api/codex/responses (route)"])
        self.assertEqual(rp.websocket_attempts(https, "ERROR: failed to connect to Responses WebSocket\n" * 5),
                         ["stderr: failed to connect to Responses WebSocket"])
        notice = {"type": "item.completed", "item": {"type": "error", "message": "Falling back from WebSockets to HTTPS"}}
        self.assertEqual(rp.websocket_attempts(https, "", [notice]), ["events: Falling back from WebSockets"])

    def test_rollouts_return_on_stdout_under_safe_names(self):
        forged = {"replay": "rollout", "name": "../../../../evil", "line": "x"}
        with tempfile.TemporaryDirectory() as d:
            res, _, events, _ = self.drive([self.fake([forged, {"type": "turn.completed", "usage": {}}], sleep=0,
                                                      rollout="rollout-2026-11-15-abc.jsonl")], Path(d))
            names = sorted(p.name for p in Path(d).iterdir())
            self.assertEqual(names, ["rollout-26-11-15-abc.jsonl.gz", "rollout-evil.jsonl.gz"])
            import gzip
            self.assertEqual(gzip.open(Path(d) / names[0], "rt").read(), '{"a": 1}\n{"b": 2}\n')
        self.assertEqual(res["rollouts"], 2)
        self.assertFalse(any("replay" in e["e"] for e in events))              # driver lines never reach events.jsonl


SOURCES = {"AGENTS.md": "agents", "docs/x.md": "doc", "scenarios/a/api/main.go": "package main",
           "scenarios/b/tmpl.go": 'fmt.Sprintf("postgres://%s:%s@%s/db", u, p, h)'}
PLANTED = {  # the files the hostile reviews read, and other credential-like files; none may reach a sandbox
    "scenarios/agent-manager/data/identity-secret.key": "k", "scenarios/agent-manager/data/agent-manager.db": "db",
    "scenarios/agent-manager/data/agent-manager.db-wal": "wal", "scenarios/system-monitor/.env": "TOKEN=1",
    "scenarios/app-monitor/ui/.env": "X=1", "data/resources/home-assistant/config/secrets.yaml": "pw: 1",
    "scenarios/b/config.json": '{"refresh_token": "' + "r" * 30 + '"}', "scenarios/b/key_test.go": "-----BEGIN PRIVATE KEY-----",
    "scenarios/b/node_modules/m/index.js": "x", "scenarios/b/server.pem": "x", "scenarios/b/secrets.json": "{}",
    "scenarios/b/db.go": 'url := "postgres://admin:hunter2@db:5432/x"',
    "scenarios/v/platforms/electron/bundle/catalog/d/.vrooli/service.json": '{"db": "postgresql://user:password@localhost:5433/d"}',
    "scenarios/v/platforms/electron/dist-electron/r/bundle/x.json": "{}",
}


def fake_workspace(root: Path, variants=("old",)) -> rp.WS:
    """A workspace with a snapshot and variants, as snapshot/freeze would leave them (no model, no CLI)."""
    ws = rp.WS(root)
    snap = ws.snapshot()
    rp.write_json(snap / "snapshot.json", {"research_home": str(rp.RESEARCH)})
    rp.write(snap / "home/topics/alpha/MISSION.md", MISSION)
    (snap / "home/cohorts").mkdir(parents=True)
    for rel, text in {**SOURCES, **PLANTED}.items():
        rp.write(root / "src" / rel, text)
    rp.copy_repo(root / "src", snap / "repo", ())
    (snap / "repo" / rp.SKILLS.relative_to(rp.REPO)).mkdir(parents=True)
    rp.write(snap / "skills/other-skill.md", "---\nname: o\n---\nOther skill body\n")
    rp.write_cli_stubs(snap / "bin", ["research-desk", "agent-manager", "prompt-manager"], rp.SKILLS, rp.SKILL_EXPORT)
    for v in variants:
        rp.write(ws.variant(v) / "skills/research-coordination/SKILL.md", f"---\nname: rc\n---\nResearch coordination {v}\n")
        rp.write(ws.variant(v) / "cohorts/c1/README.md", f"charter {v}")
    return ws


class Snapshot(unittest.TestCase):
    def test_the_repository_copy_is_an_allowlist_without_credentials(self):
        with tempfile.TemporaryDirectory() as d:
            for rel, text in {**SOURCES, **PLANTED}.items():
                rp.write(Path(d) / "src" / rel, text)
            got = rp.copy_repo(Path(d) / "src", Path(d) / "out", (Path(d) / "src/scenarios/a",))
            kept = sorted(str(p.relative_to(Path(d) / "out")) for p in (Path(d) / "out").rglob("*") if p.is_file())
        self.assertEqual(kept, ["AGENTS.md", "docs/x.md", "scenarios/b/tmpl.go"])  # scenarios/a left out by the caller
        self.assertEqual(got, {"files": 3, "credential_like_left_out": 4})    # 3 by content, secrets.json by name


def run_record(ws: rp.WS, name: str, variant: str, role: str, size: int, plan: bool = True, topic=None, reads=True):
    run = ws.runs() / name
    run.mkdir(parents=True)
    rp.write_json(run / "meta.json", {"run_id": name, "commission_id": "c1", "variant": variant, "role": role,
                                      "topic": topic, "harness": "codex", "model": "m", "effort": "e", "wall_s": 10, "cwd": "/"})
    tail = SUPERVISOR_EVENTS[-2:] if plan else [ev(7, 2, {"type": "item.completed", "item": {"type": "agent_message", "text": "no"}})]
    events = [ev(2, 2, cmd_item(1, "/bin/bash -lc 'cat /etc/hostname'", "z" * size))] * reads + tail
    (run / "events.jsonl").write_text("\n".join(json.dumps(e) for e in events) + "\n")


class Report(unittest.TestCase):
    def test_arms_pool_variants_and_the_ratio_has_an_interval(self):
        with tempfile.TemporaryDirectory() as d:
            ws = fake_workspace(Path(d), ("old", "new-a", "new-b"))
            rp.write_json(ws.prompts("c1", "old") / "freeze.json", {"commission_id": "c1", "topics": ["alpha", "beta"]})
            for i, size in enumerate((900, 1100, 1000), 1):
                run_record(ws, f"c1.old.supervisor.{i}", "old", "supervisor", size)
            for i, (v, size) in enumerate((("new-a", 450), ("new-b", 550), ("new-a", 500)), 1):
                run_record(ws, f"c1.{v}.supervisor.{i}", v, "supervisor", size, plan=i != 3)
            rp.cmd_preregister(ws, ["base=old", "new=new-a,new-b"], "reading halves",
                               ["median_below:supervisor:bytes_raw:new:600", "ratio_below:supervisor:bytes_raw:1.10"])
            with self.assertRaises(rp.Fail):
                rp.cmd_preregister(ws, ["base=old", "new=new-a"], "again", [])
            g = ws.grades() / "alpha.c1.old.supervisor.1.vs.c1.new-a.supervisor.1"
            orders = [{"A": "c1.old.supervisor.1", "B": "c1.new-a.supervisor.1", "better_run": "c1.new-a.supervisor.1",
                       "scores": {"c1.old.supervisor.1": 2, "c1.new-a.supervisor.1": 6}, "verdict": {}},
                      {"A": "c1.new-a.supervisor.1", "B": "c1.old.supervisor.1", "better_run": "c1.new-a.supervisor.1",
                       "scores": {"c1.old.supervisor.1": 3, "c1.new-a.supervisor.1": 5}, "verdict": {}}]
            rp.write_json(g / "mapping.json", {"topic": "alpha", "orders": orders})
            rp.write_json(g / "grade.json", {"orders": orders, "agree": True, "better_run": "c1.new-a.supervisor.1"})
            text = rp.cmd_report(ws, None).read_text()
            self.assertIn("`base` = `old`; `new` = `new-a`, `new-b`", text)       # arms from the pre-registration
            row = next(l for l in text.splitlines() if l.startswith("| supervisor | `new` |"))
            self.assertIn("| 3 / 3 | 0 / 1 |", row)                                # runs per arm; one failed plan
            self.assertRegex(row, r"×0\.5\d \[0\.\d\d, 0\.\d\d\]")                 # raw bytes halved, interval below 1
            self.assertIn("PASS (median 500 over 3 runs)", text)
            self.assertIn("no worse than ×1.10: PASS", text)
            self.assertIn("| supervisor | `new` | 1 | 1 | 0 | 0 |", text)        # grader: one agreeing preference
            self.assertIn("the orders agree on the better plan in 1 of 1 grades", text)
            self.assertIn("| plan failed |", text)
            self.assertNotIn("## Corrections", text)
            self.assertIn("## Did not engage (post-hoc exploratory analysis, NOT pre-registered)", text)
            self.assertIn("| supervisor · `new` | 3 | 0 (0%) |", text)
            self.assertLess(text.index("## Pre-registration"), text.index("## Did not engage"))
            rp.cmd_report(ws, None, "parser fix A")                             # a correction stays in every later report
            text = rp.cmd_report(ws, None).read_text()
            self.assertIn("## Corrections after the runs", text)
            self.assertRegex(text, r"- \S+Z: parser fix A")
            self.assertLess(text.index("## Corrections"), text.index("## Pre-registration"))

    def test_runs_that_neither_reasoned_nor_called_a_tool_are_counted_apart(self):
        self.assertFalse(rp.engaged({"tokens": {"reasoning": 0}, "tool_calls": 0}))
        self.assertTrue(rp.engaged({"tokens": {"reasoning": 0}, "tool_calls": 2}))     # read without thinking aloud
        self.assertTrue(rp.engaged({"tokens": {"reasoning": 40}, "tool_calls": 0}))    # thought, then answered
        with tempfile.TemporaryDirectory() as d:
            ws = fake_workspace(Path(d), ("old", "new-a"))
            rp.write_json(ws.prompts("c1", "old") / "freeze.json", {"commission_id": "c1", "topics": ["alpha"]})
            for i, reads in enumerate((True, False, False), 1):
                run_record(ws, f"c1.old.owner.{i}", "old", "owner", 700, topic="alpha", reads=reads)
                run_record(ws, f"c1.new-a.owner.{i}", "new-a", "owner", 300, topic="alpha")
            for run in ws.runs().iterdir():                                    # no reasoning anywhere
                ev_file = run / "events.jsonl"
                ev_file.write_text(ev_file.read_text().replace('"reasoning_output_tokens": 300', '"reasoning_output_tokens": 0'))
            text = rp.cmd_report(ws, ["base=old", "new=new-a"]).read_text()
            self.assertIn("| owner · `base` | 3 | 2 (67%) |", text)                # two answered from the prompt alone
            self.assertIn("| owner · `new` | 3 | 0 (0%) |", text)


class Statistics(unittest.TestCase):
    def test_sign_test(self):
        self.assertAlmostEqual(rp.sign_test(9, 1), 0.0215, places=4)
        self.assertEqual(rp.sign_test(5, 5), 1.0)
        self.assertIsNone(rp.sign_test(0, 0))

    def test_log_ratio_interval_covers_a_known_shift(self):
        rng = __import__("random").Random(1)
        prompts = [([rng.uniform(800, 1200) for _ in range(6)], [rng.uniform(400, 600) for _ in range(6)]) for _ in range(5)]
        ci = rp.log_ratio_ci(prompts)
        self.assertLess(ci["lo"], 0.5)
        self.assertGreater(ci["hi"], 0.45)
        self.assertLess(ci["hi"], 0.6)
        self.assertEqual((ci["prompts"], ci["runs_a"]), (5, 30))
        self.assertIsNone(rp.log_ratio_ci([([1000], [500])]))                   # one run per arm is not an interval

    def test_guidance_key_names_files_by_role(self):
        ws = rp.WS(Path("/w"))
        self.assertEqual(rp.guidance_key("scenarios/research-desk/skills/adaptive-research/SKILL.md", ws),
                         "skills/adaptive-research/SKILL.md")
        self.assertEqual(rp.guidance_key("skill:adaptive-research", ws), "skills/adaptive-research/SKILL.md")
        self.assertEqual(rp.guidance_key(f"{rp.RESEARCH}/cohorts/c-1/README.md", ws), "<cohort>/README.md")
        self.assertEqual(rp.guidance_key(f"{rp.RESEARCH}/topics/alpha/MISSION.md", ws), "<topic>/MISSION.md")
        self.assertEqual(rp.guidance_key(f"{rp.RESEARCH}/ATTENTION.md", ws), "<home>/ATTENTION.md")
        self.assertEqual(rp.guidance_key("docs/agent-system/LONG_EFFORT_DESIGN.md", ws), "docs/agent-system/LONG_EFFORT_DESIGN.md")


META = {"turn_id": "t1", "create_time": 1.5, "content_item_kinds": ["user.text"],
        "content_item_metadata": [{"provenance": {"type": "user"}}]}
CODEX_TURN = {  # a model turn as Codex 0.167 sends it, captured through the gate and trimmed
    "model": "gpt-6-luna", "stream": True, "tool_choice": "auto", "parallel_tool_calls": False,
    "reasoning": {"effort": "medium", "context": "all_turns"}, "store": False, "include": ["reasoning.encrypted_content"],
    "prompt_cache_key": "01a1", "text": {"verbosity": "low"}, "client_metadata": {"thread_id": "01a1", "turn_id": "01a2"},
    "access_programs": {"cyber": "standard"},
    "input": [
        {"type": "additional_tools", "id": "at_1", "role": "developer", "tools": [
            {"type": "namespace", "name": "functions", "description": "", "tools": [
                {"type": "custom", "name": "exec", "description": "Run JavaScript",
                 "format": {"type": "grammar", "syntax": "lark", "definition": "start: x"}},
                {"type": "function", "name": "wait", "description": "Waits", "strict": False,
                 "parameters": {"type": "object", "properties": {"cell_id": {"type": "string"}}}}]},
            {"type": "namespace", "name": "collaboration", "description": "Sub-agents", "tools": [
                {"type": "function", "name": "spawn_agent", "description": "Spawn", "strict": False, "parameters": {}}]}]},
        {"type": "message", "id": "msg_1", "role": "developer", "content": [{"type": "input_text", "text": "You are Codex."}],
         rp.META: META},
        {"type": "message", "id": "msg_2", "role": "user", "content": [{"type": "input_text", "text": "List two scenarios."},
                                                                         {"type": "input_image", "image_url": "data:image/png;base64,AA"}],
         rp.META: META},
        {"type": "configuration_update", "reasoning": {"effort": "medium"}},
        {"type": "reasoning", "id": "rs_1", "summary": [], "encrypted_content": "gAAA", "content": None, rp.META: "t1"},
        {"type": "function_call", "id": "fc_1", "call_id": "c1", "name": "wait", "namespace": "functions",
         "arguments": "{}", "status": "completed", rp.META: {"turn_id": "t1"}},
        {"type": "function_call_output", "id": "fco_1", "call_id": "c1", "output": "done", rp.META: {"turn_id": "t1"}},
        {"type": "custom_tool_call", "id": "ctc_1", "call_id": "c2", "name": "exec", "input": "ls", "status": "completed"},
        {"type": "custom_tool_call_output", "id": "ctco_1", "call_id": "c2", "output": [{"type": "input_text", "text": "a b"}]},
        {"type": "message", "id": "msg_3", "role": "assistant", "phase": "final_answer", "status": "completed",
         "content": [{"type": "output_text", "text": "a, b", "annotations": [], "logprobs": []}]},
        {"type": "compaction_trigger"}, {"type": "compaction", "id": "cmp_1", "encrypted_content": "ENC"}]}


def codex_turn(change=None) -> dict:
    """CODEX_TURN with one change applied to a copy."""
    turn = json.loads(json.dumps(CODEX_TURN))
    if change:
        change(turn)
    return turn


def tool(*tools):  # a change offering these tools beside Codex's own
    return lambda t: t["input"][0]["tools"][0]["tools"].extend(tools)


def part(*parts):  # a change adding these parts to the user's message
    return lambda t: t["input"][2]["content"].extend(parts)


def item(**it):  # a change adding this input item
    return lambda t: t["input"].append(it)


def key(k, v):  # a change setting this top-level key
    return lambda t: t.update({k: v})


class Gate(unittest.TestCase):
    """The gate on the host: CONNECT to the provider only, TLS terminated with its own CA, one checked request."""

    @staticmethod
    def tunnel(gate: rp.Gate, sni: str) -> ssl.SSLSocket:
        s = socket.socket(socket.AF_UNIX)
        s.connect(str(gate.sock))
        s.sendall(f"CONNECT {rp.PROVIDER}:443 HTTP/1.1\r\n\r\n".encode())
        assert b" 200 " in s.recv(200)
        return ssl.create_default_context(cafile=str(gate.ca)).wrap_socket(s, server_hostname=sni)

    @staticmethod
    def raw(gate: rp.Gate, data: bytes) -> bytes:
        s = socket.socket(socket.AF_UNIX)
        s.connect(str(gate.sock))
        s.sendall(data)
        got = s.recv(300)
        s.close()
        return got

    def test_only_the_three_routes_are_allowed_exactly_with_the_query_codex_sends(self):
        for method, target in (("POST", "/backend-api/codex/responses"), ("GET", "/backend-api/codex/models?client_version=0.167.0"),
                               ("GET", "/backend-api/wham/accounts/check")):
            self.assertTrue(rp.Gate.allowed(method, target), target)
        for method, target in (("GET", "/"), ("GET", "/backend-api/codex/../../x"), ("GET", "/backend-api/codex/%2e%2e/x"),
                               ("GET", "http://example.com/backend-api/codex/models"), ("DELETE", "/backend-api/codex/models"),
                               ("GET", "/backend-api/wham/accounts/check-x"), ("GET", "/backend-api/wham/accounts/check/x"),
                               ("GET", "/backend-api/codex/responses"), ("POST", "/backend-api/codex/responses?x=1"),
                               ("POST", "/backend-api/codex/responses/compact"), ("POST", "/backend-api/codex/analytics-events/events"),
                               ("GET", "//backend-api/codex/models"), ("GET", "/backend-api/codex/models/"),
                               ("GET", "/backend-api/codex/models?client_version=0.0.0&q=" + "A" * 3000),
                               ("GET", "/backend-api/codex/models?x=1"), ("GET", "/backend-api/codex/models?"),
                               ("GET", "/backend-api/wham/accounts/check?x=1")):
            self.assertFalse(rp.Gate.allowed(method, target), target)

    def test_a_model_turn_as_codex_sends_it_is_forwarded_as_the_gates_own_serialization(self):
        why, body = rp.Gate.model_request(json.dumps(CODEX_TURN, indent=1).encode())
        self.assertIsNone(why)
        self.assertEqual(body, json.dumps(CODEX_TURN, ensure_ascii=False).encode())
        self.assertEqual(rp.Gate.model_request(b'{"model": "m", "input": [], "text": {"verbosity": "\\u006cow"}}'),
                         (None, b'{"model": "m", "input": [], "text": {"verbosity": "low"}}'))

    def test_a_model_turn_is_refused_unless_strict_json(self):
        web = '{"type": "web_search"}'
        for body in (b"abc", b"[]", b"\xff", b"[" * 100000, b'{"model": "m", "input": [], "x": NaN}',
                     b'{"model": "m", "input": [], "text": {"verbosity": Infinity}}',
                     b'{"model": "m", "input": [], "client_metadata": {"a": 1e999}}', b'{"model": "m", "input": [], "x": -1e400}',
                     json.dumps(CODEX_TURN).encode("utf-16"),
                     ('{"model": "m", "input": [], "tools": [%s], "tools": []}' % web).encode(),  # parser differential
                     ('{"model": "m", "input": [], "tools": [], "tools": [%s]}' % web).encode(),
                     ('{"model": "m", "input": [{"type": "additional_tools", "role": "developer", "tools": [%s], "tools": []}]}'
                      % web).encode(),
                     b'{"model": "m", "input": [{"type": "additional_tools", "role": "developer",'
                     b' "tools": [{"type": "function", "type": "web_search", "name": "f"}]}]}'):
            why, out = rp.Gate.model_request(body)
            self.assertTrue(why and out == b"", body[:60])

    def test_a_model_turn_is_refused_unless_shaped_as_codex_sends_it(self):
        url = "https://attacker.example/x?d=SECRET"
        cases = {  # each one change to a turn Codex sends; the reviewers' probes plus their neighbours
            "top-level tools": key("tools", []), "tool_resources": key("tool_resources", {"file_search": {}}),
            "previous_response_id": key("previous_response_id", "resp_1"), "conversation": key("conversation", "conv_1"),
            "background": key("background", True), "prompt": key("prompt", {"id": "pmpt_1"}),
            "instructions": key("instructions", "x"), "web_search_options": key("web_search_options", {}),
            "include of tool results": key("include", ["web_search_call.action.sources", "file_search_call.results"]),
            "text format": key("text", {"format": {"type": "json_schema"}}), "store": key("store", True),
            "tool_choice allowed_tools": key("tool_choice", {"type": "allowed_tools", "tools": [{"type": "web_search"}]}),
            "tool_choice string": key("tool_choice", "web_search_preview"), "tool_choice required": key("tool_choice", "required"),
            "client_metadata nested": key("client_metadata", {"a": {"b": url}}), "input missing": lambda t: t.pop("input"),
            "input a string": key("input", "hi"), "reasoning extra": key("reasoning", {"effort": "high", "x": 1}),
            "web_search tool": tool({"type": "web_search"}),             "web_search beside a function": tool({"type": "function", "name": "f"}, {"type": "image_generation"}),
            "mcp tool at a URL": tool({"type": "mcp", "server_label": "s", "server_url": url}),
            "function with server fields": tool({"type": "function", "name": "f", "server_url": url, "connector_id": "c",
                                                 "external_web_access": True}),
            "tool without a type": tool({"name": "x"}), "tool as a string": tool("function"),
            "namespace in a namespace": tool({"type": "namespace", "name": "n", "tools": []}),
            "custom tool format with a URL": tool({"type": "custom", "name": "c", "format": {"type": "grammar", "url": url}}),
            "tools in a message": lambda t: t["input"][2].update(tools=[{"type": "web_search"}]),
            "additional_tools from the user": lambda t: t["input"][0].update(role="user"),
            "input_image at a URL": part({"type": "input_image", "image_url": url}),
            "input_image as svg": part({"type": "input_image", "image_url": "data:image/svg+xml;base64,PHN2Zz4="}),
            "input_image as data: text": part({"type": "input_image", "image_url": "data:text/html,<img src=x>"}),
            "remote $ref in parameters": tool({"type": "function", "name": "f", "parameters": {
                "type": "object", "properties": {"a": {"$ref": url}}}}),
            "remote $id in parameters": tool({"type": "function", "name": "f", "parameters": {"items": [{"$id": url}]}}),
            "access_programs other": key("access_programs", {"cyber": "elevated"}),
            "input_image by file id": part({"type": "input_image", "file_id": "file_1"}),
            "input_file at a URL": part({"type": "input_file", "file_url": url}),
            "input_file by id": part({"type": "input_file", "file_id": "file_1"}), "input_audio": part({"type": "input_audio"}),
            "output_text citing a URL": part({"type": "output_text", "text": "x", "annotations": [{"type": "url_citation", "url": url}]}),
            "mcp_call item": item(type="mcp_call", server_label="s", name="n", arguments="{}"),
            "web_search_call item": item(type="web_search_call", action={"type": "open_page", "url": url}),
            "item_reference": item(type="item_reference", id="msg_0"), "item without a type": item(role="user", content="hi"),
            "system message": item(type="message", role="system", content=[{"type": "input_text", "text": "x"}]),
            "message content a string": item(type="message", role="user", content="hi"),
            "tool output with an image URL": item(type="function_call_output", call_id="c", output=[
                {"type": "input_image", "image_url": url}]),
            "custom tool output with a file": item(type="custom_tool_call_output", call_id="c", output=[
                {"type": "input_file", "file_url": url}]),
            "reasoning with content": lambda t: t["input"][4].update(content=[{"type": "reasoning_text", "text": "x"}]),
            "metadata carrying a URL": lambda t: t["input"][5].update({rp.META: {"source": url}}),
            "compaction with more": item(type="compaction", encrypted_content="E", server_url=url),
        }
        bodies = {name: json.dumps(codex_turn(change)) for name, change in cases.items()}
        bodies["unicode-escaped web_search"] = bodies["web_search tool"].replace('"web_search"', '"\\u0077eb_search"')
        for name, body in bodies.items():
            why, out = rp.Gate.model_request(body.encode())
            self.assertTrue(why and out == b"", name)

    def test_one_request_reaches_the_provider_as_the_provider_and_nothing_after_it(self):
        with rp.Gate(token="real-T") as gate:
            up, provider = socket.socketpair()
            gate.upstream = lambda: up
            s = self.tunnel(gate, rp.PROVIDER)                                  # the gate's certificate verifies
            sent = json.dumps(CODEX_TURN, indent=2).encode()  # forwarded as the gate's own serialization
            s.sendall(b"POST /backend-api/codex/responses HTTP/1.1\r\nHost: example.com\r\nAuthorization: Bearer dummy\r\n"
                      b"Content-Length: %d\r\nX-Codex-Beta-Features: remote_compaction_v2\r\nContent-Type: text/plain\r\n"
                      b"Connection: keep-alive\r\n\r\n%sGET / HTTP/1.1\r\nHost: example.com\r\n\r\n" % (len(sent), sent))
            provider.settimeout(1)
            got = b""
            try:
                while d := provider.recv(65536):
                    got += d
            except socket.timeout:
                pass
            self.assertTrue(got.startswith(b"POST /backend-api/codex/responses HTTP/1.1\r\nHost: chatgpt.com\r\n"), got)
            body = json.dumps(CODEX_TURN, ensure_ascii=False).encode()
            self.assertTrue(got.endswith(b"\r\nConnection: close\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n%s"
                                         % (len(body), body)), got[-300:])
            self.assertNotIn(b"text/plain", got)                               # the gate's Content-Type, not the sandbox's
            self.assertNotIn(b"example.com", got)                              # neither its Host nor the pipelined request
            self.assertIn(b"\r\nAuthorization: Bearer real-T\r\n", got)        # the gate's token, not the sandbox's
            self.assertNotIn(b"dummy", got)
            provider.sendall(b"HTTP/1.1 200 OK\r\nContent-Length: 2\r\n\r\nok")
            provider.close()
            self.assertIn(b"200 OK", s.recv(200))
        self.assertFalse(gate.dir.exists())

    def exchange(self, gate: rp.Gate, request: bytes, reply: bytes = b"HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n"):
        """One request through the gate to a stand-in provider: what the provider received (b"" if the gate
        refused) and what the client got back."""
        up, provider = socket.socketpair()
        gate.upstream = lambda: up
        s = self.tunnel(gate, rp.PROVIDER)
        s.sendall(request)
        provider.settimeout(0.5)
        got = b""
        try:
            while b"\r\n\r\n" not in got or len(got.partition(b"\r\n\r\n")[2]) < int(
                    re.search(rb"Content-Length: (\d+)", got).group(1) if b"Content-Length" in got else 0):
                if not (d := provider.recv(65536)):
                    break
                got += d
            provider.sendall(reply)
        except socket.timeout:
            pass
        provider.close()
        return got, s.recv(300)

    def test_a_turn_state_passes_only_as_the_provider_issued_it_to_this_run(self):
        turn = json.dumps(CODEX_TURN).encode()
        post = b"POST /backend-api/codex/responses HTTP/1.1\r\nX-Codex-Turn-State: %s\r\nContent-Length: %d\r\n\r\n%s"
        with rp.Gate(token="T") as gate, rp.Gate(token="T") as other:
            sent, _ = self.exchange(gate, post.replace(b"X-Codex-Turn-State: %s\r\n", b"") % (len(turn), turn),
                                    b"HTTP/1.1 200 OK\r\nx-codex-turn-state: ts.A-1/=\r\nContent-Length: 0\r\n\r\n")
            self.assertTrue(sent)
            self.assertEqual(gate.turn_states, {"ts.A-1/="})                   # issued by the provider
            sent, _ = self.exchange(gate, post % (b"ts.A-1/=", len(turn), turn))
            self.assertIn(b"\r\nX-Codex-Turn-State: ts.A-1/=\r\n", sent)     # echoed as issued: forwarded
            for g, req in ((gate, post % (b"ts.A-1/=x", len(turn), turn)),     # forged
                           (gate, post % (b"ts.A-1", len(turn), turn)),        # a prefix of the issued value
                           (gate, b"GET /backend-api/codex/models?client_version=1 HTTP/1.1\r\n"
                                  b"X-Codex-Turn-State: ts.A-1/=\r\n\r\n"),    # on a GET
                           (other, post % (b"ts.A-1/=", len(turn), turn))):    # issued in another run
                sent, back = self.exchange(g, req)
                self.assertEqual(sent, b"", req[:80])
                self.assertIn(b"X-Replay-Gate: refused", back)
            for value in (b"x" * 2049, b"a b"):                                # too long, or not one visible token
                self.exchange(other, post.replace(b"X-Codex-Turn-State: %s\r\n", b"") % (len(turn), turn),
                              b"HTTP/1.1 200 OK\r\nx-codex-turn-state: %s\r\nContent-Length: 0\r\n\r\n" % value)
            self.assertEqual(other.turn_states, set())
            self.assertEqual([e["request"] for e in gate.log if not e["allowed"]],
                             ["POST /backend-api/codex/responses (header)"] * 2 + ["GET /backend-api/codex/models (header)"])

    def test_a_gate_with_model_turns_off_refuses_every_post_and_makes_no_call(self):
        turn = json.dumps(CODEX_TURN).encode()
        post = b"POST /backend-api/codex/responses HTTP/1.1\r\nContent-Length: %d\r\n\r\n%s" % (len(turn), turn)
        with rp.Gate(token="T", model_turns=False) as gate:
            sent, back = self.exchange(gate, post)
            self.assertEqual(sent, b"")
            self.assertIn(b"X-Replay-Gate: refused", back)
            sent, _ = self.exchange(gate, b"GET /backend-api/codex/models?client_version=1 HTTP/1.1\r\n\r\n")
            self.assertTrue(sent)                                               # the other routes stay as they are
            self.assertEqual(gate.summary()["denied"], ["POST /backend-api/codex/responses (model turns off)"])
            self.assertEqual(gate.requests, 1)
        with rp.Gate(token="T") as gate:
            self.assertTrue(self.exchange(gate, post)[0])                      # the default forwards the turn

    def test_everything_else_is_refused(self):
        with rp.Gate() as gate, mock.patch.object(rp.Gate, "BODY_WAIT", 0.5):
            for data in (b"CONNECT example.com:443 HTTP/1.1\r\n\r\n", b"CONNECT chatgpt.com:80 HTTP/1.1\r\n\r\n",
                         b"GET http://example.com/ HTTP/1.1\r\n\r\n", b"GET /backend-api/codex/models HTTP/1.1\r\n\r\n",
                         b"\x16\x03\x01\x02\x00\x01\x00\x01\xfc\x03\x03"):
                self.assertIn(b"X-Replay-Gate: refused", self.raw(gate, data), data)
            for sni in ("example.com", "cloudflare.com"):                       # domain fronting
                with self.assertRaises(ssl.SSLError):
                    self.tunnel(gate, sni).sendall(b"GET / HTTP/1.1\r\nHost: example.com\r\n\r\n")
            web = b'{"model": "m", "input": [], "tools": [{"type": "web_search"}], "tools": []}'
            image = json.dumps(codex_turn(part({"type": "input_image", "image_url": "https://example.com/x"}))).encode()
            ok = json.dumps(CODEX_TURN).encode()
            post = b"POST /backend-api/codex/responses HTTP/1.1\r\n%sContent-Length: %d\r\n\r\n%s"
            for req in (b"GET / HTTP/1.1\r\nHost: example.com\r\n\r\n",
                        b"GET /backend-api/codex/models HTTP/1.1\r\nX-A: a\nHost: example.com\r\n\r\n",
                        b"GET /backend-api/codex/models HTTP/1.1\r\nX-A: a\x00b\r\n\r\n",
                        b"GET /backend-api/codex/models HTTP/1.1\r\nX A: b\r\n\r\n",
                        b"GET /backend-api/codex/models HTTP/1.1\r\njunk-line\r\n\r\n",
                        b"GET /backend-api/codex/models HTTP/1.1\r\nX-T: a\r\n b\r\n\r\n",
                        b"GET /backend-api/codex/models HTTP/1.1\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n\r\n",
                        b"GET /backend-api/codex/models HTTP/1.1\r\nContent-Length: 2\r\n\r\n{}",
                        b"POST /backend-api/codex/responses HTTP/1.1\r\nContent-Length: %d\r\n\r\n%s" % (len(web), web),
                        b"POST /backend-api/codex/responses HTTP/1.1\r\nContent-Encoding: zstd\r\nContent-Length: 2\r\n\r\n{}",
                        b"POST /backend-api/codex/responses HTTP/1.1\r\nContent-Length: %d\r\n\r\n" % (rp.BODY_CAP + 1),
                        post % (b"", len(image), image),                                  # an image the provider fetches
                        post % (b"", len(ok) - 5, ok), post % (b"", len(ok) + 50, ok),    # a Content-Length off the body
                        post % (b"Content-Length: 2\r\n", len(ok), ok),                   # two of them
                        post % (b"X-Forwarded-Host: example.com\r\n", len(ok), ok),       # a header Codex never sends
                        post % (b"X-Codex-Beta-Features: remote_compaction_v2, web_search\r\n", len(ok), ok),
                        b"POST /backend-api/codex/responses?x=1 HTTP/1.1\r\nContent-Length: 2\r\n\r\n{}",
                        b"GET /backend-api/codex/models?client_version=0.0.0&q=1 HTTP/1.1\r\n\r\n",
                        b"GET /backend-api/wham/accounts/check?x=1 HTTP/1.1\r\n\r\n",
                        b"GET /backend-api/codex/models/ HTTP/1.1\r\n\r\n"):
                s = self.tunnel(gate, rp.PROVIDER)
                s.sendall(req)
                self.assertIn(b"X-Replay-Gate: refused", s.recv(300), req)
            time.sleep(0.2)
            self.assertFalse(any(e["allowed"] for e in gate.log))
            self.assertEqual(sum(e["request"].startswith("TLS refused") for e in gate.log), 2)

    def test_a_run_has_a_request_and_byte_budget(self):
        with rp.Gate(token="T") as gate:
            gate.upstream = lambda: socket.socketpair()[0]  # never reached
            gate.requests = rp.GATE_REQUESTS
            s = self.tunnel(gate, rp.PROVIDER)
            s.sendall(b"GET /backend-api/codex/models HTTP/1.1\r\n\r\n")
            self.assertIn(b"X-Replay-Gate: refused", s.recv(300))
            gate.requests, gate.bytes = 0, rp.GATE_BYTES
            s = self.tunnel(gate, rp.PROVIDER)
            s.sendall(b"GET /backend-api/codex/models HTTP/1.1\r\n\r\n")
            self.assertIn(b"X-Replay-Gate: refused", s.recv(300))
            time.sleep(0.2)
            self.assertEqual([e["request"] for e in gate.log], ["GET /backend-api/codex/models (budget)"] * 2)

    def test_the_provider_must_resolve_to_public_addresses(self):
        for addr in ("127.0.0.1", "10.1.2.3", "::1", "169.254.169.254"):
            with mock.patch.object(rp.socket, "getaddrinfo", return_value=[(0, 0, 0, "", (addr, 443))]):
                with self.assertRaisesRegex(OSError, "non-public"):
                    rp.Gate.upstream()


@unittest.skipUnless(shutil.which("bwrap") and shutil.which("openssl"), "bubblewrap or openssl not installed")
class Sandbox(unittest.TestCase):
    """The sandbox every replay runs in, driven with shell probes and the hostile review's probes (p1-p4)
    instead of a model."""

    def setUp(self):
        base = Path.home() / ".vrooli/tmp"
        base.mkdir(parents=True, exist_ok=True)
        self.tmp = tempfile.TemporaryDirectory(dir=base)
        self.ws = fake_workspace(Path(self.tmp.name))

    def tearDown(self):
        self.tmp.cleanup()

    def run_in(self, shell: str) -> subprocess.CompletedProcess:
        cmd = rp.sandbox_cmd(["bash", "-c", shell], ws=self.ws, variant="old", cohort="c1")
        return subprocess.run(cmd, capture_output=True, text=True, timeout=60, env=rp.sandbox_env())

    def drive(self, code: str, gate: rp.Gate) -> list[str]:
        """Run Python as the one turn DRIVER runs (gate, Codex home tmpfs, login copy); its output lines."""
        cmd = rp.sandbox_cmd(rp.DRIVE, ws=self.ws, variant="old", cohort="c1", gate=gate)
        p = subprocess.run(cmd, input=json.dumps(rp.codex_header([["python3", "-I", "-c", code]])) + "\n",
                           capture_output=True, text=True, timeout=120, env=rp.sandbox_env())
        return [x for x in p.stdout.splitlines() if not x.startswith('{"replay"')]

    def test_every_write_outside_the_capped_tmpfs_is_refused(self):
        probe = f".replay-test-probe-{os.getpid()}"
        for target in (rp.REPO / probe, rp.RESEARCH / probe, rp.RESEARCH / "cohorts" / probe, rp.RESEARCH / "cohorts/c1" / probe,
                       rp.SKILLS / probe, Path.home() / probe, Path.home() / ".vrooli/state" / probe,
                       Path.home() / ".vrooli/bin" / probe, rp.EFFORT_DIR / probe, Path("/") / probe, Path("/dev") / probe,
                       Path("/etc") / probe):
            p = self.run_in(f"echo x > '{target}'")
            self.assertNotEqual(p.returncode, 0, target)
            self.assertFalse(target.exists(), target)
        p = self.run_in("touch /tmp/replay-private-probe && echo ok")
        self.assertEqual(p.stdout.strip(), "ok")
        self.assertFalse(Path("/tmp/replay-private-probe").exists())

    def test_the_variant_and_snapshot_appear_at_the_real_paths_and_nothing_else(self):
        p = self.run_in(f"cat '{rp.SKILLS}/research-coordination/SKILL.md' '{rp.RESEARCH}/cohorts/c1/README.md' "
                        f"'{rp.RESEARCH}/topics/alpha/MISSION.md' '{rp.REPO}/scenarios/a/api/main.go'; "
                        f"ls '{rp.EFFORT_DIR}' '{rp.SKILLS}/research-review' 2>&1; prompt-manager skill read other-skill; "
                        "ls /etc | wc -l; test -e /etc/shadow; echo shadow=$?")
        for text in ("Research coordination old", "charter old", "package main", "Other skill body", "shadow=1"):
            self.assertIn(text, p.stdout)
        self.assertEqual(p.stdout.count("No such file"), 2)                    # no effort folder, no harness
        self.assertLessEqual(int(p.stdout.splitlines()[-2]), len(rp.ETC_ALLOW))

    def test_services_and_host_sockets_are_unreachable(self):
        srv = socket.socket()
        srv.bind(("127.0.0.1", 0))
        srv.listen(1)
        port = srv.getsockname()[1]
        sock_path = Path(self.tmp.name) / "s.sock"
        u = socket.socket(socket.AF_UNIX)
        u.bind(str(sock_path))
        u.listen(1)
        try:
            p = self.run_in(f"python3 -I -c \"import socket; socket.create_connection(('127.0.0.1', {port}), 2)\"; echo tcp=$?; "
                            f"test -e '{sock_path}'; echo unix=$?; research-desk say hi; echo say=$?; "
                            "ls /proc | grep -c '^[0-9]'")
        finally:
            srv.close()
            u.close()
        self.assertIn("tcp=1", p.stdout)
        self.assertIn("unix=1", p.stdout)
        self.assertIn("say=69", p.stdout)
        self.assertLess(int(p.stdout.strip().splitlines()[-1]), 10)            # own pid namespace

    def test_the_environment_is_built_not_inherited(self):
        os.environ["VROOLI_AGENT_IDENTITY_TOKEN"] = "secret"
        try:
            p = self.run_in("env")
        finally:
            del os.environ["VROOLI_AGENT_IDENTITY_TOKEN"]
        self.assertNotIn("secret", p.stdout)
        self.assertIn(f"CODEX_HOME={Path.home()}/.codex", p.stdout)

    def test_p1_only_the_capped_tmpfs_mounts_are_writable(self):
        with rp.Gate() as gate:
            out = self.drive("import os\nprint(len(os.listdir('/proc/self/fd')))\n"
                             "for l in open('/proc/self/mountinfo'):\n"
                             "    f = l.split()\n"
                             "    if f[5].startswith('rw') and f[-3] != 'proc' and f[4] not in ('/dev/pts',) "
                             "and not f[4].startswith('/dev/') or f[4] == '/dev/shm':\n"
                             "        print(f[4], f[-3], os.statvfs(f[4]).f_blocks * os.statvfs(f[4]).f_frsize)", gate)
        self.assertLessEqual(int(out[0]), 4)
        self.assertEqual(sorted(x.split()[:2] for x in out[1:]),
                         sorted([["/dev/shm", "tmpfs"], ["/tmp", "tmpfs"], [str(Path.home() / ".codex"), "tmpfs"]]))
        caps = {x.split()[0]: int(x.split()[2]) for x in out[1:]}
        self.assertEqual(caps, {"/dev/shm": rp.SHM_CAP, "/tmp": rp.TMP_CAP, str(Path.home() / ".codex"): rp.CODEX_HOME_CAP})

    def test_p2_p3_writes_and_the_review_secrets_fail_inside(self):
        targets = [str(rp.REPO / "x"), str(Path.home() / ".vrooli/x"), str(rp.RESEARCH / "x"), "/run/x", "/var/x", "/usr/x",
                   "/etc/x", str(Path.home() / ".codex/plugins/x")] + [str(rp.REPO / r) for r in PLANTED]
        code = ("import json, sys\nout = {}\nfor p in json.loads(sys.argv[1]):\n"
                "    try:\n        open(p, 'a' if p.endswith('/x') else 'r').close(); out[p] = 'OPENED'\n"
                "    except OSError as e:\n        out[p] = e.strerror\nprint(json.dumps(out))")
        with rp.Gate() as gate:
            out = self.drive(code.replace("sys.argv[1]", repr(json.dumps(targets))), gate)
        got = json.loads(out[-1])
        self.assertEqual([p for p, r in got.items() if r == "OPENED"], [])

    def test_p4_connect_targets_and_domain_fronting_are_refused_inside(self):
        code = r"""
import json, socket, ssl
out = {}
def conn(t):
    s = socket.create_connection(("127.0.0.1", %d), timeout=15)
    s.sendall(("CONNECT %%s HTTP/1.1\r\nHost: %%s\r\n\r\n" %% (t, t)).encode())
    return s, s.recv(300).decode("latin1").split("\r\n")[0]
for t in ["CHATGPT.COM:443", "chatgpt.com.:443", "chatgpt.com:80", "x.chatgpt.com:443", "localhost.chatgpt.com:443",
          "evil.com#.chatgpt.com:443", "[::1]:443", "2130706433:443", "0x7f000001:443", "localhost:443",
          "chatgpt.com@evil.com:443", "evil.com:443", "ab.chatgpt.com:443"]:
    s, r = conn(t); out[t] = r; s.close()
for sni, host in [("example.com", "example.com"), ("cloudflare.com", "www.cloudflare.com"), ("chatgpt.com", "example.com")]:
    try:
        s, r = conn("chatgpt.com:443")
        t = ssl.create_default_context().wrap_socket(s, server_hostname=sni)
        t.sendall(("GET / HTTP/1.1\r\nHost: %%s\r\nConnection: close\r\n\r\n" %% host).encode())
        out["front " + sni + " " + host] = t.recv(400).decode("latin1")
    except Exception as e:
        out["front " + sni + " " + host] = "ERR " + type(e).__name__
print(json.dumps(out))
""" % rp.PROXY_PORT
        with rp.Gate() as gate:
            got = json.loads(self.drive(code, gate)[-1])
            log = gate.summary()
        for t, r in got.items():
            self.assertTrue(r.startswith("ERR") or "X-Replay-Gate: refused" in r or r == "HTTP/1.1 403 Forbidden", (t, r))
        self.assertEqual(log["allowed"], {})

    def test_no_login_copy_is_ever_written_to_the_host(self):
        self.assertFalse(hasattr(rp, "make_codex_home"))
        replay_tmp = Path.home() / ".vrooli/tmp/research-replay"
        before = set(os.listdir(replay_tmp)) if replay_tmp.is_dir() else set()
        code = ("import json, os, stat\nh = os.environ['CODEX_HOME']\na = json.load(open(h + '/auth.json'))\n"
                "print(json.dumps({'refresh': a.get('tokens', {}).get('refresh_token'), "
                "'mode': oct(stat.S_IMODE(os.stat(h + '/auth.json').st_mode)), 'fs': [l.split()[-3] for l in "
                "open('/proc/self/mountinfo') if l.split()[4] == h]}))")
        with rp.Gate() as gate:
            gate_dir = gate.dir
            files_in_gate = []
            out = self.drive(code, gate)
            files_in_gate = sorted(p.name for p in gate.dir.iterdir())
        got = json.loads(out[-1])
        self.assertEqual(got["fs"], ["tmpfs"])                                  # the login copy lives in the sandbox's tmpfs
        self.assertEqual(got["mode"], "0o600")
        if (Path.home() / ".codex/auth.json").exists():
            self.assertEqual(got["refresh"], "")
        self.assertNotIn("auth.json", files_in_gate)
        self.assertFalse(gate_dir.exists())
        self.assertEqual(set(os.listdir(replay_tmp)) if replay_tmp.is_dir() else set(), before)


if __name__ == "__main__":
    unittest.main()
