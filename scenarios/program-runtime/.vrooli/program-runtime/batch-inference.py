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
    child = lib.ai_gateway.classify_batch(
        corpus=corpus, labels=labels,
        instruction=inputs.get("instruction", "Choose the primary failure class."))
    result = program.child_envelope(child, invalid_class="invalid_response", where="classify",
                                    adopt=("status", "signals", "errors", "evidence"))
    envelope["evidence"].append({"program": "ai-gateway.classify-batch", "artifact": child.meta()})
    return "report"


def step_report():
    envelope["phase"] = "report"
    program.report()
    return None


STATES = {"validate": step_validate, "classify": step_classify, "report": step_report}
program.run(STATES, "validate")
