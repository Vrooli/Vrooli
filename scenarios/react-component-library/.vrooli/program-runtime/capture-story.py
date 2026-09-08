import hashlib
import json
import os
import subprocess
from pathlib import Path
from urllib.parse import quote

try:
    inputs
except NameError:
    inputs = {}
inputs = inputs if isinstance(inputs, dict) else {}

asset_id = str(inputs.get("asset_id", "") or "").strip()
requested_version = str(inputs.get("version", "") or "").strip()
requested_stories = inputs.get("stories", [])
viewports = inputs.get("viewports", ["desktop", "mobile"])
themes = inputs.get("themes", ["light", "dark"])

envelope = {
    "program": "react-component-library.capture-story",
    "version": "1",
    "status": "failed",
    "phase": "validate",
    "inputs": {"asset_id": asset_id, "version": requested_version, "stories": requested_stories, "viewports": viewports, "themes": themes},
    "signals": {"asset_id": asset_id, "version": None, "combinations": [], "baseline_count": 0, "changed_count": 0},
    "errors": [],
    "evidence": [],
}


def fail(kind, detail):
    envelope["errors"].append({"class": kind, "detail": str(detail)[:300]})
    envelope["phase"] = "report"
    return envelope


def repo_root():
    here = Path.cwd().resolve()
    for candidate in (here, *here.parents):
        if (candidate / "scenarios" / "react-component-library" / "catalog").is_dir():
            return candidate
    return Path(os.environ.get("VROOLI_ROOT", here))


def find_component(root):
    for path in (root / "scenarios" / "react-component-library" / "library").rglob("component.json"):
        try:
            doc = json.loads(path.read_text())
        except (OSError, ValueError):
            continue
        if doc.get("catalogId") == asset_id:
            return path, doc
    return None, None


def stable_key(version, story, viewport, theme):
    raw = "\x00".join((asset_id, version, story, viewport, theme))
    return hashlib.sha256(raw.encode()).hexdigest()[:20]


def fingerprint(capture_result):
    resolved = capture_result.get("resolved") or {}
    findings = capture_result.get("findings") or []
    return {
        "status": capture_result.get("status"),
        "route": resolved.get("route", ""),
        "rung": resolved.get("rung", 0),
        "findings": sorted((item.get("rule", ""), item.get("severity", ""), item.get("selector", ""), item.get("message", "")) for item in findings),
        "artifact_count": len(capture_result.get("artifacts") or []),
    }


def run_capture(root, url, viewport, theme):
    command = [
        "ui-health", "capture", "--scenario", "react-component-library", "--surface", url,
        "--viewport", viewport, "--theme", theme, "--fail-on", "error", "--json",
    ]
    try:
        completed = subprocess.run(command, cwd=str(root), text=True, capture_output=True, timeout=110, check=False)
    except subprocess.TimeoutExpired as exc:
        return {"status": "failed", "error": "capture timed out: " + str(exc)}
    try:
        result = json.loads(completed.stdout)
    except (TypeError, ValueError):
        return {"status": "failed", "error": (completed.stderr or completed.stdout or "ui-health returned no JSON")[-500:]}
    if completed.returncode != 0 and result.get("status") not in ("ok", "partial"):
        result["error"] = result.get("error") or completed.stderr[-500:]
    return result


if not asset_id:
    fail("invalid_input", "asset_id is required")
else:
    root = repo_root()
    component_path, component = find_component(root)
    if component is None:
        fail("asset_not_found", "no component manifest declares catalog id " + asset_id)
    else:
        version = requested_version or str(component.get("latest", "")).strip()
        version_path = component_path.parent / "versions" / version
        story_path = version_path / "story.json"
        if not version or not story_path.is_file():
            fail("story_not_found", "no story contract for " + asset_id + "@" + version)
        else:
            try:
                story_doc = json.loads(story_path.read_text())
                available = [str(item.get("id", "")).strip() for item in story_doc.get("stories", []) if item.get("id")]
            except (OSError, ValueError) as exc:
                fail("story_not_found", exc)
                available = []
            if requested_stories:
                stories = [str(item).strip() for item in requested_stories if str(item).strip() in available]
                missing = [str(item) for item in requested_stories if str(item).strip() not in available]
                if missing:
                    fail("story_not_found", "unknown story ids: " + ", ".join(missing))
                    stories = []
            else:
                stories = available
            if stories:
                envelope["phase"] = "collect"
                envelope["signals"]["version"] = version
                baseline_dir = root / "scenarios" / "react-component-library" / "captures" / "story-baselines"
                baseline_dir.mkdir(parents=True, exist_ok=True)
                for story in stories:
                    preview = "/assets/" + quote(asset_id, safe="") + "/preview?story=" + quote(story, safe="") + "&version=" + quote(version, safe="")
                    for viewport in viewports if isinstance(viewports, list) else []:
                        for theme in themes if isinstance(themes, list) else []:
                            viewport = str(viewport).strip().lower()
                            theme = str(theme).strip().lower()
                            item = {"story": story, "viewport": viewport, "theme": theme, "url": preview}
                            capture_result = run_capture(root, "http://localhost:23906" + preview, viewport, theme)
                            item["capture"] = capture_result
                            item["changed"] = False
                            item["baseline"] = False
                            key = stable_key(version, story, viewport, theme)
                            baseline_path = baseline_dir / (key + ".json")
                            current_fingerprint = fingerprint(capture_result)
                            if capture_result.get("status") == "ok" and not baseline_path.exists():
                                baseline_path.write_text(json.dumps({"key": key, "fingerprint": current_fingerprint, "captured": capture_result}, indent=2) + "\n")
                                item["baseline"] = True
                                envelope["signals"]["baseline_count"] += 1
                            elif baseline_path.exists():
                                try:
                                    baseline = json.loads(baseline_path.read_text()).get("fingerprint", {})
                                    item["changed"] = baseline != current_fingerprint
                                    envelope["signals"]["changed_count"] += int(item["changed"])
                                except (OSError, ValueError):
                                    item["changed"] = True
                            envelope["signals"]["combinations"].append(item)
                envelope["phase"] = "decide"
                statuses = [item.get("capture", {}).get("status") for item in envelope["signals"]["combinations"]]
                envelope["status"] = "ok" if statuses and all(status == "ok" for status in statuses) else "partial"
                envelope["evidence"].append("ui-health capture per story x viewport x theme")
                envelope["phase"] = "report"

print(json.dumps(envelope, allow_nan=False))
