"""browser-automation-studio.do-task v1 — S4 orchestrator: recall -> find -> try up to k workflows -> navigate fallback -> decide -> report.

Contract: do-task.json (memory block: scope bas-usage, reads in collect, writes in report).
Skill:    browser-automation-studio (usage) — the [S4] leaf "just do the browser task".

Phases: validate -> collect -> classify -> decide -> act -> report.
Submit with --async; exceeds the synchronous bound (budget.wall_ms 900000).
The smoke-flow and find-flows logic is inlined (run_candidate, the token scorer) because lib.<name>() accepts no
inputs today; this is a blocker filed as W1 against program-runtime (lib.<scenario>.<name>(input=...)), not a design.
When that lands, run_candidate collapses to lib.browser_automation_studio.smoke_flow(workflow_id=...).
model is required only when a session is given (the navigation fallback); no model slug is defaulted here.
Persisting an authored candidate needs workflows/create (no governed binding): the decide phase
records author_recommended and the report names the command; nothing is persisted here.
Memory writes happen once, in report: one task-record, one site-note per new failure class, one
workflow-verdict per candidate tried. A memory outage is degraded (memory_unavailable), not fatal.
"""

try:
    inputs
except NameError:
    inputs = {}
task = str(inputs.get("task", "") or "").strip()
scenario = str(inputs.get("scenario", "") or "").strip()
k = int(inputs.get("k", 3))
recurrence_threshold = int(inputs.get("recurrence_threshold", 2))
workflow_ids = [str(w) for w in (inputs.get("workflow_ids") or []) if w]
session = str(inputs.get("session", "") or "").strip()
max_steps = int(inputs.get("max_steps", 10))
model = str(inputs.get("model", "") or "").strip()  # required with session; never defaulted to a slug
memory_scope = str(inputs.get("memory_scope", "bas-usage") or "bas-usage").strip()
recurrence_similarity = float(inputs.get("recurrence_similarity", 0.8))
nav_max_tokens = int(inputs.get("nav_max_tokens", 20000))

envelope = {
    "program": "browser-automation-studio.do-task", "version": "1",
    "status": "failed", "phase": "validate",
    "inputs": {"task": task, "scenario": scenario or None, "k": k, "recurrence_threshold": recurrence_threshold,
               "workflow_ids": workflow_ids, "session": session or None, "max_steps": max_steps, "model": model,
               "memory_scope": memory_scope, "recurrence_similarity": recurrence_similarity,
               "nav_max_tokens": nav_max_tokens},
    "signals": {"prior_attempts": 0, "candidates": [], "attempts": [], "passed_workflow_id": None,
                "navigation": None, "author_recommended": False, "author_reason": None,
                "memory_written": False, "memory_skipped_reason": None,
                "memory_writes": {"task-record": 0, "site-note": 0, "workflow-verdict": 0}},
    "errors": [], "evidence": [],
}
handles = {}
phases_entered = []
STOP = {"the", "a", "an", "to", "of", "and", "on", "in", "for", "with", "page", "test", "check", "open", "go"}
OUTCOME = {"completed": "reached", "awaiting_human": "human_pause", "max_steps_reached": "budget",
           "loop_detected": "budget", "navigating": "in_progress", "idle": "in_progress",
           "failed": "failed", "aborted": "failed"}


def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["phase_failed"] = where
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


def classify_failure(text):
    t = (text or "").lower()
    if any(k in t for k in ("waiting for selector", "waiting for locator", "element to be visible",
                            "element not found", "no element", "selector", "locator")):
        return "selector_not_found"
    if any(k in t for k in ("timeout", "timed out", "exceeded")):
        return "timeout"
    if any(k in t for k in ("401", "403", "unauthorized", "forbidden", "sign in", "log in", "login")):
        return "auth_required"
    return "step_failed"


def tokens(text):
    return {w for w in "".join(ch if ch.isalnum() else " " for ch in (text or "").lower()).split() if w not in STOP and len(w) > 2}


def run_candidate(wid):
    """Inlined smoke-flow: one execute with wait, then executions.get; returns an attempt record."""
    attempt = {"workflow_id": wid, "execution_id": None, "execution_status": None, "outcome": None, "class": None, "detail": None}
    try:
        rows = browser_automation_studio.workflows.execute(workflow_id=wid, wait=True).head(1)
    except Exception as exc:
        status, klass = classify_transport(exc)
        attempt.update(outcome="error", **{"class": klass, "detail": str(exc)[:160]})
        attempt["transport_status"] = status
        return attempt
    if not rows:
        attempt.update(outcome="error", **{"class": "binding_error", "detail": "execute returned no row"})
        return attempt
    r = rows[0]
    attempt["execution_id"] = r.get("executionId")
    attempt["execution_status"] = r.get("status")
    err = r.get("error")
    if attempt["execution_id"]:
        envelope["evidence"].append(f"execution:{attempt['execution_id']}")
        try:
            ex = browser_automation_studio.executions.get(execution_id=attempt["execution_id"]).head(1)
            if ex:
                e = ex[0].get("execution", ex[0])
                attempt["execution_status"] = e.get("status") or attempt["execution_status"]
                err = e.get("error") or err
        except Exception as exc:
            envelope["errors"].append({"class": "binding_error", "detail": f"executions.get: {str(exc)[:120]}", "where": "act"})
    st = attempt["execution_status"] or ""
    if st == "EXECUTION_STATUS_COMPLETED":
        attempt["outcome"] = "passed"
    elif st in ("EXECUTION_STATUS_RUNNING", "EXECUTION_STATUS_PENDING"):
        attempt.update(outcome="still_running", **{"class": "timeout", "detail": f"{st} after wait"})
    else:
        attempt.update(outcome="failed", **{"class": classify_failure(err), "detail": (err or f"status {st}")[:160]})
    return attempt


# ---- phases ----------------------------------------------------------------
def step_validate():
    if not task:
        return fail("failed", "invalid_input", "task is required", "validate")
    if not (1 <= k <= 5):
        return fail("failed", "invalid_input", f"k={k} outside 1..5", "validate")
    if not (1 <= max_steps <= 25):
        return fail("failed", "invalid_input", f"max_steps={max_steps} outside 1..25", "validate")
    if session and not model:
        return fail("failed", "model_required", "model is required when session is given: vision-navigation has no default and callers must not hardcode a slug", "validate")
    return "collect"


def step_collect():  # COLLECT · memory recall (declared) + workflow inventory
    envelope["phase"] = "collect"
    phases_entered.append("collect")
    try:
        mem = vrooli_memory.recall.recall(query=task, scope=memory_scope, limit=10)
        handles["mem_hits"] = mem.head(10)
        # recall rows carry score as a string; a prior attempt counts only above the similarity floor
        envelope["signals"]["prior_attempts"] = sum(
            1 for h in handles["mem_hits"]
            if str(h.get("text", "")).startswith("task-record:")
            and float(h.get("score") or 0) >= recurrence_similarity)
    except Exception as exc:
        handles["mem_hits"] = []
        handles["memory_down"] = True
        envelope["errors"].append({"class": "memory_unavailable", "detail": str(exc)[:160], "where": "collect"})
    try:
        handles["wf"] = browser_automation_studio.workflows.list(limit=100)
        handles["wf"].count()
    except Exception as exc:
        status, klass = classify_transport(exc)
        return fail(status, klass, exc, "collect")
    return "classify"


def step_classify():  # CLASSIFY · deterministic fit over names and folders; explicit workflow_ids win
    envelope["phase"] = "classify"
    phases_entered.append("classify")
    tt = tokens(task)
    phrase = task.lower()
    scored = []
    for r in handles["wf"].head(100):
        text = f"{r.get('name', '')} {r.get('folderPath', '')}"
        ov = len(tt & tokens(text))
        explicit = r.get("id") in workflow_ids
        if explicit or ov >= 1 or phrase in text.lower():
            scored.append({"id": r.get("id"), "name": r.get("name"), "folder": r.get("folderPath"),
                           "overlap": 99 if explicit else ov, "fit": "explicit" if explicit else ("strong" if ov >= 3 else "weak")})
    scored.sort(key=lambda c: (-c["overlap"], str(c["name"])))
    envelope["signals"]["candidates"] = scored[:k]
    return "decide"


def step_decide():  # DECIDE · pure: which candidates to try, whether navigation is allowed
    envelope["phase"] = "decide"
    phases_entered.append("decide")
    handles["plan"] = [c["id"] for c in envelope["signals"]["candidates"] if c.get("id")]
    handles["navigate"] = bool(session)
    return "act"


def step_act():  # ACT · try candidates in order, stop at first pass; then the navigation fallback
    envelope["phase"] = "act"
    phases_entered.append("act")
    for wid in handles["plan"]:
        attempt = run_candidate(wid)
        envelope["signals"]["attempts"].append(attempt)
        if attempt.get("transport_status") in ("unavailable", "refused"):
            return fail(attempt["transport_status"], attempt["class"], attempt["detail"], "act")
        if attempt["outcome"] == "passed":
            envelope["signals"]["passed_workflow_id"] = wid
            envelope["status"] = "ok" if not envelope["errors"] else "partial"
            return "report"
    if handles["navigate"]:
        nav = {"navigation_id": None, "status": None, "outcome": None, "step_count": None, "total_tokens": None}
        try:
            rows = browser_automation_studio.vision_navigation.start(session=session, prompt=task, max_steps=max_steps, model=model).head(1)
            nav["navigation_id"] = rows[0].get("navigationId") if rows else None
            if nav["navigation_id"]:
                envelope["evidence"].append(f"navigation:{nav['navigation_id']}")
                srows = browser_automation_studio.vision_navigation.status(navigation_id=nav["navigation_id"]).head(1)
                if srows:
                    nav["status"] = srows[0].get("status")
                    nav["step_count"] = srows[0].get("stepCount")
                    tt = srows[0].get("totalTokens")
                    nav["total_tokens"] = int(tt) if tt not in (None, "") else None
                    nav["outcome"] = OUTCOME.get(str(nav["status"]), "unknown")
        except Exception as exc:
            status, klass = classify_transport(exc)
            envelope["signals"]["navigation"] = nav
            if status in ("unavailable", "refused"):
                return fail(status, klass, exc, "act")   # a dead driver is not a failed task
            envelope["errors"].append({"class": klass if klass != "binding_error" else "navigation_failed", "detail": str(exc)[:160], "where": "act"})
        envelope["signals"]["navigation"] = nav
        if nav.get("total_tokens") is not None and nav["total_tokens"] > nav_max_tokens:
            envelope["errors"].append({"class": "budget_exhausted",
                                       "detail": f"navigation spent {nav['total_tokens']} tokens over nav_max_tokens {nav_max_tokens}",
                                       "where": "act"})
            nav["outcome"] = "budget"
        clean = nav.get("outcome") == "reached"
        recurring = envelope["signals"]["prior_attempts"] >= recurrence_threshold
        if clean and recurring:
            envelope["signals"]["author_recommended"] = True
            envelope["signals"]["author_reason"] = (
                f"recurrence {envelope['signals']['prior_attempts']} >= {recurrence_threshold} and a clean navigation trace; "
                "persist a candidate by hand: workflows create --folder-path candidates --flow-file <draft> (no governed binding)")
        elif clean:
            envelope["signals"]["author_reason"] = f"clean trace but recurrence {envelope['signals']['prior_attempts']} < {recurrence_threshold}"
        if nav.get("outcome") == "reached":
            envelope["status"] = "partial" if envelope["errors"] else "ok"
            return "report"
        if nav.get("outcome") in ("human_pause", "in_progress"):
            envelope["status"] = "partial"
            envelope["errors"].append({"class": "navigation_pending", "detail": f"status {nav.get('status')}; re-read by hand", "where": "act"})
            return "report"
        return fail("failed", "budget_exhausted" if nav.get("outcome") == "budget" else "navigation_failed",
                    f"navigation status {nav.get('status')}", "act")
    if not handles["plan"]:
        return fail("failed", "no_candidates", "no workflow matched the task and no session was given for navigation", "act")
    last = envelope["signals"]["attempts"][-1]
    return fail("failed", last["class"] or "step_failed", last["detail"] or "all candidates failed", "act")


def step_report():  # REPORT · declared memory writes, then the one print
    envelope["phase"] = "report"
    phases_entered.append("report")
    # The skill never learns from a rejected input or a dead driver: write only when the
    # program actually attempted the task and the outcome describes the site, not the fleet.
    if handles.get("memory_down"):
        envelope["signals"]["memory_skipped_reason"] = "memory_unavailable"
    elif "act" not in phases_entered:
        envelope["signals"]["memory_skipped_reason"] = f"no attempt was made (stopped in {envelope.get('phase_failed') or phases_entered[-1] if phases_entered else 'validate'})"
    elif envelope["status"] not in ("ok", "partial", "failed"):
        envelope["signals"]["memory_skipped_reason"] = f"status {envelope['status']} describes the driver, not the task"
    else:
        outcome = envelope["status"]
        notes = [("task-record", f"task-record: {task} | outcome {outcome} | passed {envelope['signals']['passed_workflow_id']} | "
                                 f"attempts {len(envelope['signals']['attempts'])} | nav {(envelope['signals']['navigation'] or {}).get('outcome')}",
                  {"trigger": task, "approach": f"do-task k={k}", "evidence": ",".join(envelope["evidence"][:6]) or "none", "outcome": outcome})]
        seen = set()
        for a in envelope["signals"]["attempts"]:
            notes.append(("workflow-verdict", f"workflow-verdict: {a['workflow_id']} {a['outcome']} {a.get('class') or ''} for task {task}",
                          {"trigger": task, "approach": f"execute {a['workflow_id']}", "evidence": a.get("execution_id") or "none", "outcome": a["outcome"]}))
            if a.get("class") in ("selector_not_found", "timeout", "auth_required") and a["class"] not in seen and scenario:
                seen.add(a["class"])
                notes.append(("site-note", f"site-note: {scenario} {a['class']} — {a.get('detail')}",
                              {"trigger": task, "approach": f"execute {a['workflow_id']}", "evidence": a.get("execution_id") or "none", "outcome": a["class"]}))
        for kind, body, wr in notes:
            try:
                vrooli_memory.journal.note(body=body[:400], scope=memory_scope, kind=kind, **wr).count()
                envelope["signals"]["memory_writes"][kind] += 1
                envelope["signals"]["memory_written"] = True
            except Exception as exc:
                envelope["errors"].append({"class": "memory_unavailable", "detail": f"note {kind}: {str(exc)[:120]}", "where": "report"})
                if envelope["status"] == "ok":
                    envelope["status"] = "partial"  # the task outcome stands; the record of it is incomplete
                break
    print(envelope)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "decide": step_decide, "act": step_act, "report": step_report}
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
