"""The workspace review distinguishes absent evidence from incomplete reads."""
import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parents[3] / "kernel"))
from host.standalone import execute as run_program  # noqa: E402

SOURCE = Path(__file__).parents[1] / "research-workspace-review.py"


def execute(root):
    return run_program(str(SOURCE), {"research_root": str(root)})[1]


def test_aliases_are_counted_once_and_current_view_size_is_distinct(tmp_path):
    runs = tmp_path / "topics" / "one" / "runs"
    actual = runs / "actual"
    actual.mkdir(parents=True)
    (actual / "README.md").write_text("guide")
    (actual / "CHECKPOINT.md").write_text("checkpoint")
    (actual / "CURRENT.md").write_text("now")
    (runs / "alias").symlink_to(actual, target_is_directory=True)
    observed = execute(tmp_path)
    assert observed["status"] == "ok"
    assert observed["signals"]["unique_runs"] == 1
    assert observed["signals"]["largest"][0]["checkpoint_bytes"] == 10
    assert observed["signals"]["largest"][0]["current_view_bytes"] == 3


def test_outside_alias_is_not_inspected_and_coverage_is_partial(tmp_path):
    root = tmp_path / "research"
    runs = root / "topics" / "one" / "runs"
    runs.mkdir(parents=True)
    outside = tmp_path / "outside"
    outside.mkdir()
    (runs / "escape").symlink_to(outside, target_is_directory=True)
    observed = execute(root)
    assert observed["status"] == "partial"
    assert observed["signals"]["skipped_outside_root"] == 1
    assert observed["signals"]["unique_runs"] == 0


def test_missing_workspace_is_unavailable_not_an_empty_success(tmp_path):
    observed = execute(tmp_path)
    assert observed["status"] == "unavailable"
    assert observed["errors"][0]["class"] == "workspace_unavailable"


def test_relative_root_is_rejected():
    for root in ("relative", "/invalid\0path"):
        observed = execute(root)
        assert observed["status"] == "failed"
        assert observed["errors"][0]["class"] == "invalid_input"


def test_unreadable_guide_is_unknown_not_absent(tmp_path, monkeypatch):
    run = tmp_path / "topics" / "one" / "runs" / "one"
    run.mkdir(parents=True)
    (run / "README.md").write_text("guide")
    real_stat = os.stat
    def denied(path, *args, **kwargs):
        if str(path) == str(run / "README.md"):
            raise PermissionError("denied")
        return real_stat(path, *args, **kwargs)
    monkeypatch.setattr(os, "stat", denied)
    observed = execute(tmp_path)
    assert observed["status"] == "partial"
    assert observed["signals"]["without_guide"] == 0
    assert observed["errors"][0]["class"] == "incomplete_read"


def test_topic_bound_discloses_incomplete_coverage(tmp_path):
    topics = tmp_path / "topics"
    topics.mkdir()
    for number in range(513):
        (topics / ("topic%03d" % number)).mkdir()
    observed = execute(tmp_path)
    assert observed["status"] == "partial"
    assert observed["signals"]["truncated"] is True
