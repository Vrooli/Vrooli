"""Exercise deployed local companion binding with an unmapped native X window.
No screenshot, accessibility text, focus change or input. Not an Electron journey.
"""
import ctypes
import datetime
import http.client
import json
import os
import pathlib
import socket
import subprocess
import tempfile
import uuid

ROOT = pathlib.Path('/run/user/1000/vrooli-desktop-owner')
OUT = pathlib.Path(__file__).with_name('native-companion-binding-live.json')
config = json.loads((ROOT / 'helper.json').read_text())
os.environ['XAUTHORITY'] = config['xauthority_file']
x = ctypes.CDLL('libX11.so.6')
x.XOpenDisplay.argtypes = [ctypes.c_char_p]; x.XOpenDisplay.restype = ctypes.c_void_p
x.XDefaultRootWindow.argtypes = [ctypes.c_void_p]; x.XDefaultRootWindow.restype = ctypes.c_ulong
x.XCreateSimpleWindow.argtypes = [ctypes.c_void_p, ctypes.c_ulong, ctypes.c_int, ctypes.c_int, ctypes.c_uint, ctypes.c_uint, ctypes.c_uint, ctypes.c_ulong, ctypes.c_ulong]; x.XCreateSimpleWindow.restype = ctypes.c_ulong
x.XSync.argtypes = [ctypes.c_void_p, ctypes.c_int]
x.XDestroyWindow.argtypes = [ctypes.c_void_p, ctypes.c_ulong]
x.XCloseDisplay.argtypes = [ctypes.c_void_p]
display = x.XOpenDisplay((':' + str(config['display'])).encode())
assert display, 'configured X server unavailable'
window = x.XCreateSimpleWindow(display, x.XDefaultRootWindow(display), 0, 0, 1, 1, 0, 0, 0)
x.XSync(display, 0)
assert window

def camel(value):
    if isinstance(value, dict):
        return {k.split('_')[0] + ''.join(part.title() for part in k.split('_')[1:]): camel(v) for k, v in value.items()}
    return value

class OwnerConnection(http.client.HTTPConnection):
    def connect(self):
        self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.sock.settimeout(5)
        self.sock.connect(str(ROOT / 'desktop-owner.sock'))

def owner_call(method, body):
    connection = OwnerConnection('desktop-owner', timeout=5)
    try:
        connection.request('POST', '/vrooli.device_control.v1.desktop.DesktopOwnerService/' + method,
            json.dumps(body),
            {'Content-Type': 'application/json', 'Connect-Protocol-Version': '1'})
        response = connection.getresponse()
        return {'http_status': response.status, 'body': json.loads(response.read(16385))}
    finally:
        connection.close()

def capture(session, xid):
    return owner_call('CaptureCompanionActivation', {'session': camel(session), 'companionWindow': str(xid)})

data = {'observed_at': datetime.datetime.now(datetime.timezone.utc).isoformat(), 'scope': __doc__,
        'native_window_mapped': False, 'request': {'surface': config['surface'], 'control': False, 'ttl_seconds': 30, 'request_id': str(uuid.uuid4())}}
def save(): OUT.write_text(json.dumps(data, indent=2) + '\n')
save()
with tempfile.TemporaryDirectory() as td:
    def cli(op, body):
        path = pathlib.Path(td) / 'request.json'; path.write_text(json.dumps(body)); path.chmod(0o600)
        result = subprocess.run(['device-control', 'desktop', op, '--socket', str(ROOT / 'desktop-owner.sock'), '--request', str(path), '--json'], capture_output=True, text=True, timeout=45)
        if result.returncode: raise RuntimeError(result.stderr.strip())
        return json.loads(result.stdout)
    session = None
    try:
        data['open'] = cli('open', data['request']); session = data['open']['session']; save()
        data['capture'] = capture(session, window); save()
        assert data['capture']['http_status'] == 200, data['capture']
        assert data['capture']['body']['contextId']
        bounds = data['capture']['body']['sourceBounds']
        assert 0 < bounds['width'] <= 65535 and 0 < bounds['height'] <= 65535
        data['wrong_window'] = capture(session, x.XDefaultRootWindow(display))
        assert data['wrong_window']['http_status'] != 200
        newer = capture(session, window)
        assert newer['http_status'] == 200
        data['newer_capture'] = newer
        old_request = {'session': camel(session), 'contextId': data['capture']['body']['contextId']}
        new_request = {'session': camel(session), 'contextId': newer['body']['contextId']}
        data['delete_old'] = owner_call('DeleteActivation', old_request)
        assert data['delete_old']['http_status'] == 200
        data['read_newer'] = owner_call('ReadActivation', new_request)
        assert data['read_newer']['http_status'] == 200
        assert data['read_newer']['body']['sourceBounds'] == newer['body']['sourceBounds']
        data['delete_current'] = owner_call('DeleteActivation', new_request)
        assert data['delete_current']['http_status'] == 200
        data['read_deleted'] = owner_call('ReadActivation', new_request)
        assert data['read_deleted']['http_status'] != 200
        data['status'] = 'passed'
    except Exception as error:
        data['status'] = 'failed'; data['error'] = str(error)
        if session is None:
            try:
                data['reconcile'] = cli('reconcile-open', data['request']); session = data['reconcile'].get('session')
            except Exception as recovery: data['reconcile_error'] = str(recovery)
    finally:
        if session:
            try:
                data['stop'] = cli('stop', {'session': session}); data['cleanup'] = cli('read-cleanup', {'session': session})
            except Exception as cleanup: data['cleanup_error'] = str(cleanup)
        x.XDestroyWindow(display, window); x.XSync(display, 0); x.XCloseDisplay(display)
        data['native_window_destroyed'] = True; save()
print(json.dumps(data))
if data.get('status') != 'passed' or not data.get('cleanup', {}).get('released'): raise SystemExit(1)
