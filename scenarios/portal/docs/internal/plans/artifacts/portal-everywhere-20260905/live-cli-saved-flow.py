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
    first='日本語 العربية café e\u0301 🧪'; second=' — done'
    steps=[{'id':'first','kind':'desktop-text','target':'go-unicode-entry','arguments':{'window':'Go AT-SPI Unicode fixture','text':first,'position':2}}, {'id':'second','kind':'desktop-text','target':'go-unicode-entry','arguments':{'window':'Go AT-SPI Unicode fixture','text':second,'position':2+len(first)}}]
    steps.append({'id':'verify','kind':'desktop-text-assert','target':'go-unicode-entry','arguments':{'window':'Go AT-SPI Unicode fixture','text':'é|'+first+second+'tail'}})
    request={'session':opened['session'],'run_id':'physical-flow','application_id':applications[0]['application_id'],'application_revision':catalog['revision'],'flow':{'transport':'desktop','steps':steps}}
    result=call('run-flow',request);duplicate=call('run-flow',request)
    assert result['disposition']=='passed' and result['confirmed']==3
    assert result==duplicate
    assert persisted.read_text()=='é|'+first+second+'tail'
    promoted=call('promote-flow',{'session':opened['session'],'source_session':opened['session'],'source_run_id':request['run_id'],'context_key':'gtk-unicode:v1'})
    fetched=call('get-saved-flow',{'session':opened['session'],'id':promoted['id'],'version':promoted['version'],'context_key':'gtk-unicode:v1'})
    assert fetched==promoted
    child.terminate();child.wait(timeout=3)
    child=subprocess.Popen(['/usr/bin/python3',str(fixture),str(persisted)],env=env,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL,text=True)
    assert select.select([child.stdout],[],[],5)[0], 'fresh GTK fixture readiness timeout'
    ready=json.loads(child.stdout.readline());assert ready['pid']==child.pid
    catalog=call('applications',{'session':opened['session']})
    applications=[a for a in catalog['applications'] if a.get('name')=='gtk_fixture.py' and a.get('process_id')==child.pid]
    assert len(applications)==1
    replay_request={'session':opened['session'],'id':promoted['id'],'version':promoted['version'],'context_key':'gtk-unicode:v1','run_id':'saved-physical-replay','application_id':applications[0]['application_id'],'application_revision':catalog['revision']}
    replayed=call('run-saved-flow',replay_request); replay_duplicate=call('run-saved-flow',replay_request)
    assert replayed['disposition']=='passed' and replayed['confirmed']==3
    assert replayed==replay_duplicate
    assert persisted.read_text()=='é|'+first+second+'tail'
    evidence={'target':opened['session']['surface']['target'],'desktop_session':opened['session']['desktop_session_id'],'fixture_pid':child.pid,'two_insertions_and_terminal_assertion_passed':True,'saved_id':promoted['id'],'saved_version':promoted['version'],'fresh_application_saved_replay_passed':True,'saved_duplicate_record_equal':True,'saved_flow_record':replayed,'unicode_persisted_exactly_once':True,'duplicate_run_record_equal':True,'flow_record':result}
   finally:call('stop',{'session':opened['session']})
   evidence['stop_succeeded']=True
   evidence['application_discovery']=True; evidence['window_resolution_unique']=True
   Path(__file__).with_suffix('.json').write_text(json.dumps(evidence,indent=2)+'\n');print(json.dumps(evidence))
  finally:
   child.terminate()
   try:child.wait(timeout=3)
   except subprocess.TimeoutExpired:child.kill();child.wait()
finally:
 x.XSetInputFocus(display,focus,revert,0);x.XSync(display,0);x.XCloseDisplay(display)
