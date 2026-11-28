#!/usr/bin/env python3
"""Tests for packet.py. Run: python3 -m unittest test_packet (from this directory)."""

from __future__ import annotations

import contextlib
import datetime as dt
import io
import json
import os
import re
import subprocess
import sys
import tempfile
import unittest
from unittest import mock
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import packet  # noqa: E402
from test_mission_check import WORKSTREAM, mission  # noqa: E402

CONTRACT_LINES = (
    "State: see STATE.md beside this file.",
    "Read entries of `OPERATOR.md` beside this file that are newer than the last one you acted on, at every unit start; they are the operator's words and win over this charter",
    "Write each decision as a needs-you item in its topic's `DECISIONS.md` (grammar in the research-coordination skill); research-desk delivers it, or you do by the fallback below.",
    "its row in `research-desk needs-you` says why; fix that so the\ndesk sends it, never by hand.",
    "`stat -c %Y` on `OPERATOR.md` and the card topics'",
    "`Commission: programme-five-20261112`",
    "### <UTC> · delivered <ID> · via desk",
    "`Route failure · <UTC> · <route> · <command> · <first error line> · direct path taken`",
)


class PacketTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.home = Path(self.tmp.name)
        (self.home / "cohorts").mkdir()
        (self.home / "ATTENTION.md").write_text("# Attention\n\n## Actionable now\n\nNo actionable items are recorded.\n")
        self.topic("alpha", mission())
        self.topic("beta", mission(rows=[
            WORKSTREAM.format(id="B1", hours="40", status="landed 2026-11-01"),
            WORKSTREAM.format(id="B2", hours="40", status="approved"),
            WORKSTREAM.format(id="B3", hours="40", status="waiting-operator 2026-11-10"),
            WORKSTREAM.format(id="B4", hours="40", status="open"),
            WORKSTREAM.format(id="B5", hours="40", status="active"),
        ]))
        (self.home / "topics" / "alpha" / "FILMS.md").write_text("# Films\n\n## Wanted\n\n- a film\n")
        self._today = packet.today
        packet.today = lambda: dt.date(2026, 11, 12)

    def tearDown(self):
        packet.today = self._today
        self.tmp.cleanup()

    def topic(self, name, text):
        d = self.home / "topics" / name
        d.mkdir(parents=True)
        (d / "MISSION.md").write_text(text.replace("# T — mission", f"# {name.title()} things — mission"))

    def spec(self, **over):
        s = {"topics": ["alpha", "beta"], "allowance_hours": 12, "stance_quotes": ["Make it big."],
             "slug": "programme-five", "window": "fifth"}
        s.update(over)
        path = self.home / "spec.json"
        path.write_text(json.dumps(s))
        return str(path)

    def run_cli(self, *argv):
        out = io.StringIO()
        with contextlib.redirect_stdout(out):
            code = packet.main(list(argv))
        return code, json.loads(out.getvalue())

    def preview(self, **over):
        return self.run_cli("preview", "--home", str(self.home), "--spec", self.spec(**over))

    def build(self, **over):
        return self.run_cli("build", "--home", str(self.home), "--spec", self.spec(**over))

    def test_preview_renders_the_contract_texts_and_writes_nothing(self):
        code, res = self.preview(emphasis="Finish the films first.")
        self.assertEqual(code, 0, res["errors"])
        self.assertEqual(res["schema"], "research-packet-result/v1")
        self.assertEqual((res["ok"], res["written"], res["commission_id"]), (True, False, "programme-five-20261112"))
        self.assertEqual(sorted(self.home.joinpath("cohorts").iterdir()), [])
        charter, stance, kernel = res["texts"]["charter"], res["texts"]["stance"], res["texts"]["kernel"]
        for line in CONTRACT_LINES:
            self.assertIn(line, charter)
        self.assertIn('> "Make it big."', charter)
        self.assertIn('"Make it big."', stance)
        self.assertIn("Finish the films first.", stance)
        self.assertTrue(stance.rstrip().endswith('Reply "ready" and wait for it.'))
        self.assertIn('Stance: "Finish the films first"', kernel)
        self.assertLessEqual(len(kernel), 600)
        self.assertNotIn("{{", charter + stance + kernel)
        self.assertIn("`FILMS.md`", charter)
        self.assertIn(str(packet.PATHS["footprint"]), charter)
        self.assertIn("`model` and `reasoning_effort` omitted", charter)
        self.assertTrue(all(g["ok"] for g in res["gates"]))

    def test_build_uses_retained_feedback_without_contacting_hub(self):
        topic = self.home / "topics" / "alpha"
        (topic / "reports").mkdir()
        (topic / "reports" / "previous.json").write_text('{"ask_id":"unavailable"}')
        (topic / "feedback").mkdir()
        (topic / "feedback" / "steering.md").write_text("Use the shorter opening.")
        with mock.patch("subprocess.run", side_effect=AssertionError("build called a service")):
            code, res = self.build()
        self.assertEqual(code, 0, res["errors"])
        self.assertTrue(res["written"])
        snapshot = self.home / "cohorts" / res["commission_id"] / "FEEDBACK.md"
        self.assertIn("Use the shorter opening.", snapshot.read_text())

    def test_without_an_order_cards_take_active_approved_then_open_and_say_it_is_not_priority(self):
        _, res = self.preview()
        self.assertEqual(res["cards"][0], {"topic": "alpha", "mode": "build", "order": ["A2", "A4", "A1"],
                                           "order_source": "mission", "read_first": None, "clause_cap": 100})
        self.assertEqual(res["cards"][1]["order"], ["B5", "B2", "B4"])
        charter, kernel = res["texts"]["charter"], res["texts"]["kernel"]
        self.assertIn(packet.CHOICE_LABEL, charter)
        self.assertIn("listed for it below, chosen by scorecard movement,", charter)
        self.assertNotIn("in order", kernel)

    def test_the_operator_order_notes_and_read_first_land_on_the_card_verbatim(self):
        notes = "1. **A4 first hour:** fetch the corpus.\n- **Return:** local only; tell the beta owner when it exists."
        read = self.home / "topics" / "beta" / "CHECKPOINT.md"
        read.write_text("# checkpoint\n")
        _, res = self.preview(order={"alpha": ["A4", "A1"]}, card_notes={"alpha": notes},
                              read_first={"beta": str(read)})
        charter = res["texts"]["charter"]
        self.assertEqual([c["order"] for c in res["cards"]], [["A4", "A1"], ["B5", "B2", "B4"]])
        self.assertEqual([c["order_source"] for c in res["cards"]], ["operator", "mission"])
        card = charter.split("### Card 1", 1)[1].split("### Card 2", 1)[0]
        self.assertIn(packet.ORDER_LABEL, card)
        self.assertIn(notes, card)
        self.assertLess(card.index("**A4**"), card.index("**A1**"))
        self.assertIn("**A4** (25 h · owner): Demo runs live", card)  # full magnitude and showcase, untruncated
        self.assertIn(f"**Read first:** `{read}`", charter)
        self.assertIn("in the order a card gives, else by scorecard movement", charter)
        self.assertIn("(in order where given)", res["texts"]["kernel"])
        _, res = self.preview(order={"alpha": ["A1"], "beta": ["B4", "B2"]})
        self.assertIn("listed for it below, in order,", res["texts"]["charter"])
        self.assertIn("showcases in order,", res["texts"]["kernel"])

    def test_cards_carry_what_the_mission_states(self):
        alpha = self.home / "topics" / "alpha" / "MISSION.md"
        alpha.write_text(alpha.read_text()
                         .replace("Build directly.", "Build directly; the beta owner leaves a demo asset.\n\n"
                                  "**Return in this window.** Results stay local.")
                         .replace("- Trap one.", "- **Reports instead of films**: a catalogue (of tools) is not a film. More words follow here.")
                         .replace("- No money.", "- [Spend](https://example.com) and tools: free and local tools only; Matt owns any purchase."))
        _, res = self.preview(order={"alpha": ["A1"]})
        charter = res["texts"]["charter"]
        alpha_card, beta_card = charter.split("### Card 1", 1)[1].split("### Card 2", 1)
        beta_card = beta_card.split("## Manager procedure", 1)[0]
        self.assertIn("- **Also open** (taken when the card's workstreams land or wait): A2, A4", alpha_card)
        self.assertNotIn("Waits on the operator", alpha_card)  # nothing waits: no line
        alpha = self.home / "topics" / "alpha" / "MISSION.md"
        alpha.write_text(alpha.read_text().replace("## Approval flow", "## Later\n\n- **A9** — More · 40–80 h\n\n## Approval flow", 1))
        _, res = self.preview(order={"alpha": ["A1"]})
        self.assertIn("wait): A2, A4; then the mission's Later list: A9", res["texts"]["charter"])
        self.assertNotIn("Also open", beta_card)  # B5, B2, B4 are every live row
        self.assertIn("- **Waits on the operator:** B3 (waiting-operator); the rest is in the mission's Approval flow.", beta_card)
        self.assertIn("- **Return:** the mission's \"Return in this window\".", alpha_card)
        self.assertIn("- **Hand-off:** the mission names Card 2 (`beta`).", alpha_card)
        self.assertNotIn("Hand-off", beta_card)
        self.assertIn("- **Not this** (first clauses): Reports instead of films: a catalogue is not a film; Trap two.", alpha_card)
        self.assertIn("- **Boundaries** (first clauses): Spend and tools: free and local tools only.", alpha_card)

    def test_mission_excerpts_shrink_until_the_charter_fits_and_say_so(self):
        _, full = self.preview()
        limits = packet.load_limits()
        limits["gates"]["charter_bytes"] = len(full["texts"]["charter"].encode()) - 1
        with mock.patch.object(packet, "load_limits", return_value=limits):
            code, res = self.preview()
        self.assertEqual(code, 0, res["errors"])
        self.assertEqual({c["clause_cap"] for c in res["cards"]}, {0})
        self.assertEqual(res["texts"]["charter"].count("the charter gate left no room for them here"), 2)
        self.assertNotIn("(first clauses)", res["texts"]["charter"])

    def test_a_quote_with_an_ellipsis_is_never_called_verbatim(self):
        _, res = self.preview()
        self.assertIn("## Operator stance (verbatim;", res["texts"]["charter"])
        self.assertIn("My words, verbatim:", res["texts"]["stance"])
        for cut in ("Make it big … and bold.", "Make it big... and bold."):
            _, res = self.preview(stance_quotes=["Finish the films first.", cut])
            self.assertNotIn("verbatim", res["texts"]["stance"])
            self.assertIn("## Operator stance (shortened where an ellipsis marks a cut;", res["texts"]["charter"])
            self.assertIn("My words, shortened where an ellipsis marks a cut:", res["texts"]["stance"])
            self.assertIn(cut, res["texts"]["stance"])  # offered exactly as written

    def test_six_topics_fit_with_two_workstreams_per_card(self):
        names = ["alpha", "beta"] + [f"t{i}" for i in range(4)]
        long = "A showcase sentence that a reviewer can replay end to end on real material. " * 6
        for name in names[2:]:
            self.topic(name, mission(rows=[WORKSTREAM.format(id=f"{name.upper()}{i}", hours="80", status="open")
                                           .replace("Demo runs live", long) for i in range(1, 6)]))
        superseded = ["programme-three-20261111", "programme-four-20261110"]
        for cid in superseded:
            (self.home / "cohorts" / cid).mkdir()
        code, res = self.preview(topics=names, supersedes=superseded, run_alongside=["programme-two-20261101"])
        self.assertEqual(code, 0, res["errors"])
        self.assertEqual({len(c["order"]) for c in res["cards"]}, {2})
        self.assertEqual({c["clause_cap"] for c in res["cards"]}, {100})
        self.assertEqual([packet.default_count(n, 4) for n in range(1, 7)], [4, 4, 4, 3, 2, 2])

    def test_packet_limits_come_from_limits_json(self):
        limits = packet.load_limits()
        limits["packet"].update(topics_max=1, allowance_hours_max=6, ordered_workstreams_max=1)
        with mock.patch.object(packet, "load_limits", return_value=limits):
            for over, words in (({}, "1-1 topic names"), ({"topics": ["alpha"]}, "from 1 to 6"),
                                ({"topics": ["alpha"], "allowance_hours": 6, "order": {"alpha": ["A1", "A2"]}}, "1-1 distinct")):
                code, res = self.preview(**over)
                self.assertEqual(code, 2, over)
                self.assertIn(words, res["errors"][0]["message"])
            code, res = self.preview(topics=["alpha"], allowance_hours=6)
            self.assertEqual((code, [len(c["order"]) for c in res["cards"]]), (0, [1]))

    def test_build_writes_once_and_never_replaces_a_folder(self):
        code, res = self.build(carry_baseline_from=None, owner_model_tier="gpt-6-astra")
        self.assertEqual((code, res["written"], res["texts"]), (0, True, None))
        folder = self.home / "cohorts" / "programme-five-20261112"
        self.assertEqual(sorted(p.name for p in folder.iterdir()), ["FEEDBACK.md", "KERNEL.txt", "PACKET.json", "README.md", "STANCE-MESSAGE.txt"])
        meta = json.loads((folder / "PACKET.json").read_text())
        self.assertEqual(meta["schema"], "research-desk.packet/v1")
        self.assertEqual(meta["spec"]["supervisor_model"], "gpt-6-luna")
        self.assertEqual(meta["spec"]["owner_model_tier"], "gpt-6-astra")
        self.assertIn("`model` set to `gpt-6-astra`", (folder / "README.md").read_text())
        before = (folder / "README.md").read_bytes()
        code, res = self.build()
        self.assertEqual((code, res["commission_id"]), (0, "programme-five-20261112-2"))
        self.assertEqual((folder / "README.md").read_bytes(), before)
        self.assertFalse([p for p in (self.home / "cohorts").iterdir() if p.name.startswith(".")])

    def test_build_carries_the_baseline_of_a_superseded_commission(self):
        old = self.home / "cohorts" / "programme-five-20261111"
        old.mkdir()
        (old / "BASELINE.json").write_text('{"gpu": []}')
        code, res = self.build(supersedes=old.name, carry_baseline_from=old.name)
        self.assertEqual(code, 0, res["errors"])
        self.assertEqual((Path(res["folder"]) / "BASELINE.json").read_text(), '{"gpu": []}')
        self.assertIn("Supersedes `programme-five-20261111`; `BASELINE.json` carried",
                      (Path(res["folder"]) / "README.md").read_text())
        self.assertEqual(json.loads((Path(res["folder"]) / "PACKET.json").read_text())["spec"]["supersedes"], [old.name])

    def test_supersedes_names_every_listed_commission_and_refuses_unknown_ones(self):
        for cid in ("programme-three-20261111", "programme-four-20261110"):
            (self.home / "cohorts" / cid).mkdir()
        code, res = self.preview(supersedes=["programme-three-20261111", "programme-four-20261110"])
        self.assertEqual(code, 0, res["errors"])
        self.assertIn("Supersedes `programme-three-20261111` and `programme-four-20261110`.\n", res["texts"]["charter"])
        for bad in ("programme-two-20261101", ["programme-three-20261111", "programme-two-20261101"],
                    ["programme-three-20261111", "programme-three-20261111"], [7], {"id": "x"}):
            code, res = self.preview(supersedes=bad)
            self.assertEqual((code, res["errors"][0]["code"]), (2, "invalid_spec"), bad)

    def test_invalid_specs_exit_2_and_write_nothing(self):
        cases = {
            "unknown_topic": {"topics": ["alpha", "nope"]},
            "invalid_spec": {"allowance_hours": 0},
        }
        for code_name, over in cases.items():
            code, res = self.build(**over)
            self.assertEqual((code, res["errors"][0]["code"]), (2, code_name))
        outside = str(Path(__file__).resolve())
        for over in ({"stance_quotes": []}, {"colour": "red"}, {"read_first": {"alpha": "/no/such/file"}},
                     {"read_first": {"alpha": outside}}, {"read_first": {"alpha": str(self.home / "topics")}},
                     {"carry_baseline_from": "missing-20261101"}, {"topics": ["a"] * 7},
                     {"order": {"alpha": ["A3"]}}, {"order": {"alpha": ["Z9"]}}, {"order": {"gamma": ["A1"]}},
                     {"order": {"alpha": []}}, {"order": {"alpha": ["A1", "A1"]}},
                     {"order": {"beta": ["B2", "B4", "B5", "B2", "B4"]}},
                     {"card_notes": {"alpha": "## a heading"}}, {"card_notes": {"alpha": " "}},
                     {"card_notes": {"gamma": "x"}}, {"supervisor_model": "gpt-9"},
                     {"supervisor_effort": "ultra"}, {"owner_model_tier": "cheap"}):
            code, res = self.build(**over)
            self.assertEqual(code, 2, over)
        self.assertEqual(list((self.home / "cohorts").iterdir()), [])

    def test_a_long_stance_keeps_the_leading_quotes_and_points_to_the_charter(self):
        quotes = ["First words. " * 120, "Second words. " * 100, "Third words. " * 100]
        code, res = self.preview(stance_quotes=quotes)
        self.assertEqual(code, 0, res["errors"])
        stance, charter = res["texts"]["stance"], res["texts"]["charter"]
        self.assertLessEqual(len(stance), 3500)
        self.assertIn(quotes[0].strip(), stance)
        self.assertNotIn("Third words.", stance)
        self.assertRegex(stance, r"\(\d more of my quotes, verbatim, are in the charter under Operator stance")
        for q in quotes:
            self.assertIn(q.strip(), charter)

    def test_a_charter_over_its_gate_carries_fewer_unordered_workstreams_per_card(self):
        long_row = "| {id} | Land it | Dim A | 40 h | " + "A long showcase sentence. " * 60 + " | open |"
        for name, prefix in (("gamma", "G"), ("delta", "D")):
            self.topic(name, mission(rows=[long_row.format(id=f"{prefix}{i}") for i in range(1, 5)]))
        code, res = self.preview(topics=["gamma", "delta"])
        self.assertEqual(code, 0, res["errors"])
        self.assertLessEqual(len(res["texts"]["charter"].encode()), packet.load_limits()["gates"]["charter_bytes"])
        self.assertTrue(all(1 < len(c["order"]) < 4 for c in res["cards"]), [c["order"] for c in res["cards"]])

    def test_a_failed_stance_gate_exits_1_and_writes_nothing(self):
        code, res = self.build(stance_quotes=["word " * 800])
        self.assertEqual(code, 1)
        failed = [g["name"] for g in res["gates"] if not g["ok"]]
        self.assertEqual(failed, ["stance_chars"])
        self.assertEqual(list((self.home / "cohorts").iterdir()), [])

    def test_a_charter_over_its_size_gate_warns_what_to_cut_and_is_still_built(self):
        code, res = self.build(card_notes={"beta": "Long operator note. " * 700})
        self.assertEqual(code, 0, res["errors"])
        self.assertEqual(res["errors"], [])
        gate = f'{packet.load_limits()["gates"]["charter_bytes"]:,}'
        overflow = [w["message"] for w in res["warnings"] if w["code"] == "gate_overflow"]
        self.assertEqual(len(overflow), 1, res["warnings"])
        self.assertRegex(overflow[0], rf"^charter is [\d,]+ bytes, [\d,]+ over {gate}; largest parts: Card 2 — Beta things \(notes 14,\d{{3}}\)")
        self.assertNotIn("YTDLP", overflow[0])
        self.assertEqual([g["name"] for g in res["gates"] if not g["ok"]], ["charter_bytes"])
        self.assertEqual(len(list((self.home / "cohorts").iterdir())), 1)

    def test_mission_errors_exit_2(self):
        self.topic("thin", mission(rows=[WORKSTREAM.format(id="T1", hours="40", status="landed")]))
        code, res = self.preview(topics=["thin"])
        self.assertEqual(code, 2)
        self.assertTrue(any(e["code"] == "mission_error" for e in res["errors"]))

    def test_the_topic_plan_orders_the_card_and_names_each_brief(self):
        work = self.home / "topics" / "alpha" / "work"
        (work / "A1-api").mkdir(parents=True)
        (work / "A1-api" / "BRIEF.md").write_text("brief")
        (work / "README.md").write_text("# Plan\n\n| Brief | Workstream | State | Note |\n|---|---|---|---|\n"
                                        "| `A3-old` | A3 | landed | |\n| `A4-next` | A4 | planned | |\n"
                                        "| [A1-api](A1-api/BRIEF.md) | A1 | ready | mentions A2 |\n\n## Risks\n\n| A2 risk | | | | | |\n")
        code, res = self.preview()
        self.assertEqual(code, 0, res["errors"])
        alpha, beta = res["cards"]
        self.assertEqual((alpha["order"], alpha["order_source"]), (["A4", "A1"], "plan"))  # landed and note-only rows skipped
        self.assertEqual(beta["order_source"], "mission")
        charter = res["texts"]["charter"]
        self.assertIn(packet.PLAN_LABEL, charter)
        self.assertIn("Brief: `work/A1-api/BRIEF.md`.", charter)  # relative to the card's Folder

    def test_a_retired_topic_is_refused(self):
        self.topic("done", mission().replace("Mode: build", "Mode: build\nRetired: 2026-11-15 — landed", 1))
        code, res = self.preview(topics=["done"])
        self.assertEqual(code, 2)
        self.assertIn("done: retired 2026-11-15 — landed; reopen it before commissioning", [e["message"] for e in res["errors"]])

    def test_unknown_or_unfilled_placeholders_are_errors(self):
        with self.assertRaises(packet.Fail) as ctx:
            packet.render("{{nope}}", {})
        self.assertEqual(ctx.exception.code, "unknown_placeholder")
        with self.assertRaises(packet.Fail) as ctx:
            packet.render("{{x}}", {"x": None})
        self.assertEqual(ctx.exception.code, "unfilled_placeholder")

    def test_kernel_excerpt_fits_the_budget(self):
        self.assertEqual(packet.kernel_excerpt("Short words.", 50), "Short words")
        self.assertEqual(packet.kernel_excerpt("A long first sentence here. Tiny.", 12), "Tiny")
        cut = packet.kernel_excerpt("one two three four five six seven", 15)
        self.assertTrue(len(cut) <= 15 and cut.endswith("..."))

    def test_check_gates_built_older_and_missing_folders(self):
        _, built = self.build()
        code, res = self.run_cli("check", "--home", str(self.home), built["commission_id"])
        self.assertEqual(code, 0, res["errors"])
        self.assertEqual([c["order"] for c in res["cards"]], [["A2", "A4", "A1"], ["B5", "B2", "B4"]])
        self.assertEqual({c["order_source"] for c in res["cards"]}, {"mission"})
        _, built = self.build(order={"alpha": ["A4"]}, card_notes={"alpha": "Then A1 and A2, in that order."})
        _, res = self.run_cli("check", "--home", str(self.home), built["commission_id"])
        self.assertEqual((res["cards"][0]["order"], res["cards"][0]["order_source"]), (["A4"], "operator"))
        older = self.home / "cohorts" / "night-20261030"
        older.mkdir()
        (older / "README.md").write_text(f"# Old\n\nSee {self.home}/topics/alpha/MISSION.md and {self.home}/gone.md\n" + "x" * 30000)
        code, res = self.run_cli("check", "--home", str(self.home), older.name)
        self.assertEqual(code, 1)
        failed = {g["name"] for g in res["gates"] if not g["ok"]}
        self.assertEqual(failed, {"charter_bytes", "stance_chars", "stance_ending", "kernel_chars", "paths_exist"})
        self.assertEqual([g["name"] for g in res["gates"] if g.get("advisory")], ["charter_bytes"])
        self.assertEqual(res["cards"][0]["topic"], "alpha")
        code, res = self.run_cli("check", "--home", str(self.home), "absent-20261101")
        self.assertEqual((code, res["errors"][0]["code"]), (3, "not_found"))

    def test_an_answer_waiting_in_a_card_topic_is_named_in_the_charter_start_step(self):
        _, quiet = self.preview()
        self.assertIn("none wait at build", quiet["texts"]["charter"])
        stamp = "2026-11-15T09:00Z"
        brief = self.home / "topics" / "alpha" / "work" / "A9"
        brief.mkdir(parents=True)
        (brief / "BRIEF.md").write_text("x")
        (self.home / "topics" / "alpha" / "INBOX.md").write_text(
            f"### {stamp} · from operator · answer\nA9: keep — Keep.\nPacket: {brief / 'BRIEF.md'}\nSource: hub ask a1\n")
        _, res = self.preview()
        start = res["texts"]["charter"].split("1. **Start.**", 1)[1].split("2. **The unit loop**", 1)[0]
        self.assertRegex(start, r"start its executor now \(`spawn <ID>`, above\), before any owner and not at a review \(1 wait at build: A9 \(alpha, [\d.]+ h\)\)")

    def test_the_charter_opens_with_a_loop_card_and_a_measured_pace_line(self):
        _, res = self.preview()
        top = res["texts"]["charter"].split("\n\n", 2)[1].splitlines()  # after the title
        self.assertEqual([l.split(":")[0] for l in top], ["Setpoint", "Sensors", "Actuators", "Manager", "Tooling"])
        self.assertIn("Last 10 days: 0 landed on these topics", top[0])  # B1 landed 2026-11-01, 11 days before the fixed today
        self.assertIn("Manager:   portfolio steward. Last blocker removed:", top[3])
        rows = [WORKSTREAM.format(id=f"A{i}", hours="40", status=st) for i, st in
                enumerate(["landed 2026-11-10", "landed 2026-11-12", "open", "open"], 1)]
        (self.home / "topics" / "alpha" / "MISSION.md").write_text(mission(rows=rows).replace("# T — mission", "# Alpha things — mission"))
        _, res = self.preview()
        self.assertIn("Last 10 days: 2 landed on these topics", res["texts"]["charter"])
        self.assertIn("`Goal: <status>`", res["texts"]["charter"])

    def test_skill_files_stay_within_recorded_ceilings(self):
        limits = packet.load_limits()
        skills = packet.SKILL.parent
        for rel, ceiling in limits["ceilings"]["skill_bytes"].items():
            self.assertLessEqual((skills / rel).stat().st_size, ceiling, rel)

    def test_the_model_check_rests_on_the_launch_command_and_still_starts_owners(self):
        _, res = self.preview()
        start = res["texts"]["charter"].split("1. **Start.**", 1)[1].split("2. **The unit loop**", 1)[0]
        self.assertIn("never from your self-description", start)
        self.assertIn("No output: record `unknown`, no item.", start)
        self.assertIn("Neither stops the start.", start)
        self.assertIn("owners inherit this session's model", start)
        self.assertNotIn("start no owner", start)
        probe = re.search(r"^   `(p=\$\$; .+)`$", start, re.MULTILINE).group(1)
        # Model a process tree ending at PID 1 without a Codex launch, regardless of the
        # test runner's own ancestors. Keep grep and the shell real.
        fake_ps = self.home / "ps"
        fake_ps.write_text('#!/bin/sh\nif [ "$1" = "-o" ]; then echo 1; else echo "sh -c probe"; fi\n')
        fake_ps.chmod(0o755)
        env = {**os.environ, "PATH": f"{self.home}:{os.defpath}"}
        plain = subprocess.run(["sh", "-c", probe], capture_output=True, text=True, timeout=10, env=env)
        self.assertEqual(plain.stdout, "")
        # Under a process launched as `codex ... --model X`, the probe prints that launch command.
        codex = subprocess.run(["codex", "-c", probe, "--model", "gpt-test"], executable="/bin/sh",
                               capture_output=True, text=True, timeout=10)
        self.assertRegex(codex.stdout, r"^codex .*--model gpt-test$")


class NeedsYouGrammarTest(unittest.TestCase):
    """The grammar is stated where each reader looks: the reference (an agent without the desk),
    the charter template (the supervisor) and the file contract (the desk). They must agree."""

    def test_every_copy_names_the_same_heading_and_lines(self):
        contract = (packet.REPO / "scenarios/research-desk/docs/reference/FILE-CONTRACT.md").read_text(encoding="utf-8")
        # The charter keeps no copy of its own: it sends the scheduler to operator-attention.md.
        self.assertIn("Grammar, packet and page: `{{operator_attention}}`",
                      (packet.TEMPLATES / "charter.md").read_text(encoding="utf-8"))
        copies = {
            "operator-attention.md": (packet.SKILL / "references/operator-attention.md").read_text(encoding="utf-8"),
            "FILE-CONTRACT.md §5": contract[contract.index("## 5. Decisions"):],
        }
        for name, text in copies.items():
            for needle in ("[A-Z0-9][A-Z0-9_-]{0,127}", "[a-z0-9-]{1,32}"):
                self.assertIn(needle, text, name)
            self.assertRegex(text, r"(?m)^### [A-Z0-9][A-Z0-9_-]{0,127} — \S", name)
            for line in ("Recommended", "Options", "Packet", "Secret", "Commission"):
                self.assertRegex(text, rf"(?m)(^|`){line}: \S", f"{name} lacks {line}:")



class ReviewBriefsTest(unittest.TestCase):
    def test_review_briefs_are_verbatim_copies_of_long_effort_design(self):
        doc = (packet.REPO / "docs/agent-system/LONG_EFFORT_DESIGN.md").read_text(encoding="utf-8")
        section = doc[doc.index("### 2a."):doc.index("## 3.")]
        blocks = re.findall(r"```text\n(.*?)\n```", section, re.S)
        copy = packet.PATHS["review_briefs"].read_text(encoding="utf-8")
        self.assertEqual(len(blocks), 3)
        self.assertEqual(re.findall(r"```text\n(.*?)\n```", copy, re.S), blocks)

    def test_owners_follow_listed_workstreams_up_to_the_cap(self):
        self.assertEqual(packet.owner_split([2]), [2])          # one topic, two whole workstreams: two owners
        self.assertEqual(packet.owner_split([4, 1]), [4, 1])
        self.assertEqual(packet.owner_split([4, 4]), [3, 2])    # capped at OWNERS_MAX
        self.assertEqual(packet.owner_split([0, 0, 0, 0, 0, 0]), [1] * 6)  # never fewer than one per card

if __name__ == "__main__":
    unittest.main()
