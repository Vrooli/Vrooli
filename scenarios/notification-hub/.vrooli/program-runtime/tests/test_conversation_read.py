import contextlib
import io
import json
import runpy
from pathlib import Path

SOURCE = Path(__file__).parents[1] / "conversation-read.py"


class FakeProgram:
    def __init__(self, inputs):
        self._inputs = inputs
        self.output = None
        self.error = None

    def inputs(self):
        return self._inputs

    def run(self, states, state):
        while state:
            state = states[state]()

    def classify(self, _exc):
        return "failed", "binding_error"

    def fail(self, status, klass, detail, where):
        self.error = {"status": status, "class": klass, "detail": detail, "where": where}
        return "report"

    def report(self, envelope):
        self.output = envelope
        if self.error:
            envelope["status"] = self.error["status"]
            envelope["errors"] = [self.error]


class FakeHandle:
    def __init__(self, response):
        self.response = response

    def raw(self):
        return self.response


class FakeNotificationHub:
    def __init__(self, response):
        self.response = response
        self.calls = []
        self.conversations = self

    def get(self, **kwargs):
        self.calls.append(kwargs)
        return FakeHandle(self.response)


def execute(inputs, response):
    program = FakeProgram(inputs)
    hub = FakeNotificationHub(response)
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        runpy.run_path(str(SOURCE), init_globals={"program": program, "notification_hub": hub})
    return program.output, hub.calls


def test_exact_id_reads_exact_owner_ask_without_a_secondary_write():
    synthetic_ask = {"id": "fixture-ask-1", "subjectKey": "fixture-key", "question": "synthetic private question",
                     "answer": "synthetic private answer", "state": "answered"}
    output, calls = execute({"ask_id": "fixture-ask-1"}, {"ask": synthetic_ask})
    assert output["status"] == "ok"
    assert output["signals"]["ask"] == synthetic_ask
    assert calls == [{"ask_id": "fixture-ask-1"}]
    assert output["evidence"] == ["notification-hub/conversations/get"]
    assert "synthetic private question" not in json.dumps(output["evidence"])


def test_exact_subject_key_is_forwarded_and_verified():
    synthetic_ask = {"id": "fixture-ask-2", "subjectKey": "exact-fixture-key", "question": "synthetic only"}
    output, calls = execute({"subject_key": "exact-fixture-key"}, {"ask": synthetic_ask})
    assert output["status"] == "ok"
    assert calls == [{"subject_key": "exact-fixture-key"}]

    mismatch, calls = execute({"subject_key": "requested-key"}, {"ask": synthetic_ask})
    assert mismatch["status"] == "failed"
    assert mismatch["errors"][0]["class"] == "invalid_response"
    assert "ask" not in mismatch["signals"]
    assert calls == [{"subject_key": "requested-key"}]


def test_missing_or_conflicting_identity_is_rejected_before_owner_call():
    for inputs in ({}, {"ask_id": "fixture-ask", "subject_key": "fixture-key"}):
        output, calls = execute(inputs, {"ask": {"id": "fixture-ask"}})
        assert output["status"] == "failed"
        assert not calls
