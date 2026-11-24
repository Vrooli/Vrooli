import unittest
import json
import hashlib
import subprocess
import tempfile
from unittest.mock import patch
from pathlib import Path
import sys

sys.path.insert(0, str(Path(__file__).resolve().parent))
from score_ui_verdicts import (
    independence_audit,
    independent_case_failures,
    index_corpus,
    main,
    score,
    validate_assembly_receipt,
    validate_evaluator_manifest,
)


def case(case_id, truth="defect", phase="experience"):
    return {"case_id": case_id, "phase": phase, "truth": truth, "evidence": "source-backed fixture"}


def prediction(case_id, verdict="passed", phase="experience"):
    return {"case_id": case_id, "phase": phase, "verdict": verdict}


def eligible_case(case_id, truth="defect", phase="experience", snapshot=None):
    row = {
        **case(case_id, truth, phase),
        "scenario": "demo",
        "build_identity": "subject-build-1",
        "route": "/home",
        "state": "default",
        "viewport": "390x844",
        "snapshot_sha256": snapshot or hashlib.sha256(case_id.encode()).hexdigest(),
        "adjudication": {
            "method": "blind-independent",
            "phase_blinded": True,
            "reviewer_count": 2,
            "rubric_ref": "rubrics/ui-v1.md",
        },
    }
    return row


def proven_prediction(case_row, verdict="passed", evaluator="ui-health", evaluator_build="eval-build-1"):
    return {
        "case_id": case_row["case_id"],
        "phase": case_row["phase"],
        "verdict": verdict,
        "scenario": case_row["scenario"],
        "subject_build_identity": case_row["build_identity"],
        "route": case_row["route"],
        "state": case_row["state"],
        "viewport": case_row["viewport"],
        "snapshot_sha256": case_row["snapshot_sha256"],
        "evaluator": evaluator,
        "evaluator_build_identity": evaluator_build,
        "evaluator_source_sha256": "f" * 64,
        "command": "test-genie replay --phase experience",
        "execution_mode": "replay",
    }


def evaluator_manifest(phase="experience", evaluator="ui-health", build="eval-build-1"):
    return {
        "phases": {
            phase: {
                "evaluator": evaluator,
                "build_identity": build,
                "command": "test-genie replay --phase experience",
                "execution_mode": "replay",
                "source_sha256": "f" * 64,
            }
        }
    }


def write_rows(path, rows):
    path.write_text("".join(json.dumps(row) + "\n" for row in rows), encoding="utf-8")


def write_assembly_receipt(path, corpus_path, count):
    path.write_text(json.dumps({
        "schema": "ui-verdict-corpus-assembly/v1",
        "rubric_ref": "rubrics/ui-v1.md",
        "case_packet_sha256": "a" * 64,
        "reviewer_a_sha256": "b" * 64,
        "reviewer_b_sha256": "c" * 64,
        "adjudicator_sha256": None,
        "input_cases": count,
        "scored_cases": count,
        "excluded_insufficient_cases": 0,
        "scored_corpus_sha256": hashlib.sha256(corpus_path.read_bytes()).hexdigest(),
        "excluded_audit_sha256": "d" * 64,
    }), encoding="utf-8")


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
            "snapshot_sha256": "a" * 64,
            "adjudication": review,
        }
        second = {**first, "case_id": "claim-b", "rule_code": "contrast"}
        third = {**first, "case_id": "claim-c", "snapshot_sha256": "b" * 64}
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
            corpus_path, before_path, after_path, receipt_path, before_manifest_path, after_manifest_path, output_path, json_path = (root / name for name in ("corpus.jsonl", "before.jsonl", "after.jsonl", "receipt.json", "before.json", "after.json", "report.md", "report.json"))
            rows = [eligible_case("a", "clean")]
            predictions = [proven_prediction(rows[0], "passed")]
            write_rows(corpus_path, rows)
            write_assembly_receipt(receipt_path, corpus_path, len(rows))
            write_rows(before_path, predictions)
            write_rows(after_path, predictions)
            before_manifest_path.write_text(json.dumps(evaluator_manifest()), encoding="utf-8")
            after_manifest_path.write_text(json.dumps(evaluator_manifest()), encoding="utf-8")
            argv = ["score_ui_verdicts.py", "--corpus", str(corpus_path), "--before", str(before_path), "--after", str(after_path), "--assembly-receipt", str(receipt_path), "--output", str(output_path), "--json-output", str(json_path), "--phases", "experience", "--before-evaluator", str(before_manifest_path), "--after-evaluator", str(after_manifest_path)]
            with patch("sys.argv", argv):
                self.assertEqual(main(), 2)
            markdown = output_path.read_text(encoding="utf-8")
            self.assertIn("insufficient-eligible-cases", markdown)
            self.assertIn("withheld", markdown)
            payload = json.loads(json_path.read_text(encoding="utf-8"))
            self.assertEqual(payload["status"], "insufficient-eligible-cases")
            self.assertIsNone(payload["phase_results"]["experience"]["rates"])

    def test_operator_artifact_invocation_is_discoverable_and_fails_closed(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            corpus_path, before_path, after_path, receipt_path, before_manifest_path, after_manifest_path = (
                root / name for name in ("corpus.jsonl", "before.jsonl", "after.jsonl", "receipt.json", "before.json", "after.json")
            )
            rows = [eligible_case("single-case", "defect")]
            predictions = [proven_prediction(rows[0], "not-evaluated")]
            write_rows(corpus_path, rows)
            write_assembly_receipt(receipt_path, corpus_path, len(rows))
            write_rows(before_path, predictions)
            write_rows(after_path, predictions)
            before_manifest_path.write_text(json.dumps(evaluator_manifest()), encoding="utf-8")
            after_manifest_path.write_text(json.dumps(evaluator_manifest()), encoding="utf-8")
            artifact_dir = root / "review" / "run-001"
            command = [
                sys.executable, str(Path(__file__).with_name("score_ui_verdicts.py")),
                "--corpus", str(corpus_path), "--before", str(before_path), "--after", str(after_path),
                "--assembly-receipt", str(receipt_path), "--before-evaluator", str(before_manifest_path),
                "--after-evaluator", str(after_manifest_path), "--phases", "experience", "--artifact-dir", str(artifact_dir),
            ]
            completed = subprocess.run(command, text=True, capture_output=True, check=False)
            self.assertEqual(completed.returncode, 2, completed.stderr)
            self.assertIn("ui-verdict-trust-artifacts.json", completed.stdout)
            index_path = artifact_dir / "ui-verdict-trust-artifacts.json"
            report_path = artifact_dir / "ui-verdict-trust.md"
            json_path = artifact_dir / "ui-verdict-trust.json"
            index = json.loads(index_path.read_text(encoding="utf-8"))
            scored = json.loads(json_path.read_text(encoding="utf-8"))
            self.assertEqual(index["schema"], "ui-verdict-trust-artifacts/v1")
            self.assertEqual(index["artifacts"]["markdown_report"]["sha256"], hashlib.sha256(report_path.read_bytes()).hexdigest())
            self.assertEqual(index["artifacts"]["json_report"]["sha256"], hashlib.sha256(json_path.read_bytes()).hexdigest())
            self.assertEqual(index["inputs"]["corpus"]["sha256"], hashlib.sha256(corpus_path.read_bytes()).hexdigest())
            self.assertEqual(scored["status"], "insufficient-eligible-cases")
            self.assertEqual(scored["phase_results"]["experience"]["after"]["false_negatives"], 1)
            self.assertIsNone(scored["phase_results"]["experience"]["rates"])

    def test_artifact_directory_cannot_overwrite_scoring_input(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            colliding_input = root / "ui-verdict-trust.md"
            colliding_input.write_text("corpus placeholder\n", encoding="utf-8")
            command = [
                sys.executable, str(Path(__file__).with_name("score_ui_verdicts.py")),
                "--corpus", str(colliding_input), "--before", str(root / "before.jsonl"),
                "--after", str(root / "after.jsonl"), "--assembly-receipt", str(root / "receipt.json"),
                "--before-evaluator", str(root / "before.json"), "--after-evaluator", str(root / "after.json"),
                "--artifact-dir", str(root),
            ]
            completed = subprocess.run(command, text=True, capture_output=True, check=False)
            self.assertNotEqual(completed.returncode, 0)
            self.assertIn("would overwrite scoring input", completed.stderr)
            self.assertEqual(colliding_input.read_text(encoding="utf-8"), "corpus placeholder\n")

    def test_cli_publishes_paired_per_phase_rates_only_after_fifty_eligible_cases(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            corpus_path, before_path, after_path, receipt_path, before_manifest_path, after_manifest_path, output_path, json_path = (root / name for name in ("corpus.jsonl", "before.jsonl", "after.jsonl", "receipt.json", "before.json", "after.json", "report.md", "report.json"))
            rows = [eligible_case(f"case-{index}", "defect" if index % 2 == 0 else "clean") for index in range(50)]
            before_rows = [proven_prediction(row, "passed" if row["truth"] == "defect" else "failed") for row in rows]
            after_rows = [proven_prediction(row, "failed" if row["truth"] == "defect" else "passed") for row in rows]
            write_rows(corpus_path, rows)
            write_assembly_receipt(receipt_path, corpus_path, len(rows))
            write_rows(before_path, before_rows)
            write_rows(after_path, after_rows)
            before_manifest_path.write_text(json.dumps(evaluator_manifest()), encoding="utf-8")
            after_manifest_path.write_text(json.dumps(evaluator_manifest()), encoding="utf-8")
            argv = ["score_ui_verdicts.py", "--corpus", str(corpus_path), "--before", str(before_path), "--after", str(after_path), "--assembly-receipt", str(receipt_path), "--output", str(output_path), "--json-output", str(json_path), "--phases", "experience", "--before-evaluator", str(before_manifest_path), "--after-evaluator", str(after_manifest_path)]
            with patch("sys.argv", argv):
                self.assertEqual(main(), 0)
            payload = json.loads(json_path.read_text(encoding="utf-8"))
            phase = payload["phase_results"]["experience"]
            self.assertEqual(payload["status"], "measured")
            self.assertEqual(phase["status"], "measured")
            self.assertEqual(phase["rates"]["before"]["false_positive"], {"numerator": 25, "denominator": 25, "rate": 1.0})
            self.assertEqual(phase["rates"]["before"]["false_negative"], {"numerator": 25, "denominator": 25, "rate": 1.0})
            self.assertEqual(phase["rates"]["after"]["false_positive"], {"numerator": 0, "denominator": 25, "rate": 0.0})
            self.assertEqual(phase["rates"]["after"]["false_negative"], {"numerator": 0, "denominator": 25, "rate": 0.0})
            self.assertEqual(len(payload["input_sha256"]["corpus"]), 64)

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
        with self.assertRaisesRegex(ValueError, "duplicate corpus case_id/phase"):
            index_corpus([case("same"), case("same")])

    def test_same_case_can_be_scored_separately_for_each_phase(self):
        rows = [case("same", "defect", "experience"), case("same", "clean", "workflow")]
        predictions = [prediction("same", "failed", "experience"), prediction("same", "passed", "workflow")]
        results = score(rows, predictions)
        self.assertEqual(results["experience"]["counts"]["true_positives"], 1)
        self.assertEqual(results["workflow"]["counts"]["true_negatives"], 1)

    def test_invalid_truth_type_is_rejected(self):
        with self.assertRaisesRegex(ValueError, "truth must be defect or clean"):
            index_corpus([{**case("bad"), "truth": ["defect"]}])

    def test_independence_requires_all_exact_case_identity_fields(self):
        row = eligible_case("a")
        row.pop("viewport")
        result = independence_audit([row])["experience"]
        self.assertEqual(result["independently_adjudicated_clusters"], 0)
        self.assertEqual(result["ineligible_claims"], 1)

    def test_manifest_rejects_missing_or_stale_case_build_and_context(self):
        row = eligible_case("a")
        corpus = {("a", "experience"): row}
        manifest = evaluator_manifest()
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "manifest.json"
            path.write_text(json.dumps(manifest), encoding="utf-8")
            missing = proven_prediction(row)
            del missing["route"]
            with self.assertRaisesRegex(ValueError, "missing or stale provenance: route"):
                validate_evaluator_manifest(path, corpus, [missing], "after")
            stale = proven_prediction(row)
            stale["subject_build_identity"] = "old-subject-build"
            with self.assertRaisesRegex(ValueError, "subject_build_identity"):
                validate_evaluator_manifest(path, corpus, [stale], "after")
            stale = proven_prediction(row)
            stale["evaluator_build_identity"] = "old-evaluator-build"
            with self.assertRaisesRegex(ValueError, "evaluator_build_identity"):
                validate_evaluator_manifest(path, corpus, [stale], "after")
            stale = proven_prediction(row)
            stale["snapshot_sha256"] = "e" * 64
            with self.assertRaisesRegex(ValueError, "snapshot_sha256"):
                validate_evaluator_manifest(path, corpus, [stale], "after")
            workflow_row = eligible_case("workflow-case", phase="workflow")
            with self.assertRaisesRegex(ValueError, "no identity for phase"):
                validate_evaluator_manifest(path, {("workflow-case", "workflow"): workflow_row}, [proven_prediction(workflow_row, evaluator="workflow-health")], "after")

    def test_assembly_receipt_must_match_exact_corpus_and_rubric(self):
        row = eligible_case("receipt-case")
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            corpus_path, receipt_path = root / "corpus.jsonl", root / "receipt.json"
            write_rows(corpus_path, [row])
            write_assembly_receipt(receipt_path, corpus_path, 1)
            validate_assembly_receipt(receipt_path, corpus_path, {("receipt-case", "experience"): row})
            row["adjudication"]["rubric_ref"] = "rubrics/other.md"
            with self.assertRaisesRegex(ValueError, "does not match assembly receipt rubric"):
                validate_assembly_receipt(receipt_path, corpus_path, {("receipt-case", "experience"): row})
            row["adjudication"]["rubric_ref"] = "rubrics/ui-v1.md"
            corpus_path.write_text(corpus_path.read_text(encoding="utf-8") + "\n", encoding="utf-8")
            with self.assertRaisesRegex(ValueError, "does not match the exact corpus bytes"):
                validate_assembly_receipt(receipt_path, corpus_path, {("receipt-case", "experience"): row})

    def test_all_verdict_outcomes_have_explicit_counts_and_error_semantics(self):
        rows = [
            case("defect-pass", "defect"),
            case("defect-fail", "defect"),
            case("defect-uneval", "defect"),
            case("clean-pass", "clean"),
            case("clean-fail", "clean"),
            case("clean-uneval", "clean"),
        ]
        predictions = [
            prediction("defect-pass", "passed"),
            prediction("defect-fail", "failed"),
            prediction("defect-uneval", "not-evaluated"),
            prediction("clean-pass", "passed"),
            prediction("clean-fail", "failed"),
            prediction("clean-uneval", "not-evaluated"),
        ]
        counts = score(rows, predictions)["experience"]["counts"]
        self.assertEqual((counts["passed"], counts["failed"], counts["not_evaluated"]), (2, 2, 2))
        self.assertEqual((counts["false_positives"], counts["false_negatives"]), (1, 2))
        self.assertEqual((counts["true_positives"], counts["true_negatives"]), (1, 2))

    def test_fifty_independent_cases_publish_rates_with_denominators(self):
        rows = [eligible_case(f"case-{index}", "defect" if index % 2 == 0 else "clean") for index in range(50)]
        audit = independence_audit(rows)
        result = score(rows, [prediction(row["case_id"], "failed" if row["truth"] == "defect" else "passed") for row in rows])
        from score_ui_verdicts import phase_summaries

        summary = phase_summaries(result, result, audit, ["experience"], 50)["experience"]
        self.assertEqual(summary["status"], "measured")
        self.assertEqual(summary["rates"]["after"]["false_positive"], {"numerator": 0, "denominator": 25, "rate": 0.0})
        self.assertEqual(summary["rates"]["after"]["false_negative"], {"numerator": 0, "denominator": 25, "rate": 0.0})

    def test_scores_are_broken_out_by_rule(self):
        rows = [case("a"), {**case("b"), "case_id": "b", "rule_code": "overflow"}]
        result = score(rows, [prediction("a", "passed"), prediction("b", "failed")])
        self.assertEqual(result["experience"]["rules"]["unspecified"]["cases"], 1)
        self.assertEqual(result["experience"]["rules"]["overflow"]["false_positives"], 0)


if __name__ == "__main__":
    unittest.main()
