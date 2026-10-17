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


def run_board(inputs=None, engines=None, health=None, report=None, errors=None):
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
        experiment=SimpleNamespace(report=binding("report", report)),
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
        self.assertEqual({c[0] for c in calls}, {"engines", "health"})
        self.assertEqual(board["signals"]["target_revision"], "portable-voice-v1")
        self.assertEqual(len(board["signals"]["rows"]), 15)
        for row in board["signals"]["rows"]:
            self.assertTrue(row["unavailable"])
            self.assertIsNone(row["reading"])
            self.assertIsNone(row["in_band"])
        self.assertIsNotNone(board["signals"]["rows"][0]["target"])
        self.assertTrue(all(row["target"] for row in board["signals"]["rows"]))

    def test_invalid_input_never_invokes_bindings(self):
        for values in ({"experiment_id": "../secret"}, {"experiment_id": 4}, {"experiment_id": "x" * 129}, {"unknown": True}, []):
            with self.subTest(values=values):
                board, calls = run_board(inputs=values)
                self.assertEqual(board["status"], "failed")
                self.assertEqual(board["errors"][0]["class"], "invalid_input")
                self.assertEqual(calls, [])

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
