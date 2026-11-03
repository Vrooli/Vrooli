"""Operate Git Control Tower's production settings and real dependency API."""
import pathlib
import sys
from playwright.sync_api import sync_playwright, expect
url, evidence=sys.argv[1:]
with sync_playwright() as p:
    browser=p.chromium.launch(executable_path="/usr/bin/google-chrome",headless=True,args=["--no-sandbox"])
    for width,height in [(1440,1000),(390,844)]:
        page=browser.new_page(viewport={"width":width,"height":height})
        page.goto(url)
        if width < 600:
            page.get_by_test_id("mobile-menu-button").click(timeout=30000)
            page.get_by_role("button",name="Settings",exact=False).last.click()
        else:
            page.get_by_test_id("settings-button").click(timeout=30000)
        page.get_by_role("button",name="Integrations",exact=True).first.click()
        panel=page.locator('[data-rcl-integrations]')
        expect(panel.get_by_role("button",name="By capability",exact=True)).to_be_visible(timeout=30000)
        response=page.request.get(url+"/api/v1/capabilities")
        assert response.ok,response.text()
        body=response.json()
        dependencies=body.get("data",body)["capabilities"]
        assert dependencies
        for dep in dependencies:
            for use in dep.get("uses",[]):
                expect(panel.get_by_text(use["name"],exact=True).first).to_be_visible()
        agents=next(d for d in dependencies if d["dependencySlug"]=="agent-manager")
        assert len(agents["uses"])>=3
        panel.get_by_role("button",name="By dependency",exact=True).click()
        for dep in dependencies:
            for use in dep.get("uses",[]):
                expect(panel.get_by_text(use["description"],exact=False).first).to_be_visible()
        assert page.evaluate("document.documentElement.scrollWidth<=innerWidth")
        page.screenshot(path=str(pathlib.Path(evidence)/f"TestJ14_SecondAppAdoptsIt-{width}.png"))
        panel.get_by_text("Run agents and review their progress", exact=False).first.scroll_into_view_if_needed()
        page.screenshot(path=str(pathlib.Path(evidence)/f"TestJ14_SecondAppAdoptsIt-{width}-rows.png"))
        page.close()
    browser.close()
print("Second production app operated against real registry and API at desktop and phone widths")
