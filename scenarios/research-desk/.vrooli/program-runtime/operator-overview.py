import json

inputs = program.inputs()
envelope = {
    "program": "research-desk.operator-overview",
    "version": "1",
    "status": "failed",
    "phase": "validate",
    "inputs": {},
    "signals": {},
    "errors": [],
    "evidence": [],
}
fail = program.fail
clipped = False
TOPIC_ROW_LIMIT = 8


def bound(value, depth=0):
    global clipped
    if isinstance(value, str):
        if len(value) > 512:
            clipped = True
            return value[:512] + "…"
        return value
    if isinstance(value, list):
        if len(value) > 8:
            clipped = True
        return [bound(item, depth + 1) for item in value[:8]]
    if isinstance(value, dict):
        if depth >= 6 and value:
            clipped = True
            return {"truncated": True}
        return {key: bound(item, depth + 1) for key, item in value.items()}
    return value


def step_validate():
    if not isinstance(inputs, dict) or set(inputs) != {"view"} or inputs.get("view") not in ("needs-you", "topics"):
        return fail("failed", "invalid_input", "view must be needs-you or topics", "validate")
    envelope["inputs"] = {"view": inputs["view"]}
    return "collect"


def step_collect():
    envelope["phase"] = "collect"
    binding_id = "research-desk/operator/needs-you" if inputs["view"] == "needs-you" else "research-desk/activity/topics"
    row_key = "items" if inputs["view"] == "needs-you" else "topics"
    try:
        if inputs["view"] == "needs-you":
            result = research_desk.operator.needs_you(rows="items")
        else:
            result = research_desk.activity.topics(rows="topics")
        metadata = dict(result.meta())
        if inputs["view"] == "needs-you":
            # The inbox contains private decision bodies, options, answers and paths.
            # Keep only owner-reported counts and observation completeness in the envelope.
            unreadable = metadata.get("unreadable")
            unreadable_count = len(unreadable) if isinstance(unreadable, list) else None
            item_count = metadata.get("itemCount")
            older_count = metadata.get("olderListCount")
            safe_item_count = item_count if type(item_count) is int and 0 <= item_count <= 1_000_000_000 else None
            safe_older_count = older_count if type(older_count) is int and 0 <= older_count <= 1_000_000_000 else None
            observed_at = metadata.get("observedAt")
            known_counts = safe_item_count is not None and safe_older_count is not None
            envelope["signals"]["result"] = {
                "item_count": safe_item_count,
                "older_list_count": safe_older_count,
                "observed_at": observed_at[:64] if isinstance(observed_at, str) else None,
                "unreadable_count": unreadable_count,
                "details_redacted": True,
                "truncated": False,
            }
            envelope["evidence"].append(binding_id)
            envelope["status"] = "ok" if known_counts and unreadable_count == 0 else "partial"
            return "report"
        if inputs["view"] == "topics":
            rows = result.head(TOPIC_ROW_LIMIT)
            def safe_count(value):
                return value if type(value) is int and 0 <= value <= 1_000_000_000 else None
            summaries = []
            for row in rows:
                if not isinstance(row, dict):
                    continue
                check = row.get("missionCheck", row.get("mission_check", {}))
                if not isinstance(check, dict):
                    check = {}
                topic = row.get("topic")
                if not isinstance(topic, str):
                    continue
                summaries.append({
                    "topic": topic[:128],
                    "commission_count": len(row.get("commissionIds", row.get("commission_ids", []))) if isinstance(row.get("commissionIds", row.get("commission_ids", [])), list) else None,
                    "live_count": len(row.get("liveCommissionIds", row.get("live_commission_ids", []))) if isinstance(row.get("liveCommissionIds", row.get("live_commission_ids", [])), list) else None,
                    "paused_count": len(row.get("pausedCommissionIds", row.get("paused_commission_ids", []))) if isinstance(row.get("pausedCommissionIds", row.get("paused_commission_ids", [])), list) else None,
                    "mission_open": safe_count(check.get("open")),
                    "mission_active": safe_count(check.get("active")),
                    "mission_waiting_operator": safe_count(check.get("waitingOperator", check.get("waiting_operator"))),
                    "mission_landed": safe_count(check.get("landed")),
                })
            count = result.count()
            envelope["signals"]["result"] = {
                "topic_count": count,
                "rows": summaries,
                "truncated": count > len(summaries),
            }
            envelope["evidence"].append(binding_id)
            envelope["status"] = "partial" if count > len(summaries) else "ok"
            return "report"
        metadata.pop(row_key, None)
        rows = result.head(100)
        envelope["signals"]["result"] = bound(metadata)
        envelope["signals"]["result"]["rows"] = bound(rows)
        envelope["signals"]["result"]["rowCount"] = result.count()
        envelope["signals"]["result"]["truncated"] = result.count() > len(rows) or clipped
        envelope["evidence"].append(binding_id)
        envelope["status"] = "partial" if envelope["signals"]["result"]["truncated"] else "ok"
        return "report"
    except Exception as exc:
        status, klass = program.classify(exc)
        # Owner errors can contain private inbox details; expose only a safe status/class.
        detail = "private owner read failed; details omitted"
        return fail(status, klass, detail, "collect")


def step_report():
    envelope["phase"] = "report"
    print(json.dumps(envelope, ensure_ascii=False, allow_nan=False, separators=(",", ":")))
    return None


program.run({"validate": step_validate, "collect": step_collect, "report": step_report}, "validate")
