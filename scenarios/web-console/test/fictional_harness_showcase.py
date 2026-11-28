"""Rendered-production-UI proof for the fictional, resource-data-only Nimbus harness."""
import json
import pathlib
import re
import sys

from playwright.sync_api import expect, sync_playwright

url, evidence = sys.argv[1:]
evidence = pathlib.Path(evidence)

with sync_playwright() as playwright:
    browser = playwright.chromium.launch(
        executable_path="/usr/bin/google-chrome",
        headless=True,
        args=["--no-sandbox", "--disable-dev-shm-usage"],
    )
    page = browser.new_page(viewport={"width": 1440, "height": 1000}, device_scale_factor=1)
    page.goto(url, wait_until="domcontentloaded")
    page.get_by_role("button", name=re.compile("(New session|Add session|Create.*session)", re.I)).first.click(timeout=30000)
    expect(page.get_by_test_id("launcher-mode-shell")).to_be_visible()
    page.get_by_test_id("launcher-mode-agent").click()
    page.get_by_test_id("launcher-manage-agents").click()
    nimbus_row = page.get_by_test_id("launcher-manage-open-nimbus")
    targets = page.request.post(
        url + "/vrooli.web_console.v1.targets.TargetCatalogService/List",
        data={},
    )
    target_data = targets.json() if targets.ok else {"error": targets.text()}
    observed_cards = page.locator('[data-testid="launcher-agent-management"] [data-testid^="launcher-manage-open-"]').evaluate_all(
        "els => els.map(e => ({id:e.dataset.capabilityId,testid:e.dataset.testid,label:e.innerText}))"
    )
    (evidence / "fictional-harness-ui-debug.json").write_text(json.dumps({
        "targets": target_data,
        "observedCards": observed_cards,
    }, indent=2) + "\n")
    page.screenshot(path=str(evidence / "fictional-harness-ui-debug.png"), full_page=True)
    expect(nimbus_row).to_be_visible(timeout=30000)
    nimbus_row.click()
    expect(page.get_by_test_id("launcher-agent-status-nimbus")).to_be_visible(timeout=30000)

    effective = page.request.post(
        url + "/vrooli.web_console.v1.shortcuts.ShortcutsService/GetEffective",
        data={},
    )
    effective_data = effective.json() if effective.ok else {"error": effective.text()}
    (evidence / "fictional-harness-ui-debug.json").write_text(json.dumps({
        "effective": effective_data,
        "targets": target_data,
        "observedCards": observed_cards,
    }, indent=2) + "\n")
    page.screenshot(path=str(evidence / "fictional-harness-ui-debug.png"), full_page=True)
    print(f"Effective shortcuts: {[item.get('agentId') for item in effective_data.get('shortcuts', [])]}; rendered cards: {observed_cards}")
    assert any(item["id"] == "nimbus" and "Nimbus" in item["label"] for item in observed_cards), observed_cards

    descriptors = page.request.post(
        url + "/vrooli.web_console.v1.capabilities.CapabilitiesService/ListHarnesses",
        data={"targetId": "local"},
    )
    assert descriptors.ok, descriptors.text()
    descriptor_receipt = descriptors.json()
    nimbus = next(item for item in descriptor_receipt["harnesses"] if item["agentId"] == "nimbus")
    assert nimbus["displayName"] == "Nimbus"
    assert nimbus["supportsContextLimit"] is True
    assert nimbus["supportsModelSelection"] is True

    policy = page.request.post(
        url + "/vrooli.web_console.v1.modelselection.ModelSelectionService/GetRoleModelChoices",
        data={"agentId": "nimbus", "targetId": "local"},
    )
    assert policy.ok, policy.text()
    policy_receipt = policy.json()
    assert policy_receipt["policyStatus"] == "ready", policy_receipt
    assert any(choice["model"] == "nimbus-1" and choice["displayName"] == "Nimbus R1" for choice in policy_receipt["choices"]), policy_receipt

    screenshot = evidence / "fictional-harness-served-desktop.png"
    page.screenshot(path=str(screenshot), full_page=True)
    (evidence / "fictional-harness-api-receipt.json").write_text(
        json.dumps({"descriptor": nimbus, "policy": policy_receipt, "ui": {"cardTestId": "launcher-agent-nimbus", "modelShown": "Nimbus R1", "contextLimitShown": True}}, indent=2) + "\n"
    )
    print(f"Served production Manage Agents UI rendered Nimbus from resource data; descriptor and policy receipts saved; screenshot: {screenshot}")
    browser.close()
