"""Read one authorized Notification Hub conversation without copying it elsewhere."""
inputs = program.inputs()
ask_id = inputs.get("ask_id", "")
subject_key = inputs.get("subject_key", "")
envelope = {
    "program": "notification-hub.conversation-read", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"selector": "ask_id" if ask_id else "subject_key" if subject_key else "none"},
    "signals": {}, "errors": [], "evidence": [],
}


def step_validate():
    selectors = [value for value in (ask_id, subject_key) if isinstance(value, str) and value.strip()]
    if len(selectors) != 1 or (ask_id and subject_key):
        return program.fail("failed", "invalid_input", "provide exactly one of ask_id or subject_key", "validate")
    if not all(isinstance(value, str) and len(value) <= 256 for value in (ask_id, subject_key)):
        return program.fail("failed", "invalid_input", "selector must be a string no longer than 256 characters", "validate")
    return "collect"


def step_collect():
    envelope["phase"] = "collect"
    try:
        args = {"ask_id": ask_id} if ask_id else {"subject_key": subject_key}
        response = notification_hub.conversations.get(**args).raw()
        ask = response.get("ask") if isinstance(response, dict) else None
        if not isinstance(ask, dict):
            return program.fail("failed", "invalid_response", "owner returned no ask object", "collect")
        if ask_id and ask.get("id") != ask_id:
            return program.fail("failed", "invalid_response", "owner returned a different ask ID", "collect")
        if subject_key and ask.get("subjectKey") != subject_key:
            return program.fail("failed", "invalid_response", "owner returned a different subject key", "collect")
        envelope["signals"]["ask"] = ask
        envelope["evidence"].append("notification-hub/conversations/get")
        envelope["status"] = "ok"
    except Exception as exc:
        status, klass = program.classify(exc)
        # Keep private owner error messages out of the program receipt.
        return program.fail(status, klass, "authorized conversation read failed; private details omitted", "collect")
    return "report"


def step_report():
    envelope["phase"] = "report"
    program.report(envelope)
    return None


program.run({"validate": step_validate, "collect": step_collect, "report": step_report}, "validate")
