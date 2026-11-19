import json
import tempfile
import unittest
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
import research_report as report
import evidence_record


class ReportTest(unittest.TestCase):
    def test_sources_preserve_evidence_and_unknowns(self):
        with tempfile.TemporaryDirectory() as tmp:
            home = Path(tmp)
            topic = home / "topics" / "example"
            (topic / "work").mkdir(parents=True)
            (topic / "feedback").mkdir()
            run = topic / "runs" / "c1"
            run.mkdir(parents=True)
            (topic / "MISSION.md").write_text("## Scorecard\n\n| Dimension | Today | Target |\n|---|---|---|\n| Quality | unmeasured | measured |\n")
            (topic / "work/README.md").write_text("Work state: waiting")
            (topic / "feedback/a.json").write_text(json.dumps({"note": "try this"}))
            (topic / "reporting").mkdir()
            (topic / "reporting/report.json").write_text(json.dumps({"sections": ["evidence"], "viewers": ["image"]}))
            (run / "CHECKPOINT.md").write_text("No measurement yet")
            (run / "README.md").write_text("Run narrative")
            (run / "evidence").mkdir()
            (run / "evidence" / "detail.md").write_text("not a checkpoint")
            result = report.collect(topic, "c1")
            self.assertEqual(result["scorecard"], [{"dimension": "Quality", "today": "unmeasured", "target": "measured"}])
            self.assertEqual(result["sources"]["ledger"]["status"], "unavailable")
            self.assertIsNone(result["sources"]["ledger"]["text"])
            self.assertIn("try this", result["sources"]["feedback"]["items"][0]["text"])
            self.assertEqual(result["sources"]["checkpoints"]["items"][0]["text"], "No measurement yet")
            self.assertEqual({p["text"] for p in result["sources"]["run_records"]["items"]}, {"No measurement yet", "Run narrative"})
            self.assertEqual(result["sources"]["commission_state"]["status"], "unavailable")
            self.assertEqual(result["report_config"]["sections"], ["evidence"])

    def test_evidence_record_helper_writes_and_history_is_immutable(self):
        record = {"id": "cmp-1", "title": "A versus B", "format": "text", "decidedBy": "agent", "workstream": "LM2",
                  "sides": {"a": {"label": "A", "text": "old"}, "b": {"label": "B", "text": "new"}},
                  "verdict": {"text": "B is clearer."}, "method": ["Same prompt"], "limits": []}
        with tempfile.TemporaryDirectory() as tmp:
            run = Path(tmp) / "run"
            path = evidence_record.write(run, record)
            self.assertEqual(json.loads(path.read_text()), record)
            topic = Path(tmp) / "topic"
            when = __import__("datetime").datetime(2026, 11, 17, tzinfo=__import__("datetime").timezone.utc)
            history = report.write_history(topic, "c1", {"evidence": [record]}, {"ask_id": "a1"}, {"answer": "ok"}, when)
            self.assertEqual(json.loads((history / "receipt.json").read_text())["ask_id"], "a1")
            self.assertEqual(json.loads((topic / "reporting/latest.json").read_text())["path"], str(history.relative_to(topic)))

    def test_historical_evidence_without_workstream_still_validates_but_new_write_requires_it(self):
        legacy = {"id": "old-record", "title": "Old comparison", "format": "text", "decidedBy": "agent",
                  "sides": {"a": {"label": "A", "text": "old"}, "b": {"label": "B", "text": "new"}},
                  "verdict": {"text": "B is clearer."}, "method": [], "limits": []}
        evidence_record.validate(legacy)
        with tempfile.TemporaryDirectory() as tmp:
            with self.assertRaisesRegex(ValueError, "require a workstream"):
                evidence_record.write(Path(tmp), legacy)
            legacy["workstream"] = "LM2"
            target = evidence_record.write(Path(tmp), legacy)
            self.assertEqual(json.loads(target.read_text())["workstream"], "LM2")

    def test_supervisor_must_judge_every_dimension_once(self):
        scorecard = [{"dimension": "A"}, {"dimension": "B"}]
        good = {"scorecard": [{"dimension": "A", "level": "early", "reason": "No live proof"}, {"dimension": "B", "level": "unavailable", "reason": "No measurement"}]}
        report.validate_review(scorecard, good)
        for bad in ({"scorecard": good["scorecard"][:1]}, {"scorecard": [good["scorecard"][0]] * 2}, {"scorecard": [{"dimension": "A", "level": "early", "reason": ""}, good["scorecard"][1]]}):
            with self.assertRaises(ValueError):
                report.validate_review(scorecard, bad)

    def test_commission_cannot_escape_runs_directory(self):
        with tempfile.TemporaryDirectory() as tmp:
            with self.assertRaises(ValueError):
                report.collect(Path(tmp), "../other")

    def test_build_uses_plan_and_qualitative_judgment(self):
        data = {"topic": "example", "commission": "c1", "scorecard": [{"dimension": "Quality", "today": "Unmeasured", "target": "Measured"}],
                "sources": {"work_plan": {"text": "| Brief | Workstream | State | Note |\n|---|---|---|---|\n| [A](A/BRIEF.md) | Quality work | waiting-time | No measurement yet |\n"}}}
        result = report.build(data, {"meta": {"title": "Work is waiting"}, "scorecard": [{"dimension": "Quality", "level": "Early", "reason": "No measurement yet"}]})
        self.assertEqual(result["workstreams"]["items"][0]["state"], "waiting-time")
        self.assertEqual(result["scorecard"]["dimensions"][0]["reason"], "No measurement yet")
        self.assertNotIn("now", result["scorecard"]["dimensions"][0])
        self.assertNotIn("weight", result["workstreams"]["items"][0])
        self.assertEqual(result["measurements"]["source"]["status"], "unavailable")
        self.assertEqual(result["feedback"]["source"]["status"], "unavailable")
        self.assertEqual(result["next"]["source"]["status"], "unavailable")
        self.assertEqual(result["states"][0]["label"], "Waiting on time")
        self.assertEqual(result["states"][0]["tone"], "warn")
        with_ask = report.build(data, {"meta": {"title": "Waiting"}, "scorecard": [{"dimension": "Quality", "level": "Early", "reason": "Unmeasured"}],
                                     "asks": [{"id": "ask-1", "title": "Choose next target"}]})
        self.assertEqual(with_ask["asks"][0]["id"], "ask-1")
        with self.assertRaisesRegex(ValueError, "source status"):
            report.build(data, {"meta": {"title": "Waiting"}, "scorecard": [{"dimension": "Quality", "level": "Early", "reason": "Unmeasured"}], "measurements": {"items": []}})

    def test_current_schema_accepts_a_bounded_review_and_reports_precise_failures(self):
        with tempfile.TemporaryDirectory() as tmp:
            topic = Path(tmp) / "local-model-performance"
            (topic / "work").mkdir(parents=True)
            (topic / "MISSION.md").write_text("## Scorecard\n\n| Dimension | Today | Target |\n|---|---|---|\n| Runtime | Measured | Better |\n")
            (topic / "work/README.md").write_text("# Plan\n")
            data = report.collect(topic, "dry-build")
            review = {"meta": {"title": "Bounded current-schema review", "lede": "One real comparison is recorded; wider claims remain open."},
                      "scorecard": [{"dimension": "Runtime", "level": "early", "reason": "One workload is measured."}],
                      "measurements": {"source": {"status": "real", "label": "retained run evidence"},
                                       "items": [{"title": "GPU runtime", "unit": "s", "values": [{"label": "CPU", "value": 5.3}, {"label": "GPU", "value": .1}],
                                                  "better": "lower", "verdict": "good", "note": "One workload; not a fleet result."}]},
                      "feedback": {"source": {"status": "partial", "label": "operator answer"},
                                   "items": [{"said": "Improve measured runtime.", "changed": "Added GPU runtime comparison.", "status": "partly", "why": "Fleet-wide proof remains open."}]}}
            payload = report.build(data, review)
            report.validate_payload(payload)
            invalid = {**payload, "meta": {**payload["meta"], "lede": "x" * 141}}
            with self.assertRaisesRegex(ValueError, r"schema violation at \$\.meta\.lede"):
                report.validate_payload(invalid)

    def test_short_rejects_overlong_lines_instead_of_ellipsis_truncation(self):
        with self.assertRaisesRegex(ValueError, "maximum is 8"):
            report.short("A complete sentence that is too long.", 8)

    def test_lede_speedup_must_match_plotted_time_median(self):
        valid = {"meta": {"lede": "GPU ran 46.4× faster."},
                 "measurements": {"items": [{"unit": "ms", "values": [{"value": 2831}, {"value": 61}]}]}}
        report.validate_lede_speedups(valid)
        invalid = {**valid, "meta": {"lede": "GPU ran 54.9× faster."}}
        with self.assertRaisesRegex(ValueError, "do not match plotted time medians"):
            report.validate_lede_speedups(invalid)

    def test_evidence_workstream_and_ask_unblocks_are_written(self):
        record = {"id": "cmp-1", "title": "A versus B", "format": "text", "decidedBy": "agent",
                  "sides": {"a": {"label": "A", "text": "old"}, "b": {"label": "B", "text": "new"}},
                  "verdict": {"text": "B is clearer."}, "method": [], "limits": []}
        data = {"topic": "example", "commission": "c1", "scorecard": [{"dimension": "Quality", "today": "Measured", "target": "Better"}],
                "evidence": [record], "sources": {"work_plan": {"text": ""}}}
        review = {"meta": {"title": "Short review"}, "scorecard": [{"dimension": "Quality", "level": "early", "reason": "One test."}],
                  "evidenceWorkstreams": {"cmp-1": "LM2"},
                  "asks": [{"id": "ask-1", "title": "Choose next step", "workstream": "LM2"}]}
        result = report.build(data, review)
        self.assertEqual(result["evidence"][0]["workstream"], "LM2")
        self.assertEqual(result["asks"][0]["unblocks"], "Unblocks LM2 work")
        review.pop("evidenceWorkstreams")
        with self.assertRaisesRegex(ValueError, "requires a workstream mapping"):
            report.build(data, review)

    def test_measure_evidence_facts_are_human_rounded_and_match_plotted_values(self):
        record = {"id": "runtime", "title": "Runtime", "format": "measure", "decidedBy": "automated",
                  "sides": {"a": {"label": "CPU"}, "b": {"label": "GPU"}},
                  "items": [{"label": "Median", "a": 2831, "b": 61, "unit": "ms"}],
                  "facts": [{"label": "Speedup from warm medians", "a": 1, "b": 54.8688},
                            {"label": "Mask IoU", "a": [1, 1, 1], "b": [1, 1, 1]}],
                  "verdict": {"text": "GPU warm forward was 54.9× faster; all three CPU/CUDA masks had IoU 1.0."},
                  "method": [], "limits": []}
        data = {"topic": "example", "commission": "c1", "scorecard": [{"dimension": "Quality", "today": "Measured", "target": "Better"}],
                "evidence": [record], "sources": {"work_plan": {"text": ""}}}
        review = {"meta": {"title": "Short review"}, "scorecard": [{"dimension": "Quality", "level": "early", "reason": "One test."}],
                  "evidenceWorkstreams": {"runtime": "LM2"}}
        result = report.build(data, review)["evidence"][0]
        self.assertEqual(result["verdict"]["headline"], "46.4× faster")
        self.assertNotIn("faster", result["verdict"]["text"])
        self.assertIn("all three CPU/CUDA masks had IoU 1.0", result["verdict"]["text"])
        self.assertEqual(result["facts"], [{"label": "Speedup from warm medians", "a": "1.0×", "b": "46.4×", "win": "b"}])


if __name__ == "__main__":
    unittest.main()
