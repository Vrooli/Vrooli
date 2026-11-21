"""Exercise renderer authority against an unrelated real loopback HTTP origin."""
import json
import pathlib
import sys
from playwright.sync_api import Error, sync_playwright

endpoint, renderer_url, unrelated_url, evidence = sys.argv[1:]
with sync_playwright() as p:
    browser = p.chromium.connect_over_cdp(endpoint)
    page = next(page for context in browser.contexts for page in context.pages if page.url.startswith(renderer_url))
    received = lambda: page.evaluate("async () => Boolean(await window.desktop?.auth?.getLocalSessionToken?.())")
    assert received(), "owned renderer lost its authenticated local session"
    # Ordinary user navigation must remain inside the shell-owned renderer.
    page.evaluate("url => { const link = document.createElement('a'); link.href = url; link.textContent = 'Unrelated test origin'; document.body.append(link); link.click(); }", unrelated_url)
    page.wait_for_timeout(500)
    assert page.url.startswith(renderer_url), "foreign user navigation committed"
    navigation_blocked = True
    # CDP may bypass normal navigation events. Even then the native authority
    # must reject a foreign document instead of trusting its loopback hostname.
    try:
        page.goto(unrelated_url, timeout=10000, wait_until="domcontentloaded")
    except Error:
        pass
    foreign_received = received() if page.url.startswith(unrelated_url) else False
    assert not foreign_received, "unrelated loopback document received the runtime token"
    if not page.url.startswith(renderer_url):
        page.goto(renderer_url, timeout=30000, wait_until="domcontentloaded")
    assert received(), "owned renderer could not resume after refused navigation"
    result = {"ordinaryNavigationBlocked": navigation_blocked, "unrelatedOriginReceivedToken": foreign_received, "ownedRendererUsable": True}
    out = pathlib.Path(evidence)
    out.mkdir(parents=True, exist_ok=True)
    (out / "j9-renderer-authority.json").write_text(json.dumps(result, indent=2))
    page.screenshot(path=str(out / "j9-renderer-authority.png"), full_page=True)
    print(json.dumps(result), flush=True)
    browser.close()
