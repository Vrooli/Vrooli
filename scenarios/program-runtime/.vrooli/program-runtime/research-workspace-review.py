"""Summarize the caller-authorized research root without changing it."""
import json
import itertools
import os
import stat

inputs = program.inputs()
envelope = program.envelope("program-runtime.research-workspace-review", "1")
envelope["signals"] = {"unique_runs": 0, "without_guide": 0, "without_checkpoint": 0,
                       "largest": [], "without_guide_examples": [], "without_checkpoint_examples": [],
                       "truncated": False, "skipped_outside_root": 0}
root = ""


def step_validate():
    global root
    value = inputs.get("research_root", "")
    if not isinstance(value, str) or not value or "\0" in value or len(value) > 1024 or not os.path.isabs(value):
        return program.fail("failed", "invalid_input", "research_root must be an authorized absolute directory", "validate")
    root = os.path.realpath(value)
    envelope["inputs"] = {"research_root": root}
    return "collect"


def step_collect():
    envelope["phase"] = "collect"
    signals = envelope["signals"]
    topics = os.path.join(root, "topics")
    if os.path.commonpath((root, os.path.realpath(topics))) != root:
        return program.fail("unavailable", "outside_root", "topics resolves outside the selected root", "collect")
    try:
        with os.scandir(topics) as entries:
            topic_names = sorted(entry.name for entry in itertools.islice(entries, 513))
    except OSError as exc:
        return program.fail("unavailable", "workspace_unavailable", str(exc)[:180], "collect")
    seen = set()
    rows = []
    errors = []
    examined = 0
    if len(topic_names) > 512:
        signals["truncated"] = True
    for topic in topic_names[:512]:
        runs = os.path.join(topics, topic, "runs")
        if os.path.commonpath((root, os.path.realpath(runs))) != root:
            signals["skipped_outside_root"] += 1
            continue
        try:
            with os.scandir(runs) as entries:
                names = sorted(entry.name for entry in itertools.islice(entries, 5001 - examined))
        except FileNotFoundError:
            continue
        except OSError as exc:
            errors.append(str(exc)[:180])
            continue
        for name in names:
            if examined >= 5000:
                signals["truncated"] = True
                break
            examined += 1
            path = os.path.join(runs, name)
            real = os.path.realpath(path)
            if os.path.commonpath((root, real)) != root:
                signals["skipped_outside_root"] += 1
                continue
            try:
                is_directory = stat.S_ISDIR(os.stat(real).st_mode)
            except OSError as exc:
                errors.append(str(exc)[:180])
                continue
            if not is_directory or real in seen:
                continue
            seen.add(real)
            sizes = {}
            for filename in ("README.md", "CHECKPOINT.md", "CURRENT.md"):
                candidate = os.path.join(real, filename)
                if os.path.commonpath((root, os.path.realpath(candidate))) != root:
                    signals["skipped_outside_root"] += 1
                    sizes[filename] = "unknown"
                    continue
                try:
                    info = os.stat(candidate)
                    sizes[filename] = info.st_size if stat.S_ISREG(info.st_mode) else None
                except FileNotFoundError:
                    sizes[filename] = None
                except OSError as exc:
                    sizes[filename] = "unknown"
                    errors.append(str(exc)[:180])
            relative_path = os.path.relpath(real, root)
            if len(relative_path) > 240:
                signals["truncated"] = True
            shown = relative_path[:240]
            if sizes["README.md"] is None:
                signals["without_guide"] += 1
                if len(signals["without_guide_examples"]) < 8:
                    signals["without_guide_examples"].append(shown)
            if sizes["CHECKPOINT.md"] is None:
                signals["without_checkpoint"] += 1
                if len(signals["without_checkpoint_examples"]) < 8:
                    signals["without_checkpoint_examples"].append(shown)
            checkpoint = sizes["CHECKPOINT.md"]
            current = sizes["CURRENT.md"]
            rows.append({"path": shown, "checkpoint_bytes": checkpoint,
                         "current_view_bytes": checkpoint if current is None else current})
    rows.sort(key=lambda row: row["checkpoint_bytes"] if isinstance(row["checkpoint_bytes"], int) else -1, reverse=True)
    signals["largest"] = rows[:8]
    signals["unique_runs"] = len(seen)
    if errors:
        envelope["errors"].append({"class": "incomplete_read", "detail": errors[0], "where": "collect"})
    incomplete = errors or signals["truncated"] or signals["skipped_outside_root"]
    envelope["status"] = "partial" if incomplete else "ok"
    envelope["evidence"].append({"path": root, "scope": "direct topics/*/runs/*; sizes and missing files are review signals"})
    return "report"


def step_report():
    envelope["phase"] = "report"
    print(json.dumps(envelope, allow_nan=False, separators=(",", ":")))
    return None


program.run({"validate": step_validate, "collect": step_collect, "report": step_report}, "validate")
