import contextlib
import io
import json
import runpy
import unittest
from pathlib import Path
from types import SimpleNamespace

ROOT = Path(__file__).resolve().parents[1]


class Handle:
    def head(self, limit):
        return [{"id": "r1", "label": "Run", "status": "RUN_STATUS_COMPLETE",
                 "phase": "RUN_PHASE_DONE", "executionMode": "EXECUTION_MODE_IMPORTED",
                 "progressPercent": 100, "importSourceHarness": "claude-code",
                 "updatedAt": "t1", "importSourceSessionId": "private-session",
                 "summary": {"text": "private result"}, "errorMsg": "private error"}]

    def count(self):
        return 1

    def meta(self):
        return {"total": 4, "hasMore": False}


class ProgramStub:
    def inputs(self):
        return {"limit": 5, "offset": 0}

    def envelope(self, name, version, phase, inputs):
        return {"program": name, "version": version, "status": "failed", "phase": phase,
                "inputs": inputs, "signals": {}, "errors": [], "evidence": []}

    def fail(self, status, error_class, detail, phase):
        raise AssertionError((status, error_class, detail, phase))

    def run(self, states, state):
        while state:
            state = states[state]()


class RunInventoryTests(unittest.TestCase):
    def test_only_cli_inventory_fields_are_projected(self):
        handle = Handle()
        manager = SimpleNamespace(run=SimpleNamespace(list=lambda **kwargs: handle))
        output = io.StringIO()
        with contextlib.redirect_stdout(output):
            runpy.run_path(str(ROOT / "run-inventory.py"), init_globals={"program": ProgramStub(), "agent_manager": manager})
        result = json.loads(output.getvalue())
        row = result["signals"]["runs"][0]
        self.assertEqual(set(row), {"id", "label", "status", "phase", "execution_mode", "progress_percent", "source", "updated_at"})
        self.assertEqual(row["source"], "claude-code")
        self.assertTrue(result["signals"]["has_more"])
        self.assertNotIn("importSourceSessionId", row)
        self.assertNotIn("summary", row)
        self.assertNotIn("errorMsg", row)


if __name__ == "__main__":
    unittest.main()
