import contextlib
import io
import json
import unittest
from pathlib import Path
from types import SimpleNamespace


SOURCE = Path(__file__).resolve().parents[2] / ".vrooli/program-runtime/concurrent-fanout.py"


class Handle:
    def __init__(self, value):
        self.value = value

    def count(self):
        if isinstance(self.value, Exception):
            raise self.value
        return self.value


class Program:
    def inputs(self):
        return {}

    def fail(self, *args):
        raise AssertionError(args)

    def classify(self, exc):
        return "unavailable", "binding_error"

    def run(self, states, state):
        while state is not None:
            state = states[state]()


def run_fanout(values, gather_transform=lambda results: results):
    def binding(value):
        def call(**_kwargs):
            if isinstance(value, Exception):
                raise value
            return value if isinstance(value, Handle) else Handle(value)
        return call

    namespace = {
        "program": Program(),
        "gather": lambda *calls: gather_transform([call() for call in calls]),
        "agent_manager": SimpleNamespace(measures=SimpleNamespace(run_volume=binding(values[0]))),
        "ai_gateway": SimpleNamespace(measures=SimpleNamespace(total=binding(values[1]))),
        "program_runtime": SimpleNamespace(programs=SimpleNamespace(mine=binding(values[2]))),
    }
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(SOURCE.read_text(), str(SOURCE), "exec"), namespace)
    return json.loads(output.getvalue())


class ConcurrentFanoutTest(unittest.TestCase):
    def test_middle_failure_keeps_later_result_in_its_position(self):
        result = run_fanout([2, RuntimeError("unavailable"), 7])
        self.assertEqual(result["status"], "partial")
        self.assertEqual(result["signals"]["result_counts"], [2, None, 7])
        outcomes = result["signals"]["result_outcomes"]
        self.assertEqual([o["binding"] for o in outcomes], [
            "agent-manager/measures/run-volume", "ai-gateway/measures/total", "program-runtime/programs/mine"
        ])
        self.assertEqual(outcomes[1]["error_class"], "binding_error")

    def test_all_failures_are_unavailable_and_keep_three_slots(self):
        result = run_fanout([RuntimeError("a"), RuntimeError("b"), RuntimeError("c")])
        self.assertEqual(result["status"], "unavailable")
        self.assertEqual(result["signals"]["result_counts"], [None, None, None])
        self.assertEqual(len(result["errors"]), 3)

    def test_count_failure_is_kept_in_its_position(self):
        # The handle itself succeeds but its deferred count fails.
        result = run_fanout([2, Handle(RuntimeError("count failed")), 7])
        self.assertEqual(result["status"], "partial")
        self.assertEqual(result["signals"]["result_counts"], [2, None, 7])
        self.assertEqual(result["errors"][0]["where"], "classify")

    def test_missing_or_extra_gather_results_are_not_success(self):
        missing = run_fanout([2, 3, 7], lambda results: results[:2])
        self.assertEqual(missing["status"], "partial")
        self.assertEqual(missing["signals"]["result_counts"], [2, 3, None])
        extra = run_fanout([2, 3, 7], lambda results: results + [Handle(9)])
        self.assertEqual(extra["status"], "partial")
        self.assertEqual(extra["signals"]["result_counts"], [2, 3, 7])
        self.assertEqual(extra["errors"][0]["class"], "ambiguous_response")


if __name__ == "__main__":
    unittest.main()
