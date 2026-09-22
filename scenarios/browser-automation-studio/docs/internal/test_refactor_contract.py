"""Preparation regressions: omissions and unknown evidence must not look complete."""

import contextlib
import copy
import io
import json
import runpy
import unittest

from refactor_contract import SCENARIO, validate
from refactor_regressions import classify


class ProgramHarness:
    def __init__(self, inputs):
        self.request = inputs

    def inputs(self):
        return self.request

    def attach(self, envelope):
        self.envelope = envelope

    def fail(self, status, klass, detail, where):
        self.envelope.update(status=status, phase=where)
        self.envelope["errors"].append({"class": klass, "detail": str(detail), "where": where})
        return "report"

    def run(self, states, state):
        while state is not None:
            state = states[state]()


def board(inputs):
    # No BAS/lib bindings exist here. The preparation profile must be usable
    # before broken product sensors exist, without treating them as passing.
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        runpy.run_path(str(SCENARIO / ".vrooli/program-runtime/setpoint-read.py"),
                       init_globals={"program": ProgramHarness(inputs)})
    return json.loads(output.getvalue()), len(output.getvalue().encode())


class PreparationTest(unittest.TestCase):
    def setUp(self):
        self.contract = json.loads((SCENARIO / "docs/internal/REFRACTOR_CONTRACT.json").read_text())

    def test_preparation_is_consistent(self):
        self.assertEqual([], validate(self.contract))

    def test_unavailable_validation_cannot_block_or_be_relabelled_passed(self):
        for key in ("unavailable_validation_stops_work", "unknown_evidence_counts_as_pass"):
            with self.subTest(key=key):
                changed = copy.deepcopy(self.contract)
                changed["continuation_policy"][key] = True
                self.assertTrue(validate(changed))

    def test_plan_dependency_and_automatic_completion_are_rejected(self):
        for section, key in (("work_model", "plan_manager_allowed"),
                             ("work_model", "phased_plan_allowed"),
                             ("continuation_policy", "auto_complete_on_green"),
                             ("continuation_policy", "auto_complete_after_clean_reviews")):
            with self.subTest(key=key):
                changed = copy.deepcopy(self.contract)
                changed[section][key] = True
                self.assertTrue(validate(changed))

    def test_dropped_or_duplicated_journey_is_rejected(self):
        changed = copy.deepcopy(self.contract)
        changed["journeys"][-1] = changed["journeys"][0]
        self.assertTrue(validate(changed))

    def test_missing_band_and_producer_cannot_be_prepared(self):
        changed = copy.deepcopy(self.contract)
        changed["rows"][0]["band"] = ""
        changed["rows"][0]["producer_roots"] = []
        self.assertGreaterEqual(len(validate(changed)), 2)

    def test_unavailable_evidence_is_not_product_completion(self):
        result, size = board({"profile": "rehabilitation"})
        self.assertEqual("ok", result["status"])
        self.assertFalse(result["signals"]["product_qualified"])
        self.assertEqual(len(self.contract["rows"]), result["signals"]["unmet"])
        self.assertLessEqual(size, 4096)
        for row in result["signals"]["rows"]:
            self.assertTrue(row["unavailable"])
            self.assertIsNone(row["reading"])
            self.assertIsNone(row["in_band"])

    def test_unknown_profile_fails_without_domain_calls(self):
        result, _ = board({"profile": "pretend-release"})
        self.assertEqual("failed", result["status"])
        self.assertEqual("invalid_input", result["errors"][0]["class"])

    def test_probe_success_requires_nonempty_valid_behavioral_results(self):
        self.assertEqual("unavailable", classify({"results": []}))
        self.assertEqual("unavailable", classify({"results": [{"id": "a"}]}))
        self.assertEqual("failed", classify({"results": [{"id": "a", "expected_behavior_met": False}]}))
        self.assertEqual("passed", classify({"results": [{"id": "a", "expected_behavior_met": True}]}))


if __name__ == "__main__":
    unittest.main()
