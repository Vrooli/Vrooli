"""Archive only explicitly reviewed terminal snapshots through the Swarm owner."""

inputs = program.inputs()
envelope = program.envelope("swarm-manager.archive-terminal", "1")
selected = inputs.get("items", [])


def step_validate():
    envelope["phase"] = "validate"
    if not isinstance(selected, list) or not (1 <= len(selected) <= 50):
        return program.fail("failed", "invalid_input", "one to 50 reviewed terminal snapshots are required", "validate")
    for item in selected:
        if not isinstance(item, dict) or item.get("expected_status") not in ["completed", "dropped"] or not item.get("expected_updated") or not item.get("kind") or not item.get("name"):
            return program.fail("failed", "invalid_input", "every item needs kind, name, expected_status and expected_updated", "validate")
    return "collect"


def step_collect():
    envelope["phase"] = "collect"
    describe("swarm-manager/backlog/archive-terminal")
    envelope["signals"]["selected"] = len(selected)
    return "decide"


def step_decide():
    envelope["phase"] = "decide"
    envelope["signals"]["selected"] = len(selected)
    return "act"


def step_act():
    envelope["phase"] = "act"
    try:
        result = swarm_manager.backlog.archive_terminal(items=selected, execute=inputs.get("execute", False), _confirm=True)
        meta = result.meta()
        codes = dict(result.group_by("code"))
        envelope["signals"].update({"executed": bool(meta.get("executed", False)), "eligible": int(meta.get("eligibleCount", 0)), "archived": int(meta.get("archivedCount", 0)), "skipped": int(meta.get("skippedCount", 0)), "codes": codes, "examples": result.head(5)})
        envelope["evidence"] = ["backlog:" + item["kind"] + "/" + item["name"] for item in selected]
        rejected = sum(codes.get(code, 0) for code in ["changed", "refused", "missing", "unavailable"])
        envelope["status"] = "partial" if rejected else "ok"
        if codes.get("refused", 0) == len(selected):
            envelope["status"] = "refused"
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, str(exc), "act")
    return "report"


def step_report():
    envelope["phase"] = "report"
    program.report()
    return None


program.run({"validate": step_validate, "collect": step_collect, "decide": step_decide, "act": step_act, "report": step_report}, "validate")
