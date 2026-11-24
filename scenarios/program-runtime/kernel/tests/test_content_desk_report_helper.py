"""Content Desk readers use the shared final reporter without changing read/status policy."""
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

    def count(self):
        return len(self.rows)


class NotFound(Exception):
    http_status = 404


def execute(name, inputs, bindings):
    path = REPO / "scenarios/content-desk/.vrooli/program-runtime" / f"{name}.py"
    contract = json.loads(path.with_suffix(".json").read_text())
    helper = ProgramHelper(contract["helpers"]["program"], contract["outputs"])
    reports = []
    original_report = helper.report

    def report(envelope=None):
        reports.append(envelope)
        return original_report(envelope)

    helper.report = report
    namespace = {"inputs": inputs, "gather": lambda *calls: [call() for call in calls], **bindings}
    helper._bind(namespace)
    namespace["program"] = helper
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        exec(compile(path.read_text(), str(path), "exec"), namespace)
    return contract, reports, output.getvalue()


def assert_one_report(contract, reports, raw, status):
    result = json.loads(raw)
    assert len(reports) == 1
    assert result["program"] == contract["name"]
    assert result["status"] == status
    assert result["phase"] == "report"
    return result


def board_bindings(*, campaign_error=None, capability_error=None, offer_error=None):
    calls = []

    def read(name, values=(), error=None, **kwargs):
        calls.append((name, kwargs))
        if error:
            raise error
        return Rows(values)

    bindings = {
        "content_desk": SimpleNamespace(
            campaigns=SimpleNamespace(list=lambda **kwargs: read("campaigns", error=campaign_error, **kwargs)),
            artifacts=SimpleNamespace(list=lambda **kwargs: read("drafts", **kwargs)),
            capabilities=SimpleNamespace(list=lambda **kwargs: read("capabilities", error=capability_error, **kwargs)),
        ),
        "offer_desk": SimpleNamespace(offers=SimpleNamespace(
            release_ladder=lambda **kwargs: read("offer_ladder", error=offer_error, **kwargs))),
    }
    return bindings, calls


def test_capabilities_read_reports_successful_absence_and_unavailable_read():
    calls = []
    capability = {"id": "cap-1", "medium": "text", "owner": "marketing",
                  "definitionStatus": "defined", "implementationStatus": "implemented",
                  "operationalReadiness": "ready", "outputQuality": "good",
                  "distributionConnectivity": "connected", "readinessLimitations": ["fixture"],
                  "latestQualification": {"observedAt": "2026-11-24", "maxAgeSeconds": 60}}

    def list_capabilities(**kwargs):
        calls.append(("list", kwargs))
        return Rows([capability])

    def get_missing(**kwargs):
        calls.append(("get", kwargs))
        raise NotFound("missing")

    bindings = {"content_desk": SimpleNamespace(capabilities=SimpleNamespace(
        list=list_capabilities, get=get_missing))}
    contract, reports, raw = execute("capabilities-read", {"medium": "text", "capability": "missing"}, bindings)
    result = assert_one_report(contract, reports, raw, "ok")
    assert calls == [("list", {"medium": "text"}), ("get", {"alias": "missing"})]
    assert result["signals"]["total"] == 1
    assert result["signals"]["focus_found"] is False
    assert result["signals"]["readiness"]["definition_status"] == {"defined": 1}

    def unavailable(**kwargs):
        calls.append(("list-error", kwargs))
        raise RuntimeError("connection refused")

    calls.clear()
    bindings = {"content_desk": SimpleNamespace(capabilities=SimpleNamespace(list=unavailable, get=get_missing))}
    contract, reports, raw = execute("capabilities-read", {}, bindings)
    result = assert_one_report(contract, reports, raw, "unavailable")
    assert calls == [("list-error", {})]
    assert result["errors"][0]["class"] == "scenario_unreachable"
    assert result["errors"][0]["where"] == "collect"


def test_board_read_preserves_required_and_optional_read_status_policy():
    bindings, calls = board_bindings()
    contract, reports, raw = execute("board-read", {}, bindings)
    result = assert_one_report(contract, reports, raw, "ok")
    assert calls == [("campaigns", {}), ("drafts", {}), ("capabilities", {}),
                     ("offer_ladder", {"rows": "entries"})]
    assert result["signals"]["changes_status"] == "no_baseline"
    assert result["signals"]["campaign_total"] == result["signals"]["draft_total"] == 0
    assert result["signals"]["capability_read_status"] == result["signals"]["offer_read_status"] == "read"

    bindings, calls = board_bindings(capability_error=RuntimeError("connection refused"),
                                     offer_error=RuntimeError("connection refused"))
    contract, reports, raw = execute("board-read", {}, bindings)
    result = assert_one_report(contract, reports, raw, "partial")
    assert len(calls) == 4
    assert result["signals"]["capability_read_status"] == result["signals"]["offer_read_status"] == "unavailable"
    assert [error["class"] for error in result["errors"]] == [
        "capabilities_unavailable", "offer_readiness_unavailable"]

    bindings, calls = board_bindings(campaign_error=RuntimeError("connection refused"))
    contract, reports, raw = execute("board-read", {}, bindings)
    result = assert_one_report(contract, reports, raw, "unavailable")
    assert len(calls) == 4
    assert result["errors"][0]["class"] == "scenario_unreachable"
    assert result["errors"][0]["where"] == "collect"
