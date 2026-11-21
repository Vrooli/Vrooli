"""Real served UI, real API and native restart. Provider limit capture is synthetic."""
import os
import pathlib
import sys
from playwright.sync_api import sync_playwright, expect

url, session, evidence = sys.argv[1:]
with sync_playwright() as p:
    browser = p.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True,
                               args=["--no-sandbox", "--disable-dev-shm-usage"])
    for name, width, height in [("desktop", 1440, 1000), ("phone", 390, 844)]:
        context = browser.new_context(viewport={"width": width, "height": height},
                                      extra_http_headers={"Authorization": "LocalSession " + os.environ["BACKUP_LOCAL_TOKEN"]})
        page = context.new_page()
        page.on("pageerror", lambda error: print("PAGE_ERROR", error))
        page.goto(url + "?session=" + session)
        status = page.get_by_test_id("coding-backup-status").filter(has_text="Plan limit")
        expect(status.first).to_be_visible(timeout=30000)
        status.first.click()
        dialog = page.get_by_test_id("coding-backup-dialog")
        expect(dialog).to_be_visible()
        expect(dialog).to_contain_text("new session with a summary")
        expect(dialog).to_contain_text("5,000 credits total")
        dialog.evaluate("async el => { await Promise.all(el.getAnimations({subtree:true}).filter(a => a.effect.getComputedTiming().iterations !== Infinity).map(a => a.finished.catch(() => {}))); }")
        expect(dialog.get_by_role("button", name="Continue on credits", exact=True)).to_be_in_viewport()
        page.screenshot(path=str(pathlib.Path(evidence, f"restart-{name}.png")), full_page=True)
        if name == "desktop":
            with page.expect_response(lambda r: r.url.endswith("/api/v1/coding-backup/restart") and r.request.method == "POST") as response:
                dialog.get_by_role("button", name="Continue on credits", exact=True).click()
            assert response.value.status == 200, response.value.text()
            created = response.value.json()["id"]
            print("CREATED_SESSION=" + created)
            expect(page.get_by_test_id("coding-backup-status").filter(has_text="Vrooli credits").first).to_be_visible(timeout=30000)
        context.close()
    browser.close()
