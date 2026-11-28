"""Reader contract regressions, not qualification of the proposed product."""

import contextlib
import importlib.util
import io
import json
from pathlib import Path
import re
from types import SimpleNamespace
import unittest


SCENARIO = Path(__file__).resolve().parents[1]
SOURCE = SCENARIO / ".vrooli/program-runtime/setpoint-read.py"
HELPER = SCENARIO.parent / "program-runtime/kernel/host/program_helper.py"
spec = importlib.util.spec_from_file_location("ttd_program_helper", HELPER)
helper = importlib.util.module_from_spec(spec)
spec.loader.exec_module(helper)
CHILD_ROWS = ["registered-under-scenario-pack", "usage-id-present", "improve-id-present",
              "set-token-size", "read-counts", "programs-declared", "programs-named-in-usage-skill"]


class Handle:
    def __init__(self, rows=None, meta=None):
        self.rows = rows or []
        self.metadata = meta or {}

    def head(self, limit):
        return self.rows[:limit]

    def count(self):
        return len(self.rows)

    def meta(self):
        return self.metadata


def healthy_child():
    return {"program": "prompt-manager.skill-set-read", "version": "1", "status": "ok",
            "errors": [], "signals": {"usage_present": True, "improve_present": True,
            "rows": [{"row": row, "reading": 0, "target": None, "in_band": None,
                      "unavailable": False, "reason": None} for row in CHILD_ROWS]}}


def run_board(inputs=None, child=None, ontology=None, errors=None):
    calls = []
    errors = errors or {}

    def binding(name, result):
        def invoke(**kwargs):
            calls.append((name, kwargs))
            if name in errors:
                raise errors[name]
            return result
        return invoke

    runtime_helper = helper.ProgramHelper()
    environment = {
        "inputs": inputs or {}, "program": runtime_helper,
        "gather": lambda *jobs: [job() for job in jobs],
        "tech_tree_designer": SimpleNamespace(
            plan=SimpleNamespace(list=binding("plans", Handle())),
            ontology=SimpleNamespace(coverage=binding("ontology", Handle(meta=ontology)))),
        "lib": SimpleNamespace(prompt_manager=SimpleNamespace(skill_set_read=binding(
            "child", Handle([healthy_child() if child is None else child], {"digest": "fixture-digest"})))),
    }
    runtime_helper._bind(environment)
    stdout = io.StringIO()
    with contextlib.redirect_stdout(stdout):
        exec(compile(SOURCE.read_text(), str(SOURCE), "exec"), environment)
    lines = stdout.getvalue().splitlines()
    if len(lines) != 1:
        raise AssertionError("one JSON envelope must be emitted")
    if len(stdout.getvalue().encode()) > 65536:
        raise AssertionError("declared output bound exceeded")
    return json.loads(lines[0]), calls


class SetpointReadTests(unittest.TestCase):
    def assert_unknown(self, board):
        self.assertEqual(board["signals"]["acceptance"], "unknown")
        self.assertEqual(board["signals"]["unresolved_outcomes"], 18)
        for row in board["signals"]["rows"]:
            self.assertIsNone(row["reading"])
            self.assertIsNone(row["target"])
            self.assertIsNone(row["in_band"])
            self.assertTrue(row["unavailable"])
            self.assertEqual(row["reason"], "pending_telemetry")

    def test_every_published_outcome_survives_and_is_never_accepted(self):
        board, calls = run_board()
        published = re.findall(r"^- \[[ x]\] (OT-P\d-\d+)", (SCENARIO / "PRD.md").read_text(), re.M)
        self.assertEqual([row["row"] for row in board["signals"]["rows"]], published)
        self.assertEqual(board["status"], "ok")
        self.assertEqual(len(calls), 3)
        self.assert_unknown(board)

    def test_inventory_only_does_not_call_owners(self):
        board, calls = run_board({"collect_diagnostics": False})
        self.assertEqual(calls, [])
        self.assertEqual(board["status"], "ok")
        self.assertEqual(board["signals"]["diagnostics"], [])
        self.assert_unknown(board)

    def test_domain_rejects_non_boolean_before_owner_reads(self):
        for bad in ["false", 0, [], {}]:
            with self.subTest(bad=bad):
                board, calls = run_board({"collect_diagnostics": bad})
                self.assertEqual(board["status"], "failed")
                self.assertEqual(board["errors"][0]["class"], "invalid_input")
                self.assertEqual(calls, [])
                self.assert_unknown(board)

    def test_protojson_zero_is_a_valid_unbanded_diagnostic(self):
        board, _ = run_board(ontology={})
        row = board["signals"]["diagnostics"][1]
        self.assertEqual(row["reading"], {"capabilities": 0, "scenarios": 0})
        self.assertFalse(row["unavailable"])
        self.assertIsNone(row["in_band"])

    def test_unavailable_owner_preserves_other_reads(self):
        board, _ = run_board(errors={"plans": RuntimeError("scenario is unreachable")})
        self.assertEqual(board["status"], "partial")
        rows = board["signals"]["diagnostics"]
        self.assertEqual(rows[0]["reason"], "scenario_unreachable")
        self.assertIsNone(rows[0]["reading"])
        self.assertFalse(rows[1]["unavailable"])
        self.assertFalse(rows[2]["unavailable"])
        self.assert_unknown(board)

    def test_stale_or_failed_source_cannot_be_healthy_empty_ontology(self):
        board, _ = run_board(ontology={"graphError": "source unavailable", "totalCapabilities": 5})
        self.assertEqual(board["status"], "partial")
        self.assertIsNone(board["signals"]["diagnostics"][1]["reading"])
        self.assertEqual(board["signals"]["diagnostics"][1]["reason"], "unreliable:graph_source")

    def test_failed_partial_missing_or_wrong_child_is_not_success(self):
        children = [{}, {**healthy_child(), "status": "failed"},
                    {**healthy_child(), "status": "partial"},
                    {**healthy_child(), "program": "unrelated"},
                    {**healthy_child(), "version": "2"}]
        for child in children:
            with self.subTest(child=child.get("status")):
                board, _ = run_board(child=child)
                self.assertEqual(board["status"], "partial")
                self.assertIsNone(board["signals"]["diagnostics"][2]["reading"])
                self.assert_unknown(board)

    def test_ok_child_with_missing_or_unknown_or_duplicate_rows_is_incomplete(self):
        for mutation in ["missing", "unknown", "duplicate", "no-reading", "stale"]:
            child = healthy_child()
            rows = child["signals"]["rows"]
            if mutation == "missing":
                rows.pop()
            elif mutation == "unknown":
                rows[0]["unavailable"] = True
            elif mutation == "duplicate":
                rows[1]["row"] = rows[0]["row"]
            elif mutation == "no-reading":
                rows[0]["reading"] = None
            else:
                rows[0]["reason"] = "unreliable:stale"
            with self.subTest(mutation=mutation):
                board, _ = run_board(child=child)
                self.assertEqual(board["status"], "partial")
                self.assert_unknown(board)

    def test_kernel_failure_retains_inventory_and_reports_once(self):
        board, _ = run_board(errors={"plans": NameError("missing runtime name")})
        self.assertEqual(board["status"], "failed")
        self.assertEqual(board["errors"][0]["class"], "kernel_runtime")
        self.assert_unknown(board)

    def test_proposals_cannot_supply_approval_or_measurement(self):
        board, _ = run_board({"collect_diagnostics": False, "approved": True,
                              "target": 0, "receipt": {"status": "passed"}})
        self.assert_unknown(board)


if __name__ == "__main__":
    unittest.main()
