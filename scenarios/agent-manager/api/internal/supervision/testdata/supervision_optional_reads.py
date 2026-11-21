"""Exercise degraded read paths for the two supervision read programs."""

import contextlib
import io
import json
import sys
from pathlib import Path
from types import SimpleNamespace

sys.path.insert(0, sys.argv[3])
from host.program_helper import ProgramHelper


class Handle:
    def __init__(self, rows=None, metadata=None, count=None):
        self.rows = rows or []
        self.metadata = metadata or {}
        self.row_count = len(self.rows) if count is None else count

    def head(self, limit):
        return self.rows[:limit]

    def meta(self):
        return self.metadata

    def count(self):
        return self.row_count


class OwnerCalls:
    def __init__(self, responses):
        self.responses = responses

    def __getattr__(self, name):
        def call(**_):
            value = self.responses[name]
            if isinstance(value, Exception):
                raise value
            return value

        return call


def run(source_path, inputs, calls):
    output = io.StringIO()
    helper = ProgramHelper()
    environment = {
        "inputs": inputs,
        "program": helper,
        "agent_manager": SimpleNamespace(watch=OwnerCalls(calls)),
        "gather": lambda *functions: [function() for function in functions],
    }
    helper._bind(environment)
    source = Path(source_path).read_text(encoding="utf-8")
    with contextlib.redirect_stdout(output):
        exec(compile(source, source_path, "exec"), environment)
    rows = output.getvalue().splitlines()
    assert len(rows) == 1, rows
    return json.loads(rows[0])


watch = {
    "watch": {
        "watchId": "watch-1",
        "status": "active",
        "revision": "3",
        "spec": {"policyVersion": "supervision-v2"},
        "lastDecision": {"decisionId": "decision-1", "disposition": "quiet", "classification": "quiet"},
    },
    "cursorResetRequired": False,
    "subjectStates": [],
}
case_result = run(
    sys.argv[1],
    {"watch_id": "watch-1"},
    {
        "inspect": Handle(metadata=watch, count=0),
        "actions": RuntimeError("action history unavailable"),
        "policy_outcomes": Handle(
            rows=[{"observedClass": "event_count"}],
            metadata={"coverage": "complete"},
        ),
    },
)
assert case_result["status"] == "partial", case_result
assert case_result["signals"]["actions_available"] is False, case_result
assert case_result["signals"]["outcomes_available"] is True, case_result
assert case_result["signals"]["assessed_sample"] == 1, case_result

experiment_result = run(
    sys.argv[2],
    {"policy_version": "supervision-v2"},
    {
        "policy_get": Handle(
            metadata={
                "policy": {"version": "supervision-v2", "evaluatorDigest": "digest-2"},
                "state": "candidate",
                "digest": "policy-digest-2",
                "evaluation": None,
            }
        ),
        "policy_outcomes": RuntimeError("outcome coverage unavailable"),
    },
)
assert experiment_result["status"] == "partial", experiment_result
assert experiment_result["signals"]["outcomes_available"] is False, experiment_result
assert experiment_result["signals"]["assessed_sample"] is None, experiment_result
assert experiment_result["signals"]["next_action"] == "retry_outcome_read", experiment_result
print("supervision optional-read regressions passed")
