"""General browser task entry point: bounded authorized reuse or navigation.

Task identity includes site, normalized goal, profile, environment, and postconditions.
Navigation verifies/extracts on the final page without replay. A recorded flow stays an
unqualified candidate unless the caller authorizes qualification with `qualify=true`, which
persists it and then independently verifies the persisted revision against the task's own
postconditions. Only that second execution earns a preference, so the expensive AI navigation
path converges into a cheap deterministic workflow instead of being rediscovered every run.
"""
import hashlib
import json

inputs = program.inputs()
task = str(inputs.get("task", "") or "").strip()
site = str(inputs.get("site", "") or inputs.get("url", "") or "").split("//")[-1].split("/")[0]
site_key = site or str(inputs.get("scenario", "") or "unknown")
task_digest = hashlib.sha256(" ".join(task.lower().split()).encode("utf-8")).hexdigest()[:16]
identity = learn.task(scope="bas-usage", operation="browser-automation-studio.do-task",
                      key={"site": site_key, "task": task_digest, "profile": inputs.get("profile_id", ""), "environment": inputs.get("environment_revision", ""), "contract": hashlib.sha256(json.dumps({"postconditions": inputs.get("postconditions", []), "extraction": inputs.get("extraction", [])}, sort_keys=True).encode()).hexdigest()[:16]})

envelope = {"program": "browser-automation-studio.do-task", "version": "5", "status": "failed",
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
    earlier = inputs.get("advice_feedback_ref") or str(inputs.get("advice_attempt_id", "") or "").strip()
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
    conditions = inputs.get("postconditions", [])
    if not isinstance(conditions, list) or len(conditions) > 16 or any(not isinstance(a, dict) or not a.get("selector") or not a.get("mode") for a in conditions):
        return fail("failed", "invalid_input", "postconditions must be bounded typed assertions", "validate")
    extraction = inputs.get("extraction", [])
    if not isinstance(extraction, list) or len(extraction) > 16 or any(not isinstance(e, dict) or not e.get("name") or not e.get("selector") or type(e.get("limit", 10)) is not int or not 0 <= e.get("limit", 10) <= 100 for e in extraction):
        return fail("failed", "invalid_input", "extraction must contain bounded named selector specifications", "validate")
    authorized = inputs.get("authorized_workflows", [])
    if not isinstance(authorized, list) or len(authorized) > 100 or any(not isinstance(a, str) for a in authorized):
        return fail("failed", "invalid_input", "authorized_workflows must be a bounded list of revision identities", "validate")
    if inputs.get("effect_policy", "explicit") not in ("explicit", "read_only"):
        return fail("failed", "invalid_input", "effect_policy must be explicit or read_only", "validate")
    if inputs.get("auto_select") and (not inputs.get("authorized_workflows") or not inputs.get("postconditions")):
        return fail("failed", "invalid_input", "auto_select requires authorized_workflows and task postconditions", "validate")
    if "qualify" in inputs and type(inputs["qualify"]) is not bool:
        return fail("failed", "invalid_input", "qualify must be a boolean", "validate")
    if inputs.get("qualify") and not inputs.get("postconditions"):
        # Qualification re-executes the task. Without an assertion it would buy nothing.
        return fail("failed", "invalid_input", "qualify requires postconditions; an unasserted replay cannot verify anything", "validate")
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
    envelope["signals"]["output"] = (child.get("signals") or {}).get("output", {})
    envelope["signals"]["learning"]["children"] = [((child.get("signals") or {}).get("learning") or {})]
    envelope["evidence"] = child.get("evidence", [])[:10]
    envelope["errors"] = child.get("errors", [])[:5]
    envelope["status"] = child.get("status", "failed")


def workflow_eligible(workflow_id, version):
    status = learn.result_status(artifact={"kind": "workflow", "owner": "browser-automation-studio", "id": str(workflow_id), "revision": str(version)})
    return status.get("available") is True and status.get("eligible") is True


def route_workflow():
    workflow_id, version = inputs["workflow_id"], inputs["version"]
    if not workflow_eligible(workflow_id, version):
        return fail("refused", "learning_ineligible", "Workflow revision is contradicted or its eligibility cannot be checked", "act")
    with learn.step("verify") as step:
        child = child_result(lib.browser_automation_studio.smoke_flow(
            workflow_id=workflow_id, version=version, parameters=inputs.get("parameters", {}), postconditions=inputs.get("postconditions", []),
            effect_policy=inputs.get("effect_policy", "explicit"), extraction=inputs.get("extraction", [])))
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


def qualify_candidate(candidate):
    """Turn a reached navigation into a durable, independently verified workflow revision.

    Navigation already executed the task's effects, and its own trace can never verify the
    flow rebuilt from it. Qualification therefore costs one authored ad hoc run plus one
    execution of the persisted revision, so the caller must authorize repeating the task
    (`qualify=true`) and must declare postconditions, or the run proves nothing.

    The persisted revision is graded by the same child and the same postconditions the reuse
    path uses in route_workflow, so a preference note earned here is comparable to one earned
    by reuse. Returns the qualified option id, or None with the reason recorded on the envelope.
    """
    with learn.step("qualify-author") as step:
        authored = child_result(lib.browser_automation_studio.author_flow(
            flow=candidate, project_id=inputs.get("project_id", ""),
            name=inputs.get("name", "") or ("task-" + task_digest),
            folder=inputs.get("folder", "candidates")))
        step.outcome(child_outcome(authored), authored.get("evidence", []))
    signals = authored.get("signals") or {}
    workflow_id, version = signals.get("workflow_id"), signals.get("version")
    if not verified_child(authored) or not workflow_id or not version:
        envelope["errors"].append({"class": "qualification_failed", "detail": "Candidate did not persist as a verified revision", "where": "act"})
        return None
    envelope["signals"]["qualified_workflow"] = {"workflow_id": workflow_id, "version": version}
    with learn.step("qualify-verify") as step:
        verified = child_result(lib.browser_automation_studio.smoke_flow(
            workflow_id=workflow_id, version=version, parameters=inputs.get("parameters", {}),
            postconditions=inputs.get("postconditions", []), effect_policy=inputs.get("effect_policy", "explicit"),
            extraction=inputs.get("extraction", [])))
        step.outcome(child_outcome(verified), verified.get("evidence", []))
    option = option_id(workflow_id, version)
    if not verified_child(verified):
        # The persisted revision is not reusable. Record why so the next run does not re-select it.
        fingerprint = str(((verified.get("errors") or [{}])[0] or {}).get("class") or "qualification_failed")
        learn.note("avoid", {"option_id": option, "fingerprint": fingerprint}, evidence=verified.get("evidence", [])[:5])
        envelope["errors"].append({"class": "qualification_failed", "detail": "Persisted revision failed independent verification: " + fingerprint, "where": "act"})
        return None
    envelope["evidence"] = list(dict.fromkeys(envelope["evidence"] + verified.get("evidence", [])))[:10]
    learn.note("preference", {"option_id": option}, evidence=verified.get("evidence", [])[:5])
    learn.result("qualified-workflow", artifact={"kind": "workflow", "owner": "browser-automation-studio",
                                                 "id": str(workflow_id), "revision": str(version)})
    envelope["signals"]["qualification"] = "qualified"
    return option


def qualification_refusal(candidate):
    """Explain why an available candidate was not qualified, without failing the task."""
    if not inputs.get("qualify"):
        return "verification_required", "Navigation completed; pass qualify=true with postconditions to persist and independently verify this candidate"
    if not inputs.get("postconditions"):
        return "qualification_unavailable", "qualify requires postconditions; an unasserted replay cannot verify anything"
    return "", ""


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
    kwargs["postconditions"] = inputs.get("postconditions", [])
    kwargs["extraction"] = inputs.get("extraction", [])
    kwargs["effect_policy"] = inputs.get("effect_policy", "explicit")
    recalled = learn.recall(kinds=["target-note"], limit=5)
    kwargs["knowledge"] = json.dumps(recalled.get("notes", {}), sort_keys=True)[:4000]
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
    candidate = signals.get("candidate_flow")
    has_candidate = isinstance(candidate, dict) and bool(candidate.get("nodes"))
    can_qualify = bool(inputs.get("qualify")) and bool(inputs.get("postconditions"))
    if signals.get("outcome") == "verified_success":
        envelope["signals"]["output"] = signals.get("output", {})
        envelope["signals"]["candidate_flow"] = candidate
        envelope["signals"]["qualification"] = "candidate"
        # The task already succeeded on the live page. Qualification only decides whether a
        # durable revision is earned; a failure here must never downgrade a verified task.
        if has_candidate and can_qualify:
            qualify_candidate(candidate)
        elif has_candidate:
            klass, detail = qualification_refusal(candidate)
            if klass:
                envelope["errors"].append({"class": klass, "detail": detail, "where": "act"})
        return finish("verified_success", envelope["evidence"], reused=False)
    if signals.get("outcome") != "reached":
        # in_progress, human_pause or budget: the caller resumes with navigation_id.
        return finish(navigate_status, envelope["evidence"], reused=False)
    if not has_candidate:
        envelope["status"] = "partial"
        envelope["errors"].append({"class": "no_candidates", "detail": "navigation reached its goal but recorded no replayable steps", "where": "act"})
        return finish("unknown", envelope["evidence"], reused=False)
    envelope["signals"]["candidate_flow"] = candidate
    envelope["signals"]["qualification"] = "candidate"
    # Navigation reached its goal but proved nothing: its own trace can never verify the flow
    # rebuilt from it. Only an authorized re-execution of the persisted revision verifies.
    if can_qualify and qualify_candidate(candidate):
        envelope["status"] = "ok"
        envelope["signals"]["replay_eligible"] = True
        return finish("verified_success", envelope["evidence"], reused=False)
    envelope["signals"]["replay_eligible"] = False
    envelope["status"] = "partial"
    klass, detail = qualification_refusal(candidate)
    if klass:
        envelope["errors"].append({"class": klass, "detail": detail, "where": "act"})
    return finish("unknown", envelope["evidence"], reused=False)


def route_search():
    with learn.step("find") as find_step:
        result = child_result(lib.browser_automation_studio.find_flows(task=task, scenario=inputs.get("scenario", ""), k=5))
        find_status = (result.get("signals") or {}).get("outcome", "unknown")
        find_step.outcome(find_status if find_status in ("verified_success", "failed", "unavailable", "unknown") else "unknown", result.get("evidence", []))
    candidates = ((result.get("signals") or {}).get("candidates") or [])[:5]
    envelope["signals"]["candidates"] = candidates
    runnable = [c for c in candidates if c.get("runnable_by_id") and c.get("id") and c.get("version")]
    if inputs.get("auto_select"):
        authorized = inputs.get("authorized_workflows", [])
        runnable = [c for c in runnable if option_id(c["id"], c["version"]) in authorized]
    runnable = [c for c in runnable if workflow_eligible(c["id"], c["version"])]
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
        if inputs.get("auto_select"):
            inputs["workflow_id"], inputs["version"] = selected_id, int(selected_version)
            return route_workflow()
    if inputs.get("session") and inputs.get("model"):
        return route_navigation()
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
        if inputs.get("navigation_id") or (inputs.get("session") and not inputs.get("auto_select")):
            return route_navigation()
        return route_search()
    except Exception as exc:
        status, klass = program.classify(exc)
        envelope["errors"].append({"class": klass, "detail": str(exc)[:160], "where": "act"})
        envelope["status"] = status
        return finish("failed" if status == "failed" else "unavailable")


def step_report():
    envelope["phase"] = "report"
    selected = inputs.get("workflow_id")
    artifact = {"kind": "workflow", "owner": "browser-automation-studio", "id": selected, "revision": str(inputs["version"])} if selected else None
    if not artifact and envelope["signals"].get("navigation_id"):
        artifact = {"kind": "navigation", "owner": "browser-automation-studio", "id": envelope["signals"]["navigation_id"], "revision": "1"}
    envelope["signals"]["learning"]["feedback_ref"] = learn.result("task", artifact=artifact)
    print(envelope)
    return None


STATES = {"validate": step_validate, "act": step_act, "report": step_report}
state = "validate"
program.run(STATES, state)
