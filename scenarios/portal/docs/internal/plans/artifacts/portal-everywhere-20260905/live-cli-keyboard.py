"""Bounded real X11 keyboard acceptance through the installed owner CLI.
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
pending = bind('XPending', I, D)
next_event = bind('XNextEvent', I, D, c.c_void_p)
destroy = bind('XDestroyWindow', I, D, W)
close = bind('XCloseDisplay', I, D)

class KeyEvent(c.Structure):
    _fields_ = [('type', I), ('serial', W), ('send_event', I), ('display', D),
                ('window', W), ('root', W), ('subwindow', W), ('time', W),
                ('x', I), ('y', I), ('x_root', I), ('y_root', I),
                ('state', U), ('keycode', U), ('same_screen', I)]

get_focus = bind('XGetInputFocus', I, D, c.POINTER(W), c.POINTER(I))
set_focus = bind('XSetInputFocus', I, D, W, I, W)
keymap = bind('XQueryKeymap', I, D, c.c_void_p)
lookup = bind('XLookupKeysym', W, c.c_void_p, I)

def no_keys_down():
    keys = (c.c_ubyte * 32)(); keymap(display, keys)
    return not any(keys)

display = open_display((':' + str(config['helper']['display'])).encode())
assert display, 'fixture could not connect to explicitly configured X display'
root = root_window(display)
original_focus, original_revert = W(), I()
get_focus(display, c.byref(original_focus), c.byref(original_revert))
assert no_keys_down(), 'user already holds a key'
window = create(display, root, 400, 300, 160, 100, 1, 0x222222, 0x265b87)
assert window
select(display, window, (1 << 0) | (1 << 1))
name(display, window, b'Vrooli disposable keyboard acceptance')
map_window(display, window); sync(display, 0)
time.sleep(0.3)

set_focus(display, window, 1, 0); sync(display, 0)
events = []

def drain():
    sync(display, 0)
    while pending(display):
        event = (c.c_long * 24)(); next_event(display, event)
        button = c.cast(event, c.POINTER(KeyEvent)).contents
        if button.type in (2, 3) and button.window == window:
            events.append({'type': 'press' if button.type == 2 else 'release', 'keysym': int(lookup(event, 0)), 'synthetic_flag': bool(button.send_event)})

opened = None
try:
    with tempfile.TemporaryDirectory(prefix='keyboard-fixture-', dir=base) as temporary:
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
        def act(command_id, kind, key):
            focused, revert = W(), I()
            get_focus(display, c.byref(focused), c.byref(revert))
            assert focused.value == window, 'fixture lost focus; no keyboard input permitted'
            return call('act', {'session': opened['session'], 'command_id': command_id,
                               'geometry_revision': observed['geometry_revision'],
                               'action': {'key': {'kind': kind, 'key': key}}})
        try:
            drain(); assert not events
            receipt = act('fixture-key-once', 'KIND_PRESS', 'a')
            duplicate = act('fixture-key-once', 'KIND_PRESS', 'a')
            drain()
            assert receipt['receipt']['outcome'] == 'applied'
            assert duplicate == receipt
            assert [e['type'] for e in events] == ['press', 'release'], events
            assert all(e['keysym'] == ord('a') for e in events), events
            assert no_keys_down()
            act('fixture-modifier-down', 'KIND_DOWN', 'ShiftLeft')
            assert not no_keys_down()
        finally:
            call('stop', {'session': opened['session']})
        drain()
        assert no_keys_down(), 'Stop left a key held'
        assert [(e['type'], e['keysym']) for e in events[-2:]] == [('press', 0xffe1), ('release', 0xffe1)], events
        evidence = {'target': opened['session']['surface']['target'], 'os_session': config['helper']['session_id'],
                    'fixture': 'disposable focused X11 window', 'events': events, 'receipt': receipt['receipt'],
                    'duplicate_receipt_equal': True, 'stop_released_modifier': True,
                    'no_held_keys': True, 'observation': observed}
        set_focus(display, original_focus, original_revert, 0); sync(display, 0)
        focused, revert = W(), I(); get_focus(display, c.byref(focused), c.byref(revert))
        evidence['focus_restored'] = focused.value == original_focus.value
        assert evidence['focus_restored']
        Path(__file__).with_suffix('.json').write_text(json.dumps(evidence, indent=2) + '\n')
        print(json.dumps(evidence, indent=2))
finally:
    set_focus(display, original_focus, original_revert, 0)
    destroy(display, window); sync(display, 0); close(display)
