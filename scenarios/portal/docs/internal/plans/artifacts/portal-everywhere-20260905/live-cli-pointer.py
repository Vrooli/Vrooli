"""Bounded real X11 pointer acceptance through the installed owner CLI.
Creates only a disposable fixture window. Never uses Xlib to inject input.
"""
import ctypes as c
import json
import os
from pathlib import Path
import subprocess
import tempfile
import time

base = Path('/run/user/1000/vrooli-desktop-owner')
config = json.loads((base / 'owner.json').read_text())
os.environ['XAUTHORITY'] = config['helper']['xauthority_file']
x = c.CDLL('libX11.so.6')
D, W, I, U = c.c_void_p, c.c_ulong, c.c_int, c.c_uint

def bind(name, result, *args):
    fn = getattr(x, name); fn.restype = result; fn.argtypes = args; return fn

open_display = bind('XOpenDisplay', D, c.c_char_p)
root_window = bind('XDefaultRootWindow', W, D)
create = bind('XCreateSimpleWindow', W, D, W, I, I, U, U, U, W, W)
select = bind('XSelectInput', I, D, W, c.c_long)
name = bind('XStoreName', I, D, W, c.c_char_p)
map_window = bind('XMapRaised', I, D, W)
sync = bind('XSync', I, D, I)
translate = bind('XTranslateCoordinates', I, D, W, W, I, I, c.POINTER(I), c.POINTER(I), c.POINTER(W))
query = bind('XQueryPointer', I, D, W, c.POINTER(W), c.POINTER(W), c.POINTER(I), c.POINTER(I), c.POINTER(I), c.POINTER(I), c.POINTER(U))
pending = bind('XPending', I, D)
next_event = bind('XNextEvent', I, D, c.c_void_p)
destroy = bind('XDestroyWindow', I, D, W)
close = bind('XCloseDisplay', I, D)

class ButtonEvent(c.Structure):
    _fields_ = [('type', I), ('serial', W), ('send_event', I), ('display', D),
                ('window', W), ('root', W), ('subwindow', W), ('time', W),
                ('x', I), ('y', I), ('x_root', I), ('y_root', I),
                ('state', U), ('button', U), ('same_screen', I)]

display = open_display((':' + str(config['helper']['display'])).encode())
assert display, 'fixture could not connect to explicitly configured X display'
root = root_window(display)
window = create(display, root, 400, 300, 160, 100, 1, 0x222222, 0x265b87)
assert window
select(display, window, (1 << 2) | (1 << 3))
name(display, window, b'Vrooli disposable pointer acceptance')
map_window(display, window); sync(display, 0)
time.sleep(0.3)

def pointer(window_id):
    root_out, child = W(), W(); rx, ry, wx, wy = I(), I(), I(), I(); mask = U()
    assert query(display, window_id, c.byref(root_out), c.byref(child), c.byref(rx), c.byref(ry), c.byref(wx), c.byref(wy), c.byref(mask))
    return rx.value, ry.value, mask.value, child.value

original = pointer(root)
assert original[2] & (256 | 512 | 1024 | 2048 | 4096) == 0, 'user already holds a pointer button'
px, py, child = I(), I(), W()
assert translate(display, window, root, 80, 50, c.byref(px), c.byref(py), c.byref(child))
events = []

def drain():
    sync(display, 0)
    while pending(display):
        event = (c.c_long * 24)(); next_event(display, event)
        button = c.cast(event, c.POINTER(ButtonEvent)).contents
        if button.type in (4, 5) and button.window == window:
            events.append({'type': 'press' if button.type == 4 else 'release', 'button': button.button,
                           'x': button.x, 'y': button.y, 'synthetic_flag': bool(button.send_event)})

opened = None
try:
    with tempfile.TemporaryDirectory(prefix='pointer-fixture-', dir=base) as temporary:
        work = Path(temporary)
        def call(op, body, extra=()):
            request = work / (op + '.json'); request.write_text(json.dumps(body)); request.chmod(0o600)
            result = subprocess.run(['device-control', 'desktop', op, '--socket', str(base / 'desktop-owner.sock'),
                                     '--request', str(request), '--json', *extra], capture_output=True, text=True, timeout=20)
            if result.returncode: raise RuntimeError(result.stderr[:1500])
            return json.loads(result.stdout)
        opened = call('open', {'surface': config['helper']['surface'], 'ttl_seconds': 60, 'control': True})
        try:
            observed = call('observe', {'session': opened['session']}, ('--output', str(work / 'fixture.png')))
        except Exception:
            call('stop', {'session': opened['session']})
            raise
        def act(command_id, kind, x_pos, y_pos):
            return call('act', {'session': opened['session'], 'command_id': command_id,
                               'geometry_revision': observed['geometry_revision'],
                               'action': {'pointer': {'kind': kind, 'button': 'BUTTON_PRIMARY',
                                          'display_id': observed['display_id'], 'x': x_pos, 'y': y_pos}}})
        try:
            act('fixture-move', 'KIND_MOVE', px.value, py.value)
            current = pointer(root)
            assert current[:2] == (px.value, py.value)
            # Refuse to click unless the fixture is the actual window under the pointer.
            under = current[3]
            for _ in range(16):
                if under == window: break
                assert under, 'fixture is occluded; no click permitted'
                under = pointer(under)[3]
            assert under == window, 'fixture not found under pointer'
            drain(); assert not events
            receipt = act('fixture-click-once', 'KIND_CLICK', px.value, py.value)
            duplicate = act('fixture-click-once', 'KIND_CLICK', px.value, py.value)
            drain()
            assert receipt['receipt']['outcome'] == 'applied'
            assert duplicate == receipt
            assert [e['type'] for e in events] == ['press', 'release'], events
            assert all(e['button'] == 1 and e['x'] == 80 and e['y'] == 50 for e in events)
            assert pointer(root)[2] & 256 == 0
            evidence = {'target': opened['session']['surface']['target'], 'os_session': config['helper']['session_id'],
                        'fixture': 'disposable X11 window', 'events': events, 'receipt': receipt['receipt'],
                        'duplicate_receipt_equal': True, 'no_held_primary_button': True,
                        'observation': observed}
        finally:
            try:
                act('fixture-restore-pointer', 'KIND_MOVE', original[0], original[1])
                restored = pointer(root)[:2] == original[:2]
            finally:
                call('stop', {'session': opened['session']})
        evidence['pointer_restored'] = restored
        evidence['stop_succeeded'] = True
        Path(__file__).with_suffix('.json').write_text(json.dumps(evidence, indent=2) + '\n')
        print(json.dumps(evidence, indent=2))
finally:
    destroy(display, window); sync(display, 0); close(display)
