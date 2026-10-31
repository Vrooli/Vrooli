#!/usr/bin/env python3
"""Capture a set of RCL UI routes for retained visual evidence."""
import json, os, pathlib, sys
from playwright.sync_api import sync_playwright

base = os.environ.get('RCL_BASE', 'http://localhost:23906')
output = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else '/tmp/rclshots'); output.mkdir(parents=True, exist_ok=True)
routes = json.loads(os.environ.get('ROUTES', '["/"]'))
width = int(os.environ.get('W', '1440')); height = int(os.environ.get('H', '900')); full = os.environ.get('FULL', '1') == '1'
with sync_playwright() as playwright:
    browser = playwright.chromium.launch(executable_path='/usr/bin/google-chrome', args=['--no-sandbox', '--disable-dev-shm-usage'])
    page = browser.new_context(viewport={'width': width, 'height': height}, device_scale_factor=1).new_page()
    errors = []
    page.on('console', lambda message: errors.append(f'[{message.type}] {message.text}') if message.type in ('error', 'warning') else None)
    page.on('pageerror', lambda error: errors.append(f'[pageerror] {error}'))
    for route in routes:
        name = route.strip('/').replace('/', '_') or 'root'
        try: page.goto(base + route, wait_until='networkidle', timeout=25000)
        except Exception as error: print(f'{route}: goto issue {error}')
        page.wait_for_timeout(1400); path = output / f'{name}.png'; page.screenshot(path=str(path), full_page=full)
        print(f'{route} -> {path} (title={page.title()!r})')
    if errors:
        print('--- console ---')
        for error in errors[:40]: print(error[:300])
    browser.close()
