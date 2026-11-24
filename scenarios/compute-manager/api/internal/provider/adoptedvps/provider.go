// Package adoptedvps represents an existing, operator-owned VPS inside the
// Compute Manager provider interface. Create records the explicit target after
// shared admission; it never creates or mutates a provider resource.
package adoptedvps

import (
	"context"
	"errors"
	"net"
	"regexp"
	"strings"
	"time"

	"compute-manager/internal/provider"
)

const Name = "adopted-vps"

var ErrOperatorOwned = errors.New("adopted VPS is operator-owned; Compute Manager cannot destroy it")

type Adapter struct {
	Now func() time.Time
}

func (*Adapter) Name() string { return Name }
func (*Adapter) Facts() provider.BillingFacts {
	return provider.BillingFacts{RoundingUnit: time.Minute, MinimumBillable: time.Minute, StoppedStillBills: true}
}

func (a *Adapter) Create(_ context.Context, spec provider.Spec) (provider.Instance, error) {
	if spec.Region != "manual" || !strings.HasPrefix(spec.Size, "declared-") {
		return provider.Instance{}, errors.New("adopted-vps requires the exact manual region and a declared offer")
	}
	if spec.AdoptedTarget == nil {
		return provider.Instance{}, errors.New("adopted-vps target metadata is required")
	}
	host := strings.TrimSpace(spec.AdoptedTarget.Host)
	user := strings.TrimSpace(spec.AdoptedTarget.User)
	if !validHost(host) || !validUser(user) {
		return provider.Instance{}, errors.New("adopted-vps host or SSH user is invalid")
	}
	intentID := strings.TrimSpace(spec.Tags["vrooli-intent-id"])
	if intentID == "" {
		return provider.Instance{}, errors.New("adopted-vps requires a persisted Compute Manager intent")
	}
	now := time.Now
	if a != nil && a.Now != nil {
		now = a.Now
	}
	return provider.Instance{
		ID: "adopted-vps:" + intentID, Region: "manual", Size: spec.Size,
		Image: spec.Image, Address: host, SSHUser: user, SSHPort: 22,
		CreatedAt: now().UTC(), Tags: cloneTags(spec.Tags),
	}, nil
}

func (*Adapter) Describe(context.Context, string) (provider.Instance, error) {
	return provider.Instance{}, errors.New("adopted-vps inventory is limited to explicitly persisted Compute Manager records")
}

func (*Adapter) List(context.Context) ([]provider.Instance, error) {
	return []provider.Instance{}, nil
}

func (*Adapter) Destroy(context.Context, string) error { return ErrOperatorOwned }

var hostnameLabel = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)
var unixUser = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.-]{0,63}$`)

func validHost(host string) bool {
	if host == "" || len(host) > 253 || strings.ContainsAny(host, "\r\n\t /\\@") {
		return false
	}
	if net.ParseIP(host) != nil {
		return true
	}
	for _, label := range strings.Split(strings.TrimSuffix(host, "."), ".") {
		if !hostnameLabel.MatchString(label) {
			return false
		}
	}
	return true
}

func validUser(user string) bool {
	return unixUser.MatchString(user)
}

func cloneTags(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
