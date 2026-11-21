"""Exercise native touch in the running Web Console. Requires Python Playwright, Chrome and an active local session."""
import os
from pathlib import Path
from tempfile import TemporaryDirectory
from urllib.parse import urlencode

from playwright.sync_api import sync_playwright


DOCUMENT = """<!doctype html><meta name="viewport" content="width=device-width,initial-scale=1">
<style>body{margin:0;min-width:850px;min-height:1600px}summary,label,a{display:block;padding:15px}details{margin-top:120px}</style>
<details><summary>Choose projects</summary><label><input type="checkbox">Include subscriptions</label></details>
<a href="./next.md">Read the evidence</a>"""


def gesture(cdp, kind, points):
    cdp.send("Input.dispatchTouchEvent", {
        "type": kind,
        "touchPoints": [{"x": x, "y": y, "id": i} for i, (x, y) in enumerate(points)],
    })


def exercise(page, url):
    page.goto(url)
    frame = page.locator('iframe[sandbox="allow-scripts"]:visible').last
    content = frame.content_frame
    disclosure = content.get_by_text("Choose projects", exact=True)

    def ready():
        disclosure.wait_for()
        child = frame.element_handle().content_frame()
        child.wait_for_load_state("load")
        page.wait_for_function("""() => Array.from(document.querySelectorAll('[data-wc-html-inspection-scale] iframe'))
          .some(frame => {const r = frame.getBoundingClientRect(); return r.width > 0 && Math.abs(r.y) < .5;})""")

    ready()
    # Compatibility mouse events on an offset cross-process frame may land above the touch.
    disclosure.tap()
    checkbox = content.get_by_label("Include subscriptions")
    checkbox.tap()
    assert checkbox.is_checked(), "native tap must activate the intended control exactly once"
    source = page.get_by_role("button", name="Source", exact=True)
    source.tap()
    page.get_by_role("button", name="Preview", exact=True).tap()
    ready()
    content.get_by_text("Read the evidence").tap()
    page.get_by_role("heading", name="Touch evidence").wait_for()
    page.get_by_test_id("file-preview-back").tap()
    ready()
    cdp = page.context.new_cdp_session(page)
    grabber = page.get_by_test_id("messages-file-viewer-panel.grabber").bounding_box()
    assert grabber["y"] >= frame.bounding_box()["height"], "the grabber must leave the document unobstructed"
    x = grabber["x"] + grabber["width"] / 2
    y = grabber["y"] + grabber["height"] / 2
    gesture(cdp, "touchStart", [(x, y)])
    for step in range(1, 9):
        gesture(cdp, "touchMove", [(x, y + (page.viewport_size["height"] - 3 - y) * step / 8)])
        page.wait_for_timeout(16)
    gesture(cdp, "touchEnd", [])
    frame.wait_for(state="hidden")

    page.goto("about:blank")
    page.goto(url)
    ready()
    # The backdrop that used to be above the drawer is now visible along its bottom.
    page.touchscreen.tap(20, page.viewport_size["height"] - 12)
    frame.wait_for(state="hidden")

    page.goto("about:blank")
    page.goto(url)
    ready()
    child = frame.element_handle().content_frame()
    # Drag across overflowing content using the browser's native scroll gesture.
    for path in [[(200, 390), (200, 350), (200, 300), (200, 235), (200, 165)],
                 [(310, 300), (270, 300), (220, 300), (155, 300), (85, 300)]]:
        gesture(cdp, "touchStart", path[:1])
        for point in path[1:]:
            gesture(cdp, "touchMove", [point])
            page.wait_for_timeout(30)
        gesture(cdp, "touchEnd", [])
        page.wait_for_timeout(150)
    child.wait_for_function("scrollX > 0 && scrollY > 0")
    gesture(cdp, "touchStart", [(130, 230), (220, 230)])
    for distance in (60, 75, 95):
        gesture(cdp, "touchMove", [(175-distance, 230), (175+distance, 230)])
        page.wait_for_timeout(30)
    gesture(cdp, "touchEnd", [])
    page.wait_for_function("Number(document.querySelector('[data-wc-html-inspection-scale]')?.dataset.wcHtmlInspectionScale) > 1.2")


if __name__ == "__main__":
    with TemporaryDirectory(prefix="web-console-html-touch-") as directory, sync_playwright() as pw:
        fixture = Path(directory) / "report.html"
        fixture.write_text(DOCUMENT)
        fixture.with_name("next.md").write_text("# Touch evidence\n")
        query = {"file": str(fixture)}
        if session := os.environ.get("WC_WEB_CONSOLE_SESSION_ID"):
            query["session"] = session
        url = os.environ.get("WC_UI_URL", "http://localhost:21233").rstrip("/") + "/?" + urlencode(query)
        with pw.chromium.launch(executable_path=os.environ.get("CHROME_EXECUTABLE", "/usr/bin/google-chrome")) as browser:
            phone = browser.new_page(viewport={"width": 390, "height": 844}, has_touch=True, is_mobile=True, device_scale_factor=2)
            exercise(phone, url)
            desktop = browser.new_page(viewport={"width": 1280, "height": 840})
            desktop.goto(url)
            frame = desktop.locator('iframe[sandbox="allow-scripts"]:visible').last
            frame.content_frame.get_by_text("Choose projects", exact=True).click()
            assert frame.content_frame.get_by_label("Include subscriptions").is_visible()
            assert frame.bounding_box()["y"] > 0, "desktop keeps the controls above its preview"
            print("Native phone controls, trusted file links, Back, two-axis scroll, pinch, Close and desktop passed.")
