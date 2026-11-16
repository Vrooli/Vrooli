#!/usr/bin/env python3
"""Tests for replay.py: parsing, brief rendering, metrics, grading inputs and the sandbox. No model is called.

Run: python3 -I test_replay.py   (from this folder)
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
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



class Statistics(unittest.TestCase):
    def test_sign_test(self):
        self.assertAlmostEqual(rp.sign_test(9, 1), 0.0215, places=4)
        self.assertEqual(rp.sign_test(5, 5), 1.0)
        self.assertIsNone(rp.sign_test(0, 0))


class Harness(unittest.TestCase):
    """The turn loop and the old-skill shim, driven by fake agents (no model)."""

    def fake(self, lines: list[dict], sleep: int) -> list[str]:
        code = ("import json, time\n"
                f"for e in json.loads({json.dumps(lines)!r}):\n"
                "    print(json.dumps(e), flush=True)\n"
                f"time.sleep({sleep})\n")
        return [sys.executable, "-I", "-c", code]

    def run_stream(self, cmd, until):
        import io
        buf = io.StringIO()
        t0 = __import__("time").time()
        res = rp.stream(cmd, dict(os.environ), t0 + 60, 1, buf, t0, until=until)
        return res, __import__("time").time() - t0, [json.loads(l) for l in buf.getvalue().splitlines()]

    def test_stance_turn_ends_on_the_reply(self):
        cmd = self.fake([{"type": "thread.started", "thread_id": "th"},
                         {"type": "item.completed", "item": {"type": "agent_message", "text": "Ready."}}], sleep=50)
        res, took, events = self.run_stream(cmd, "reply")
        self.assertEqual(res["killed"], "answered")
        self.assertEqual(res["thread"], "th")
        self.assertLess(took, 25)
        self.assertEqual(events[0]["turn"], 1)

    def test_plan_turn_waits_for_the_json(self):
        cmd = self.fake([{"type": "item.completed", "item": {"type": "agent_message", "text": "Reading the charter."}},
                         {"type": "item.completed", "item": {"type": "agent_message", "text": '{"role": "owner"}'}}], sleep=50)
        res, took, _ = self.run_stream(cmd, "plan")
        self.assertEqual(res["killed"], "answered")
        self.assertLess(took, 25)

    def test_a_turn_that_ends_itself_is_not_killed(self):
        cmd = self.fake([{"type": "item.completed", "item": {"type": "agent_message", "text": '{"role": "owner"}'}},
                         {"type": "turn.completed", "usage": {}}], sleep=0)
        res, _, _ = self.run_stream(cmd, "plan")
        self.assertIsNone(res["killed"])
        self.assertEqual(res["exit"], 0)




def fake_workspace(root: Path, variants=("old",)) -> rp.WS:
    """A workspace with a snapshot and variants, as snapshot/freeze would leave them (no model, no CLI)."""
    ws = rp.WS(root)
    snap = ws.snapshot()
    rp.write_json(snap / "snapshot.json", {"research_home": str(rp.RESEARCH)})
    rp.write(snap / "home/topics/alpha/MISSION.md", MISSION)
    (snap / "home/cohorts").mkdir(parents=True)
    rp.write(snap / "repo/AGENTS.md", "agents")
    rp.write(snap / "skills/other-skill.md", "---\nname: o\n---\nOther skill body\n")
    rp.write_cli_stubs(snap / "bin", ["research-desk", "agent-manager", "prompt-manager"], rp.SKILLS, rp.SKILL_EXPORT)
    for v in variants:
        rp.write(ws.variant(v) / "skills/research-coordination/SKILL.md", f"---\nname: rc\n---\nResearch coordination {v}\n")
        rp.write(ws.variant(v) / "cohorts/c1/README.md", f"charter {v}")
    return ws


def run_record(ws: rp.WS, name: str, variant: str, role: str, size: int, plan: bool = True, topic=None):
    run = ws.runs() / name
    run.mkdir(parents=True)
    rp.write_json(run / "meta.json", {"run_id": name, "commission_id": "c1", "variant": variant, "role": role,
                                      "topic": topic, "harness": "codex", "model": "m", "effort": "e", "wall_s": 10, "cwd": "/"})
    tail = SUPERVISOR_EVENTS[-2:] if plan else [ev(7, 2, {"type": "item.completed", "item": {"type": "agent_message", "text": "no"}})]
    events = [ev(2, 2, cmd_item(1, "/bin/bash -lc 'cat /etc/hostname'", "z" * size))] + tail
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


class Statistics(unittest.TestCase):
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


class Gate(unittest.TestCase):
    def test_only_the_provider_is_tunnelled(self):
        with rp.Gate() as gate:
            for target in ("chatgpt.com:443", "ab.chatgpt.com:443"):
                self.assertTrue(gate.allowed(target), target)
            for target in ("chatgpt.com:80", "evilchatgpt.com:443", "auth.openai.com:443", "127.0.0.1:443",
                           "localhost:443", "example.com:443", "[::1]:443"):
                self.assertFalse(gate.allowed(target), target)
            import socket
            s = socket.socket(socket.AF_UNIX)
            s.connect(str(gate.sock))
            s.sendall(b"CONNECT 127.0.0.1:22 HTTP/1.1\r\n\r\n")
            self.assertIn(b"403", s.recv(100))
            s.close()
            for _ in range(50):
                if gate.log:
                    break
                __import__("time").sleep(0.02)
            self.assertEqual(gate.summary()["denied"], ["CONNECT 127.0.0.1:22 HTTP/1.1"])
        self.assertFalse(gate.dir.exists())


@unittest.skipUnless(shutil.which("bwrap"), "bubblewrap not installed")
class Sandbox(unittest.TestCase):
    """The sandbox every replay runs in, driven with shell probes instead of a model."""

    def setUp(self):
        base = Path.home() / ".vrooli/tmp"
        base.mkdir(parents=True, exist_ok=True)
        self.tmp = tempfile.TemporaryDirectory(dir=base)
        self.ws = fake_workspace(Path(self.tmp.name))

    def tearDown(self):
        self.tmp.cleanup()

    def run_in(self, shell: str, **kw) -> subprocess.CompletedProcess:
        cmd = rp.sandbox_cmd(["bash", "-c", shell], ws=self.ws, variant="old", cohort="c1", **kw)
        return subprocess.run(cmd, capture_output=True, text=True, timeout=60, env=rp.sandbox_env(proxy=False))

    def test_every_write_outside_tmp_is_refused(self):
        probe = f".replay-test-probe-{os.getpid()}"
        for target in (rp.REPO / probe, rp.RESEARCH / probe, rp.RESEARCH / "cohorts/c1" / probe, rp.SKILLS / probe,
                       Path.home() / probe, Path.home() / ".vrooli/state" / probe, Path.home() / ".vrooli/bin" / probe,
                       rp.EFFORT_DIR / probe):
            p = self.run_in(f"echo x > '{target}'")
            self.assertNotEqual(p.returncode, 0, target)
            self.assertFalse(target.exists(), target)
        p = self.run_in("touch /tmp/replay-private-probe && echo ok")
        self.assertEqual(p.stdout.strip(), "ok")
        self.assertFalse(Path("/tmp/replay-private-probe").exists())

    def test_the_variant_and_snapshot_appear_at_the_real_paths(self):
        p = self.run_in(f"cat '{rp.SKILLS}/research-coordination/SKILL.md' '{rp.RESEARCH}/cohorts/c1/README.md' "
                        f"'{rp.RESEARCH}/topics/alpha/MISSION.md'; ls '{rp.EFFORT_DIR}' | wc -l; "
                        f"ls '{rp.SKILLS}/research-review' 2>&1 | head -1; prompt-manager skill read other-skill")
        self.assertIn("Research coordination old", p.stdout)
        self.assertIn("charter old", p.stdout)
        self.assertIn("\n0\n", p.stdout)                                        # the effort folder is empty
        self.assertIn("No such file", p.stdout)                                 # the harness is not there
        self.assertIn("Other skill body", p.stdout)

    def test_services_and_host_sockets_are_unreachable(self):
        import socket
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

if __name__ == "__main__":
    unittest.main()
