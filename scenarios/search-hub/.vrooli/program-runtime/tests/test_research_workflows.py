"""Offline behavior checks for the repeated Search Hub workflow routes."""
import contextlib
import io
import json
import runpy
from pathlib import Path

SOURCE = Path(__file__).parents[1] / "research.py"


class Handle:
    def __init__(self, rows=None, meta=None):
        self.rows = list(rows or [])
        self._meta = meta or {}

    def meta(self):
        return self._meta

    def count(self):
        return len(self.rows)

    def head(self, count):
        return self.rows[:count]

    def materialize(self, count):
        return self.rows[:count]

    def filter(self, predicate):
        return Handle([row for row in self.rows if predicate(row)], self._meta)


class FakeProgram:
    def __init__(self, inputs, events):
        self._inputs, self.events = inputs, events
        self.output = None

    def inputs(self):
        return self._inputs

    def run(self, states, state):
        while state:
            state = states[state]()

    def report(self, value):
        self.output = value

    def fail(self, status, klass, detail, where):
        self.output = {"status": status, "phase": where, "signals": {},
                       "errors": [{"class": klass, "detail": str(detail), "where": where}], "evidence": []}
        return "report"

    def classify(self, _exc):
        return "failed", "binding_error"


class Query:
    def __init__(self, events):
        self.events = events
        self.calls = []

    def query(self, **kwargs):
        self.events.append("query")
        self.calls.append(kwargs)
        return Handle([{"id": "doc-1", "title": "safe title", "providerId": "fixture", "type": "doc",
                        "path": "fixture/path", "score": 0.8, "snippet": "fixture snippet"}],
                      {"corporaSearched": ["fixture"], "groups": [], "degraded": False,
                       "partial": False, "rerankerLeg": "fixture", "latencyMs": "3"})


class Status:
    def __init__(self, events, value="running"):
        self.events, self.value = events, value

    def __call__(self, **kwargs):
        self.events.append(("status", kwargs))
        return Handle([{"status": self.value}])


class Discovery:
    def __init__(self, events):
        self.events = events

    def discover(self, **kwargs):
        self.events.append(("discover", kwargs))
        return Handle([{"id": "skill-1", "type": "skill", "content": "private skill body"}])


class SkillRead:
    def __init__(self, events):
        self.events = events

    def read(self, **kwargs):
        self.events.append(("skill_read", kwargs))
        return Handle([{"id": "skill-1", "content": "private skill body"}])


class Memory:
    def __init__(self, events):
        self.events = events

    def note(self, **kwargs):
        self.events.append(("note", kwargs))
        return Handle(meta={"entry": {"id": "entry-opaque"}})


def execute(inputs, status="running"):
    events = []
    program = FakeProgram(inputs, events)
    query = Query(events)
    stdout = io.StringIO()
    bindings = {
        "program": program,
        "search_hub": type("SearchHub", (), {"query": type("QueryGroup", (), {"query": query.query})()})(),
        "prompt_manager": type("Prompt", (), {
            "discover": type("Discover", (), {"discover": Discovery(events).discover})(),
            "skill": type("Skill", (), {"read": SkillRead(events).read})(),
        })(),
        "vrooli": type("Vrooli", (), {"scenario": type("Scenario", (), {"status": Status(events, status)})()})(),
        "vrooli_memory": type("MemoryBinding", (), {"journal": type("Journal", (), {"note": Memory(events).note})()})(),
    }
    with contextlib.redirect_stdout(stdout):
        runpy.run_path(str(SOURCE), init_globals=bindings)
    return program.output, events, query.calls


def test_query_discover_note_route_keeps_query_and_result_content_out_of_memory():
    query_text = "fixture secret query text"
    output, events, calls = execute({"query": query_text, "include_discover": True, "capture_note": True})
    assert output["status"] == "ok"
    assert [event if isinstance(event, str) else event[0] for event in events] == ["query", "discover", "note"]
    assert events[1][1] == {"queries": [query_text], "complexity": "moderate", "limit": 5,
                            "type": "skill", "rows": "results"}
    note = events[2][1]
    serialized_note = json.dumps(note)
    assert query_text not in serialized_note and "private skill body" not in serialized_note
    assert note["body"].startswith("Search Hub discovery outcome: verdict=answered;")
    assert output["signals"]["memory_capture"] == {"status": "written", "entry_ref": "entry-opaque"}
    assert calls == [{"text": query_text, "limit": 10, "explain": True, "rows": "ranked"}]


def test_query_status_repeat_route_reuses_exact_query_once_only_when_running():
    inputs = {"query": "same query", "effort": "fast", "repeat_after_status": True}
    output, events, calls = execute(inputs, status="running")
    assert output["status"] == "ok"
    assert [event if isinstance(event, str) else event[0] for event in events] == ["query", "status", "query"]
    assert calls == [calls[0], calls[0]]
    assert output["signals"]["status_check"] == {"status": "running"}
    assert output["signals"]["first_query"] == {"ranked_count": 1, "corpora_count": 1}

    output, events, calls = execute(inputs, status="stopped")
    assert output["status"] == "unavailable"
    assert [event if isinstance(event, str) else event[0] for event in events] == ["query", "status"]
    assert len(calls) == 1
    assert output["signals"]["status_check"] == {"status": "stopped"}


def test_workflow_flags_require_exact_safe_inputs():
    output, events, calls = execute({"query": "query", "capture_note": True})
    assert output["status"] == "failed"
    assert not events and not calls
