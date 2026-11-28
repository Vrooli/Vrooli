import unittest
import hashlib
import json
import tempfile
from pathlib import Path
from unittest.mock import patch

from assemble_ui_verdict_corpus import assemble, main
from score_ui_verdicts import independence_audit, independent_case_failures, score


def case(case_id, snapshot):
    return {
        "case_id": case_id,
        "phase": "experience",
        "scenario": "example-scenario",
        "build_identity": "git:0123456789abcdef",
        "route": "/home",
        "state": "default",
        "viewport": "390x844",
        "snapshot_sha256": snapshot * 64,
        "evidence_ref": f"frozen-evidence/{case_id}",
        "rule_code": "layout-overflow",
    }


def ballot(case_id, truth, reason=None):
    if truth == "insufficient":
        return {"case_id": case_id, "truth": truth, "insufficient_reason": reason or "missing state evidence"}
    return {"case_id": case_id, "truth": truth, "reason": reason or "rubric criterion applied", "evidence_node": "node-1"}


class AssembleCorpusTests(unittest.TestCase):
    def test_cli_writes_hashed_assembly_receipt(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            cases = [case("one", "a")]
            ballots = [ballot("one", "clean")]
            paths = {name: root / name for name in ("cases.jsonl", "a.jsonl", "b.jsonl", "out.jsonl", "excluded.jsonl", "receipt.json")}
            paths["cases.jsonl"].write_text(json.dumps(cases[0]) + "\n", encoding="utf-8")
            for key in ("a.jsonl", "b.jsonl"):
                paths[key].write_text(json.dumps(ballots[0]) + "\n", encoding="utf-8")
            argv = [
                "assemble_ui_verdict_corpus.py",
                "--cases", str(paths["cases.jsonl"]),
                "--reviewer-a", str(paths["a.jsonl"]),
                "--reviewer-b", str(paths["b.jsonl"]),
                "--rubric-ref", "rubric-v1",
                "--output", str(paths["out.jsonl"]),
                "--excluded-output", str(paths["excluded.jsonl"]),
                "--receipt-output", str(paths["receipt.json"]),
            ]
            with patch("sys.argv", argv):
                self.assertEqual(main(), 0)
            receipt = json.loads(paths["receipt.json"].read_text(encoding="utf-8"))
            self.assertEqual(receipt["scored_cases"], 1)
            self.assertEqual(receipt["excluded_insufficient_cases"], 0)
            self.assertEqual(receipt["reviewer_a_sha256"], hashlib.sha256(paths["a.jsonl"].read_bytes()).hexdigest())

    def test_cli_refuses_to_overwrite_case_input(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            paths = {name: root / name for name in ("cases.jsonl", "a.jsonl", "b.jsonl", "excluded.jsonl", "receipt.json")}
            paths["cases.jsonl"].write_text(json.dumps(case("one", "a")) + "\n", encoding="utf-8")
            for name in ("a.jsonl", "b.jsonl"):
                paths[name].write_text(json.dumps(ballot("one", "clean")) + "\n", encoding="utf-8")
            original = paths["cases.jsonl"].read_bytes()
            argv = [
                "assemble_ui_verdict_corpus.py",
                "--cases", str(paths["cases.jsonl"]),
                "--reviewer-a", str(paths["a.jsonl"]),
                "--reviewer-b", str(paths["b.jsonl"]),
                "--rubric-ref", "rubric-v1",
                "--output", str(paths["cases.jsonl"]),
                "--excluded-output", str(paths["excluded.jsonl"]),
                "--receipt-output", str(paths["receipt.json"]),
            ]
            with patch("sys.argv", argv):
                self.assertEqual(main(), 2)
            self.assertEqual(paths["cases.jsonl"].read_bytes(), original)

    def test_agreement_adjudication_and_insufficient_exclusion(self):
        cases = [case("agree", "a"), case("disagree", "b"), case("insufficient", "c")]
        reviewer_a = [ballot("agree", "clean"), ballot("disagree", "defect"), ballot("insufficient", "insufficient")]
        reviewer_b = [ballot("agree", "clean"), ballot("disagree", "clean"), ballot("insufficient", "insufficient")]
        adjudicator = [ballot("disagree", "defect"), ballot("insufficient", "insufficient", "state could not be reconstructed")]

        scored, excluded = assemble(cases, reviewer_a, reviewer_b, adjudicator, "rubrics/experience-v1.md")

        self.assertEqual([row["case_id"] for row in scored], ["agree", "disagree"])
        self.assertEqual(scored[0]["adjudication"]["reviewer_count"], 2)
        self.assertEqual(scored[1]["truth"], "defect")
        self.assertEqual(scored[1]["adjudication"]["reviewer_count"], 3)
        self.assertEqual(excluded[0]["reason"], "state could not be reconstructed")
        self.assertNotIn("reason", scored[0])

    def test_disagreement_requires_a_third_vote(self):
        with self.assertRaisesRegex(ValueError, "missing 1 adjudication"):
            assemble([case("x", "d")], [ballot("x", "defect")], [ballot("x", "clean")], None, "rubric-v1")

    def test_each_reviewer_must_cover_exact_packet(self):
        with self.assertRaisesRegex(ValueError, "reviewer B coverage"):
            assemble([case("x", "e")], [ballot("x", "clean")], [], None, "rubric-v1")

    def test_case_packet_rejects_embedded_verdicts(self):
        candidate = {**case("x", "f"), "verdict": "passed"}
        with self.assertRaisesRegex(ValueError, "contains evaluator output fields"):
            assemble([candidate], [ballot("x", "clean")], [ballot("x", "clean")], None, "rubric-v1")

    def test_case_packet_rejects_nested_prediction_fields(self):
        candidate = {**case("x", "f"), "evidence": {"phase_output": {"status": "passed"}}}
        with self.assertRaisesRegex(ValueError, "evidence.phase_output"):
            assemble([candidate], [ballot("x", "clean")], [ballot("x", "clean")], None, "rubric-v1")

    def test_unhashable_label_is_rejected_as_invalid(self):
        malformed = {"case_id": "x", "truth": ["clean"]}
        with self.assertRaisesRegex(ValueError, "truth must be defect"):
            assemble([case("x", "f")], [malformed], [ballot("x", "clean")], None, "rubric-v1")

    def test_same_snapshot_rule_claims_remain_one_case_cluster_for_scorer(self):
        first = case("x-one", "a")
        second = {**case("x-two", "a"), "rule_code": "touch-target"}
        scored, excluded = assemble([first, second], [ballot("x-one", "clean"), ballot("x-two", "defect")], [ballot("x-one", "clean"), ballot("x-two", "defect")], None, "rubric-v1")
        self.assertEqual(len(scored), 2)
        self.assertEqual({row["snapshot_sha256"] for row in scored}, {"a" * 64})
        self.assertEqual(excluded, [])

    def test_fifty_blind_cases_flow_into_independence_gate_and_phase_rates(self):
        import hashlib

        cases = []
        reviewer_a = []
        reviewer_b = []
        predictions = []
        for index in range(50):
            case_id = f"case-{index:03d}"
            truth = "defect" if index < 25 else "clean"
            cases.append({
                "case_id": case_id,
                "phase": "experience",
                "scenario": f"scenario-{index % 10}",
                "build_identity": f"build-{index % 2}",
                "route": f"/route-{index % 5}",
                "state": f"state-{index % 3}",
                "viewport": "390x844",
                "snapshot_sha256": hashlib.sha256(case_id.encode()).hexdigest(),
                "evidence_ref": f"frozen/{case_id}",
            })
            reviewer_a.append(ballot(case_id, truth))
            reviewer_b.append(ballot(case_id, truth))
            verdict = "passed"
            if truth == "defect" and index < 19:
                verdict = "failed"
            elif truth == "defect" and index == 19:
                verdict = "not-evaluated"
            elif truth == "clean" and index < 27:
                verdict = "failed"
            predictions.append({"case_id": case_id, "phase": "experience", "verdict": verdict})

        corpus, excluded = assemble(cases, reviewer_a, reviewer_b, None, "rubric-v1")
        audit = independence_audit(corpus)
        self.assertEqual(excluded, [])
        self.assertEqual(audit["experience"]["independently_adjudicated_clusters"], 50)
        self.assertEqual(independent_case_failures(audit, ["experience"], 50), [])
        counts = score(corpus, predictions)["experience"]["counts"]
        self.assertEqual(counts["false_positives"], 2)
        self.assertEqual(counts["false_negatives"], 6)
        self.assertEqual(counts["not_evaluated"], 1)


if __name__ == "__main__":
    unittest.main()
