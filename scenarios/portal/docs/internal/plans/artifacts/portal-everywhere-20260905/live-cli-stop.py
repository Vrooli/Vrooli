"""50 physical local Stop trials; CLI dispatch-to-receipt upper-bounds lease invalidation.

Cohort: control lease after a completed physical observation, no held input or
concurrent mutation. This does not measure congested, browser, or remote Stop.
"""
import ctypes as c
from datetime import datetime, timezone
import json
import math
import os
from pathlib import Path
import sqlite3
import subprocess
import tempfile
import time

base = Path('/run/user/1000/vrooli-desktop-owner')
config = json.loads((base / 'owner.json').read_text())
assert not json.loads((base / 'grants.json').read_text())['active'], 'desktop already admitted'
os.environ['XAUTHORITY'] = config['helper']['xauthority_file']
x = c.CDLL('libX11.so.6')
x.XOpenDisplay.argtypes = [c.c_char_p]; x.XOpenDisplay.restype = c.c_void_p
x.XDefaultRootWindow.argtypes = [c.c_void_p]; x.XDefaultRootWindow.restype = c.c_ulong
x.XCloseDisplay.argtypes = [c.c_void_p]
x.XQueryPointer.argtypes = [c.c_void_p, c.c_ulong, c.POINTER(c.c_ulong), c.POINTER(c.c_ulong), *([c.POINTER(c.c_int)] * 4), c.POINTER(c.c_uint)]
display = x.XOpenDisplay((':' + str(config['helper']['display'])).encode())
assert display

def current_pointer():
    root, child = c.c_ulong(), c.c_ulong()
    rx, ry, wx, wy = c.c_int(), c.c_int(), c.c_int(), c.c_int()
    mask = c.c_uint()
    assert x.XQueryPointer(display, x.XDefaultRootWindow(display), c.byref(root), c.byref(child), c.byref(rx), c.byref(ry), c.byref(wx), c.byref(wy), c.byref(mask))
    return rx.value, ry.value

def state():
    with sqlite3.connect('file:' + str(base / 'helper-state/sessions.sqlite') + '?mode=ro', uri=True, timeout=2) as db:
        return json.loads(db.execute('SELECT state FROM device_control_desktop_sessions WHERE destination=?', (config['helper']['session_id'],)).fetchone()[0])

evidence = {'started_at': datetime.now(timezone.utc).isoformat(), 'target': config['helper']['surface']['target'], 'desktop_session': config['helper']['session_id'], 'cohort': 'local installed CLI, after completed physical capture, no held input or concurrent mutation', 'measurement': 'monotonic CLI subprocess dispatch to successful Stop receipt; includes CLI startup and exceeds durable lease invalidation time', 'requested_trials': 50, 'trials': []}
try:
    with tempfile.TemporaryDirectory(prefix='stop-benchmark-', dir=base) as temporary:
        work = Path(temporary)
        def call(operation, body, extra=(), refuse=False):
            request = work / (operation + '.json')
            request.write_text(json.dumps(body)); request.chmod(0o600)
            start = time.monotonic_ns()
            result = subprocess.run(['device-control', 'desktop', operation, '--socket', str(base / 'desktop-owner.sock'), '--request', str(request), '--json', *extra], capture_output=True, text=True, timeout=12)
            elapsed = (time.monotonic_ns() - start) / 1e6
            if refuse:
                assert result.returncode != 0 and 'permission_denied' in result.stderr, result.stderr[:300]
                return None, elapsed
            assert result.returncode == 0, operation + ': ' + result.stderr[:500]
            return json.loads(result.stdout), elapsed
        for index in range(50):
            opened, _ = call('open', {'surface': config['helper']['surface'], 'ttl_seconds': 30, 'control': True})
            session = opened['session']; stopped = False
            try:
                png = work / 'capture.png'
                observed, _ = call('observe', {'session': session}, ('--output', str(png)))
                png.unlink()
                px, py = current_pointer()
                before = state()
                assert before['lease']['ref']['session_id'] == session['session_id']
                _, latency = call('stop', {'session': session}); stopped = True
                after = state()
                assert not after.get('lease') and not after.get('cleanup_pending')
                assert not json.loads((base / 'grants.json').read_text())['active']
                # A well-formed pointer move to its current location limits impact
                # if a regression admits this request. The expected result is refusal.
                call('act', {'session': session, 'command_id': 'post-stop-' + str(index), 'geometry_revision': observed['geometry_revision'], 'action': {'pointer': {'kind': 'KIND_MOVE', 'button': 'BUTTON_PRIMARY', 'display_id': observed['display_id'], 'x': px, 'y': py}}}, refuse=True)
                evidence['trials'].append({'trial': index + 1, 'stop_receipt_ms': round(latency, 3), 'epoch': before['epoch'], 'durable_lease_cleared': True, 'post_stop_input_denied': True})
            finally:
                if not stopped:
                    call('stop', {'session': session})
        values = sorted(t['stop_receipt_ms'] for t in evidence['trials'])
        evidence.update(p95_ms=values[math.ceil(.95 * len(values)) - 1], max_ms=max(values), completed_trials=len(values), target_ms=250)
        evidence['passed'] = evidence['p95_ms'] <= 250
        assert evidence['passed'], evidence['p95_ms']
finally:
    x.XCloseDisplay(display)
    evidence['finished_at'] = datetime.now(timezone.utc).isoformat()
    Path(__file__).with_suffix('.json').write_text(json.dumps(evidence, indent=2) + '\n')
    print(json.dumps({k: v for k, v in evidence.items() if k != 'trials'}))
