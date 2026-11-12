"""Project the current native Stop wrapper; substitute only COM syscalls.
No Windows effect occurs. The actual wrapper must QI IRegisteredTask, call Stop
at slot 23 with flags zero, reject S_FALSE, and release both acquired references.
"""
import json,pathlib,subprocess,tempfile
root=pathlib.Path(__file__).resolve().parents[2]
source=(root/'windows_owned_task_windows.go').read_text()
start=source.index('type windowsRegisteredTask struct')
body=source[start:]
body=body.replace('stopWindowsOwnedRegisteredTask','independentRegisteredStop').replace('windows.GUID','independentGUID').replace('syscall.SyscallN','independentSyscall')
harness='''package platform
import("fmt";"testing";"unsafe")
type windowsTaskHRESULT uint32
func(e windowsTaskHRESULT) Error()string{return fmt.Sprintf("HRESULT %x",uint32(e))}
type independentGUID struct{Data1 uint32;Data2,Data3 uint16;Data4 [8]byte}
type taskDispatch struct{vtable *[7]uintptr}
var independentNative *windowsRegisteredTask
var independentHR uintptr
var independentCalls []uintptr
var independentDispatchReleases int
func(*taskDispatch) release(){independentDispatchReleases++}
func withTaskScheduler(fn func(*taskDispatch)error)error{return fn(&taskDispatch{&[7]uintptr{1}})}
func taskRegistered(folder *taskDispatch,name string)(*taskDispatch,error){if name!="exact-task"{panic("wrong task")};return folder,nil}
func independentSyscall(fn uintptr,args ...uintptr)(uintptr,uintptr,error){
 independentCalls=append(independentCalls,fn)
 switch fn {
 case 1:
  iid:=(*independentGUID)(unsafe.Pointer(args[1]))
  if *iid!=(independentGUID{0x9c86f320,0xdee3,0x4dd1,[8]byte{0xb9,0x72,0xa3,0x03,0xf2,0x6b,0x06,0x1e}}){panic("wrong IID")}
  *(**windowsRegisteredTask)(unsafe.Pointer(args[2]))=independentNative
  return 0,0,nil
 case 23:if len(args)!=2||args[1]!=0{panic("wrong Stop flags")};return independentHR,0,nil
 case 2:return 0,0,nil
 default:panic("wrong native slot")
 }
}
func TestFreshRegisteredStopRejectsSFalse(t *testing.T){
 for _,hr:=range []uintptr{0,1,0x41301,0x80070005}{
  independentNative=&windowsRegisteredTask{&[25]uintptr{2:2,23:23}}
  independentHR=hr;independentCalls=nil;independentDispatchReleases=0
  err:=independentRegisteredStop("exact-task")
  if (err==nil)!=(hr==0){t.Fatalf("HRESULT %x became %v",hr,err)}
  if fmt.Sprint(independentCalls)!="[1 23 2]"||independentDispatchReleases!=1{t.Fatal("QI/Stop/release boundary",independentCalls,independentDispatchReleases)}
 }
}
'''
with tempfile.TemporaryDirectory(prefix='vrooli-stop-status-') as temporary:
 directory=pathlib.Path(temporary)
 fixture=directory/'fixture.go'
 fixture.write_text(harness+'\n'+body)
 overlay=directory/'overlay.json'
 overlay.write_text(json.dumps({'Replace':{str(root/'fresh_stop_status_test.go'):str(fixture)}}))
 raise SystemExit(subprocess.run(['go','test','-overlay',str(overlay),'-run','^TestFreshRegisteredStopRejectsSFalse','-v','.'],cwd=root).returncode)
