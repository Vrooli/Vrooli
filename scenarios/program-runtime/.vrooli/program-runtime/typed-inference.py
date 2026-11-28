"""Classify a small corpus through AI Gateway and preserve its child envelope."""

inputs = program.inputs()
envelope = program.envelope("program-runtime.typed-inference", "3")
envelope["signals"] = {}
corpus = inputs.get("corpus", [
    "The provider timed out during a retry.",
    "The user supplied an invalid request field.",
])
labels = inputs.get("labels", ["infra", "user"])


def step_validate():
    if not isinstance(corpus, list) or not 1 <= len(corpus) <= 8:
        return program.fail("failed", "invalid_input", "corpus must contain 1 to 8 texts", "validate")
    envelope["inputs"] = {"documents": len(corpus), "labels": labels}
    return "classify"


def step_classify():
    envelope["phase"] = "classify"
    child = lib.ai_gateway.classify_batch(
        corpus=corpus, labels=labels,
        instruction=inputs.get("instruction", "Choose the primary failure class."))
    result = program.child_envelope(child, invalid_class="invalid_response", where="classify",
                                    expected_digest="0e07025e32853027f155dd4216da6f0a28a705b0c887f88fcdcf0fe35b76edd8",
                                    adopt=("status", "signals", "errors", "evidence"))
    envelope["evidence"].append({"program": "ai-gateway.classify-batch", "artifact": child.meta()})
    return "report"


def step_report():
    envelope["phase"] = "report"
    program.report()
    return None


STATES = {"validate": step_validate, "classify": step_classify, "report": step_report}
program.run(STATES, "validate")
