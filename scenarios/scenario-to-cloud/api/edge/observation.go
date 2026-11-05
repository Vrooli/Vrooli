package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"scenario-to-cloud/domain"

	edgev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/edge"
	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/errors"
)

// ProducerRef identifies this producer on the wire.
const ProducerRef = "scenario-to-cloud:edge:v1"

// SchemaVersion is the observation schema version.
const SchemaVersion = "1"

// HostLookup resolves a host to its addresses.
type HostLookup func(ctx context.Context, host string) ([]string, error)

// CertificateProbe reads the live certificate of a host.
type CertificateProbe func(ctx context.Context, host string) (CertificateFacts, error)

// ReadinessProbe answers whether the workload is reachable; local runs on
// the target against the loopback listener, external from the producer
// against the public host.
type ReadinessProbe func(ctx context.Context, host string, port int) error

// ObserveInputs is everything Observe reads. Probes are optional: an absent
// probe yields not_checked rather than a verdict.
type ObserveInputs struct {
	Spec        domain.EdgeSpec
	Lookup      HostLookup
	Certificate CertificateProbe
	Journal     RenewalEvidence
	Local       ReadinessProbe
	External    ReadinessProbe
	Now         time.Time
}

// Observe evaluates every route of the spec. It never records a secret: the
// DNS provider is referenced through the spec's descriptor only, and the
// journal evidence carries messages, not tokens.
func Observe(ctx context.Context, in ObserveInputs) domain.EdgeObservation {
	now := in.Now
	if now.IsZero() {
		now = time.Now()
	}
	now = now.UTC()
	obs := domain.EdgeObservation{
		SchemaVersion:    SchemaVersion,
		DeploymentID:     in.Spec.DeploymentID,
		Domain:           in.Spec.Domain,
		SpecDigest:       in.Spec.Digest,
		Routes:           append([]domain.EdgeRoute{}, in.Spec.Routes...),
		PrivateListeners: append([]domain.EdgePrivateListener{}, in.Spec.PrivateListeners...),
		DNS:              []domain.EdgeDNSBinding{},
		TLS:              []domain.EdgeTLSState{},
		ACMEEnvironment:  in.Spec.ACMEEnvironment,
		ObservedAt:       now,
		ProducerRef:      ProducerRef,
	}
	allBound := len(in.Spec.Routes) > 0
	var actions []domain.NextActionHint
	for _, route := range in.Spec.Routes {
		var observed []string
		var lookupErr error
		if in.Lookup != nil {
			observed, lookupErr = in.Lookup(ctx, route.Host)
		} else {
			lookupErr = fmt.Errorf("dns lookup not available")
		}
		binding := BindDNS(route.Host, observed, lookupErr, in.Spec.IPPolicy)
		obs.DNS = append(obs.DNS, binding)
		if !binding.Match {
			allBound = false
			actions = append(actions, NextActionDNS(in.Spec.DeploymentID))
		}
		facts := CertificateFacts{}
		if in.Certificate != nil {
			if got, err := in.Certificate(ctx, route.Host); err == nil {
				facts = got
			} else {
				facts = CertificateFacts{Present: false, ValidationError: err.Error()}
			}
		}
		tlsState := ObserveTLS(route.Host, facts, in.Journal, now)
		obs.TLS = append(obs.TLS, tlsState)
		if warn, fail := TLSNeedsAttention(tlsState); warn || fail {
			actions = append(actions, NextActionRenew(in.Spec.DeploymentID))
		}
	}
	var localErr, externalErr error
	localChecked, externalChecked := in.Local != nil, in.External != nil
	if localChecked {
		for _, route := range in.Spec.Routes {
			if err := in.Local(ctx, route.Host, route.UpstreamPort); err != nil {
				localErr = err
				break
			}
		}
	}
	if externalChecked && allBound {
		for _, route := range in.Spec.Routes {
			if err := in.External(ctx, route.Host, 443); err != nil {
				externalErr = err
				break
			}
		}
	}
	obs.Readiness = Readiness(localErr, externalErr, allBound)
	if !localChecked {
		obs.Readiness.Local = domain.EdgeReadinessCheck{Status: "not_checked"}
	}
	if !externalChecked && allBound {
		obs.Readiness.External = domain.EdgeReadinessCheck{Status: "not_checked"}
	}
	obs.NextActions = dedupeActions(actions)
	return obs
}

func dedupeActions(in []domain.NextActionHint) []domain.NextActionHint {
	seen := map[string]bool{}
	out := make([]domain.NextActionHint, 0, len(in))
	for _, a := range in {
		if seen[a.Reference] {
			continue
		}
		seen[a.Reference] = true
		out = append(out, a)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Reference < out[j].Reference })
	return out
}

// ToProto converts the observation to its wire message.
func ToProto(obs domain.EdgeObservation) *edgev1.EdgeObservation {
	out := &edgev1.EdgeObservation{
		SchemaVersion:   obs.SchemaVersion,
		DeploymentId:    obs.DeploymentID,
		Domain:          obs.Domain,
		SpecDigest:      obs.SpecDigest,
		AcmeEnvironment: obs.ACMEEnvironment,
		ObservedAt:      timestamppb.New(obs.ObservedAt),
		ProducerRef:     obs.ProducerRef,
		Readiness: &edgev1.Readiness{
			Local:    &edgev1.ReadinessCheck{Status: obs.Readiness.Local.Status, ReasonCode: obs.Readiness.Local.ReasonCode, Detail: obs.Readiness.Local.Detail},
			External: &edgev1.ReadinessCheck{Status: obs.Readiness.External.Status, ReasonCode: obs.Readiness.External.ReasonCode, Detail: obs.Readiness.External.Detail},
		},
	}
	for _, r := range obs.Routes {
		out.Routes = append(out.Routes, &edgev1.EdgeRoute{Host: r.Host, UpstreamPort: int32(r.UpstreamPort), ListenerId: r.ListenerID}) //nolint:gosec // ports are bounded
	}
	for _, l := range obs.PrivateListeners {
		out.PrivateListeners = append(out.PrivateListeners, &edgev1.PrivateListener{Id: l.ID, Owner: l.Owner, PortName: l.PortName, Port: int32(l.Port), Reason: l.Reason}) //nolint:gosec // ports are bounded
	}
	for _, d := range obs.DNS {
		out.Dns = append(out.Dns, &edgev1.DNSBinding{
			Host:       d.Host,
			Ipv4:       &edgev1.AddressBinding{Expected: d.IPv4.Expected, Observed: d.IPv4.Observed, State: d.IPv4.State},
			Ipv6:       &edgev1.AddressBinding{Expected: d.IPv6.Expected, Observed: d.IPv6.Observed, State: d.IPv6.State},
			Match:      d.Match,
			ReasonCode: d.ReasonCode,
		})
	}
	for _, t := range obs.TLS {
		out.Tls = append(out.Tls, &edgev1.TLSState{Host: t.Host, Issuer: t.Issuer, NotAfter: t.NotAfter, DaysLeft: int32(t.DaysLeft), RenewalState: t.RenewalState, ReasonCode: t.ReasonCode, Detail: t.Detail, AcmeEnvironment: t.ACMEEnvironment}) //nolint:gosec // day counts are small
	}
	for _, a := range obs.NextActions {
		out.NextActions = append(out.NextActions, &errorsv1.NextAction{Owner: a.Owner, Kind: a.Kind, Reference: a.Reference, Label: a.Label})
	}
	return out
}

var jsonOptions = protojson.MarshalOptions{UseProtoNames: true, EmitDefaultValues: true}

// Response wraps the observation in its versioned envelope.
func Response(obs *edgev1.EdgeObservation) *edgev1.GetEdgeObservationResponse {
	return &edgev1.GetEdgeObservationResponse{SchemaVersion: SchemaVersion, Observation: obs}
}

// MarshalResponseJSON encodes the envelope with proto field names.
func MarshalResponseJSON(obs *edgev1.EdgeObservation) (json.RawMessage, error) {
	if obs == nil {
		return nil, fmt.Errorf("observation is nil")
	}
	raw, err := jsonOptions.Marshal(Response(obs))
	if err != nil {
		return nil, err
	}
	return json.RawMessage(raw), nil
}

// ContainsSecretLike reports whether raw carries a value that looks like a
// credential; producers assert it is false before serving an observation.
func ContainsSecretLike(raw []byte, canaries ...string) bool {
	text := string(raw)
	for _, canary := range canaries {
		if canary != "" && strings.Contains(text, canary) {
			return true
		}
	}
	return false
}
