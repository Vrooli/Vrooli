"""Operate the real optional Agent Manager slice and Web Console usage bars."""
import json
import os
import pathlib
import re
import sys
from playwright.sync_api import sync_playwright, expect

endpoint, mode = sys.argv[1:3]
os.set_blocking(sys.stdout.fileno(), True)
with sync_playwright() as p:
    browser = p.chromium.connect_over_cdp(endpoint)
    page = next(page for context in browser.contexts for page in context.pages if page.url.startswith("http://127.0.0.1:"))
    page.wait_for_function("window.desktop?.components", timeout=30000)
    responses = []
    try:
        if mode == "install":
            before = page.evaluate("window.desktop.components.list()")
            assert before["installed"] == [], before
            page.get_by_test_id("empty-workspace-settings").or_(page.get_by_test_id("nav-settings")).first.click()
            page.get_by_test_id("settings-tab-integrations").click()
            row = page.get_by_test_id("optional-component-agent-manager-usage")
            expect(row).to_be_visible(timeout=30000)
            row.get_by_role("button", name="Add to this app", exact=True).click()
            expect(row.get_by_role("status")).to_have_text("Added · starts next time", timeout=60000)
            after = page.evaluate("window.desktop.components.list()")
            assert len(after["installed"]) == 1, after
            entry = after["installed"][0]["signed"]["entry"]
            assert entry["id"] == "agent-manager-usage" and entry["owner"] == "agent-manager", entry
            print(json.dumps({"installed":entry["id"], "archive_sha256":entry["sha256"]}), flush=True)
        elif mode == "usage":
            def remember(response):
                if "CapabilitiesService/AgentUsage" in response.url and response.status == 200:
                    data = response.json()
                    if data.get("agentId") == "claude":
                        responses.append(data)
            page.on("response", remember)
            page.get_by_test_id("new-terminal-button").click(timeout=30000)
            page.get_by_test_id("launcher-mode-agent").click()
            claude = page.locator('[data-testid="launcher-agent-grid"] [data-capability-id="claude"]')
            expect(claude.locator('[data-usage-row="reading"]')).to_have_count(2, timeout=45000)
            expect(claude.get_by_role("progressbar", name=re.compile("23.*reset", re.I))).to_be_visible()
            expect(claude.get_by_role("progressbar", name=re.compile("11.*reset", re.I))).to_be_visible()
            assert responses, "No real Web Console usage response observed"
            reading = responses[-1]
            assert reading["match"] == "likely" and reading["source"] == "Claude Code statusline", reading
            assert sorted((int(w["windowMinutes"]), w["usedPercent"]) for w in reading["windows"]) == [(300,23),(10080,11)], reading
            print(json.dumps({"usage":reading,"bars":claude.get_by_role("progressbar").evaluate_all("els => els.map(e => ({label:e.getAttribute('aria-label'),value:e.getAttribute('aria-valuenow')}))")}), flush=True)
        else:
            raise AssertionError(mode)
    except Exception:
        print(json.dumps({"responses": responses[-4:], "visible_text": page.locator("body").inner_text()[-6000:]}), file=sys.stderr, flush=True)
        raise
    finally:
        if os.environ.get("J11_EVIDENCE"):
            page.screenshot(path=str(pathlib.Path(os.environ["J11_EVIDENCE"], "j11-usage-"+mode+".png")),full_page=True)
        browser.close()
