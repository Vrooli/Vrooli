"""Operate the actual AppImage renderer over its target-owned CDP connection."""
import json
import pathlib
import re
import sys
from playwright.sync_api import sync_playwright, expect

endpoint, renderer_url, check, evidence = sys.argv[1:]
with sync_playwright() as p:
    browser = p.chromium.connect_over_cdp(endpoint)
    pages = [page for context in browser.contexts for page in context.pages]
    page = next(page for page in pages if page.url.startswith(renderer_url))
    response = page.request.post(renderer_url.rstrip("/") + "/vrooli.web_console.v1.capabilities.CapabilitiesService/AgentUpdate", data={"targetId":"local", "agentId":"codex"})
    print("AgentUpdate", response.status, response.text()[:3000], flush=True)
    pathlib.Path(evidence, "j9-api-check.json").write_text(response.text())
    try:
        page.get_by_role("button", name=re.compile("(New session|Add session|Create.*session)", re.I)).first.click(timeout=30000)
        page.get_by_test_id("launcher-mode-agent").click()
        page.get_by_test_id("launcher-manage-agents").click()
        page.get_by_test_id("launcher-manage-open-codex").click(timeout=30000)
        status = page.get_by_test_id("launcher-agent-status-codex")
        expect(status).not_to_have_attribute("data-harness-state", "checking", timeout=60000)
        if check == "agent":
            expect(status).to_have_attribute("data-harness-state", "found", timeout=60000)
            expect(page.get_by_test_id("launcher-agent-three-facts")).to_have_attribute("data-sign-in-state", "signed_out", timeout=60000)
        else:
            # Repository lifecycle controls have no owner in a bundle. Manage's
            # read-only refresh and doctor still execute the native lifecycle.
            expect(page.get_by_text(re.compile(r"vrooli (scenario|resource|setup)"))).to_have_count(0)
            expect(page.get_by_role("button", name=re.compile(r"Start (Audio Tools|Bridge)"))).to_have_count(0)
            doctor = page.get_by_test_id("launcher-doctor-codex")
            expect(doctor).to_be_enabled(timeout=60000)
            doctor.click()
            expect(page.get_by_test_id("launcher-action-done")).to_be_visible(timeout=60000)
            expect(page.get_by_test_id("launcher-action-error")).to_have_count(0)
    finally:
        out = pathlib.Path(evidence)
        out.mkdir(parents=True, exist_ok=True)
        page.screenshot(path=str(out / f"j9-{check}.png"), full_page=True)
        browser.close()
