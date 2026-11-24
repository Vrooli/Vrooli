"""Focused checks for generated helper fixtures and parent source execution."""
import contextlib
import io
import json
import sys
from collections import Counter
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from fault_matrix import library_pins, main


def test_declared_child_digest_pin_resolves_literal_assignment():
    source = """PIN = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
handle = lib.example.child()
result = program.child_envelope(handle, expected_digest=PIN)
"""
    assert library_pins(source) == {"example.child": "a" * 64}


def test_generated_fault_matrix_discovers_library_and_records_requirements_and_fallbacks():
    output = io.StringIO()
    with contextlib.redirect_stdout(output):
        assert main() == 0
    report = json.loads(output.getvalue())

    assert report["evidence_kind"] == "generated_fixture"
    assert report["native_qualification"] is False
    assert report["program_count"] == 19
    assert report["fault_classes"] == ["unavailable", "degraded", "healthy-empty", "malformed", "partial"]
    assert report["row_count"] == report["passed_rows"] == 95
    assert report["failed_rows"] == 0
    for row in report["matrix"]:
        assert row["capability_requirements"]
        fallback = row["observable_fallback"]
        assert fallback["behavior"]
        assert fallback["status"] == row["observed_status"]

    source_execution = report["program_source_execution"]
    assert source_execution["execution_kind"] == "program_runtime_session_with_injected_library_handles"
    assert source_execution["native_qualification"] is False
    assert source_execution["program_count"] == 19
    assert source_execution["fault_classes"] == report["fault_classes"]
    assert source_execution["row_count"] == source_execution["passed_rows"] == 95
    assert source_execution["failed_rows"] == 0
    assert len(source_execution["matrix"]) == 95
    source_programs = {row["program"] for row in source_execution["matrix"]}
    assert source_programs == {row["program"] for row in report["matrix"]}
    assert Counter(row["fault"] for row in source_execution["matrix"]) == Counter({fault: 19 for fault in report["fault_classes"]})
    assert len({(row["program"], row["fault"]) for row in source_execution["matrix"]}) == 95
    for row in source_execution["matrix"]:
        assert row["passed"] is True
        assert row["native_qualification"] is False
        assert row["execution_kind"] == source_execution["execution_kind"]
        assert row["injected_dependencies"]
        assert row["observed_child_envelopes"]
        assert row["no_retry"] is True
        assert row["capability_requirements"]
        assert row["parent_status"]
        assert row["parent_phase"]
        assert row["parent_status"] in row["parent_status_enum"]
        assert row["parent_phase"] in row["parent_phase_enum"]
        fallback = row["observable_fallback"]
        assert fallback["observed"] is True
        assert fallback["status"] == row["parent_status"]
        assert fallback["phase"] == row["parent_phase"]
        assert row["parent_status"] != "ok" or fallback["error_classes"] or fallback["signal_markers"]
        for child in row["observed_child_envelopes"]:
            assert child["status"] == row["expected_status_by_dependency"][child["dependency"]]
