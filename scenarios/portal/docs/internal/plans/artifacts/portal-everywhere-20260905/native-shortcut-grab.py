"""Bounded conflict fixture for an explicitly owner-created virtual X display."""
import ctypes
import json
import re
import sys

assert re.fullmatch(r':\d+', sys.argv[1]) and int(sys.argv[1][1:]) >= 99
x = ctypes.CDLL('libX11.so.6')
x.XOpenDisplay.argtypes = [ctypes.c_char_p]
x.XOpenDisplay.restype = ctypes.c_void_p
x.XDefaultRootWindow.argtypes = [ctypes.c_void_p]
x.XDefaultRootWindow.restype = ctypes.c_ulong
x.XStringToKeysym.argtypes = [ctypes.c_char_p]
x.XStringToKeysym.restype = ctypes.c_ulong
x.XKeysymToKeycode.argtypes = [ctypes.c_void_p, ctypes.c_ulong]
x.XKeysymToKeycode.restype = ctypes.c_uint
x.XGrabKey.argtypes = [ctypes.c_void_p, ctypes.c_int, ctypes.c_uint, ctypes.c_ulong, ctypes.c_int, ctypes.c_int, ctypes.c_int]
x.XSync.argtypes = [ctypes.c_void_p, ctypes.c_int]
x.XCloseDisplay.argtypes = [ctypes.c_void_p]
errors = []
handler_type = ctypes.CFUNCTYPE(ctypes.c_int, ctypes.c_void_p, ctypes.c_void_p)
@handler_type
def on_error(_display, _error):
    errors.append('X grab refused')
    return 0
x.XSetErrorHandler.argtypes = [handler_type]
x.XSetErrorHandler(on_error)
display = x.XOpenDisplay(sys.argv[1].encode())
assert display, 'managed display unavailable'
try:
    x.XGrabKey(display, x.XKeysymToKeycode(display, x.XStringToKeysym(b'space')), 5, x.XDefaultRootWindow(display), 0, 1, 1)
    x.XSync(display, 0)
    assert not errors, errors
    print(json.dumps({'state':'grabbed','display':sys.argv[1],'shortcut':'Control+Shift+Space'}), flush=True)
    sys.stdin.read(1)
finally:
    x.XCloseDisplay(display)
