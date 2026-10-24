"""browser-automation-studio.find-flows v3 — rank existing typed workflows for a task.

Contract: find-flows.json.
Skill:    browser-automation-studio (usage) — the [S3] leaf "does a typed workflow exist?".

Phases: validate -> collect -> classify -> report. Read-only.
Sources: workflows/list (persisted BAS workflows), workflow-health/workflows/search (scenario-owned
bas/ assets; skipped without a scenario), search-hub/query/query with rows="ranked" over the
workflow.flow and workflow.fragment types. No recall: this is an S3 step and consumes no advice;
prior attempts are recalled once by the usage skill in the bas-usage scope.
Fit is a deterministic token-overlap label; ai.classify runs at most once, either to break a tie at the k
boundary or, when no lexical candidate exists, to judge persisted workflow names against the task.
Persisted candidates carry their version so a caller can form the learning identity <id>@<version>.

Reading each owner's row shape is a learn.act section ("row-normalization"): three owners spell
identity, name and folder differently, and a renamed field used to yield a nameless candidate that
sorted last and vanished below k, which reads to the caller as "no workflow exists". The section
carries its own stable learning key, so its evidence pools across every search rather than
restarting per task. It owns an attempt record for that reason; it still consumes no advice.
"""
import hashlib

inputs = program.inputs()
task = str(inputs.get("task", "") or "").strip()
scenario = str(inputs.get("scenario", "") or "").strip()
k = int(inputs.get("k", 5))
learn.task(scope="bas-usage", operation="browser-automation-studio.find-flows",
           key={"scenario": scenario, "task": hashlib.sha256(" ".join(task.lower().split()).encode("utf-8")).hexdigest()})

envelope = {
    "program": "browser-automation-studio.find-flows", "version": "3",
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


# The tagged rows the current normalization section is accounting for; the verifier reads them so
# it judges the output against what each owner actually returned.
NORMALIZE_SOURCE = []
NORMALIZE_SCHEMA = {
    "type": "object", "required": ["normalized"], "additionalProperties": False,
    "properties": {"normalized": {"type": "array", "maxItems": 200, "items": {
        "type": "object", "additionalProperties": False,
        "required": ["index", "id", "name", "folder", "snippet", "version"],
        "properties": {"index": {"type": "integer", "minimum": 0},
                       "id": {"type": "string", "maxLength": 512},
                       "name": {"type": "string", "maxLength": 512},
                       "folder": {"type": "string", "maxLength": 512},
                       "snippet": {"type": "string", "maxLength": 2048},
                       "version": {"type": "integer", "minimum": 0}}}}},
}
def verify_normalization(output):
    """Judge the mapping against the rows the owners returned, not against the mapping.

    A renamed field shows up as an extracted value that is empty while the row still carries
    text, or as a value that appears nowhere in the row. Both fail here rather than becoming a
    nameless candidate that silently ranks last and disappears below k.
    """
    entries = output["normalized"]
    if sorted(entry["index"] for entry in entries) != list(range(len(NORMALIZE_SOURCE))):
        return ("failed", ["find:accounting-incomplete"])
    for entry in entries:
        item = NORMALIZE_SOURCE[entry["index"]]
        row = item["row"]
        present = {str(value) for value in row.values()
                   if isinstance(value, (str, int, float)) and not isinstance(value, bool)}
        if not entry["id"] or entry["id"] not in present:
            return ("failed", ["find:id-not-in-source"])
        text_fields = {value for key, value in row.items()
                       if key != "id" and isinstance(value, str) and value.strip()}
        if text_fields and not entry["name"]:
            return ("failed", ["find:name-empty-while-row-has-text"])
        for field in ("name", "folder", "snippet"):
            if entry[field] and entry[field] not in present:
                return ("failed", ["find:" + field + "-not-in-source"])
        expected = int(row.get("version") or 0) if item["source"] == "workflows" else 0
        if entry["version"] != expected:
            return ("failed", ["find:version-mismatch"])
    return ("verified_success", ["find:normalized-" + str(len(entries)),
                                 "find:rows-" + str(len(NORMALIZE_SOURCE))])


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
    del NORMALIZE_SOURCE[:]
    for key, source, limit in (("wf", "workflows", 100), ("wh", "workflow-health", k), ("sh", "search-hub", k)):
        handle = handles.get(key)
        if handle is None:
            continue
        for row in handle.head(limit):
            if isinstance(row, dict):
                NORMALIZE_SOURCE.append({"index": len(NORMALIZE_SOURCE), "source": source, "row": row})
    normalized = []
    if NORMALIZE_SOURCE:
        try:
            acted = learn.act(
                "row-normalization",
                "Extract the identity, name, folder, snippet and version each owner's row carries.",
                {"rows": list(NORMALIZE_SOURCE)}, NORMALIZE_SCHEMA,
                # Declared as the compatibility anchor, not for ambient access: these are the three
                # response shapes the mapping reads, so a contract change invalidates the fragment.
                ["browser-automation-studio/workflows/list", "workflow-health/workflows/search",
                 "search-hub/query/query"],
                attempts=2,
                # The mapping reads response shapes, not this search: one key for every task.
                key="bas/flow-row-shapes/v1",
                verify=verify_normalization, verifier_revision="row-fields/v1")
            normalized = acted["output"]["normalized"]
            envelope["signals"]["normalization_source"] = acted["fragment_source"]
        except Exception as exc:
            # Refusing beats ranking rows whose fields could not be read: an unnamed candidate
            # sorts last and vanishes below k, which reads to the caller as "no workflow exists".
            envelope["signals"]["normalization_source"] = "unavailable"
            envelope["errors"].append({"class": "normalization_unavailable", "detail": str(exc)[:160], "where": "classify"})
    scored = []
    for entry in sorted(normalized, key=lambda e: e["index"]):
        item = NORMALIZE_SOURCE[entry["index"]]
        source, row = item["source"], item["row"]
        text = f"{entry['name']} {entry['snippet']} {entry['folder']}"
        ov = len(tt & tokens(text))
        label = fit_label(ov, phrase in text.lower())
        if label == "none":
            continue
        candidate = {"source": source, "id": entry["id"], "name": entry["name"], "folder": entry["folder"],
                     "overlap": ov, "fit": label, "runnable_by_id": source == "workflows"}
        if source == "workflows":
            candidate["version"] = entry["version"]
        elif source == "workflow-health":
            candidate["mutating"], candidate["leaf_type"] = row.get("mutating"), row.get("leafType")
        else:
            candidate["provider"] = row.get("providerId")
        scored.append(candidate)
    scored.sort(key=lambda c: (-c["overlap"], c["source"], str(c["name"])))
    persisted = [e for e in normalized if NORMALIZE_SOURCE[e["index"]]["source"] == "workflows"][:20]
    if not scored and persisted:
        # Lexical miss (stop words ate the task, or names are terse): spend the one inference
        # call judging persisted workflow names, which are the only runnable candidates.
        try:
            verdicts = ai.classify(texts=[f"{e['name']} ({e['folder']})" for e in persisted],
                                   labels=["fits", "does_not_fit"],
                                   instruction=f"Does this saved browser workflow accomplish the task: {task}?")
            envelope["signals"]["tie_broken_by_ai"] = True
            for e, v in zip(persisted, verdicts.head(len(persisted))):
                if isinstance(v, dict) and v.get("label") == "fits":
                    scored.append({"source": "workflows", "id": e["id"], "version": e["version"], "name": e["name"],
                                   "folder": e["folder"], "overlap": 0, "fit": "judged", "runnable_by_id": True, "ai": "fits"})
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
    # The verified part of a search is the normalization: whether each owner's rows could be read.
    # Finding no matching workflow is a legitimate answer, never a failure.
    source = envelope["signals"].get("normalization_source")
    if source and source != "unavailable":
        learn.outcome("verified_success", ["find:normalization-" + source,
                                           "find:candidates-" + str(len(envelope["signals"]["candidates"]))])
    elif source == "unavailable" or envelope["status"] in ("unavailable", "refused"):
        learn.outcome("unavailable", [])
    else:
        learn.outcome("unknown", [])
    # A caller that later finds this candidate set was wrong grades the search itself with this
    # reference; no domain read is repeated.
    envelope["signals"]["learning"] = {"feedback_ref": learn.result("search")}
    print(envelope)
    return None


STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify, "report": step_report}
state = "validate"
program.run(STATES, state)
