"""Offer Desk's triage and release-path reports stay read-only and owner-governed."""
import contextlib
import io
import json
import sys
from pathlib import Path
from types import SimpleNamespace

KERNEL = Path(__file__).resolve().parents[1]
REPO = KERNEL.parents[2]
sys.path.insert(0, str(KERNEL))
sys.path.insert(0, str(KERNEL / "host"))

from host.program_helper import ProgramHelper  # noqa: E402


class Rows:
    def __init__(self, rows=()):
        self.rows = list(rows)

    def head(self, limit):
        return self.rows[:limit]


def execute(name, inputs, bindings):
    path = REPO / "scenarios/offer-desk/.vrooli/program-runtime" / f"{name}.py"
    contract = json.loads(path.with_suffix(".json").read_text())
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    reports = []
    original_report = helper.report

    def report(envelope=None):
        reports.append(envelope)
        return original_report(envelope)

    helper.report = report
    namespace = {"inputs": inputs, **bindings}
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(path.read_text(), str(path), "exec"), namespace)
    result = json.loads(output.getvalue())
    assert len(reports) == 1
    assert result["program"] == contract["name"]
    assert result["phase"] == "report"
    return result


def triage_bindings(*, board_error=None, proposals_error=None):
    calls = []

    def board_show(**kwargs):
        calls.append(("board", kwargs))
        if board_error:
            raise board_error
        return Rows()

    def proposals(**kwargs):
        calls.append(("proposals", kwargs))
        if proposals_error:
            raise proposals_error
        return Rows()

    inference_calls = []

    def infer(*_args, **_kwargs):
        inference_calls.append(True)
        raise AssertionError("empty-board fixture must not invoke inference")

    bindings = {
        "offer_desk": SimpleNamespace(offers=SimpleNamespace(board_show=board_show, gates_proposals=proposals)),
        "ai": SimpleNamespace(batch=infer),
    }
    return bindings, calls, inference_calls


def release_bindings(*, ladder_error=None, enabling_error=None):
    calls = []
    ladder = [{"deliverable": {"id": "n1", "name": "Product", "status": "ACTIVE",
                               "releaseRank": 1, "actualAccountId": "account-1"},
               "enablers": [{"node": {"id": "n2", "name": "Gate", "status": "ACTIVE"}}],
               "unlockedRamps": []}]
    enabling = [{"node": {"id": "n1"}, "derivedUrgency": 8},
                {"node": {"id": "n2"}, "derivedUrgency": 5}]

    def read(name, values, error, **kwargs):
        calls.append((name, kwargs))
        if error:
            raise error
        return Rows(values)

    bindings = {"offer_desk": SimpleNamespace(offers=SimpleNamespace(
        release_ladder=lambda **kwargs: read("ladder", ladder, ladder_error, **kwargs),
        enabling_list=lambda **kwargs: read("enabling", enabling, enabling_error, **kwargs),
        release_prerequisites=lambda **kwargs: read("prerequisites", [], None, **kwargs),
    )), "gather": lambda *calls: [call() for call in calls]}
    return bindings, calls


def test_attention_triage_keeps_empty_board_and_optional_error_without_inference():
    bindings, calls, inference_calls = triage_bindings()
    result = execute("attention-triage", {"max_items": 4}, bindings)
    assert calls == [("board", {"rows": "entries"}), ("proposals", {"rows": "proposals"})]
    assert inference_calls == []
    assert result["status"] == "ok"
    assert result["evidence"] == ["offer-desk/offers/board-show", "offer-desk/offers/gates-proposals"]
    assert result["signals"]["triage_reason"] == "no_candidate"
    assert result["signals"]["inference_calls"] == 0

    bindings, calls, inference_calls = triage_bindings(proposals_error=RuntimeError("connection refused"))
    result = execute("attention-triage", {}, bindings)
    assert len(calls) == 2
    assert inference_calls == []
    assert result["status"] == "partial"
    assert result["signals"]["proposals_read"] is False
    assert result["errors"][0]["where"] == "collect:proposals"

    bindings, calls, inference_calls = triage_bindings(board_error=RuntimeError("connection refused"))
    result = execute("attention-triage", {}, bindings)
    assert calls == [("board", {"rows": "entries"})]
    assert inference_calls == []
    assert result["status"] == "unavailable"
    assert result["errors"][0]["where"] == "collect"


def test_release_path_uses_only_declared_reads_and_preserves_failure_policy():
    bindings, calls = release_bindings()
    result = execute("release-path", {}, bindings)
    assert calls == [("ladder", {"include_retired": False, "rows": "entries"}),
                     ("enabling", {"include_retired": False, "rows": "enabling"})]
    assert result["status"] == "ok"
    assert result["signals"]["prerequisites_requested"] is False
    assert result["signals"]["prerequisites"] is None
    assert result["signals"]["blocked_count"] == 1
    assert result["signals"]["path"][0]["blocked_by"][0]["urgency"] == 5

    bindings, calls = release_bindings(enabling_error=RuntimeError("optional read failed"))
    result = execute("release-path", {}, bindings)
    assert len(calls) == 2
    assert result["status"] == "partial"
    assert result["errors"][0]["where"] == "collect:enabling"
    assert result["errors"][0]["class"] == "binding_error"
    assert result["signals"]["enabling_ranked"] == []

    bindings, calls = release_bindings(ladder_error=RuntimeError("connection refused"),
                                       enabling_error=RuntimeError("connection refused"))
    result = execute("release-path", {}, bindings)
    assert len(calls) == 2
    assert result["status"] == "unavailable"
    assert result["signals"] == {}
