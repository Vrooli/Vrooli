"""Real Chrome -> production browser auth/checkout -> signed mock Stripe webhook."""
import json
import pathlib
import re
import sys
from decimal import Decimal
from playwright.sync_api import sync_playwright, expect

base, check, result, evidence = sys.argv[1:5]
verify = len(sys.argv) > 5 and sys.argv[5] == "verify"
pathlib.Path(evidence).mkdir(parents=True, exist_ok=True)

def verify_wallet(page, context):
    expected = json.loads(pathlib.Path(result).read_text())
    page.goto(base + "/account/credits")
    expect(page.get_by_role("heading", name="Your credits", exact=True)).to_be_visible(timeout=30000)
    expect(page.get_by_role("button", name="Billing account")).to_be_visible()
    assert page.locator('select[aria-label="Billing account"]').input_value() == expected["account"]
    response = context.request.get(base + "/api/v1/usage/business-account-summary?business_account_id=" + expected["account"])
    assert response.ok, response.text()
    wallet = response.json()["wallet"]
    assert wallet["available"] == expected["available"], (wallet, expected)
    assert wallet["on_hold"] == expected["on_hold"], (wallet, expected)
    assert wallet["livemode"] is False
    pricing = context.request.post(base + '/landing_page_business_suite.v1.PricingService/GetPricing', data={}).json()["pricing"]["bundle"]
    def text(value):
        number = Decimal(value) * Decimal(str(pricing['displayCreditsMultiplier']))
        return f"{number:,.3f}".rstrip('0').rstrip('.') + ' ' + pricing['displayCreditsLabel']
    expect(page.get_by_test_id('credits-available')).to_have_text(text(expected['available']))
    expect(page.get_by_test_id('credits-on-hold')).to_have_text(text(expected['on_hold']))
    expect(page.get_by_role('heading', name='Monthly spending cap', exact=True)).to_be_visible()
    for reason in expected.get('reasons', []):
        expect(page.get_by_text(reason, exact=True)).to_be_visible()
    if 'history' in expected:
        actual = {item['feature_id']: {'credits': item['credits'], 'calls': item['calls']} for item in wallet['history']}
        assert actual == expected['history'], (actual, expected['history'])
        expect(page.locator('tbody tr')).to_have_count(len(expected['history']))
        for feature, item in expected['history'].items():
            page.get_by_role('button', name='Filter history by feature').click()
            page.get_by_role('option', name=feature, exact=True).click()
            row = page.get_by_role('row').filter(has=page.get_by_role('rowheader', name=feature, exact=True))
            expect(row).to_contain_text(text(item['credits']))
            expect(page.locator('tbody tr')).to_have_count(1)
        page.get_by_role('button', name='Filter history by feature').click()
        page.get_by_role('option', name='All features', exact=True).click()
        expect(page.locator('tbody tr')).to_have_count(len(expected['history']))
    foreign = context.request.get(base + '/api/v1/usage/business-account-summary?business_account_id=account')
    assert foreign.status == 403, 'Another account wallet must remain private'
    page.reload()
    expect(page.get_by_test_id('credits-available')).to_have_text(text(expected['available']))
    page.screenshot(path=str(pathlib.Path(evidence) / (check + '-desktop.png')), full_page=True)
    print('Real wallet amounts, holds/cap, feature filter, account isolation and reload passed.')

with sync_playwright() as pw:
    browser = pw.chromium.launch(executable_path="/usr/bin/google-chrome", headless=True, args=["--no-sandbox"])
    context = browser.new_context(viewport={"width": 1440, "height": 1000}, **({"storage_state": result + ".state"} if verify else {}))
    page = context.new_page()
    page.on("pageerror", lambda error: print("Browser error:", str(error)))
    page.on("requestfailed", lambda request: print("Browser request failed:", request.url, request.failure))
    page.on("console", lambda message: print("Browser console:", message.text) if message.type == "error" else None)
    def failure_hook(kind, value, traceback):
        print("Browser URL:", page.url)
        print("Browser body:", page.locator("body").inner_text()[:3000])
        page.screenshot(path=str(pathlib.Path(evidence) / (check + "-failure.png")), full_page=True)
        sys.__excepthook__(kind, value, traceback)
    try:
        if not verify:
            page.goto(base + "/account/credits")
            page.get_by_test_id("email-input").fill("buyer@example.invalid")
            page.get_by_test_id("submit-button").click()
            expect(page.get_by_test_id("code-input")).to_be_visible()
            mail = context.request.get(base + "/__j15/mail")
            assert mail.ok, mail.text()
            code = mail.json()["code"]
            assert re.fullmatch(r"\d{6}", code)
            page.get_by_test_id("code-input").fill(code)
            page.get_by_role("button", name="Not now", exact=True).click()
            expect(page.get_by_role("heading", name="Your credits", exact=True)).to_be_visible(timeout=30000)
            expect(page.get_by_role("button", name="Billing account")).to_be_visible()
            account = page.locator('select[aria-label="Billing account"]').input_value()
            assert account
            before = context.request.get(base + "/api/v1/usage/business-account-summary?business_account_id=" + account)
            assert before.ok, before.text()
            page.get_by_role("heading", name="Add credits", exact=True).locator("..").get_by_role("button").first.click()
            expect(page.get_by_role("heading", name="Mock Stripe · test only")).to_be_visible()
            page.get_by_label("Test card number").fill("4242 4242 4242 4242")
            with page.expect_response(lambda response: response.url.endswith('/pay')) as payment:
                page.get_by_role("button", name="Pay with test card").click()
            assert payment.value.status == 303, f"Mock payment returned {payment.value.status}"
            expect(page.get_by_role("heading", name="Your credits", exact=True)).to_be_visible(timeout=30000)
            assert "checkout=complete" in page.url
            page.screenshot(path=str(pathlib.Path(evidence) / (check + "-desktop.png")), full_page=True)
            device = context.request.post(base + '/api/v1/ai/keys/device', data={"business_account_id": account, "max_credits": 5000})
            assert device.ok, f'Device issuance status {device.status}'
            key = device.json()['key']
            pathlib.Path(result).write_text(json.dumps({"account": account, "device": key}))
            context.storage_state(path=result + ".state")
            print("Signed in using the real queued email code, opened account-bound checkout, paid the mock test card, and returned through the signed webhook.")
        else:
            verify_wallet(page, context)
        page.set_viewport_size({"width": 390, "height": 844})
        assert page.evaluate("document.documentElement.scrollWidth <= innerWidth + 1"), "Phone overflow"
        page.screenshot(path=str(pathlib.Path(evidence) / (check + "-phone.png")), full_page=True)
    except BaseException:
        failure_hook(*sys.exc_info())
        raise
    browser.close()
