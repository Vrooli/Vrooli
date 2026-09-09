"""browser-automation-studio.do-task v3 — route a browser task through learned choices, navigation, and verified flows.

Contract: do-task.json.
Skill:    browser-automation-studio (usage) — the [S3] entry point.

Four routes, exactly one per run:
  workflow_id + version   execute the selected revision through smoke-flow; grade an earlier
                          recommendation when advice_attempt_id names it.
  flow                    validate, run once, and save a typed candidate through author-flow.
  session or navigation_id  bounded AI navigation through navigate-intent; a reached navigation is
                          turned into a candidate flow, authored, and verified in the same run when
                          wait_millis allows it, otherwise resume later with navigation_id.
  (none)                  rank existing flows through find-flows and recommend one from memory;
                          selection stays explicit (selection_required).

Learning identity is site plus a digest of the normalized task text, so preferences for
"open the workflows page" never pool with "delete a workflow" on the same site. Options and
preference notes share one identity, `<workflow_id>@<version>`.
"""
import hashlib

inputs = program.inputs()
task = str(inputs.get("task", "") or "").strip()
site = str(inputs.get("site", "") or inputs.get("url", "") or "").split("//")[-1].split("/")[0]
site_key = site or str(inputs.get("scenario", "") or "unknown")
task_digest = hashlib.sha256(" ".join(task.lower().split()).encode("utf-8")).hexdigest()[:16]
identity = learn.task(scope="bas-usage", operation="browser-automation-studio.do-task",
                      key={"site": site_key, "task": task_digest})

envelope = {"program": "browser-automation-studio.do-task", "version": "3", "status": "failed",
            "phase": "validate", "inputs": {}, "signals": {"result": None, "candidates": [],
            "reused_workflow": False, "outcome": "unknown", "learning": {"attempt_id": identity.get("attempt_id"),
            "task_id": identity.get("task_id"), "key": identity.get("key")}}, "errors": [], "evidence": []}


def fail(status, klass, detail, where):
    envelope["status"] = status
    envelope["errors"].append({"class": klass, "detail": str(detail)[:160], "where": where})
    return "report"


def child_result(handle):
    rows = handle.head(1)
    if not rows or not isinstance(rows[0], dict):
        raise ValueError("Child program returned no result envelope")
    return rows[0]


def option_id(workflow_id, version):
    return str(workflow_id) + "@" + str(version)


def grade_advice(status):
    """Close the loop for a search-then-execute split: the run that executed the recommended
    revision grades the earlier run that recommended it."""
    earlier = str(inputs.get("advice_attempt_id", "") or "").strip()
    if not earlier or status not in ("verified_success", "failed"):
        return
    disposition = "supported" if status == "verified_success" else "contradicted"
    evidence = envelope["evidence"][:20] or ["outcome:" + status]
    try:
        envelope["signals"]["learning"]["graded"] = learn.feedback(earlier, disposition, evidence)
    except Exception as exc:
        envelope["errors"].append({"class": "memory_unavailable", "detail": "feedback: " + str(exc)[:120], "where": "act"})


def finish(status, evidence=None, reused=None):
    envelope["signals"]["outcome"] = status
    grade_advice(status)
    measurements = {"reused_workflow": bool(reused)} if reused is not None else None
    learn.outcome(status, list(dict.fromkeys((evidence or [])[:20])), measurements=measurements)
    return "report"


def step_validate():
    if not task:
        return fail("failed", "invalid_input", "task must be a nonempty string", "validate")
    for field in ("workflow_id", "project_id", "session", "model", "scenario", "site", "url", "navigation_id", "advice_attempt_id"):
        if field in inputs and (not isinstance(inputs[field], str) or len(inputs[field]) > 512):
            return fail("failed", "invalid_input", "Invalid " + field, "validate")
    selected = sum(bool(inputs.get(k)) for k in ("workflow_id", "flow", "session", "navigation_id"))
    if selected > 1:
        return fail("failed", "invalid_input", "Select one workflow, candidate flow, navigation session, or navigation to resume", "validate")
    if inputs.get("workflow_id") and (type(inputs.get("version")) is not int or inputs["version"] < 1):
        return fail("failed", "invalid_input", "Execution requires the exact selected version", "validate")
    wait = inputs.get("wait_millis", 0)
    if type(wait) is not int or not 0 <= wait <= 300000:
        return fail("failed", "invalid_input", "wait_millis must be an integer in 0..300000", "validate")
    envelope["inputs"] = {"task": task[:160], "workflow_id": inputs.get("workflow_id"), "site": site[:160],
                          "navigation_id": inputs.get("navigation_id"), "advice_attempt_id": inputs.get("advice_attempt_id")}
    return "act"


def verified_child(child):
    signals = child.get("signals") or {}
    return child.get("status") == "ok" and signals.get("outcome") == "verified_success" and bool(child.get("evidence"))


def child_outcome(child):
    value = (child.get("signals") or {}).get("outcome", "unknown")
    if value == "verified_success" and not verified_child(child):
        return "unknown"
    return value if value in ("verified_success", "failed", "unavailable", "unknown") else "unknown"


def adopt_child(child):
    envelope["signals"]["result"] = child.get("signals")
    envelope["evidence"] = child.get("evidence", [])[:10]
    envelope["errors"] = child.get("errors", [])[:5]
    envelope["status"] = child.get("status", "failed")


def route_workflow():
    workflow_id, version = inputs["workflow_id"], inputs["version"]
    with learn.step("verify") as step:
        child = child_result(lib.browser_automation_studio.smoke_flow(
            workflow_id=workflow_id, version=version, parameters=inputs.get("parameters", {})))
        step.outcome(child_outcome(child), child.get("evidence", []))
    adopt_child(child)
    envelope["signals"]["reused_workflow"] = verified_child(child)
    if verified_child(child):
        learn.note("preference", {"option_id": option_id(workflow_id, version)}, evidence=envelope["evidence"][:5])
        return finish("verified_success", envelope["evidence"], reused=True)
    status = child_outcome(child)
    if status == "failed":
        fingerprint = str(((child.get("errors") or [{}])[0] or {}).get("class") or "step_failed")
        learn.note("avoid", {"option_id": option_id(workflow_id, version), "fingerprint": fingerprint}, evidence=envelope["evidence"][:5])
    return finish(status, envelope["evidence"], reused=True)


def route_candidate():
    with learn.step("author") as step:
        child = child_result(lib.browser_automation_studio.author_flow(
            flow=inputs["flow"], project_id=inputs.get("project_id", ""),
            name=inputs.get("name", "candidate"), workflow_id=inputs.get("repair_workflow_id", ""),
            expected_version=inputs.get("expected_version", 0)))
        step.outcome(child_outcome(child), child.get("evidence", []))
    adopt_child(child)
    signals = child.get("signals") or {}
    if verified_child(child) and signals.get("workflow_id") and signals.get("version"):
        learn.note("preference", {"option_id": option_id(signals["workflow_id"], signals["version"])}, evidence=envelope["evidence"][:5])
        return finish("verified_success", envelope["evidence"], reused=False)
    return finish(child_outcome(child), envelope["evidence"], reused=False)


def route_navigation():
    resume = str(inputs.get("navigation_id", "") or "")
    if not resume and not inputs.get("model"):
        return fail("failed", "model_required", "Select a navigator model before starting navigation", "act")
    kwargs = {"prompt": task, "max_steps": inputs.get("max_steps", 10), "wait_millis": inputs.get("wait_millis", 0)}
    if resume:
        kwargs["navigation_id"] = resume
        kwargs["session"] = inputs.get("session", "") or "resume"
        kwargs["model"] = inputs.get("model", "") or "resume"
    else:
        kwargs["session"] = inputs["session"]
        kwargs["model"] = inputs["model"]
    with learn.step("navigate") as navigate:
        child = child_result(lib.browser_automation_studio.navigate_intent(**kwargs))
        navigate_status = child_outcome(child)
        navigate.outcome(navigate_status, child.get("evidence", []))
    adopt_child(child)
    signals = child.get("signals") or {}
    envelope["signals"]["navigation_id"] = signals.get("navigation_id")
    if signals.get("outcome") == "failed":
        learn.note("avoid", {"fingerprint": "navigation:" + str(signals.get("status") or "failed")}, evidence=envelope["evidence"][:5])
        return finish("failed", envelope["evidence"], reused=False)
    if signals.get("outcome") != "reached":
        # in_progress, human_pause or budget: the caller resumes with navigation_id.
        return finish(navigate_status, envelope["evidence"], reused=False)
    candidate = signals.get("candidate_flow")
    if not isinstance(candidate, dict) or not candidate.get("nodes"):
        envelope["status"] = "partial"
        envelope["errors"].append({"class": "no_candidates", "detail": "navigation reached its goal but recorded no replayable steps", "where": "act"})
        return finish("unknown", envelope["evidence"], reused=False)
    with learn.step("author") as author:
        authored = child_result(lib.browser_automation_studio.author_flow(
            flow=candidate, project_id=inputs.get("project_id", ""), name=(inputs.get("name") or task[:60] or "candidate")))
        author.outcome(child_outcome(authored), authored.get("evidence", []))
    a = authored.get("signals") or {}
    if not verified_child(authored) or not a.get("workflow_id") or not a.get("version"):
        envelope["status"] = authored.get("status", "failed")
        envelope["errors"] += authored.get("errors", [])[:5]
        envelope["evidence"] = (envelope["evidence"] + authored.get("evidence", []))[:10]
        return finish(child_outcome(authored), envelope["evidence"], reused=False)
    with learn.step("verify") as verify:
        smoked = child_result(lib.browser_automation_studio.smoke_flow(
            workflow_id=a["workflow_id"], version=a["version"], parameters={}))
        verify.outcome(child_outcome(smoked), smoked.get("evidence", []))
    envelope["signals"]["result"] = smoked.get("signals")
    envelope["signals"]["workflow_id"] = a["workflow_id"]
    envelope["signals"]["version"] = a["version"]
    envelope["errors"] += smoked.get("errors", [])[:5]
    envelope["evidence"] = (envelope["evidence"] + authored.get("evidence", []) + smoked.get("evidence", []))[:10]
    envelope["status"] = smoked.get("status", "failed")
    envelope["signals"]["reused_workflow"] = False
    if verified_child(smoked):
        learn.note("preference", {"option_id": option_id(a["workflow_id"], a["version"])}, evidence=envelope["evidence"][:5])
        learn.note("target-note", {"fact": "navigation for this task reached its goal in " + str(signals.get("step_count") or "?") + " steps and replays as " + option_id(a["workflow_id"], a["version"]), "ttl_days": 30})
        return finish("verified_success", envelope["evidence"], reused=False)
    return finish(child_outcome(smoked), envelope["evidence"], reused=False)


def route_search():
    with learn.step("find") as find_step:
        result = child_result(lib.browser_automation_studio.find_flows(task=task, scenario=inputs.get("scenario", ""), k=5))
        find_status = (result.get("signals") or {}).get("outcome", "unknown")
        find_step.outcome(find_status if find_status in ("verified_success", "failed", "unavailable", "unknown") else "unknown", result.get("evidence", []))
    candidates = ((result.get("signals") or {}).get("candidates") or [])[:5]
    envelope["signals"]["candidates"] = candidates
    runnable = [c for c in candidates if c.get("runnable_by_id") and c.get("id") and c.get("version")]
    options = list(dict.fromkeys(option_id(c["id"], c["version"]) for c in runnable))
    if options:
        choice = learn.choose(options, options[0])
        if choice["selected_id"] is None:
            envelope["signals"]["learning"]["advice"] = choice["learning"]["advice"][:10]
            return fail("partial", "selection_required", "All matching workflows are excluded by learning evidence", "act")
        selected_id, _, selected_version = choice["selected_id"].partition("@")
        envelope["signals"]["recommended_workflow"] = {"workflow_id": selected_id, "version": int(selected_version) if selected_version.isdigit() else selected_version,
                                                       "option_id": choice["selected_id"], "source": choice["source"], "why": choice.get("why", [])[:3]}
        envelope["signals"]["learning"]["advice"] = choice["learning"]["advice"][:10]
    envelope["status"] = "partial"
    envelope["errors"].append({"class": "selection_required", "detail": "Select a matching workflow UUID and version (pass advice_attempt_id to grade this recommendation), or supply a candidate/session", "where": "act"})
    return finish("unknown")


def step_act():
    envelope["phase"] = "act"
    try:
        if inputs.get("workflow_id"):
            return route_workflow()
        if inputs.get("flow"):
            return route_candidate()
        if inputs.get("session") or inputs.get("navigation_id"):
            return route_navigation()
        return route_search()
    except Exception as exc:
        status, klass = program.classify(exc)
        envelope["errors"].append({"class": klass, "detail": str(exc)[:160], "where": "act"})
        envelope["status"] = status
        return finish("failed" if status == "failed" else "unavailable")


def step_report():
    envelope["phase"] = "report"
    print(envelope)
    return None


STATES = {"validate": step_validate, "act": step_act, "report": step_report}
state = "validate"
program.run(STATES, state)
