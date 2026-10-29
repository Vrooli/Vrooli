"""Classify the runtime example corpus through AI Gateway's shared workflow."""

inputs = program.inputs()
envelope = program.envelope("program-runtime.batch-inference", "3")
envelope["signals"] = {}
corpus = inputs.get("corpus", [
    "The provider timed out during a retry.",
    "The user supplied an invalid request field.",
    "The deployment lost its database connection.",
])
labels = inputs.get("labels", ["infra", "user"])


def step_validate():
    if not isinstance(corpus, list) or not 1 <= len(corpus) <= 32:
        return program.fail("failed", "invalid_input", "corpus must contain 1 to 32 texts", "validate")
    envelope["inputs"] = {"documents": len(corpus), "labels": labels}
    return "classify"


def step_classify():
    envelope["phase"] = "classify"
    try:
        child = lib.ai_gateway.classify_batch(
            corpus=corpus, labels=labels,
            instruction=inputs.get("instruction", "Choose the primary failure class."))
        rows = child.head(1)
        if not rows:
            return program.fail("failed", "missing_result", "AI Gateway returned no child envelope", "classify")
        result = rows[0]
        if (not isinstance(result, dict)
                or result.get("status") not in ("ok", "partial", "failed", "unavailable", "refused")
                or not isinstance(result.get("signals"), dict)
                or not isinstance(result.get("errors"), list)
                or not isinstance(result.get("evidence"), list)):
            return program.fail("failed", "invalid_response", "AI Gateway returned a malformed child envelope", "classify")
        envelope["status"] = result["status"]
        envelope["signals"] = result["signals"]
        envelope["errors"] = result["errors"]
        envelope["evidence"] = result["evidence"] + [{"program": "ai-gateway.classify-batch", "artifact": child.meta()}]
    except Exception as exc:
        status, klass = program.classify(exc)
        return program.fail(status, klass, str(exc), "classify")
    return "report"


def step_report():
    envelope["phase"] = "report"
    program.report()
    return None


STATES = {"validate": step_validate, "classify": step_classify, "report": step_report}
program.run(STATES, "validate")
