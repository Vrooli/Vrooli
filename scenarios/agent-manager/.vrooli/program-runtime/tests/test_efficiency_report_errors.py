import contextlib
import io
import json
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[5]
HOST = ROOT / "scenarios" / "program-runtime" / "kernel" / "host"
PROGRAM_SOURCE = Path(__file__).resolve().parents[1] / "efficiency-report.py"
CONTRACT_PATH = Path(__file__).resolve().parents[1] / "efficiency-report.json"

sys.path.insert(0, str(HOST))
from program_helper import ProgramHelper, bridge_error  # noqa: E402


class FakeHandle:
    def head(self, _limit):
        return [{}]

    def meta(self):
        return {}


class FakeMeasures:
    def __init__(self, first_error):
        self.first_error = first_error
        self.calls = []

    def __getattr__(self, name):
        def invoke(**_kwargs):
            self.calls.append(name)
            if name == "run_volume":
                raise self.first_error
            return FakeHandle()

        return invoke


class EfficiencyReportErrorTests(unittest.TestCase):
    def run_program(self, owner_error):
        program = ProgramHelper()
        measures = FakeMeasures(owner_error)

        def gather(*calls):
            # The production gather raises when an unguarded read fails.
            return [call() for call in calls]

        environment = {
            "inputs": {"preset": "24h", "group_by": "profile"},
            "program": program,
            "agent_manager": type("AgentManager", (), {"measures": measures})(),
            "gather": gather,
        }
        program._bind(environment)
        stdout = io.StringIO()
        with contextlib.redirect_stdout(stdout):
            exec(compile(PROGRAM_SOURCE.read_text(), str(PROGRAM_SOURCE), "exec"), environment)
        return json.loads(stdout.getvalue()), measures.calls

    def test_owner_403_is_reported_as_remote_error_without_retry(self):
        owner_error = bridge_error(
            "agent-manager/measures/run-volume",
            403,
            {
                "error": "invoke agent-manager/measures/run-volume: remote status 403 Forbidden: agent authority does not include global management or approval",
                "class": "remote_error",
                "status": "failed",
                "http_status": 403,
            },
        )

        envelope, calls = self.run_program(owner_error)

        self.assertEqual("failed", envelope["status"])
        self.assertEqual("remote_error", envelope["errors"][0]["class"])
        self.assertIn("403", envelope["errors"][0]["detail"])
        self.assertEqual(["run_volume"], calls)

    def test_runtime_grant_refusal_remains_refused_without_retry(self):
        owner_error = ProgramHelper.Refused(
            "the session does not have the required grant",
            binding_id="agent-manager/measures/run-volume",
        )

        envelope, calls = self.run_program(owner_error)

        self.assertEqual("refused", envelope["status"])
        self.assertEqual("no_grant", envelope["errors"][0]["class"])
        self.assertEqual(["run_volume"], calls)

    def test_contract_declares_remote_error(self):
        contract = json.loads(CONTRACT_PATH.read_text())

        self.assertIn("failed", contract["outputs"]["status"]["enum"])
        self.assertIn("refused", contract["outputs"]["status"]["enum"])
        self.assertIn("remote_error", contract["outputs"]["errors"]["classes"])
        self.assertIn("no_grant", contract["outputs"]["errors"]["classes"])


if __name__ == "__main__":
    unittest.main()
