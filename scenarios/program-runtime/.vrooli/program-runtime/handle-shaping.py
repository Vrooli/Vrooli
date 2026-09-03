"""program-runtime.handle-shaping v1 — shape rows in the kernel without materializing the source handles.

Contract: handle-shaping.json.
Skill:    program-runtime (usage tree: "join or reshape rows before reading them").
Demonstrates: filter, join, sort, select, agg on Handles; the output is head(3) and one aggregate.

Phases: validate -> classify -> report. No binding, no inference.
"""

try:
    inputs
except NameError:
    inputs = {}
top_n = int(inputs.get("top_n", 3))

envelope = {
    "program": "program-runtime.handle-shaping", "version": "1",
    "status": "failed", "phase": "validate", "inputs": {"top_n": top_n},
    "signals": {"top": [], "paid_total": None, "paid_count": 0}, "errors": [], "evidence": [],
}


def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:240], "where": where})
    return "report"


def classify_transport(exc):
    """Map a bridge exception to (status, class). Copied verbatim from program-contracts.md."""
    if isinstance(exc, (NameError, AttributeError)):
        raise exc                                   # kernel_runtime: a bound name is missing; never relabel
    text = str(exc)
    for needle in ("is unreachable", "bridge unavailable", "scenario_not_running",
                   "no running runtime ports", "connection refused"):
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


def step_validate():  # VALIDATE
    if not (1 <= top_n <= 10):
        return fail("failed", "invalid_input", f"top_n={top_n} outside 1..10", "validate")
    return "classify"


def step_classify():  # CLASSIFY · in-kernel shaping over local Handles
    envelope["phase"] = "classify"
    orders = Handle([
        {"customer_id": "c-1", "status": "paid", "amount": 24},
        {"customer_id": "c-2", "status": "paid", "amount": 31},
        {"customer_id": "c-1", "status": "paid", "amount": 18},
        {"customer_id": "c-3", "status": "pending", "amount": 9},
    ], "orders")
    customers = Handle([
        {"customer_id": "c-1", "name": "Ada"},
        {"customer_id": "c-2", "name": "Grace"},
        {"customer_id": "c-3", "name": "Edsger"},
    ], "customers")
    try:
        paid = orders.filter(lambda row: row["status"] == "paid")
        shaped = paid.join(customers, "customer_id").sort("amount", reverse=True).select("name", "amount")
        envelope["signals"]["top"] = shaped.head(top_n)
        envelope["signals"]["paid_total"] = paid.agg("amount", "sum")
        envelope["signals"]["paid_count"] = paid.count()
    except Exception as exc:
        return fail("failed", "handle_error", exc, "classify")
    envelope["evidence"].append("local Handles: orders(4) x customers(3)")
    envelope["status"] = "ok"
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "classify": step_classify, "report": step_report}
state = "validate"
while state:
    try:
        state = STATES[state]()
    except Exception as exc:  # the one catch that guarantees an envelope on every path
        if envelope.get("phase") == "report":
            raise
        envelope["status"] = "failed"
        envelope["errors"].append({"class": "kernel_runtime", "detail": str(exc)[:240], "where": envelope.get("phase") or state})
        state = "report"
