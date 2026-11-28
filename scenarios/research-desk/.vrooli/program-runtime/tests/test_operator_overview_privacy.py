import contextlib
import io
import json
import runpy


class FakeHandle:
    def meta(self):
        return {
            "itemCount": 3,
            "olderListCount": 1,
            "observedAt": "2026-11-23T00:00:00Z",
            "items": [{"id": "PRIVATE-ID", "body": "PRIVATE-BODY", "answers": ["PRIVATE-ANSWER"]}],
            "unreadable": [{"path": "/private/path", "error": "PRIVATE-ERROR"}],
            "secret": "PRIVATE-SECRET",
        }

    def head(self, _count):
        raise AssertionError("needs-you must not materialize private item rows")

    def count(self):
        raise AssertionError("needs-you must use owner counts only")


class FakeProgram:
    def __init__(self):
        self.envelope = None

    def inputs(self):
        return {"view": "needs-you"}

    def fail(self, status, klass, message, phase):
        raise AssertionError((status, klass, message, phase))

    def classify(self, error):
        raise error

    def run(self, steps, start):
        step = start
        while step:
            step = steps[step]()


class FakeResearchDesk:
    class operator:
        @staticmethod
        def needs_you(**_kwargs):
            return FakeHandle()


program = FakeProgram()
stdout = io.StringIO()
with contextlib.redirect_stdout(stdout):
    runpy.run_path(
        "scenarios/research-desk/.vrooli/program-runtime/operator-overview.py",
        init_globals={"program": program, "research_desk": FakeResearchDesk},
    )

raw = stdout.getvalue()
envelope = json.loads(raw)
assert envelope["status"] == "partial"  # unreadable data must remain visible as incompleteness.
assert envelope["signals"]["result"] == {
    "item_count": 3,
    "older_list_count": 1,
    "observed_at": "2026-11-23T00:00:00Z",
    "unreadable_count": 1,
    "details_redacted": True,
    "truncated": False,
}
for private in ("PRIVATE-ID", "PRIVATE-BODY", "PRIVATE-ANSWER", "PRIVATE-ERROR", "PRIVATE-SECRET", "/private/path"):
    assert private not in raw

class ErrorProgram(FakeProgram):
    def classify(self, _error):
        return "failed", "remote_error"

    def fail(self, status, klass, message, phase):
        self.failure = {"status": status, "class": klass, "detail": str(message), "phase": phase}
        return "report"


class BrokenResearchDesk:
    class operator:
        @staticmethod
        def needs_you(**_kwargs):
            raise RuntimeError("PRIVATE-BODY /private/error/path")


error_program = ErrorProgram()
error_stdout = io.StringIO()
with contextlib.redirect_stdout(error_stdout):
    runpy.run_path(
        "scenarios/research-desk/.vrooli/program-runtime/operator-overview.py",
        init_globals={"program": error_program, "research_desk": BrokenResearchDesk},
    )
error_raw = error_stdout.getvalue()
assert "PRIVATE-BODY" not in error_raw
assert "/private/error/path" not in error_raw

class TopicHandle:
    def meta(self):
        return {"topics": [{"body": "PRIVATE-TOPIC-BODY"}], "researchHome": "/private/home",
                "warnings": ["PRIVATE-WARNING"], "privateDetail": "PRIVATE-METADATA"}

    def head(self, count):
        assert count == 8
        return [{"topic": f"topic-{i}", "commissionIds": [f"private-{i}"],
                 "liveCommissionIds": ["private-live"],
                 "missionCheck": {"open": i, "active": 1, "waitingOperator": 2, "landed": 3,
                                  "errors": ["PRIVATE-MISSION-ERROR"]},
                 "title": "PRIVATE-TITLE", "mission": {"path": "/private/mission"}}
                for i in range(8)]

    def count(self):
        return 47


class TopicProgram(FakeProgram):
    def inputs(self):
        return {"view": "topics"}


class FakeTopicResearchDesk:
    class activity:
        @staticmethod
        def topics(**kwargs):
            assert kwargs == {"rows": "topics"}
            return TopicHandle()


topic_stdout = io.StringIO()
with contextlib.redirect_stdout(topic_stdout):
    runpy.run_path(
        "scenarios/research-desk/.vrooli/program-runtime/operator-overview.py",
        init_globals={"program": TopicProgram(), "research_desk": FakeTopicResearchDesk},
    )
topic_raw = topic_stdout.getvalue()
topic_envelope = json.loads(topic_raw)
assert len(topic_raw.encode()) < 4096
assert topic_envelope["status"] == "partial"
assert topic_envelope["signals"]["result"]["topic_count"] == 47
assert topic_envelope["signals"]["result"]["truncated"] is True
assert len(topic_envelope["signals"]["result"]["rows"]) == 8
for private in ("PRIVATE-TOPIC-BODY", "/private/home", "PRIVATE-WARNING", "PRIVATE-METADATA",
                "PRIVATE-TITLE", "/private/mission", "PRIVATE-MISSION-ERROR", "private-live"):
    assert private not in topic_raw
