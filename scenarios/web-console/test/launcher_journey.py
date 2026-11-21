"""Operate the built Web Console in real Chrome against its isolated real API."""
import hashlib
import pathlib
import json
import os
import time
import re
import sys
from playwright.sync_api import sync_playwright, expect

url, check, evidence = sys.argv[1:]

def open_drawer(page):
    page.get_by_role("button", name=re.compile("(New session|Add session|Create.*session)", re.I)).first.click(timeout=30000)
    expect(page.get_by_test_id("launcher-mode-shell")).to_be_visible()

def order_check(page):
    page.get_by_test_id("launcher-mode-agent").click()
    page.get_by_test_id("launcher-manage-agents").click()
    custom = page.get_by_test_id("launcher-order-custom")
    expect(custom).to_be_visible()
    if custom.get_attribute("aria-selected") != "true":
        with page.expect_response(lambda r: "UpdateAgentPreferences" in r.url and r.status == 200):
            custom.click()
    rows = page.locator('[data-testid^="launcher-agent-card-"]')
    expect(rows).to_have_count(5, timeout=30000)
    original = rows.evaluate_all("els => els.map(e => e.dataset.testid)")
    with page.expect_response(lambda r: "UpdateAgentPreferences" in r.url and r.status == 200):
        page.locator('[data-testid^="launcher-agent-grip-"]').first.press("Alt+ArrowDown")
    expected = [original[1], original[0], *original[2:]]
    expect(rows.first).to_have_attribute("data-testid", expected[0])
    page.reload()
    open_drawer(page)
    page.get_by_test_id("launcher-mode-agent").click()
    page.get_by_test_id("launcher-manage-agents").click()
    expect(rows.first).to_have_attribute("data-testid", expected[0])
    expect(rows).to_have_count(len(expected), timeout=30000)
    actual = rows.evaluate_all("els => els.map(e => e.dataset.testid)")
    assert actual == expected, (expected, actual)
    with page.expect_response(lambda r: "UpdateAgentPreferences" in r.url and r.status == 200):
        page.get_by_test_id("launcher-order-recent").click()
    expect(page.get_by_test_id("launcher-order-recent")).to_have_attribute("aria-selected", "true")
    page.reload()
    open_drawer(page)
    page.get_by_test_id("launcher-mode-agent").click()
    page.get_by_test_id("launcher-manage-agents").click()
    expect(page.get_by_test_id("launcher-order-recent")).to_have_attribute("aria-selected", "true")
    with page.expect_response(lambda r: "UpdateAgentPreferences" in r.url and r.status == 200):
        page.get_by_test_id("launcher-order-custom").click()
    expect(rows.first).to_have_attribute("data-testid", expected[0])

def height_check(page):
    readings = []
    page.on("response", lambda r: readings.append(r) if "/AgentUsage" in r.url else None)
    page.evaluate("""() => {
      window.cardSamples = [];
      const sample = () => {
        const cards = [...document.querySelectorAll('[data-testid="launcher-agent-grid"] [role="radio"]')];
        if (cards.length) window.cardSamples.push(cards.map(e => ({id:e.dataset.capabilityId, h:e.getBoundingClientRect().height})));
        window.cardFrame = requestAnimationFrame(sample);
      }; sample();
    }""")
    page.get_by_test_id("launcher-mode-agent").click()
    expect(page.get_by_test_id("launcher-agent-grid")).to_be_visible()
    page.wait_for_timeout(15000)
    assert readings, "No real AgentUsage responses were observed"
    samples = page.evaluate("() => { cancelAnimationFrame(window.cardFrame); return window.cardSamples; }")
    assert len(samples) > 10
    heights = [card["h"] for sample in samples for card in sample]
    assert max(heights) - min(heights) <= 1, (min(heights), max(heights))
    print(f"Measured {len(samples)} animation frames across delayed descriptor/usage loads; card height {min(heights)}..{max(heights)}")

def context_check(page):
    page.get_by_test_id("launcher-mode-agent").click()
    fixture = json.loads(pathlib.Path(__file__).with_name("launcher_context_windows.json").read_text())
    expected_windows = {m["slug"]: m["context_window"] for m in fixture["codex"]["models"]}
    expected_windows.update({m: v["context_window_tokens"] for m, v in fixture["claude"]["models"].items()})
    for agent in ["codex", "claude-code"]:
        card = page.locator(f'[data-testid="launcher-agent-grid"] [data-capability-id="{agent}"]')
        if agent == "claude-code" and card.count() == 0:
            card = page.locator('[data-testid="launcher-agent-grid"] [data-capability-id="claude"]')
        card.click()
        expect(page.get_by_test_id("launcher-model-loading")).to_have_count(0)
        picker = page.get_by_test_id("launcher-model-choice")
        expect(picker).to_be_visible(timeout=30000)
        # Read the same authoritative model endpoint as the UI. Values remain
        # independently checked against the recorded resource observations.
        response = page.request.post(url + "/vrooli.web_console.v1.modelselection.ModelSelectionService/GetRoleModelChoices", data={"agentId": "claude" if agent == "claude-code" else agent, "targetId":"local"})
        assert response.ok, response.text()
        choices = response.json().get("choices", [])
        assert choices, response.text()
        policy_path = pathlib.Path(__file__).resolve().parents[3]/"resources"/agent/"model-policy.json"
        aliases = json.loads(policy_path.read_text()).get("model_aliases", {})
        checked = 0
        for choice in choices:
            model = choice["model"]
            tokens = int(choice.get("contextWindowTokens", 0))
            canonical = aliases.get(model, {}).get("canonical_model", model)
            if "/" in canonical:
                canonical = canonical.split("/",1)[1]
            expected = expected_windows.get(canonical, expected_windows.get(model,0))
            assert tokens == expected, (agent,model,canonical,tokens,expected)
            value = json.dumps([choice.get("source", ""), model], separators=(",", ":"))
            if picker.evaluate("e => e.tagName") == "SELECT":
                picker.select_option(value)
            else:
                picker.click()
                label = choice.get("displayName", model)
                if choice.get("isDefault"):
                    label += " · Default"
                page.get_by_role("option", name=label, exact=True).click()
            shown = page.get_by_test_id("launcher-context-window")
            expect(shown).to_be_visible()
            if tokens:
                formatted = f"{tokens / 1000000:g}M" if tokens >= 1000000 else f"{tokens / 1000:g}K"
                expect(shown).to_contain_text(formatted)
            else:
                expect(shown).to_contain_text(re.compile("unknown", re.I))
            limits = [int(value) for value in choice.get("contextLimitTokens",[])]
            assert all(value <= tokens for value in limits) if tokens else True
            if limits:
                limit_picker = page.get_by_test_id("launcher-context-limit-select")
                selected_limit = limits[-1]
                label = f"{selected_limit / 1000000:g}M" if selected_limit >= 1000000 else f"{selected_limit / 1000:g}K"
                if limit_picker.evaluate("e => e.tagName") == "SELECT":
                    values = limit_picker.locator("option").evaluate_all("els => els.map(e => e.value)")
                    assert sorted(values) == sorted(["default", *map(str,limits)])
                    limit_picker.select_option(str(selected_limit))
                else:
                    limit_picker.click()
                    expect(page.get_by_role("option")).to_have_count(len(limits)+1)
                    page.get_by_role("option",name=label,exact=True).click()
                expect(page.get_by_test_id("launcher-context-hint")).to_contain_text(label)
                if tokens:
                    expect(page.get_by_test_id("launcher-context-gauge")).to_have_attribute("data-gauge-percent",str(int(selected_limit/tokens*100+0.5)))
            expect(page.get_by_test_id("launcher-save-usage")).to_be_visible()
            checked += 1
        assert checked > 0
    for unsupported in ["opencode","grok","agy"]:
        page.locator(f'[data-testid="launcher-agent-grid"] [data-capability-id="{unsupported}"]').click()
        expect(page.get_by_test_id("launcher-context-limit")).to_have_count(0)
        expect(page.get_by_test_id("launcher-save-usage")).to_have_count(0)
    # Leave the operated numeric rows visible for independent visual review.
    page.locator('[data-testid="launcher-agent-grid"] [data-capability-id="codex"]').click()
    expect(page.get_by_test_id("launcher-context-window")).to_contain_text("K")

def usage_check(page):
    owner = os.environ["LAUNCHER_QUOTA_URL"]
    reset = int(time.time()) + 4 * 86400
    response = page.request.post(owner + "/__journey/sample?reset=1", data={"seven_day":{"used_percentage":37,"resets_at":reset}})
    assert response.status == 204, response.text()
    page.get_by_test_id("launcher-mode-agent").click()
    claude = page.locator('[data-testid="launcher-agent-grid"] [data-capability-id="claude"]')
    expect(claude.locator('[data-usage-row="reading"]')).to_have_count(1, timeout=30000)
    expect(claude.get_by_role("progressbar")).to_have_attribute("aria-label", re.compile("37.*reset", re.I))
    grok = page.locator('[data-testid="launcher-agent-grid"] [data-capability-id="grok"]')
    expect(grok.locator('[data-usage-row="reading"]')).to_have_count(0)
    response = page.request.post(owner + "/__journey/sample", data={"seven_day":{"used_percentage":37,"resets_at":reset},"five_hour":{"used_percentage":52,"resets_at":int(time.time())+7200}})
    assert response.status == 204, response.text()
    page.reload()
    open_drawer(page)
    page.get_by_test_id("launcher-mode-agent").click()
    expect(claude.locator('[data-usage-row="reading"]')).to_have_count(2, timeout=30000)
    expect(claude.get_by_role("progressbar", name=re.compile("52.*reset",re.I))).to_be_visible()
    expect(claude.get_by_role("progressbar", name=re.compile("37.*reset",re.I))).to_be_visible()
    page.get_by_test_id("launcher-manage-agents").click()
    page.get_by_test_id("launcher-manage-open-claude").click()
    expect(page.get_by_test_id("launcher-usage-source")).to_contain_text("Claude Code statusline")
    expect(page.get_by_test_id("launcher-usage-source")).to_contain_text("different account")
    action = page.get_by_test_id("launcher-action-primary-claude")
    expect(action).to_be_visible()
    bounds = action.bounding_box()
    assert bounds and bounds["x"] >= 0 and bounds["x"] + bounds["width"] <= page.viewport_size["width"], bounds

def remove_managed(page, agent):
    page.get_by_test_id(f"launcher-cleanup-preview-{agent}").click()
    expect(page.get_by_test_id("launcher-cleanup-plan")).to_be_visible(timeout=30000)
    page.get_by_test_id(f"launcher-uninstall-{agent}").click()
    expect(page.get_by_test_id("launcher-action-done")).to_contain_text("Removed the previewed",timeout=30000)

def native_install_check(page):
    full_stack = check == "TestJ8_FullDeploymentRouteOperated"
    if not full_stack:
        assert not any((pathlib.Path(p)/"vrooli").exists() for p in os.environ["PATH"].split(":")), "Desktop fixture unexpectedly has vrooli on PATH"
    all_harnesses = full_stack or check == "TestJ8_InstallUpdateUninstallDoctorPerHarness"
    agents = ["claude","codex","opencode","agy","grok"] if all_harnesses else ["claude"]
    for index,agent in enumerate(agents):
        if index:
            page.reload()
            open_drawer(page)
        page.get_by_test_id("launcher-mode-agent").click()
        page.get_by_test_id("launcher-manage-agents").click()
        page.get_by_test_id(f"launcher-manage-open-{agent}").click(timeout=30000)
        expect(page.get_by_test_id(f"launcher-agent-status-{agent}")).to_have_attribute("data-harness-state","missing",timeout=30000)
        page.get_by_test_id(f"launcher-agent-install-{agent}").click()
        page.wait_for_function("() => document.querySelector('[data-testid=launcher-action-done]') || document.querySelector('[data-testid=launcher-action-error]')",timeout=240000)
        if not full_stack:
            expect(page.get_by_test_id("launcher-action-done")).to_contain_text("installed in your user-owned prefix",timeout=1000)
        else:
            expect(page.get_by_test_id("launcher-action-done")).to_be_visible(timeout=1000)
        expect(page.get_by_test_id(f"launcher-agent-status-{agent}")).to_have_attribute("data-harness-state","up_to_date",timeout=30000)
        executable = pathlib.Path.home()/".local/bin"/agent
        assert executable.is_symlink() and str(executable.resolve()).startswith(str(pathlib.Path.home()))
        page.get_by_test_id(f"launcher-doctor-{agent}").click()
        expect(page.get_by_test_id("launcher-action-done")).to_be_visible(timeout=30000)
        remove_managed(page,agent)
        assert not executable.exists()
        if all_harnesses:
            resource = "claude-code" if agent=="claude" else agent
            response = page.request.post(url+("/__journey/old-release" if agent=="agy" else "/__journey/old/"+resource), timeout=240000)
            assert response.status == 204,response.text()
            page.reload()
            open_drawer(page)
            page.get_by_test_id("launcher-mode-agent").click()
            page.get_by_test_id("launcher-manage-agents").click()
            page.get_by_test_id(f"launcher-manage-open-{agent}").click(timeout=30000)
            expect(page.get_by_test_id(f"launcher-agent-status-{agent}")).to_have_attribute("data-harness-state","update_ready",timeout=30000)
            page.get_by_test_id(f"launcher-action-primary-{agent}").click()
            expect(page.get_by_test_id("launcher-action-done")).to_contain_text("Updated",timeout=240000)
            expect(page.get_by_test_id(f"launcher-agent-status-{agent}")).to_have_attribute("data-harness-state","up_to_date",timeout=30000)
            remove_managed(page,agent)
            print(f"{agent}: real install, doctor, cleanup, older-release update and uninstall passed")


def cleanup_check(page):
    response = page.request.post(url+"/__journey/reset-managed")
    assert response.status == 204, response.text()
    found = pathlib.Path.home()/"found"/"claude"
    before = hashlib.sha256(found.read_bytes()).hexdigest()
    before_stat = found.stat()
    page.get_by_test_id("launcher-mode-agent").click()
    page.get_by_test_id("launcher-manage-agents").click()
    page.get_by_test_id("launcher-manage-open-claude").click(timeout=30000)
    expect(page.get_by_test_id("launcher-agent-status-claude")).to_have_attribute("data-harness-state","found",timeout=30000)
    page.get_by_test_id("launcher-doctor-claude").click()
    expect(page.get_by_test_id("launcher-action-done")).to_be_visible(timeout=30000)
    page.get_by_test_id("launcher-cleanup-preview-claude").click()
    preview = page.get_by_test_id("launcher-cleanup-plan")
    expect(preview).to_be_visible(timeout=30000)
    remove = preview.locator("[data-cleanup-remove]").all_text_contents()
    preserve = preview.locator("[data-cleanup-preserve]").all_text_contents()
    assert remove and str(found) in preserve, (remove,preserve)
    assert all(pathlib.Path(p).exists() and str(pathlib.Path.home()) in p for p in remove)
    page.get_by_test_id("launcher-uninstall-claude").click()
    expect(page.get_by_test_id("launcher-action-done")).to_contain_text("Removed the previewed",timeout=30000)
    assert all(not pathlib.Path(p).exists() for p in remove), remove
    assert all(pathlib.Path(p).exists() for p in preserve), preserve
    after_stat = found.stat()
    assert (after_stat.st_ino,after_stat.st_mode,after_stat.st_mtime_ns)==(before_stat.st_ino,before_stat.st_mode,before_stat.st_mtime_ns)
    assert hashlib.sha256(found.read_bytes()).hexdigest() == before
    assert len(list(found.parent.iterdir())) == 1, "The found binary was renamed or backed up"
    expect(page.get_by_test_id("launcher-agent-status-claude")).to_have_attribute("data-harness-state","found")

def update_check(page):
    response = page.request.post(url+"/__journey/old-release")
    assert response.status == 204, response.text()
    page.get_by_test_id("launcher-mode-agent").click()
    page.get_by_test_id("launcher-manage-agents").click()
    page.get_by_test_id("launcher-manage-open-agy").click(timeout=30000)
    action = page.get_by_test_id("launcher-action-agy")
    expect(action).to_have_attribute("data-harness-state", "update_ready", timeout=30000)
    expect(page.get_by_test_id("launcher-version")).to_contain_text("1.1.22")
    page.get_by_test_id("launcher-action-primary-agy").click()
    expect(page.get_by_test_id("launcher-action-done")).to_be_visible(timeout=240000)
    expect(page.get_by_test_id("launcher-action-done")).to_contain_text("Updated 1.1.22 to")
    expect(page.get_by_test_id("launcher-agent-status-agy")).to_have_attribute("data-harness-state", "up_to_date", timeout=30000)
    path = pathlib.Path.home()/".local"/"bin"/"agy"
    assert path.is_symlink() and "1.1.22-" not in str(path.resolve())
    if check == "TestJ8_FullDeploymentRouteOperated":
        page.get_by_test_id("launcher-doctor-agy").click()
        expect(page.get_by_test_id("launcher-action-done")).to_be_visible(timeout=30000)
        remove_managed(page,"agy")
        assert not path.exists()

def manage_check(page):
    response = page.request.post(url+"/__journey/unlink")
    assert response.status == 204, response.text()
    page.get_by_test_id("launcher-mode-agent").click()
    expect(page.get_by_test_id("launcher-manage-count")).to_be_visible(timeout=30000)
    assert int(page.get_by_test_id("launcher-manage-count").inner_text()) > 0
    page.get_by_test_id("launcher-manage-agents").click()
    page.get_by_test_id("launcher-manage-open-claude").click()
    action = page.get_by_test_id("launcher-action-claude")
    expect(action).to_have_attribute("data-harness-state","unlinked",timeout=30000)
    expect(action).to_contain_text("not linked")
    if check == "TestJ7_EligibilityReasonShownAndFixable":
        page.get_by_test_id("launcher-action-primary-claude").click()
        expect(page.get_by_test_id("launcher-action-done")).to_be_visible(timeout=60000)
        expect(page.get_by_test_id("launcher-action-done")).to_contain_text("now links to Vrooli")
        assert pathlib.Path(os.environ["HOME"],".local/bin/claude").is_symlink()
        expect(page.get_by_test_id("launcher-action-error")).to_have_count(0)

with sync_playwright() as p:
    browser = p.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True,
                                args=["--no-sandbox", "--disable-dev-shm-usage"])
    try:
        for name, width, height in [("desktop", 1440, 1000), ("phone", 390, 844)]:
            headers = {"Authorization":"LocalSession "+os.environ["LAUNCHER_LOCAL_TOKEN"]} if os.environ.get("LAUNCHER_LOCAL_TOKEN") else {}
            context = browser.new_context(viewport={"width": width, "height": height}, extra_http_headers=headers)
            page = context.new_page()
            page.on("requestfailed",lambda request: print("API request failed:",request.url,request.failure) if request.failure != "net::ERR_ABORTED" and ("/api/" in request.url or "/vrooli." in request.url) else None)
            page.goto(url)
            try:
                open_drawer(page)
                if check in ("TestJ8_DesktopUserOwnedInstallOperated","TestJ8_InstallUpdateUninstallDoctorPerHarness","TestJ8_FullDeploymentRouteOperated"):
                    native_install_check(page)
                elif check in ("TestJ8_FoundCopyStaysUntouched","TestJ8_CleanupPreviewEqualsRemoval"):
                    cleanup_check(page)
                elif check in ("TestJ7_ManageUpdateWorksOnIsolatedHome","TestJ8_FullDeploymentRouteOperated"):
                    update_check(page)
                elif check == "TestJ7_RawTerminalShowsNoAgentControls":
                    expect(page.get_by_test_id("launcher-agent-grid")).to_have_count(0)
                    expect(page.get_by_test_id("launcher-manage-agents")).to_have_count(0)
                    page.get_by_test_id("launcher-mode-shell").click()
                    expect(page.get_by_test_id("launcher-agent-grid")).to_have_count(0)
                    expect(page.get_by_test_id("launcher-manage-agents")).to_have_count(0)
                    expect(page.get_by_test_id("launcher-mode-shell")).to_have_attribute("aria-checked", "true")
                elif check == "TestJ7_CardOrderChosenOrRecentSurvivesReload":
                    order_check(page)
                elif check == "TestJ7_CardsDoNotShiftOnLoad":
                    height_check(page)
                elif check == "TestJ7_ContextChoicesShowRealNumbersPerModel":
                    context_check(page)
                elif check == "TestJ7_UsageWindowsComeFromDataWithResetTimes":
                    usage_check(page)
                elif check in ("TestJ7_ManageShowsBubbleWhenActionWaits", "TestJ7_EligibilityReasonShownAndFixable"):
                    manage_check(page)
                else:
                    raise AssertionError(f"Unimplemented journey: {check}")
                page.screenshot(path=str(pathlib.Path(evidence) / f"{check}-{name}.png"))
                print(f"{check}: {name} {width}x{height} passed")
            except Exception:
                page.screenshot(path=str(pathlib.Path(evidence) / f"failure-{name}.png"))
                print(page.locator("body").inner_text()[:12000])
                print(page.get_by_role("button").evaluate_all("els => els.map(e => ({text:e.textContent, label:e.getAttribute('aria-label'), title:e.title}))"))
                raise
            finally:
                context.close()
    finally:
        browser.close()
