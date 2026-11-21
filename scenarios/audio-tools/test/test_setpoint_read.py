"""Board behavior with governed bindings replaced, not audio qualification."""
import contextlib
import io
import importlib.util
import json
import re
from pathlib import Path
from types import SimpleNamespace
import unittest


SOURCE = Path(__file__).resolve().parents[1] / ".vrooli/program-runtime/setpoint-read.py"
HELPER_SOURCE = SOURCE.parents[3] / "program-runtime/kernel/host/program_helper.py"
helper_spec = importlib.util.spec_from_file_location("audio_board_program_helper", HELPER_SOURCE)
program_helper = importlib.util.module_from_spec(helper_spec)
helper_spec.loader.exec_module(program_helper)


class Handle:
    def __init__(self, value):
        self.value = value

    def head(self, limit):
        return self.value[:limit]

    def meta(self):
        return self.value

    def raw(self):
        return self.value


def run_board(inputs=None, engines=None, health=None, evidence=None, report=None, errors=None):
    calls = []
    errors = errors or {}

    def binding(name, value):
        def invoke(**kwargs):
            calls.append((name, kwargs))
            if name in errors:
                raise errors[name]
            return Handle(value)
        return invoke

    api = SimpleNamespace(
        stt=SimpleNamespace(engines=binding("engines", engines if engines is not None else [{"id": "local", "available": True, "nativeStreaming": True}])),
        health=SimpleNamespace(show=binding("health", health if health is not None else [{"capability": "CAPABILITY_STT", "effectiveState": "PROVIDER_STATE_AVAILABLE"}])),
        experiment=SimpleNamespace(
            list_evidence=binding("evidence", evidence if evidence is not None else []),
            report=binding("report", report)),
    )
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(SOURCE.read_text(), str(SOURCE), "exec"), {
            # Inject the declared runtime interface. Keep malformed values for
            # domain validation tests; this fixture does not certify the
            # kernel's separate input-schema gate or session lifecycle.
            "program": SimpleNamespace(
                inputs=lambda: {} if inputs is None else inputs,
                classify=program_helper.ProgramHelper().classify,
            ),
            "audio_tools": api,
            "gather": lambda *jobs: [job() for job in jobs],
        })
    lines = output.getvalue().splitlines()
    if len(lines) != 1:
        raise AssertionError("The board must print exactly one envelope")
    return json.loads(lines[0]), calls


def completed_report(**cell_fields):
    cell = {"engineId": "local", "strategy": "native", "refWords": 100,
            "wer": 0.12, "rtf": 0.4, "finalizationLatencyP95Ms": 800,
            "replayLane": "REPLAY_LANE_REALTIME"}
    cell.update(cell_fields)
    return {"experiment": {"id": "experiment-1", "status": "EXPERIMENT_STATUS_SUCCEEDED", "finishedAt": "2020-01-01T00:00:00Z"},
            "report": {"perStrategy": [cell], "latencyMeasured": True}}


class EvidenceBoardTests(unittest.TestCase):
    def test_every_published_target_remains_visible_in_the_board(self):
        # Structural traceability only; this does not validate product behavior.
        prd = (SOURCE.parents[2] / "PRD.md").read_text()
        target_ids = set(re.findall(r"^- \[[ x]\] (OT-P\d-\d+)", prd, re.MULTILINE))
        board, _ = run_board()
        exposed = set()
        for row in board["signals"]["rows"]:
            exposed.update(re.findall(r"OT-P\d-\d+", row["target"] or ""))
        self.assertTrue(target_ids)
        self.assertEqual(exposed, target_ids)

    def test_successful_inventory_keeps_every_outcome_unknown(self):
        board, calls = run_board()
        self.assertEqual(board["status"], "ok")
        self.assertEqual(board["signals"]["acceptance"], "unknown")
        self.assertEqual({c[0] for c in calls}, {"engines", "health", "evidence"})
        self.assertEqual(board["signals"]["target_revision"], "portable-voice-v1")
        self.assertEqual(len(board["signals"]["rows"]), 15)
        for row in board["signals"]["rows"]:
            if row["row"] in {"shared-voice-capabilities", "device-qualification"}:
                self.assertEqual(row["reason"], "excluded:mandate_scope")
                self.assertFalse(row["unavailable"])
            else:
                self.assertTrue(row["unavailable"])
                self.assertIsNone(row["reading"])
                self.assertIsNone(row["in_band"])
        self.assertIsNotNone(board["signals"]["rows"][0]["target"])
        self.assertTrue(all(row["target"] for row in board["signals"]["rows"]))

    def test_owner_receipt_join_resolves_all_required_rows_from_one_build(self):
        required = {
            "dictation-trust", "provider-parity", "speaker-policy", "provider-evidence",
            "mobile-recovery", "interactive-latency", "corpus-quality", "owned-settlement",
            "explicit-routing", "host-portability", "private-lifecycle", "acceptance-integrity",
            "adapter-maintainability",
        }
        evidence = [{
            "engineId": "local", "modelId": "model-1", "strategy": "native", "kind": "fault",
            "artifactRef": "qualification:portable-voice-v1:build-1", "passed": True,
            "machineJson": json.dumps({
                "host": "runner", "goos": "linux", "goarch": "amd64",
                "target_revision": "portable-voice-v1", "method": "audio-tools-acceptance-v1",
                "cohort": "voice-pilot-en-v1", "build": "build-1", "device_profile": "D1",
                "policy_revision": "policy-v1", "corpus_revision": "corpus-v1",
                "acceptance": {name: {"in_band": True, "negative_controls_passed": True} for name in required},
            }),
        }]
        board, _ = run_board(evidence=evidence)
        self.assertEqual(board["signals"]["acceptance"], "in_band")
        self.assertTrue(board["signals"]["acceptance_join"]["in_band"])
        self.assertEqual(board["signals"]["acceptance_join"]["resolved_rows"], len(required))
        for row in board["signals"]["rows"]:
            if row["row"] in required:
                self.assertTrue(row["in_band"], row)
                self.assertIsInstance(row["reading"], dict)
            else:
                self.assertEqual(row["reason"], "excluded:mandate_scope")

    def test_owner_receipt_join_rejects_stale_and_mixed_build_claims(self):
        claim = {"in_band": True, "negative_controls_passed": True}
        stale = {"target_revision": "portable-voice-v1", "method": "audio-tools-acceptance-v1",
                 "cohort": "c", "build": "old", "device_profile": "D1", "policy_revision": "p", "corpus_revision": "c1",
                 "stale": True, "acceptance": {"dictation-trust": claim}}
        mixed = {"target_revision": "portable-voice-v1", "method": "audio-tools-acceptance-v1",
                 "cohort": "c", "build": "new", "device_profile": "D1", "policy_revision": "p", "corpus_revision": "c1",
                 "acceptance": {"dictation-trust": claim}}
        evidence = []
        for i, machine in enumerate((stale, mixed)):
            evidence.append({"engineId": "local", "modelId": "m", "strategy": "s", "kind": "fault",
                              "artifactRef": f"a-{i}", "passed": True, "machineJson": json.dumps(machine)})
        board, _ = run_board(evidence=evidence)
        trust = next(item for item in board["signals"]["rows"] if item["row"] == "dictation-trust")
        self.assertTrue(trust["in_band"])
        self.assertEqual(trust["reading"]["build"], "new")
        self.assertFalse(board["signals"]["acceptance_join"]["in_band"])

    def test_simulated_receipt_cannot_certify_unmeasured_rows(self):
        required = {
            "dictation-trust", "provider-parity", "speaker-policy", "provider-evidence",
            "mobile-recovery", "interactive-latency", "corpus-quality", "owned-settlement",
            "explicit-routing", "host-portability", "private-lifecycle", "acceptance-integrity",
            "adapter-maintainability",
        }
        broad_simulation = {
            "target_revision": "portable-voice-v1", "method": "audio-tools-acceptance-v1",
            "cohort": "voice-pilot-en-v1", "build": "simulated-policy-v1", "device_profile": "SIMULATED",
            "policy_revision": "policy-v1", "corpus_revision": "voice-pilot-en-v1",
            "execution_lane": "simulated",
            "details": {"execution_lane": "simulated"},
            "acceptance": {name: {"in_band": True, "negative_controls_passed": True} for name in required},
        }
        board, _ = run_board(evidence=[{"engineId": "simulated", "modelId": "v1", "strategy": "simulation",
                                        "kind": "fault", "artifactRef": "simulated-broad", "passed": True,
                                        "machineJson": json.dumps(broad_simulation)}])
        self.assertEqual(board["signals"]["acceptance"], "unknown")
        self.assertEqual(board["signals"]["acceptance_join"]["resolved_rows"], 0)

    def test_simulated_receipt_resolves_only_the_rows_it_exercised(self):
        supported = {"explicit-routing", "owned-settlement", "private-lifecycle",
                     "acceptance-integrity", "adapter-maintainability"}
        receipt = {
            "target_revision": "portable-voice-v1", "method": "audio-tools-acceptance-v1",
            "cohort": "voice-pilot-en-v1", "build": "simulated-policy-v1", "device_profile": "D1",
            "policy_revision": "policy-v1", "corpus_revision": "voice-pilot-en-v1",
            "execution_lane": "simulated", "supported_rows": sorted(supported),
            "acceptance": {name: {"in_band": True, "negative_controls_passed": True}
                           for name in supported},
        }
        board, _ = run_board(evidence=[{"engineId": "simulated", "modelId": "v1", "strategy": "simulation",
                                        "kind": "fault", "artifactRef": "simulated-policy", "passed": True,
                                        "machineJson": json.dumps(receipt)}])
        self.assertEqual(board["signals"]["acceptance"], "unknown")
        self.assertEqual(board["signals"]["acceptance_join"]["resolved_rows"], len(supported))
        for item in board["signals"]["rows"]:
            if item["row"] in supported:
                self.assertTrue(item["in_band"], item)
            elif item["row"] not in {"shared-voice-capabilities", "device-qualification"}:
                self.assertIsNone(item["in_band"], item)

    def test_owner_receipts_with_mixed_context_cannot_be_combined(self):
        required = {
            "dictation-trust", "provider-parity", "speaker-policy", "provider-evidence",
            "mobile-recovery", "interactive-latency", "corpus-quality", "owned-settlement",
            "explicit-routing", "host-portability", "private-lifecycle", "acceptance-integrity",
            "adapter-maintainability",
        }
        def receipt(context, names):
            return json.dumps({"target_revision": "portable-voice-v1", "method": "audio-tools-acceptance-v1",
                               "cohort": "c", "build": "b", "device_profile": "D1",
                               "policy_revision": context, "corpus_revision": "corpus",
                               "acceptance": {name: {"in_band": True, "negative_controls_passed": True} for name in names}})
        evidence = [
            {"engineId": "local", "modelId": "m", "strategy": "s", "kind": "fault", "artifactRef": "a1",
             "passed": True, "machineJson": receipt("policy-a", required - {"corpus-quality"})},
            {"engineId": "local", "modelId": "m", "strategy": "s", "kind": "fault", "artifactRef": "a2",
             "passed": True, "machineJson": receipt("policy-b", {"corpus-quality"})},
        ]
        board, _ = run_board(evidence=evidence)
        self.assertFalse(board["signals"]["acceptance_join"]["in_band"])
        self.assertTrue(next(item for item in board["signals"]["rows"] if item["row"] == "corpus-quality")["in_band"])

    def test_out_of_band_owner_claim_is_visible_as_failure(self):
        evidence = [{"engineId": "local", "modelId": "m", "strategy": "s", "kind": "fault",
                     "artifactRef": "a", "passed": True, "machineJson": json.dumps({
                         "target_revision": "portable-voice-v1", "method": "audio-tools-acceptance-v1",
                         "cohort": "c", "build": "b", "device_profile": "D1", "policy_revision": "p",
                         "corpus_revision": "c1", "acceptance": {
                             "dictation-trust": {"in_band": False, "negative_controls_passed": True}}})}]
        board, _ = run_board(evidence=evidence)
        trust = next(item for item in board["signals"]["rows"] if item["row"] == "dictation-trust")
        self.assertEqual(trust["reason"], "failed:receipt_claim_out_of_band")

    def test_failed_target_aware_receipt_is_not_ignored_by_the_join(self):
        evidence = [{"engineId": "local", "modelId": "m", "strategy": "s", "kind": "fault",
                     "artifactRef": "failed-artifact", "passed": False, "machineJson": json.dumps({
                         "target_revision": "portable-voice-v1", "method": "audio-tools-acceptance-v1",
                         "cohort": "c", "build": "b", "device_profile": "D1", "policy_revision": "p",
                         "corpus_revision": "c1", "acceptance": {
                             "dictation-trust": {"in_band": True, "negative_controls_passed": True}}})}]
        board, _ = run_board(evidence=evidence)
        trust = next(item for item in board["signals"]["rows"] if item["row"] == "dictation-trust")
        self.assertEqual(trust["reason"], "failed:receipt_claim_out_of_band")
        self.assertEqual(board["signals"]["acceptance_join"]["resolved_rows"], 0)

    def test_repeated_evidence_rows_use_raw_response_projection(self):
        evidence = [{"engineId": "local", "modelId": "m", "strategy": "s", "kind": "fault",
                     "artifactRef": "artifact", "passed": False}]
        board, calls = run_board(evidence=evidence)
        self.assertEqual(board["signals"]["evidence_integrity"]["receipt_count"], 1)
        self.assertIn(("evidence", {}), calls)

    def test_acceptance_claim_after_legacy_history_is_not_lost_at_old_bound(self):
        legacy = [{"engineId": "local", "modelId": "m", "strategy": "s", "kind": "fault",
                   "artifactRef": f"legacy-{i}", "passed": True} for i in range(64)]
        required = {
            "dictation-trust", "provider-parity", "speaker-policy", "provider-evidence",
            "mobile-recovery", "interactive-latency", "corpus-quality", "owned-settlement",
            "explicit-routing", "host-portability", "private-lifecycle", "acceptance-integrity",
            "adapter-maintainability",
        }
        receipt = {"target_revision": "portable-voice-v1", "method": "audio-tools-acceptance-v1",
                   "cohort": "c", "build": "b", "device_profile": "D1", "policy_revision": "p",
                   "corpus_revision": "c1", "acceptance": {
                       name: {"in_band": True, "negative_controls_passed": True} for name in required}}
        legacy.append({"engineId": "local", "modelId": "m", "strategy": "s", "kind": "fault",
                       "artifactRef": "current", "passed": True, "machineJson": json.dumps(receipt)})
        board, _ = run_board(evidence=legacy)
        self.assertEqual(board["signals"]["acceptance"], "in_band")
        self.assertEqual(board["signals"]["acceptance_join"]["inspected_receipts"], 65)

    def test_invalid_input_never_invokes_bindings(self):
        for values in ({"experiment_id": "../secret"}, {"experiment_id": 4}, {"experiment_id": "x" * 129}, {"unknown": True}, []):
            with self.subTest(values=values):
                board, calls = run_board(inputs=values)
                self.assertEqual(board["status"], "failed")
                self.assertEqual(board["errors"][0]["class"], "invalid_input")
                self.assertEqual(calls, [])

    def test_empty_owner_receipts_remain_unqualified(self):
        board, _ = run_board(evidence=[])
        self.assertEqual(board["signals"]["evidence_integrity"]["receipt_count"], 0)
        self.assertFalse(board["signals"]["evidence_integrity"]["in_band"])
        self.assertIn("no_owner_receipts", board["signals"]["evidence_integrity"]["reason"])

    def test_missing_repeated_evidence_field_is_not_coerced_to_empty_telemetry(self):
        board, _ = run_board(evidence={"metadata": {"source": "owner"}})
        integrity = board["signals"]["evidence_integrity"]
        self.assertIsNone(integrity["receipt_count"])
        self.assertIn("shape_or_bound", integrity["reason"])
        self.assertFalse(integrity["in_band"])
        self.assertTrue(all(item["in_band"] is None for item in board["signals"]["rows"]
                            if item["reason"] != "excluded:mandate_scope"))

    def test_failed_owner_receipt_is_negative_evidence(self):
        evidence = [{"engineId": "local", "modelId": "model-1", "strategy": "native",
                     "kind": "fault", "artifactRef": "bas:run-1", "passed": False}]
        board, _ = run_board(evidence=evidence)
        integrity = board["signals"]["evidence_integrity"]
        self.assertEqual(integrity["failed_receipts"], 1)
        self.assertFalse(integrity["in_band"])
        self.assertIn("negative_control", integrity["reason"])

    def test_retained_failed_attempt_does_not_corrupt_valid_owner_audit(self):
        positive = {"engineId": "local", "modelId": "model-1", "strategy": "native",
                    "kind": "fault", "artifactRef": "good-artifact", "passed": True}
        failed = {"engineId": "local", "modelId": "model-1", "strategy": "native",
                  "kind": "fault", "artifactRef": "failed-artifact", "passed": False}
        board, _ = run_board(evidence=[positive, failed])
        integrity = board["signals"]["evidence_integrity"]
        self.assertTrue(integrity["in_band"])
        self.assertEqual(integrity["failed_receipts"], 1)
        self.assertEqual(integrity["reason"], "retained_failed_attempts")

    def test_partial_failure_preserves_other_measurements_and_redacts_error(self):
        board, _ = run_board(errors={"engines": RuntimeError("connection refused secret-transcript")})
        self.assertEqual(board["status"], "partial")
        self.assertEqual(board["errors"][0]["class"], "scenario_unreachable")
        self.assertEqual(board["signals"]["diagnostics"][-1]["reading"], 1)
        self.assertNotIn("secret-transcript", json.dumps(board))

    def test_all_failed_reads_are_unavailable_not_zero(self):
        board, _ = run_board(errors={key: RuntimeError("connection refused") for key in ("engines", "health")})
        self.assertEqual(board["status"], "unavailable")
        self.assertEqual(board["signals"]["readable"], 0)
        self.assertTrue(all(d["reading"] is None for d in board["signals"]["diagnostics"]))

    def test_empty_inventory_is_a_measured_zero_not_acceptance(self):
        board, _ = run_board(engines=[], health=[])
        self.assertEqual(board["signals"]["diagnostics"][0]["reading"], 0)
        self.assertIsNone(board["signals"]["diagnostics"][0]["in_band"])

    def test_oversized_inventory_does_not_claim_exact_count(self):
        board, _ = run_board(engines=[{"id": str(i), "available": True} for i in range(17)])
        self.assertIsNone(board["signals"]["diagnostics"][0]["reading"])
        self.assertIn("bound", board["signals"]["diagnostics"][0]["reason"])

    def test_selected_old_report_is_diagnostic_and_never_browser_latency(self):
        report = completed_report()
        report["experiment"]["recipe"] = {"realizedReference": "private speech"}
        board, calls = run_board(inputs={"experiment_id": "experiment-1"}, report=report)
        self.assertIn(("report", {"id": "experiment-1"}), calls)
        self.assertEqual(board["signals"]["replay_metrics"][0]["wer"], 0.12)
        self.assertEqual(board["signals"]["replay_metrics"][0]["replay_finalization_p95_ms"], 800)
        self.assertFalse(board["signals"]["experiment"]["acceptance_eligible"])
        self.assertEqual(board["signals"]["acceptance"], "unknown")
        self.assertNotIn("private speech", json.dumps(board))

    def test_failed_canceled_or_wrong_experiment_cannot_supply_quality(self):
        for status, identity in (("EXPERIMENT_STATUS_FAILED", "experiment-1"), ("EXPERIMENT_STATUS_CANCELED", "experiment-1"), ("EXPERIMENT_STATUS_SUCCEEDED", "wrong")):
            report = completed_report()
            report["experiment"].update(status=status, id=identity)
            board, _ = run_board(inputs={"experiment_id": "experiment-1"}, report=report)
            self.assertEqual(board["signals"]["replay_metrics"], [])
            self.assertFalse(board["signals"]["experiment"]["acceptance_eligible"])

    def test_missing_denominator_nan_and_absent_metrics_stay_unknown(self):
        for fields in ({"refWords": 0}, {"refWords": None}, {"wer": float("nan")}, {"wer": None}):
            board, _ = run_board(inputs={"experiment_id": "experiment-1"}, report=completed_report(**fields))
            self.assertIsNone(board["signals"]["replay_metrics"][0]["wer"])
        report = completed_report()
        del report["report"]["perStrategy"][0]["wer"]
        board, _ = run_board(inputs={"experiment_id": "experiment-1"}, report=report)
        self.assertIsNone(board["signals"]["replay_metrics"][0]["wer"])

    def test_measured_zero_wer_is_preserved_and_wer_above_one_not_clamped(self):
        for wer in (0, 1.5):
            board, _ = run_board(inputs={"experiment_id": "experiment-1"}, report=completed_report(wer=wer))
            self.assertEqual(board["signals"]["replay_metrics"][0]["wer"], wer)

    def test_unmeasured_latency_is_not_imputed_from_quality(self):
        report = completed_report()
        report["report"]["latencyMeasured"] = False
        board, _ = run_board(inputs={"experiment_id": "experiment-1"}, report=report)
        self.assertIsNone(board["signals"]["replay_metrics"][0]["replay_finalization_p95_ms"])

    def test_failed_child_report_does_not_inherit_inventory_success(self):
        board, _ = run_board(inputs={"experiment_id": "experiment-1"}, errors={"report": RuntimeError("deadline exceeded")})
        self.assertEqual(board["status"], "partial")
        self.assertEqual(board["signals"]["replay_metrics"], [])
        self.assertEqual(board["errors"][0]["class"], "deadline_exceeded")

    def test_successful_job_preserves_failing_cell_verdict(self):
        board, _ = run_board(inputs={"experiment_id": "experiment-1"}, report=completed_report(verdict="quarantined"))
        self.assertEqual(board["signals"]["replay_metrics"][0]["verdict"], "quarantined")
        self.assertEqual(board["signals"]["acceptance"], "unknown")

    def test_programming_error_is_not_reported_as_down_scenario(self):
        board, _ = run_board(errors={"engines": AttributeError("wrong binding")})
        self.assertEqual(board["status"], "failed")
        self.assertEqual(board["errors"][0]["class"], "kernel_runtime")


if __name__ == "__main__":
    unittest.main()
