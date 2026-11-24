#!/usr/bin/env bash
# Regenerate every capture in shots/ from the HTML in this folder.
# Requires Google Chrome; no network and no dev server.
set -euo pipefail
cd "$(dirname "$0")"

CHROME="${CHROME:-$(command -v google-chrome || command -v chromium)}"
shot() { # name, width, height, scale, url
  "$CHROME" --headless=new --disable-gpu --hide-scrollbars --no-sandbox \
    --window-size="$2,$3" --force-device-scale-factor="$4" \
    --screenshot="shots/$1.png" "file://$PWD/$5" 2>/dev/null
}

shot desktop-dark        1440 940 1 "desktop.html?theme=dark"
shot desktop-light       1440 940 1 "desktop.html?theme=light"
shot focus-dark          1440 940 1 "focus.html?theme=dark"
shot focus-light         1440 940 1 "focus.html?theme=light"
shot mobile-dark          390 844 2 "mobile.html?theme=dark"
shot mobile-light         390 844 2 "mobile.html?theme=light"
shot mobile-detail-dark   390 844 2 "mobile-detail.html?theme=dark"
shot mobile-detail-light  390 844 2 "mobile-detail.html?theme=light"
shot mobile-320-dark      320 700 2 "mobile.html?theme=dark"
shot mobile-320-light     320 700 2 "mobile.html?theme=light"
shot mobile-detail-320-dark  320 700 2 "mobile-detail.html?theme=dark"
shot mobile-detail-320-light 320 700 2 "mobile-detail.html?theme=light"

# The case study needs a full-page capture, so it uses Playwright when present.
if command -v node >/dev/null && [ -d "$HOME/Vrooli/node_modules/.pnpm/playwright@1.62.1" ]; then
  node - "$PWD" <<'JS'
  const { chromium } = require(process.env.HOME + '/Vrooli/node_modules/.pnpm/playwright@1.62.1/node_modules/playwright');
  (async () => {
    const dir = process.argv[2];
    const b = await chromium.launch({ executablePath: '/usr/bin/google-chrome', args: ['--no-sandbox'] });
    const p = await b.newPage({ viewport: { width: 1440, height: 1000 } });
    await p.goto('file://' + dir + '/index.html', { waitUntil: 'load' });
    await p.screenshot({ path: 'shots/case-cover.png' });
    await p.screenshot({ path: 'shots/case-full.png', fullPage: true });
    await b.close();
  })();
JS
fi

echo "captured $(ls shots/*.png | wc -l) screenshots"
