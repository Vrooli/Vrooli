"""List Plan Manager inventory without disclosing plan-authored content."""
import json

inputs = program.inputs()
limit = inputs.get("limit", 40)
offset = inputs.get("offset", 0)
envelope = program.envelope("plan-manager.plan-inventory", "1", phase="validate", inputs={})
fail = program.fail
work = {}


def step_validate():
    if not isinstance(limit, int) or isinstance(limit, bool) or not 1 <= limit <= 40:
        return fail("failed", "invalid_input", "limit must be an integer from 1 to 40", "validate")
    if not isinstance(offset, int) or isinstance(offset, bool) or offset < 0:
        return fail("failed", "invalid_input", "offset must be a nonnegative integer", "validate")
    envelope["inputs"] = {"limit": limit, "offset": offset, "include_archived": False}
    return "collect"


def step_collect():
    envelope["phase"] = "collect"
    try:
        work["plans"] = plan_manager.plans.list(include_archived=False, limit=limit, offset=offset)
        return "report"
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, str(exc)[:160], "collect")


def step_report():
    envelope["phase"] = "report"
    handle = work["plans"]
    plans = []
    for row in handle.head(limit):
        plans.append({key: row.get(key) for key in ("id", "slug", "status", "updatedAt") if key in row})
    meta = handle.meta()
    source_count = handle.count()
    total = meta.get("total")
    if not isinstance(total, int) or total < 0 or total < offset + len(plans):
        return fail("failed", "invalid_response", "plan list returned no trustworthy total for this page", "report")
    if source_count > limit or len(plans) > limit:
        return fail("failed", "invalid_response", "plan list binding ignored the requested page bound", "report")
    has_more = offset + len(plans) < total
    envelope["signals"] = {"plans": plans, "count": len(plans), "total": total, "has_more": has_more, "offset": offset, "limit": limit}
    envelope["status"] = "partial" if has_more else "ok"
    envelope["evidence"] = ["plan-manager/plans/list"]
    print(json.dumps(envelope, separators=(",", ":")))
    return None


program.run({"validate": step_validate, "collect": step_collect, "report": step_report}, "validate")
