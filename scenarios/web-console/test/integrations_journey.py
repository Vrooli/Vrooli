"""J14 operates the production panel and real registry API; no browser mocks."""
import json
import pathlib
import sys
from playwright.sync_api import sync_playwright, expect

url, check, evidence = sys.argv[1:]
root = pathlib.Path(__file__).resolve().parents[3]

def snapshot(page):
    response = page.request.post(url + "/vrooli.web_console.v1.capabilities.CapabilitiesService/Get", data={})
    assert response.ok, response.text()
    result = response.json()["capabilities"]
    assert result
    return result

def open_panel(page):
    page.goto(url)
    page.get_by_test_id("workspace-root").wait_for(timeout=30000)
    page.get_by_test_id("empty-workspace-settings").click(timeout=10000)
    page.get_by_test_id("settings-tab-integrations").click(timeout=30000)
    panel = page.get_by_test_id("integrations-panel")
    expect(panel.get_by_role("button", name="By capability", exact=True)).to_be_visible(timeout=30000)
    return panel

with sync_playwright() as p:
    browser = p.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True, args=["--no-sandbox"])
    for width, height in [(1440, 1000), (390, 844)]:
        page = browser.new_page(viewport={"width":width,"height":height})
        panel = open_panel(page)
        deps = snapshot(page)
        expect(panel.locator('[data-rcl-integrations-summary-title]')).not_to_have_text("Checking…", timeout=30000)
        if check == "TestJ14_CapabilityAndDependencyViewsAgree":
            for dep in deps:
                for use in dep.get("uses", []):
                    expect(panel.get_by_text(use["name"], exact=True).first).to_be_visible()
            panel.get_by_role("button", name="By dependency", exact=True).click()
            for dep in deps:
                uses = dep.get("uses", [])
                for use in uses:
                    expect(panel.get_by_text(use["description"], exact=False).first).to_be_visible()
        elif check == "TestJ14_InstalledVersionsShown":
            installed = [d for d in deps if d.get("installation",{}).get("version")]
            assert installed, "No real installed dependency versions were observed"
            for dep in installed:
                assert dep["installation"]["location"] in ["machine","bundled"]
                expect(panel.get_by_text("Version " + dep["installation"]["version"], exact=False).first).to_be_visible()
            assert not next(d for d in deps if d["id"]=="openrouter").get("installation"), "Hosted API acquired an installation version"
        elif check == "TestJ14_EachIntegrationSaysWhatItDoes":
            assert all(d.get("uses") for d in deps), "A dependency lacks consumer purpose"
            for dep in deps:
                for use in dep["uses"]:
                    assert len(use["description"]) > 15
                    expect(panel.get_by_text(use["description"], exact=False).first).to_be_visible()
        elif check == "TestJ14_AgenticFeaturesShown":
            agents = next(d for d in deps if d["dependencySlug"]=="agent-manager")
            assert {u["id"] for u in agents["uses"]} >= {"agent-runs","agent-history","agent-usage"}
            for use in agents["uses"]:
                expect(panel.get_by_text(use["name"], exact=True).first).to_be_visible()
        elif check == "TestJ14_OldMappingDeleted":
            for relative in ["scenarios/web-console/ui/src/components/IntegrationsPanel.tsx", "scenarios/git-control-tower/ui/src/components/SettingsTabIntegrations.tsx"]:
                source = (root / relative).read_text()
                assert all(name not in source for name in ["ITEM_COPY", "APP_COPY", "function capabilityItem", "function versionOf"]), relative
                assert "dependencies={caps}" in source
            expect(panel.get_by_role("button", name="By dependency", exact=True)).to_be_visible()
        elif check == "TestJ14_CreditsRowShown":
            expect(panel.get_by_text("Vrooli credits",exact=True)).to_be_visible()
            expect(panel.get_by_text("Sign in and select a business account",exact=False)).to_be_visible()
        elif check == "TestJ14_SecondCapabilitySamePattern":
            # Audio and managed-agent uses are distinct owners using the same
            # declarative registry and shared projection, with no UI ID switch.
            for slug in ["audio-tools", "agent-manager"]:
                dep = next(d for d in deps if d["dependencySlug"]==slug)
                assert dep["uses"] and all(u["id"] != slug for u in dep["uses"])
                for use in dep["uses"]:
                    expect(panel.get_by_text(use["name"],exact=True).first).to_be_visible()
            panel.get_by_role("button",name="By dependency",exact=True).click()
            expect(panel.get_by_text("Dictation, read-aloud and wake word",exact=False)).to_be_visible()
            expect(panel.get_by_text("Launch and follow managed agents",exact=False)).to_be_visible()
        else:
            raise AssertionError("Unknown check " + check)
        assert page.evaluate("document.documentElement.scrollWidth <= window.innerWidth"), "Horizontal page overflow"
        page.screenshot(path=str(pathlib.Path(evidence)/f"{check}-{width}.png"))
        panel.get_by_text("Launch and follow managed agents", exact=False).first.scroll_into_view_if_needed()
        page.screenshot(path=str(pathlib.Path(evidence)/f"{check}-{width}-rows.png"))
        if check == "TestJ14_InstalledVersionsShown":
            version = installed[0]["installation"]["version"]
            hint = panel.get_by_text("Version " + version, exact=False).first
            hint_id = hint.get_attribute("id")
            assert hint_id, "Installed-version row has no accessible description"
            panel.locator(f'button[aria-describedby~="{hint_id}"]').click()
            panel.locator('[data-rcl-integrations-facts] > summary').click()
            expect(panel.get_by_text("On this machine", exact=True)).to_be_visible()
            page.screenshot(path=str(pathlib.Path(evidence)/f"{check}-{width}-detail.png"))
        page.close()
    browser.close()
print("Operated",check,"on desktop and phone using the production API and UI")
