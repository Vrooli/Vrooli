package dns

import (
	"context"
	"fmt"

	maildns "github.com/vrooli/vrooli/packages/maildns-go"
)

// VerifyMailDNS uses the scenario-to-cloud DNS reader as the resolver seam so
// deployment preflight and the running landing-page API share one verifier.
func VerifyMailDNS(ctx context.Context, domainName string, requirements []maildns.ProviderRequirement) maildns.Report {
	return maildns.New(recordSetMailResolver{}).Verify(ctx, domainName, requirements)
}

type recordSetMailResolver struct{}

func (recordSetMailResolver) LookupTXT(ctx context.Context, host string) ([]string, error) {
	set, err := LookupRecordSet(ctx, host)
	if err != nil {
		return nil, err
	}
	values := make([]string, 0, len(set.TXT))
	for _, value := range set.TXT {
		values = append(values, value.Value)
	}
	return values, nil
}

func (recordSetMailResolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	set, err := LookupRecordSet(ctx, host)
	if err != nil {
		return "", err
	}
	if len(set.CNAME) == 0 {
		return "", fmt.Errorf("CNAME %s not found", host)
	}
	return set.CNAME[0].Value, nil
}

var _ maildns.Resolver = recordSetMailResolver{}
