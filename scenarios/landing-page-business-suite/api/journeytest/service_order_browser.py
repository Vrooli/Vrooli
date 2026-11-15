import sys
from playwright.sync_api import sync_playwright

url = sys.argv[1]
with sync_playwright() as p:
    browser = p.chromium.launch(headless=True, executable_path="/usr/bin/google-chrome")
    page = browser.new_page(
        viewport={"width": 390, "height": 844},
        device_scale_factor=3,
        is_mobile=True,
        has_touch=True,
    )
    page.goto(url)
    dimensions = page.evaluate("({width: innerWidth, document: document.documentElement.scrollWidth})")
    assert dimensions["width"] == 390 and dimensions["document"] <= dimensions["width"], dimensions
    assert page.evaluate("(() => { const r = document.querySelector('#buy').getBoundingClientRect(); return r.top >= 0 && r.bottom <= innerHeight; })()")
    assert page.get_by_text("Sandbox simulation only — no real money moved.").is_visible()
    page.get_by_role("button", name="Create sandbox order").click()
    page.wait_for_function("document.querySelector('#buy').disabled")
    page.evaluate("document.querySelector('#buy').click()")
    page.wait_for_function("document.querySelector('#result').textContent.includes('retry same order')")
    assert page.get_by_role("button", name="Create sandbox order").is_enabled()
    page.get_by_role("button", name="Create sandbox order").click()
    page.wait_for_function("() => { try { return JSON.parse(document.querySelector('#result').textContent).state === 'authorized' } catch { return false } }")
    order = page.evaluate("JSON.parse(document.querySelector('#result').textContent)")
    assert order["offer_id"] == "company-market-scan"
    assert order["amount_minor"] == 4900 and order["currency"] == "USD"
    assert order["order_id"] and order["job_id"] and order["mode"] == "sandbox"
    assert order["job_input"]["sector"] == "battery storage"
    page.get_by_role("button", name="Verify and capture").click()
    page.wait_for_function("() => { try { return JSON.parse(document.querySelector('#result').textContent).state === 'ledger_pending' } catch { return false } }")
    page.get_by_role("button", name="Reconcile test ledger").click()
    page.wait_for_function("() => { try { return JSON.parse(document.querySelector('#result').textContent).state === 'reconciled' } catch { return false } }")
    order = page.evaluate("JSON.parse(document.querySelector('#result').textContent)")
    assert order["ledger_posting_id"]
    print(f"browser sandbox order={order['order_id']} job={order['job_id']} posting={order['ledger_posting_id']}")
    browser.close()
