#!/usr/bin/env python3
"""Served-browser smoke for the RCL workspace authoring journey.

Run while RCL is serving the UI and API (Python Playwright + installed Chrome):

  python3 tools/workbench_browser_journey.py

Override the URLs when the scenario uses different ports. The UI should be
served through its normal scenario proxy so REST and Connect requests exercise
the same path as an operator's browser.
"""

from __future__ import annotations

import argparse
import json
import re
import urllib.parse
import urllib.request
from pathlib import Path

from playwright.sync_api import Page, TimeoutError as PlaywrightTimeoutError, sync_playwright


def rpc(api_url: str, method: str, body: dict) -> dict:
    url = f"{api_url.rstrip('/')}/vrooli.react_component_library.v1.workspace.WorkspaceService/{method}"
    request = urllib.request.Request(
        url,
        data=json.dumps(body).encode(),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(request, timeout=15) as response:
        return json.loads(response.read())


def assert_phone_no_overflow(page: Page, width: int, height: int, route: str = "page") -> None:
    page.set_viewport_size({"width": width, "height": height})
    page.wait_for_timeout(150)
    metrics = page.evaluate(
        "({width: innerWidth, scrollWidth: document.documentElement.scrollWidth})"
    )
    assert metrics["scrollWidth"] <= metrics["width"], f"{route} at {width}px overflow: {metrics}"


def locate_eight_screen_source(source_root: Path | None = None) -> Path:
    if source_root is not None:
        candidates = [source_root.expanduser()]
    else:
        candidates = [
            Path.home()
            / ".vrooli/research/topics/experience-engineering/operator-experiments/"
            "rcl-eight-screen-workspace-20261121/data/eight-screens-source",
            Path.home() / "design-studies/eight-screens",
            Path(__file__).resolve().parents[3] / "experience/design-studies/eight-screens",
        ]
    for candidate in candidates:
        if (candidate / "index.html").is_file() and (candidate / "css").is_dir():
            return candidate
    searched = ", ".join(str(path) for path in candidates)
    raise FileNotFoundError(
        "Eight-screen HTML/CSS source bundle was not found. "
        f"Searched: {searched}. Pass --source-root to choose its current location."
    )


def run(ui_url: str, api_url: str, source_root: Path | None = None) -> dict:
    report: dict = {"keyboard": {}, "phone": {}, "workspace": {}, "drag_drop": {}, "capture_drift": {}}
    created_id: str | None = None
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(headless=True, channel="chrome")
        page = browser.new_page(viewport={"width": 1440, "height": 900})
        page.set_default_timeout(8_000)
        errors: list[str] = []
        api_responses: list[dict[str, str | int | bool]] = []
        capture_requests: list[str] = []
        page.on("pageerror", lambda error: errors.append(str(error)))
        page.on(
            "request",
            lambda request: capture_requests.append(request.url)
            if "captureworkspacestate" in request.url.casefold()
            else None,
        )

        def observe_response(response) -> None:
            nonlocal created_id
            path = urllib.parse.urlparse(response.url).path
            is_create = path.endswith("/WorkspaceService/CreateWorkspace")
            is_capture = "captureworkspacestate" in path.casefold()
            if not path.startswith("/api/") and not is_create and not is_capture:
                return
            api_responses.append({"status": response.status, "url": response.url})
            if response.status < 400 and is_create:
                try:
                    created_id = response.json().get("workspace", {}).get("id") or created_id
                except PlaywrightTimeoutError:
                    pass
            if is_capture:
                try:
                    body = response.json()
                    api_responses[-1]["body"] = json.dumps(body)[:500]
                    api_responses[-1]["capture_id"] = body.get("captureId", "")
                    api_responses[-1]["evidence_present"] = bool(body.get("evidence"))
                except Exception as error:
                    api_responses[-1]["body"] = f"response body unavailable: {type(error).__name__}"

        page.on("response", observe_response)
        try:
            page.goto(ui_url.rstrip("/") + "/", wait_until="domcontentloaded")
            page.get_by_role("link", name="Start").wait_for()
            page.keyboard.press("Control+k")
            palette = page.get_by_role("dialog")
            palette.wait_for()
            assert page.locator("input:focus").count() == 1, "Command palette did not focus search"
            page.keyboard.press("Escape")
            page.keyboard.press("g")
            page.keyboard.press("d")
            page.locator('[data-experience-surface="design-overview"]').wait_for()
            report["keyboard"] = {"command_palette_focus": True, "g_d_opens_design": True}

            destination_routes = (
                ("/", "start-page"),
                ("/library", "library-page"),
                ("/design", "design-index-page"),
                ("/review", "review-page"),
                ("/stats", "stats-page"),
            )
            destination_viewports: dict[str, dict[str, str]] = {}
            for route, test_id in destination_routes:
                viewport_results: dict[str, str] = {}
                for width, height in ((390, 844), (320, 720)):
                    # Load each route at its target viewport. Resizing an
                    # already-mounted sliding mobile nav can report a brief
                    # transition frame as document overflow even though the
                    # settled screen fits.
                    page.set_viewport_size({"width": width, "height": height})
                    page.goto(ui_url.rstrip("/") + route, wait_until="domcontentloaded")
                    page.get_by_test_id(test_id).wait_for()
                    page.wait_for_timeout(250)
                    assert_phone_no_overflow(page, width, height, route)
                    viewport_results[str(width)] = "no overflow"
                destination_viewports[route] = viewport_results
            report["phone"]["destinations"] = destination_viewports

            page.goto(ui_url.rstrip("/") + "/design", wait_until="domcontentloaded")
            page.get_by_test_id("design-index-page").wait_for()

            for width, height in ((390, 844), (320, 720)):
                assert_phone_no_overflow(page, width, height)
            nav_labels = page.locator("nav").all_inner_texts()
            assert any("Start" in label and "Library" in label and "Design" in label for label in nav_labels), "Phone navigation is missing"
            report["phone"]["design_index"] = {"390": "no overflow", "320": "no overflow", "bottom_navigation": True}

            page.set_viewport_size({"width": 1440, "height": 900})
            page.get_by_role("button", name="New workspace").click()
            page.wait_for_function("new URL(location.href).searchParams.has('workspace')")
            query = urllib.parse.parse_qs(urllib.parse.urlparse(page.url).query)
            created_id = query.get("workspace", [created_id])[0]
            assert created_id, f"Workspace URL has no workspace id: {page.url}"
            page.locator('[data-experience-surface="design-workspace"]').first.wait_for()
            report["workspace"]["created_id"] = created_id

            # The new workspace starts with custom text and a real RCL slot.
            page.get_by_role("button", name="Start with one clear idea.").click()
            page.get_by_role("tab", name=re.compile("Checks")).click()
            body = page.locator("body").inner_text()
            assert "No horizontal overflow" in body, "Workspace health did not measure its frame"
            assert "States are declared" in body, "Workspace health omits the state check"
            report["workspace"]["health"] = "RCL/custom coverage and frame health rendered"

            page.get_by_role("tab", name="Layers").click()
            layer_buttons = page.locator(".studio-tree-select")
            assert layer_buttons.count() >= 3, "Layer tree is missing editable nodes"
            source = layer_buttons.nth(3)
            target = layer_buttons.nth(4)
            before = layer_buttons.all_inner_texts()
            # Drop after the adjacent sibling; dropping immediately before it
            # is a valid operation but leaves the relative order unchanged.
            source.drag_to(target, target_position={"x": 3, "y": 23})
            page.wait_for_timeout(250)
            after = page.locator(".studio-tree-select").all_inner_texts()
            report["drag_drop"]["layer_tree_order_changed"] = before != after
            if before == after:
                # Keep the browser check honest if native drag events are
                # suppressed by the runner; exercise the published keyboard
                # reordering path and report pointer support separately.
                page.locator(".studio-tree-select").nth(3).focus()
                page.keyboard.press("Space")
                page.keyboard.press("ArrowDown")
                page.keyboard.press("Enter")
                page.wait_for_timeout(150)
                after = page.locator(".studio-tree-select").all_inner_texts()
                report["drag_drop"]["keyboard_reorder_changed"] = before != after
            assert report["drag_drop"].get("keyboard_reorder_changed", report["drag_drop"]["layer_tree_order_changed"]), f"Neither pointer nor keyboard reordering changed the layer tree: {before}"

            page.get_by_role("button", name="Phone").click()
            assert "390 × 844" in page.locator("body").inner_text(), "Phone display was not selected"
            report["phone"]["workspace_display"] = "390 × 844 selected"
            assert_phone_no_overflow(page, 390, 844)
            page.set_viewport_size({"width": 1440, "height": 900})

            page.get_by_role("tab", name="Inspect").click()
            page.get_by_role("button", name="Start with one clear idea.").click()
            page.get_by_text("RCL building blocks", exact=True).wait_for()
            report["workspace"]["replacement_picker"] = "selected custom layer exposes RCL replacement candidates"
            replacement = page.locator('.studio-node-replacement-option[aria-label^="Replace with "]').first
            if replacement.count():
                candidate = replacement.get_attribute("aria-label")
                replacement.click()
                page.wait_for_timeout(250)
                report["workspace"]["replacement_applied"] = candidate
            else:
                report["workspace"]["replacement_applied"] = "no compatible candidate was offered"

            page.get_by_role("button", name="Import an HTML page").click()
            dialog = page.get_by_test_id("html-import")
            dialog.locator('input[type="file"]').first.set_input_files(
                {
                    "name": "browser-smoke.html",
                    "mimeType": "text/html",
                    "buffer": b"<html><head><title>Browser smoke</title></head><body><main><section><h1>Imported heading</h1></section><section><p>Original editable sentence.</p></section></main></body></html>",
                }
            )
            page.get_by_test_id("html-import-name").fill("Browser smoke page")
            page.get_by_test_id("html-import-import").wait_for(state="visible")
            page.get_by_test_id("html-import-import").click()
            page.get_by_text("Original editable sentence.", exact=True).first.wait_for()
            page.get_by_role("button", name=re.compile("Original editable sentence")).click()
            page.get_by_role("tab", name="Inspect").click()
            page.get_by_role("textbox", name="Text").fill("Edited browser journey sentence.")
            page.get_by_text("Edited browser journey sentence.", exact=True).first.wait_for()
            page.wait_for_timeout(900)
            page.set_viewport_size({"width": 320, "height": 720})
            assert_phone_no_overflow(page, 320, 720)
            page.get_by_role("button", name="Inspect").click()
            mobile_text = page.get_by_role("textbox", name="Text")
            assert mobile_text.is_visible(), "Text inspector is not reachable at 320px"
            mobile_text.fill("Edited at 320px.")
            assert mobile_text.input_value() == "Edited at 320px.", "320px inspector did not apply the text edit"
            page.wait_for_function(
                "document.body.innerText.includes('SAVED · REVISION') && !document.body.innerText.includes('SAVING…')"
            )
            report["phone"]["workspace_editing_320"] = {
                "inspector_visible": mobile_text.is_visible(),
                "text_edit_saved": True,
                "no_overflow": True,
            }
            report["workspace"]["html_import"] = "imported a local HTML page as a new workspace page"
            report["workspace"]["edit_saved"] = "changed an imported text layer through the inspector, including at 320px"

            page.set_viewport_size({"width": 1440, "height": 900})
            canvas_node = page.locator('.studio-frame-page [data-node-type="text"]').filter(has_text="Imported heading").first
            target_node = page.locator('.studio-frame-page [data-node-type="stack"]').filter(has=page.locator('[data-node-type="text"]', has_text="Edited at 320px.")).last
            source_id = canvas_node.get_attribute("data-node-id")
            target_id = target_node.get_attribute("data-node-id")
            assert source_id and target_id and source_id != target_id, "Canvas source and destination need distinct ids"
            source_bounds = canvas_node.bounding_box()
            target_bounds = target_node.bounding_box()
            assert source_bounds and target_bounds, "Could not measure canvas layer bounds"
            start = (source_bounds["x"] + source_bounds["width"] / 2, source_bounds["y"] + source_bounds["height"] / 2)
            end = (target_bounds["x"] + target_bounds["width"] / 2, target_bounds["y"] + target_bounds["height"] / 2)
            page.mouse.move(*start)
            page.mouse.down()
            page.mouse.move(*end, steps=8)
            page.mouse.up()
            page.wait_for_timeout(250)
            page.get_by_role("button", name=re.compile("Edited at 320px")).wait_for()
            source_layer = page.locator(f'[data-node-id="{source_id}"]')
            assert source_layer.count() == 1, "Dragged layer disappeared"
            parent_id = source_layer.evaluate("node => node.parentElement?.closest('[data-node-id]')?.dataset.nodeId")
            assert parent_id == target_id, f"Canvas drop did not reparent the layer into its target: {parent_id} != {target_id}"
            page.wait_for_timeout(900)
            page.reload(wait_until="domcontentloaded")
            page.get_by_role("tab", name=re.compile("Browser smoke page")).click()
            page.get_by_text("Edited at 320px.", exact=True).first.wait_for()
            source_layer = page.locator(f'[data-node-id="{source_id}"]')
            source_layer.wait_for()
            saved_parent_id = source_layer.evaluate("node => node.parentElement?.closest('[data-node-id]')?.dataset.nodeId")
            assert saved_parent_id == target_id, f"Canvas reparent was not persisted: {saved_parent_id} != {target_id}"
            report["drag_drop"]["canvas_node_drag_supported"] = True
            report["drag_drop"]["canvas_node_reparented"] = {"source": source_id, "target": target_id, "persisted": True}

            source_root = locate_eight_screen_source(source_root)
            html_file = source_root / "index.html"
            css_files = sorted((source_root / "css").glob("*.css"))
            assert html_file.is_file() and css_files, "Eight-screen HTML/CSS source bundle is missing"
            page.get_by_role("button", name="Import an HTML page").click()
            dialog = page.get_by_test_id("html-import")
            bundle = [{"name": html_file.name, "mimeType": "text/html", "buffer": html_file.read_bytes()}]
            bundle.extend({"name": css.name, "mimeType": "text/css", "buffer": css.read_bytes()} for css in css_files)
            dialog.locator('input[type="file"]').first.set_input_files(bundle)
            page.get_by_test_id("html-import-region-search").fill("money dark")
            page.get_by_text("Showing 1 of", exact=False).wait_for()
            matching_regions = dialog.locator(".hi-region")
            matching_labels = matching_regions.all_inner_texts()
            money_dark_region = matching_regions.filter(has_text=re.compile(r"money dark", re.IGNORECASE))
            assert money_dark_region.count() == 1, f"Expected one money dark region, found {matching_labels}"
            region_label = money_dark_region.first.inner_text()
            money_dark_region.first.click()
            page.get_by_test_id("html-import-name").fill("Money dark source import")
            page.get_by_test_id("html-import-import").click()
            page.get_by_role("tab", name=re.compile("Money dark source import", re.IGNORECASE)).wait_for()
            page.get_by_text("Maya Lindqvist", exact=True).first.wait_for()
            report["workspace"]["source_import"] = {
                "html": str(html_file),
                "css": [str(css) for css in css_files],
                "query": "money dark",
                "selected_region": region_label,
                "imported_screen_visible": "Maya Lindqvist",
            }
            # Capture a durable state from the saved synthetic page; the source
            # import is recent and may still be autosaving while this journey
            # proceeds. This lets the smoke distinguish BAS admission failures
            # from a missing state in an unsaved import.
            page.get_by_role("tab", name=re.compile("Browser smoke page")).click()
            page.wait_for_function(
                "document.body.innerText.includes('SAVED · REVISION') && !document.body.innerText.includes('SAVING…')"
            )
            page.get_by_role("tab", name="Checks").click()

            capture = page.get_by_role("button", name=re.compile("Capture state"))
            capture_available = capture.count() > 0
            capture_enabled = capture.is_enabled() if capture_available else False
            capture_title = capture.get_attribute("title") if capture_available else None
            capture_response = None
            if capture_enabled:
                try:
                    with page.expect_response(
                        lambda response: "captureworkspacestate" in response.url.casefold(),
                        timeout=180_000,
                    ) as capture_response_info:
                        capture.first.click()
                    response = capture_response_info.value
                    capture_response = {"status": response.status, "url": response.url}
                    try:
                        body = response.json()
                        capture_response["body"] = json.dumps(body)[:500]
                        capture_response["capture_id"] = body.get("captureId", "")
                        capture_response["evidence_present"] = bool(body.get("evidence"))
                    except Exception as error:
                        capture_response["body"] = f"response body unavailable: {type(error).__name__}"
                except PlaywrightTimeoutError:
                    pass
            report["capture_drift"] = {
                "compare_control": page.get_by_role("button", name=re.compile("Compare captures")).count() > 0,
                "capture_message": page.locator("[role=alert]").all_inner_texts(),
                "capture_controls": page.locator("body").inner_text().find("Capture this state") >= 0,
                "capture_button_available": capture_available,
                "capture_button_enabled": capture_enabled,
                "capture_button_title": capture_title,
                "capture_requests": capture_requests,
                "capture_api_response": capture_response,
            }
            if capture_response is None:
                wait_gap = (
                    "CaptureWorkspaceState was requested but returned no response before the 180-second wait; "
                    if capture_requests
                    else "No CaptureWorkspaceState request was issued; "
                )
                report.setdefault("gaps", []).append(
                    wait_gap + "live capture and visual-drift history remain unverified."
                )
            elif capture_response["status"] != 200 or not capture_response.get("evidence_present"):
                report.setdefault("gaps", []).append(
                    f"State capture did not return evidence (HTTP {capture_response['status']}); visual-drift history remains unverified."
                )
        finally:
            try:
                if created_id:
                    rpc(api_url, "DeleteWorkspace", {"id": created_id})
                    report["workspace"]["deleted_via_api"] = True
            finally:
                browser.close()
        assert not errors, f"Browser errors: {errors}"
        report["page_errors"] = errors
        report["status"] = "partial" if report.get("gaps") else "passed"
    return report


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--ui-url", default="http://127.0.0.1:23906")
    parser.add_argument("--api-url", default="http://127.0.0.1:17193")
    parser.add_argument(
        "--source-root",
        type=Path,
        help="Directory containing the moved eight-screen index.html and css/ bundle.",
    )
    args = parser.parse_args()
    print(json.dumps(run(args.ui_url, args.api_url, args.source_root), indent=2))


if __name__ == "__main__":
    main()
