"""Screenshot a named phone or TV, or try an app on it: optionally install an APK, open it, screenshot it."""
import json

inputs = program.inputs()
learn.task(operation="device-control.check-app", key={"device": inputs.get("device", ""), "app": inputs.get("app", "")})

envelope = {"program": "device-control.check-app", "version": "1", "status": "failed", "phase": "validate", "signals": {"outcome": "unknown"}, "errors": [], "evidence": []}


def lines(value):
    return [line for line in str(value or "").split("\n") if line][:5]


try:
    if not isinstance(inputs, dict):
        raise ValueError("object inputs required")
    for name in ("device", "actor"):
        if not isinstance(inputs.get(name), str) or not inputs[name].strip():
            raise ValueError("required string: " + name)
    apk, app = inputs.get("apk") or "", inputs.get("app") or ""
    shot = bool(inputs.get("screenshot", True))
    if not apk and not app and not shot:
        raise ValueError("apk, app, or screenshot is required")
    envelope["phase"] = "act"
    if not apk and not app:
        # Screenshot only: wake the screen and capture whatever is showing.
        captured = device_control.device.screenshot(device=inputs["device"], actor=inputs["actor"], kind="screenshot", seconds=0, package="", wake=True, home=bool(inputs.get("home", False)))
        c = captured.raw() if hasattr(captured, "raw") else captured.meta()
        shots = [str(i) for i in (c.get("evidenceIds") or c.get("evidence_ids") or [])][:5]
        ok = bool(shots)
        envelope["signals"].update({"launch_status": "", "detail": str(c.get("message", ""))[:240] if not ok else "", "next_actions": lines(c.get("nextAction", c.get("next_action", "")))})
    elif apk:
        # One owner call installs, proves the package, opens it and screenshots it.
        response = device_control.device.install(device=inputs["device"], actor=inputs["actor"], apk=apk, open=True, screenshot=shot)
        payload = response.raw() if hasattr(response, "raw") else response.meta()
        ok = payload.get("status") == "installed_verified" and payload.get("launchStatus", payload.get("launch_status")) == "foreground_verified"
        envelope["signals"].update({
            "install_status": payload.get("status", "failed"),
            "package": payload.get("package", ""),
            "version": payload.get("versionName", payload.get("version_name", "")),
            "launch_status": payload.get("launchStatus", payload.get("launch_status", "")),
            "foreground_package": payload.get("foregroundPackage", payload.get("foreground_package", "")),
            "detail": str(payload.get("detail", ""))[:240],
            "next_actions": lines(payload.get("nextAction", payload.get("next_action", ""))),
        })
        shots = [str(i) for i in (payload.get("evidenceIds") or payload.get("evidence_ids") or [])][:5]
    else:
        # Already installed (the TV path): open, then capture. No install is sent.
        opened = device_control.device.open(device=inputs["device"], actor=inputs["actor"], app=app, wake=True)
        o = opened.raw() if hasattr(opened, "raw") else opened.meta()
        ok = o.get("status") == "foreground_verified"
        envelope["signals"].update({
            "launch_status": o.get("status", "failed"),
            "package": o.get("package", ""),
            "foreground_package": o.get("foregroundPackage", o.get("foreground_package", "")),
            "detail": str(o.get("detail", ""))[:240],
            "next_actions": lines(o.get("nextAction", o.get("next_action", ""))),
        })
        shots = []
        if ok and shot:
            captured = device_control.device.screenshot(device=inputs["device"], actor=inputs["actor"], kind="screenshot", seconds=0, package="", wake=False, home=False)
            c = captured.raw() if hasattr(captured, "raw") else captured.meta()
            shots = [str(i) for i in (c.get("evidenceIds") or c.get("evidence_ids") or [])][:5]
            if not shots:
                ok = False
                envelope["signals"]["detail"] = str(c.get("message", "screenshot failed"))[:240]
                envelope["signals"]["next_actions"] = lines(c.get("nextAction", c.get("next_action", "")))
    if ok and (shots or not shot):
        envelope["status"] = "ok"
        envelope["signals"]["outcome"] = "verified_success"
        envelope["evidence"] = shots or ["foreground:" + envelope["signals"]["foreground_package"]]
    else:
        envelope["status"] = "failed"
        envelope["errors"].append({"class": "check_failed", "where": "act", "detail": envelope["signals"].get("detail") or "the app did not reach the foreground with a screenshot"})
    envelope["signals"]["fetch"] = "device-control device evidence <evidence_id> --out <file>" if shots else ""
except ValueError as exc:
    envelope["errors"].append({"class": "invalid_input", "where": envelope["phase"], "detail": str(exc)[:180]})
except Exception as exc:
    envelope["status"] = "unavailable"
    envelope["errors"].append({"class": "binding_error", "where": envelope["phase"], "detail": str(exc)[:180]})
envelope["phase"] = "report"
status = envelope["signals"].get("outcome", "unknown")
envelope["signals"]["learning"] = {"feedback_ref": learn.result("check-app")}
learn.outcome(status if status in ("verified_success", "failed", "unavailable", "unknown") else "unknown",
              envelope["evidence"] if status == "verified_success" else [])
print(json.dumps(envelope, allow_nan=False))
