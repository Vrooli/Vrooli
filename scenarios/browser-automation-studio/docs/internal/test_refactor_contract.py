"""Preparation regressions: omissions and unknown evidence must not look complete."""

import contextlib
import copy
import io
import json
import runpy
import unittest
from types import SimpleNamespace

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

    def classify(self, exc):
        return "unavailable", "no_governed_binding"

    def run(self, states, state):
        while state is not None:
            state = states[state]()


def board(inputs):
    # No BAS/lib bindings exist here. The preparation profile must be usable
    # before broken product sensors exist, without treating them as passing.
    output = io.StringIO()
    empty = SimpleNamespace(head=lambda limit: [])
    performance = SimpleNamespace(sweep=SimpleNamespace(workload_get=lambda **_: empty))
    test_genie = SimpleNamespace(runs=SimpleNamespace(list=lambda **_: empty))
    vrooli = SimpleNamespace(scenario=SimpleNamespace(
        status=lambda **_: SimpleNamespace(raw=lambda: {
            "runtime": {"buildIdentity": "sha256:current"},
        })
    ))
    with contextlib.redirect_stdout(output):
        namespace = runpy.run_path(str(SCENARIO / ".vrooli/program-runtime/setpoint-read.py"),
                                   init_globals={"program": ProgramHarness(inputs),
                                                 "performance_health": performance,
                                                 "test_genie": test_genie,
                                                 "vrooli": vrooli})
    return json.loads(output.getvalue()), len(output.getvalue().encode()), namespace


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
        result, size, _ = board({"profile": "rehabilitation"})
        self.assertEqual("ok", result["status"])
        self.assertFalse(result["signals"]["product_qualified"])
        self.assertEqual(len(self.contract["rows"]), result["signals"]["unmet"])
        self.assertLessEqual(size, 4096)
        for row in result["signals"]["rows"]:
            self.assertTrue(row["unavailable"])
            self.assertIsNone(row["reading"])
            self.assertIsNone(row["in_band"])

    def test_unknown_profile_fails_without_domain_calls(self):
        result, _, _ = board({"profile": "pretend-release"})
        self.assertEqual("failed", result["status"])
        self.assertEqual("invalid_input", result["errors"][0]["class"])

    def test_probe_success_requires_nonempty_valid_behavioral_results(self):
        self.assertEqual("unavailable", classify({"results": []}))
        self.assertEqual("unavailable", classify({"results": [{"id": "a"}]}))
        self.assertEqual("failed", classify({"results": [{"id": "a", "expected_behavior_met": False}]}))
        self.assertEqual("passed", classify({"results": [{"id": "a", "expected_behavior_met": True}]}))

    def test_profile_sensor_uses_exact_fresh_composite_phase(self):
        _, _, namespace = board({"profile": "rehabilitation"})
        qualifies = namespace["latest_rehabilitation_run"]
        phase = {"name": "rehabilitation-evidence", "status": "passed"}
        good = {"planned_phases": ["rehabilitation-evidence"], "status": "passed",
                "phases": [phase], "completed_at": "2026-09-24T05:00:00Z"}
        self.assertIs(qualifies([good], "2026-09-24T04:50:00Z"), good)
        self.assertIsNone(qualifies([good], "2026-09-24T05:01:00Z"))
        self.assertIsNone(qualifies([{**good, "planned_phases": ["unit", "rehabilitation-evidence"]}], "2026-09-24T04:50:00Z"))
        self.assertIsNone(qualifies([{**good, "phases": [phase, {"name": "unit", "status": "passed"}]}],
                                    "2026-09-24T04:50:00Z"))
        failed_overall = {**good, "status": "failed",
                          "phases": [{"name": "rehabilitation-evidence", "status": "failed"}]}
        self.assertIs(qualifies([failed_overall], "2026-09-24T04:50:00Z"), failed_overall)
        self.assertIsNone(qualifies([{**good, "status": "running"}], "2026-09-24T04:50:00Z"))

    def test_capability_standing_reads_only_the_named_provider_capability(self):
        _, _, namespace = board({"profile": "rehabilitation"})
        findings = {"phases": [{
            "name": "rehabilitation-evidence",
            "phase_presentation": {"capabilities": [
                {"id": "profile-durability", "current_level": "L1", "clean": True},
                {"id": "cancellation-recovery", "current_level": "L0", "clean": False},
            ]},
        }]}
        self.assertEqual("L1", namespace["capability_standing"](findings, "profile-durability")["level"])
        self.assertEqual("L0", namespace["capability_standing"](findings, "cancellation-recovery")["level"])
        self.assertIsNone(namespace["capability_standing"](findings, "interactive-feedback"))


if __name__ == "__main__":
    unittest.main()
