"""Preparation regressions: omissions and unknown evidence must not look complete."""

import contextlib
import copy
import io
import json
import runpy
import unittest
from types import SimpleNamespace

from refactor_contract import SCENARIO, validate, validate_protocol_evidence_references
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


def board(inputs, *, freshness_binding=True):
    # No BAS outcome bindings exist here. Model a successful lifecycle freshness
    # guard by default so the harness tests unavailable product sensors, not a
    # missing build-identity precondition.
    output = io.StringIO()
    empty = SimpleNamespace(head=lambda limit: [])
    performance = SimpleNamespace(sweep=SimpleNamespace(workload_get=lambda **_: empty))
    test_genie = SimpleNamespace(runs=SimpleNamespace(list=lambda **_: empty))
    scenario = SimpleNamespace(status=lambda **_: SimpleNamespace(raw=lambda: {
        "runtime": {"buildIdentity": "sha256:current"},
    }))
    if freshness_binding:
        scenario.freshness = lambda **_: SimpleNamespace(raw=lambda: {
            "success": True,
            "stale": False,
            "checks": [{"target": "fixture-api", "stale": False}],
        })
    vrooli = SimpleNamespace(scenario=scenario)
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

    def test_continuous_goal_uses_large_epochs_and_bounded_active_state(self):
        goal = (SCENARIO / "docs/internal/REFRACTOR_GOAL.md").read_text()
        protocol = (SCENARIO / "docs/internal/TESTING.md").read_text()
        progress = (SCENARIO / "docs/internal/REFRACTOR_PROGRESS.md").read_text()
        feedback = (SCENARIO / "docs/internal/OPERATOR_FEEDBACK.md").read_text()
        normalized_protocol = " ".join(protocol.split())

        self.assertLessEqual(len(goal), 2048)
        for required in (
            "normally survives multiple context compactions",
            "One work unit, one score row or whatever fits before the next",
            "Compaction, a status request or an interruption creates an **in-epoch",
            "rebuild/restart the managed candidate once",
            "The agent never commits",
            "active resume packet",
            "96 KiB",
            "128 files, 16 MiB and 250,000 text",
        ):
            self.assertIn(required, normalized_protocol)

        self.assertLessEqual(sum(len(item.encode()) for item in (goal, protocol, progress, feedback)),
                             96 * 1024)
        self.assertIn("## Active candidate epoch", progress)
        self.assertNotIn("recent append-only execution history", progress)

    def test_evidence_references_must_resolve_to_current_tests(self):
        for replacement in (
            "api/missing_test.go :: TestDoesNotExist",
            "api/services/session-profile/persistence/file_repository_test.go :: TestRenamedWithoutUpdatingContract",
        ):
            with self.subTest(reference=replacement):
                changed = copy.deepcopy(self.contract)
                changed["journeys"][0]["evidence"] = [replacement]
                errors = validate(changed)
                self.assertTrue(any("evidence reference" in error.lower() for error in errors), errors)

    def test_j06_partial_fault_tests_do_not_qualify_the_journey(self):
        journey = next(item for item in self.contract["journeys"] if item["id"] == "BAS-RH-J06")
        self.assertEqual("planned", journey["state"])
        self.assertEqual([], journey["evidence"], "partial links must not change the hashed acceptance contract")
        protocol = (SCENARIO / "docs/internal/TESTING.md").read_text()
        references = [
            ("api/services/session-profile/persistence/file_repository_test.go", "TestFileRepositoryCommitPreservesAcknowledgedSnapshot"),
            ("api/services/session-profile/persistence/file_repository_test.go", "TestFileRepositoryUpdateFailurePreservesSnapshot"),
            ("api/services/recording/service_test.go", "TestJournalHistorySurvivesPaginationReopenAndConcurrentWriters"),
            ("api/services/recording/service_test.go", "TestJournalSameIDRetryRecoversAcrossServiceProcessDeath"),
        ]
        for path, name in references:
            with self.subTest(test=name):
                source = (SCENARIO / path).read_text()
                self.assertIn(f"| BAS-RH-J06 | `{path} :: {name}` |", protocol)
                self.assertIn(name, source)
        self.assertGreaterEqual((SCENARIO / references[0][0]).read_text().count("[REQ:BAS-RH-J06]"), 2)

    def test_partial_protocol_references_cannot_go_stale(self):
        protocol = (SCENARIO / "docs/internal/TESTING.md").read_text()
        broken = protocol.replace("TestFileRepositoryUpdateFailurePreservesSnapshot", "TestRenamedWithoutUpdatingProtocol")
        errors = validate_protocol_evidence_references(self.contract["journeys"], SCENARIO, broken)
        self.assertTrue(any("evidence reference test" in error.lower() for error in errors), errors)

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

    def test_missing_freshness_binding_withholds_all_rows(self):
        result, _, _ = board({"profile": "rehabilitation"}, freshness_binding=False)
        self.assertEqual("partial", result["status"])
        self.assertFalse(result["signals"]["product_qualified"])
        self.assertFalse(result["signals"]["candidate_freshness"]["fresh"])
        self.assertEqual(len(self.contract["rows"]), result["signals"]["unmet"])
        for row in result["signals"]["rows"]:
            self.assertTrue(row["unavailable"])
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
