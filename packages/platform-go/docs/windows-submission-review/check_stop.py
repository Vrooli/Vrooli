"""Execute the actual Windows stop control flow with hermetic native snapshots.

Only platform calls are renamed to fakes. The function body comes from the
current Windows source; no Windows operation runs on this Linux reviewer host.
"""
import json
import pathlib
import subprocess
import tempfile

root = pathlib.Path(__file__).resolve().parents[2]
source = (root / "windows_owned_task_windows.go").read_text()
start = source.index("func stopWindowsOwnedTask(")
end = source.index("\nfunc deleteWindowsOwnedTask(", start)
body = source[start:end]
for old, new in {
    "stopWindowsOwnedTask": "independentStopWindowsOwnedTask",
    "windowsOwnedTaskSnapshot": "independentTaskSnapshot",
    "taskNativeErrorMissing": "independentTaskMissing",
    "windowsSetTaskEnabled": "independentSetTaskEnabled",
    "WindowsStopTaskInstance": "independentStopTaskInstance",
    "stopWindowsOwnedRegisteredTask": "independentStopRegisteredTask",
}.items():
    body = body.replace(old, new)

harness = r'''
package platform
import ("context"; "errors"; "testing"; "time")
var independentSnapshot WindowsTaskSnapshot
var independentStopError error
var independentKeepQueued bool
var independentStops int
func independentTaskSnapshot(windowsOwnedPayload) (WindowsTaskSnapshot,error) {return independentSnapshot,nil}
func independentTaskMissing(error) bool {return false}
func independentSetTaskEnabled(string,bool) error {independentSnapshot.Enabled=false; independentSnapshot.State=1; return nil}
func independentStopTaskInstance(string,string) error {return nil}
func independentStopRegisteredTask(name string) error {
 if name!=independentSnapshot.Name || independentSnapshot.Enabled {panic("wrong task or Stop before disable")}
 independentStops++
 if independentKeepQueued {independentSnapshot.State=2}
 return independentStopError
}
func TestIndependentQueuedSubmissionCannotBecomeAbsentByDisable(t *testing.T) {
 p:=windowsOwnedFixture(t)
 independentSnapshot=WindowsTaskSnapshot{Name:windowsOwnedTaskName(p.Intent),State:2,Enabled:true}
 independentStopError=errors.New("native cancellation unproven")
 for n:=0;n<2;n++ {
  if err:=independentStopWindowsOwnedTask(context.Background(),p,windowsOwnedRootRecord{}); !errors.Is(err,independentStopError) {t.Fatalf("attempt %d erased unresolved queue: %v",n,err)}
  if independentStops!=n+1 {t.Fatal("retry omitted fresh exact task cancellation")}
 }
 independentStopError=nil
 independentKeepQueued=true
 if err:=independentStopWindowsOwnedTask(context.Background(),p,windowsOwnedRootRecord{}); !errors.Is(err,ErrOperationNotQuiescent) {t.Fatalf("post-Stop queued state falsely quiescent: %v",err)}
 independentKeepQueued=false
 if err:=independentStopWindowsOwnedTask(context.Background(),p,windowsOwnedRootRecord{}); err!=nil {t.Fatalf("successful exact cancellation plus disabled empty observation refused: %v",err)}
 if independentStops!=4 {t.Fatal("not every attempt canceled the exact registered task")}
}
'''

with tempfile.TemporaryDirectory(prefix="vrooli-submission-review-") as temporary:
    directory = pathlib.Path(temporary)
    fixture = directory / "fixture.go"
    fixture.write_text(harness + "\n" + body + "\n")
    overlay = directory / "overlay.json"
    overlay.write_text(json.dumps({"Replace": {str(root / "independent_native_stop_test.go"): str(fixture)}}))
    result = subprocess.run(["go", "test", "-overlay", str(overlay), "-run", "^TestIndependentQueuedSubmission", "-v", "."], cwd=root)
    raise SystemExit(result.returncode)
