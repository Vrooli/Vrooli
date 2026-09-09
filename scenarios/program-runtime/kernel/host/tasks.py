"""Runtime-owned learning boundaries; every effect uses the governed Namespace.

The bridge checkpoints execution, while the shared Memory programs interpret
advice and write memory. Context is copied across gather and never stored in
user globals. A task record is authoritative after any transport interruption.
"""
import contextvars
import datetime
import hashlib
import json
import urllib.request
import urllib.error
import uuid
import secrets

from contracts import resolve_inputs, _json_copy

ACTIVE_TASK = contextvars.ContextVar("program_runtime_task", default=None)


def path_value(value, path):
    for key in path.split("."):
        if not isinstance(value, dict):
            return None
        value = value.get(key)
    return value


class Tasks:
    def __init__(self, owner, invocation_context, handle, timeout):
        self.owner = owner
        self.invocation_context = invocation_context
        self.handle = handle
        self.timeout = timeout

    def _bridge(self, action, **kwargs):
        if not self.owner._bridge_url:
            raise RuntimeError("task checkpoint bridge unavailable; domain not admitted")
        endpoint = self.owner._bridge_url.rsplit("/bindings/", 1)[0] + "/tasks/" + action
        context = self.invocation_context.get()
        payload = dict(kwargs, session_id=self.owner._session_id, program_id=context.get("program_id", ""))
        data = json.dumps(payload, allow_nan=False).encode()
        if len(data) > 256 * 1024:
            raise ValueError("task checkpoint exceeds 256 KiB")
        request = urllib.request.Request(endpoint, data=data, headers={"Content-Type": "application/json"}, method="POST")
        try:
            with urllib.request.urlopen(request, timeout=self.timeout) as response:
                return json.loads(response.read(2 * 1024 * 1024))
        except urllib.error.HTTPError as exc:
            raise RuntimeError("task checkpoint refused: " + exc.read(1024).decode(errors="replace")) from exc

    def _spec(self, operation, expected_digest=""):
        specs = [spec for group in self.owner.lib._contracts.values() for spec in group.values()]
        for spec in specs:
            name = spec.get("name", "")
            identity = name if "." in name else spec.get("scenario", "") + "." + name
            if identity == operation and (not expected_digest or spec.get("digest") == expected_digest):
                return spec
        return self._bridge("resolve", operation=operation, digest=expected_digest)

    def current(self):
        """Return a copy of the active identity and bounded preparation evidence."""
        current = _json_copy(ACTIVE_TASK.get() or {})
        current.pop("resume_token", None)
        return current

    def get(self, *, attempt_id, resume_token=""):
        resume_token = resume_token or (ACTIVE_TASK.get() or {}).get("resume_token", "")
        return self.handle([self._bridge("get", attempt_id=attempt_id, resume_token=resume_token)], "tasks.get")

    def resume(self, *, attempt_id, resume_token):
        """Requeue only the frozen finish intent; never execute domain code."""
        return self.handle([self._bridge("resume", attempt_id=attempt_id, resume_token=resume_token)], "tasks.resume")

    def fragment_get(self, *, step_key):
        """Read the best durable learn.act fragment for a step key."""
        return self.handle([self._bridge("fragment_get", step_key=step_key)], "tasks.fragment_get")

    def fragment_list(self):
        return self.handle([self._bridge("fragment_list")], "tasks.fragment_list")

    def delivery_metrics(self):
        return self.handle([self._bridge("delivery_metrics")], "tasks.delivery_metrics")

    def fragment_promote(self, *, step_key, min_verified=5, operation="", reviewed_by="", evidence=None,
                         fixtures=None, publish=False, expected_digest=""):
        """Prepare and optionally publish a reviewed baseline; retain learn.act in source."""
        from promotion import promote_source
        record = self._bridge("fragment_get", step_key=step_key)
        fragment = record.get("fragment") or {}
        if not record.get("found") or fragment.get("verified", 0) < int(min_verified) or fragment.get("contradicted_since_edit", 0):
            raise ValueError("fragment lacks eligible verification evidence")
        spec = self._spec(operation, expected_digest)
        declaration = spec.get("declaration") or {}
        candidate = promote_source(spec.get("source", fragment.get("source", "")), fragment.get("step_name"),
            fragment.get("fragment"), {}, None, compatibility=fragment.get("compatibility"),
            reviewed_by=reviewed_by, evidence=evidence, fixtures=fixtures)
        if publish:
            if not candidate["promoted"]:
                raise ValueError("publication requires a reviewed unambiguous baseline")
            receipt = self._bridge("fragment_publish", operation=operation, digest=spec.get("digest", ""),
                                  step_name=fragment["step_name"], step_key=step_key, min_verified=int(min_verified), baseline=candidate["candidate"]["baseline"])
            candidate["publication"] = receipt
        return self.handle([candidate], "tasks.fragment_promote")

    def prepare(self, *, operation, inputs, task_id="", attempt_id="", resume_token="", expected_digest=""):
        if ACTIVE_TASK.get():
            raise ValueError("an active task already owns this attempt; nested operations inherit it")
        spec = self._spec(operation, expected_digest)
        metadata = spec.get("declaration", {}).get("learning_task")
        if not metadata:
            raise ValueError("operation has no learning_task registration")
        resolved = resolve_inputs(spec["declaration"], inputs)
        finish_spec = self._spec("vrooli-memory.finish-attempt")
        task_id = task_id or "task-" + str(uuid.uuid4())
        attempt_id = attempt_id or "attempt-" + str(uuid.uuid4())
        resume_token = resume_token or "prt_resume_v1_" + secrets.token_urlsafe(32)
        response = self._bridge("begin", operation=operation, digest=spec["digest"],
                                task_id=task_id, attempt_id=attempt_id, inputs=resolved,
                                finish_digest=finish_spec["digest"], resume_token=resume_token)
        record = response["record"]
        record["resume_token"] = resume_token
        if not response["created"]:
            return self.handle([record], "tasks.prepare")
        trigger = str(resolved.get("task") or resolved.get("intent") or operation)
        trigger = " ".join(trigger.split())[:512]
        prepare_inputs = {key: record[key] for key in (
            "scope", "task_id", "context_key", "started_at", "task_started_at", "attempt_number", "provenance")}
        prepare_inputs.update(operation=metadata["operation"], trigger=trigger,
                              approach=operation + "@" + spec["digest"], advice_limit=5)
        seed = {key: value for key, value in prepare_inputs.items() if key not in ("scope", "advice_limit")}
        seed.update(attempt_id=record["attempt_id"], recall_status="unavailable", advice=[])
        preparation = {"signals": {"attempt": seed, "recall_status": "unavailable", "advice_candidates": []}}
        # Install context before invoking the shared helper: any registered
        # nested read must remain inside this single top-level attempt.
        token = ACTIVE_TASK.set(record)
        try:
            try:
                prep_spec = self._spec("vrooli-memory.prepare-attempt")
                preparation = self.owner._execute_library_source(prep_spec, (), prepare_inputs).head(1)[0]
            except Exception as exc:
                preparation["errors"] = [{"class": "recall_unavailable", "detail": str(exc)[:240]}]
            signals = preparation.setdefault("signals", {})
            # Runtime identity is allocated before recall. Memory's standalone
            # helper can derive an ID, but it cannot replace this checkpoint ID.
            signals["attempt"] = dict(seed, **signals.get("attempt", {}))
            signals["attempt"]["attempt_id"] = record["attempt_id"]
            if signals.get("recall_status") == "matched":
                signals["attempt"].pop("recall_status", None)
            record = self._bridge("start", attempt_id=record["attempt_id"], prepare=preparation)
            record["resume_token"] = resume_token
        finally:
            ACTIVE_TASK.reset(token)
        ACTIVE_TASK.set(record)
        return self.handle([record], "tasks.prepare")

    def finish(self, *, attempt_id, result, advice=None, resume_token=""):
        resume_token = resume_token or (ACTIVE_TASK.get() or {}).get("resume_token", "")
        record = self._bridge("get", attempt_id=attempt_id, resume_token=resume_token)
        if record["state"] == "completed" and record["delivery"] != "blocked":
            return self.handle([record], "tasks.finish")
        spec = self._spec(record["operation"], record["digest"])
        metadata = spec["declaration"]["learning_task"]
        result = _json_copy(result)
        outcome_spec = metadata["outcome"]
        status = path_value(result, outcome_spec["status_path"])
        outcome = outcome_spec["mapping"].get(status, "unknown") if isinstance(status, str) else "unknown"
        evidence = []
        for path in outcome_spec["evidence_paths"]:
            values = path_value(result, path)
            for value in values if isinstance(values, list) else []:
                if isinstance(value, str) and value.strip() and len(value.encode()) <= 512 and value not in evidence:
                    evidence.append(value)
        evidence = evidence[:20]
        if outcome == "verified_success" and not evidence:
            outcome = "unknown"
        preparation = record.get("prepare", {}).get("signals", {})
        attempt = _json_copy(preparation.get("attempt", {}))
        attempt.update(attempt_id=record["attempt_id"], outcome=outcome, evidence_refs=evidence,
                       finished_at=record.get("finish_inputs", {}).get("attempt", {}).get("finished_at")
                       or datetime.datetime.now(datetime.timezone.utc).isoformat())
        if outcome == "failed":
            causes = sorted({(str(e.get("class", "")), str(e.get("where", "")))
                             for e in result.get("errors", []) if isinstance(e, dict)})
            fingerprint = json.dumps([record["operation"], record["context_key"], causes], sort_keys=True)
            attempt["failure_fingerprint"] = "failure-v1:" + hashlib.sha256(fingerprint.encode()).hexdigest()
        if advice is None:
            advice = path_value(result, "signals.learning.advice")
        if advice is None:
            advice = []
        if advice is not None:
            if not isinstance(advice, list) or len(advice) > 10:
                raise ValueError("advice must contain at most 10 explicit decisions")
            candidates = {c["entry_id"] for c in preparation.get("advice_candidates", [])}
            if any(not isinstance(a, dict) or a.get("entry_id") not in candidates for a in advice):
                raise ValueError("advice decisions must name recalled candidates")
            assessed = {a["entry_id"] for a in advice}
            # Keep every exposed candidate; silence is neither adoption nor rejection.
            advice = advice + [{"entry_id": c["entry_id"], "decision": "unassessed",
                               "verdict": "unknown", "evidence_refs": []}
                              for c in preparation.get("advice_candidates", [])
                              if c["entry_id"] not in assessed]
            attempt["advice"] = _json_copy(advice)
            if advice:
                attempt["recall_status"] = "matched"
        measurements = path_value(result, "signals.learning.measurements") or {}
        if not isinstance(measurements, dict) or set(measurements) - {
                "first_action_at", "tool_round_trips", "visual_reasoning_calls", "reused_workflow"}:
            raise ValueError("unknown learning measurement; supply only observed effort or reuse")
        # Omission means unknown. The runtime never substitutes elapsed execution
        # time for first action or nested binding counts for outer-agent effort.
        attempt.update(_json_copy(measurements))
        observations = path_value(result, "signals.learning.observations") or []
        if not isinstance(observations, list) or len(observations) > 10:
            raise ValueError("at most 10 learning observations")
        pending = {"scope": record["scope"], "attempt": attempt, "observations": _json_copy(observations)}
        completed = self._bridge("complete", attempt_id=attempt_id, result=result,
                                 finish_inputs=pending, outcome=outcome, resume_token=resume_token)
        completed["resume_token"] = resume_token
        active = ACTIVE_TASK.get()
        if active and active.get("attempt_id") == attempt_id:
            ACTIVE_TASK.set(None)
        return self.handle([completed], "tasks.finish")

    def run(self, *, operation, inputs, task_id="", attempt_id="", resume_token="", expected_digest=""):
        spec = self._spec(operation, expected_digest)
        if not spec.get("declaration", {}).get("learning_task"):
            raise ValueError("operation has no learning_task registration")
        if ACTIVE_TASK.get():
            if task_id or attempt_id or resume_token:
                raise ValueError("nested task cannot override inherited identity")
            return self.owner._execute_library_source(spec, (), inputs)
        token = ACTIVE_TASK.set(None)
        try:
            record = self.prepare(operation=operation, inputs=inputs, task_id=task_id,
                                  attempt_id=attempt_id, resume_token=resume_token, expected_digest=expected_digest).head(1)[0]
            if record["state"] != "running" or not ACTIVE_TASK.get():
                if record.get("result") is not None:
                    return self._result(record)
                raise RuntimeError("attempt already exists; inspect tasks.get before any new domain action")
            try:
                result = self.owner._execute_library_source(spec, (), inputs).head(1)[0]
            except Exception as exc:
                # An exception cannot establish whether an effect occurred.
                result = {"status": "failed", "signals": {"outcome": "unknown"},
                          "errors": [{"class": "domain_interrupted", "detail": str(exc)[:240]}], "evidence": []}
            completed = self.finish(attempt_id=record["attempt_id"], result=result).head(1)[0]
            return self._result(completed)
        finally:
            ACTIVE_TASK.reset(token)

    def _result(self, record):
        result = _json_copy(record["result"])
        # Domain envelope/status stay intact. Delivery is a separate namespace.
        result["learning"] = {key: record.get(key) for key in (
            "task_id", "attempt_id", "attempt_number", "outcome", "delivery", "last_error", "resume_token")}
        preparation = record.get("prepare", {}).get("signals", {})
        candidates = preparation.get("advice_candidates", [])
        visible = [{"entry_id": candidate["entry_id"],
                    "text": candidate["text"].encode("utf-8")[:240].decode("utf-8", errors="ignore"),
                    "text_truncated": bool(candidate.get("text_truncated")) or len(candidate["text"].encode("utf-8")) > 240}
                   for candidate in candidates[:3]]
        decisions = record.get("finish_inputs", {}).get("attempt", {}).get("advice", [])
        result["learning"].update(
            recall_status=preparation.get("recall_status", "unavailable"),
            advice_candidates=visible,
            advice_candidates_omitted=max(0, len(candidates) - len(visible)),
            advice_decisions=[{key: decision.get(key) for key in ("entry_id", "decision", "verdict")}
                              for decision in decisions[:3]],
            guidance="Retrieved suggestions do not imply adoption. Inspect this attempt for full advice; use explicit decisions and evidence when applying it.")
        return self.handle([result], "tasks.run")
