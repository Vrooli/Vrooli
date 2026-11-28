"""Read a privacy-bounded summary of one Plan Manager plan."""
import json

inputs = program.inputs()
plan_id = inputs.get("id", "")
envelope = program.envelope("plan-manager.plan-read", "1", phase="validate", inputs={})
fail = program.fail
work = {}


def step_validate():
    if not isinstance(plan_id, str) or not plan_id.strip() or len(plan_id) > 128:
        return fail("failed", "invalid_input", "id must be a nonempty plan id or slug up to 128 characters", "validate")
    envelope["inputs"] = {"id": plan_id}
    return "collect"


def step_collect():
    envelope["phase"] = "collect"
    try:
        work["plan"] = plan_manager.plans.get(id=plan_id)
        return "classify"
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, str(exc)[:160], "collect")


def step_classify():
    meta = work["plan"].meta()
    plan = meta.get("plan")
    if not isinstance(plan, dict) or not plan.get("id") or not plan.get("slug"):
        return fail("failed", "invalid_response", "GetPlan response lacks plan identity", "classify")
    if plan_id not in (plan.get("id"), plan.get("slug")):
        return fail("failed", "identity_mismatch", "GetPlan returned a different plan", "classify")
    phases = plan.get("phases") or []
    if not isinstance(phases, list):
        return fail("failed", "invalid_response", "plan phases is not a list", "classify")
    work["signals"] = {
        "plan": {key: plan.get(key) for key in ("id", "slug", "title", "status", "updatedAt", "contentHash") if key in plan},
        "phase_count": len(phases),
        "phases": [{key: phase.get(key) for key in ("id", "title", "status") if key in phase} | {"step_count": len(phase.get("steps") or [])} for phase in phases[:20] if isinstance(phase, dict)],
        "phases_truncated": len(phases) > 20,
    }
    return "report"


def step_report():
    envelope["phase"] = "report"
    envelope["signals"] = work["signals"]
    envelope["status"] = "partial" if work["signals"]["phases_truncated"] else "ok"
    if work["signals"]["phases_truncated"]:
        envelope["errors"].append({"class": "output_limit", "detail": "phase index capped at 20 entries", "where": "report"})
    envelope["evidence"] = ["plan-manager/plans/get"]
    print(json.dumps(envelope, separators=(",", ":")))
    return None


program.run({"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}, "validate")
