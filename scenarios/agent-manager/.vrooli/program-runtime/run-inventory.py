"""Read a curated Agent Manager run inventory snapshot."""
import json

inputs = program.inputs()
limit = inputs.get("limit", 20)
offset = inputs.get("offset", 0)
status_filter = inputs.get("status", "")
tag_prefix = inputs.get("tag_prefix", "")
envelope = program.envelope("agent-manager.run-inventory", "1", phase="validate", inputs={})
fail = program.fail
work = {}


def step_validate():
    if not isinstance(limit, int) or isinstance(limit, bool) or not 1 <= limit <= 50:
        return fail("failed", "invalid_input", "limit must be an integer from 1 to 50", "validate")
    if not isinstance(offset, int) or isinstance(offset, bool) or offset < 0:
        return fail("failed", "invalid_input", "offset must be a nonnegative integer", "validate")
    if not isinstance(status_filter, str) or len(status_filter) > 48 or not isinstance(tag_prefix, str) or len(tag_prefix) > 128:
        return fail("failed", "invalid_input", "status and tag_prefix must be bounded strings", "validate")
    envelope["inputs"] = {"limit": limit, "offset": offset, "status": status_filter, "tag_prefix": tag_prefix}
    return "collect"


def step_collect():
    envelope["phase"] = "collect"
    kwargs = {"limit": limit, "offset": offset}
    if status_filter:
        kwargs["status"] = status_filter
    if tag_prefix:
        kwargs["tag_prefix"] = tag_prefix
    try:
        work["runs"] = agent_manager.run.list(**kwargs)
        return "report"
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, str(exc)[:160], "collect")


def step_report():
    envelope["phase"] = "report"
    handle = work["runs"]
    rows = []
    for row in handle.head(limit):
        execution_mode = row.get("executionMode")
        source = row.get("importSourceHarness") if execution_mode == "EXECUTION_MODE_IMPORTED" else "native"
        rows.append({
            "id": row.get("id"), "label": row.get("label"), "status": row.get("status"),
            "phase": row.get("phase"), "execution_mode": execution_mode,
            "progress_percent": row.get("progressPercent"), "source": source,
            "updated_at": row.get("updatedAt"),
        })
    meta = handle.meta()
    page_count = handle.count()
    total = meta.get("total")
    if not isinstance(total, int) or total < 0 or total < offset + page_count:
        return fail("failed", "invalid_response", "run list returned no trustworthy total for this page", "report")
    if page_count > limit or len(rows) > limit:
        return fail("failed", "invalid_response", "run list exceeded the requested page bound", "report")
    envelope["signals"] = {"runs": rows, "count": page_count, "total": total, "has_more": offset + page_count < total, "offset": offset, "limit": limit}
    envelope["status"] = "ok"
    envelope["evidence"] = ["agent-manager/run/list"]
    print(json.dumps(envelope, separators=(",", ":")))
    return None


program.run({"validate": step_validate, "collect": step_collect, "report": step_report}, "validate")
