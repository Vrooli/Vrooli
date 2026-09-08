"""Own a managed X11/Electron profile for presentation checks; no routed-data claim."""
import hashlib
import os
import tempfile
import json
import pathlib
import sys
import subprocess
import select
import urllib.error
import urllib.request

BASE = 'http://127.0.0.1:19925/api/v1/livedesktop/sessions'
ARTIFACT = pathlib.Path(__file__).with_name('native-companion-live.json')
APP = pathlib.Path(os.environ.get(
    'VROOLI_ELECTRON_INTEGRATION_APP',
    '/home/matthalloran8/Vrooli/scenarios/portal/platforms/electron/dist-electron/linux-unpacked/portal-desktop',
))

def call(url, body=None, method='POST'):
    request = urllib.request.Request(url, data=json.dumps(body).encode() if body is not None else None,
        headers={'Content-Type': 'application/json'}, method=method)
    try:
        with urllib.request.urlopen(request, timeout=55) as response:
            return response.status, json.load(response)
    except urllib.error.HTTPError as error:
        return error.code, json.load(error)

if '--fixture-backing' in sys.argv:
    # Test-only compositor reference for the exact xmessage client selected by
    # the managed integration fixture. Never used by production image capture.
    import ctypes as c
    from ctypes.util import find_library
    x=c.CDLL(find_library('X11')); composite=c.CDLL(find_library('Xcomposite'))
    x.XOpenDisplay.argtypes=[c.c_char_p];x.XOpenDisplay.restype=c.c_void_p
    x.XSync.argtypes=[c.c_void_p,c.c_int];x.XCloseDisplay.argtypes=[c.c_void_p]
    for name in ['XCompositeRedirectWindow','XCompositeUnredirectWindow']:
        getattr(composite,name).argtypes=[c.c_void_p,c.c_ulong,c.c_int]
    errors=[]
    handler=c.CFUNCTYPE(c.c_int,c.c_void_p,c.c_void_p)(lambda *_: errors.append(True) or 0)
    x.XSetErrorHandler.argtypes=[type(handler)];x.XSetErrorHandler(handler)
    display=x.XOpenDisplay(None)
    if not display:raise RuntimeError('fixture display unavailable')
    window=int(sys.argv[sys.argv.index('--fixture-backing')+1])
    try:
        composite.XCompositeRedirectWindow(display,window,0);x.XSync(display,0)
        if errors:raise RuntimeError('fixture backing unavailable')
        print('ready',flush=True)
        sys.stdin.read()
    finally:
        composite.XCompositeUnredirectWindow(display,window,0);x.XSync(display,0);x.XCloseDisplay(display)
    sys.exit(0)

if len(sys.argv) > 1 and sys.argv[1] == 'stop':
    data = json.loads(ARTIFACT.read_text())
    data['cleanup'] = call(BASE + '/' + data['session_id'], method='DELETE')
    ARTIFACT.write_text(json.dumps(data, indent=2) + '\n')
    print(json.dumps(data['cleanup']))
    sys.exit(0 if data['cleanup'][0] == 200 else 1)

if '--selected-desktop-context' in sys.argv:
    # Explicit test-owned target adapter launch on the helper's selected desktop.
    config = json.loads(pathlib.Path('/run/user/1000/vrooli-desktop-owner/helper.json').read_text())
    def camel(value):
        if isinstance(value, dict):
            return {k.split('_')[0] + ''.join(part.title() for part in k.split('_')[1:]): camel(v) for k, v in value.items()}
        return value
    with tempfile.TemporaryDirectory(prefix='portal-context-integration-') as folder:
        source = pathlib.Path(folder) / 'source.json'
        source.write_text(json.dumps({'version': 2, 'socket': '/run/user/1000/vrooli-desktop-owner/desktop-owner.sock', 'surface': camel(config['surface']), 'includeImage': '--image' in sys.argv}))
        source.chmod(0o600)
        env = dict(os.environ, DISPLAY=':' + str(config['display']), XAUTHORITY=config['xauthority_file'],
            VROOLI_PORTAL_CONTEXT_INTEGRATION='1', VROOLI_PORTAL_CONTEXT_IMAGE='1' if '--image' in sys.argv else '0', VROOLI_DESKTOP_ACTIVATION_CONFIG=str(source),
            VROOLI_ELECTRON_INTEGRATION_APP=str(APP), GOCACHE='/tmp/portal-desktop-go-cache')
        result = subprocess.run(['go', 'test', './target', '-run', '^TestStartPortalCompanionContextOnSelectedDesktop$', '-count=1', '-v'],
            cwd='/home/matthalloran8/Vrooli/scenarios/scenario-to-desktop/api', env=env, timeout=90)
        sys.exit(result.returncode)

previous = json.loads(ARTIFACT.read_text()) if ARTIFACT.exists() else {}
history = previous.pop('previous_attempts', [])
if previous: history.append(previous)

status, session = call(BASE, {'scenario_name': 'portal', 'width': 1280, 'height': 900, 'platform': 'linux'})
assert status == 201, session
sid = session['id']
data = {'session_id': sid, 'display_id': session.get('display_id'), 'previous_attempts': history, 'scope': 'Managed desktop and isolated Electron profile; no routed application-data isolation claim'}
ARTIFACT.write_text(json.dumps(data, indent=2) + '\n')
grabber = None
try:
    if '--reserve-shortcut' in sys.argv:
        grabber = subprocess.Popen([sys.executable, str(pathlib.Path(__file__).with_name('native-shortcut-grab.py')), session['display_id']], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
        assert select.select([grabber.stdout], [], [], 5)[0], 'shortcut reservation timeout'
        reservation = json.loads(grabber.stdout.readline())
        assert reservation['state'] == 'grabbed', reservation
        data['shortcut_reservation'] = reservation
    status, reply = call(BASE + '/' + sid + '/launch-electron-validation', {
        'app_path': str(APP), 'scenario_name': 'portal', 'context_id': 'portal-native-' + sid,
        'artifact_digest': hashlib.sha256(APP.read_bytes()).hexdigest(), 'target_id': sid,
        'journey_id': 'portal-native-presentation', 'profile_id': 'local-native-presentation',
        'isolation_lease_id': sid, 'renderer_url_prefix': 'http://127.0.0.1:24965'})
    data.update(status=status, launch=reply)
    assert status == 200, reply
except BaseException:
    data['cleanup'] = call(BASE + '/' + sid, method='DELETE')
    raise
finally:
    if grabber is not None:
        grabber.stdin.close()
        try: grabber.wait(timeout=3)
        except subprocess.TimeoutExpired: grabber.kill(); grabber.wait()
        data['shortcut_reservation_released'] = True
    ARTIFACT.write_text(json.dumps(data, indent=2) + '\n')
print(json.dumps({'session_id': sid, 'status': status}))
