"""Observe real packaged application state before and after electron-updater."""
import json
import os
import pathlib
import sys
from playwright.sync_api import sync_playwright
endpoint, mode = sys.argv[1:3]
with sync_playwright() as p:
    browser = p.chromium.connect_over_cdp(endpoint)
    page = next(page for context in browser.contexts for page in context.pages if page.url.startswith("http://127.0.0.1:"))
    page.wait_for_function("window.desktop && localStorage.getItem('wc.deviceId')", timeout=30000)
    if mode == "before":
        page.evaluate("window.desktop.storage.writeFile('journey-note.txt', 'My saved desktop note')")
    state = page.evaluate("""async () => ({
        origin: location.origin,
        deviceId: localStorage.getItem('wc.deviceId'),
        note: await window.desktop.storage.readTextFile('journey-note.txt'),
        info: await window.desktopAPI.system.info()
    })""")
    print(json.dumps(state), flush=True)
    if os.environ.get("J10_EVIDENCE"):
        page.screenshot(path=str(pathlib.Path(os.environ["J10_EVIDENCE"], "j10-update-" + mode + ".png")), full_page=True)
    if mode == "after":
        # Shut down the successor through Electron's own browser lifecycle.
        # Browser.close can stop Chromium before it returns a CDP response. Queue
        # the command without waiting for that response so normal shutdown does
        # not hold the Go parent open until its process timeout.
        session = browser.new_browser_cdp_session()
        session._impl_obj._channel.send_no_reply(
            "send", None, {"method": "Browser.close", "params": None}
        )
    else:
        browser.close()
