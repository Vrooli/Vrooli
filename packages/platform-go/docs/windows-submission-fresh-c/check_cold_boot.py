"""Project actual Windows stop flow into hermetic platform fakes; no native effects."""
import json
import pathlib
import subprocess
import tempfile
root = pathlib.Path(__file__).resolve().parents[2]
source = (root / 'operation_windows.go').read_text()
start = source.index('func stopOwnedOperation(')
end = source.index('\nvar windowsOpenJob', start)
body = source[start:end]
for old, new in {
 'stopOwnedOperation':'freshCStop', 'releaseOwnedOperation':'freshCRelease', 'windowsOwnedJobAbsent':'freshCJobAbsent', 'windowsOwnedRecordKey':'freshCRecordKey', 'windowsOwnedProcessStart':'freshCOldPID', 'deleteWindowsOwnedTask':'freshCDeleteTask', 'registry.DeleteKey':'freshCDeleteLedger', 'registry.USERS':'0', 'lockWindowsOwnedSubmission':'freshCLock',
 'readWindowsOperatorPayload':'freshCReadPayload','readWindowsOwnedRecord':'freshCReadRoot',
 'windowsOwnedBootID':'freshCBoot','revokeWindowsOwnedSubmission':'freshCRevoke',
 'stopWindowsOwnedTask':'freshCTaskStop','openWindowsOwnedJobAccess':'freshCOpenJob',
 'validateWindowsOwnedJobHandle':'freshCValidateJob','windowsOwnedJobPopulated':'freshCPopulated',
 'windows.':'freshCNative.',
}.items(): body=body.replace(old,new)
harness=r'''
package platform
import("context";"testing";"time")
var freshCPayload windowsOwnedPayload
var freshCRoot windowsOwnedRootRecord
func freshCRecordKey(OperationIntent)string{return "exact-old-invocation"}
var freshCAbsences []bool
var freshCCalls []string
func freshCJobAbsent(string)(bool,error){freshCCalls=append(freshCCalls,"absent"); if len(freshCAbsences)==0 {panic("unexpected Job query")};v:=freshCAbsences[0];freshCAbsences=freshCAbsences[1:];return v,nil}
func freshCOldPID(int)(string,bool,error){panic("reused old PID must not be queried")}
func freshCDeleteTask(windowsOwnedPayload)error{freshCCalls=append(freshCCalls,"delete-task");return nil}
func freshCDeleteLedger(int,string)error{freshCCalls=append(freshCCalls,"delete-ledger");return nil}
var freshCNative=struct{READ_CONTROL uint32;TerminateJobObject func(uint64,uint32)error;CloseHandle func(uint64)error}{0,func(uint64,uint32)error{panic("old PID/Job must never be killed")},func(uint64)error{return nil}}
func freshCLock(context.Context,OperationIntent)(func(),error){return func(){},nil}
func freshCReadPayload(OperationIntent)(windowsOwnedPayload,error){return freshCPayload,nil}
func freshCReadRoot(OperationIntent)(windowsOwnedRootRecord,error){return freshCRoot,nil}
func freshCBoot()(string,error){return "22222222-2222-4222-8222-222222222222",nil}
func freshCRevoke(OperationIntent)error{freshCCalls=append(freshCCalls,"revoke");return nil}
func freshCTaskStop(_ context.Context,_ windowsOwnedPayload,r windowsOwnedRootRecord)error{if r!=(windowsOwnedRootRecord{}) {panic("old root/GUID authority propagated")};freshCCalls=append(freshCCalls,"stop-task");return nil}
func freshCOpenJob(string,uint32)(uint64,error){return 0,nil}
func freshCValidateJob(uint64,windowsOwnedPayload)error{panic("no old kernel Job exists")}
func freshCPopulated(string)(bool,error){return false,nil}
func TestFreshCColdBootCanRetireFrozenNoTriggerSubmission(t *testing.T){
 freshCPayload=windowsTaskFixture(t)
 freshCPayload.BootID="11111111-1111-4111-8111-111111111111"
 freshCRoot=windowsOwnedRootRecord{}
 ref:=windowsRef(freshCPayload)
 // Actual lookup builds the same ref from retained payload, not current boot.
 // Resource same-boot cleanup then calls Stop before it may persist quiescence.
 freshCAbsences=[]bool{true,true}; freshCCalls=nil
 receipt,err:=freshCStop(context.Background(),ref)
 if err!=nil || !receipt.NativeQuiescent {t.Fatalf("old boot frozen submission stranded despite absent Job and canceled exact task: %+v %v",receipt,err)}
 for _,absence:=range [][]bool{{false},{true,false}} {
  freshCAbsences=absence;freshCCalls=nil
  if _,err:=freshCStop(context.Background(),ref);err==nil {t.Fatal("new Job accepted as old-boot quiescence")}
 }
 freshCRoot=windowsOwnedRootRecord{PID:7331,Start:"old",Incarnation:"old",TaskInstance:"old-GUID"}
 ref.RootPID=freshCRoot.PID;ref.RootStartTime=freshCRoot.Start;ref.Native.RootIncarnation=freshCRoot.Incarnation
 freshCAbsences=[]bool{true,true};freshCCalls=nil
 if _,err:=freshCStop(context.Background(),ref);err!=nil {t.Fatalf("old-root stop: %v",err)}
 freshCAbsences=[]bool{true,true};freshCCalls=nil
 if err:=freshCRelease(context.Background(),ref);err!=nil {t.Fatalf("old-root release: %v",err)}
 want:=[]string{"absent","revoke","stop-task","absent","delete-task","delete-ledger"}
 if len(freshCCalls)!=len(want){t.Fatalf("release effects: %v",freshCCalls)}
 for n:=range want {if freshCCalls[n]!=want[n]{t.Fatalf("release order: %v",freshCCalls)}}
}
'''
with tempfile.TemporaryDirectory(prefix='vrooli-submission-fresh-c-') as td:
 d=pathlib.Path(td); fixture=d/'fixture.go';fixture.write_text(harness+'\n'+body+'\n')
 overlay=d/'overlay.json';overlay.write_text(json.dumps({'Replace':{str(root/'fresh_c_coldboot_test.go'):str(fixture)}}))
 result=subprocess.run(['go','test','-overlay',str(overlay),'-run','^TestFreshCColdBoot','-v','.'],cwd=root)
 raise SystemExit(result.returncode)
