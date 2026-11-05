"""Failure-triage keeps shape evidence linked and reports incomplete summaries."""
import contextlib
import io
import json
from pathlib import Path
from types import SimpleNamespace as NS
import unittest

ROOT = Path(__file__).parents[1]


class Handle:
    def __init__(self, rows, metadata):
        self.rows = rows
        self.metadata = metadata

    def count(self):
        return len(self.rows)

    def head(self, limit):
        return self.rows[:limit]

    def meta(self):
        return dict(self.metadata)


class FailureTriageTests(unittest.TestCase):
    def run_program(self, rows, metadata):
        output = io.StringIO()
        handle = Handle(rows, metadata)

        def drive(states, state):
            while state:
                state = states[state]()

        program = NS(
            inputs=lambda: {"include_operator": False},
            fail=lambda *args: self.fail(f"unexpected failure path: {args}"),
            run=drive,
        )
        runtime = NS(programs=NS(mine=lambda **_: handle))
        with contextlib.redirect_stdout(output):
            exec(compile((ROOT / "failure-triage.py").read_text(), "failure-triage", "exec"),
                 {"program": program, "program_runtime": runtime})
        return json.loads(output.getvalue())

    @staticmethod
    def row(shape, count, sample, first, last):
        return {"shape": shape, "count": str(count), "sampleProgramId": sample,
                "firstSeen": first, "lastSeen": last}

    def test_top_evidence_is_shape_keyed_and_keeps_time_context(self):
        result = self.run_program([
            self.row("ambiguous_response", 24, "prog-ambiguous", "2026-01-01", "2026-02-01"),
            self.row("bridge_transport", 4, "prog-bridge", "2026-01-02", "2026-02-02"),
            self.row("kernel_runtime", 202, "prog-kernel", "2026-01-03", "2026-02-03"),
            self.row("unresolved_name", 27, "prog-name", "2026-01-04", "2026-02-04"),
        ], {"count": "4"})

        signals = result["signals"]
        self.assertEqual("ok", result["status"])
        self.assertTrue(signals["complete"])
        self.assertEqual(["kernel_runtime", "unresolved_name", "ambiguous_response"], signals["top"])
        self.assertEqual({"kernel_runtime": "prog-kernel", "unresolved_name": "prog-name",
                          "ambiguous_response": "prog-ambiguous"}, signals["evidence_by_shape"])
        self.assertEqual(["prog-kernel", "prog-name", "prog-ambiguous"], result["evidence"])
        self.assertEqual({"count": 202, "first_seen": "2026-01-03", "last_seen": "2026-02-03",
                          "sample_program_id": "prog-kernel"}, signals["shape_details"]["kernel_runtime"])

    def test_more_than_twenty_rows_is_partial_and_marks_truncation(self):
        rows = [self.row(f"failure_shape_{i:02d}", 100 - i, f"prog_{i:032x}-0000-0000-0000-000000000000",
                         "2026-10-01T12:34:56.123456789Z", "2026-10-29T12:34:56.123456789Z")
                for i in range(21)]
        result = self.run_program(rows, {"count": "21"})

        self.assertEqual("partial", result["status"])
        self.assertFalse(result["signals"]["complete"])
        self.assertTrue(result["signals"]["truncated"])
        self.assertEqual(21, result["signals"]["shapes"])
        self.assertEqual(20, result["signals"]["returned_shapes"])
        self.assertIn("shape_rows_truncated", [error["class"] for error in result["errors"]])
        self.assertLess(len(json.dumps(result).encode("utf-8")), 65536,
                        "the bounded summary must fit the declared 64 KB output tier")

    def test_source_count_mismatch_is_partial(self):
        result = self.run_program([
            self.row("kernel_runtime", 3, "prog-k", "first", "last"),
            self.row("unresolved_name", 2, "prog-u", "first", "last"),
        ], {"count": "3"})

        self.assertEqual("partial", result["status"])
        self.assertFalse(result["signals"]["complete"])
        self.assertIn("source_count_mismatch", [error["class"] for error in result["errors"]])

    def test_missing_source_count_is_partial(self):
        result = self.run_program([self.row("kernel_runtime", 3, "prog-k", "first", "last")], {})

        self.assertEqual("partial", result["status"])
        self.assertFalse(result["signals"]["complete"])
        self.assertIn("source_count_unavailable", [error["class"] for error in result["errors"]])

    def test_missing_ranked_sample_keeps_summary_partial(self):
        result = self.run_program([
            self.row("kernel_runtime", 3, "", "first", "last"),
            self.row("unresolved_name", 2, "prog-u", "first", "last"),
        ], {"count": "2"})

        self.assertEqual("partial", result["status"])
        self.assertFalse(result["signals"]["complete"])
        self.assertNotIn("kernel_runtime", result["signals"]["evidence_by_shape"])
        self.assertIn("shape_evidence_missing", [error["class"] for error in result["errors"]])

    def test_non_integer_source_count_is_not_coerced(self):
        result = self.run_program([self.row("kernel_runtime", 3, "prog-k", "first", "last")],
                                  {"count": 1.5})

        self.assertEqual("partial", result["status"])
        self.assertFalse(result["signals"]["complete"])
        self.assertIn("source_count_unavailable", [error["class"] for error in result["errors"]])

    def test_contract(self):
        import jsonschema
        contract = json.loads((ROOT / "failure-triage.json").read_text())
        schema = json.loads((ROOT.parents[1] / "schemas/program-contract.schema.json").read_text())
        jsonschema.validate(contract, schema)
        self.assertIn("partial", contract["outputs"]["status"]["enum"])


if __name__ == "__main__":
    unittest.main()
