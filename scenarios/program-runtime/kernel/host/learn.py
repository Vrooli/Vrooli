"""In-program learning verbs.

The object is deliberately execution-local. It allocates identity before the
first bridge call and starts the durable checkpoint only when a learn verb is
used. The task shim and the ten verbs therefore share one receipt boundary.
"""
from __future__ import annotations

import hashlib
import json
import secrets
import contextlib
import contextvars
import threading
import time
import fnmatch
import datetime
from typing import Any

from tasks import ACTIVE_TASK, Tasks
from fragments import StepFailed, wrap, digest, normalize, validate_baseline

ACTIVE_STEP = contextvars.ContextVar("program_runtime_step", default=None)


FLEET_NOTE_KINDS = {
    "preference", "parameter", "target-note", "avoid", "trace", "correction"
}
NOTE_REQUIRED = {
    "preference": {"option_id"}, "parameter": {"name", "value"},
    "target-note": {"fact"}, "avoid": set(), "trace": set(),
    "correction": {"corrects", "body"},
}
NOTE_FIELDS = {
    "preference": {"option_id"}, "parameter": {"name", "value"},
    "target-note": {"fact", "selector", "ttl_days"},
    "avoid": {"fingerprint", "option_id"},
    "trace": {"fragment", "bindings", "row_samples", "output", "attribution"},
    "correction": {"corrects", "body"},
}


class _Step(contextlib.AbstractContextManager):
    def __init__(self, learn, name, key=None, was=None):
        self.learn, self.name, self.requested_key, self.was = learn, _text(name, "step name"), key, was
        self.step_id = None
        self.record = None
        self._token = None

    def __enter__(self):
        self.learn.used = True
        self.learn._begin()
        parent_id = ACTIVE_STEP.get()
        parent = self.learn._steps.get(parent_id) if parent_id else None
        depth = (parent["depth"] + 1) if parent else 1
        if len(self.learn._steps) >= 32 or depth > 4:
            self.learn.bound_errors.append("learning_bound")
            self.record = {"step_id": "bound-" + secrets.token_urlsafe(6), "name": self.name, "key": self.learn.key or self.learn._default_key(), "outcome": "unknown", "evidence": [], "notes": [], "depth": depth, "bound": True, "error_class": "learning_bound"}
            self.learn._steps[self.record["step_id"]] = self.record
            return self
        parent_key = parent["key"] if parent else (self.learn.key or self.learn._default_key())
        step_key = _canonical_key(self.requested_key) if self.requested_key is not None else hashlib.sha256((self.learn.operation + "\x00" + self.name + "\x00" + parent_key).encode()).hexdigest()
        self.step_id = "step-" + secrets.token_urlsafe(10)
        self.record = {"step_id": self.step_id, "name": self.name, "key": step_key, "parent_step_id": parent_id, "parent_attempt_id": self.learn.attempt_id, "outcome": "unknown", "evidence": [], "notes": [], "decisions": [], "depth": depth, "opened_at": _now(), "closed_at": ""}
        with self.learn._step_lock:
            self.learn._steps[self.step_id] = self.record
        self._token = ACTIVE_STEP.set(self.step_id)
        if self.was is not None:
            self.note("correction", {"corrects": str(self.was), "body": "renamed"})
        return self

    def __exit__(self, exc_type, exc, tb):
        if self.record and not self.record.get("bound"):
            if exc is not None:
                if self.record.get("error_class") != "uncertain_effects" and self.record.get("outcome") != "unavailable":
                    self.record["outcome"] = "failed"
                    self.record["error_class"] = type(exc).__name__
                    self.record["failure_fingerprint"] = _failure_fingerprint(self.learn.operation, self.name, type(exc).__name__)
            self.record["closed_at"] = _now()
        if self._token is not None:
            ACTIVE_STEP.reset(self._token)
        return False

    def outcome(self, status, evidence=None):
        if not self.record or self.record.get("bound"):
            return {"status": "unknown", "evidence": []}
        _validate_outcome(status, evidence)
        self.record["outcome"] = status
        self.record["evidence"] = list(dict.fromkeys(evidence or []))[:20]
        if status == "failed":
            self.record["failure_fingerprint"] = _failure_fingerprint(self.learn.operation, self.name, "step_failed")
            for entry_id in self.record.pop("target_notes", []):
                if entry_id:
                    self.note("correction", {"corrects": entry_id, "body": "target-note contradicted by " + self.record["failure_fingerprint"]})
        if status in ("verified_success", "failed"):
            verdict = "supported" if status == "verified_success" else "contradicted"
            for decision in self.record.get("decisions", []):
                if decision.get("decision") == "applied" and decision.get("verdict") == "unknown":
                    decision["verdict"] = verdict
                    decision["derived"] = True
                    decision["evidence_refs"] = ["outcome:" + status]
        return {"status": status, "evidence": self.record["evidence"]}

    def note(self, kind, body, evidence=None):
        if not self.record or self.record.get("bound"):
            return {"kind": kind, "body": body, "evidence": []}
        item = self.learn.note(kind, body, evidence)
        if self.learn._notes and self.learn._notes[-1] is item:
            self.learn._notes.pop()
        self.record["notes"].append(item)
        return item

    def recall(self, *args, **kwargs):
        return self.learn.recall(*args, **kwargs)


class RecallHit(dict):
    def __init__(self, owner, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._owner = owner

    def zoom(self, limit=20):
        binding = getattr(getattr(getattr(self._owner, "vrooli_memory"), "recall"), "zoom")
        return binding(node_id=self.get("node_id", self.get("entry_id", "")), limit=min(max(int(limit), 1), 20))


class RecallResult(dict):
    def repeats(self, outcome=None, since=None):
        rows = self.get("attempts", [])
        if outcome:
            rows = [row for row in rows if row.get("outcome") == outcome]
        return len(rows)

    @property
    def last_outcome(self):
        return self.get("attempts", [{}])[-1].get("outcome") if self.get("attempts") else None

    @property
    def last_fingerprint(self):
        return self.get("attempts", [{}])[-1].get("failure_fingerprint", "") if self.get("attempts") else ""


class Learn:
    _fragment_cache = {}
    _fragment_stats = {}
    _scope_scenario_aliases = {"bas": "browser-automation-studio"}
    def __init__(self, owner, invocation_context, input_getter, program_name: str = "", free_text_inputs=(), note_kinds=None, bindings=(), baselines=None):
        self.owner = owner
        self.baselines = dict(baselines or {})
        self.binding_contracts = list(bindings) if not isinstance(bindings, set) else sorted(bindings)
        self._fragment_delivery = []
        self.invocation_context = invocation_context
        self.input_getter = input_getter
        context = invocation_context.get({})
        self.program_name = program_name or str(context.get("program_id", "program"))
        self.free_text_inputs = set(free_text_inputs or ())
        self.note_kinds = dict(note_kinds or {})
        self.writable_bindings = ({item.get("id", "") for item in bindings if isinstance(item, dict) and item.get("effect") == "write"}
                                  if not isinstance(bindings, set) else set(bindings))
        self.scope = self.program_name.split(".", 1)[0] + "-usage"
        self._scope_explicit = False
        self.operation = self.program_name
        self.key = None
        self._key_explicit = False
        self.task_id = "task-" + secrets.token_urlsafe(16)
        self.attempt_id = "attempt-" + secrets.token_urlsafe(16)
        self.resume_token = "prt_resume_v1_" + secrets.token_urlsafe(32)
        self.started = False
        self.used = False
        self._record = None
        self._notes: list[dict[str, Any]] = []
        self._outcome = "unknown"
        self._evidence: list[str] = []
        self._decisions: list[dict[str, Any]] = []
        self._start_token = None
        # not_requested until the program asks memory for anything; a run that
        # never recalls must not read as a memory outage in the measures.
        self.recall_status = "not_requested"
        self._recall_cache = {}
        self._last_recall = None
        self._steps = {}
        self._step_lock = threading.RLock()
        self.bound_errors = []
        self._examples = []
        self._fragment_candidates = []
        self._measurements: dict[str, Any] = {}
        self._feedback: list[dict[str, Any]] = []

    def reset(self, program_name: str = ""):
        self.__init__(self.owner, self.invocation_context, self.input_getter, program_name or self.program_name, self.free_text_inputs, self.note_kinds, self.binding_contracts, self.baselines)

    @property
    def identity(self):
        return {"task_id": self.task_id, "attempt_id": self.attempt_id,
                "resume_token": self.resume_token, "scope": self.scope,
                "operation": self.operation, "key": self.key or self._default_key()}

    def _default_key(self):
        inputs = self.input_getter() if callable(self.input_getter) else {}
        if not isinstance(inputs, dict):
            inputs = {}
        stable = {k: v for k, v in inputs.items() if k not in self.free_text_inputs and not str(k).endswith("_free_text")}
        body = json.dumps(stable, sort_keys=True, separators=(",", ":"), ensure_ascii=True, default=str)
        return hashlib.sha256(body.encode("utf-8")).hexdigest()

    def _begin(self):
        if self.started:
            return
        self.started = True
        self._source = self.invocation_context.get({}).get("source", "")
        if self.key is None:
            self.key = self._default_key()
        if not getattr(self.owner, "_bridge_url", ""):
            self._record = {"scope": self.scope, "task_id": self.task_id,
                            "attempt_id": self.attempt_id, "operation": self.operation,
                            "context_key": self.key, "provenance": self._provenance(),
                            "started_at": _now(), "task_started_at": _now(), "attempt_number": 1,
                            "delivery": "delivered", "state": "running"}
            return
        tasks = Tasks(self.owner, self.invocation_context, self.owner.handle if hasattr(self.owner, "handle") else None, getattr(self.owner, "_budgets", 100.0))
        spec = tasks._spec(self.operation)
        declaration = spec.get("declaration") or {}
        self.baselines = dict((declaration.get("learning") or {}).get("baselines", self.baselines))
        self.binding_contracts = declaration.get("bindings", self.binding_contracts)
        self.free_text_inputs.update(key for key, value in declaration.get("inputs", {}).items() if value.get("free_text"))
        self._source = spec.get("source", self._source)
        if not self._key_explicit:
            self.key = self._default_key()
        try:
            finish = tasks._spec("vrooli-memory.finish-attempt")
        except Exception:
            finish = {}
        begin_scope = {"scope": self.scope} if self._scope_explicit else {}
        response = tasks._bridge("begin", operation=self.operation, digest=spec.get("digest", ""), **begin_scope,
                                 task_id=self.task_id, attempt_id=self.attempt_id,
                                 inputs=self.input_getter() if callable(self.input_getter) else {},
                                 finish_digest=finish.get("digest", ""), resume_token=self.resume_token, context_key=self.key)
        self._record = response["record"]
        self._record["resume_token"] = self.resume_token
        self.key = self._record.get("context_key") or self.key
        if not response.get("created"):
            return
        prep = {"signals": {"attempt": {"attempt_id": self.attempt_id,
            "task_id": self.task_id, "operation": self.operation,
            "context_key": self.key, "scope": self.scope,
            "started_at": self._record.get("started_at", _now()),
            "task_started_at": self._record.get("task_started_at", _now()),
            "attempt_number": self._record.get("attempt_number", 1),
            "provenance": self._record.get("provenance", "agent"),
            "recall_status": "unavailable", "advice": []},
            "recall_status": "unavailable", "advice_candidates": []}}
        self._record = tasks._bridge("start", attempt_id=self.attempt_id, prepare=prep)
        self._record["resume_token"] = self.resume_token
        self._start_token = ACTIVE_TASK.set(self._record)

    def task(self, scope=None, operation=None, key=None):
        if self.used:
            raise ValueError("learn.task must be the first learning verb")
        if operation is not None:
            self.operation = _text(operation, "operation")
        if scope is not None:
            self.scope = _text(scope, "scope")
            self._scope_explicit = True
            owner_scenario = self.operation.split(".", 1)[0]
            scope_scenario = self.scope.removesuffix("-usage")
            scope_scenario = self._scope_scenario_aliases.get(scope_scenario, scope_scenario)
            if scope_scenario and scope_scenario != owner_scenario and scope_scenario + "/learning/record" not in self.writable_bindings:
                raise ValueError("foreign learning scope requires a declared write binding")
        if key is not None:
            self._key_explicit = True
            self.key = _canonical_key(key)
        self.used = True
        self._begin()
        return self.identity

    def step(self, name, key=None, was=None):
        return _Step(self, name, key, was)

    def outcome(self, status, evidence=None, measurements=None):
        _validate_outcome(status, evidence)
        evidence = evidence or []
        if measurements is not None:
            self._measurements = _validate_measurements(measurements)
        self.used = True
        self._begin()
        self._outcome, self._evidence = status, list(dict.fromkeys(evidence[:20]))
        if status in ("verified_success", "failed"):
            verdict = "supported" if status == "verified_success" else "contradicted"
            for decision in self._decisions:
                if decision.get("decision") == "applied" and decision.get("verdict") == "unknown":
                    decision["verdict"] = verdict
                    decision["derived"] = True
                    decision["evidence_refs"] = ["outcome:" + status]
        return {"status": status, "evidence": self._evidence}

    def feedback(self, attempt_id, disposition, evidence, correction=""):
        """Grade an earlier attempt's advice from a later run.

        A search-then-execute split (recommend in one run, execute in another)
        can only close its verdict here: the later run names the earlier
        attempt and memory appends an observation to it. Memory's measure
        pairs the observation with the attempt as supported/contradicted
        feedback. The write is immediate and bounded. Failed delivery is retained in this
        run's frozen finish intent without rerunning domain work.
        """
        attempt_id = _text(attempt_id, "attempt_id")
        if disposition not in ("supported", "contradicted"):
            raise ValueError("feedback disposition must be supported or contradicted")
        if not isinstance(evidence, list) or not evidence or any(not isinstance(v, str) or not v.strip() for v in evidence):
            raise ValueError("feedback requires non-empty evidence")
        if not isinstance(correction, str) or len(correction.encode("utf-8")) > 2048:
            raise ValueError("feedback correction must be bounded text")
        self.used = True
        self._begin()
        provenance = self._provenance()
        observation = {"attempt_id": attempt_id, "disposition": disposition,
                       "evidence_refs": list(dict.fromkeys(evidence[:20])),
                       "method_revision": "learn.feedback", "correction": ("correction/v1 " + json.dumps({"corrects": attempt_id, "body": correction}, sort_keys=True)) if correction else "",
                       "provenance": provenance, "observed_at": _now()}
        observation["observation_id"] = "observation-" + hashlib.sha256(json.dumps(
            observation, sort_keys=True, separators=(",", ":"), ensure_ascii=True).encode("utf-8")).hexdigest()
        scope = (self._record or {}).get("scope", self.scope)
        receipt = {"attempt_id": attempt_id, "disposition": disposition, "observation_id": observation["observation_id"],
                   "delivery": "failed", "scope": scope, "observation": observation}
        self._deliver_feedback(receipt)
        self._feedback.append(receipt)
        return receipt

    def _deliver_feedback(self, receipt):
        """One observe call. An attempt whose own capture is still in flight answers not-found;
        that is deferred, not failed, and finalize retries it once the domain work is done."""
        try:
            namespace = getattr(self.owner, "vrooli_memory")
            response = namespace.learning.observe(scope=receipt["scope"], observation=receipt["observation"])
            rows = response.head(1) if hasattr(response, "head") else [response]
            row = rows[0] if rows and isinstance(rows[0], dict) else {}
            receipt["entry_id"] = row.get("entryId", row.get("entry_id", ""))
            receipt["delivery"] = "delivered" if receipt["entry_id"] else "failed"
            receipt.pop("error", None)
        except Exception as exc:
            text = str(exc)
            receipt["error"] = text[:240]
            receipt["delivery"] = "deferred" if ("not_found" in text or "404" in text or "no rows" in text) else "failed"
        return receipt["delivery"] == "delivered"

    def _provenance(self):
        value = (self._record or {}).get("provenance") or self.invocation_context.get({}).get("provenance", "agent")
        return str(value).lower().removeprefix("provenance_")

    def recall(self, query=None, key=None, kinds=None, scope=None, since=None, limit=5, depth="auto"):
        self.used = True
        self._begin()
        limit = max(1, min(int(limit), 20))
        kinds = [str(kind) for kind in (kinds or [])]
        scope = scope or (self._record or {}).get("scope", self.scope)
        key = self.key if key is None else key
        if not query:
            # Notes are text-encoded as "<kind>/v1 {...}"; leading with the kind
            # markers makes a semantic recall land on notes instead of on the
            # attempt records that otherwise dominate a usage scope.
            query = (" ".join(kind + "/v1" for kind in kinds) + " " if kinds else "") + self.operation + " " + str(key)
        if kinds:
            # Kind filtering happens after the fetch, so fetch the full bound.
            limit = 20
        cache_key = (scope, str(key), query, tuple(kinds), str(since or ""), limit, depth)
        if cache_key in self._recall_cache:
            result = self._recall_cache[cache_key]
            current_step = ACTIVE_STEP.get()
            if current_step and current_step in self._steps:
                self._steps[current_step]["decisions"] = _bound_decisions([
                    {"entry_id": hit.get("entry_id", ""), "decision": "unassessed", "verdict": "unknown", "evidence_refs": []}
                    for hit in result.get("hits", [])
                ])
            return result
        result = RecallResult(hits=[], notes={}, attempts=[], recall_status="unavailable")
        try:
            namespace = getattr(self.owner, "vrooli_memory")
            binding = namespace.recall.recall
            rows = binding(scope=scope, query=query, limit=limit).head(limit)
            for row in rows:
                if not isinstance(row, dict):
                    continue
                context_key = row.get("context_key", row.get("contextKey", ""))
                if context_key and key != "*" and context_key != key:
                    continue
                if row.get("operation") and row["operation"] != self.operation:
                    continue
                if row.get("provenance") == "test" and self._provenance() != "test":
                    continue
                entry_id = row.get("entry_id", row.get("entryId", row.get("id", "")))
                text = row.get("text", row.get("summary", row.get("body", "")))
                hit = RecallHit(self.owner, {"entry_id": entry_id, "text": str(text)[:512], "summary": row.get("summary", ""), "depth": row.get("depth", "leaf"), "node_id": row.get("node_id", row.get("nodeId", entry_id)), "created_at": row.get("created_at", row.get("createdAt", ""))})
                for field in ("supported", "supported_count", "contradicted", "contradicted_count", "evidence_reliable", "evidenceReliable"):
                    if field in row:
                        hit[field] = row[field]
                kind = row.get("kind", "")
                if not kind and isinstance(text, str) and text.startswith("learning-observation/v1 "):
                    try:
                        observation, _ = json.JSONDecoder().raw_decode(text.removeprefix("learning-observation/v1 "))
                        correction = observation.get("correction", "")
                        candidate_kind, encoded_body = correction.split("/v1 ", 1)
                        if candidate_kind in FLEET_NOTE_KINDS or candidate_kind == "example":
                            kind, hit["body"] = candidate_kind, json.loads(encoded_body)
                    except (ValueError, TypeError, json.JSONDecodeError):
                        pass
                if not kind and isinstance(text, str):
                    for candidate_kind in ("preference", "avoid"):
                        marker = candidate_kind + "/v1 "
                        offset = text.find(marker)
                        if offset < 0:
                            continue
                        try:
                            body, _ = json.JSONDecoder().raw_decode(text[offset + len(marker):])
                        except json.JSONDecodeError:
                            continue
                        kind, hit["body"] = candidate_kind, body
                        break
                if kind:
                    hit["kind"] = kind
                    hit.setdefault("body", row.get("body", {}))
                if not row.get("outcome") and isinstance(text, str) and "learning-attempt/v1 " in text:
                    # Attempt records are text-encoded too; decode them so repeats(),
                    # last_outcome and last_fingerprint reflect what memory holds.
                    try:
                        decoded, _ = json.JSONDecoder().raw_decode(text[text.find("learning-attempt/v1 ") + len("learning-attempt/v1 "):])
                        if isinstance(decoded, dict):
                            row = dict(row, attempt_id=decoded.get("attemptId", decoded.get("attempt_id", "")),
                                       outcome=decoded.get("outcome", "unknown"),
                                       failure_fingerprint=decoded.get("failureFingerprint", decoded.get("failure_fingerprint", "")),
                                       operation=decoded.get("operation", ""), finished_at=decoded.get("finishedAt", ""),
                                       step_name=decoded.get("stepName", ""),
                                       context_key=decoded.get("contextKey", decoded.get("context_key", "")),
                                       provenance=decoded.get("provenance", ""))
                            hit["attempt_id"] = row["attempt_id"]
                    except (ValueError, TypeError):
                        pass
                if row.get("context_key") and key != "*" and row["context_key"] != key:
                    continue
                if row.get("operation") and row["operation"] != self.operation:
                    continue
                if row.get("provenance") == "test" and self._provenance() != "test":
                    continue
                if kinds and kind not in kinds:
                    continue
                created_at = hit.get("created_at", "")
                if since and created_at and str(created_at) < str(since):
                    continue
                if depth == "leaf" and hit.get("depth") != "leaf":
                    continue
                result["hits"].append(hit)
                if kind:
                    result["notes"].setdefault(kind, []).append(hit.get("body", {}))
                if row.get("outcome"):
                    result["attempts"].append(row)
            result["attempts"].sort(key=lambda row: str(row.get("finished_at", row.get("finishedAt", ""))))
            result["recall_status"] = "matched" if result["hits"] else "no_match"
            if depth == "tree":
                result["hits"].sort(key=lambda hit: (0 if hit.get("depth") in ("summary", "tree") else 1, str(hit.get("created_at", ""))))
            self.recall_status = result["recall_status"]
        except Exception:
            self.recall_status = "unavailable"
        self._recall_cache[cache_key] = result
        self._last_recall = result
        current_step = ACTIVE_STEP.get()
        if current_step and current_step in self._steps:
            self._steps[current_step]["decisions"] = _bound_decisions([
                {"entry_id": hit.get("entry_id", ""), "decision": "unassessed", "verdict": "unknown", "evidence_refs": []}
                for hit in result.get("hits", [])
            ])
            self._steps[current_step]["target_notes"] = [
                hit.get("entry_id", "") for hit in result.get("hits", []) if hit.get("kind") == "target-note"
            ]
        else:
            self._decisions = _bound_decisions([
                {"entry_id": hit.get("entry_id", ""), "decision": "unassessed", "verdict": "unknown", "evidence_refs": []}
                for hit in result.get("hits", [])
            ])
        return result

    def choose(self, options, default):
        if not isinstance(options, list) or not options:
            raise ValueError("choose options must be a non-empty list")
        if default not in options:
            raise ValueError("choose default must be one of options")
        # Always recall with the note kinds: the last recall may have been a
        # different query whose hits carry no preferences at all. The cache
        # makes a repeat of the same kind-targeted query free.
        ids = [item if isinstance(item, str) else item.get("id", item.get("option_id", "")) for item in options]
        # Notes are text-encoded as "<kind>/v1 {"option_id": ...}". Leading the recall with the
        # kind markers and the candidate option ids themselves makes the semantic query land on
        # exactly the notes that can change this decision, instead of on the attempt records
        # that accumulate in a usage scope and push the notes below the twenty-row bound.
        query = "preference/v1 avoid/v1 option_id " + " ".join(str(item) for item in ids[:10] if item) + " " + self.operation
        recalled = self.recall(query=query, kinds=["preference", "avoid"])
        avoided = {body.get("option_id") for body in recalled.get("notes", {}).get("avoid", []) if isinstance(body, dict)}
        candidates = [item for item in ids if item and item not in avoided]
        weights = {item: 0.0 for item in candidates}
        why = []
        preference_hits = []
        for hit in recalled.get("hits", []):
            body = hit.get("body", {})
            if hit.get("kind") != "preference" or not isinstance(body, dict) or body.get("option_id") not in weights:
                continue
            if hit.get("evidence_reliable", hit.get("evidenceReliable", True)) is False:
                continue
            supported = int(hit.get("supported", hit.get("supported_count", 0)) or 0)
            contradicted = int(hit.get("contradicted", hit.get("contradicted_count", 0)) or 0)
            base = supported - contradicted if supported or contradicted else 1
            weight = float(base) * _age_factor(hit.get("created_at", ""))
            weights[body["option_id"]] += weight
            preference_hits.append((hit, body["option_id"], weight))
            why.append({"entry_id": hit.get("entry_id", ""), "option_id": body["option_id"], "weight": weight})
        positive = {key: value for key, value in weights.items() if value > 0}
        best = max(positive.values()) if positive else 0
        winners = [key for key, value in positive.items() if value == best]
        eligible = [item for item in candidates if weights[item] >= 0]
        default_id = default if isinstance(default, str) else default.get("id", default.get("option_id", ""))
        fallback = default_id if default_id in eligible else (eligible[0] if eligible else None)
        selected = winners[0] if len(winners) == 1 else fallback
        self._decisions = _bound_decisions([{"entry_id": item.get("entry_id", ""), "decision": "applied" if isinstance(item.get("body"), dict) and item["body"].get("option_id") == selected else "unassessed", "decision_change": "selected:" + str(selected) if isinstance(item.get("body"), dict) and item["body"].get("option_id") == selected else "", "verdict": "unknown", "evidence_refs": []} for item in recalled.get("hits", [])])
        current_step = ACTIVE_STEP.get()
        if current_step and current_step in self._steps:
            self._steps[current_step]["decisions"] = list(self._decisions)
        return {"selected_id": selected, "source": "unavailable" if selected is None else "advice" if len(winners) == 1 and selected in weights and weights[selected] > 0 else "default", "conflicting": len(winners) > 1, "why": why,
                "learning": {"advice": self._decisions}}

    def infer(self, name, intent, inputs, schema, demos="key", verify=None):
        if not isinstance(schema, dict) or not isinstance(inputs, dict):
            raise ValueError("infer requires object inputs and schema")
        properties = schema.get("properties", {})
        route = "judge"
        if len(properties) == 1 and isinstance(next(iter(properties.values()), {}), dict) and next(iter(properties.values()), {}).get("enum"):
            route = "classify"
        elif isinstance(properties, dict) and properties:
            route = "extract"
        with self.step(name) as step:
            try:
                ai = getattr(self.owner, "ai")
                source = json.dumps({"intent": intent, "inputs": inputs}, sort_keys=True)
                examples = self.recall(kinds=["example"], key="*" if demos == "scope" else None, limit=8)
                examples["hits"] = [hit for hit in examples.get("hits", [])
                                    if hit.get("evidence_reliable", hit.get("evidenceReliable", True)) is not False
                                    and int(hit.get("contradicted", 0) or 0) == 0]
                instruction = ""
                step.record["demos_used"] = min(len(examples.get("hits", [])), 8)
                if examples.get("hits"):
                    instruction = "Demonstrations:\n" + "\n".join(hit.get("text", "") for hit in examples["hits"][:8])
                if route == "classify":
                    handle = ai.classify(source=source, schema=schema, instruction=instruction)
                elif route == "extract":
                    handle = ai.extract(source=source, schema=schema, instruction=instruction)
                else:
                    handle = ai.judge(source=source, schema=schema, instruction=instruction)
                output = handle.head(1)[0] if hasattr(handle, "head") else handle
                if isinstance(output, dict):
                    raw_value = output.get("valueJson", output.get("value_json"))
                    if isinstance(raw_value, str):
                        try:
                            output = json.loads(raw_value)
                        except json.JSONDecodeError:
                            pass
                    elif set(output).issubset({"value", "text", "validated", "usage", "model", "provider"}) and "value" in output:
                        output = output["value"]
                if not _matches_schema(output, schema):
                    raise ValueError("schema_mismatch")
                if verify is not None:
                    status, evidence = _verify_fragment(verify, output)
                    step.outcome(status, evidence)
                    if status != "verified_success":
                        raise StepFailed("inference_verification_failed")
                    self._examples.append({"inputs": inputs, "output": output, "step": str(name)})
                else:
                    step.outcome("unknown", [])
                return output
            except Exception:
                step.outcome("failed", []) if step.record and step.record.get("outcome") == "unknown" else None
                raise

    def act(self, name, intent, inputs, schema, bindings, attempts=3, fallback_fragment=None,
            *, verify=None, verifier_revision="", compatibility=None, baseline=None,
            min_verified=3, min_contexts=2, max_age_days=30, allow_ai=True):
        """Execute a verified section; AI is bounded adaptation, never the correctness oracle."""
        _validate_binding_specs(bindings, "act")
        if not callable(verify) or not isinstance(verifier_revision, str) or not verifier_revision.strip():
            raise ValueError("act requires verify and verifier_revision")
        if not isinstance(inputs, dict) or not isinstance(schema, dict):
            raise ValueError("act requires object inputs and schema")
        if not 1 <= int(min_verified) <= 100 or not 1 <= int(min_contexts) <= 64 or not 0 < float(max_age_days) <= 365:
            raise ValueError("invalid fragment eligibility policy")
        self.used = True
        self._begin()
        if baseline is None:
            baseline = self.baselines.get(str(name))
        allowed = set(bindings)
        specifications = getattr(self.owner, "_binding_contract_specs", self.binding_contracts)
        relevant = [item for item in specifications if isinstance(item, dict) and any(fnmatch.fnmatchcase(item.get("id", ""), pattern) for pattern in bindings)]
        # Availability is deliberately excluded: outages do not change code compatibility.
        relevant = [{k: v for k, v in item.items() if k not in ("reachable", "reachability_reason", "demand_start")} for item in relevant]
        contract = {"version": 2, "intent": intent, "schema": schema, "bindings": sorted(bindings),
                    "binding_contracts": sorted(relevant, key=lambda item: item.get("id", "")),
                    "verifier_revision": verifier_revision, "environment": compatibility or {}}
        contract_hash = digest(contract)
        with self.step(name) as step:
            provenance = self._provenance()
            cohort = "cohort:" + (provenance if provenance in ("test", "replay") else "live")
            cache_key = (self.operation, self.key or self._default_key(), str(name), contract_hash, cohort)
            step_key = "fragment-v2:" + digest(list(cache_key))
            candidate = self._fragment_cache.get(cache_key)
            if getattr(self.owner, "_bridge_url", ""):
                try:
                    durable = Tasks(self.owner, self.invocation_context, None, 100.0)._bridge("fragment_get", step_key=step_key)
                    # An authoritative no-match also invalidates an older process-local candidate.
                    candidate = durable.get("fragment") if durable.get("found") else None
                    self._fragment_cache.pop(cache_key, None)
                except Exception as exc:
                    self._fragment_delivery.append({"step": str(name), "delivery": "unavailable", "operation": "read", "error": str(exc)[:160]})
            if candidate and not isinstance(candidate, dict):
                candidate = None
            eligible = False
            if candidate:
                age = _age_days(candidate.get("last_verified_at", ""))
                eligible = (candidate.get("verified", 0) >= int(min_verified)
                            and candidate.get("contexts", 0) >= int(min_contexts)
                            and not candidate.get("contradicted_since_edit", 0)
                            and age <= float(max_age_days))
            fragment = candidate.get("fragment") if eligible else None
            source = "cache" if fragment else "model"
            baseline_error = None
            if fragment is None and baseline:
                try:
                    fragment = validate_baseline(baseline, contract)
                    source = "baseline"
                except Exception as exc:
                    baseline_error = str(exc)
            # Legacy embedded fallbacks remain executable but undergo the same verifier.
            if fragment is None and fallback_fragment:
                fragment, source = normalize(fallback_fragment), "fallback"
            errors = []
            model_calls = 0
            used_baseline = source in ("baseline", "fallback")
            forbidden_hashes = set()
            for _ in range(max(1, min(int(attempts), 5))):
                invocation_start = len(getattr(self.owner, "_invocations", []))
                attempted_effects = []
                try:
                    if fragment is None:
                        if not allow_ai:
                            raise StepFailed("adaptation_unavailable", baseline_error=baseline_error)
                        model_calls += 1
                        context = {"intent": intent, "inputs": inputs, "schema": schema,
                                   "bindings": sorted(bindings), "binding_contracts": relevant,
                                   "previous_candidate": candidate.get("fragment", "") if candidate else "",
                                   "failures": errors[-3:]}
                        encoded = json.dumps(context, sort_keys=True, allow_nan=False)
                        if len(encoded.encode()) > 32768:
                            raise StepFailed("adaptation_context_bound")
                        reply = getattr(self.owner, "ai").write(source=encoded, schema={"type": "object", "properties": {"fragment": {"type": "string"}}, "required": ["fragment"]},
                            instruction="Return JSON containing only fragment: def step(inputs, bindings). Use the declared binding allow-set. Fix the reported failures. Do not import modules or access private attributes.")
                        fragment = _fragment_reply(reply)
                        source = "model"
                    fragment = normalize(fragment)
                    if digest(fragment) in forbidden_hashes:
                        raise StepFailed("repeated_failed_candidate")
                    output = wrap(fragment)(inputs, _AllowedBindings(self.owner, allowed, effects=attempted_effects))
                    if not _matches_schema(output, schema):
                        raise StepFailed("schema_mismatch")
                    status, evidence = _verify_fragment(verify, output)
                    if status == "unavailable":
                        raise StepFailed("verification_unavailable")
                    if status != "verified_success":
                        raise StepFailed("verification_failed", evidence=evidence)
                    step.outcome(status, evidence)
                    step.record["fragment_source"] = source
                    step.record["model_calls"] = model_calls
                    self._fragment_candidates.append({"cache_key": cache_key, "step_key": step_key,
                        "fragment": fragment, "step_name": str(name), "input_digest": digest(inputs),
                        "compatibility": contract, "evidence": evidence, "previous": candidate or {},
                        "attempt_id": self._record.get("attempt_id", self.attempt_id), "source": source})
                    # Keep raw inputs and outputs out of Memory traces and portable assets.
                    step.note("trace", {"fragment": "sha256:" + digest(fragment), "bindings": sorted(allowed),
                                        "output": {"digest": digest(output)}})
                    return {"output": output, "fragment_source": source, "model_calls": model_calls,
                            "compatibility_digest": contract_hash, "step_key": step_key,
                            "verified": True, "eligibility": "reviewed" if source == "baseline" else "qualified" if source == "cache" else "probation"}
                except Exception as exc:
                    invoked = getattr(self.owner, "_invocations", [])[invocation_start:]
                    wrote = bool(attempted_effects) or any(item.get("effect") in ("write", "destructive") for item in invoked if isinstance(item, dict))
                    unavailable = _fragment_unavailable(exc)
                    errors.append({"class": getattr(exc, "reason", type(exc).__name__), "detail": str(exc)[:240],
                                   "fragment": fragment[:4096] if isinstance(fragment, str) else ""})
                    if fragment and not unavailable:
                        forbidden_hashes.add(digest(fragment))
                        self._reject_fragment(cache_key, step_key, fragment, str(name), contract, step.step_id)
                    if wrote:
                        step.record["outcome"] = "unknown"
                        step.record["error_class"] = "uncertain_effects"
                        raise StepFailed("uncertain_effects", errors=errors) from exc
                    if unavailable:
                        # A model outage may fall back to the reviewed baseline, but never retries domain writes.
                        if not used_baseline and baseline:
                            try:
                                fragment = validate_baseline(baseline, contract)
                                source, used_baseline = "baseline", True
                                continue
                            except Exception:
                                pass
                        step.outcome("unavailable", [])
                        raise StepFailed("adaptation_unavailable", errors=errors) from exc
                    candidate = {"fragment": fragment or ""}
                    fragment, source = None, "model"
            step.outcome("failed", [])
            raise StepFailed("fragment_failed", errors=errors)

    def _reject_fragment(self, cache_key, step_key, fragment, step_name, compatibility, attempt_id):
        self._fragment_cache.pop(cache_key, None)
        self._fragment_stats.pop(cache_key, None)
        self._fragment_write(step_key=step_key, fragment=fragment, contradicted_delta=1,
                             step_name=step_name, compatibility=compatibility, attempt_id=attempt_id)

    def _fragment_write(self, **request):
        if not getattr(self.owner, "_bridge_url", ""):
            return
        try:
            Tasks(self.owner, self.invocation_context, None, 100.0)._bridge("fragment_put", **request,
                source=getattr(self, "_source", ""),
                source_program_id=self.invocation_context.get({}).get("program_id", ""))
            self._fragment_delivery.append({"step": request.get("step_name", ""), "delivery": "delivered", "operation": "write"})
        except Exception as exc:
            self._fragment_delivery.append({"step": request.get("step_name", ""), "delivery": "unavailable", "operation": "write", "error": str(exc)[:160]})

    def note(self, kind, body, evidence=None):
        kind = _text(kind, "kind")
        if kind == "example":
            raise ValueError("example notes are kernel-written only")
        if kind not in FLEET_NOTE_KINDS and kind not in self.note_kinds:
            raise ValueError("unknown learning note kind")
        if not isinstance(body, dict):
            raise ValueError("invalid learning note body")
        if kind in NOTE_FIELDS:
            if set(body) - NOTE_FIELDS[kind] or not NOTE_REQUIRED[kind].issubset(body):
                raise ValueError("invalid learning note body")
            if kind == "avoid" and not (body.get("fingerprint") or body.get("option_id")):
                raise ValueError("avoid requires fingerprint or option_id")
            if kind == "trace" and "bindings" in body and not isinstance(body["bindings"], list):
                raise ValueError("trace bindings must be an array")
        elif not isinstance(self.note_kinds[kind], dict):
            raise ValueError("invalid learning note schema")
        _bound_note_values(body)
        encoded = json.dumps(body, sort_keys=True, separators=(",", ":"), ensure_ascii=True)
        if len(encoded.encode("utf-8")) > 4096:
            raise ValueError("learning note is too large")
        if evidence is not None and (not isinstance(evidence, list) or any(not isinstance(v, str) or not v.strip() for v in evidence)):
            raise ValueError("evidence must be a list of non-empty strings")
        self.used = True
        self._begin()
        item = {"kind": kind, "body": body, "evidence": list(evidence or [])[:20]}
        self._notes.append(item)
        return item

    def delegate(self, name, brief, inputs, schema, bindings, capabilities=None, verify=None, wait_seconds=30):
        _validate_binding_specs(bindings, "delegate")
        if not isinstance(capabilities, (dict, type(None))):
            raise ValueError("delegate capabilities must be an object")
        if not callable(verify):
            raise ValueError("delegate verify must be callable")
        with self.step(name) as step:
            prior = self.recall(kinds=["target-note", "avoid"], limit=3)
            preamble = "Prior knowledge for this task (from memory): " + json.dumps({
                "target_notes": prior.get("notes", {}).get("target-note", []),
                "avoid": prior.get("notes", {}).get("avoid", []),
                "last_attempts": [{"outcome": row.get("outcome"), "approach": row.get("approach", "")} for row in prior.get("attempts", [])[-3:]],
            }, sort_keys=True, separators=(",", ":"))[:4000]
            request = {"brief": preamble + "\n" + _text(brief, "brief"), "inputs": inputs,
                       "bindings": bindings, "capabilities": capabilities or {}}
            started = getattr(self.owner, "agent").start(**request)
            collected = getattr(self.owner, "agent").collect(started, wait_seconds=max(0, min(int(wait_seconds), 300)))
            rows = collected.head(1) if hasattr(collected, "head") else [collected]
            output = rows[0] if rows else {}
            if not _matches_schema(output, schema):
                step.outcome("failed", [])
                raise StepFailed("schema_mismatch")
            verification = verify(output)
            if not isinstance(verification, (tuple, list)) or len(verification) != 2:
                raise ValueError("delegate verify must return (status, evidence)")
            status, evidence = verification
            step.note("parameter", {"name": "approach", "value": "agent-manager"})
            step.note("trace", {"bindings": bindings, "output": output, "attribution": "unavailable"})
            step.outcome(status, evidence)
            return output

    def finalize(self, result):
        if not self.started:
            return result
        if self._record is None:
            return result
        result = result if isinstance(result, dict) else {"status": "ok", "evidence": []}
        if result.get("status") == "failed" and self._outcome == "verified_success":
            self._outcome, self._evidence = "unknown", []
            for decision in self._decisions:
                if decision.get("derived"):
                    decision.update(verdict="unknown", evidence_refs=[], derived=False)

        # Failed feedback is sealed into the server-owned finish outbox below.
        if self._outcome == "verified_success":
            for example in self._examples:
                stable = {key: value for key, value in example["inputs"].items()
                          if key not in self.free_text_inputs and not str(key).endswith("_free_text")}
                encoded = json.dumps(stable, sort_keys=True, separators=(",", ":"), ensure_ascii=True, default=str)
                self._notes.append({"kind": "example", "body": {
                    "inputs_digest": hashlib.sha256(encoded.encode("utf-8")).hexdigest(),
                    "inputs_redacted": stable,
                    "output": example["output"],
                }, "evidence": ["inference:" + example["step"]]})
        attempt = {"attempt_id": self._record.get("attempt_id", self.attempt_id),
                   "task_id": self._record.get("task_id", self.task_id),
                   "operation": self._record.get("operation", self.operation),
                   "context_key": self._record.get("context_key", self.key),
                   "started_at": self._record.get("started_at", _now()),
                   "task_started_at": self._record.get("task_started_at", _now()),
                   "attempt_number": self._record.get("attempt_number", 1),
                   "provenance": self._record.get("provenance", "agent"),
                   "finished_at": _now(), "outcome": self._outcome,
                   "trigger": self.operation, "approach": "learn.verbs/v1",
                   "evidence_refs": self._evidence, "recall_status": self.recall_status,
                   "advice": self._decisions}
        attempt.update(self._measurements)
        if self._outcome == "failed":
            attempt["failure_fingerprint"] = getattr(self, "_failure_fingerprint", _failure_fingerprint(self.operation, "program", "failed"))
        child_attempts = []
        child_observations = []
        for step in self._steps.values():
            if step.get("bound"):
                continue
            child = {"attempt_id": step["step_id"], "task_id": attempt["task_id"],
                     "operation": attempt["operation"], "context_key": step["key"],
                     "started_at": step.get("opened_at", attempt["started_at"]),
                     "finished_at": step.get("closed_at") or attempt["finished_at"],
                     "task_started_at": attempt["task_started_at"], "attempt_number": attempt["attempt_number"],
                     "provenance": attempt["provenance"], "trigger": "step:" + step["name"],
                     "approach": "learn.step/v1", "outcome": step.get("outcome", "unknown"),
                     "evidence_refs": step.get("evidence", []), "recall_status": "no_match",
                     "parent_attempt_id": attempt["attempt_id"], "step_name": step["name"], "advice": step.get("decisions", [])}
            if child["advice"]:
                child["recall_status"] = "matched"
            if child["outcome"] == "failed":
                child["failure_fingerprint"] = step.get("failure_fingerprint", _failure_fingerprint(self.operation, step["name"], "failed"))
            child_attempts.append(child)
            for item in step.get("notes", []):
                child_observations.append((child["attempt_id"], item))
        attempts = [attempt] + child_attempts
        observations = [{"attempt_id": attempt["attempt_id"], "disposition": "unknown",
                         "evidence_refs": item["evidence"], "method_revision": "learn." + item["kind"],
                         "correction": item["kind"] + "/v1 " + json.dumps(item["body"], sort_keys=True, separators=(",", ":")),
                        "provenance": attempt["provenance"], "observed_at": attempt["finished_at"]}
                        for item in self._notes]
        observations.extend({"attempt_id": step_id, "disposition": "unknown",
                             "evidence_refs": item["evidence"], "method_revision": "learn." + item["kind"],
                             "correction": item["kind"] + "/v1 " + json.dumps(item["body"], sort_keys=True, separators=(",", ":")),
                             "provenance": attempt["provenance"], "observed_at": attempt["finished_at"]}
                            for step_id, item in child_observations)
        for receipt in self._feedback:
            if receipt["delivery"] != "delivered":
                observations.append(receipt["observation"])
                receipt["delivery"] = "pending" if getattr(self.owner, "_bridge_url", "") else "failed"
        finish_inputs = {"scope": self._record.get("scope", self.scope), "attempt": attempt, "attempts": attempts, "observations": observations}
        learning_receipt = {"steps": [{"name": s["name"], "key": s["key"], "outcome": s["outcome"], "attempt_id": s.get("step_id", ""), "parent_attempt_id": attempt["attempt_id"], "error_class": s.get("error_class", ""), "bound": bool(s.get("bound")), "demos_used": s.get("demos_used", 0)} for s in self._steps.values()],
                            "advice": self._decisions, "attempts": attempts,
                            "recall_status": self.recall_status, "measurements": dict(self._measurements),
                            "feedback": [{k: v for k, v in item.items() if k not in ("observation", "scope")} for item in self._feedback]}
        for candidate in self._fragment_candidates:
            if self._outcome != "verified_success" or result.get("status") == "failed":
                # Root failures do not prove every successful child was wrong.
                continue
            cache_key = candidate["cache_key"]
            previous = candidate["previous"]
            if previous.get("fragment") != candidate["fragment"]:
                previous = {}
            contexts = set(previous.get("input_digests", []))
            contexts.add(candidate["input_digest"])
            record = {"fragment": candidate["fragment"], "verified": previous.get("verified", 0) + 1,
                      "contexts": max(previous.get("contexts", 0), len(contexts)),
                      "input_digests": sorted(contexts)[:64], "last_verified_at": _now(),
                      "contradicted_since_edit": 0}
            self._fragment_cache[cache_key] = record
            self._fragment_write(step_key=candidate["step_key"], fragment=candidate["fragment"],
                verified_delta=1, step_name=candidate["step_name"], compatibility=candidate["compatibility"],
                attempt_id=candidate["attempt_id"], input_digest=candidate["input_digest"], evidence=candidate["evidence"], cache_hit=candidate["source"] == "cache")
        learning_receipt["fragments"] = list(self._fragment_delivery)
        if getattr(self.owner, "_bridge_url", ""):
            tasks = Tasks(self.owner, self.invocation_context, None, 100.0)
            completed = tasks._bridge("complete", attempt_id=self.attempt_id, result=result,
                finish_inputs=finish_inputs,
                outcome=self._outcome, resume_token=self.resume_token)
            result["learning"] = {**learning_receipt, **{key: completed.get(key) for key in
                ("task_id", "attempt_id", "attempt_number", "outcome", "delivery", "last_error", "resume_token")}}
        else:
            result["learning"] = {"task_id": self.task_id, "attempt_id": self.attempt_id,
                "attempt_number": 1, "outcome": self._outcome, "delivery": "delivered",
                "resume_token": self.resume_token, "notes": self._notes, "observations": observations, "advice": self._decisions,
                "attempts": attempts, "steps": learning_receipt["steps"], "recall_status": self.recall_status,
                "fragments": list(self._fragment_delivery), "measurements": dict(self._measurements), "feedback": [{k: v for k, v in item.items() if k not in ("observation", "scope")} for item in self._feedback]}
        return result

    def _durable_step_key(self, cache_key):
        return "|".join(str(item) for item in cache_key)


def _text(value, name):
    if not isinstance(value, str) or not value.strip() or len(value.encode("utf-8")) > 1024:
        raise ValueError(name + " must be bounded text")
    return value.strip()


def _canonical_key(value):
    if isinstance(value, str):
        return _text(value, "key")
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=True, default=str).encode()).hexdigest()


def _now():
    return datetime.datetime.now(datetime.timezone.utc).isoformat(timespec="milliseconds").replace("+00:00", "Z")


def _failure_fingerprint(operation, step, error_class):
    raw = json.dumps([str(operation), str(step), str(error_class)], separators=(",", ":"), sort_keys=True)
    return "failure-v1:" + hashlib.sha256(raw.encode("utf-8")).hexdigest()


def _age_factor(created_at):
    if not created_at:
        return 1.0
    try:
        stamp = datetime.datetime.fromisoformat(str(created_at).replace("Z", "+00:00"))
        if stamp.tzinfo is None:
            return 1.0
        age_days = max(0.0, (datetime.datetime.now(datetime.timezone.utc) - stamp).total_seconds() / 86400.0)
        return 0.5 ** (age_days / 30.0)
    except (TypeError, ValueError, OverflowError):
        return 1.0


ADVICE_DECISION_LIMIT = 10


def _bound_decisions(decisions):
    """Memory records at most ten advice uses per attempt; a kind-targeted recall
    can expose twenty rows. Keep every applied decision, then the earliest exposures."""
    applied = [d for d in decisions if d.get("decision") == "applied"]
    rest = [d for d in decisions if d.get("decision") != "applied"]
    return (applied + rest)[:ADVICE_DECISION_LIMIT]


MEASUREMENT_KEYS = {"first_action_at", "tool_round_trips", "visual_reasoning_calls", "reused_workflow"}


def _validate_measurements(measurements):
    """The four optional attempt measurements Memory's compare-outcomes reads."""
    if not isinstance(measurements, dict) or set(measurements) - MEASUREMENT_KEYS:
        raise ValueError("measurements accepts only " + ", ".join(sorted(MEASUREMENT_KEYS)))
    for name in ("tool_round_trips", "visual_reasoning_calls"):
        if name in measurements and (type(measurements[name]) is not int or not 0 <= measurements[name] <= 100000):
            raise ValueError(name + " must be an int in 0..100000")
    if "reused_workflow" in measurements and type(measurements["reused_workflow"]) is not bool:
        raise ValueError("reused_workflow must be a bool")
    if "first_action_at" in measurements and (not isinstance(measurements["first_action_at"], str) or not measurements["first_action_at"].strip()):
        raise ValueError("first_action_at must be an RFC3339 timestamp")
    return dict(measurements)


def _validate_outcome(status, evidence):
    if status not in ("verified_success", "failed", "unavailable", "unknown"):
        raise ValueError("invalid learning outcome")
    evidence = evidence or []
    if not isinstance(evidence, list) or any(not isinstance(v, str) or not v.strip() for v in evidence):
        raise ValueError("evidence must be a list of non-empty strings")
    if status == "verified_success" and not evidence:
        raise ValueError("verified_success requires evidence")


def _matches_schema(value, schema):
    """Bounded JSON Schema subset; unsupported validation keywords are explicit errors."""
    import math
    import re
    if isinstance(schema, bool):
        return schema
    if not isinstance(schema, dict):
        raise ValueError("schema must be an object or boolean")
    supported = {"type", "properties", "required", "additionalProperties", "items", "enum", "const",
                 "minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum", "minLength", "maxLength", "pattern",
                 "minItems", "maxItems", "uniqueItems", "minProperties", "maxProperties", "anyOf", "allOf", "oneOf", "not",
                 "description", "title", "$schema", "$id", "default", "examples"}
    if set(schema) - supported:
        raise ValueError("unsupported schema keyword: " + sorted(set(schema) - supported)[0])
    if "enum" in schema and not any(type(value) is type(item) and value == item for item in schema["enum"]):
        return False
    if "const" in schema and (type(value) is not type(schema["const"]) or value != schema["const"]):
        return False
    for key in ("anyOf", "allOf", "oneOf"):
        if key in schema:
            matches = [_matches_schema(value, child) for child in schema[key]]
            if not ({"anyOf": any(matches), "allOf": all(matches), "oneOf": sum(matches) == 1}[key]):
                return False
    if "not" in schema and _matches_schema(value, schema["not"]):
        return False
    kind = schema.get("type")
    if isinstance(kind, list):
        return any(_matches_schema(value, dict(schema, type=item)) for item in kind)
    types = {"object": lambda: isinstance(value, dict), "array": lambda: isinstance(value, list),
             "boolean": lambda: type(value) is bool, "string": lambda: isinstance(value, str),
             "integer": lambda: type(value) is int, "number": lambda: type(value) in (int, float) and math.isfinite(value),
             "null": lambda: value is None}
    if kind is not None and (kind not in types or not types[kind]()):
        return False
    if isinstance(value, dict):
        properties = schema.get("properties", {})
        if any(key not in value for key in schema.get("required", [])):
            return False
        if not schema.get("minProperties", 0) <= len(value) <= schema.get("maxProperties", 10000):
            return False
        return all(_matches_schema(child, properties.get(key, schema.get("additionalProperties", True))) for key, child in value.items())
    if isinstance(value, list):
        if not schema.get("minItems", 0) <= len(value) <= min(schema.get("maxItems", 10000), 10000):
            return False
        if schema.get("uniqueItems") and len({digest(item) for item in value}) != len(value):
            return False
        return all(_matches_schema(item, schema.get("items", {})) for item in value)
    if isinstance(value, str):
        return (schema.get("minLength", 0) <= len(value) <= schema.get("maxLength", 65536)
                and ("pattern" not in schema or re.search(schema["pattern"], value) is not None))
    if type(value) in (int, float):
        return (math.isfinite(value) and schema.get("minimum", -math.inf) <= value <= schema.get("maximum", math.inf)
                and schema.get("exclusiveMinimum", -math.inf) < value < schema.get("exclusiveMaximum", math.inf))
    return True


class _AllowedBindings:
    def __init__(self, owner, allowed, path="", dry_run=False, effects=None):
        self._owner, self._allowed, self._path, self._dry_run = owner, allowed, path, dry_run
        self._effects = effects if effects is not None else []

    def __getattr__(self, part):
        path = (self._path + "." + part).strip(".")
        binding_id = path.replace("_", "-").replace(".", "/")
        if any(fnmatch.fnmatchcase(binding_id, pattern.replace("_", "-").replace(".", "/")) for pattern in self._allowed):
            target = self._owner
            for item in path.split("."):
                target = getattr(target, item)
            if self._dry_run and getattr(target, "effect", "read") in ("write", "destructive"):
                return _DryRunBinding(binding_id)
            return _FragmentBinding(target, self._effects, binding_id)
        if any(pattern.replace("_", "-").replace(".", "/").startswith(binding_id + "/") or
               pattern.replace("_", "-").replace(".", "/").startswith(binding_id + "/*")
               for pattern in self._allowed):
            return _AllowedBindings(self._owner, self._allowed, path, self._dry_run, self._effects)
        raise StepFailed("binding_not_allowed", binding=binding_id)


class _FragmentBinding:
    """Expose invocation only, never the bridge object's mutable transport or audit fields."""
    __slots__ = ("_target", "_effects", "_binding_id")

    def __init__(self, target, effects, binding_id):
        self._target, self._effects, self._binding_id = target, effects, binding_id

    def __call__(self, *args, **kwargs):
        # A lost response does not prove that the remote write never happened.
        if getattr(self._target, "effect", "read") in ("write", "destructive"):
            self._effects.append(self._binding_id)
        return self._target(*args, **kwargs)


class _DryRunBinding:
    def __init__(self, binding_id):
        self.binding_id = binding_id
        self.effect = "write"

    def __call__(self, *args, **kwargs):
        raise StepFailed("shadow_write_refused", binding=self.binding_id)


def _validate_binding_specs(bindings, verb):
    if not isinstance(bindings, list) or (not bindings and verb != "act") or any(not isinstance(item, str) or not item.strip() for item in bindings):
        raise ValueError(verb + " bindings must be a non-empty list of binding ids")
    for item in bindings:
        if len(item.encode("utf-8")) > 256 or any(char.isspace() for char in item):
            raise ValueError(verb + " binding ids must be bounded and whitespace-free")
        parts = item.split("/")
        if len(parts) != 3 or not all(parts) or "*" in parts[0] or any("*" in part and part != "*" for part in parts[1:]):
            raise ValueError(verb + " binding ids must use scenario/group/command with optional group or command globs")


def _bound_note_values(value):
    """Keep note payloads bounded before they enter the sealed finish intent."""
    if isinstance(value, str):
        if len(value.encode("utf-8")) > 512:
            raise ValueError("learning note text is too large")
    elif isinstance(value, dict):
        for child in value.values():
            _bound_note_values(child)
    elif isinstance(value, list):
        if len(value) > 64:
            raise ValueError("learning note list is too large")
        for child in value:
            _bound_note_values(child)


def _age_days(stamp):
    try:
        value = datetime.datetime.fromisoformat(str(stamp).replace("Z", "+00:00"))
        return max(0, (datetime.datetime.now(datetime.timezone.utc) - value).total_seconds() / 86400)
    except (ValueError, TypeError):
        return float("inf")


def _fragment_reply(reply):
    value = reply.head(1)[0] if hasattr(reply, "head") else reply
    for _ in range(4):
        if isinstance(value, dict):
            value = next((value[k] for k in ("fragment", "valueJson", "value_json", "value", "output", "text") if k in value), None)
        elif isinstance(value, str):
            if value.strip().startswith("def step("):
                return value.strip()
            if value.strip().startswith("```"):
                return value.strip().split("```", 2)[1].removeprefix("python").strip()
            try:
                value = json.loads(value)
            except json.JSONDecodeError:
                break
    raise StepFailed("fragment_reply_invalid")


def _verify_fragment(verify, output):
    value = verify(output)
    if not isinstance(value, (tuple, list)) or len(value) != 2:
        raise ValueError("verify must return (status, evidence)")
    status, evidence = value
    _validate_outcome(status, evidence)
    return status, evidence


def _fragment_unavailable(exc):
    reason = str(exc).lower()
    return isinstance(exc, (ConnectionError, TimeoutError, OSError)) or any(word in reason for word in (
        "unavailable", "unreachable", "connection refused", "deadline_exceeded", "spend_exceeded", "refused_no_grant", "bridge_transport"))
