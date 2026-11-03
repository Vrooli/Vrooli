"""Operate optional dependencies in the real packaged Web Console renderer."""
import json
import os
import pathlib
import sys
from playwright.sync_api import sync_playwright, expect

endpoint, mode = sys.argv[1:3]
os.set_blocking(sys.stdout.fileno(), True)

def summary(status):
    # The signed service manifest can be large. Keep stdout bounded; the runtime
    # already verified it and these are the observed fields this check consumes.
    return {
        "installed": [
            {"state": row["state"], "generation": row["generation"],
             "reason": row.get("reason", ""),
             "activation_disabled": row.get("activation_disabled", False),
             "previous": {"version": row["previous"]["signed"]["entry"]["version"], "generation": row["previous"]["generation"]} if row.get("previous") else None,
             "signed": {"entry": {key: row["signed"]["entry"][key] for key in ("id", "owner", "version", "sha256", "tree_sha256")}}}
            for row in status["installed"]
        ]
    }

with sync_playwright() as p:
    browser = p.chromium.connect_over_cdp(endpoint)
    page = next(page for context in browser.contexts for page in context.pages if page.url.startswith("http://127.0.0.1:"))
    viewport = os.environ.get("J11_VIEWPORT", "desktop")
    if viewport == "phone":
        page.set_viewport_size({"width":390,"height":844})
    if mode == "update-startup-failure":
        # This release was published after the same-home restart. Reload the
        # renderer so restored Settings cannot retain the earlier catalog query.
        page.reload()
    page.wait_for_function("window.desktop?.components", timeout=30000)
    try:
        settings = page.get_by_test_id("empty-workspace-settings").or_(page.get_by_test_id("nav-settings")).first
        if mode.startswith("inspect"):
            page.wait_for_function("document.body.innerText.length > 0", timeout=30000)
            if mode == "inspect-settings":
                settings.click()
                page.get_by_test_id("settings-tab-integrations").click()
                optional = page.get_by_role("region", name="Optional dependencies")
                expect(optional).to_be_visible(timeout=30000)
                optional.scroll_into_view_if_needed()
            print(json.dumps(page.evaluate("""() => ({url: location.href, text:document.body.innerText, buttons:[...document.querySelectorAll('button')].map(b=>({text:b.innerText,label:b.getAttribute('aria-label'),title:b.title,testId:b.getAttribute('data-testid')}))})""")), flush=True)
            sys.exit(0)
        full_app_url = None
        before = page.evaluate("window.desktop.components.list()")
        if not page.get_by_test_id("optional-component-audio-tools").is_visible():
            settings.click()
            # Settings restores the selected page. On a phone, an already
            # selected Integrations page replaces the tab list entirely.
            tab = page.get_by_test_id("settings-tab-integrations")
            target = page.get_by_test_id("optional-component-audio-tools")
            expect(tab.or_(target).first).to_be_visible(timeout=30000)
            if not target.is_visible():
                tab.click()
        row = page.get_by_test_id("optional-component-audio-tools")
        expect(row).to_be_visible(timeout=30000)
        if mode == "install":
            assert before["installed"] == [], before
            row.get_by_role("button", name="Add to this app", exact=True).click()
            expect(row.get_by_role("status")).to_have_text("Added · starts next time", timeout=60000)
        elif mode == "installed":
            expect(row.get_by_role("status")).to_have_text("Installed", timeout=30000)
        elif mode == "both-offered":
            expect(row.get_by_role("button", name="Add to this app", exact=True)).to_be_visible()
            link = row.get_by_role("link", name="Get the full app", exact=True)
            expect(link).to_be_visible()
            full_app_url = link.get_attribute("href")
            assert before["installed"] == [], before
        elif mode == "update-failure":
            row.get_by_role("button", name="Update to 0.0.2", exact=True).click()
            expect(page.get_by_text("The dependency could not be added or updated. Your current installation is unchanged.", exact=True)).to_be_visible(timeout=60000)
            expect(row.get_by_role("status")).to_have_text("Installed")
        elif mode == "update-startup-failure":
            row.get_by_role("button", name="Update to 0.0.3", exact=True).click()
            expect(row.get_by_role("status")).to_have_text("Added · starts next time", timeout=60000)
        elif mode == "activation-disabled":
            expect(row.get_by_role("status")).to_have_text("Needs update", timeout=30000)
            expect(row.get_by_text("The dependency could not start and no healthy compatible version is available. Update or remove it.", exact=True)).to_be_visible()
        elif mode in ("remove-keep", "remove-delete"):
            row.get_by_role("button", name="Remove", exact=True).click()
            choices = row.get_by_role("group", name="Remove dependency")
            expect(choices.get_by_role("button", name="Remove and keep data", exact=True)).to_be_visible()
            expect(choices.get_by_role("button", name="Remove and delete data", exact=True)).to_be_visible()
            if os.environ.get("J11_EVIDENCE"):
                page.screenshot(path=str(pathlib.Path(os.environ["J11_EVIDENCE"], "j11-remove-choices" + ("-phone" if viewport == "phone" else "") + ".png")), full_page=True)
            choices.get_by_role("button", name="Cancel", exact=True).click()
            assert len(page.evaluate("window.desktop.components.list()")['installed']) == 1
            row.get_by_role("button", name="Remove", exact=True).click()
            choices.get_by_role("button", name="Remove and keep data" if mode == "remove-keep" else "Remove and delete data", exact=True).click()
            expect(row.get_by_role("button", name="Add to this app", exact=True)).to_be_visible(timeout=60000)
        else:
            raise AssertionError(mode)
        after = page.evaluate("window.desktop.components.list()")
        print(json.dumps({"before": summary(before), "after": summary(after), "full_app_url": full_app_url}), flush=True)
    finally:
        if os.environ.get("J11_EVIDENCE"):
            page.screenshot(path=str(pathlib.Path(os.environ["J11_EVIDENCE"], "j11-" + mode + ("-phone" if viewport == "phone" else "") + ".png")), full_page=True)
        browser.close()
