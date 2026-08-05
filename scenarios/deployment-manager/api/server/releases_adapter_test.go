package server

import (
	"context"
	"errors"
	"testing"

	"deployment-manager/deployments"
	"deployment-manager/releases"
)

type adapterLPBSClient struct {
	out *deployments.LPBSVerifyResult
	err error
}

func (c adapterLPBSClient) CheckDeployReadiness(context.Context, *deployments.LPBSReadinessRequest) (*deployments.LPBSReadinessResult, error) {
	return nil, nil
}
func (c adapterLPBSClient) Verify(context.Context, *deployments.LPBSVerifyRequest) (*deployments.LPBSVerifyResult, error) {
	return c.out, c.err
}

func TestReleasesVerifierAdapterMapsOutcomeAndNilCases(t *testing.T) {
	req := &releases.VerifyCall{AppKey: "app", Channel: "stable", Platform: "linux", ExpectedVersion: "1", Deep: true}
	adapter := releasesVerifierAdapter{inner: adapterLPBSClient{out: &deployments.LPBSVerifyResult{AppKey: "app", Channel: "stable", Platform: "linux", ExpectedVersion: "1", ObservedVersion: "1", Match: true, SHA512Match: true}}}
	out, err := adapter.Verify(context.Background(), req)
	if err != nil || out == nil || !out.Match || out.ObservedVersion != "1" {
		t.Fatalf("mapped outcome = %#v, %v", out, err)
	}
	for name, adapter := range map[string]releasesVerifierAdapter{
		"nil inner":  {},
		"nil output": {inner: adapterLPBSClient{}},
	} {
		t.Run(name, func(t *testing.T) {
			out, err := adapter.Verify(context.Background(), req)
			if err != nil || out == nil {
				t.Fatalf("outcome = %#v, %v", out, err)
			}
		})
	}
	wantErr := errors.New("downstream")
	_, err = (releasesVerifierAdapter{inner: adapterLPBSClient{err: wantErr}}).Verify(context.Background(), req)
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v", err)
	}
}
