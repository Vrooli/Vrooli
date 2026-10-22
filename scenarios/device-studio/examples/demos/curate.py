#!/usr/bin/env python3
"""Verify, inspect, or create the six durable Device Studio demo projects."""
import argparse
import copy
import hashlib
import json
import math
import os
from pathlib import Path
import sys
import time
import urllib.error
import urllib.request
import uuid

ROOT = Path(__file__).resolve().parent
REPO_ROOT = ROOT.parents[3]
MANIFEST = json.loads((ROOT / "manifest.json").read_text())
STATE_PATH = ROOT / "run-state.json"
API = "http://127.0.0.1:19349"

# These are stable public starter-to-review-route associations. Revisions and
# digests are always resolved from the live public owner below; never pin a
# catalog head from a historical run-state when preparing an append.
FAMILY_REVISION2 = {
    "ipad-screen-uv-oriented-180": ("ipad-portrait-cupfree-fit-v2", "tablet_app"),
    "generic-book-fold": ("generic-book-fold-still-v3-left-screen-vflip-static", "phone_frame0"),
    "generic-round-watch-v2": ("generic-round-watch-body-floating-still-v4", "phone_frame0"),
    "monitor-screen-mapped-v2": ("innerscene-monitor-study-hero-screenmap-v2", "monitor_app"),
    "generic-television-concept-v3": ("generic-television-thin-bezel-still-v3", "laptop_app"),
    "generic-smart-display-concept-v4": ("generic-smart-display-tabletop-still-v7", "laptop_app"),
    "generic-digital-frame-concept-v4-glassless": ("innerscene-digital-frame-tabletop-still-v4-glassless", "laptop_app"),
    "generic-digital-frame-wall-v4-glassless": ("innerscene-digital-frame-wall-still-v5-mounted", "laptop_app"),
}

WATCH_SYNTHETIC_SOURCE = "watch_synthetic_calibration"
WATCH_SYNTHETIC_SCENE = {
    "id": "generic-round-watch-body-floating-study-v4",
    "prior_revision": 1,
    "prior_sha256": "ad1d1c0eab51eca518c1473faab44e707a37e53c4abad6626a24fb82991b7e60",
    "revision": 2,
    "sha256": "24adf72a2ca5e6ea879d51a213055d258e48f41421e70e28c9a7ee87d258a60b",
    "camera_id": "generic-round-watch-body-floating-study-v4-camera-v1",
    "f_number": 16,
    "focus_distance_m": 0.16099745801147922,
}
WATCH_SYNTHETIC_PRESET = {
    "id": "generic-round-watch-body-floating-still-v4",
    "prior_revision": 2,
    "prior_sha256": "6e1b1ebb066b6e7e359808a183f83f36f77c98a9d8eb42edceb878f33cd2ef8e",
    "revision": 3,
    "sha256": "3be114c5bb76e6bf19eb686ce2b4d9139b4ee1843780398d91a2c4db3206928c",
}
WATCH_SYNTHETIC_SHOT = {
    "id": "18467919-aa4f-4a55-92bf-e3bf9a202d1a",
    "name": "Candidate 03 — generic round watch · synthetic 12:48 dial calibration",
    "prior_revision": 3,
    "prior_recipe_sha256": "8ca67729955776b0c383d0c90c40d5c0d81903edc7cf9b3ae90b4141821fe578",
    "revision": 4,
    "recipe_sha256": "9c6c33a2fc4510b983d91e9e82fd48363ba6afad8ff8a0891e345b9950512931",
    "preset_sha256": "3be114c5bb76e6bf19eb686ce2b4d9139b4ee1843780398d91a2c4db3206928c",
    "source_sha256": "68e83ddbcd27a7f74dfd659018c79f7c1b4cf1d31e86f1bf703c2c9203bf97f4",
}


class Stop(RuntimeError):
    pass


def sha(path):
    h = hashlib.sha256()
    with path.open("rb") as f:
        for block in iter(lambda: f.read(1024 * 1024), b""):
            h.update(block)
    return h.hexdigest()


def verify_files():
    checked = 0
    for key, item in MANIFEST["assets"].items():
        path = ROOT / item["path"]
        got = sha(path) if path.is_file() else "missing"
        if got != item["sha256"]:
            raise Stop(f"asset {key} hash mismatch: expected {item['sha256']}, got {got}")
        media_type = item.get("media_type")
        signature = path.read_bytes()[:12] if path.is_file() else b""
        valid = (media_type == "image/png" and path.suffix.lower() == ".png" and signature.startswith(b"\x89PNG\r\n\x1a\n")) or (media_type == "video/mp4" and path.suffix.lower() == ".mp4" and signature[4:8] == b"ftyp")
        if not valid:
            raise Stop(f"asset {key} media type is not backed by its pinned PNG/MP4 signature")
        checked += 1
    for item in MANIFEST.get("provenance_files", []):
        path = ROOT / item["path"]
        got = sha(path) if path.is_file() else "missing"
        if got != item["sha256"]:
            raise Stop(f"provenance hash mismatch at {path}: {got}")
        checked += 1
    optional_refs = list(MANIFEST.get("archived_reference_outputs", {}).values()) + MANIFEST.get("archived_source_manifests", [])
    absent = []
    for item in optional_refs:
        path = REPO_ROOT / item["path"]
        if not path.is_file():
            absent.append(item["path"])
            continue
        got = sha(path)
        if got != item["sha256"]:
            raise Stop(f"archived reference hash mismatch at {path}: {got}")
        checked += 1
    if absent:
        print("archived references not present (no output claim): " + ", ".join(absent))
    for example in MANIFEST["examples"]:
        if example.get("motion_input"):
            path = ROOT / example["motion_input"]
            got = sha(path) if path.is_file() else "missing"
            if got != example["motion_input_sha256"]:
                raise Stop(f"motion input {path.name} hash mismatch: {got}")
            checked += 1
    print(f"verified {checked} source, motion, provenance, and present archived-reference hashes")


def request(path, body=None, content_type="application/json"):
    if body is None:
        raw = None
    elif isinstance(body, bytes):
        raw = body
    else:
        raw = json.dumps(body, separators=(",", ":")).encode()
    req = urllib.request.Request(API + path, data=raw, headers={"Content-Type": content_type}, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=20) as response:
            return json.loads(response.read())
    except urllib.error.HTTPError as e:
        detail = e.read().decode("utf-8", "replace")[:800]
        raise Stop(f"POST {path} returned HTTP {e.code}: {detail}") from e
    except (urllib.error.URLError, TimeoutError) as e:
        raise Stop(f"Device Studio API unavailable at {API}: {e}") from e


def rpc(service, path_service, method, body):
    return request(f"/vrooli.device_studio.v1.{service}.{path_service}/{method}", body)


def get_health():
    try:
        with urllib.request.urlopen(API + "/health", timeout=4) as response:
            health = json.loads(response.read())
    except Exception as e:
        raise Stop(f"Device Studio must already be live at {API}; this helper never starts it ({e})") from e
    if not health.get("readiness"):
        raise Stop(f"Device Studio is not ready: {health}")
    return health


def pages(service, path_service, method, key, body, cursor_key, cursor_value=""):
    rows = []
    for _ in range(20):
        request_body = dict(body)
        if cursor_value:
            request_body[cursor_key] = cursor_value
        page = rpc(service, path_service, method, request_body)
        rows.extend(page.get(key, []))
        cursor_value = page.get("nextPageToken") or page.get("nextAfterPresetId") or page.get("nextAfterDeviceId") or page.get("nextAfterSceneId") or page.get("nextAfterMotionId") or ""
        if not cursor_value:
            return rows
    raise Stop(f"{method} exceeded the bounded 20-page read")


def project_rows():
    return pages("projects", "ProjectsService", "ListProjects", "projects", {"pageSize": 100}, "pageToken")


def catalog_rows(domain, rpc_name, key, cursor):
    response = rpc(domain, f"{domain.title()}Service", rpc_name, {"pageSize": 100})
    rows = list(response.get(key, []))
    after = {"nextAfterDeviceId":"afterDeviceId","nextAfterSceneId":"afterSceneId","nextAfterMotionId":"afterMotionId"}[cursor]
    for _ in range(20):
        if not response.get(cursor):
            return rows
        response = rpc(domain, f"{domain.title()}Service", rpc_name, {"pageSize": 100, after: response[cursor]})
        rows.extend(response.get(key, []))
    raise Stop(f"{rpc_name} exceeded the bounded 20-page read")


def plan():
    health = get_health()
    projects = project_rows()
    presets = pages("shots", "ShotsService", "ListPhysicalShotPresets", "presets", {"pageSize":100}, "afterPresetId")
    catalogs = {
        "devices": catalog_rows("devices", "ListDeviceCandidates", "candidates", "nextAfterDeviceId"),
        "scenes": catalog_rows("scenes", "ListSceneCandidates", "candidates", "nextAfterSceneId"),
        "motions": catalog_rows("motions", "ListMotionCandidates", "candidates", "nextAfterMotionId"),
    }
    result = {"health": health.get("version"), "projects": len(projects), "existingDemoNames": [p["name"] for p in projects if p["name"].startswith(MANIFEST["prefix"])], "physicalPresets": [{"id": p.get("presetId"), "revision": p.get("revision"), "sha256": p.get("sha256"), "candidateOnly": p.get("candidateOnly")} for p in presets], "catalogs": {k: [{"id": r.get({"devices":"deviceId","scenes":"sceneId","motions":"motionId"}[k]), "revision": r.get("revision"), "sha256": r.get("sha256"), "state": r.get("state")} for r in v] for k, v in catalogs.items()}}
    print(json.dumps(result, indent=2))


def read_state():
    if not STATE_PATH.exists():
        return {"schema": "device-studio-demo-run-state/v1", "projects": {}, "media": {}, "shots": {}, "presets": {}, "familyCandidates": {}, "inspections": {}}
    state = json.loads(STATE_PATH.read_text())
    state.setdefault("familyCandidates", {})
    return state


def write_state(state):
    temp = STATE_PATH.with_suffix(".json.tmp")
    temp.write_text(json.dumps(state, indent=2) + "\n")
    os.replace(temp, STATE_PATH)


def find_or_create_project(example, state, all_projects):
    name = example["project"]
    matches = [p for p in all_projects if p["name"] == name]
    pinned = state["projects"].get(example["id"])
    if len(matches) > 1:
        raise Stop(f"duplicate exact project name {name!r}; preserving all and refusing to choose")
    if matches:
        if not pinned or matches[0]["id"] != pinned:
            raise Stop(f"project {name!r} exists without its matching run-state ID; refusing to adopt it")
        return matches[0]["id"]
    if pinned:
        raise Stop(f"run-state project {pinned} for {name!r} is missing; preserving checkpoint and stopping")
    response = rpc("projects", "ProjectsService", "CreateProject", {"name": name})
    project = response.get("project") or {}
    if not project.get("id") or project.get("name") != name:
        raise Stop(f"CreateProject returned unexpected identity for {name!r}")
    state["projects"][example["id"]] = project["id"]
    write_state(state)
    all_projects.append(project)
    return project["id"]


def list_media(project_id):
    return pages("projects", "ProjectsService", "ListSourceMedia", "sourceMedia", {"projectId": project_id, "pageSize": 100}, "pageToken")


def multipart_upload(project_id, path, media_type):
    boundary = "ds-demo-" + uuid.uuid4().hex
    content = path.read_bytes()
    filename = path.name.replace('"', "")
    payload = (f"--{boundary}\r\nContent-Disposition: form-data; name=\"file\"; filename=\"{filename}\"\r\nContent-Type: {media_type}\r\n\r\n").encode() + content + f"\r\n--{boundary}--\r\n".encode()
    return request(f"/api/v1/projects/{project_id}/media", payload, f"multipart/form-data; boundary={boundary}")


def ensure_media(project_id, example_id, asset_id, state):
    entry = MANIFEST["assets"][asset_id]
    path = ROOT / entry["path"]
    if sha(path) != entry["sha256"]:
        raise Stop(f"source changed before upload: {asset_id}")
    state_key = f"{example_id}:{asset_id}"
    pinned = state["media"].get(state_key)
    rows = list_media(project_id)
    digest_rows = [m for m in rows if m.get("sha256") == entry["sha256"]]
    if pinned:
        row = next((m for m in digest_rows if m.get("id") == pinned), None)
        if not row:
            raise Stop(f"pinned media {pinned} is missing or no longer matches {asset_id}")
        return row
    if len(digest_rows) > 1:
        raise Stop(f"multiple exact source copies for {asset_id} in project {project_id}; refusing to guess")
    if digest_rows:
        state["media"][state_key] = digest_rows[0]["id"]
        write_state(state)
        return digest_rows[0]
    name_conflict = [m for m in rows if m.get("originalFilename") == path.name]
    if name_conflict:
        raise Stop(f"same filename with different bytes already exists in project {project_id}: {path.name}")
    response = multipart_upload(project_id, path, entry["media_type"])
    media = response.get("source_media") or {}
    if media.get("sha256") != entry["sha256"] or not media.get("id"):
        raise Stop(f"media import did not return the pinned source identity for {asset_id}")
    state["media"][state_key] = media["id"]
    write_state(state)
    return media


def create_shot(project_id, example_id, name, snapshot, state):
    key = f"{example_id}:{name}"
    rows = pages("shots", "ShotsService", "ListShots", "shots", {"projectId": project_id, "pageSize": 100}, "pageToken")
    matches = [s for s in rows if s.get("name") == name]
    pinned = state["shots"].get(key)
    if len(matches) > 1:
        raise Stop(f"duplicate shot {name!r} in project {project_id}")
    if matches:
        if not pinned or matches[0].get("id") != pinned.get("id"):
            raise Stop(f"shot {name!r} exists without matching run-state ID; refusing to adopt it")
        shot_id = pinned["id"]
    else:
        if pinned:
            raise Stop(f"pinned shot {pinned} for {name!r} is missing")
        response = rpc("shots", "ShotsService", "CreateShot", {"projectId": project_id, "name": name, "snapshot": snapshot})
        shot = response.get("shot") or {}
        revision = response.get("revision") or {}
        if not shot.get("id") or revision.get("revision") != "1":
            raise Stop(f"CreateShot returned unexpected identity for {name!r}")
        shot_id = shot["id"]
        state["shots"][key] = {"id": shot_id, "revision": 1}
        write_state(state)
    opened = rpc("shots", "ShotsService", "GetShot", {"id": shot_id})
    head = opened.get("head") or {}
    if int(head.get("revision", 0)) != 1:
        raise Stop(f"shot {shot_id} head is not revision 1")
    revision = rpc("shots", "ShotsService", "GetShotRevision", {"shotId": shot_id, "revision": 1}).get("revision") or {}
    saved_snapshot = revision.get("snapshot") or {}
    if not saved_snapshot:
        raise Stop(f"shot {shot_id} reopened without an editable snapshot")
    expected_screen = (snapshot.get("presentation") or {}).get("sourceMediaId")
    if (saved_snapshot.get("presentation") or {}).get("sourceMediaId") != expected_screen:
        raise Stop(f"shot {shot_id} reopened with a different screen source binding")
    expected_scene = snapshot.get("scene") or {}
    saved_scene = saved_snapshot.get("scene") or {}
    for field in ("plateMediaId", "activeMaskMediaId", "hardwareMaskMediaId"):
        if expected_scene.get(field) and saved_scene.get(field) != expected_scene[field]:
            raise Stop(f"shot {shot_id} reopened with a different {field} binding")
    expected_video = snapshot.get("videoTiming") or {}
    saved_video = saved_snapshot.get("videoTiming") or {}
    for field in ("sourceMediaId", "sourceSha256", "inspectionReceiptId", "inspectionReceiptSha256", "clockSha256"):
        if expected_video.get(field) and saved_video.get(field) != expected_video[field]:
            raise Stop(f"shot {shot_id} reopened with a different video timing binding: {field}")
    for field in ("trimStartPts", "trimEndPts"):
        if field in expected_video and integer(saved_video.get(field)) != integer(expected_video[field]):
            raise Stop(f"shot {shot_id} reopened with a different video timing binding: {field}")
    expected_physical = snapshot.get("physicalBinding") or {}
    saved_physical = saved_snapshot.get("physicalBinding") or {}
    for label in ("device", "scene", "motion"):
        if expected_physical.get(label) and saved_physical.get(label) != expected_physical[label]:
            raise Stop(f"shot {shot_id} reopened with a different exact {label} candidate binding")
    compiled = rpc("shots", "ShotsService", "CompileShot", {"shotId": shot_id, "revision": 1}).get("compiled") or {}
    if not compiled.get("renderEligible"):
        issues = compiled.get("issues", [])
        raise Stop(f"shot {shot_id} failed compilation: {json.dumps(issues)[:1200]}")
    resolved_fields = (("screen", "sourceMediaId"), ("plate", "plateMediaId"), ("activeMask", "activeMaskMediaId"), ("hardwareMask", "hardwareMaskMediaId"))
    resolved = []
    for field, source_field in resolved_fields:
        source_id = expected_screen if field == "screen" else expected_scene.get(source_field)
        item = compiled.get(field)
        if source_id and (not item or item.get("id") != source_id or not item.get("sha256")):
            raise Stop(f"shot {shot_id} compiled with a different {field} source identity")
        if item:
            resolved.append(item)
    state["shots"][key] = {"id": shot_id, "revision": 1, "compiled": True, "recipeSha256": compiled.get("recipeSha256"), "sourceMediaSha256": [m.get("sha256") for m in resolved]}
    write_state(state)
    return shot_id


def fixed_laptop_snapshot(media):
    return {"scene":{"plateMediaId":media["laptop_plate"],"activeMaskMediaId":media["laptop_mask"],"displayWidthPixels":1920,"displayHeightPixels":1200,"plateQuad":{"topLeft":{"x":540,"y":54},"topRight":{"x":1407,"y":53},"bottomRight":{"x":1377,"y":620},"bottomLeft":{"x":491,"y":560}},"coordinateConvention":"COORDINATE_CONVENTION_PLATE_RASTER_EDGE_PIXELS_V1","depictionLabel":"Retained generated photographic laptop plate"},"presentation":{"sourceMediaId":media["laptop_app"],"captureArea":"CAPTURE_AREA_APP_ONLY","kind":"PRESENTATION_KIND_BROWSER_DESKTOP","fitMode":"SCREEN_FIT_MODE_CONTAIN","browserDesktopChrome":{"windowTitle":"Meridian Studio","addressText":"studio.example.test","showWindowControls":True,"showToolbar":True,"appearance":"PRESENTATION_APPEARANCE_LIGHT","window":{"x":800,"y":800,"width":8400,"height":8000},"showMenuBar":True,"showDock":True}},"motionMode":"MOTION_MODE_FIXED"}


def fixed_phone_snapshot(media):
    return {"scene":{"plateMediaId":media["phone_plate"],"activeMaskMediaId":media["phone_active_mask"],"hardwareMaskMediaId":media["phone_hardware_mask"],"displayWidthPixels":1206,"displayHeightPixels":2622,"plateQuad":{"topLeft":{"x":451.729248046875,"y":129.1783905029297},"topRight":{"x":828.1875,"y":59.907470703125},"bottomRight":{"x":1026.576416015625,"y":825.94873046875},"bottomLeft":{"x":611.436279296875,"y":905.3809814453125}},"coordinateConvention":"COORDINATE_CONVENTION_PLATE_RASTER_EDGE_PIXELS_V1","depictionLabel":"Approved photographic phone/table scene with authentic app-only frame"},"presentation":{"sourceMediaId":media["phone_frame0"],"captureArea":"CAPTURE_AREA_APP_ONLY","kind":"PRESENTATION_KIND_PHONE_SYSTEM","fitMode":"SCREEN_FIT_MODE_CONTAIN","phoneSystemChrome":{"clockText":"9:41","batteryPercent":84,"showCellularIndicator":True,"showWifiIndicator":True,"showHomeIndicator":True,"showStatus":True,"showBatteryPercent":False,"appearance":"PRESENTATION_APPEARANCE_AUTO"}},"motionMode":"MOTION_MODE_FIXED"}


def inspect_video(project_id, media_id, example_id, state):
    media = rpc("projects", "ProjectsService", "GetSourceMedia", {"projectId":project_id,"id":media_id}).get("sourceMedia") or {}
    if media.get("readiness", "").endswith("READY") and media.get("inspectionReceipt"):
        return media, media["inspectionReceipt"]
    if media.get("readiness", "").endswith("REJECTED"):
        raise Stop(f"video inspection rejected the source: {media.get('inspectionIssueCode')} {media.get('inspectionIssueMessage')}")
    if not media.get("readiness", "").endswith("PENDING"):
        ensured = rpc("projects", "ProjectsService", "EnsureSourceMediaInspection", {"projectId":project_id,"mediaId":media_id})
        media = ensured.get("sourceMedia") or media
    for _ in range(30):
        if media.get("readiness", "").endswith("READY") and media.get("inspectionReceipt"):
            return media, media["inspectionReceipt"]
        if media.get("readiness", "").endswith("REJECTED"):
            raise Stop(f"video inspection rejected the source: {media.get('inspectionIssueCode')} {media.get('inspectionIssueMessage')}")
        time.sleep(1)
        media = rpc("projects", "ProjectsService", "GetSourceMedia", {"projectId":project_id,"id":media_id}).get("sourceMedia") or {}
    job_id = media.get("inspectionJobId") or (media.get("inspectionJob") or {}).get("id")
    state["inspections"][example_id] = {"mediaId":media_id,"jobId":job_id,"status":media.get("readiness")}
    write_state(state)
    raise Stop(f"video inspection remains {media.get('readiness')} after bounded 30-second owner readback; rerun seed later")


def integer(value):
    return int(value or 0)


def video_entry_snapshot(project_id, media, receipt, plate_id, active_id, hardware_id):
    clock_sha = receipt.get("clockSha256")
    points, token = [], ""
    for _ in range(6):
        page = rpc("projects", "ProjectsService", "GetSourceMediaClockPage", {"projectId":project_id,"mediaId":media["id"],"expectedClockSha256":clock_sha,"pageSize":100,"pageToken":token})
        if page.get("clockSha256") != clock_sha:
            raise Stop("fresh source-clock response digest differs from its inspection receipt")
        points.extend(page.get("points", []))
        token = page.get("nextPageToken", "")
        if not token:
            break
    if token or not points:
        raise Stop("source clock exceeded the bounded six-page read or returned no points")
    time_num, time_den = integer(receipt.get("timeBaseNumerator")), integer(receipt.get("timeBaseDenominator"))
    start = integer(receipt.get("streamStartPts"))
    if time_num <= 0 or time_den <= 0:
        raise Stop("fresh inspection has no valid source time base")
    trim_end = start + (10 * time_den) // time_num
    if (10 * time_den) % time_num:
        raise Stop("ten seconds does not map to an exact source PTS tick")
    if not any(integer(p.get("pts")) == trim_end for p in points):
        raise Stop(f"fresh source clock has no frame boundary at the exact ten-second trim PTS {trim_end}")
    video_timing = {"sourceMediaId":media["id"],"sourceSha256":media["sha256"],"sourceSizeBytes":str(media["sizeBytes"]),"inspectionJobId":receipt["jobId"],"inspectionAttemptId":receipt["attemptId"],"inspectionReceiptId":receipt["id"],"inspectionReceiptSha256":receipt["sha256"],"clockArtifactId":receipt["clockArtifactRef"],"clockSha256":clock_sha,"clockSizeBytes":str(receipt["clockArtifactSizeBytes"]),"selectedVideoStream":integer(receipt.get("videoStreamIndex")),"timeBaseNum":str(time_num),"timeBaseDen":str(time_den),"streamStartPts":str(start),"sourceEndPts":str(receipt["sourceEndPts"]),"lastFrameDurationTicks":str(receipt["lastFrameDurationTicks"]),"trimStartPts":str(start),"trimEndPts":str(trim_end),"playbackRateNum":1,"playbackRateDen":1,"outputRateNum":24,"outputRateDen":1,"eofPolicy":"VIDEO_EOF_POLICY_REJECT","videoCompositionProfileId":"device-studio-screen-entry-video-v1","videoCompositionProfileVersion":1}
    return {"scene":{"plateMediaId":plate_id,"activeMaskMediaId":active_id,"hardwareMaskMediaId":hardware_id,"displayWidthPixels":1206,"displayHeightPixels":2622,"plateQuad":{"topLeft":{"x":451.729248046875,"y":129.1783905029297},"topRight":{"x":828.1875,"y":59.907470703125},"bottomRight":{"x":1026.576416015625,"y":825.94873046875},"bottomLeft":{"x":611.436279296875,"y":905.3809814453125}},"coordinateConvention":"COORDINATE_CONVENTION_PLATE_RASTER_EDGE_PIXELS_V1","depictionLabel":"Approved photographic phone/table scene; source-continuous camera entry"},"presentation":{"sourceMediaId":media["id"],"captureArea":"CAPTURE_AREA_APP_ONLY","kind":"PRESENTATION_KIND_PHONE_SYSTEM","fitMode":"SCREEN_FIT_MODE_CONTAIN","phoneSystemChrome":{"clockText":"9:41","batteryPercent":84,"showCellularIndicator":True,"showWifiIndicator":True,"showHomeIndicator":True,"showStatus":True,"showBatteryPercent":False,"appearance":"PRESENTATION_APPEARANCE_AUTO"}},"motionMode":"MOTION_MODE_CROP_PUSH","cameraMotion":{"durationOutputFrames":72,"profileId":"screen-entry-projective-contain-v1","profileVersion":1,"matteRgb":{"red":17,"green":20,"blue":28}},"videoTiming":video_timing}


def initializer(starter_id):
    return rpc("physical", "PhysicalStarterService", "InitializeReviewedStarter", {"starterId": starter_id})


def preset(ref):
    response = rpc("shots", "ShotsService", "GetPhysicalShotPreset", {"presetId":ref["id"],"revision":integer(ref["revision"])})
    row = response.get("revision") or {}
    if row.get("sha256") != ref.get("sha256"):
        raise Stop(f"preset readback digest differs for {ref['id']} revision {ref['revision']}")
    return row


def save_motion(example):
    candidate = json.loads((ROOT / example["motion_input"]).read_text())
    motion_id = example["motion_id"]
    try:
        head = rpc("motions", "MotionsService", "GetMotionHead", {"motionId":motion_id}).get("revision") or {}
        if head.get("sha256") != example["motion_sha256"] or integer(head.get("revision")) != example["motion_revision"]:
            raise Stop(f"motion ID {motion_id} already has a different owner head; preserving it")
        return {"id":motion_id,"revision":example["motion_revision"],"sha256":head["sha256"]}
    except Stop as e:
        if "HTTP 404" not in str(e):
            raise
    response = rpc("motions", "MotionsService", "SaveMotionCandidate", candidate)
    row = response.get("revision") or {}
    if row.get("sha256") != example["motion_sha256"] or integer(row.get("revision")) != example["motion_revision"]:
        raise Stop(f"saved motion candidate {motion_id} did not match its pinned revision/hash")
    return {"id":motion_id,"revision":example["motion_revision"],"sha256":row["sha256"]}


def save_preset(preset_id, name, setup, state):
    old = state["presets"].get(preset_id)
    try:
        if old:
            return preset({"id":preset_id,"revision":old["revision"],"sha256":old["sha256"]})
        head = rpc("shots", "ShotsService", "GetPhysicalShotPresetHead", {"presetId":preset_id}).get("revision") or {}
        raise Stop(f"preset ID {preset_id} already exists at revision {head.get('revision')} without a matching curation pin")
    except Stop as e:
        if "HTTP 404" not in str(e):
            raise
    row = (rpc("shots", "ShotsService", "SavePhysicalShotPreset", {"presetId":preset_id,"expectedRevision":0,"name":name,"setup":setup}).get("revision") or {})
    if row.get("candidateOnly") is not True or not row.get("sha256"):
        raise Stop(f"new setup preset {preset_id} was not returned as a candidate-only revision")
    state["presets"][preset_id] = {"revision":integer(row.get("revision")),"sha256":row["sha256"]}
    write_state(state)
    return row


def refs_from_setup(setup):
    refs = {}
    for label in ("device", "scene", "motion"):
        ref = setup.get(label) or {}
        if not ref.get("id") or not ref.get("sha256") or not integer(ref.get("revision")):
            raise Stop(f"physical preset setup lacks exact {label} revision identity")
        refs[label] = dict(ref)
    return refs


def build_physical_binding(setup, width, height, motion_output=None, backend="PHYSICAL_RENDER_BACKEND_OPTIX"):
    binding = {key:setup[key] for key in ("device","scene","motion","displayId","deviceSupportId","sceneSupportId","cameraId","sceneSupportWorld","supportPivotM","deviceRootRest") if key in setup}
    binding.update({"screenWidthPixels":width,"screenHeightPixels":height,"rendererProfileId":"device-studio-physical-cycles-v1","rendererProfileVersion":1,"backend":backend})
    if motion_output:
        binding["motionOutput"] = motion_output
    else:
        binding["outputProfile"] = "PHYSICAL_STILL_OUTPUT_PROFILE_2560X1440_V1"
        binding["motionSampleTick"] = {"numerator":"0","denominator":"1"}
    return binding


def physical_presentation(example_id, media_id):
    if example_id == "phone-rotation":
        return {"sourceMediaId":media_id,"captureArea":"CAPTURE_AREA_APP_ONLY","kind":"PRESENTATION_KIND_PHONE_SYSTEM","fitMode":"SCREEN_FIT_MODE_CONTAIN","phoneSystemChrome":{"clockText":"9:41","batteryPercent":84,"showCellularIndicator":True,"showWifiIndicator":True,"showHomeIndicator":True,"showStatus":True,"showBatteryPercent":False,"appearance":"PRESENTATION_APPEARANCE_AUTO"}}
    return {"sourceMediaId":media_id,"captureArea":"CAPTURE_AREA_APP_ONLY","kind":"PRESENTATION_KIND_BROWSER_DESKTOP","fitMode":"SCREEN_FIT_MODE_CONTAIN","browserDesktopChrome":{"windowTitle":"Meridian — product preview","addressText":"studio.local/projects/meridian","showWindowControls":True,"showToolbar":True,"appearance":"PRESENTATION_APPEARANCE_LIGHT","window":{"x":800,"y":800,"width":8400,"height":8000},"showMenuBar":True,"showDock":True}}


def store_physical_demo(example, project_id, media_id, state):
    pid = f"ds-demo-{example['id']}"
    motion = save_motion(example)
    motion_ref = {"id":motion["id"],"revision":motion["revision"],"sha256":motion["sha256"]}
    if pid in state["presets"]:
        saved = preset({"id":pid,"revision":state["presets"][pid]["revision"],"sha256":state["presets"][pid]["sha256"]})
    else:
        starter = initializer(example["starter_id"])
        ref = (starter.get("phoneStudyPreset") or starter.get("initializedPreset")) if example["id"] == "phone-rotation" else starter.get("initializedPreset")
        if not ref:
            raise Stop(f"starter {example['starter_id']} returned no preset revision")
        base = preset(ref)
        setup = dict(base.get("setup") or {})
        refs_from_setup(setup)
        setup["motion"] = motion_ref
        profiles = setup.get("motionProfiles", [])
        if "PHYSICAL_MOTION_OUTPUT_PROFILE_1080P_V2" not in profiles:
            setup["motionProfiles"] = profiles + ["PHYSICAL_MOTION_OUTPUT_PROFILE_1080P_V2"]
        saved = save_preset(pid, example["shot"], setup, state)
    reopened = preset({"id":pid,"revision":integer(saved.get("revision")),"sha256":saved.get("sha256")})
    setup = reopened["setup"]
    if setup.get("motion") != motion_ref:
        raise Stop(f"reopened preset {pid} changed its exact motion binding")
    motion_output = {"profileId":"device-studio-physical-motion-1080p-v2","profileVersion":2,"width":1920,"height":1080,"outputRateNumerator":30 if example["id"] == "phone-rotation" else 24,"outputRateDenominator":1,"frameCount":300 if example["id"] == "phone-rotation" else 240}
    if example["id"] == "phone-rotation":
        width, height = 1206, 2622
    else:
        width, height = 1644, 1000
    snapshot = {"presentation":physical_presentation(example["id"], media_id),"motionMode":"MOTION_MODE_FIXED","physicalBinding":build_physical_binding(setup,width,height,motion_output)}
    shot_id = create_shot(project_id, example["id"], example["shot"], snapshot, state)
    state["shots"][f"{example['id']}:{example['shot']}"]["preset"] = {"id":pid,"revision":integer(saved.get("revision")),"sha256":saved.get("sha256")}
    write_state(state)
    return shot_id


def display_dimensions(ref, setup):
    response = rpc("devices", "DevicesService", "GetDeviceRevision", {"deviceId":ref["id"],"revision":integer(ref["revision"])})
    row = response.get("revision") or {}
    if row.get("sha256") != ref.get("sha256"):
        raise Stop(f"device owner readback digest differs for {ref['id']}")
    displays = (((row.get("candidate") or {}).get("definition") or {}).get("displays") or [])
    display = next((d for d in displays if d.get("id") == setup.get("displayId")), None)
    metric = (display or {}).get("coordinateMetric") or {}
    w, h = float(metric.get("widthMm") or 0), float(metric.get("heightMm") or 0)
    if w <= 0 or h <= 0:
        raise Stop(f"device {ref['id']} display has no positive owner coordinate metric")
    height = 1200
    return max(128, round(height * w / h)), height


def family_showcase(project_id, media_ids, state):
    example = next(e for e in MANIFEST["examples"] if e["id"] == "family-showcase")
    if len(example["starter_ids"]) != len(set(example["starter_ids"])) or set(example["source_assignments"]) != set(example["starter_ids"]):
        raise Stop("family starter IDs and source assignments must be unique and exact")
    for index, starter_id in enumerate(example["starter_ids"], 1):
        name = f"Candidate {index:02d} — {starter_id}"
        prior = state["shots"].get(f"family-showcase:{name}") or {}
        pinned = prior.get("preset") or state["familyCandidates"].get(starter_id)
        if pinned:
            ref = {key:pinned[key] for key in ("id", "revision", "sha256")}
            row = preset(ref)
        else:
            result = initializer(starter_id)
            ref = result.get("initializedPreset")
            if not ref or result.get("candidateOnly") is not True:
                raise Stop(f"family starter {starter_id} did not return an explicit candidate preset")
            state["familyCandidates"][starter_id] = {"id":ref["id"],"revision":integer(ref["revision"]),"sha256":ref["sha256"],"candidateOnly":True}
            write_state(state)
            row = preset(ref)
        setup = row.get("setup") or {}
        refs = refs_from_setup(setup)
        width, height = display_dimensions(refs["device"], setup)
        media_key = example["source_assignments"].get(starter_id)
        if media_key not in media_ids:
            raise Stop(f"family starter {starter_id} has no manifest-pinned source assignment")
        snapshot = {"presentation":{"sourceMediaId":media_ids[media_key],"captureArea":"CAPTURE_AREA_APP_ONLY","kind":"PRESENTATION_KIND_PLAIN","fitMode":"SCREEN_FIT_MODE_CONTAIN"},"motionMode":"MOTION_MODE_FIXED","physicalBinding":build_physical_binding(setup,width,height,backend="PHYSICAL_RENDER_BACKEND_CPU")}
        shot_id = create_shot(project_id, "family-showcase", name, snapshot, state)
        state["shots"][f"family-showcase:{name}"]["preset"] = {"id":ref["id"],"revision":integer(ref["revision"]),"sha256":ref["sha256"],"candidateOnly":True,"starterId":starter_id}
        write_state(state)


def family_route_rows():
    """Resolve only the configured public route IDs from current owner reads."""
    rows = pages("shots", "ShotsService", "ListPhysicalShotPresets", "presets", {"pageSize":100}, "afterPresetId")
    by_id = {}
    for row in rows:
        preset_id = row.get("presetId")
        if preset_id not in {value[0] for value in FAMILY_REVISION2.values()}:
            continue
        current = by_id.get(preset_id)
        if current is None or integer(row.get("revision")) > integer(current.get("revision")):
            by_id[preset_id] = row
    devices = catalog_rows("devices", "ListDeviceCandidates", "candidates", "nextAfterDeviceId")
    device_states = {(row.get("deviceId"), integer(row.get("revision"))): row.get("state") for row in devices}
    result = {}
    for starter, (preset_id, source_key) in FAMILY_REVISION2.items():
        row = by_id.get(preset_id)
        if not row or not row.get("sha256"):
            raise Stop(f"current public preset {preset_id} is missing; preserving all family shot histories")
        exact = preset({"id":preset_id,"revision":integer(row.get("revision")),"sha256":row["sha256"]})
        setup = exact.get("setup") or {}
        refs = refs_from_setup(setup)
        if device_states.get((refs["device"]["id"], refs["device"]["revision"])) != "qualified":
            raise Stop(f"{starter} route device {refs['device']['id']} revision {refs['device']['revision']} is not currently qualified")
        result[starter] = {"preset":{"id":preset_id,"revision":integer(row["revision"]),"sha256":row["sha256"]},"setup":setup,"sourceKey":source_key}
    return result


def source_for_family(project_id, source_key, state):
    key = f"family-showcase:{source_key}"
    media_id = state.get("media", {}).get(key)
    asset = MANIFEST["assets"].get(source_key)
    if not asset:
        raise Stop(f"family source {source_key} has no manifest SHA pin")
    allowed_assets = next(e for e in MANIFEST["examples"] if e["id"] == "family-showcase").get("source_assets", [])
    if source_key not in allowed_assets:
        raise Stop(f"family source {source_key} is not explicitly assigned to the showcase source asset set")
    if not media_id:
        return ensure_media(project_id, "family-showcase", source_key, state)
    rows = list_media(project_id)
    matches = [row for row in rows if row.get("id") == media_id]
    if len(matches) != 1 or matches[0].get("sha256") != asset["sha256"]:
        raise Stop(f"family source {media_id} is missing or differs from the current pinned source bytes")
    return matches[0]


def family_revision2_snapshot(old_snapshot, setup, source_id):
    snapshot = copy.deepcopy(old_snapshot)
    presentation = snapshot.setdefault("presentation", {})
    presentation.update({"sourceMediaId":source_id,"captureArea":"CAPTURE_AREA_APP_ONLY","kind":"PRESENTATION_KIND_PLAIN","fitMode":"SCREEN_FIT_MODE_CONTAIN"})
    refs = refs_from_setup(setup)
    width, height = display_dimensions(refs["device"], setup)
    snapshot["physicalBinding"] = build_physical_binding(setup, width, height, backend="PHYSICAL_RENDER_BACKEND_CPU")
    snapshot["motionMode"] = "MOTION_MODE_FIXED"
    snapshot.pop("videoTiming", None)
    snapshot.pop("cameraMotion", None)
    return snapshot


def digest_json(value):
    return hashlib.sha256(json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False).encode()).hexdigest()


def append_family_revision2(project_id, state, starter, route, source):
    name = next(f"Candidate {index:02d} — {starter}" for index, item in enumerate(
        next(e for e in MANIFEST["examples"] if e["id"] == "family-showcase")["starter_ids"], 1) if item == starter)
    key = f"family-showcase:{name}"
    pin = state["shots"].get(key)
    if not pin or not pin.get("id") or integer(pin.get("revision")) not in (1, 2):
        raise Stop(f"{name} has no exact revision-1 or accepted revision-2 curator pin; refusing adoption")
    shot_id = pin["id"]
    opened = rpc("shots", "ShotsService", "GetShot", {"id":shot_id})
    head = opened.get("head") or {}
    head_revision = integer(head.get("revision"))
    pending = state.setdefault("familyRevision2", {}).get(key)
    if integer(pin.get("revision")) == 2:
        accepted = (pending or {}).get("accepted") or {}
        if head_revision != 2 or accepted.get("revision") != 2 or accepted.get("recipeSha256") != head.get("recipeSha256"):
            raise Stop(f"{name} no longer matches the accepted revision-2 curator pin; refusing to overwrite a user edit")
        current = rpc("shots", "ShotsService", "GetShotRevision", {"shotId":shot_id,"revision":2}).get("revision") or {}
        current_snapshot = current.get("snapshot") or {}
        if digest_json(current_snapshot) != accepted.get("targetSnapshotSha256"):
            raise Stop(f"{name} accepted revision 2 changed after its checkpoint; preserving it and stopping")
        if (current_snapshot.get("presentation") or {}).get("sourceMediaId") != source["id"]:
            raise Stop(f"{name} accepted revision 2 no longer binds its pinned source")
        for label in ("device", "scene", "motion"):
            if (current_snapshot.get("physicalBinding") or {}).get(label) != route["setup"].get(label):
                raise Stop(f"{name} accepted revision 2 no longer binds the current exact {label} route")
        compiled = rpc("shots", "ShotsService", "CompileShot", {"shotId":shot_id,"revision":2}).get("compiled") or {}
        if not compiled.get("renderEligible") or compiled.get("recipeSha256") != accepted.get("recipeSha256"):
            raise Stop(f"{name} accepted revision 2 no longer compiles to its accepted recipe")
        return {"name":name,"id":shot_id,"revision":2,"recipeSha256":compiled["recipeSha256"],
                "sourceMediaId":source["id"],"sourceMediaSha256":source["sha256"],"preset":route["preset"],"resumed":True}
    if head_revision == 1:
        if pin.get("recipeSha256") and head.get("recipeSha256") and head["recipeSha256"] != pin["recipeSha256"]:
            raise Stop(f"{name} revision 1 differs from the stored curator recipe pin")
        old = rpc("shots", "ShotsService", "GetShotRevision", {"shotId":shot_id,"revision":1}).get("revision") or {}
        old_snapshot = old.get("snapshot") or {}
        if not old_snapshot:
            raise Stop(f"{name} revision 1 has no reopenable snapshot")
        target = family_revision2_snapshot(old_snapshot, route["setup"], source["id"])
        target_sha = digest_json(target)
        if pending and pending.get("targetSnapshotSha256") != target_sha:
            raise Stop(f"{name} has an unresolved revision-2 write-ahead plan that differs from the current owner route")
        if not pending:
            pending = {"expectedRevision":1,"targetRevision":2,"targetSnapshotSha256":target_sha,
                       "preset":route["preset"],"sourceMediaId":source["id"],"sourceMediaSha256":source["sha256"]}
            state["familyRevision2"][key] = pending
            state["schema"] = "device-studio-demo-run-state/v2"
            write_state(state)
        saved = rpc("shots", "ShotsService", "SaveShot", {"shotId":shot_id,"expectedRevision":1,"name":name,"snapshot":target})
        response_revision = saved.get("revision") or {}
        if integer(response_revision.get("revision")) != 2:
            raise Stop(f"SaveShot for {name} returned no exact revision-2 receipt; preserving write-ahead plan")
        head_revision = 2
    elif head_revision == 2:
        if not pending:
            raise Stop(f"{name} is revision 2 without this curator's durable pending checkpoint; refusing to adopt it")
        revision = rpc("shots", "ShotsService", "GetShotRevision", {"shotId":shot_id,"revision":2}).get("revision") or {}
        if digest_json(revision.get("snapshot") or {}) != pending.get("targetSnapshotSha256"):
            raise Stop(f"{name} revision 2 differs from this curator's write-ahead plan")
    else:
        accepted = state.setdefault("familyRevision2", {}).get(key, {}).get("accepted") or {}
        if head_revision != 2 or accepted.get("revision") != head_revision or accepted.get("recipeSha256") != head.get("recipeSha256"):
            raise Stop(f"{name} has unexpected head revision {head_revision}; refusing to overwrite a user edit")
    revision = rpc("shots", "ShotsService", "GetShotRevision", {"shotId":shot_id,"revision":2}).get("revision") or {}
    snapshot = revision.get("snapshot") or {}
    if not snapshot or (snapshot.get("presentation") or {}).get("sourceMediaId") != source["id"]:
        raise Stop(f"{name} revision 2 reopened with a different source binding")
    physical = snapshot.get("physicalBinding") or {}
    for label in ("device", "scene", "motion"):
        if physical.get(label) != route["setup"].get(label):
            raise Stop(f"{name} revision 2 reopened with a different exact {label} route")
    compiled = rpc("shots", "ShotsService", "CompileShot", {"shotId":shot_id,"revision":2}).get("compiled") or {}
    if not compiled.get("renderEligible"):
        raise Stop(f"{name} revision 2 failed compilation: {json.dumps(compiled.get('issues', []))[:1200]}")
    prior = {key:pin.get(key) for key in ("id", "revision", "compiled", "recipeSha256", "sourceMediaSha256", "preset") if key in pin}
    history = pin.setdefault("history", [])
    if not history or history[-1] != prior:
        history.append(prior)
    pin.update({"revision":2,"compiled":True,"recipeSha256":compiled.get("recipeSha256"),
                "sourceMediaSha256":[source["sha256"]],"preset":dict(route["preset"],candidateOnly=True,starterId=starter)})
    pending = state["familyRevision2"][key]
    pending["accepted"] = {"revision":2,"recipeSha256":compiled.get("recipeSha256"),"targetSnapshotSha256":digest_json(snapshot)}
    write_state(state)
    return {"name":name,"id":shot_id,"revision":2,"recipeSha256":compiled.get("recipeSha256"),"sourceMediaId":source["id"],"sourceMediaSha256":source["sha256"],"preset":route["preset"]}


def family_revise2():
    get_health()
    state = read_state()
    project_id = state.get("projects", {}).get("family-showcase")
    if project_id != "c952d9c8-eee6-40a7-8d75-2a1eea702261":
        raise Stop("family-showcase project pin differs from the explicitly authorized PRIMARY target")
    projects = [row for row in project_rows() if row.get("id") == project_id]
    if len(projects) != 1 or projects[0].get("name") != "Device Studio demo — Remaining family showcase":
        raise Stop("authorized family project identity did not reopen exactly")
    routes = family_route_rows()
    # Tablet and monitor are supplied as exact transferred primary-owner routes.
    # Until then, retain those revision-1 shots untouched and never substitute old routes.
    starters = [starter for starter in FAMILY_REVISION2 if starter in routes]
    results = []
    for starter in starters:
        index = next(i for i, value in enumerate(
            next(e for e in MANIFEST["examples"] if e["id"] == "family-showcase")["starter_ids"], 1) if value == starter)
        key = f"family-showcase:Candidate {index:02d} — {starter}"
        pin = state.get("shots", {}).get(key) or {}
        pinned_revision = integer(pin.get("revision"))
        if starter == "generic-round-watch-v2" and pinned_revision == WATCH_SYNTHETIC_SHOT["revision"]:
            results.append(append_watch_synthetic_revision4(project_id, state))
            continue
        source = source_for_family(project_id, routes[starter]["sourceKey"], state)
        if pinned_revision == 3:
            results.append(append_family_optix_revision3(project_id, state, starter, routes[starter], source))
        else:
            results.append(append_family_revision2(project_id, state, starter, routes[starter], source))
    missing = [starter for starter in FAMILY_REVISION2 if starter not in routes]
    print(json.dumps({"status":"family_routes_reopened_and_compiled","projectId":project_id,"updated":results,
                      "pendingRoutes":missing},indent=2))


def optix_revision3_snapshot(snapshot):
    updated = copy.deepcopy(snapshot)
    binding = updated.get("physicalBinding") or {}
    if not binding:
        raise Stop("physical binding is missing; refusing to change render backend")
    binding["backend"] = "PHYSICAL_RENDER_BACKEND_OPTIX"
    updated["physicalBinding"] = binding
    return updated


def append_family_optix_revision3(project_id, state, starter, route, source):
    index = next(index for index, item in enumerate(
        next(e for e in MANIFEST["examples"] if e["id"] == "family-showcase")["starter_ids"], 1) if item == starter)
    name = f"Candidate {index:02d} — {starter}"
    key = f"family-showcase:{name}"
    pin = state.get("shots", {}).get(key)
    if not pin or not pin.get("id") or integer(pin.get("revision")) not in (2, 3):
        raise Stop(f"{name} has no exact revision-2 or accepted revision-3 curation pin")
    shot_id = pin["id"]
    head = (rpc("shots", "ShotsService", "GetShot", {"id":shot_id}).get("head") or {})
    head_revision = integer(head.get("revision"))
    pending = state.setdefault("familyRevision3", {}).get(key)
    if integer(pin.get("revision")) == 3:
        accepted = (pending or {}).get("accepted") or {}
        current = rpc("shots", "ShotsService", "GetShotRevision", {"shotId":shot_id,"revision":3}).get("revision") or {}
        snapshot = current.get("snapshot") or {}
        if head_revision != 3 or head.get("recipeSha256") != accepted.get("recipeSha256") or digest_json(snapshot) != accepted.get("targetSnapshotSha256"):
            raise Stop(f"{name} no longer matches the accepted revision-3 checkpoint; preserving it")
        compiled = rpc("shots", "ShotsService", "CompileShot", {"shotId":shot_id,"revision":3}).get("compiled") or {}
        if not compiled.get("renderEligible") or compiled.get("recipeSha256") != accepted.get("recipeSha256"):
            raise Stop(f"{name} accepted revision 3 no longer compiles to its pinned recipe")
        return {"name":name,"id":shot_id,"revision":3,"recipeSha256":compiled["recipeSha256"],"backend":"PHYSICAL_RENDER_BACKEND_OPTIX","resumed":True}
    if head_revision == 2:
        accepted2 = (state.get("familyRevision2", {}).get(key, {}).get("accepted") or {})
        revision2 = rpc("shots", "ShotsService", "GetShotRevision", {"shotId":shot_id,"revision":2}).get("revision") or {}
        snapshot2 = revision2.get("snapshot") or {}
        if not accepted2.get("recipeSha256") or digest_json(snapshot2) != accepted2.get("targetSnapshotSha256"):
            raise Stop(f"{name} revision 2 differs from its accepted curator checkpoint; refusing to overwrite")
        if (snapshot2.get("presentation") or {}).get("sourceMediaId") != source["id"]:
            raise Stop(f"{name} revision 2 source no longer matches its pinned current source")
        for label in ("device", "scene", "motion"):
            if (snapshot2.get("physicalBinding") or {}).get(label) != route["setup"].get(label):
                raise Stop(f"{name} revision 2 {label} differs from current owner route")
        target = optix_revision3_snapshot(snapshot2)
        target_sha = digest_json(target)
        if pending and pending.get("targetSnapshotSha256") != target_sha:
            raise Stop(f"{name} unresolved revision-3 write-ahead plan differs from this exact backend change")
        if not pending:
            pending = {"expectedRevision":2,"targetRevision":3,"targetSnapshotSha256":target_sha,
                       "priorRevision":2,"priorRecipeSha256":accepted2["recipeSha256"],"backend":"PHYSICAL_RENDER_BACKEND_OPTIX"}
            state["familyRevision3"][key] = pending
            write_state(state)
        response = rpc("shots", "ShotsService", "SaveShot", {"shotId":shot_id,"expectedRevision":2,"name":name,"snapshot":target})
        if integer((response.get("revision") or {}).get("revision")) != 3:
            raise Stop(f"{name} SaveShot returned no exact revision-3 receipt; preserving the checkpoint")
        head_revision = 3
    elif head_revision == 3 and pending:
        revision3 = rpc("shots", "ShotsService", "GetShotRevision", {"shotId":shot_id,"revision":3}).get("revision") or {}
        if digest_json(revision3.get("snapshot") or {}) != pending.get("targetSnapshotSha256"):
            raise Stop(f"{name} revision 3 differs from the pending backend change; refusing adoption")
    else:
        raise Stop(f"{name} has unexpected head revision {head_revision}; refusing to overwrite a user edit")
    revision3 = rpc("shots", "ShotsService", "GetShotRevision", {"shotId":shot_id,"revision":3}).get("revision") or {}
    snapshot3 = revision3.get("snapshot") or {}
    if digest_json(snapshot3) != pending.get("targetSnapshotSha256") or (snapshot3.get("physicalBinding") or {}).get("backend") != "PHYSICAL_RENDER_BACKEND_OPTIX":
        raise Stop(f"{name} revision 3 did not reopen as the exact OptiX-only change")
    compiled = rpc("shots", "ShotsService", "CompileShot", {"shotId":shot_id,"revision":3}).get("compiled") or {}
    if not compiled.get("renderEligible"):
        raise Stop(f"{name} revision 3 failed compilation: {json.dumps(compiled.get('issues', []))[:1200]}")
    history = pin.setdefault("history", [])
    prior2 = {"id":pin["id"],"revision":2,"compiled":pin.get("compiled"),"recipeSha256":pin.get("recipeSha256"),"sourceMediaSha256":pin.get("sourceMediaSha256"),"preset":pin.get("preset")}
    if not any(entry.get("revision") == 2 for entry in history):
        history.append(prior2)
    pin.update({"revision":3,"compiled":True,"recipeSha256":compiled.get("recipeSha256"),"backend":"PHYSICAL_RENDER_BACKEND_OPTIX"})
    pending["accepted"] = {"revision":3,"recipeSha256":compiled.get("recipeSha256"),"targetSnapshotSha256":digest_json(snapshot3)}
    write_state(state)
    return {"name":name,"id":shot_id,"revision":3,"recipeSha256":compiled.get("recipeSha256"),"backend":"PHYSICAL_RENDER_BACKEND_OPTIX"}


def family_optix_revise3():
    get_health()
    state = read_state()
    project_id = state.get("projects", {}).get("family-showcase")
    if project_id != "c952d9c8-eee6-40a7-8d75-2a1eea702261":
        raise Stop("family-showcase project pin differs from the explicitly authorized PRIMARY target")
    routes = family_route_rows()
    results = []
    for starter, route in routes.items():
        index = next(i for i, value in enumerate(
            next(e for e in MANIFEST["examples"] if e["id"] == "family-showcase")["starter_ids"], 1) if value == starter)
        key = f"family-showcase:Candidate {index:02d} — {starter}"
        if starter == "generic-round-watch-v2" and integer((state.get("shots", {}).get(key) or {}).get("revision")) == WATCH_SYNTHETIC_SHOT["revision"]:
            results.append(append_watch_synthetic_revision4(project_id, state))
            continue
        source = source_for_family(project_id, route["sourceKey"], state)
        results.append(append_family_optix_revision3(project_id, state, starter, route, source))
    print(json.dumps({"status":"family_native_routes_reopened_and_compiled","projectId":project_id,"updated":results},indent=2))


def watch_synthetic_snapshot(prior_snapshot, source_media_id):
    """Apply only the pinned synthetic source and optical scene revision."""
    updated = copy.deepcopy(prior_snapshot)
    updated.setdefault("presentation", {})["sourceMediaId"] = source_media_id
    binding = updated.get("physicalBinding") or {}
    binding["scene"] = {"id": WATCH_SYNTHETIC_SCENE["id"], "revision": WATCH_SYNTHETIC_SCENE["revision"], "sha256": WATCH_SYNTHETIC_SCENE["sha256"]}
    updated["physicalBinding"] = binding
    return updated


def ensure_watch_synthetic_media(project_id, state):
    item = MANIFEST["assets"].get(WATCH_SYNTHETIC_SOURCE)
    if not item:
        raise Stop("manifest is missing the pinned synthetic watch calibration source")
    path = ROOT / item["path"]
    if sha(path) != WATCH_SYNTHETIC_SHOT["source_sha256"] or item.get("sha256") != WATCH_SYNTHETIC_SHOT["source_sha256"]:
        raise Stop("synthetic watch calibration source differs from its reviewed digest")
    state_key = f"family-showcase:{WATCH_SYNTHETIC_SOURCE}"
    pinned = state.get("media", {}).get(state_key)
    rows = list_media(project_id)
    matches = [row for row in rows if row.get("sha256") == WATCH_SYNTHETIC_SHOT["source_sha256"]]
    if len(matches) > 1:
        raise Stop("multiple copies of the exact watch calibration source exist; refusing to choose")
    if pinned:
        match = next((row for row in matches if row.get("id") == pinned), None)
        if not match:
            raise Stop("pinned watch calibration media is missing or has changed")
        return match
    if matches:
        row = matches[0]
    else:
        conflicts = [row for row in rows if row.get("originalFilename") == path.name]
        if conflicts:
            raise Stop("watch calibration filename exists with different bytes; refusing to upload another copy")
        response = multipart_upload(project_id, path, item["media_type"])
        row = response.get("source_media") or {}
        if row.get("sha256") != WATCH_SYNTHETIC_SHOT["source_sha256"] or not row.get("id"):
            raise Stop("watch calibration upload returned an unexpected source identity")
        row = next((candidate for candidate in list_media(project_id) if candidate.get("id") == row["id"]), None)
        if not row or row.get("sha256") != WATCH_SYNTHETIC_SHOT["source_sha256"]:
            raise Stop("watch calibration upload did not reopen by exact public source identity")
    state.setdefault("media", {})[state_key] = row["id"]
    write_state(state)
    return row


def append_watch_synthetic_revision4(project_id, state):
    key = "family-showcase:Candidate 03 — generic-round-watch-v2"
    pin = state.get("shots", {}).get(key)
    if not pin or pin.get("id") != WATCH_SYNTHETIC_SHOT["id"]:
        raise Stop("watch family shot identity differs from its pinned source project; refusing adoption")
    source = ensure_watch_synthetic_media(project_id, state)
    scene_head = rpc("scenes", "ScenesService", "GetSceneHead", {"sceneId": WATCH_SYNTHETIC_SCENE["id"]}).get("revision") or {}
    if integer(scene_head.get("revision")) == WATCH_SYNTHETIC_SCENE["prior_revision"]:
        if scene_head.get("sha256") != WATCH_SYNTHETIC_SCENE["prior_sha256"]:
            raise Stop("watch scene prior revision differs from the authored optical baseline")
        candidate = copy.deepcopy(scene_head.get("candidate") or {})
        cameras = [camera for camera in candidate.get("cameraFrames", []) if camera.get("id") == WATCH_SYNTHETIC_SCENE["camera_id"]]
        if len(cameras) != 1:
            raise Stop("watch scene no longer has exactly one pinned camera")
        camera = cameras[0]
        focus = camera.get("focus") or {}
        if camera.get("dofFNumber") != 8 or focus.get("mode") != "DISTANCE" or focus.get("distanceM") != WATCH_SYNTHETIC_SCENE["focus_distance_m"]:
            raise Stop("watch camera focus baseline changed; refusing to overwrite it")
        camera["dofFNumber"] = WATCH_SYNTHETIC_SCENE["f_number"]
        response = rpc("scenes", "ScenesService", "SaveSceneCandidate", {"expectedHead":WATCH_SYNTHETIC_SCENE["prior_revision"],"candidate":candidate})
        scene_head = response.get("revision") or {}
    if integer(scene_head.get("revision")) != WATCH_SYNTHETIC_SCENE["revision"] or scene_head.get("sha256") != WATCH_SYNTHETIC_SCENE["sha256"]:
        raise Stop("watch scene head is not the exact reviewed f/16 revision; preserving it")

    preset = rpc("shots", "ShotsService", "GetPhysicalShotPresetHead", {"presetId":WATCH_SYNTHETIC_PRESET["id"]}).get("revision") or {}
    if integer(preset.get("revision")) == WATCH_SYNTHETIC_PRESET["prior_revision"]:
        if preset.get("sha256") != WATCH_SYNTHETIC_PRESET["prior_sha256"]:
            raise Stop("watch preset prior revision differs from the reviewed baseline")
        setup = copy.deepcopy(preset.get("setup") or {})
        setup["scene"] = {"id":WATCH_SYNTHETIC_SCENE["id"],"revision":WATCH_SYNTHETIC_SCENE["revision"],"sha256":WATCH_SYNTHETIC_SCENE["sha256"]}
        response = rpc("shots", "ShotsService", "SavePhysicalShotPreset", {"presetId":WATCH_SYNTHETIC_PRESET["id"],"expectedRevision":WATCH_SYNTHETIC_PRESET["prior_revision"],"name":preset.get("name", "")+" · synthetic dial sharpness v1","setup":setup})
        preset = response.get("revision") or {}
    if integer(preset.get("revision")) != WATCH_SYNTHETIC_PRESET["revision"] or preset.get("sha256") != WATCH_SYNTHETIC_PRESET["sha256"]:
        raise Stop("watch preset head is not the exact reviewed scene binding; preserving it")

    head = rpc("shots", "ShotsService", "GetShot", {"id":WATCH_SYNTHETIC_SHOT["id"]}).get("head") or {}
    revision = integer(head.get("revision"))
    if revision == WATCH_SYNTHETIC_SHOT["prior_revision"]:
        accepted = state.get("familyRevision3", {}).get(key, {}).get("accepted") or {}
        old = rpc("shots", "ShotsService", "GetShotRevision", {"shotId":WATCH_SYNTHETIC_SHOT["id"],"revision":revision}).get("revision") or {}
        old_snapshot = old.get("snapshot") or {}
        if head.get("recipeSha256") != WATCH_SYNTHETIC_SHOT["prior_recipe_sha256"] or digest_json(old_snapshot) != accepted.get("targetSnapshotSha256"):
            raise Stop("watch shot revision 3 differs from the last accepted curator checkpoint")
        target = watch_synthetic_snapshot(old_snapshot, source["id"])
        response = rpc("shots", "ShotsService", "SaveShot", {"shotId":WATCH_SYNTHETIC_SHOT["id"],"expectedRevision":revision,"name":WATCH_SYNTHETIC_SHOT["name"],"snapshot":target})
        written = response.get("revision") or {}
        if integer(written.get("revision")) != WATCH_SYNTHETIC_SHOT["revision"]:
            raise Stop("watch SaveShot returned an unexpected revision")
        head = rpc("shots", "ShotsService", "GetShot", {"id":WATCH_SYNTHETIC_SHOT["id"]}).get("head") or {}
    if integer(head.get("revision")) != WATCH_SYNTHETIC_SHOT["revision"] or head.get("recipeSha256") != WATCH_SYNTHETIC_SHOT["recipe_sha256"]:
        raise Stop("watch shot head is not the exact saved synthetic-calibration revision")
    saved = rpc("shots", "ShotsService", "GetShotRevision", {"shotId":WATCH_SYNTHETIC_SHOT["id"],"revision":WATCH_SYNTHETIC_SHOT["revision"]}).get("revision") or {}
    snapshot = saved.get("snapshot") or {}
    if (snapshot.get("presentation") or {}).get("sourceMediaId") != source["id"] or (snapshot.get("physicalBinding") or {}).get("scene") != {"id":WATCH_SYNTHETIC_SCENE["id"],"revision":WATCH_SYNTHETIC_SCENE["revision"],"sha256":WATCH_SYNTHETIC_SCENE["sha256"]}:
        raise Stop("watch shot no longer uses the pinned synthetic source and optical scene")
    compiled = rpc("shots", "ShotsService", "CompileShot", {"shotId":WATCH_SYNTHETIC_SHOT["id"],"revision":WATCH_SYNTHETIC_SHOT["revision"]}).get("compiled") or {}
    if not compiled.get("renderEligible") or compiled.get("recipeSha256") != WATCH_SYNTHETIC_SHOT["recipe_sha256"]:
        raise Stop("watch synthetic-calibration revision no longer compiles to the accepted recipe")
    pin.update({"revision":WATCH_SYNTHETIC_SHOT["revision"],"compiled":True,"recipeSha256":compiled["recipeSha256"],"sourceMediaSha256":[source["sha256"]],"preset":{"id":WATCH_SYNTHETIC_PRESET["id"],"revision":WATCH_SYNTHETIC_PRESET["revision"],"sha256":WATCH_SYNTHETIC_PRESET["sha256"],"candidateOnly":True,"starterId":"generic-round-watch-v2"}})
    checkpoint = state.setdefault("familyWatchRevision4", {}).setdefault(key, {})
    checkpoint.update({"revision":WATCH_SYNTHETIC_SHOT["revision"],"recipeSha256":compiled["recipeSha256"],"scene":dict(WATCH_SYNTHETIC_SCENE),"preset":dict(WATCH_SYNTHETIC_PRESET),"sourceMediaId":source["id"],"sourceSha256":source["sha256"],"candidateOnly":True,"claim":"synthetic dial calibration, not an app capture"})
    write_state(state)
    return {"id":WATCH_SYNTHETIC_SHOT["id"],"revision":WATCH_SYNTHETIC_SHOT["revision"],"recipeSha256":compiled["recipeSha256"],"sourceMediaId":source["id"],"sourceSha256":source["sha256"],"candidateOnly":True}


def family_watch_synthetic_revise4():
    verify_files()
    get_health()
    state = read_state()
    project_id = state.get("projects", {}).get("family-showcase")
    if project_id != "c952d9c8-eee6-40a7-8d75-2a1eea702261":
        raise Stop("family-showcase project pin differs from the explicitly authorized PRIMARY target")
    projects = [row for row in project_rows() if row.get("id") == project_id]
    if len(projects) != 1 or projects[0].get("name") != "Device Studio demo — Remaining family showcase":
        raise Stop("authorized family showcase project did not reopen exactly")
    result = append_watch_synthetic_revision4(project_id, state)
    print(json.dumps({"status":"watch_synthetic_revision4_reopened_and_compiled","projectId":project_id,"candidateOnly":True,"updated":result},indent=2))


def seed():
    verify_files()
    get_health()
    state = read_state()
    projects = project_rows()
    for example in MANIFEST["examples"]:
        project_id = find_or_create_project(example, state, projects)
        media_ids = {}
        for asset_id in example.get("source_assets", []):
            row = ensure_media(project_id, example["id"], asset_id, state)
            media_ids[asset_id] = row["id"]
        if example["type"] == "fixed_photo":
            snapshot = fixed_laptop_snapshot(media_ids) if example["id"] == "laptop-photo" else fixed_phone_snapshot(media_ids)
            create_shot(project_id, example["id"], example["shot"], snapshot, state)
        elif example["type"] == "physical_motion":
            source_id = media_ids["phone_frame0"] if example["id"] == "phone-rotation" else media_ids["laptop_app"]
            store_physical_demo(example, project_id, source_id, state)
        elif example["type"] == "video_entry":
            video_id = media_ids["phone_recording"]
            video, receipt = inspect_video(project_id, video_id, example["id"], state)
            snapshot = video_entry_snapshot(project_id, video, receipt, media_ids["phone_plate"], media_ids["phone_active_mask"], media_ids["phone_hardware_mask"])
            create_shot(project_id, example["id"], example["shot"], snapshot, state)
        elif example["type"] == "candidate_showcase":
            family_showcase(project_id, media_ids, state)
    state["status"] = "six_projects_saved_reopened_compiled"
    write_state(state)
    print(json.dumps({"status":state["status"],"projects":state["projects"],"shots":state["shots"],"rendering":"not submitted; coordinate the root-owned render lane"},indent=2))


def main():
    global API
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=("verify", "plan", "seed", "family-revise2", "family-optix-revise3", "family-watch-synthetic-revise4"))
    parser.add_argument("--api", default=API, help="already-running Device Studio API base; never auto-started")
    args = parser.parse_args()
    API = args.api.rstrip("/")
    try:
        if args.command == "verify":
            verify_files()
        elif args.command == "plan":
            verify_files()
            plan()
        elif args.command == "family-revise2":
            family_revise2()
        elif args.command == "family-optix-revise3":
            family_optix_revise3()
        elif args.command == "family-watch-synthetic-revise4":
            family_watch_synthetic_revise4()
        else:
            seed()
    except Stop as e:
        print(f"STOP: {e}", file=sys.stderr)
        raise SystemExit(2)


if __name__ == "__main__":
    main()
