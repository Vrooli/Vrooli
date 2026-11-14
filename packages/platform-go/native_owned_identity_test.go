package platform

import (
	"encoding/json"
	"strings"
	"testing"
)

func nativeFixtureRef(t *testing.T, target string) OperationRef {
	t.Helper()
	i := OperationIntent{Owner: "fixture", AttemptID: "native", InvocationID: strings.Repeat("a", 32), SpecDigest: strings.Repeat("b", 64)}
	i.NativeAccount = "sid:S-1-5-21-1-2-3-1001"
	unit := operationUnitName(i)
	n := &NativeWorkloadIdentity{Version: 2, Platform: target, Name: windowsOwnedServiceName(i), InvocationID: i.InvocationID, Account: i.NativeAccount, ContainmentID: "bounded", BootID: "12345678-1234-1234-1234-123456789abc"}
	n.Manager, n.Domain = "operator-job", "operator"
	n.RootIncarnation = "windows:123:45600"
	n.Account = windowsOwnedAccount(i)
	n.ContainmentID = windowsOwnedJob(i)
	if target == "darwin" {
		i.NativeAccount = "uid:501"
		n.Account = i.NativeAccount
		n.ProcessOnly, n.RootUniqueID = true, "987654"
		unit = operationUnitName(i)
		n.Manager, n.Domain, n.Name = "launchd", "user/501", darwinOwnedLabel(i)
		n.ContainmentID = n.Domain + "/" + n.Name
		n.RootIncarnation = "darwin:123:456:7"
	}
	start := "116444736000000456"
	if target == "darwin" {
		start = "456000007"
	}
	return OperationRef{Intent: i, Unit: unit, Scope: ScopeRef{Kind: ScopeKindNative, Path: n.Domain + "/" + n.Name}, RootPID: 123, RootStartTime: start, NativeInvocationID: i.InvocationID, Native: n}
}
func TestNativeOwnedReceiptRoundTripEquality(t *testing.T) {
	for _, target := range []string{"darwin", "windows"} {
		t.Run(target, func(t *testing.T) {
			r := nativeFixtureRef(t, target)
			if err := r.ValidateIdentity(); err != nil {
				t.Fatal(err)
			}
			b, err := json.Marshal(r)
			if err != nil {
				t.Fatal(err)
			}
			var decoded OperationRef
			if err = json.Unmarshal(b, &decoded); err != nil {
				t.Fatal(err)
			}
			if !r.Equal(decoded) || !ServiceRef(r).Equal(ServiceRef(decoded)) {
				t.Fatal("decoded receipt lost value identity")
			}
			if !r.MatchesIncarnation(r.Native.RootIncarnation) || r.MatchesIncarnation(r.Native.RootIncarnation+"0") {
				t.Fatal("native incarnation was not exact")
			}
			decoded.Native.ContainmentID = "other"
			if r.Equal(decoded) {
				t.Fatal("accepted another containment identity")
			}
		})
	}
}
func TestNativeOwnedIdentityRejectsMixedAndMalformedEvidence(t *testing.T) {
	mutations := []func(*OperationRef){
		func(r *OperationRef) { r.Native.Version = 1 },
		func(r *OperationRef) { r.RootStartTime = "116444736000000457" }, func(r *OperationRef) { r.Native.Platform = "linux" },
		func(r *OperationRef) { r.Native.Name = "foreign" }, func(r *OperationRef) { r.Native.Account = "" },
		func(r *OperationRef) { r.Native.RootIncarnation = "windows:124:456" }, func(r *OperationRef) { r.Native.RootIncarnation = "windows:123:0" },
		func(r *OperationRef) { r.ScopeInode = 1 }, func(r *OperationRef) { r.Scope.Kind = ScopeKindCgroup },
		func(r *OperationRef) { r.Native.BootID = "unknown" }, func(r *OperationRef) { r.Native.InvocationID = strings.Repeat("0", 32) },
	}
	for idx, mutate := range mutations {
		r := nativeFixtureRef(t, "windows")
		mutate(&r)
		if r.ValidateIdentity() == nil {
			t.Errorf("mutation %d accepted", idx)
		}
	}
}

func TestLegacyOwnedReceiptOmitsNativeExtension(t *testing.T) {
	i := OperationIntent{Owner: "fixture", AttemptID: "legacy", InvocationID: strings.Repeat("a", 32), SpecDigest: strings.Repeat("b", 64)}
	b, err := json.Marshal(i)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"owner":"fixture","attempt_id":"legacy","invocation_id":"` + strings.Repeat("a", 32) + `","spec_digest":"` + strings.Repeat("b", 64) + `"}`
	if string(b) != want {
		t.Fatalf("legacy intent JSON changed: %s", b)
	}
	r := OperationRef{Intent: i}
	b, err = json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"native"`) {
		t.Fatalf("legacy receipt gained native field: %s", b)
	}
}
