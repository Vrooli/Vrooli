"""Bounded mutation pilot program; Unit Health remains the execution owner."""
import re

inputs = program.inputs()
env = program.envelope("unit-health.mutation-pilot", "1", inputs=inputs)
env["signals"] = {"summary": {}, "receipts": []}


def fail(status, klass, detail, where):
    env["status"] = status
    env["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where})
    return "report"


def step_validate():
    if not isinstance(inputs, dict):
        return fail("failed", "invalid_input", "inputs must be an object", "validate")
    scenario = inputs.get("scenario")
    if not isinstance(scenario, str) or not re.fullmatch(r"[a-z0-9][a-z0-9-]{0,127}", scenario):
        return fail("failed", "invalid_input", "scenario must be an exact scenario id", "validate")
    seed = inputs.get("seed")
    if not isinstance(seed, str) or not seed.strip() or len(seed) > 256:
        return fail("failed", "invalid_input", "seed is required and bounded", "validate")
    operators = inputs.get("operators", ["boundary", "negate-condition", "return-constant"])
    if not isinstance(operators, list) or not operators or len(operators) > 3 or any(not isinstance(item, str) for item in operators):
        return fail("failed", "invalid_input", "operators must be a bounded non-empty array", "validate")
    supported = {"boundary", "negate-condition", "return-constant"}
    if any(item not in supported for item in operators):
        return fail("failed", "invalid_input", "operators contain an unsupported value", "validate")
    max_mutants = inputs.get("max_mutants", 20)
    if type(max_mutants) is not int or max_mutants < 1 or max_mutants > 50:
        return fail("failed", "invalid_input", "max_mutants must be within [1,50]", "validate")
    workspace = inputs.get("workspace", "api")
    package = inputs.get("package", "./internal/testquality/...")
    if not isinstance(workspace, str) or not workspace or not isinstance(package, str) or not package:
        return fail("failed", "invalid_input", "workspace and package are required", "validate")
    env["inputs"] = {"scenario": scenario, "workspace": workspace, "package": package, "operators": operators, "max_mutants": max_mutants, "seed": seed}
    env["status"] = "ok"
    return "collect"


def value(container, snake, camel, default=None):
    if not isinstance(container, dict):
        return default
    return container.get(snake, container.get(camel, default))


def step_collect():
    env["phase"] = "collect"
    args = dict(env["inputs"])
    try:
        read = program.guarded(lambda: unit_health.mutation.pilot(**args, rows="receipts"))
        handle = read()
        if isinstance(handle, Exception):
            status, klass = program.classify(handle)
            return fail("unavailable" if klass == "scenario_unreachable" else "partial", klass, str(handle), "collect")
        payload = handle.raw() if hasattr(handle, "raw") else {}
        if not isinstance(payload, dict):
            return fail("partial", "binding_error", "mutation binding returned no object", "collect")
        summary = value(payload, "summary", "summary", {}) or {}
        receipts = value(payload, "receipts", "receipts", []) or []
        env["signals"]["summary"] = {
            "generated": int(value(summary, "generated", "generated", 0) or 0),
            "killed": int(value(summary, "killed", "killed", 0) or 0),
            "survived": int(value(summary, "survived", "survived", 0) or 0),
            "invalid": int(value(summary, "invalid", "invalid", 0) or 0),
            "equivalent": int(value(summary, "equivalent", "equivalent", 0) or 0),
            "out_of_contract": int(value(summary, "out_of_contract", "outOfContract", 0) or 0),
            "infrastructure_failure": int(value(summary, "infrastructure_failure", "infrastructureFailure", 0) or 0),
            "unknown": int(value(summary, "unknown", "unknown", 0) or 0),
            "kill_rate": float(value(summary, "kill_rate", "killRate", 0.0) or 0.0),
        }
        env["signals"]["receipts"] = receipts[:50]
        env["evidence"].append({"binding": "unit-health/mutation/pilot", "run_id": str(value(payload, "run_id", "runId", ""))[:96], "workspace_path": str(value(payload, "workspace_path", "workspacePath", ""))[:240], "generated": len(receipts)})
        limitations = value(payload, "limitations", "limitations", []) or []
        if limitations:
            env["errors"].append({"class": "partial_results", "detail": "; ".join(str(item) for item in limitations)[:240], "where": "collect"})
            env["status"] = "partial"
        else:
            env["status"] = "ok"
        return "report"
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail("unavailable" if klass == "scenario_unreachable" else "partial", klass, str(exc), "collect")


def step_report():
    env["phase"] = "report"
    return env


state = "validate"
while state:
    if state == "validate":
        state = step_validate()
    elif state == "collect":
        state = step_collect()
    elif state == "report":
        step_report()
        program.report()
        state = None
