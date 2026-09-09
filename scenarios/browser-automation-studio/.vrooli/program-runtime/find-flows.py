"""browser-automation-studio.find-flows v2 — rank existing typed workflows for a task.

Contract: find-flows.json.
Skill:    browser-automation-studio (usage) — the [S3] leaf "does a typed workflow exist?".

Phases: validate -> collect -> classify -> report. Read-only.
Sources: workflows/list (persisted BAS workflows), workflow-health/workflows/search (scenario-owned
bas/ assets; skipped without a scenario), search-hub/query/query with rows="ranked" over the
workflow.flow and workflow.fragment types. No memory: this is an S3 step; prior attempts are
recalled once by the usage skill in the bas-usage scope.
Fit is a deterministic token-overlap label; ai.classify runs at most once, either to break a tie at the k
boundary or, when no lexical candidate exists, to judge persisted workflow names against the task.
Persisted candidates carry their version so a caller can form the learning identity <id>@<version>.
"""

inputs = program.inputs()
task = str(inputs.get("task", "") or "").strip()
scenario = str(inputs.get("scenario", "") or "").strip()
k = int(inputs.get("k", 5))

envelope = {
    "program": "browser-automation-studio.find-flows", "version": "2",
    "status": "failed", "phase": "validate",
    "inputs": {"task": task, "scenario": scenario or None, "k": k},
    "signals": {"candidates": [], "sources": {}, "tie_broken_by_ai": False},
    "errors": [], "evidence": [],
}
handles = {}
STOP = {"the", "a", "an", "to", "of", "and", "on", "in", "for", "with", "page", "test", "check", "open", "go"}


fail = program.fail


def tokens(text):
    return {w for w in "".join(ch if ch.isalnum() else " " for ch in (text or "").lower()).split() if w not in STOP and len(w) > 2}


def fit_label(overlap, phrase_hit):
    if phrase_hit or overlap >= 3:
        return "strong"
    if overlap >= 1:
        return "weak"
    return "none"


# ---- phases ----------------------------------------------------------------
def step_validate():
    if not task:
        return fail("failed", "invalid_input", "task is required", "validate")
    if not (1 <= k <= 20):
        return fail("failed", "invalid_input", f"k={k} outside 1..20", "validate")
    return "collect"


def step_collect():  # COLLECT · governed reads; each source degrades independently
    envelope["phase"] = "collect"
    src = envelope["signals"]["sources"]
    try:
        handles["wf"] = browser_automation_studio.workflows.list(limit=100)
        src["workflows_list"] = handles["wf"].count()
    except Exception as exc:
        status, klass = program.classify(exc)
        return fail(status, klass, exc, "collect")
    if scenario:
        try:
            handles["wh"] = workflow_health.workflows.search(query=task, scenario=scenario, limit=k)
            src["workflow_health"] = handles["wh"].count()
        except Exception as exc:
            src["workflow_health"] = None
            envelope["errors"].append({"class": "binding_error", "detail": f"workflow-health search: {str(exc)[:160]}", "where": "collect"})
    else:
        src["workflow_health"] = None  # optional source skipped: not an error
    try:
        # rows="ranked": the response carries four repeated fields; the ranked hits are the rows.
        handles["sh"] = search_hub.query.query(text=task, type=["workflow.flow", "workflow.fragment"], limit=k, rows="ranked")
        src["search_hub"] = handles["sh"].count()
    except Exception as exc:
        src["search_hub"] = None
        status, klass = program.classify(exc)
        envelope["errors"].append({"class": klass, "detail": f"search-hub: {str(exc)[:160]}", "where": "collect"})
    return "classify"


def step_classify():  # CLASSIFY · deterministic fit first; one ai.classify only for a tie at the k boundary
    envelope["phase"] = "classify"
    tt = tokens(task)
    phrase = task.lower()
    scored = []
    for r in handles["wf"].head(100):
        text = f"{r.get('name', '')} {r.get('folderPath', '')}"
        ov = len(tt & tokens(text))
        label = fit_label(ov, phrase in text.lower())
        if label != "none":
            scored.append({"source": "workflows", "id": r.get("id"), "version": r.get("version"), "name": r.get("name"), "folder": r.get("folderPath"),
                           "overlap": ov, "fit": label, "runnable_by_id": True})
    if handles.get("wh") is not None:
        for r in handles["wh"].head(k):
            text = f"{r.get('title', '')} {r.get('snippet', '')} {r.get('path', '')}"
            ov = len(tt & tokens(text))
            label = fit_label(ov, phrase in text.lower())
            if label != "none":
                scored.append({"source": "workflow-health", "id": r.get("id"), "name": r.get("title"), "folder": r.get("path"),
                               "overlap": ov, "fit": label, "runnable_by_id": False,
                               "mutating": r.get("mutating"), "leaf_type": r.get("leafType")})
    if handles.get("sh") is not None:
        for r in handles["sh"].head(k):
            text = f"{r.get('title', '')} {r.get('snippet', '')} {r.get('path', '')}"
            ov = len(tt & tokens(text))
            label = fit_label(ov, phrase in text.lower())
            if label != "none":
                scored.append({"source": "search-hub", "id": r.get("id"), "name": r.get("title"), "folder": r.get("path"),
                               "overlap": ov, "fit": label, "runnable_by_id": False, "provider": r.get("providerId")})
    scored.sort(key=lambda c: (-c["overlap"], c["source"], str(c["name"])))
    if not scored and handles["wf"].count():
        # Lexical miss (stop words ate the task, or names are terse): spend the one inference
        # call judging persisted workflow names, which are the only runnable candidates.
        persisted = handles["wf"].head(min(handles["wf"].count(), 20))
        try:
            verdicts = ai.classify(texts=[f"{r.get('name', '')} ({r.get('folderPath', '')})" for r in persisted],
                                   labels=["fits", "does_not_fit"],
                                   instruction=f"Does this saved browser workflow accomplish the task: {task}?")
            envelope["signals"]["tie_broken_by_ai"] = True
            for r, v in zip(persisted, verdicts.head(len(persisted))):
                if isinstance(v, dict) and v.get("label") == "fits":
                    scored.append({"source": "workflows", "id": r.get("id"), "version": r.get("version"), "name": r.get("name"),
                                   "folder": r.get("folderPath"), "overlap": 0, "fit": "judged", "runnable_by_id": True, "ai": "fits"})
        except Exception as exc:
            envelope["errors"].append({"class": "inference_unavailable", "detail": str(exc)[:160], "where": "classify"})
    elif len(scored) > k and scored[k - 1]["overlap"] == scored[k]["overlap"]:
        tied = [c for c in scored if c["overlap"] == scored[k - 1]["overlap"]]
        try:
            verdicts = ai.classify(texts=[f"{c['name']} ({c['folder']})" for c in tied], labels=["fits", "does_not_fit"],
                                   instruction=f"Does this browser workflow accomplish the task: {task}?")
            # batch rows are {"label": <str>, "text": <source>}; the kernel has already validated the label
            for c, v in zip(tied, verdicts.head(len(tied))):
                c["ai"] = v.get("label") if isinstance(v, dict) else None
            envelope["signals"]["tie_broken_by_ai"] = True
            scored.sort(key=lambda c: (-c["overlap"], 0 if c.get("ai") == "fits" else 1, c["source"], str(c["name"])))
        except Exception as exc:
            envelope["errors"].append({"class": "inference_unavailable", "detail": str(exc)[:160], "where": "classify"})
    envelope["signals"]["candidates"] = scored[:k]
    envelope["evidence"] = [c["id"] for c in scored[:k] if c.get("id")]
    envelope["status"] = "ok" if not envelope["errors"] else "partial"
    return "report"


def step_report():  # REPORT
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
