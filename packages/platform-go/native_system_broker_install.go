package platform

import "errors"

// Native effects stay behind the explicit installer. This engine makes the
// fail-before-registration boundary fixture-testable without running a daemon.
type nativeSystemBrokerInstallOps struct {
	Stage    func(nativeSystemBrokerPlan) error
	Register func(nativeSystemBrokerPlan) (ServiceInstallResult, error)
}

func installNativeSystemBrokerWithOps(target string, o ServiceInstallOptions, roots nativeSystemBrokerRoots, ops nativeSystemBrokerInstallOps) (ServiceInstallResult, error) {
	p, e := buildNativeSystemBrokerPlan(target, o, roots)
	if e != nil {
		return ServiceInstallResult{}, e
	}
	if ops.Stage == nil || ops.Register == nil {
		return ServiceInstallResult{}, errors.New("platform: native broker installer is incomplete")
	}
	if e = ops.Stage(p); e != nil {
		return ServiceInstallResult{}, e
	}
	return ops.Register(p)
}
