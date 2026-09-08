"""Physical desktop semantic fixture; all text mutation uses installed owner CLI."""
import ctypes as c
import json
import os
from pathlib import Path
import select
import subprocess
import tempfile
import urllib.request
import re

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
   identity=json.loads((base/'portal-test-account.json').read_text())
   def rpc(service,method,body,token=None):
    headers={'Content-Type':'application/json','Connect-Protocol-Version':'1'}
    if token:headers['Authorization']='Bearer '+token
    request=urllib.request.Request('http://127.0.0.1:17476/vrooli.portal.v1.surfaces.'+service+'/'+method,data=json.dumps(body).encode(),headers=headers)
    with urllib.request.urlopen(request,timeout=40) as response:
     assert response.headers['Cache-Control']=='no-store'
     return json.load(response)
   login=rpc('OperatorSessionService','Login',{'email':identity['email'],'password':identity['password']})
   token=login['tokens']['accessToken']
   def snake(value):
    if isinstance(value,dict):return {re.sub(r'(?<!^)(?=[A-Z])','_',key).lower():snake(v) for key,v in value.items()}
    if isinstance(value,list):return [snake(v) for v in value]
    return value
   def call(operation,body,extra=()):
    method={'open':'Open','applications':'Applications','run-flow':'RunFlow','stop':'Stop'}[operation]
    return snake(rpc('DesktopSessionService',method,body,token))
   opened=call('open',{'surface':config['helper']['surface'],'ttl_seconds':30,'control':True})
   try:
    catalog=call('applications',{'session':opened['session']})
    applications=[a for a in catalog['applications'] if a.get('name')=='gtk_fixture.py' and a.get('process_id')==child.pid]
    assert len(applications)==1
    first='日本語 العربية café e\u0301 🧪'; second=' — done'
    steps=[{'id':'first','kind':'desktop-text','target':'go-unicode-entry','arguments':{'window':'Go AT-SPI Unicode fixture','text':first,'position':2}}, {'id':'second','kind':'desktop-text','target':'go-unicode-entry','arguments':{'window':'Go AT-SPI Unicode fixture','text':second,'position':2+len(first)}}]
    request={'session':opened['session'],'run_id':'physical-flow','application_id':applications[0]['application_id'],'application_revision':catalog['revision'],'flow':{'transport':'desktop','steps':steps}}
    result=call('run-flow',request);duplicate=call('run-flow',request)
    assert result['disposition']=='passed' and result['confirmed']==2
    assert result==duplicate
    assert persisted.read_text()=='é|'+first+second+'tail'
    evidence={'target':opened['session']['surface']['target'],'desktop_session':opened['session']['desktop_session_id'],'fixture_pid':child.pid,'two_step_flow_passed':True,'unicode_persisted_exactly_once':True,'duplicate_run_record_equal':True,'flow_record':result}
   finally:call('stop',{'session':opened['session']})
   rpc('OperatorSessionService','Logout',{'accessToken':token})
   evidence['portal_account_authorized']=True; evidence['logout_succeeded']=True
   evidence['stop_succeeded']=True
   evidence['application_discovery']=True; evidence['window_resolution_unique']=True
   Path(__file__).with_suffix('.json').write_text(json.dumps(evidence,indent=2)+'\n');print(json.dumps(evidence))
  finally:
   child.terminate()
   try:child.wait(timeout=3)
   except subprocess.TimeoutExpired:child.kill();child.wait()
finally:
 x.XSetInputFocus(display,focus,revert,0);x.XSync(display,0);x.XCloseDisplay(display)
