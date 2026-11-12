#!/usr/bin/env python3
"""Tests for packet.py. Run: python3 -m unittest test_packet (from this directory)."""

from __future__ import annotations

import contextlib
import datetime as dt
import io
import json
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import packet  # noqa: E402
from test_mission_check import WORKSTREAM, mission  # noqa: E402

CONTRACT_LINES = (
    "State: see STATE.md beside this file.",
    "Read entries of `OPERATOR.md` beside this file that are newer than the last one you acted on, at every unit start; they are the operator's words and win over this charter",
    "Write each decision as a needs-you item in `STATE.md` (grammar in the research-coordination skill); research-desk delivers it, or you do by the fallback below. This overrides research-coordination step 9 and LONG_EFFORT_DESIGN D17 on delivery.",
    "### <UTC> · delivered <ID> · via desk",
    "`Delivered: <route> <UTC>`",
    "`Commission: programme-five-20261112`",
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

    def test_without_an_order_cards_take_active_approved_then_open_and_say_it_is_not_priority(self):
        _, res = self.preview()
        self.assertEqual(res["cards"][0], {"topic": "alpha", "mode": "build", "order": ["A2", "A4", "A1"],
                                           "order_source": "mission", "read_first": None})
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
        self.assertEqual([packet.default_count(n) for n in range(1, 7)], [4, 4, 4, 3, 2, 2])

    def test_build_writes_once_and_never_replaces_a_folder(self):
        code, res = self.build(carry_baseline_from=None, owner_model_tier="gpt-6-astra")
        self.assertEqual((code, res["written"], res["texts"]), (0, True, None))
        folder = self.home / "cohorts" / "programme-five-20261112"
        self.assertEqual(sorted(p.name for p in folder.iterdir()), ["KERNEL.txt", "PACKET.json", "README.md", "STANCE-MESSAGE.txt"])
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

    def test_a_failed_gate_exits_1_writes_nothing_and_says_what_to_cut(self):
        code, res = self.build(stance_quotes=["word " * 800])
        self.assertEqual(code, 1)
        failed = [g["name"] for g in res["gates"] if not g["ok"]]
        self.assertEqual(failed, ["stance_chars"])
        self.assertEqual(list((self.home / "cohorts").iterdir()), [])
        code, res = self.build(card_notes={"beta": "Long operator note. " * 600})
        self.assertEqual(code, 1)
        overflow = [e["message"] for e in res["errors"] if e["code"] == "gate_overflow"]
        self.assertEqual(len(overflow), 1, res["errors"])
        self.assertRegex(overflow[0], r"^charter is [\d,]+ bytes, [\d,]+ over 25,000; largest parts: Card 2 — Beta things \(notes 12,\d{3}\)")
        self.assertNotIn("YTDLP", overflow[0])
        self.assertEqual(list((self.home / "cohorts").iterdir()), [])

    def test_mission_errors_exit_2(self):
        self.topic("thin", mission(rows=[WORKSTREAM.format(id="T1", hours="40", status="open")]))
        code, res = self.preview(topics=["thin"])
        self.assertEqual(code, 2)
        self.assertTrue(any(e["code"] == "mission_error" for e in res["errors"]))

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
        self.assertEqual(res["cards"][0]["topic"], "alpha")
        code, res = self.run_cli("check", "--home", str(self.home), "absent-20261101")
        self.assertEqual((code, res["errors"][0]["code"]), (3, "not_found"))

    def test_skill_files_stay_within_recorded_ceilings(self):
        limits = packet.load_limits()
        skills = packet.SKILL.parent
        for rel, ceiling in limits["ceilings"]["skill_bytes"].items():
            self.assertLessEqual((skills / rel).stat().st_size, ceiling, rel)

    def test_the_model_mismatch_rule_still_starts_owners_with_a_named_model(self):
        _, res = self.preview()
        start = res["texts"]["charter"].split("1. **Start.**", 1)[1].split("2. **The unit loop**", 1)[0]
        self.assertIn("still start the owners, with `model` set to\n   `gpt-6-luna`", start)
        self.assertNotIn("start no owner", start)


if __name__ == "__main__":
    unittest.main()
