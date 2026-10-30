import ast
import json
import unittest
from pathlib import Path


SOURCE = Path(__file__).parents[1] / "provider-quality-read.py"
REPORT_BUDGET = 3800


class FakeProgram:
    def __init__(self):
        self.reports = []

    def report(self, envelope):
        self.reports.append(envelope)


def load_report_helpers():
    source = ast.parse(SOURCE.read_text())
    wanted = {"_bounded_text", "_compact_status", "_compact_report", "report_bounded"}
    functions = [node for node in source.body if isinstance(node, ast.FunctionDef) and node.name in wanted]
    found = {node.name for node in functions}
    if found != wanted:
        raise AssertionError(f"missing report helper(s): {sorted(wanted - found)}")
    namespace = {
        "json": json,
        "REPORT_OUTPUT_BUDGET_BYTES": REPORT_BUDGET,
        "program": FakeProgram(),
    }
    exec(compile(ast.Module(body=functions, type_ignores=[]), str(SOURCE), "exec"), namespace)
    return namespace


class ProviderQualityReportBoundsTest(unittest.TestCase):
    def test_oversized_inputs_and_details_use_bounded_status_preserving_fallback(self):
        runtime = load_report_helpers()
        envelope = {
            "program": "search-hub.provider-quality-read",
            "version": "1",
            "status": "ok",
            "phase": "report",
            "inputs": {
                "insights_window": "7",
                "provider_group": "web-search" * 2000,
                "max_rows": 100,
            },
            "signals": {
                "provider_count": 100,
                "providers": [{"provider_id": f"provider-{i}-" + "x" * 2000} for i in range(100)],
                "unreachable": [{"provider_id": "owner-a", "reachability": "resolver error " + "x" * 12000}],
                "degraded_over_band": [],
                "under_utilized": [],
                "suites_without_runs": [],
                "suites_below_full_pass": [],
            },
            "errors": [{"class": "remote_error", "where": "collect", "detail": "x" * 20000} for _ in range(60)],
            "evidence": ["search-hub://eval-run/" + "x" * 1000 for _ in range(20)],
        }

        runtime["report_bounded"](envelope)

        self.assertEqual(len(runtime["program"].reports), 1)
        report = runtime["program"].reports[0]
        self.assertEqual(report["status"], "partial")
        self.assertTrue(report["signals"]["report_compacted"])
        self.assertEqual(report["signals"]["provider_count"], 100)
        self.assertEqual(report["signals"]["unreachable_count"], 1)
        self.assertLessEqual(len(report["inputs"]["provider_group"]), 80)
        self.assertEqual(report["errors"][0]["class"], "remote_error")
        self.assertEqual(report["signals"]["errors_count"], 60)
        self.assertEqual(report["signals"]["evidence_count"], 20)
        self.assertLessEqual(len(json.dumps(report, allow_nan=False).encode()), REPORT_BUDGET)

    def test_oversized_failure_keeps_original_failure_and_error_class(self):
        runtime = load_report_helpers()
        envelope = {
            "program": "search-hub.provider-quality-read",
            "version": "1",
            "status": "failed",
            "phase": "validate",
            "inputs": {"insights_window": "7", "provider_group": "g" * 10000, "max_rows": 101},
            "signals": {},
            "errors": [{"class": "invalid_input", "where": "validate", "detail": "bad input " + "x" * 12000}],
            "evidence": [],
        }

        runtime["report_bounded"](envelope)

        report = runtime["program"].reports[0]
        self.assertEqual(report["status"], "failed")
        self.assertEqual(report["errors"][0]["class"], "invalid_input")
        self.assertEqual(report["errors"][0]["where"], "validate")
        self.assertTrue(report["signals"]["report_compacted"])
        self.assertEqual(report["inputs"]["max_rows"], 101)
        self.assertLessEqual(len(json.dumps(report, allow_nan=False).encode()), REPORT_BUDGET)

    def test_small_report_is_passed_through_unchanged(self):
        runtime = load_report_helpers()
        envelope = {"program": "search-hub.provider-quality-read", "status": "ok", "errors": []}

        runtime["report_bounded"](envelope)

        self.assertIs(runtime["program"].reports[0], envelope)
        self.assertEqual(envelope["status"], "ok")
        self.assertNotIn("report_compacted", envelope)

    def test_compacted_non_success_statuses_are_preserved(self):
        for status in ("partial", "refused", "unavailable"):
            with self.subTest(status=status):
                runtime = load_report_helpers()
                envelope = {
                    "program": "search-hub.provider-quality-read",
                    "status": status,
                    "inputs": {"provider_group": "x" * 10000},
                    "signals": {},
                    "errors": [],
                }

                runtime["report_bounded"](envelope)

                self.assertEqual(runtime["program"].reports[0]["status"], status)

    def test_report_phase_uses_bounded_reporter(self):
        source = ast.parse(SOURCE.read_text())
        report = next(node for node in source.body if isinstance(node, ast.FunctionDef) and node.name == "step_report")
        calls = [node for node in ast.walk(report) if isinstance(node, ast.Call) and isinstance(node.func, ast.Name)]
        self.assertTrue(any(call.func.id == "report_bounded" for call in calls))


if __name__ == "__main__":
    unittest.main()
