import contextlib
import io
import json
import sys
import unittest
from pathlib import Path

try:
    import jsonschema
except ImportError:
    jsonschema = None


ROOT = Path(__file__).resolve().parents[5]
HOST = ROOT / "scenarios" / "program-runtime" / "kernel" / "host"
PROGRAM_SOURCE = Path(__file__).resolve().parents[1] / "friction-digest.py"
CONTRACT_PATH = Path(__file__).resolve().parents[1] / "friction-digest.json"
SCHEMA_PATH = Path(__file__).resolve().parents[1] / "schemas" / "friction-digest.output.schema.json"

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

    def test_unproven_labels_are_named_with_their_gate_state_and_never_a_rate(self):
        envelope = self.run_program(self.record({
            "packName": "default", "packVersion": "1",
            "eligibleRuns": 10, "observedRuns": 4, "coverage": 0.4,
            "labels": [
                {"id": "agent.discovery_slow", "group": "agent", "state": "unproven", "observed": 2, "eligible": 4,
                 "coverage": 0.5, "gateState": "tuning_only", "gold": 47, "targetPrecision": 0.8,
                 "targetRecall": 0.7, "friction": True},
                {"id": "run.type", "group": "run", "state": "observed", "observed": 4, "eligible": 4,
                 "coverage": 1.0, "gateState": "unmeasured", "gold": 0, "friction": False},
            ],
        }))
        self.assertEqual("ok", envelope["status"])
        self.assertEqual("7", envelope["version"])
        section = envelope["signals"]["inferred_annotations"]
        self.assertEqual("default 1", section["pack"])
        self.assertEqual(0.4, section["coverage"])
        self.assertEqual(10, section["eligible_runs"])
        self.assertEqual(4, section["observed_runs"])
        self.assertEqual(1, section["label_count"], "non-friction facets are not carried")
        self.assertEqual(1, section["unproven_count"])
        self.assertEqual([], section["proven"])
        self.assertEqual(1, len(section["unproven"]))
        entry = section["unproven"][0]
        self.assertEqual("agent.discovery_slow", entry["id"])
        self.assertEqual("tuning_only", entry["gate_state"])
        self.assertNotIn("rate", entry, "an unproven label is evidence, not a rate")

    def test_a_proven_label_carries_its_rate_and_gate(self):
        envelope = self.run_program(self.record({
            "packName": "default", "packVersion": "1", "eligibleRuns": 10, "observedRuns": 10,
            "coverage": 1.0, "note": None,
            "labels": [{"id": "agent.discovery_slow", "group": "agent", "state": "proven", "observed": 5,
                        "eligible": 10, "coverage": 0.5, "rate": 0.5, "gateState": "proven", "gold": 40,
                        "precision": 0.9, "recall": 0.8, "targetPrecision": 0.8, "targetRecall": 0.7, "friction": True,
                        "provenObserved": 5, "gateModel": "winnow:e4b", "gateRole": "decide.local.accurate", "calibration": "calibrated", "ece": 0.04}],
        }))
        section = envelope["signals"]["inferred_annotations"]
        self.assertEqual(1, len(section["proven"]))
        proven = section["proven"][0]
        self.assertEqual(0.5, proven["rate"])
        self.assertEqual("proven", proven["gate_state"])
        self.assertEqual(0.9, proven["precision"])
        self.assertEqual(("winnow:e4b", "decide.local.accurate", "calibrated", 0.04, 5), (proven["model"], proven["resource_role"], proven["calibration"], proven["ece"], proven["proven_observed"]))
        self.assertEqual([], section["unproven"])
        self.assertEqual(0, section["unproven_count"])

    def test_a_label_with_no_rate_is_unproven_even_when_its_best_model_is_proven(self):
        # A label proven for one model but answered only by another has no rate.
        envelope = self.run_program(self.record({
            "packName": "default", "packVersion": "1", "eligibleRuns": 10, "observedRuns": 10, "coverage": 1.0,
            "labels": [{"id": "work.blocked_by_tool", "group": "work", "state": "unproven", "observed": 5,
                        "eligible": 10, "coverage": 0.5, "gateState": "proven", "gold": 44,
                        "targetPrecision": 0.8, "targetRecall": 0.7, "friction": True}],
        }))
        section = envelope["signals"]["inferred_annotations"]
        self.assertEqual([], section["proven"])
        self.assertEqual("work.blocked_by_tool", section["unproven"][0]["id"])

    def test_an_absent_section_reads_as_unknown_not_zero_friction(self):
        envelope = self.run_program(self.record(None))
        section = envelope["signals"]["inferred_annotations"]
        self.assertEqual(0.0, section["coverage"])
        self.assertEqual([], section["proven"])
        self.assertEqual([], section["unproven"])
        self.assertEqual(0, section["label_count"])
        contract = json.loads(CONTRACT_PATH.read_text())
        self.assertEqual("7", contract["version"])

    @unittest.skipIf(jsonschema is None, "jsonschema is not installed")
    def test_the_envelope_satisfies_the_declared_output_schema(self):
        schema = json.loads(SCHEMA_PATH.read_text())
        proven = {"id": "agent.discovery_slow", "group": "agent", "state": "proven", "observed": 5, "eligible": 10, "coverage": 0.5,
                  "rate": 0.5, "gateState": "proven", "gold": 40, "precision": 0.9, "recall": 0.8, "friction": True,
                  "provenObserved": 5, "gateModel": "rule", "gateRole": "", "calibration": "rule"}
        unproven = {"id": "work.blocked_by_tool", "group": "work", "state": "unproven", "observed": 5, "eligible": 10,
                    "coverage": 0.5, "gateState": "unmeasured", "gold": 0, "targetPrecision": 0.8, "targetRecall": 0.7, "friction": True}
        envelope = self.run_program(self.record({"packName": "default", "packVersion": "1", "eligibleRuns": 10, "observedRuns": 10,
                                                 "coverage": 1.0, "labels": [proven, unproven]}))
        errors = [error.message for error in jsonschema.Draft202012Validator(schema).iter_errors(envelope)]
        self.assertEqual([], errors)


if __name__ == "__main__":
    unittest.main()
