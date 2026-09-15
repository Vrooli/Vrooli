"""Behavioral tests for unit-health.setpoint-read against the real kernel program helper."""
import contextlib
import io
import json
import sys
import unittest
from pathlib import Path
from types import SimpleNamespace as NS

ROOT = Path(__file__).resolve().parents[1]
REPO = Path(__file__).resolve().parents[5]
sys.dont_write_bytecode = True
sys.path.insert(0, str(REPO / "scenarios/program-runtime/kernel"))
from host.program_helper import ProgramHelper, ScenarioUnreachable  # noqa: E402

SOURCE = (ROOT / "setpoint-read.py").read_text()
ROW_ORDER = [
    "self-maturity", "self-execution", "unknown-share", "rules-promoted", "reviewed-evidence",
    "requirement-traceability", "calibration-floor", "corpus-implemented-share", "holdout-agreement",
    "declared-vs-emitted", "mutation-signal", "fleet-adoption", "external-friction",
]


def result(rule, status, enforcement="QUALITY_ENFORCEMENT_ADVISORY", reason="QUALITY_REASON_NONE"):
    return {"ruleId": rule, "status": "QUALITY_CHECK_STATUS_" + status, "enforcement": enforcement, "reason": reason}


def payload(**changes):
    base = {
        "runId": "uh-fixture",
        "status": "passed",
        "evidenceStages": {"configured": "observed", "analyzed": "partial", "executed": "not_requested", "reviewed": "not_supplied"},
        "assessment": {
            "local": {"currentLevel": "L1", "nextLevel": "L2", "blockingFindingCodes": ["TEST_UTIL_MISSING"]},
            "capabilities": [{"id": "test_architecture", "currentLevel": "L1"}, {"id": "framework_config", "currentLevel": "L3"}],
        },
        "findings": [{"code": "TEST_UTIL_MISSING"}],
        "testQuality": {"results": [
            result("assertion-observation", "CHECKED_CLEAN"), result("assertion-observation", "UNKNOWN"),
            result("assertion-observation", "NOT_APPLICABLE"), result("focused-test", "CHECKED_CLEAN"),
            result("async-assertion", "UNKNOWN", reason="QUALITY_REASON_NOT_EXECUTED"),
        ]},
        "traceability": {"unavailableReason": "QUALITY_REASON_OWNER_UNAVAILABLE"},
    }
    base.update(changes)
    return base


class SetpointReadTests(unittest.TestCase):
    def invoke(self, inputs=None, response=None, calibration=None, raise_with=None, raise_calibration=None, digest_payload=None, raise_friction=None):
        calls = []

        def scenario(**kwargs):
            calls.append(kwargs)
            if raise_with is not None:
                raise raise_with
            scenario_payload = response or payload(workspaces=[{"id": "api"}])
            if isinstance(scenario_payload, dict):
                scenario_payload.setdefault("workspaces", [{"id": "api"}])
            return NS(raw=lambda: scenario_payload, meta=lambda: {}, head=lambda n: [])

        def calibrate(**kwargs):
            calls.append(kwargs)
            if raise_calibration is not None:
                raise raise_calibration
            return NS(raw=lambda: calibration or self.calibration_payload(), meta=lambda: {}, head=lambda n: [])

        def mutation(**kwargs):
            calls.append(kwargs)
            return NS(raw=lambda: {"summary": {"killed": 19, "survived": 1, "invalid": 0, "killRate": 0.95}, "runId": "mutation-fixture"}, meta=lambda: {}, head=lambda n: [])

        def digest(**kwargs):
            calls.append(kwargs)
            child = digest_payload or {"status": "ok", "signals": {"terminal": 1, "withEvidence": 1}}
            return NS(raw=lambda: [child], meta=lambda: {}, head=lambda n: [child])

        def friction(**kwargs):
            calls.append(kwargs)
            if raise_friction is not None:
                raise raise_friction
            child = {"status": "ok", "signals": {"recurringCount": 0, "topFingerprints": []}}
            return NS(raw=lambda: [child], meta=lambda: {}, head=lambda n: [child])

        helper = ProgramHelper()
        env = {"inputs": inputs or {}, "unit_health": NS(validate=NS(scenario=scenario), calibrate=NS(run=calibrate), mutation=NS(pilot=mutation)), "lib": NS(test_genie=NS(validation_digest=digest), agent_manager=NS(friction_digest=friction)), "program": helper,
               "gather": lambda *calls: [call() for call in calls]}
        helper._bind(env)
        stream = io.StringIO()
        with contextlib.redirect_stdout(stream):
            exec(compile(SOURCE, "setpoint-read", "exec"), env)
        lines = [line for line in stream.getvalue().splitlines() if line.strip()]
        self.assertEqual(len(lines), 1, "the program prints exactly one envelope")
        return json.loads(lines[0]), calls

    @staticmethod
    def calibration_payload(matched=True, implemented=28, retired=0, specified=28, missing=0):
        return {
            "runId": "cal-fixture", "partition": "inventory",
            "corpus": {"implemented": implemented, "retired": retired, "specified": specified,
                       "developmentFloor": "28/28@2026-09-09", "specCodesWithoutEmitter": missing},
            "cases": [{"id": f"C{i:03d}", "matched": matched} for i in range(1, implemented + 1)],
        }

    def rows(self, envelope):
        return {row["row"]: row for row in envelope["signals"]["rows"]}

    def test_plan_only_board_declares_every_row_in_order_and_declines_execution(self):
        envelope, calls = self.invoke(response=payload())
        self.assertEqual([row["row"] for row in envelope["signals"]["rows"]], ROW_ORDER)
        self.assertEqual(calls, [{"scenario": "unit-health", "include_execution": False, "rows": "findings"}, {"partition": "inventory", "rows": "cases"}, {"partition": "reviewed-holdout", "holdout": "assertion-observation-go-v1", "rule": "assertion-observation", "rows": "holdout"}, {"scenario": "unit-health", "include_execution": False, "rows": "findings"}, {"caller_scenario": "unit-health", "limit": 20}, {"scenario": "unit-health", "window_days": 7}])
        rows = self.rows(envelope)
        for row in rows.values():
            self.assertEqual(set(row), {"row", "reading", "target", "in_band", "unavailable", "reason"})
        self.assertEqual(rows["self-maturity"]["reading"]["level"], "L1")
        self.assertEqual(rows["self-maturity"]["reading"]["blocking"], ["TEST_UTIL_MISSING"])
        self.assertIs(rows["self-maturity"]["in_band"], False)
        self.assertEqual(rows["self-execution"]["reason"], "kernel_invoke_budget")
        self.assertIsNone(rows["self-execution"]["reading"])
        self.assertEqual(rows["calibration-floor"]["reading"], {"matched": 28, "implemented": 28, "floor": "28/28@2026-09-09"})
        self.assertTrue(rows["corpus-implemented-share"]["in_band"])
        self.assertTrue(rows["declared-vs-emitted"]["in_band"])
        self.assertEqual(rows["holdout-agreement"]["reason"], "pending_telemetry")
        self.assertEqual(rows["fleet-adoption"]["reading"], {"applicable": 1, "covered": 1, "uncovered": [], "child_failures": []})
        self.assertEqual(rows["external-friction"]["reading"], {"recurring_count": 0, "top_fingerprints": []})

    def test_unknown_share_excludes_not_applicable_and_never_counts_unknown_as_clean(self):
        envelope, _ = self.invoke(response=payload())
        rows = self.rows(envelope)
        share = rows["unknown-share"]["reading"]
        self.assertEqual(share["per_rule"]["assertion-observation"], 0.5)
        self.assertEqual(share["per_rule"]["focused-test"], 0.0)
        self.assertEqual(share["worst"], 0.5)
        self.assertEqual(share["not_executed"], 1, "a plan-only unknown is excluded from the denominator, never counted clean")
        self.assertIsNone(share["per_rule"]["async-assertion"])
        self.assertIs(rows["unknown-share"]["in_band"], False)
        self.assertEqual(rows["rules-promoted"]["reading"], {"promoted": 0, "rules_observed": 3, "promoted_rules": []})
        self.assertIsNone(rows["rules-promoted"]["target"])
        self.assertIsNone(rows["rules-promoted"]["in_band"])

    def test_unreachable_traceability_owner_is_unreliable_and_does_not_lower_the_board(self):
        envelope, _ = self.invoke(response=payload())
        rows = self.rows(envelope)
        self.assertEqual(rows["requirement-traceability"]["reason"], "unreliable:owner_unavailable")
        self.assertIsNone(rows["requirement-traceability"]["in_band"])
        self.assertEqual(envelope["status"], "ok", "an unreliable sensor keeps in_band null; only a failed read makes the board partial")

    def test_permanent_reasons_alone_keep_the_board_ok(self):
        response = payload(traceability={"unavailableReason": "QUALITY_REASON_NONE"})
        response["testQuality"]["results"].append(result("requirement-link", "CHECKED_CLEAN"))
        envelope, _ = self.invoke(response=response)
        rows = self.rows(envelope)
        self.assertEqual(rows["requirement-traceability"]["reading"], {"owner": "reachable", "untagged": 0})
        self.assertIs(rows["requirement-traceability"]["in_band"], True)
        self.assertEqual(envelope["status"], "ok")

    def test_executed_read_bands_self_execution_on_coverage_and_execution(self):
        response = payload(evidenceStages={"executed": "passed", "reviewed": "supplied"}, findings=[{"code": "LOW_COVERAGE"}, {"code": "LOW_COVERAGE"}])
        envelope, calls = self.invoke(inputs={"execution": True}, response=response)
        rows = self.rows(envelope)
        self.assertEqual(calls[0]["include_execution"], True)
        self.assertEqual(calls[3], {"scenario": "unit-health", "workspace": "api", "package": "./internal/testquality/...", "seed": "pilot-1", "rows": "receipts"})
        self.assertEqual(calls[-1], {"scenario": "unit-health", "window_days": 7})
        self.assertEqual(rows["self-execution"]["reading"], {"executed": "passed", "low_coverage": 2})
        self.assertIs(rows["self-execution"]["in_band"], False)
        self.assertIs(rows["reviewed-evidence"]["in_band"], True)

    def test_unreachable_owner_declines_every_sensor_row_and_keeps_declared_rows(self):
        envelope, _ = self.invoke(raise_with=ScenarioUnreachable("connection refused", binding_id="unit-health/validate/scenario"))
        rows = self.rows(envelope)
        self.assertEqual(envelope["status"], "partial")
        self.assertEqual(envelope["errors"][0]["class"], "scenario_unreachable")
        for name in ("self-maturity", "unknown-share", "reviewed-evidence", "requirement-traceability"):
            self.assertEqual(rows[name]["reason"], "scenario_unreachable")
            self.assertIsNone(rows[name]["reading"])
        self.assertEqual(rows["calibration-floor"]["reason"], "unreliable:calibration-read-failed")
        self.assertEqual(len(rows), len(ROW_ORDER))

    def test_invalid_inputs_fail_before_any_read(self):
        for inputs in ({"target": "Not A Scenario"}, {"execution": "yes"}):
            envelope, calls = self.invoke(inputs=inputs, response=payload())
            self.assertEqual(envelope["status"], "failed")
            self.assertEqual(envelope["errors"][0]["class"], "invalid_input")
            self.assertEqual(calls, [])

    def test_contract_matches_schema_and_source(self):
        import jsonschema
        schema = json.loads((REPO / "scenarios/program-runtime/schemas/program-contract.schema.json").read_text())
        contract = json.loads((ROOT / "setpoint-read.json").read_text())
        jsonschema.validate(contract, schema)
        self.assertEqual(contract["owner_skill"], "unit-health-improve")
        self.assertEqual([b["id"] for b in contract["bindings"]], ["unit-health/validate/scenario", "unit-health/calibrate/run", "unit-health/mutation/pilot", "test-genie/validation/list", "agent-manager/run/episodes", "agent-manager/run/list", "agent-manager/watch/policy-outcomes"])
        for name in ROW_ORDER:
            self.assertIn(name, contract["outputs"]["signals"]["rows"])

    def test_calibration_rows_go_out_of_band_without_inventing_a_match(self):
        calibration = self.calibration_payload(matched=False, retired=1, specified=28, missing=2)
        envelope, _ = self.invoke(calibration=calibration)
        rows = self.rows(envelope)
        self.assertEqual(rows["calibration-floor"]["reading"]["matched"], 0)
        self.assertFalse(rows["calibration-floor"]["in_band"])
        self.assertFalse(rows["corpus-implemented-share"]["in_band"])
        self.assertFalse(rows["declared-vs-emitted"]["in_band"])

    def test_calibration_binding_failure_is_unreliable_and_does_not_lower_board(self):
        envelope, _ = self.invoke(response=payload(), raise_calibration=ScenarioUnreachable("calibration unavailable", binding_id="unit-health/calibrate/run"))
        rows = self.rows(envelope)
        self.assertEqual(rows["calibration-floor"]["reason"], "unreliable:calibration-read-failed")
        self.assertIsNone(rows["calibration-floor"]["in_band"])
        self.assertEqual(envelope["status"], "ok")

    def test_reviewed_holdout_reads_budget_and_stays_advisory(self):
        calibration = self.calibration_payload()
        calibration["holdout"] = [{"labelled": 30, "observed": 30, "fpRate": 0.0, "fnRate": 0.0, "unknown": 0, "budget": 0.05, "withinBudget": True, "promotionAllowed": False}]
        envelope, _ = self.invoke(calibration=calibration)
        rows = self.rows(envelope)
        self.assertTrue(rows["holdout-agreement"]["in_band"])
        self.assertEqual(rows["holdout-agreement"]["reading"]["budget"], 0.05)
        self.assertFalse(rows["holdout-agreement"]["reading"]["promotion_allowed"])

    def test_reviewed_cohort_is_attached_to_validation_sensor(self):
        response = payload(evidenceStages={"reviewed": "supplied"})
        envelope, calls = self.invoke(inputs={"reviewed_cohort_id": "cohort-1", "reviewed_source_identity": "sha256:s", "reviewed_observation_count": 2}, response=response)
        self.assertEqual(calls[0]["reviewed_cohort_id"], "cohort-1")
        self.assertEqual(calls[0]["reviewed_source_identity"], "sha256:s")
        self.assertEqual(calls[0]["reviewed_observation_count"], 2)
        self.assertTrue(self.rows(envelope)["reviewed-evidence"]["in_band"])

    def test_mutation_signal_runs_only_with_execution_and_uses_recorded_floor(self):
        envelope, calls = self.invoke(inputs={"execution": True, "mutation_floor": 0.95}, response=payload())
        mutation = self.rows(envelope)["mutation-signal"]
        self.assertEqual(mutation["reading"], {"killed": 19, "survived": 1, "invalid": 0, "kill_rate": 0.95})
        self.assertTrue(mutation["in_band"])
        self.assertEqual(calls[3]["rows"], "receipts")

        plan_only, calls = self.invoke(response=payload())
        self.assertEqual(self.rows(plan_only)["mutation-signal"]["reason"], "kernel_invoke_budget")
        self.assertFalse(any("mutation" in call for call in calls))

    def test_fleet_uncovered_and_child_failure_are_distinct(self):
        uncovered, _ = self.invoke(digest_payload={"status": "ok", "signals": {"terminal": 0, "withEvidence": 0}})
        fleet = self.rows(uncovered)["fleet-adoption"]
        self.assertEqual(fleet["reading"]["uncovered"], ["unit-health"])
        self.assertFalse(fleet["in_band"])

        failure, _ = self.invoke(digest_payload={"status": "partial", "signals": {"terminal": 0, "withEvidence": 0}})
        fleet = self.rows(failure)["fleet-adoption"]
        self.assertEqual(fleet["reading"]["child_failures"][0]["scenario"], "unit-health")
        self.assertEqual(fleet["reading"]["uncovered"], [])

    def test_friction_child_failure_is_unreliable_not_zero(self):
        envelope, _ = self.invoke(raise_friction=ScenarioUnreachable("friction unavailable", binding_id="agent-manager/friction-digest"))
        friction = self.rows(envelope)["external-friction"]
        self.assertTrue(friction["reason"].startswith("unreliable:"))
        self.assertIsNone(friction["reading"])


if __name__ == "__main__":
    unittest.main()
