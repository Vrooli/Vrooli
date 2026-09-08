import json
import re

try:
    inputs
except NameError:
    inputs = {}
inputs = inputs if isinstance(inputs, dict) else {}

url = str(inputs.get("url", "") or "").strip()
viewport = str(inputs.get("viewport", "desktop") or "desktop").strip().lower()
theme = str(inputs.get("theme", "light") or "light").strip().lower()
direction = str(inputs.get("direction", "ltr") or "ltr").strip().lower()
wait_for = str(inputs.get("wait_for", "") or "").strip()
wait_until = str(inputs.get("wait_until", "load") or "load").strip().lower()
settle_ms = inputs.get("settle_ms", 500)
device_scale_factor = inputs.get("device_scale_factor", 1)

envelope = {
    "program": "browser-automation-studio.capture-surface", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"url": url, "viewport": viewport, "theme": theme, "direction": direction,
               "wait_for": wait_for or None, "wait_until": wait_until, "settle_ms": settle_ms,
               "device_scale_factor": device_scale_factor},
    "signals": {"artifact_id": None, "screenshot_ref": None, "snapshot_ref": None, "url": url,
                "viewport": viewport, "theme": theme, "direction": direction,
                "captured_at": None, "readiness": None},
    "errors": [], "evidence": []
}


def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where})
    return "report"


def classify_transport(exc):
    if isinstance(exc, (NameError, AttributeError)):
        raise exc
    text = str(exc)
    for needle in ("is unreachable", "bridge unavailable", "scenario_not_running", "no running runtime ports", "connection refused"):
        if needle in text:
            return ("unavailable", "scenario_unreachable")
    if "requires an explicit grant" in text:
        return ("refused", "no_grant")
    if "not run eligible" in text or "run_eligible" in text:
        return ("refused", "not_run_eligible")
    if "inference spend" in text:
        return ("refused", "inference_spend_exceeded")
    if "delegated run spend" in text:
        return ("refused", "delegated_run_spend_exceeded")
    if "no determinable primary response field" in text or "rows must be one of" in text:
        return ("failed", "ambiguous_response")
    for needle in ("accepts named proto fields", "invalid arguments for", "no proto field matches"):
        if needle in text:
            return ("failed", "invalid_input")
    if "deadline" in text:
        return ("failed", "deadline_exceeded")
    return ("failed", "binding_error")


def step_validate():
    if not url or not re.match(r"^https?://", url):
        return fail("failed", "invalid_input", "url must be an absolute http(s) URL", "validate")
    if viewport not in ("mobile", "tablet", "desktop") and not re.match(r"^\d+x\d+$", viewport):
        return fail("failed", "invalid_input", "viewport must be mobile, tablet, desktop, or WIDTHxHEIGHT", "validate")
    if theme not in ("light", "dark", "system") or direction not in ("ltr", "rtl"):
        return fail("failed", "invalid_input", "theme or direction is outside the declared enum", "validate")
    if wait_until not in ("load", "domcontentloaded", "networkidle"):
        return fail("failed", "invalid_input", "wait_until is outside the declared enum", "validate")
    if type(settle_ms) is not int or settle_ms < 0 or settle_ms > 15000:
        return fail("failed", "invalid_input", "settle_ms must be between 0 and 15000", "validate")
    if type(device_scale_factor) not in (int, float) or device_scale_factor < 0.5 or device_scale_factor > 4:
        return fail("failed", "invalid_input", "device_scale_factor must be between 0.5 and 4", "validate")
    return "act"


def dimensions_kwargs():
    if viewport == "mobile":
        return {"width": 390, "height": 844}
    if viewport == "tablet":
        return {"width": 768, "height": 1024}
    if viewport == "desktop":
        return {"width": 1440, "height": 900}
    width, height = viewport.split("x", 1)
    return {"width": int(width), "height": int(height)}


def step_act():
    envelope["phase"] = "act"
    readiness = wait_for
    if not readiness:
        readiness = {"networkidle": True} if wait_until == "networkidle" else {"timeoutMs": settle_ms}
    elif readiness.lower() == "networkidle":
        readiness = {"networkidle": True}
    else:
        readiness = {"selector": readiness}
    if wait_for and wait_until == "networkidle" and wait_for.lower() != "networkidle":
        return fail("failed", "invalid_input", "wait_for selector cannot be combined with networkidle in one capture request", "act")
    profile = {"fingerprint": {"colorScheme": "no-preference" if theme == "system" else theme}}
    kwargs = {
        "url": url, "capture": ["CAPTURE_TYPE_SCREENSHOT", "CAPTURE_TYPE_DOM_TREE"], "wait_for": readiness,
        "device_scale_factor": float(device_scale_factor), "inline_dom_tree": True,
        "direction": direction, "browser_profile": profile
    }
    kwargs.update(dimensions_kwargs())
    try:
        result = browser_automation_studio.capture.capture(**kwargs)
        rows = result.head(16)
        meta = result.meta()
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "act")
    by_type = {}
    for row in rows:
        kind = str(row.get("type") or row.get("captureType") or "").lower()
        by_type[kind] = row
    execution_id = meta.get("executionId") or meta.get("execution_id")
    envelope["signals"]["artifact_id"] = execution_id
    envelope["signals"]["captured_at"] = meta.get("capturedAt") or meta.get("captured_at")
    envelope["signals"]["readiness"] = meta.get("readiness")
    screenshot = by_type.get("capture_type_screenshot") or by_type.get("screenshot")
    snapshot = by_type.get("capture_type_dom_tree") or by_type.get("dom_tree")
    envelope["signals"]["screenshot_ref"] = (screenshot or {}).get("reference")
    envelope["signals"]["snapshot_ref"] = (snapshot or {}).get("reference")
    if execution_id:
        envelope["evidence"].append("execution:" + str(execution_id))
    if envelope["signals"]["screenshot_ref"]:
        envelope["evidence"].append("screenshot:" + str(envelope["signals"]["screenshot_ref"]))
    if envelope["signals"]["snapshot_ref"]:
        envelope["evidence"].append("snapshot:" + str(envelope["signals"]["snapshot_ref"]))
    if not envelope["signals"]["screenshot_ref"]:
        envelope["errors"].append({"class": "screenshot_failed", "detail": "capture returned no screenshot reference", "where": "act"})
    if not envelope["signals"]["snapshot_ref"]:
        envelope["errors"].append({"class": "snapshot_failed", "detail": "capture returned no computed snapshot reference", "where": "act"})
    envelope["status"] = "ok" if not envelope["errors"] else ("partial" if envelope["signals"]["screenshot_ref"] or envelope["signals"]["snapshot_ref"] else "failed")
    return "report"


def step_report():
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False))
    return None


state = "validate"
while state:
    try:
        state = {"validate": step_validate, "act": step_act, "report": step_report}[state]()
    except Exception as exc:
        if envelope.get("phase") == "report":
            raise
        envelope["status"] = "failed"
        envelope["errors"].append({"class": "kernel_runtime", "detail": str(exc)[:240], "where": envelope.get("phase") or state})
        state = "report"
