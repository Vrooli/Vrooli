package trials

import (
	internaltrials "meta-optimization-manager/internal/trials"

	trialsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/trials"
)

// This file is the only translation point between the trials proto wire enums
// and the domain vocabulary (internal/trials). The domain layer never imports
// proto (api-steer §7).

func verdictToProto(v internaltrials.Verdict) trialsv1.TrialVerdict {
	switch v {
	case internaltrials.VerdictPass:
		return trialsv1.TrialVerdict_TRIAL_VERDICT_PASS
	case internaltrials.VerdictFail:
		return trialsv1.TrialVerdict_TRIAL_VERDICT_FAIL
	case internaltrials.VerdictError:
		return trialsv1.TrialVerdict_TRIAL_VERDICT_ERROR
	default:
		return trialsv1.TrialVerdict_TRIAL_VERDICT_UNSPECIFIED
	}
}
