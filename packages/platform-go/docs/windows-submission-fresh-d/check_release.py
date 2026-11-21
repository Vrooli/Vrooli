"""Exercise the actual same-boot Release body with hermetic boundary failures.

No Windows API, task, Job, account, device or target operation executes.
"""
import json
import pathlib
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[2]
source = (root / "operation_windows.go").read_text()
start = source.index("func releaseOwnedOperation(")
body = source[start:source.index("\nvar windowsOpenJob", start)]
for old, new in {
    "releaseOwnedOperation": "freshDRelease",
    "lockWindowsOwnedSubmission": "freshDLock",
    "readWindowsOperatorPayload": "freshDReadPayload",
    "readWindowsOwnedRecord": "freshDReadRoot",
    "windowsOwnedJobAbsent": "freshDAbsent",
    "windowsOwnedBootID": "freshDBoot",
    "windowsOwnedProcessStart": "freshDProcess",
    "revokeWindowsOwnedSubmission": "freshDRevoke",
    "stopWindowsOwnedTask": "freshDStop",
    "deleteWindowsOwnedTask": "freshDDeleteTask",
    "registry.DeleteKey": "freshDDeleteLedger",
    "registry.USERS": "0",
    "windowsOwnedRecordKey": "freshDKey",
}.items():
    body = body.replace(old, new)

harness = r'''
package platform
import("context";"errors";"reflect";"testing")
var freshDP windowsOwnedPayload
var freshDR windowsOwnedRootRecord
var freshDFail string
var freshDCalls []string
var freshDJobPresent, freshDLiveRoot, freshDWrongBirth, freshDNewJob bool
var freshDBoundaryError=errors.New("unknown native boundary result")
func freshDEffect(s string)error{freshDCalls=append(freshDCalls,s);if freshDFail==s{return freshDBoundaryError};return nil}
func freshDLock(context.Context,OperationIntent)(func(),error){return func(){},nil}
func freshDReadPayload(OperationIntent)(windowsOwnedPayload,error){return freshDP,nil}
func freshDReadRoot(OperationIntent)(windowsOwnedRootRecord,error){return freshDR,nil}
func freshDAbsent(string)(bool,error){name:="initial-absence";for _,c:=range freshDCalls{if c=="stop"{name="final-absence"}};e:=freshDEffect(name);return !(freshDJobPresent || (name=="final-absence" && freshDNewJob)),e}
func freshDBoot()(string,error){return freshDP.BootID,nil}
func freshDProcess(int)(string,bool,error){e:=freshDEffect("root-birth");if freshDWrongBirth{return "different",true,e};return freshDR.Start,freshDLiveRoot,e}
func freshDRevoke(OperationIntent)error{return freshDEffect("revoke")}
func freshDStop(context.Context,windowsOwnedPayload,windowsOwnedRootRecord)error{return freshDEffect("stop")}
func freshDDeleteTask(windowsOwnedPayload)error{return freshDEffect("delete-task")}
func freshDDeleteLedger(int,string)error{return freshDEffect("delete-ledger")}
func freshDKey(OperationIntent)string{return "exact-frozen-slot"}
func freshDReset(t *testing.T)OperationRef{
 freshDP=windowsTaskFixture(t);freshDR=windowsOwnedRootRecord{PID:42,Start:"123",Incarnation:"windows:42:456"}
 freshDCalls=nil;freshDFail="";freshDJobPresent=false;freshDLiveRoot=false;freshDWrongBirth=false;freshDNewJob=false
 r:=windowsRef(freshDP);r.RootPID=freshDR.PID;r.RootStartTime=freshDR.Start;r.Native.RootIncarnation=freshDR.Incarnation;return r
}
func TestFreshDSameBootReleaseKeepsRecoveryAfterEveryUnknownBoundary(t *testing.T){
 order:=[]string{"initial-absence","root-birth","revoke","stop","final-absence","delete-task","delete-ledger"}
 for n,fail:=range order{t.Run(fail,func(t *testing.T){r:=freshDReset(t);freshDFail=fail
  err:=freshDRelease(context.Background(),r)
  if !errors.Is(err,freshDBoundaryError){t.Fatalf("unknown %s hidden: %v",fail,err)}
  if !reflect.DeepEqual(freshDCalls,order[:n+1]){t.Fatalf("continued after unknown %s: %v",fail,freshDCalls)}
 })}
}
func TestFreshDSameBootReleaseRequiresNamespaceAndExactRootQuiescence(t *testing.T){
 t.Run("retained-empty-Job",func(t *testing.T){r:=freshDReset(t);freshDJobPresent=true
  if e:=freshDRelease(context.Background(),r);!errors.Is(e,ErrOperationNotQuiescent){t.Fatal(e)}
  if !reflect.DeepEqual(freshDCalls,[]string{"initial-absence"}){t.Fatal(freshDCalls)}
 })
 t.Run("live-root",func(t *testing.T){r:=freshDReset(t);freshDLiveRoot=true
  if e:=freshDRelease(context.Background(),r);!errors.Is(e,ErrOperationNotQuiescent){t.Fatal(e)}
  if !reflect.DeepEqual(freshDCalls,[]string{"initial-absence","root-birth"}){t.Fatal(freshDCalls)}
 })
 t.Run("PID-reuse",func(t *testing.T){r:=freshDReset(t);freshDWrongBirth=true
  if e:=freshDRelease(context.Background(),r);e!=nil{t.Fatal(e)}
  if freshDCalls[len(freshDCalls)-1]!="delete-ledger"{t.Fatal(freshDCalls)}
 })
 t.Run("Job-appears-after-cancel",func(t *testing.T){r:=freshDReset(t);freshDNewJob=true
  if e:=freshDRelease(context.Background(),r);!errors.Is(e,ErrOperationNotQuiescent){t.Fatal(e)}
  if freshDCalls[len(freshDCalls)-1]!="final-absence"{t.Fatal(freshDCalls)}
 })
 t.Run("wrong-root-ref",func(t *testing.T){r:=freshDReset(t);r.RootPID++
  if e:=freshDRelease(context.Background(),r);!errors.Is(e,ErrOperationIdentity){t.Fatal(e)}
  if len(freshDCalls)!=0{t.Fatal(freshDCalls)}
 })
}
'''
with tempfile.TemporaryDirectory(prefix="vrooli-submission-fresh-d-") as td:
    directory = pathlib.Path(td)
    fixture = directory / "fixture.go"
    fixture.write_text(harness + "\n" + body)
    overlay = directory / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {str(root / "fresh_d_release_test.go"): str(fixture)}}))
    result = subprocess.run(["go", "test", "-count=1", "-overlay", str(overlay), "-run", "^TestFreshD", "-v", "."], cwd=root)
    raise SystemExit(result.returncode)
