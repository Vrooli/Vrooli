import contextlib
import io
import json
import sys
import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[5]
HOST = ROOT / "scenarios" / "program-runtime" / "kernel" / "host"
PROGRAM_SOURCE = Path(__file__).resolve().parents[1] / "friction-digest.py"
CONTRACT_PATH = Path(__file__).resolve().parents[1] / "friction-digest.json"

sys.path.insert(0, str(HOST))
from program_helper import ProgramHelper  # noqa: E402


class FakeHandle:
    def __init__(self, meta=None, rows=None):
        self._meta = meta or {}
        self._rows = rows or []

    def head(self, _limit):
        return self._rows

    def meta(self):
        return self._meta


class FakeUsage:
    def __init__(self, record):
        self.record = record

    def friction(self, **_kwargs):
        return FakeHandle(meta=self.record, rows=[{
            "fingerprint": "fp-1", "kind": "failure", "commandPath": "agent-manager run get",
            "description": "`agent-manager run get` failed 3 times", "calls": 3, "runs": 3,
            "cost": 3, "costUnit": "calls", "costShare": 0.5, "status": "open",
        }])


class FakeWatch:
    def policy_outcomes(self, **_kwargs):
        return FakeHandle(rows=[])


class FrictionDigestAnnotationTests(unittest.TestCase):
    def run_program(self, record):
        program = ProgramHelper()
        environment = {
            "inputs": {"scenario": "agent-manager", "window_days": 7},
            "program": program,
            "agent_manager": type("AgentManager", (), {"usage": FakeUsage(record), "watch": FakeWatch()})(),
        }
        program._bind(environment)
        stdout = io.StringIO()
        with contextlib.redirect_stdout(stdout):
            exec(compile(PROGRAM_SOURCE.read_text(), str(PROGRAM_SOURCE), "exec"), environment)
        return json.loads(stdout.getvalue())

    def record(self, inferred):
        return {
            "windowStart": "2026-11-12T00:00:00Z",
            "coverage": {"sentence": "seen across 10 runs"},
            "ownerProfileCount": 1, "ownerCallCount": 3,
            "belowFloor": 0, "beyondLimit": 0,
            "inferredAnnotations": inferred,
        }

    def test_unproven_labels_carry_evidence_but_never_a_rate(self):
        envelope = self.run_program(self.record({
            "packName": "default", "packVersion": "1",
            "eligibleRuns": 10, "observedRuns": 4, "coverage": 0.4,
            "labels": [
                {"id": "agent.spinning", "group": "agent", "state": "observed", "observed": 2, "eligible": 4,
                 "coverage": 0.5, "gateState": "unmeasured", "gold": 0, "targetPrecision": 0.8,
                 "targetRecall": 0.7, "friction": True},
                {"id": "run.type", "group": "run", "state": "observed", "observed": 4, "eligible": 4,
                 "coverage": 1.0, "gateState": "unmeasured", "gold": 0, "friction": False},
            ],
        }))
        self.assertEqual("ok", envelope["status"])
        self.assertEqual("5", envelope["version"])
        section = envelope["signals"]["inferred_annotations"]
        self.assertEqual("default 1", section["pack"])
        self.assertEqual(0.4, section["coverage"])
        self.assertEqual(10, section["eligible_runs"])
        self.assertEqual(4, section["observed_runs"])
        self.assertEqual(1, section["label_count"], "non-friction facets are not carried")
        self.assertEqual(1, section["unproven_count"])
        self.assertEqual(1, len(section["labels"]))
        self.assertIsNone(section["labels"][0]["rate"], "an unproven label is evidence, not a rate")
        self.assertEqual("agent.spinning", section["labels"][0]["id"])

    def test_a_proven_label_carries_its_rate(self):
        envelope = self.run_program(self.record({
            "packName": "default", "packVersion": "1", "eligibleRuns": 10, "observedRuns": 10,
            "coverage": 1.0, "note": None,
            "labels": [{"id": "agent.spinning", "group": "agent", "state": "observed", "observed": 5,
                        "eligible": 10, "coverage": 0.5, "rate": 0.5, "gateState": "proven", "gold": 40,
                        "targetPrecision": 0.8, "targetRecall": 0.7, "friction": True}],
        }))
        section = envelope["signals"]["inferred_annotations"]
        self.assertEqual(0.5, section["labels"][0]["rate"])
        self.assertEqual("proven", section["labels"][0]["gate_state"])
        self.assertEqual(0, section["unproven_count"])

    def test_an_absent_section_reads_as_unknown_not_zero_friction(self):
        envelope = self.run_program(self.record(None))
        section = envelope["signals"]["inferred_annotations"]
        self.assertEqual(0.0, section["coverage"])
        self.assertEqual([], section["labels"])
        self.assertEqual(0, section["label_count"])
        contract = json.loads(CONTRACT_PATH.read_text())
        self.assertEqual("5", contract["version"])


if __name__ == "__main__":
    unittest.main()
