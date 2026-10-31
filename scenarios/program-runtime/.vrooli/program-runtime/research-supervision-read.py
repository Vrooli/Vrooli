"""Read explicitly selected research subjects without inferring task success."""
import json

inputs = program.inputs()
envelope = program.envelope("program-runtime.research-supervision-read", "1", inputs={})
envelope["signals"] = {"sources": {}, "decision": "review_required"}


def step_validate():
    threads = inputs.get("threads", [])
    runs = inputs.get("run_ids", [])
    session = inputs.get("session_id", "")
    cursors = inputs.get("event_cursors", {})
    for label, values, maximum in (("threads", threads, 16), ("run_ids", runs, 8)):
        if not isinstance(values, list) or len(values) > maximum or any(not isinstance(x, str) or not x.strip() or len(x) > 128 for x in values) or len(set(values)) != len(values):
            return program.fail("failed", "invalid_input", label + " must be a bounded list of unique nonempty IDs", "validate")
    if not threads and not runs:
        return program.fail("failed", "invalid_input", "provide at least one explicit thread or run ID", "validate")
    if threads and (not isinstance(session, str) or not session.strip() or len(session) > 256):
        return program.fail("failed", "invalid_input", "threads require a bounded authorized session_id", "validate")
    if not isinstance(cursors, dict) or any(key not in runs or (value is not None and (isinstance(value, bool) or not isinstance(value, int) or not 0 <= value <= 9223372036854775807)) for key, value in cursors.items()):
        return program.fail("failed", "invalid_input", "event_cursors must map selected run IDs to nonnegative int64 sequences or null", "validate")
    envelope["inputs"] = {"threads": list(threads), "run_ids": list(runs), "event_cursors": dict(cursors)}
    if threads:
        envelope["inputs"]["session_id"] = session
    return "collect"


def read_child(name, call):
    try:
        handle = call()
        row = program.child_envelope(handle, invalid_class="invalid_response", where="collect:" + name)
        meta = handle.meta()
        envelope["signals"]["sources"][name] = {"status": row["status"], "signals": row["signals"], "errors": row["errors"]}
        envelope["evidence"].append({"program": name, "artifact": {k: meta[k] for k in ("program", "version", "digest") if k in meta}})
    except Exception as exc:
        envelope["signals"]["sources"][name] = {"status": "unknown", "signals": {}, "errors": [{"class": "source_unavailable", "detail": str(exc)[:180]}]}
        envelope["errors"].append({"class": "source_unavailable", "detail": name + ": " + str(exc)[:180], "where": "collect"})


def event_integer(value):
    # Protobuf JSON encodes int64 as a decimal string. Do not coerce floats or
    # booleans: rounding a cursor can cause the next review to skip evidence.
    if isinstance(value, bool):
        raise ValueError("owner returned a non-integer event value")
    if isinstance(value, str):
        if not value or len(value) > 19 or any(c < "0" or c > "9" for c in value):
            raise ValueError("owner returned a non-integer event value")
        value = int(value)
    if not isinstance(value, int) or not 0 <= value <= 9223372036854775807:
        raise ValueError("owner returned an out-of-range event value")
    return value


def read_event_pages(cursors):
    pages = []
    for run_id, cursor in cursors.items():
        page = {"run_id": run_id, "after_sequence": cursor, "status": "unknown"}
        try:
            args = {"run_id": run_id, "limit": 200, "summary_only": True}
            if cursor is not None:
                args["after_sequence"] = cursor
            handle = agent_manager.run.event_page(**args)
            if handle.count() != 0:
                raise ValueError("owner returned event payloads instead of a summary")
            meta = handle.meta()
            summary = meta.get("summary")
            if not isinstance(summary, dict) or not isinstance(summary.get("counts", {}), dict):
                raise ValueError("owner did not return an event summary")
            counts = summary.get("counts", {})
            if len(counts) > 200 or any(not isinstance(key, str) or not key or len(key.encode("utf-8")) > 128 for key in counts):
                raise ValueError("owner returned an unbounded event summary")
            values = [event_integer(count) for count in counts.values()]
            total = sum(values)
            if any(count == 0 for count in values) or total > 200:
                raise ValueError("owner returned counts outside the requested event page")
            has_more = meta.get("hasMore", False)
            if not isinstance(has_more, bool):
                raise ValueError("owner returned a non-boolean page continuation")
            if (total and summary.get("latestSequence") is None) or (not total and (summary.get("latestSequence") is not None or has_more)):
                raise ValueError("owner returned a summary without a usable cursor")
            latest = summary.get("latestSequence")
            if latest is not None:
                latest = event_integer(latest)
                if cursor is not None and latest <= cursor:
                    raise ValueError("owner returned a nonadvancing event cursor")
                # Event sequences are unique nonnegative integers, so a page
                # cannot contain more events than the available sequence span.
                if total > latest - (cursor if cursor is not None else -1):
                    raise ValueError("owner returned counts inconsistent with the event cursor")
            page.update(status="available", counts=counts,
                        next_cursor=latest if latest is not None else cursor,
                        has_more=has_more)
        except Exception as exc:
            page["error"] = str(exc)[:180]
            envelope["errors"].append({"class": "source_unavailable", "detail": "event_page " + run_id + ": " + str(exc)[:180], "where": "collect"})
        pages.append(page)
    envelope["signals"]["sources"]["event_pages"] = {
        "status": "ok" if all(page["status"] == "available" and not page["has_more"] for page in pages) else "partial",
        "pages": pages,
    }
    envelope["evidence"].append({"binding": "agent-manager/run/event-page"})


def step_collect():
    envelope["phase"] = "collect"
    threads, runs = envelope["inputs"]["threads"], envelope["inputs"]["run_ids"]
    if threads:
        read_child("native_state", lambda: lib.web_console.native_thread_state(session_id=inputs["session_id"], threads=threads))
    else:
        envelope["signals"]["sources"]["native_state"] = {"status": "not_requested"}
    if runs:
        read_child("run_evidence", lambda: lib.agent_manager.investigation_evidence(run_ids=runs))
    else:
        envelope["signals"]["sources"]["run_evidence"] = {"status": "not_requested"}
    if envelope["inputs"]["event_cursors"]:
        read_event_pages(envelope["inputs"]["event_cursors"])
    sources = envelope["signals"]["sources"].values()
    envelope["status"] = "ok" if all(source["status"] == "ok" for source in sources) else "partial"
    return "report"


def step_report():
    envelope["phase"] = "report"
    sources = envelope["signals"]["sources"]
    attention = []
    native = sources.get("native_state", {})
    native_signals = native.get("signals") if isinstance(native.get("signals"), dict) else {}
    native_rows = native_signals.get("threads") if isinstance(native_signals.get("threads"), list) else []
    for row in native_rows[:16]:
        if not isinstance(row, dict):
            continue
        goal = row.get("goalStatus", row.get("goal_status", ""))
        if isinstance(goal, str) and goal.lower() in ("blocked", "budget_limited", "failed"):
            attention.append({"source": "native_state", "subject": row.get("threadId", row.get("thread_id", "")), "signal": "goal_" + goal.lower()})
    run_evidence = sources.get("run_evidence", {})
    run_signals = run_evidence.get("signals") if isinstance(run_evidence.get("signals"), dict) else {}
    run_rows = run_signals.get("subjects") if isinstance(run_signals.get("subjects"), list) else []
    for row in run_rows[:8]:
        if not isinstance(row, dict):
            continue
        report = row.get("report") if isinstance(row.get("report"), dict) else {}
        status = report.get("status")
        if isinstance(status, str) and status.lower() in ("failed", "cancelled"):
            attention.append({"source": "run_evidence", "subject": row.get("run_id", ""), "signal": "run_" + status.lower(), "freshness": row.get("source_freshness", "unknown")})
    for page in sources.get("event_pages", {}).get("pages", []):
        if page.get("has_more") is True:
            attention.append({"source": "event_pages", "subject": page.get("run_id", ""), "signal": "event_backlog"})
    envelope["signals"]["attention"] = attention
    # print adds one newline to the captured output.
    limit = 65535
    encoded = json.dumps(envelope, allow_nan=False, separators=(",", ":"))
    if len(encoded.encode("utf-8")) > limit:
        # Omit only the largest source data until the bounded packet fits.
        # A missing source is explicitly unknown; unaffected siblings remain
        # attributable and must still be reviewed by the supervisor.
        sources = envelope["signals"]["sources"]
        sizes = [(len(json.dumps(value, allow_nan=False, separators=(",", ":")).encode("utf-8")), name)
                 for name, value in sources.items() if value.get("status") != "not_requested"]
        envelope["status"] = "partial"
        envelope["signals"]["output_truncated"] = True
        envelope["errors"].append({"class": "output_limit", "detail": "one or more source results were omitted; inspect their owners separately", "where": "report"})
        encoded = json.dumps(envelope, allow_nan=False, separators=(",", ":"))
        for _, name in sorted(sizes, reverse=True):
            if len(encoded.encode("utf-8")) <= limit:
                break
            sources[name] = {"status": "unknown", "omitted_due_to_output_limit": True}
            encoded = json.dumps(envelope, allow_nan=False, separators=(",", ":"))
        if len(encoded.encode("utf-8")) > limit:
            # Even the packet's own metadata did not fit. Do not emit an
            # oversized envelope or suggest that any source was reviewed.
            envelope["status"] = "failed"
            envelope["signals"] = {"decision": "review_required", "output_truncated": True}
            envelope["evidence"] = []
            envelope["errors"] = [{"class": "output_limit", "detail": "inspect source owners separately", "where": "report"}]
            encoded = json.dumps(envelope, allow_nan=False, separators=(",", ":"))
    print(encoded)
    return None

program.run({"validate": step_validate, "collect": step_collect, "report": step_report}, "validate")
