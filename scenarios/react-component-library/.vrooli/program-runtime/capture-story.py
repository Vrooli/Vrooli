import hashlib
import html
import json
import re
from pathlib import Path
from urllib.parse import quote, urlencode

inputs = program.inputs()
inputs = inputs if isinstance(inputs, dict) else {}

asset_id = str(inputs.get("asset_id", "") or "").strip()
requested_version = str(inputs.get("version", "") or "").strip()
requested_stories = inputs.get("stories", [])
viewports = inputs.get("viewports", ["desktop", "mobile"])
themes = inputs.get("themes", ["light", "dark"])

envelope = {
    "program": "react-component-library.capture-story",
    "version": "2",
    "status": "failed",
    "phase": "validate",
    "inputs": {"asset_id": asset_id, "version": requested_version, "stories": requested_stories, "viewports": viewports, "themes": themes},
    "signals": {"asset_id": asset_id, "version": None, "target": "isolated_story_harness", "combinations": [], "baseline_count": 0, "changed_count": 0},
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
    return here


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
    # Namespace corrected isolated-story artifacts away from legacy full-sheet
    # baselines; old images are never adopted as single-story evidence.
    raw = "\x00".join(("isolated-story-v1", asset_id, version, story, viewport, theme))
    return hashlib.sha256(raw.encode()).hexdigest()[:20]


def fingerprint(capture_result):
    findings = capture_result.get("findings") or []
    return {
        "status": capture_result.get("status"),
        "target": capture_result.get("target", ""),
        "observed_story": capture_result.get("observed_story", ""),
        "expected_story": capture_result.get("expected_story", ""),
        "story_result_passed": capture_result.get("story_result_passed", False),
        "readiness": (capture_result.get("readiness") or {}).get("outcome", ""),
        "findings": sorted((item.get("rule", ""), item.get("severity", ""), item.get("selector", ""), item.get("message", "")) for item in findings),
        "artifact_count": len(capture_result.get("artifacts") or []),
    }


def valid_viewport(viewport):
    value = str(viewport).strip().lower()
    if value in ("desktop", "tablet", "mobile"):
        return True
    match = re.fullmatch(r"([1-9][0-9]*)x([1-9][0-9]*)", value)
    if not match:
        return False
    width, height = (int(part) for part in match.groups())
    return 100 <= width <= 4000 and 100 <= height <= 4000


def dimensions(viewport):
    if viewport == "desktop":
        return {"width": 1440, "height": 900}
    if viewport == "mobile":
        return {"width": 390, "height": 844}
    if viewport == "tablet":
        return {"width": 768, "height": 1024}
    width, height = viewport.split("x", 1)
    return {"width": int(width), "height": int(height)}


def resolve_rcl_api_base():
    # Use the control-plane current owner projection instead of a fixed port.
    rows = vrooli.scenario.status(name="react-component-library").materialize(1)
    if not rows:
        raise RuntimeError("react-component-library status returned no row")
    row = rows[0] if isinstance(rows[0], dict) else {}
    scenario = row.get("scenario") if isinstance(row.get("scenario"), dict) else row
    ports = scenario.get("ports") if isinstance(scenario.get("ports"), dict) else {}
    raw_port = ports.get("API_PORT")
    try:
        port = int(raw_port)
    except (TypeError, ValueError):
        raise RuntimeError("react-component-library API_PORT is unavailable")
    if port < 1 or port > 65535:
        raise RuntimeError("react-component-library API_PORT is outside the valid range")
    return "http://127.0.0.1:" + str(port)


def isolated_story_url(api_base, library_id, version, story, theme):
    path = "/preview/" + quote(library_id, safe="") + "/harness.html"
    query = urlencode({"version": version, "story": story, "runner": "1", "motion": "reduce", "theme": theme})
    return api_base.rstrip("/") + path + "?" + query


def artifact_summary(row):
    if not isinstance(row, dict):
        return None
    summary = {}
    for key in ("type", "captureType", "reference", "sizeBytes", "size_bytes", "metadata", "primary"):
        if key in row:
            summary[key] = row[key]
    return summary or None


def run_capture(url, viewport, theme, expected_story):
    # Match RCL's first-party BASCaptureExecutor: capture one isolated harness,
    # wait for its passed story result, and screenshot its component boundary.
    readiness_selector = '[data-preview-readiness-marker][data-preview-ready="true"][data-rcl-story-status="passed"]'
    try:
        kwargs = {
            "url": url,
            "capture": ["CAPTURE_TYPE_SCREENSHOT", "CAPTURE_TYPE_DOM", "CAPTURE_TYPE_DOM_TREE"],
            "wait_for": {"selector": readiness_selector},
            "screenshot_selector": "[data-preview-sheet]",
            "dimensions": {**dimensions(viewport), "device_scale_factor": 1.0},
            "inline_dom": True,
            "inline_dom_tree": True,
            "direction": "ltr",
            "browser_profile": {"fingerprint": {"colorScheme": theme}},
        }
        result = browser_automation_studio.capture.capture(**kwargs)
        rows = result.head(16)
        meta = result.meta()
    except Exception as exc:
        return {"status": "failed", "error": str(exc)[:500], "readiness_selector": readiness_selector}
    by_type = {}
    for row in rows:
        kind = str(row.get("type") or row.get("captureType") or "").lower()
        by_type[kind] = row
    screenshot = by_type.get("capture_type_screenshot") or by_type.get("screenshot")
    dom = by_type.get("capture_type_dom") or by_type.get("dom")
    snapshot = by_type.get("capture_type_dom_tree") or by_type.get("dom_tree")
    execution_id = meta.get("executionId") or meta.get("execution_id")
    readiness = meta.get("readiness") if isinstance(meta.get("readiness"), dict) else {}
    # CaptureArtifact rows carry metadata/reference only. Inline CaptureResponse
    # fields are exposed by Handle.meta(), using protojson's camelCase names.
    dom_html = meta.get("domHtml") or meta.get("dom_html") or ""
    story_match = re.search(r'<meta\s+name=["\']story-id["\']\s+content=["\']([^"\']+)', str(dom_html), re.IGNORECASE)
    observed_story = html.unescape(story_match.group(1)) if story_match else ""
    result_match = re.search(r'<pre[^>]*id=["\']rcl-story-result["\'][^>]*>(.*?)</pre>', str(dom_html), re.IGNORECASE | re.DOTALL)
    story_result = None
    if result_match:
        try:
            story_result = json.loads(html.unescape(result_match.group(1)))
        except (TypeError, ValueError):
            story_result = None
    ready = str(readiness.get("outcome", "")).lower() == "ready"
    passed = isinstance(story_result, dict) and story_result.get("passed") is True and not story_result.get("failures")
    identity_matches = observed_story == expected_story
    artifacts = [item for item in (artifact_summary(screenshot), artifact_summary(dom), artifact_summary(snapshot)) if item]
    if not ready:
        status = "partial"
    elif not identity_matches or not passed:
        status = "failed"
    elif screenshot and dom and snapshot:
        status = "ok"
    else:
        status = "partial"
    return {
        "status": status,
        "target": "isolated_story_harness",
        "requested_url": url,
        "observed_story": observed_story or None,
        "expected_story": expected_story,
        "story_result_passed": bool(passed),
        "readiness": readiness,
        "readiness_selector": readiness_selector,
        "screenshot_selector": "[data-preview-sheet]",
        "artifacts": artifacts,
        "execution_id": execution_id,
        "findings": [],
        "errors": [] if identity_matches and passed else [{"class": "story_identity_mismatch" if not identity_matches else "story_not_passed", "detail": "captured harness DOM did not prove the requested passing story"}],
    }


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
            if not isinstance(viewports, list) or not viewports or not all(valid_viewport(value) for value in viewports):
                fail("invalid_input", "viewports must contain desktop, tablet, mobile, or WIDTHxHEIGHT values")
                stories = []
            if not isinstance(themes, list) or not themes or not all(str(value).strip().lower() in ("light", "dark") for value in themes):
                fail("invalid_input", "themes must contain light or dark")
                stories = []
            if stories:
                try:
                    api_base = resolve_rcl_api_base()
                except Exception as exc:
                    fail("scenario_unreachable", exc)
                    stories = []
            if stories:
                library_id = str(component.get("libraryId", "")).strip()
                if not library_id:
                    fail("invalid_component", "component manifest has no libraryId for the isolated harness")
                else:
                    envelope["phase"] = "collect"
                    envelope["signals"]["version"] = version
                    envelope["signals"]["target"] = "isolated_story_harness"
                    baseline_dir = root / "scenarios" / "react-component-library" / "captures" / "story-baselines"
                    baseline_dir.mkdir(parents=True, exist_ok=True)
                    for story in stories:
                        for viewport in viewports if isinstance(viewports, list) else []:
                            for theme in themes if isinstance(themes, list) else []:
                                viewport = str(viewport).strip().lower()
                                theme = str(theme).strip().lower()
                                capture_url = isolated_story_url(api_base, library_id, version, story, theme)
                                item = {"story": story, "viewport": viewport, "theme": theme, "requested_url": capture_url}
                                capture_result = run_capture(capture_url, viewport, theme, story)
                                item["capture"] = capture_result
                                for capture_error in capture_result.get("errors", []):
                                    envelope["errors"].append(capture_error)
                                if capture_result.get("status") != "ok" and not capture_result.get("errors"):
                                    envelope["errors"].append({
                                        "class": "capture_failed",
                                        "detail": str(capture_result.get("error") or "capture did not prove a ready requested story")[:300],
                                    })
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
                    if statuses and all(status == "ok" for status in statuses):
                        envelope["status"] = "ok"
                    elif any(status == "failed" for status in statuses):
                        envelope["status"] = "failed"
                    elif statuses and all(status == "unavailable" for status in statuses):
                        envelope["status"] = "unavailable"
                    else:
                        envelope["status"] = "partial"
                    envelope["evidence"].append("BAS isolated story harness per story x viewport x theme")
                    envelope["phase"] = "report"

print(json.dumps(envelope, allow_nan=False))
