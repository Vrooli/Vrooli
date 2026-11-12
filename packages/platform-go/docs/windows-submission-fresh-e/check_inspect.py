"""Challenge the current Windows Inspect source with hermetic native boundaries.

No Windows API, native task, Job, account, token or device is operated.
"""
import json
import pathlib
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[2]
source = (root / 'operation_windows.go').read_text()
start = source.index('func inspectOwnedOperation(')
body = source[start:source.index('\nfunc stopOwnedOperation(', start)]
for old, new in {
    'inspectOwnedOperation': 'freshEInspect',
    'lookupOwnedOperation': 'freshELookup',
    'readWindowsOperatorPayload': 'freshEPayload',
    'windowsOwnedJobPopulated': 'freshEPopulated',
    'windowsOwnedBootID': 'freshEBoot',
    'windowsOwnedProcessStart': 'freshEBirth',
    'readWindowsOwnedRecord': 'freshERoot',
    'readWindowsOwnedCustody': 'freshECustody',
    'verifyWindowsOwnedCustody': 'freshEVerify',
    'readWindowsOwnedOutcome': 'freshEOutcome',
}.items():
    body = body.replace(old, new)
harness = r'''
package platform
import("context";"errors";"reflect";"testing")
var freshEP windowsOwnedPayload
var freshER windowsOwnedRootRecord
var freshEC windowsOwnedCustodyRecord
var freshET windowsOwnedTerminalRecord
var freshERef OperationRef
var freshECalls []string
var freshEFail string
var freshEHasProcesses, freshELive, freshEReused, freshENewBoot bool
var freshEUnknown = errors.New("native observation unavailable")
func freshECall(s string)error{freshECalls=append(freshECalls,s);if freshEFail==s{return freshEUnknown};return nil}
func freshELookup(context.Context,OperationIntent)(OperationRef,error){return freshERef,freshECall("lookup")}
func freshEPayload(OperationIntent)(windowsOwnedPayload,error){return freshEP,freshECall("payload")}
func freshEPopulated(string)(bool,error){return freshEHasProcesses,freshECall("population")}
func freshEBoot()(string,error){s:=freshEP.BootID;if freshENewBoot{s="87654321-1234-1234-1234-123456789abc"};return s,freshECall("boot")}
func freshEBirth(int)(string,bool,error){s:=freshER.Start;if freshEReused{s="different-birth"};return s,freshELive,freshECall("birth")}
func freshERoot(OperationIntent)(windowsOwnedRootRecord,error){return freshER,freshECall("root")}
func freshECustody(OperationIntent)(windowsOwnedCustodyRecord,error){return freshEC,freshECall("custody")}
func freshEVerify(windowsOwnedPayload,windowsOwnedRootRecord)error{return freshECall("verify")}
func freshEOutcome(OperationIntent)(windowsOwnedTerminalRecord,error){return freshET,freshECall("outcome")}
func freshEReset(t *testing.T)OperationRef{
 freshEP=windowsTaskFixture(t)
 freshER=windowsOwnedRootRecord{PID:42,Start:"116444736000000010",Incarnation:"windows:42:1000",ImageID:"windows-file:fixture",ImageHandle:123,JobHandle:456,TaskInstance:"instance"}
 freshEC=windowsOwnedCustodyRecord{Version:1,Intent:freshEP.Intent,BootID:freshEP.BootID,Root:freshER}
 freshET=windowsOwnedTerminalRecord{}
 freshERef=windowsRef(freshEP);freshERef.RootPID=freshER.PID;freshERef.RootStartTime=freshER.Start;freshERef.Native.RootIncarnation=freshER.Incarnation
 freshECalls=nil;freshEFail="";freshEHasProcesses=false;freshELive=false;freshEReused=false;freshENewBoot=false
 return freshERef
}
func TestFreshEInspectRequiresKernelCustodyAndTerminalEvidence(t *testing.T){
 t.Run("unrecorded-natural-exit-is-unknown",func(t *testing.T){r:=freshEReset(t);o,e:=freshEInspect(t.Context(),r);if e!=nil||o.State!=OperationOutcomeUnknown||o.ExitKnown||o.RootAlive||o.Populated{t.Fatal(o,e)}})
 t.Run("reused-PID-is-not-old-root",func(t *testing.T){r:=freshEReset(t);freshELive=true;freshEReused=true;o,e:=freshEInspect(t.Context(),r);if e!=nil||o.RootAlive||o.ExitKnown||o.State!=OperationOutcomeUnknown{t.Fatal(o,e)}})
 t.Run("new-boot-never-queries-old-PID",func(t *testing.T){r:=freshEReset(t);freshENewBoot=true;o,e:=freshEInspect(t.Context(),r);if e!=nil||o.RootAlive{t.Fatal(o,e)};for _,s:=range freshECalls{if s=="birth"{t.Fatal("queried old PID",freshECalls)}}})
 t.Run("populated-prepared-is-not-running",func(t *testing.T){r:=freshEReset(t);freshEHasProcesses=true;freshEC=windowsOwnedCustodyRecord{};o,e:=freshEInspect(t.Context(),r);if e!=nil||o.State!=OperationOutcomeUnknown||o.ExitKnown{t.Fatal(o,e)};if freshECalls[len(freshECalls)-1]!="custody"{t.Fatal(freshECalls)}})
 t.Run("claimed-still-requires-native-verification",func(t *testing.T){r:=freshEReset(t);freshEHasProcesses=true;freshEFail="verify";o,e:=freshEInspect(t.Context(),r);if !errors.Is(e,freshEUnknown)||o.State==OperationRunning{t.Fatal(o,e)}})
 t.Run("verified-claimed-role-is-running",func(t *testing.T){r:=freshEReset(t);freshEHasProcesses=true;freshELive=true;o,e:=freshEInspect(t.Context(),r);if e!=nil||o.State!=OperationRunning||!o.RootAlive||!o.Populated||o.ExitKnown{t.Fatal(o,e)}})
 for _,exit:=range []uint32{0,37}{t.Run("exact-terminal",func(t *testing.T){r:=freshEReset(t);freshET=windowsOwnedTerminalRecord{Version:1,Intent:freshEP.Intent,BootID:freshEP.BootID,Root:freshER,ExitCode:exit};o,e:=freshEInspect(t.Context(),r);if e!=nil||o.State!=OperationStoppedOutcomeKnown||!o.ExitKnown||o.ExecMainStatus!=int(exit){t.Fatal(o,e)};if (o.Result=="success")!=(exit==0){t.Fatal(o)}})}
 t.Run("foreign-terminal-is-error",func(t *testing.T){r:=freshEReset(t);freshET=windowsOwnedTerminalRecord{Version:1,Intent:freshEP.Intent,BootID:freshEP.BootID,Root:freshER};freshET.Root.Start="foreign";o,e:=freshEInspect(t.Context(),r);if !errors.Is(e,ErrOperationIdentity)||o.ExitKnown{t.Fatal(o,e)}})
 t.Run("foreign-claimed-custody-is-error",func(t *testing.T){r:=freshEReset(t);freshEHasProcesses=true;freshEC.Root.JobHandle++;o,e:=freshEInspect(t.Context(),r);if !errors.Is(e,ErrOperationIdentity)||o.State==OperationRunning{t.Fatal(o,e)}})
 t.Run("foreign-root-ref-stops-before-native-observation",func(t *testing.T){r:=freshEReset(t);r.RootPID++;_,e:=freshEInspect(t.Context(),r);if !errors.Is(e,ErrOperationIdentity)||!reflect.DeepEqual(freshECalls,[]string{"lookup"}){t.Fatal(e,freshECalls)}})
}
func TestFreshEInspectPropagatesEveryUnavailableNativeObservation(t *testing.T){
 for _,populated:=range []bool{false,true}{
  order:=[]string{"lookup","payload","population","boot","birth","outcome","root"};if populated{order=[]string{"lookup","payload","population","boot","birth","root","custody","verify"}}
  for n,fail:=range order{t.Run(fail,func(t *testing.T){r:=freshEReset(t);freshEHasProcesses=populated;freshEFail=fail;o,e:=freshEInspect(t.Context(),r);if !errors.Is(e,freshEUnknown)||o.ExitKnown||o.State==OperationRunning{t.Fatal(o,e)};if !reflect.DeepEqual(freshECalls,order[:n+1]){t.Fatal(freshECalls,order[:n+1])}})}
 }
}
'''
with tempfile.TemporaryDirectory(prefix='vrooli-submission-fresh-e-') as td:
    directory = pathlib.Path(td)
    fixture = directory / 'fixture.go'
    fixture.write_text(harness + '\n' + body)
    overlay = directory / 'overlay.json'
    overlay.write_text(json.dumps({'Replace': {str(root / 'fresh_e_inspect_test.go'): str(fixture)}}))
    result = subprocess.run(['go', 'test', '-count=1', '-overlay', str(overlay), '-run', '^TestFreshE', '-v', '.'], cwd=root)
    raise SystemExit(result.returncode)
