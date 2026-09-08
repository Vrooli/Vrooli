"""Physical desktop semantic fixture; all text mutation uses installed owner CLI."""
import ctypes as c
import json
import os
from pathlib import Path
import select
import subprocess
import tempfile

base=Path('/run/user/1000/vrooli-desktop-owner')
config=json.loads((base/'owner.json').read_text())
env=dict(os.environ,DISPLAY=':'+str(config['helper']['display']),XAUTHORITY=config['helper']['xauthority_file'],DBUS_SESSION_BUS_ADDRESS='unix:path=/run/user/1000/bus',XDG_RUNTIME_DIR='/run/user/1000',GSETTINGS_BACKEND='memory',GIO_USE_VFS='local')
os.environ['XAUTHORITY']=env['XAUTHORITY']
x=c.CDLL('libX11.so.6');x.XOpenDisplay.restype=c.c_void_p;x.XOpenDisplay.argtypes=[c.c_char_p]
x.XGetInputFocus.argtypes=[c.c_void_p,c.POINTER(c.c_ulong),c.POINTER(c.c_int)]
x.XSetInputFocus.argtypes=[c.c_void_p,c.c_ulong,c.c_int,c.c_ulong]
x.XSync.argtypes=[c.c_void_p,c.c_int];x.XCloseDisplay.argtypes=[c.c_void_p]
display=x.XOpenDisplay(env['DISPLAY'].encode());assert display
focus,revert=c.c_ulong(),c.c_int();x.XGetInputFocus(display,c.byref(focus),c.byref(revert))
try:
 with tempfile.TemporaryDirectory(prefix='unicode-fixture-',dir=base) as temporary:
  work=Path(temporary);persisted=work/'text.txt'
  fixture=Path('/home/matthalloran8/Vrooli/scenarios/device-control/api/internal/native/atspi/testdata/gtk_fixture.py')
  child=subprocess.Popen(['/usr/bin/python3',str(fixture),str(persisted)],env=env,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True)
  try:
   assert select.select([child.stdout],[],[],5)[0], 'GTK fixture readiness timeout'
   ready=json.loads(child.stdout.readline());assert ready['pid']==child.pid
   def call(operation,body,extra=()):
    request=work/(operation+'.json');request.write_text(json.dumps(body));request.chmod(0o600)
    result=subprocess.run(['device-control','desktop',operation,'--socket',str(base/'desktop-owner.sock'),'--request',str(request),'--json',*extra],capture_output=True,text=True,timeout=15)
    if result.returncode:raise RuntimeError(operation+': '+result.stderr[:500])
    return json.loads(result.stdout)
   opened=call('open',{'surface':config['helper']['surface'],'ttl_seconds':30,'control':True})
   try:
    catalog=call('applications',{'session':opened['session']})
    applications=[a for a in catalog['applications'] if a.get('name')=='gtk_fixture.py' and a.get('process_id')==child.pid]
    assert len(applications)==1
    observed=call('observe',{'session':opened['session'],'application_id':applications[0]['application_id'],'application_revision':catalog['revision']},('--output',str(work/'image.png')))
    semantic=observed['semantic'];entries=[e for e in semantic['elements'] if e.get('name')=='go-unicode-entry' and e.get('editable')]
    assert len(entries)==1
    windows=[w for w in semantic['elements'] if w.get('element_id')==entries[0]['window_id']]
    assert len(windows)==1 and windows[0]['name']=='Go AT-SPI Unicode fixture'
    selector={'observation_revision':semantic['revision'],'window_id':windows[0]['element_id'],'name':'go-unicode-entry','editable_only':True}
    resolved=call('resolve',{'session':opened['session'],'selector':selector})
    assert resolved['disposition']=='DISPOSITION_UNIQUE'
    assert resolved['element_ids']==[entries[0]['element_id']]
    absent=call('resolve',{'session':opened['session'],'selector':dict(selector,name='nonexistent fixture field')})
    assert absent['disposition']=='DISPOSITION_ABSENT' and not absent.get('element_ids')
    text='日本語 العربية café e\u0301 🧪'
    action={'session':opened['session'],'command_id':'unicode-once','geometry_revision':observed['geometry_revision'],'action':{'text':{'text':text,'element_id':resolved['element_ids'][0],'observation_revision':semantic['revision'],'position':2}}}
    receipt=call('act',action);duplicate=call('act',action)
    assert receipt['receipt']['outcome']=='applied';assert receipt==duplicate
    assert persisted.read_text()=='é|'+text+'tail'
    evidence={'target':opened['session']['surface']['target'],'desktop_session':opened['session']['desktop_session_id'],'fixture_pid':child.pid,'width':observed['width'],'height':observed['height'],'unicode_persisted_exactly_once':True,'duplicate_receipt_equal':True,'receipt':receipt['receipt']}
   finally:call('stop',{'session':opened['session']})
   evidence['stop_succeeded']=True
   evidence['application_discovery']=True; evidence['window_resolution_unique']=True; evidence['absent_resolution']=True
   Path(__file__).with_suffix('.json').write_text(json.dumps(evidence,indent=2)+'\n');print(json.dumps(evidence))
  finally:
   child.terminate()
   try:child.wait(timeout=3)
   except subprocess.TimeoutExpired:child.kill();child.wait()
finally:
 x.XSetInputFocus(display,focus,revert,0);x.XSync(display,0);x.XCloseDisplay(display)
