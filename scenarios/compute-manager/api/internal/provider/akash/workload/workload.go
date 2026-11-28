// Package workload renders a server-owned Akash deployment profile into the
// documented SDL v2 shape and validates the bid identity that may be selected
// for that deployment. It is pure: it has no chain client or signing path.
package workload

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const SDLVersion = "2.0"
const NetworkSandbox2 = "sandbox-2"

var (
	namePattern  = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
	cpuRule      = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)
	quantityRule = regexp.MustCompile(`^([1-9][0-9]*)(Mi|Gi|Ti)$`)
	addressRule  = regexp.MustCompile(`^akash1[a-z0-9]{20,}$`)
)

// Profile contains only fields owned by the deployment caller. HealthProbe is
// retained as metadata for the later deployment verifier; Akash SDL does not
// define a health endpoint field, so it is not rendered as an invented key.
type Profile struct {
	Network     string      `json:"network"`
	ChainID     string      `json:"chain_id"`
	Service     string      `json:"service"`
	Image       string      `json:"image"`
	CPUUnits    string      `json:"cpu_units"`
	Memory      string      `json:"memory"`
	Storage     []Volume    `json:"storage"`
	Ports       []Port      `json:"ports"`
	Placement   Placement   `json:"placement"`
	MaxPrice    Rate        `json:"max_price"`
	HealthProbe HealthProbe `json:"health_probe"`
}

type Volume struct {
	Name       string `json:"name,omitempty"`
	Size       string `json:"size"`
	Persistent bool   `json:"persistent"`
	Class      string `json:"class,omitempty"`
}

type Port struct {
	Container uint16 `json:"container"`
	External  uint16 `json:"external"`
	Protocol  string `json:"protocol"`
}

type Placement struct {
	Name   string `json:"name"`
	Region string `json:"region,omitempty"`
}

// Rate is a chain-native maximum rate. It is not a fiat quote or a monthly
// admission amount; downstream admission must use a separately sourced price.
type Rate struct {
	Denom  string `json:"denom"`
	Amount uint64 `json:"amount"`
}

type HealthProbe struct {
	Port uint16 `json:"port"`
	Path string `json:"path"`
}

type Rendered struct {
	SDL         string
	Digest      string
	HealthProbe HealthProbe
	Network     string
	ChainID     string
}

// Render validates and deterministically serializes a single-service Akash
// SDL v2 deployment. Input slices are copied and sorted; the caller's profile
// is not mutated. The digest includes SDL, health metadata and network identity.
func Render(profile Profile) (Rendered, error) {
	normalized, err := normalize(profile)
	if err != nil {
		return Rendered{}, err
	}
	sdl := renderSDL(normalized)
	canonical, err := json.Marshal(struct {
		SDL     string      `json:"sdl"`
		Network string      `json:"network"`
		ChainID string      `json:"chain_id"`
		Health  HealthProbe `json:"health_probe"`
	}{sdl, normalized.Network, normalized.ChainID, normalized.HealthProbe})
	if err != nil {
		return Rendered{}, err
	}
	digest := sha256.Sum256(canonical)
	return Rendered{SDL: sdl, Digest: hex.EncodeToString(digest[:]), HealthProbe: normalized.HealthProbe, Network: normalized.Network, ChainID: normalized.ChainID}, nil
}

func normalize(p Profile) (Profile, error) {
	p.Storage = append([]Volume(nil), p.Storage...)
	p.Ports = append([]Port(nil), p.Ports...)
	p.Network = strings.TrimSpace(p.Network)
	p.ChainID = strings.TrimSpace(p.ChainID)
	p.Service = strings.TrimSpace(p.Service)
	p.Image = strings.TrimSpace(p.Image)
	p.CPUUnits = strings.TrimSpace(p.CPUUnits)
	p.Memory = strings.TrimSpace(p.Memory)
	p.Placement.Name = strings.TrimSpace(p.Placement.Name)
	p.Placement.Region = strings.TrimSpace(p.Placement.Region)
	p.MaxPrice.Denom = strings.TrimSpace(p.MaxPrice.Denom)
	p.HealthProbe.Path = strings.TrimSpace(p.HealthProbe.Path)
	if p.Network != NetworkSandbox2 || p.ChainID == "" {
		return Profile{}, errors.New("Akash workload requires sandbox-2 and an explicit chain ID")
	}
	if !namePattern.MatchString(p.Service) || !namePattern.MatchString(p.Placement.Name) {
		return Profile{}, errors.New("Akash service and placement names must be lowercase DNS labels")
	}
	if !pinnedImage(p.Image) {
		return Profile{}, errors.New("Akash image must use an explicit tag or sha256 digest and cannot use latest")
	}
	cpu, ok := new(big.Rat).SetString(p.CPUUnits)
	if !cpuRule.MatchString(p.CPUUnits) || !ok || cpu.Cmp(big.NewRat(1, 100)) < 0 || cpu.Cmp(big.NewRat(384, 1)) > 0 {
		return Profile{}, errors.New("Akash CPU units must be an exact decimal from 0.01 through 384")
	}
	if !validQuantity(p.Memory, 1<<20, 2<<40) {
		return Profile{}, errors.New("Akash memory must be between 1Mi and 2Ti")
	}
	if len(p.Storage) == 0 {
		return Profile{}, errors.New("Akash workload requires an ephemeral storage volume")
	}
	seenVolumes := map[string]bool{}
	ephemeral := 0
	for i := range p.Storage {
		v := &p.Storage[i]
		v.Name = strings.TrimSpace(v.Name)
		v.Size = strings.TrimSpace(v.Size)
		v.Class = strings.TrimSpace(v.Class)
		if !validQuantity(v.Size, 5<<20, 32<<40) {
			return Profile{}, fmt.Errorf("Akash storage volume %q must be between 5Mi and 32Ti", v.Name)
		}
		if v.Persistent {
			if !namePattern.MatchString(v.Name) || seenVolumes[v.Name] {
				return Profile{}, errors.New("Akash persistent volumes need unique lowercase DNS-label names")
			}
			if !oneOf(v.Class, "default", "beta1", "beta2", "beta3", "ram") {
				return Profile{}, fmt.Errorf("Akash persistent volume %q requires a documented storage class", v.Name)
			}
			seenVolumes[v.Name] = true
		} else {
			ephemeral++
			if v.Name != "" || v.Class != "" {
				return Profile{}, errors.New("Akash ephemeral volume must not have a name or persistent class")
			}
		}
	}
	if ephemeral != 1 {
		return Profile{}, errors.New("Akash workload requires exactly one ephemeral root volume")
	}
	if len(p.Ports) == 0 {
		return Profile{}, errors.New("Akash workload requires at least one globally exposed port")
	}
	ports := map[uint16]bool{}
	externalPorts := map[uint16]bool{}
	for _, port := range p.Ports {
		if port.Container == 0 || port.External == 0 || !oneOf(strings.ToLower(port.Protocol), "tcp", "udp") || ports[port.Container] || externalPorts[port.External] {
			return Profile{}, errors.New("Akash ports require unique nonzero container and external ports and tcp/udp protocol")
		}
		ports[port.Container] = true
		externalPorts[port.External] = true
	}
	if !oneOf(strings.ToLower(p.MaxPrice.Denom), "uact", "uakt") || p.MaxPrice.Amount == 0 {
		return Profile{}, errors.New("Akash SDL maximum price requires a positive uact or uakt amount")
	}
	if p.HealthProbe.Port == 0 || !ports[p.HealthProbe.Port] || !strings.HasPrefix(p.HealthProbe.Path, "/") || strings.ContainsAny(p.HealthProbe.Path, "\r\n\x00") || strings.Contains(p.HealthProbe.Path, "?") || strings.Contains(p.HealthProbe.Path, "#") {
		return Profile{}, errors.New("health probe requires an exposed port and an absolute path without query or fragment")
	}
	if p.Placement.Region != "" && !namePattern.MatchString(p.Placement.Region) {
		return Profile{}, errors.New("Akash placement region must be a lowercase DNS label")
	}
	p.MaxPrice.Denom = strings.ToLower(p.MaxPrice.Denom)
	for i := range p.Ports {
		p.Ports[i].Protocol = strings.ToLower(p.Ports[i].Protocol)
	}
	sort.Slice(p.Ports, func(i, j int) bool {
		if p.Ports[i].Container == p.Ports[j].Container {
			return p.Ports[i].External < p.Ports[j].External
		}
		return p.Ports[i].Container < p.Ports[j].Container
	})
	sort.Slice(p.Storage, func(i, j int) bool {
		if p.Storage[i].Persistent != p.Storage[j].Persistent {
			return !p.Storage[i].Persistent
		}
		return p.Storage[i].Name < p.Storage[j].Name
	})
	return p, nil
}

func pinnedImage(image string) bool {
	if image == "" || strings.ContainsAny(image, " \t\r\n'\"$`)") {
		return false
	}
	if index := strings.LastIndex(image, "@sha256:"); index >= 0 {
		digest := image[index+len("@sha256:"):]
		if index == 0 || len(digest) != 64 {
			return false
		}
		_, err := hex.DecodeString(digest)
		return err == nil
	}
	last := image[strings.LastIndex(image, "/")+1:]
	colon := strings.LastIndex(last, ":")
	return colon > 0 && !strings.EqualFold(last[colon+1:], "latest") && last[colon+1:] != ""
}

func validQuantity(value string, min, max uint64) bool {
	matches := quantityRule.FindStringSubmatch(value)
	if len(matches) != 3 {
		return false
	}
	n, err := strconv.ParseUint(matches[1], 10, 64)
	if err != nil {
		return false
	}
	unit := uint64(1 << 20)
	switch matches[2] {
	case "Gi":
		unit = 1 << 30
	case "Ti":
		unit = 1 << 40
	}
	if n > ^uint64(0)/unit {
		return false
	}
	bytes := n * unit
	return bytes >= min && bytes <= max
}

func renderSDL(p Profile) string {
	var b strings.Builder
	fmt.Fprintf(&b, "version: %s\nservices:\n  %s:\n    image: %s\n    expose:\n", quote(SDLVersion), p.Service, quote(p.Image))
	for _, port := range p.Ports {
		fmt.Fprintf(&b, "      - port: %d\n        as: %d\n        proto: %s\n        to:\n          - global: true\n", port.Container, port.External, port.Protocol)
	}
	fmt.Fprintf(&b, "profiles:\n  compute:\n    %s:\n      resources:\n        cpu:\n          units: %s\n        memory:\n          size: %s\n        storage:\n", p.Service, p.CPUUnits, p.Memory)
	for _, volume := range p.Storage {
		if !volume.Persistent {
			fmt.Fprintf(&b, "          - size: %s\n", volume.Size)
			continue
		}
		fmt.Fprintf(&b, "          - name: %s\n            size: %s\n            attributes:\n              persistent: true\n              class: %s\n", volume.Name, volume.Size, volume.Class)
	}
	fmt.Fprintf(&b, "  placement:\n    %s:\n", p.Placement.Name)
	if p.Placement.Region != "" {
		fmt.Fprintf(&b, "      attributes:\n        region: %s\n", quote(p.Placement.Region))
	}
	fmt.Fprintf(&b, "      pricing:\n        %s:\n          denom: %s\n          amount: %d\ndeployment:\n  %s:\n    %s:\n      profile: %s\n      count: 1\n", p.Service, p.MaxPrice.Denom, p.MaxPrice.Amount, p.Service, p.Placement.Name, p.Service)
	return b.String()
}

func quote(value string) string { return strconv.Quote(value) }

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

// BidTuple is the network identity from which one order bid is selected. DSEQ
// identifies its deployment; GSEQ/OSEQ scope the order; provider completes bid
// identity. The network and chain prevent cross-testnet replay.
type BidTuple struct {
	Network  string
	ChainID  string
	Owner    string
	DSEQ     uint64
	GSEQ     uint32
	OSEQ     uint32
	Provider string
}

func (t BidTuple) Validate() error {
	if t.Network != NetworkSandbox2 || strings.TrimSpace(t.ChainID) == "" || !addressRule.MatchString(t.Owner) || t.DSEQ == 0 || t.GSEQ == 0 || t.OSEQ == 0 || !addressRule.MatchString(t.Provider) {
		return errors.New("Akash bid tuple requires sandbox chain, owner, nonzero DSEQ/GSEQ/OSEQ and provider address")
	}
	return nil
}

func (t BidTuple) Key() string {
	return strings.Join([]string{t.Network, t.ChainID, t.Owner, strconv.FormatUint(t.DSEQ, 10), strconv.FormatUint(uint64(t.GSEQ), 10), strconv.FormatUint(uint64(t.OSEQ), 10), t.Provider}, ":")
}

// SelectedBid is the exact operator/application selected bid and its source
// validity facts. The helper validates references; it does not select a bid.
type SelectedBid struct {
	Tuple            BidTuple
	IntentID         string
	WorkloadDigest   string
	Price            Rate
	ObservedAt       time.Time
	ValidUntilHeight uint64
}

type BidScope struct {
	Network        string
	ChainID        string
	Owner          string
	IntentID       string
	WorkloadDigest string
	MaxPrice       Rate
	CurrentHeight  uint64
	Now            time.Time
	MaxAge         time.Duration
}

func (b SelectedBid) ValidateFor(scope BidScope) error {
	if err := b.Tuple.Validate(); err != nil {
		return err
	}
	if scope.Network != NetworkSandbox2 || scope.ChainID == "" || scope.Owner == "" || scope.IntentID == "" || !validDigest(scope.WorkloadDigest) || scope.CurrentHeight == 0 || scope.Now.IsZero() || scope.MaxAge <= 0 {
		return errors.New("Akash bid validation requires complete expected chain, intent, digest and freshness scope")
	}
	if b.Tuple.Network != scope.Network || b.Tuple.ChainID != scope.ChainID || b.Tuple.Owner != scope.Owner {
		return errors.New("Akash selected bid belongs to a different network, chain or owner")
	}
	if strings.TrimSpace(b.IntentID) != scope.IntentID || b.WorkloadDigest != scope.WorkloadDigest {
		return errors.New("Akash selected bid is not bound to the expected intent and workload digest")
	}
	if b.Price.Amount == 0 || !strings.EqualFold(strings.TrimSpace(b.Price.Denom), strings.TrimSpace(scope.MaxPrice.Denom)) || scope.MaxPrice.Amount == 0 || b.Price.Amount > scope.MaxPrice.Amount {
		return errors.New("Akash selected bid price is missing or exceeds the workload maximum")
	}
	if b.ObservedAt.IsZero() || b.ObservedAt.After(scope.Now) || scope.Now.Sub(b.ObservedAt) > scope.MaxAge {
		return errors.New("Akash selected bid observation is stale or invalid")
	}
	if b.ValidUntilHeight == 0 || b.ValidUntilHeight <= scope.CurrentHeight {
		return errors.New("Akash selected bid is expired at the current chain height")
	}
	return nil
}

func validDigest(digest string) bool {
	if len(digest) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(digest)
	return err == nil
}

// ValidateUniqueBidTuples refuses duplicate selection/replay entries in a
// caller-supplied batch instead of silently accepting first/last-wins behavior.
func ValidateUniqueBidTuples(bids []SelectedBid) error {
	seen := make(map[string]struct{}, len(bids))
	for _, bid := range bids {
		if err := bid.Tuple.Validate(); err != nil {
			return err
		}
		key := bid.Tuple.Key()
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate Akash bid tuple %s", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}
