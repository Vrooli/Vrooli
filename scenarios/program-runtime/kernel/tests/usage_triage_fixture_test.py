"""Deterministic owner-boundary tests for usage triage and portfolio outages."""
import contextlib
import io
import json
from pathlib import Path
import sys
from types import SimpleNamespace as NS
import unittest

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "kernel" / "host"))
from program_helper import ProgramHelper


class Handle:
    def __init__(self, rows=(), meta=None):
        self.rows, self.metadata = list(rows), meta or {}

    def head(self, limit):
        return self.rows[:limit]

    def meta(self):
        return self.metadata


def execute(source, env):
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(source.read_text(), str(source), "exec"), env)
    return json.loads(output.getvalue())


def unavailable(**_):
    raise RuntimeError("connection refused")


class FixtureContracts(unittest.TestCase):
    def usage(self, *, label="helped", classifier_down=False, agent_down=False, max_pairs=32, candidates=3):
        rows = [{"programName": f"demo.program{i}", "callerRunId": "" if agent_down else f"run{i}", "status": "FAILED"} for i in range(candidates)]
        portfolio = [{"name": f"demo.program{i}", "runs": 3, "successRate": 0.5} for i in range(candidates)]
        calls = []

        def batch(corpus, *_args, **_kwargs):
            calls.append(list(corpus))
            if classifier_down:
                raise RuntimeError("connection refused")
            return Handle([{"results": [{"validated": True, "valueJson": json.dumps(label)} for _ in corpus]}])

        def search(**_kwargs):
            return Handle([{"runId": "matched"}] * 3)

        env = {
            "inputs": {"window_days": 30, "max_pairs": max_pairs},
            "program": ProgramHelper(),
            "program_runtime": NS(programs=NS(portfolio=lambda **_: Handle(portfolio), list=lambda **_: Handle(rows))),
            "agent_manager": NS(run=NS(list=unavailable if agent_down else lambda **_: Handle()), conversation=NS(search=search)),
            "ai": NS(batch=batch), "gather": lambda *operations: [op() for op in operations],
        }
        result = execute(ROOT / ".vrooli/program-runtime/usage-triage.py", env)
        self.assertEqual(len(calls), 1)
        return result

    def test_fought_it_candidate_retains_attribution_and_excludes_helped(self):
        result = self.usage(label="fought_it", candidates=1)
        self.assertEqual(result["status"], "ok")
        self.assertEqual(result["signals"]["labels"], ["fought_it"])
        self.assertEqual(result["signals"]["cohorts"][0]["exemplar_run_ids"], ["run0"])
        self.assertEqual(result["signals"]["inference_calls"], 1)
        helped = self.usage(label="helped", candidates=1)
        self.assertEqual(helped["signals"]["cohorts"], [])

    def test_classifier_outage_never_invents_labels(self):
        result = self.usage(classifier_down=True)
        self.assertEqual(result["status"], "unavailable")
        self.assertEqual(result["signals"]["pairs_classified"], 0)
        self.assertEqual(result["signals"]["cohorts"], [])
        self.assertEqual(result["errors"][-1]["class"], "scenario_unreachable")

    def test_agent_run_list_outage_preserves_transcript_match_as_partial(self):
        result = self.usage(agent_down=True, candidates=1)
        self.assertEqual(result["status"], "partial")
        self.assertEqual(result["signals"]["attribution_basis"], "transcript_match")
        self.assertEqual(result["signals"]["pairs_classified"], 1)

    def test_candidate_cap_is_explicit(self):
        result = self.usage(max_pairs=2, candidates=3)
        self.assertEqual(result["signals"]["candidates_considered"], 3)
        self.assertTrue(result["signals"]["candidates_capped"])
        self.assertEqual(result["signals"]["pairs_classified"], 2)

    def test_portfolio_outage_keeps_budget_measurements_unknown(self):
        env = {"inputs": {"min_rung": "S4", "window_days": 1}, "program_runtime": NS(
            library=NS(list=lambda **_: Handle([{"kind": "contract", "rung": "S4", "name": "demo.p", "source": "present"}])),
            programs=NS(portfolio=unavailable))}
        result = execute(ROOT / ".vrooli/program-runtime/portfolio-audit.py", env)
        self.assertEqual(result["status"], "partial")
        for dimension in ("budget_realism", "exercised"):
            self.assertIsNone(result["signals"][dimension]["ratio"])
            self.assertIsNone(result["signals"][dimension]["count"])
            self.assertTrue(result["signals"][dimension]["unavailable"])


if __name__ == "__main__":
    unittest.main()
