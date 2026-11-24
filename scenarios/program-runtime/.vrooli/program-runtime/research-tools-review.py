"""Read one authorized tool inventory and preserve distinct health/friction signals."""
import os
import re

inputs = program.inputs()
envelope = program.envelope("program-runtime.research-tools-review", "1")
envelope["signals"] = {"inventory": {}, "scenario_slugs": [], "health": {"status": "unknown"}, "friction": [], "omitted_scenarios": 0}
MAX_SCENARIOS = 16
MAX_FRICTION_SCENARIOS = 8
MAX_FINGERPRINTS = 8
slugs = []
inventory_file = ""


def step_validate():
    global inventory_file
    root = inputs.get("workspace_root", "")
    relative = inputs.get("inventory_relative_path", "")
    days = inputs.get("window_days", 7)
    if (not isinstance(root, str) or not root or "\0" in root or len(root) > 1024
            or not os.path.isabs(root)):
        return program.fail("failed", "invalid_input", "workspace_root must be an absolute authorized directory", "validate")
    if (not isinstance(relative, str) or not relative or "\0" in relative or len(relative) > 1024
            or os.path.isabs(relative) or any(part in ("", ".", "..") for part in relative.split(os.sep))):
        return program.fail("failed", "invalid_input", "inventory_relative_path must be a normalized relative path", "validate")
    if isinstance(days, bool) or not isinstance(days, int) or not 1 <= days <= 90:
        return program.fail("failed", "invalid_input", "window_days must be an integer from 1 through 90", "validate")
    root_real = os.path.realpath(root)
    inventory_file = os.path.realpath(os.path.join(root_real, relative))
    if os.path.commonpath((root_real, inventory_file)) != root_real:
        return program.fail("failed", "path_outside_root", "inventory file resolves outside workspace_root", "validate")
    envelope["inputs"] = {"workspace_root": root_real, "inventory_relative_path": relative, "window_days": days}
    return "collect"


def read_inventory():
    try:
        with open(inventory_file, "r", encoding="utf-8") as source:
            content = source.read(128 * 1024 + 1)
    except OSError as exc:
        envelope["signals"]["inventory"] = {"status": "unavailable", "error": str(exc)[:180]}
        envelope["errors"].append({"class": "inventory_unavailable", "detail": str(exc)[:180], "where": "collect"})
        return False
    if len(content.encode("utf-8")) > 128 * 1024:
        envelope["signals"]["inventory"] = {"status": "partial", "error": "inventory exceeds 128 KiB bound"}
        envelope["errors"].append({"class": "inventory_invalid", "detail": "inventory exceeds 128 KiB bound", "where": "collect"})
        return False
    names = []
    current = None
    for line in content.splitlines():
        if line.startswith("## "):
            current = line[3:].strip()[:120]
            if current:
                names.append(current)
        match = re.match(r"^\s*- \*\*Monitoring scenarios?:\*\*\s*(.+?)\s*$", line)
        if not match:
            continue
        value = match.group(1)
        for slug in re.findall(r"`([a-z0-9][a-z0-9-]{0,62})`", value):
            if slug not in slugs:
                slugs.append(slug)
    envelope["signals"]["inventory"] = {"status": "ok" if slugs else "partial", "capabilities": len(names), "path": envelope["inputs"]["inventory_relative_path"], "has_monitoring_scenarios": bool(slugs)}
    if not slugs:
        envelope["errors"].append({"class": "inventory_invalid", "detail": "no Monitoring scenario fields were found", "where": "collect"})
        return False
    if len(slugs) > MAX_SCENARIOS:
        envelope["signals"]["omitted_scenarios"] = len(slugs) - MAX_SCENARIOS
        del slugs[MAX_SCENARIOS:]
        envelope["signals"]["inventory"]["status"] = "partial"
    envelope["signals"]["scenario_slugs"] = slugs[:]
    envelope["evidence"].append({"path": inventory_file, "scope": "explicitly selected capability inventory"})
    return True


def read_child(name, call):
    try:
        handle = call()
        row = program.child_envelope(handle, invalid_class="invalid_response", where="collect:" + name)
        meta = handle.meta()
        envelope["evidence"].append({"program": name, "artifact": {k: meta[k] for k in ("program", "version", "digest") if k in meta}})
        return row
    except Exception as exc:
        return {"status": "unknown", "signals": {}, "errors": [{"class": "child_unavailable", "detail": str(exc)[:180]}]}


def step_collect():
    envelope["phase"] = "collect"
    if not read_inventory():
        envelope["status"] = "unavailable" if envelope["signals"]["inventory"].get("status") == "unavailable" else "partial"
        return "report"
    days = envelope["inputs"]["window_days"]
    joined = ",".join(slugs)
    health = read_child("program-runtime.supervision-sweep", lambda: lib.program_runtime.supervision_sweep(scenarios=joined))
    envelope["signals"]["health"] = {"status": health.get("status", "unknown"), "signals": health.get("signals", {}), "errors": health.get("errors", []), "evidence": health.get("evidence", [])}
    for slug in slugs[:MAX_FRICTION_SCENARIOS]:
        result = read_child("agent-manager.friction-digest:" + slug, lambda scenario=slug: lib.agent_manager.friction_digest(scenario=scenario, window_days=days, top_n=5))
        signals = result.get("signals", {}) if isinstance(result.get("signals"), dict) else {}
        envelope["signals"]["friction"].append({
            "scenario": slug, "status": result.get("status", "unknown"),
            "coverage": signals.get("coverage"), "profiles": signals.get("profiles_for_scenario"),
            "calls": signals.get("calls_for_scenario"), "findings": signals.get("findings_for_scenario"),
            "recurring_count": signals.get("recurring_count"), "top_fingerprints": signals.get("top_fingerprints", [])[:MAX_FINGERPRINTS],
            "calibration_available": signals.get("supervision_calibration_available"),
            "errors": result.get("errors", [])
        })
    if len(slugs) > MAX_FRICTION_SCENARIOS:
        envelope["signals"]["omitted_scenarios"] = max(envelope["signals"]["omitted_scenarios"], len(slugs) - MAX_FRICTION_SCENARIOS)
    child_statuses = [envelope["signals"]["health"].get("status", "unknown")] + [row.get("status", "unknown") for row in envelope["signals"]["friction"]]
    envelope["status"] = "ok" if envelope["signals"]["inventory"]["status"] == "ok" and not envelope["signals"]["omitted_scenarios"] and all(status == "ok" for status in child_statuses) else "partial"
    return "report"


def step_report():
    envelope["phase"] = "report"
    program.report(envelope)
    return None


program.run({"validate": step_validate, "collect": step_collect, "report": step_report}, "validate")
