"""Project shared Memory comparison rows into the established consumer envelope."""
import json
inputs = program.inputs()
envelope = program.envelope('browser-automation-studio.learning-read', '1', phase="validate", inputs={})
try:
    if not isinstance(inputs,dict) or set(inputs)-{"from","to","operation","context_key"}:
        raise ValueError("Only explicit comparison-window selectors are accepted")
    envelope["phase"]="collect"
    handle=lib.vrooli_memory.compare_outcomes(scope="bas-usage",cohort_limit=1,**inputs)
    groups={
     "failure-recurrence":["attempts","failed","unavailable","unknown","recurringFailureFingerprints"],
     "completion-effort":["tasks","completedTasks","unresolvedTasks","leftCensoredTasks","medianAttemptsToSuccess","medianSecondsToSuccess"],
     "advice-outcomes":["appliedAdvice","rejectedAdvice","supportedAdvice","contradictedAdvice","unassessedAdvice","recallUnavailable"],
     "first-action-latency":["firstActionSamples","medianSecondsToFirstAction"],
     "agent-round-trips":["toolRoundTripSamples","medianToolRoundTrips"],
     "visual-reasoning":["visualReasoningSamples","medianVisualReasoningCalls"],
     "workflow-reuse":["reuseSamples","workflowReuseRate"]}
    child=program.child_envelope(handle,invalid_class="invalid_response",where="collect:memory-comparison",
                                 adopt=("inputs", "errors", "evidence", "status"), row_groups=groups)
    envelope["signals"]["rows"]=child.get("signals",{}).get("rows",[])
    envelope["signals"]["reliability"]=child.get("signals",{}).get("reliability")
except Exception as exc:
    envelope["errors"].append({"class":"invalid_input" if isinstance(exc,ValueError) else "kernel_runtime",
                               "detail":str(exc)[:160],"where":envelope["phase"]})
envelope["phase"]="report"
print(envelope)
