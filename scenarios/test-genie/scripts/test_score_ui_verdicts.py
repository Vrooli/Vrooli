import unittest
import json
import tempfile
from unittest.mock import patch
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
from score_ui_verdicts import independence_audit, independent_case_failures, index_corpus, main, score


def case(case_id, truth="defect", phase="experience"):
    return {"case_id": case_id, "phase": phase, "truth": truth, "evidence": "source-backed fixture"}


def prediction(case_id, verdict="passed", phase="experience"):
    return {"case_id": case_id, "phase": phase, "verdict": verdict}


class ScoreTests(unittest.TestCase):
    def test_independence_audit_collapses_claims_to_snapshot_clusters(self):
        review = {"method": "blind-independent", "phase_blinded": True, "reviewer_count": 2, "rubric_ref": "rubrics/ui-v1.md"}
        first = {
            **case("claim-a", "clean"),
            "scenario": "scenario-a",
            "build_identity": "build-1",
            "route": "/home",
            "state": "default",
            "viewport": "390x844",
            "snapshot_sha256": "snapshot-a",
            "adjudication": review,
        }
        second = {**first, "case_id": "claim-b", "rule_code": "contrast"}
        third = {**first, "case_id": "claim-c", "snapshot_sha256": "snapshot-b"}
        audit = independence_audit([first, second, third])["experience"]
        self.assertEqual(audit["claims"], 3)
        self.assertEqual(audit["snapshot_clusters"], 2)
        self.assertEqual(audit["scenario_clusters"], 1)
        self.assertEqual(audit["independently_adjudicated_clusters"], 2)
        self.assertEqual(audit["ineligible_claims"], 0)

    def test_independent_case_minimum_fails_closed_for_missing_phase_or_labels(self):
        self.assertEqual(independent_case_failures({}, ["experience", "workflow"], 50), ["experience", "workflow"])
        self.assertEqual(
            independent_case_failures({"experience": {"independently_adjudicated_clusters": 49}}, ["experience"], 50),
            ["experience"],
        )
        self.assertEqual(
            independent_case_failures({"experience": {"independently_adjudicated_clusters": 50}}, ["experience"], 50),
            [],
        )

    def test_cli_minimum_gate_returns_nonzero_and_writes_failure_report(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            corpus_path, before_path, after_path, output_path = (root / name for name in ("corpus.jsonl", "before.jsonl", "after.jsonl", "report.md"))
            rows = [case("a", "clean")]
            predictions = [prediction("a", "passed")]
            corpus_path.write_text("".join(json.dumps(row) + "\n" for row in rows), encoding="utf-8")
            for path in (before_path, after_path):
                path.write_text("".join(json.dumps(row) + "\n" for row in predictions), encoding="utf-8")
            argv = ["score_ui_verdicts.py", "--corpus", str(corpus_path), "--before", str(before_path), "--after", str(after_path), "--output", str(output_path), "--phases", "experience", "--minimum-independent-cases", "50"]
            with patch("sys.argv", argv):
                self.assertEqual(main(), 2)
            self.assertIn("Independent-case gate: FAIL", output_path.read_text(encoding="utf-8"))

    def test_not_evaluated_on_defect_is_false_negative_and_coverage_loss(self):
        result = score([case("bad")], [prediction("bad", "not-evaluated")])["experience"]["counts"]
        self.assertEqual(result["false_negatives"], 1)
        self.assertEqual(result["not_evaluated"], 1)
        self.assertEqual(result["evaluated"], 0)

    def test_failed_clean_case_is_false_positive(self):
        result = score([case("clean", "clean")], [prediction("clean", "failed")])["experience"]["counts"]
        self.assertEqual(result["false_positives"], 1)
        self.assertEqual(result["clean"], 1)

    def test_missing_predictions_are_rejected(self):
        with self.assertRaisesRegex(ValueError, "missing predictions"):
            score([case("bad")], [])

    def test_phase_mismatch_is_rejected(self):
        with self.assertRaisesRegex(ValueError, "phase does not match"):
            score([case("bad")], [prediction("bad", phase="workflow")])

    def test_duplicate_corpus_id_is_rejected(self):
        with self.assertRaisesRegex(ValueError, "duplicate corpus"):
            index_corpus([case("same"), case("same", phase="workflow")])

    def test_scores_are_broken_out_by_rule(self):
        rows = [case("a"), {**case("b"), "case_id": "b", "rule_code": "overflow"}]
        result = score(rows, [prediction("a", "passed"), prediction("b", "failed")])
        self.assertEqual(result["experience"]["rules"]["unspecified"]["cases"], 1)
        self.assertEqual(result["experience"]["rules"]["overflow"]["false_positives"], 0)


if __name__ == "__main__":
    unittest.main()
