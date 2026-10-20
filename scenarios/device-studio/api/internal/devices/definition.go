// Package devices defines backend-neutral physical device facts and conservative
// compatibility checks. Definitions may be useful while incomplete; physical
// qualification is a separate, stricter operation.
package devices

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
)

const CanonicalUnit = "m"
const DimensionUnit = "mm"

type Group string

const (
	GroupPhone             Group = "phone"
	GroupLaptop            Group = "laptop"
	GroupTablet            Group = "tablet"
	GroupFoldable          Group = "foldable"
	GroupWatch             Group = "watch"
	GroupMonitorTV         Group = "monitor_tv"
	GroupSmartDisplayFrame Group = "smart_display_frame"
	GroupVehicleHost       Group = "vehicle_host"
)

var validGroups = map[Group]bool{
	GroupPhone: true, GroupLaptop: true, GroupTablet: true, GroupFoldable: true,
	GroupWatch: true, GroupMonitorTV: true, GroupSmartDisplayFrame: true, GroupVehicleHost: true,
}

type IdentityStatus string

const (
	IdentityDocumented IdentityStatus = "documented_product"
	IdentityGeneric    IdentityStatus = "generic_design"
	IdentityConcept    IdentityStatus = "concept"
)

type Evidence struct {
	Source      string `json:"source"`
	Kind        string `json:"kind"`
	RetrievedAt string `json:"retrieved_at,omitempty"`
	Qualifier   string `json:"qualifier,omitempty"`
}

type Uncertainty struct {
	Status      string   `json:"status"` // quantified, source_unspecified, unknown
	Lower       *float64 `json:"lower,omitempty"`
	Upper       *float64 `json:"upper,omitempty"`
	Unit        string   `json:"unit,omitempty"`
	Explanation string   `json:"explanation,omitempty"`
}

type Measurement struct {
	Value         *float64    `json:"value"`
	Unit          string      `json:"unit"`
	Meaning       string      `json:"meaning"`
	Configuration string      `json:"configuration"`
	Method        string      `json:"method"` // manufacturer, measured, derived, image_estimated, design_assumption, unknown
	Uncertainty   Uncertainty `json:"uncertainty"`
	Evidence      []Evidence  `json:"evidence"`
}

type Dimensions struct {
	Width  Measurement `json:"width"`
	Height Measurement `json:"height"`
	Depth  Measurement `json:"depth"`
}

type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type Region struct {
	Status        string     `json:"status"`                   // qualified, candidate_only, unknown, not_applicable
	Shape         string     `json:"shape,omitempty"`          // rect, circle, polygon, mask; bounds are a bounding box
	GeometryRef   string     `json:"geometry_ref,omitempty"`   // required for mask-backed regions
	GeometrySpace string     `json:"geometry_space,omitempty"` // physical_mm in the display reference plane
	Bounds        *Rect      `json:"bounds,omitempty"`         // normalized top-left origin, +u right, +v down
	Evidence      []Evidence `json:"evidence,omitempty"`
}

// DisplayMetric defines the physical span represented by normalized UV extent
// [0,1]×[0,1]. UV distance is not physical Euclidean distance unless both
// spans have the same scale; non-rectangular geometry therefore uses a
// physical-mm artifact rather than inferring shape from its UV bounding box.
type DisplayMetric struct {
	WidthMM  float64    `json:"width_mm"`
	HeightMM float64    `json:"height_mm"`
	Evidence []Evidence `json:"evidence"`
}

type GeometryRelation struct {
	Subject  string     `json:"subject"`  // app_safe_area, active_area, or occlusion:<id>
	Relation string     `json:"relation"` // contained_by or disjoint
	Object   string     `json:"object"`
	Status   string     `json:"status"` // qualified, candidate_only, unknown
	Evidence []Evidence `json:"evidence,omitempty"`
}

type HardwareOcclusion struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Region Region `json:"region"`
}

type DisplaySurface struct {
	ID                 string              `json:"id"`
	Topology           string              `json:"topology"` // planar, continuous_flexible, curved, dual_rigid, unknown
	CoordinateMetric   *DisplayMetric      `json:"coordinate_metric,omitempty"`
	GeometryRelations  []GeometryRelation  `json:"geometry_relations,omitempty"`
	PhysicalBoundary   Region              `json:"physical_boundary"`
	ActiveArea         Region              `json:"active_area"`
	AppSafeArea        Region              `json:"app_safe_area"`
	HardwareOcclusions []HardwareOcclusion `json:"hardware_occlusions"`
	Evidence           []Evidence          `json:"evidence,omitempty"`
}

type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Joint struct {
	ID           string     `json:"id"`
	ParentPart   string     `json:"parent_part"`
	ChildPart    string     `json:"child_part"`
	Axis         *Vector3   `json:"axis,omitempty"`
	PivotMeters  *Vector3   `json:"pivot_m,omitempty"`
	LowerRadians *float64   `json:"lower_rad,omitempty"`
	UpperRadians *float64   `json:"upper_rad,omitempty"`
	Status       string     `json:"status"` // qualified, candidate_only, unknown
	Evidence     []Evidence `json:"evidence,omitempty"`
}

type Capability struct {
	Name       string     `json:"name"`
	Status     string     `json:"status"`                // qualified, candidate_only, unsupported, unknown
	MotionKind string     `json:"motion_kind,omitempty"` // rigid_body, articulated, camera
	JointIDs   []string   `json:"joint_ids,omitempty"`   // explicit joints required for articulated motion
	Evidence   []Evidence `json:"evidence,omitempty"`
}

// SupportedProfiles names software or host integrations carried by a device
// family. A profile is not evidence of physical dimensions or host capability.
type SupportedProfile struct {
	Name     string     `json:"name"`
	Status   string     `json:"status"` // qualified, candidate_only, unsupported, unknown
	Evidence []Evidence `json:"evidence,omitempty"`
}

type Pose struct {
	Name       string             `json:"name"`
	Status     string             `json:"status"` // qualified, candidate_only, unknown
	JointState map[string]float64 `json:"joint_state_rad,omitempty"`
	Evidence   []Evidence         `json:"evidence,omitempty"`
}

type SupportCapability struct {
	Kind        string     `json:"kind"` // tabletop, floor_stand, wall_mount, wrist, vehicle_integrated
	InterfaceID string     `json:"interface_id,omitempty"`
	Status      string     `json:"status"` // qualified, candidate_only, unknown, unsupported
	Evidence    []Evidence `json:"evidence,omitempty"`
}

type QualificationReview struct {
	Scope    string     `json:"scope"`
	Result   string     `json:"result"` // passed, rejected
	Reviewer string     `json:"reviewer"`
	Evidence []Evidence `json:"evidence"`
}

type ModelCandidate struct {
	ID          string `json:"id"`
	Status      string `json:"status"` // candidate_only, not_acquired, not_inspected
	SourcePath  string `json:"source_path,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	ListingURL  string `json:"listing_url,omitempty"`
	Author      string `json:"author,omitempty"`
	License     string `json:"license,omitempty"`
	Attribution string `json:"attribution,omitempty"`
	RightsNotes string `json:"rights_notes,omitempty"`
}

type Definition struct {
	SchemaVersion        string                `json:"schema_version"`
	ID                   string                `json:"id"`
	Group                Group                 `json:"group"`
	Kind                 string                `json:"kind"`
	IdentityStatus       IdentityStatus        `json:"identity_status"`
	Manufacturer         string                `json:"manufacturer,omitempty"`
	Model                string                `json:"model"`
	Variant              string                `json:"variant,omitempty"`
	Description          string                `json:"description,omitempty"`
	Dimensions           map[string]Dimensions `json:"dimensions"`
	Displays             []DisplaySurface      `json:"displays"`
	Joints               []Joint               `json:"joints"`
	SupportedPoses       []Pose                `json:"supported_poses"`
	SupportedMotion      []Capability          `json:"supported_motion"`
	SupportedProfiles    []SupportedProfile    `json:"supported_profiles,omitempty"`
	Supports             []SupportCapability   `json:"supports"`
	ModelCandidates      []ModelCandidate      `json:"model_candidates"`
	Provenance           []Evidence            `json:"provenance"`
	Qualification        string                `json:"qualification"` // unqualified or qualified; a pass needs a separate review record
	QualificationReviews []QualificationReview `json:"qualification_reviews,omitempty"`
}

type SupportSlot struct {
	ID                    string     `json:"id"`
	Kind                  string     `json:"kind"`
	AllowedGroups         []Group    `json:"allowed_groups"`
	RequiredInterface     string     `json:"required_interface,omitempty"`
	DimensionState        string     `json:"dimension_state,omitempty"`
	RequiredPose          string     `json:"required_pose,omitempty"`
	RequiresKnownEnvelope bool       `json:"requires_known_envelope"`
	Evidence              []Evidence `json:"evidence,omitempty"`
}

type CompatibilityStatus string

const (
	Compatible   CompatibilityStatus = "compatible"
	Incompatible CompatibilityStatus = "incompatible"
	NeedsReview  CompatibilityStatus = "needs-review"
)

type Compatibility struct {
	Status  CompatibilityStatus `json:"status"`
	Reasons []string            `json:"reasons"`
}

func (m Measurement) known() bool {
	if m.Value == nil || !finite(*m.Value) || *m.Value <= 0 || (m.Unit != "mm" && m.Unit != "cm" && m.Unit != "m") {
		return false
	}
	if m.Method != "manufacturer" && m.Method != "measured" && m.Method != "derived" && m.Method != "image_estimated" && m.Method != "design_assumption" {
		return false
	}
	for _, evidence := range m.Evidence {
		if strings.TrimSpace(evidence.Source) != "" && strings.TrimSpace(evidence.Kind) != "" {
			return true
		}
	}
	return false
}

func (m Measurement) meters() (float64, bool) {
	if !m.known() {
		return 0, false
	}
	switch m.Unit {
	case "mm":
		return *m.Value / 1000, true
	case "cm":
		return *m.Value / 100, true
	case "m":
		return *m.Value, true
	default:
		return 0, false
	}
}

func (d Definition) Validate() error {
	if d.SchemaVersion != "device-definition/v1" {
		return fmt.Errorf("%s: unsupported schema_version %q", d.ID, d.SchemaVersion)
	}
	if strings.TrimSpace(d.ID) == "" {
		return errors.New("id is required")
	}
	if !validGroups[d.Group] {
		return fmt.Errorf("%s: unknown group %q", d.ID, d.Group)
	}
	if strings.TrimSpace(d.Model) == "" {
		return fmt.Errorf("%s: model label is required", d.ID)
	}
	if d.IdentityStatus != IdentityDocumented && d.IdentityStatus != IdentityGeneric && d.IdentityStatus != IdentityConcept {
		return fmt.Errorf("%s: invalid identity_status %q", d.ID, d.IdentityStatus)
	}
	if d.Qualification != "unqualified" && d.Qualification != "qualified" {
		return fmt.Errorf("%s: invalid qualification status %q", d.ID, d.Qualification)
	}
	if d.Qualification == "qualified" && !hasPassedReview(d, "physical_geometry") {
		return fmt.Errorf("%s: qualified status requires a passed physical_geometry review with reviewer and evidence", d.ID)
	}
	for _, source := range d.Provenance {
		if strings.TrimSpace(source.Source) == "" || strings.TrimSpace(source.Kind) == "" {
			return fmt.Errorf("%s: provenance requires source and kind", d.ID)
		}
	}
	for _, review := range d.QualificationReviews {
		if review.Scope == "" || (review.Result != "passed" && review.Result != "rejected") {
			return fmt.Errorf("%s: invalid qualification review", d.ID)
		}
	}
	if len(d.Dimensions) == 0 {
		return fmt.Errorf("%s: at least one explicit dimension state is required", d.ID)
	}
	for state, dims := range d.Dimensions {
		if strings.TrimSpace(state) == "" {
			return fmt.Errorf("%s: dimension state name is required", d.ID)
		}
		for name, m := range map[string]Measurement{"width": dims.Width, "height": dims.Height, "depth": dims.Depth} {
			if m.Value == nil {
				if m.Method != "unknown" || m.Uncertainty.Status != "unknown" {
					return fmt.Errorf("%s %s.%s: null value must be explicitly unknown", d.ID, state, name)
				}
				continue
			}
			if !m.known() {
				return fmt.Errorf("%s %s.%s: known measurements need positive value, supported unit, method and evidence", d.ID, state, name)
			}
			if m.Uncertainty.Status != "quantified" && m.Uncertainty.Status != "source_unspecified" && m.Uncertainty.Status != "unknown" {
				return fmt.Errorf("%s %s.%s: invalid uncertainty status %q", d.ID, state, name, m.Uncertainty.Status)
			}
			if m.Uncertainty.Status == "quantified" && (m.Uncertainty.Lower == nil || m.Uncertainty.Upper == nil || !finite(*m.Uncertainty.Lower) || !finite(*m.Uncertainty.Upper) || *m.Uncertainty.Lower < 0 || *m.Uncertainty.Upper < *m.Uncertainty.Lower || *m.Value < *m.Uncertainty.Lower || *m.Value > *m.Uncertainty.Upper || m.Uncertainty.Unit != m.Unit) {
				return fmt.Errorf("%s %s.%s: invalid uncertainty interval", d.ID, state, name)
			}
		}
	}
	seen := map[string]bool{}
	for _, surface := range d.Displays {
		if surface.ID == "" || seen[surface.ID] {
			return fmt.Errorf("%s: display IDs must be non-empty and unique", d.ID)
		}
		seen[surface.ID] = true
		if surface.CoordinateMetric != nil {
			m := surface.CoordinateMetric
			if !finite(m.WidthMM) || !finite(m.HeightMM) || m.WidthMM <= 0 || m.HeightMM <= 0 || !hasEvidence(m.Evidence) {
				return fmt.Errorf("%s display %s: coordinate metric needs positive mm spans and source evidence", d.ID, surface.ID)
			}
		}
		for name, region := range map[string]Region{"physical_boundary": surface.PhysicalBoundary, "active_area": surface.ActiveArea, "app_safe_area": surface.AppSafeArea} {
			if err := validateRegion(region); err != nil {
				return fmt.Errorf("%s display %s %s: %w", d.ID, surface.ID, name, err)
			}
		}
		if surface.PhysicalBoundary.Status == "qualified" || surface.ActiveArea.Status == "qualified" || surface.AppSafeArea.Status == "qualified" {
			if surface.CoordinateMetric == nil {
				return fmt.Errorf("%s display %s: qualified normalized regions need an evidenced UV-to-mm coordinate metric", d.ID, surface.ID)
			}
		}
		if surface.Topology == "" {
			return fmt.Errorf("%s display %s: topology is required", d.ID, surface.ID)
		}
		if surface.Topology != "planar" && surface.Topology != "continuous_flexible" && surface.Topology != "curved" && surface.Topology != "dual_rigid" && surface.Topology != "unknown" {
			return fmt.Errorf("%s display %s: invalid topology %q", d.ID, surface.ID, surface.Topology)
		}
		if surface.PhysicalBoundary.Status == "qualified" && surface.ActiveArea.Status == "qualified" {
			if !contains(*surface.PhysicalBoundary.Bounds, *surface.ActiveArea.Bounds) {
				return fmt.Errorf("%s display %s: active-area bounds exceed physical boundary", d.ID, surface.ID)
			}
			if (surface.PhysicalBoundary.Shape != "rect" || surface.ActiveArea.Shape != "rect") && !hasRelation(surface, "active_area", "contained_by", "physical_boundary") {
				return fmt.Errorf("%s display %s: non-rectangular active boundary needs a qualified containment relation", d.ID, surface.ID)
			}
		}
		if surface.ActiveArea.Status == "qualified" && surface.AppSafeArea.Status == "qualified" {
			if !contains(*surface.ActiveArea.Bounds, *surface.AppSafeArea.Bounds) {
				return fmt.Errorf("%s display %s: app-safe-area bounds exceed active-area bounds", d.ID, surface.ID)
			}
			if (surface.ActiveArea.Shape != "rect" || surface.AppSafeArea.Shape != "rect") && !hasRelation(surface, "app_safe_area", "contained_by", "active_area") {
				return fmt.Errorf("%s display %s: non-rectangular active geometry needs a qualified app-safe containment relation", d.ID, surface.ID)
			}
		}
		for _, occlusion := range surface.HardwareOcclusions {
			if occlusion.ID == "" || occlusion.Kind == "" {
				return fmt.Errorf("%s display %s: occlusion ID and kind are required", d.ID, surface.ID)
			}
			if err := validateRegion(occlusion.Region); err != nil {
				return fmt.Errorf("%s display %s occlusion %s: %w", d.ID, surface.ID, occlusion.ID, err)
			}
			if occlusion.Region.Status == "qualified" && surface.PhysicalBoundary.Status == "qualified" && !contains(*surface.PhysicalBoundary.Bounds, *occlusion.Region.Bounds) {
				return fmt.Errorf("%s display %s: qualified occlusion lies outside physical boundary", d.ID, surface.ID)
			}
			if occlusion.Region.Status == "qualified" && surface.AppSafeArea.Status == "qualified" && overlaps(*surface.AppSafeArea.Bounds, *occlusion.Region.Bounds) {
				if surface.AppSafeArea.Shape == "rect" && occlusion.Region.Shape == "rect" {
					return fmt.Errorf("%s display %s: app-safe area overlaps qualified hardware occlusion %s", d.ID, surface.ID, occlusion.ID)
				}
				if !hasRelation(surface, "app_safe_area", "disjoint", "occlusion:"+occlusion.ID) {
					return fmt.Errorf("%s display %s: overlapping non-rectangular bounds need a qualified disjoint relation for occlusion %s", d.ID, surface.ID, occlusion.ID)
				}
			}
		}
		for _, relation := range surface.GeometryRelations {
			if relation.Subject == "" || relation.Object == "" || (relation.Relation != "contained_by" && relation.Relation != "disjoint") || (relation.Status != "qualified" && relation.Status != "candidate_only" && relation.Status != "unknown") {
				return fmt.Errorf("%s display %s: invalid geometry relation", d.ID, surface.ID)
			}
		}
	}
	jointIDs := map[string]bool{}
	for _, j := range d.Joints {
		if j.ID == "" || jointIDs[j.ID] {
			return fmt.Errorf("%s: joint IDs must be non-empty and unique", d.ID)
		}
		jointIDs[j.ID] = true
		if j.ParentPart == "" || j.ChildPart == "" {
			return fmt.Errorf("%s joint %s: parent and child parts are required", d.ID, j.ID)
		}
		if j.Status != "qualified" && j.Status != "candidate_only" && j.Status != "unknown" {
			return fmt.Errorf("%s joint %s: invalid status %q", d.ID, j.ID, j.Status)
		}
		if j.Status == "qualified" {
			if j.Axis == nil || j.PivotMeters == nil || j.LowerRadians == nil || j.UpperRadians == nil {
				return fmt.Errorf("%s joint %s: qualified joint needs axis, pivot and limits", d.ID, j.ID)
			}
			if !finiteVec(*j.Axis) || !finiteVec(*j.PivotMeters) || math.Abs(vecLength(*j.Axis)-1) > 1e-6 {
				return fmt.Errorf("%s joint %s: axis must be a finite unit vector and pivot finite", d.ID, j.ID)
			}
			if !finite(*j.LowerRadians) || !finite(*j.UpperRadians) || *j.LowerRadians > *j.UpperRadians {
				return fmt.Errorf("%s joint %s: invalid radian limits", d.ID, j.ID)
			}
		}
	}
	for _, p := range d.SupportedPoses {
		if strings.TrimSpace(p.Name) == "" || (p.Status != "qualified" && p.Status != "candidate_only" && p.Status != "unknown") {
			return fmt.Errorf("%s: invalid pose", d.ID)
		}
		for id, angle := range p.JointState {
			if strings.TrimSpace(id) == "" || !finite(angle) {
				return fmt.Errorf("%s pose %s: invalid joint state", d.ID, p.Name)
			}
		}
	}
	for _, c := range d.SupportedMotion {
		if c.Name == "" || (c.Status != "qualified" && c.Status != "candidate_only" && c.Status != "unsupported" && c.Status != "unknown") {
			return fmt.Errorf("%s: invalid motion capability", d.ID)
		}
		if c.MotionKind != "" && c.MotionKind != "rigid_body" && c.MotionKind != "articulated" && c.MotionKind != "camera" {
			return fmt.Errorf("%s motion %s: invalid motion_kind %q", d.ID, c.Name, c.MotionKind)
		}
		seenJoints := map[string]bool{}
		for _, id := range c.JointIDs {
			if strings.TrimSpace(id) == "" || seenJoints[id] {
				return fmt.Errorf("%s motion %s: joint IDs must be non-empty and unique", d.ID, c.Name)
			}
			seenJoints[id] = true
		}
	}
	for _, p := range d.SupportedProfiles {
		if strings.TrimSpace(p.Name) == "" || (p.Status != "qualified" && p.Status != "candidate_only" && p.Status != "unsupported" && p.Status != "unknown") {
			return fmt.Errorf("%s: invalid supported profile", d.ID)
		}
	}
	for _, s := range d.Supports {
		if s.Kind == "" || (s.Status != "qualified" && s.Status != "candidate_only" && s.Status != "unknown" && s.Status != "unsupported") {
			return fmt.Errorf("%s: invalid support capability", d.ID)
		}
	}
	for _, a := range d.ModelCandidates {
		if strings.TrimSpace(a.ID) == "" {
			return fmt.Errorf("%s: model candidate ID is required", d.ID)
		}
		if a.SHA256 != "" {
			if len(a.SHA256) != 64 {
				return fmt.Errorf("%s model %s: sha256 must contain 64 hex characters", d.ID, a.ID)
			}
			if _, err := hex.DecodeString(a.SHA256); err != nil {
				return fmt.Errorf("%s model %s: sha256 must be hexadecimal", d.ID, a.ID)
			}
		}
		if a.Status != "candidate_only" && a.Status != "not_acquired" && a.Status != "not_inspected" {
			return fmt.Errorf("%s model %s: invalid candidate status %q", d.ID, a.ID, a.Status)
		}
	}
	return nil
}

func hasRelation(surface DisplaySurface, subject, relation, object string) bool {
	for _, item := range surface.GeometryRelations {
		if item.Subject == subject && item.Relation == relation && item.Object == object && item.Status == "qualified" && hasEvidence(item.Evidence) {
			return true
		}
	}
	return false
}

func (d Definition) ValidateSeed() error {
	if err := d.Validate(); err != nil {
		return err
	}
	if d.Qualification != "unqualified" || len(d.QualificationReviews) != 0 {
		return fmt.Errorf("%s: seed definitions must remain unqualified and contain no qualification review", d.ID)
	}
	return nil
}

func hasPassedReview(d Definition, scope string) bool {
	for _, review := range d.QualificationReviews {
		if review.Scope != scope || review.Result != "passed" || strings.TrimSpace(review.Reviewer) == "" {
			continue
		}
		for _, evidence := range review.Evidence {
			if strings.TrimSpace(evidence.Source) != "" && strings.TrimSpace(evidence.Kind) != "" {
				return true
			}
		}
	}
	return false
}

func validateRegion(r Region) error {
	switch r.Status {
	case "qualified", "candidate_only":
		if r.Bounds == nil {
			return errors.New("region bounds required for known/candidate region")
		}
	case "unknown", "not_applicable":
		if r.Bounds != nil {
			return errors.New("unknown/not-applicable region must not invent bounds")
		}
	default:
		return fmt.Errorf("invalid region status %q", r.Status)
	}
	if r.Bounds != nil {
		b := r.Bounds
		if !finite(b.X) || !finite(b.Y) || !finite(b.Width) || !finite(b.Height) || b.X < 0 || b.Y < 0 || b.Width <= 0 || b.Height <= 0 || b.X+b.Width > 1 || b.Y+b.Height > 1 {
			return errors.New("bounds must be finite normalized rectangle inside [0,1]")
		}
	}
	if r.Shape != "" && r.Shape != "unknown" && r.Shape != "rect" && r.Shape != "rounded_rect" && r.Shape != "circle" && r.Shape != "polygon" && r.Shape != "mask" {
		return fmt.Errorf("invalid region shape %q", r.Shape)
	}
	if r.Status == "qualified" {
		if r.Shape == "" || r.Shape == "unknown" {
			return errors.New("qualified region needs a concrete shape")
		}
		if !hasEvidence(r.Evidence) {
			return errors.New("qualified region needs shape evidence")
		}
		if r.Shape != "rect" && (strings.TrimSpace(r.GeometryRef) == "" || r.GeometrySpace != "physical_mm") {
			return errors.New("qualified non-rectangular region needs a physical-mm geometry_ref")
		}
	}
	if r.GeometryRef != "" && r.GeometrySpace != "physical_mm" {
		return errors.New("geometry_ref must identify a physical-mm artifact")
	}
	return nil
}

func hasEvidence(evidence []Evidence) bool {
	for _, item := range evidence {
		if strings.TrimSpace(item.Source) != "" && strings.TrimSpace(item.Kind) != "" {
			return true
		}
	}
	return false
}

func (d Definition) PhysicalQualification() Compatibility {
	if err := d.Validate(); err != nil {
		return Compatibility{Status: Incompatible, Reasons: []string{"definition is invalid: " + err.Error()}}
	}
	result := Compatibility{Status: Compatible}
	addReview := func(reason string) {
		result.Reasons = append(result.Reasons, reason)
		if result.Status != Incompatible {
			result.Status = NeedsReview
		}
	}
	if !hasPassedReview(d, "physical_geometry") {
		addReview("a passed physical_geometry review with reviewer and evidence is absent")
	}
	for state, dims := range d.Dimensions {
		for name, m := range map[string]Measurement{"width": dims.Width, "height": dims.Height, "depth": dims.Depth} {
			if !m.known() {
				addReview("dimension " + state + "." + name + " is unknown")
				continue
			}
			if m.Uncertainty.Status == "unknown" {
				addReview("dimension " + state + "." + name + " has unknown uncertainty")
			}
			if m.Method == "design_assumption" {
				addReview("dimension " + state + "." + name + " is a design assumption, not a physical measurement")
			}
		}
	}
	if len(d.Displays) == 0 {
		addReview("no display topology is defined")
	}
	for _, s := range d.Displays {
		for name, r := range map[string]Region{"physical boundary": s.PhysicalBoundary, "active area": s.ActiveArea, "app-safe area": s.AppSafeArea} {
			if r.Status != "qualified" || r.Bounds == nil {
				addReview("display " + s.ID + " " + name + " is not qualified")
			}
		}
		for _, o := range s.HardwareOcclusions {
			if o.Region.Status != "qualified" || o.Region.Bounds == nil {
				addReview("display " + s.ID + " occlusion " + o.ID + " is not qualified")
			}
		}
	}
	for _, j := range d.Joints {
		if j.Status != "qualified" {
			addReview("joint " + j.ID + " limits/pivot are not qualified")
		}
	}
	if d.Qualification != "qualified" {
		addReview("definition status is unqualified")
	}
	if len(result.Reasons) > 0 {
		result.Status = NeedsReview
	}
	return result
}

func CheckSupport(d Definition, slot SupportSlot) Compatibility {
	if err := d.Validate(); err != nil {
		return Compatibility{Status: Incompatible, Reasons: []string{"device definition is invalid: " + err.Error()}}
	}
	result := Compatibility{Status: Compatible}
	if slot.ID == "" || slot.Kind == "" {
		return Compatibility{Status: NeedsReview, Reasons: []string{"support slot identity or kind is missing"}}
	}
	if len(slot.AllowedGroups) > 0 {
		allowed := false
		for _, g := range slot.AllowedGroups {
			if g == d.Group {
				allowed = true
				break
			}
		}
		if !allowed {
			return Compatibility{Status: Incompatible, Reasons: []string{"device group is not allowed by support slot"}}
		}
	}
	var found *SupportCapability
	for i := range d.Supports {
		if d.Supports[i].Kind == slot.Kind {
			found = &d.Supports[i]
			break
		}
	}
	if found == nil {
		return Compatibility{Status: Incompatible, Reasons: []string{"device declares no support capability for slot kind"}}
	}
	if found.Status == "unsupported" {
		return Compatibility{Status: Incompatible, Reasons: []string{"device explicitly does not support slot kind"}}
	}
	if found.InterfaceID != "" && slot.RequiredInterface != "" && found.InterfaceID != slot.RequiredInterface && found.Status == "qualified" {
		return Compatibility{Status: Incompatible, Reasons: []string{"qualified interface does not match required interface"}}
	}
	if found.Status != "qualified" {
		result.Status = NeedsReview
		result.Reasons = append(result.Reasons, "device support capability is not qualified")
	}
	if slot.RequiredInterface != "" && found.InterfaceID != slot.RequiredInterface {
		result.Status = NeedsReview
		result.Reasons = append(result.Reasons, "candidate interface does not establish a match")
	}
	if slot.DimensionState == "" {
		result.Status = NeedsReview
		result.Reasons = append(result.Reasons, "support requires an explicit dimension state")
	} else if dims, ok := d.Dimensions[slot.DimensionState]; !ok {
		result.Status = NeedsReview
		result.Reasons = append(result.Reasons, "support dimension state is not declared: "+slot.DimensionState)
	} else if slot.RequiresKnownEnvelope && (!dims.Width.known() || !dims.Height.known() || !dims.Depth.known()) {
		result.Status = NeedsReview
		result.Reasons = append(result.Reasons, "support envelope is not established for dimension state "+slot.DimensionState)
	}
	if slot.RequiredPose != "" {
		poseResult := CheckPose(d, slot.RequiredPose)
		if poseResult.Status == Incompatible {
			return poseResult
		}
		if poseResult.Status != Compatible {
			result.Status = NeedsReview
			result.Reasons = append(result.Reasons, "required pose is not qualified")
		}
	}
	return result
}

// CheckPose only treats a named pose as compatible when it is qualified and
// every referenced joint is itself qualified and within its declared limits.
func CheckPose(d Definition, name string) Compatibility {
	if err := d.Validate(); err != nil {
		return Compatibility{Status: Incompatible, Reasons: []string{"device definition is invalid: " + err.Error()}}
	}
	for _, pose := range d.SupportedPoses {
		if pose.Name != name {
			continue
		}
		if pose.Status == "unknown" || pose.Status == "candidate_only" {
			return Compatibility{Status: NeedsReview, Reasons: []string{"requested pose is not qualified"}}
		}
		if pose.Status != "qualified" {
			return Compatibility{Status: Incompatible, Reasons: []string{"requested pose is not supported"}}
		}
		joints := make(map[string]Joint, len(d.Joints))
		for _, joint := range d.Joints {
			joints[joint.ID] = joint
		}
		for id, joint := range joints {
			if _, ok := pose.JointState[id]; !ok {
				return Compatibility{Status: NeedsReview, Reasons: []string{"pose omits a declared joint state: " + id}}
			}
			if joint.Status != "qualified" {
				return Compatibility{Status: NeedsReview, Reasons: []string{"pose depends on an unqualified joint: " + id}}
			}
		}
		for id, angle := range pose.JointState {
			joint, ok := joints[id]
			if !ok || joint.Status != "qualified" || joint.LowerRadians == nil || joint.UpperRadians == nil {
				return Compatibility{Status: NeedsReview, Reasons: []string{"pose references an unqualified joint: " + id}}
			}
			if !finite(angle) || angle < *joint.LowerRadians || angle > *joint.UpperRadians {
				return Compatibility{Status: Incompatible, Reasons: []string{"pose angle is outside the qualified joint limits: " + id}}
			}
		}
		return Compatibility{Status: Compatible, Reasons: []string{}}
	}
	return Compatibility{Status: Incompatible, Reasons: []string{"requested pose is not declared"}}
}

// CheckMotion distinguishes supported motion from unknown or merely candidate
// motion. Articulation is explicit in MotionKind and JointIDs, not inferred
// from a display label.
func CheckMotion(d Definition, name string) Compatibility {
	if err := d.Validate(); err != nil {
		return Compatibility{Status: Incompatible, Reasons: []string{"device definition is invalid: " + err.Error()}}
	}
	for _, motion := range d.SupportedMotion {
		if motion.Name != name {
			continue
		}
		switch motion.Status {
		case "unsupported":
			return Compatibility{Status: Incompatible, Reasons: []string{"requested motion is explicitly unsupported"}}
		case "unknown", "candidate_only":
			return Compatibility{Status: NeedsReview, Reasons: []string{"requested motion is not qualified"}}
		case "qualified":
			switch motion.MotionKind {
			case "rigid_body", "camera":
				if len(motion.JointIDs) != 0 {
					return Compatibility{Status: Incompatible, Reasons: []string{"non-articulated motion must not declare joints"}}
				}
				return Compatibility{Status: Compatible}
			case "articulated":
				if len(motion.JointIDs) == 0 {
					return Compatibility{Status: NeedsReview, Reasons: []string{"articulated motion has no explicit joint IDs"}}
				}
				joints := make(map[string]Joint, len(d.Joints))
				for _, joint := range d.Joints {
					joints[joint.ID] = joint
				}
				for _, id := range motion.JointIDs {
					joint, ok := joints[id]
					if !ok || joint.Status != "qualified" || joint.Axis == nil || joint.PivotMeters == nil || joint.LowerRadians == nil || joint.UpperRadians == nil {
						return Compatibility{Status: NeedsReview, Reasons: []string{"motion references an unqualified joint: " + id}}
					}
				}
				return Compatibility{Status: Compatible}
			default:
				return Compatibility{Status: NeedsReview, Reasons: []string{"qualified motion has no explicit motion_kind"}}
			}
		default:
			return Compatibility{Status: NeedsReview, Reasons: []string{"motion status is invalid"}}
		}
	}
	return Compatibility{Status: Incompatible, Reasons: []string{"requested motion is not declared"}}
}

func LoadDefinitions(dir string) ([]Definition, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	defs := make([]Definition, 0, len(paths))
	ids := make(map[string]string, len(paths))
	identities := make(map[string]string, len(paths))
	for _, p := range paths {
		b, e := os.ReadFile(p)
		if e != nil {
			return nil, e
		}
		var d Definition
		decoder := json.NewDecoder(bytes.NewReader(b))
		decoder.DisallowUnknownFields()
		if e = decoder.Decode(&d); e != nil {
			return nil, fmt.Errorf("%s: %w", p, e)
		}
		var trailing any
		if e = decoder.Decode(&trailing); e != io.EOF {
			if e == nil {
				return nil, fmt.Errorf("%s: trailing JSON value after device definition", p)
			}
			return nil, fmt.Errorf("%s: invalid trailing content: %w", p, e)
		}
		if e = d.ValidateSeed(); e != nil {
			return nil, e
		}
		if prior, exists := ids[d.ID]; exists {
			return nil, fmt.Errorf("duplicate device definition ID %q in %s and %s", d.ID, prior, p)
		}
		ids[d.ID] = p
		identity := strings.Join([]string{strings.ToLower(strings.TrimSpace(string(d.Group))), strings.ToLower(strings.TrimSpace(d.Kind)), strings.ToLower(strings.TrimSpace(d.Manufacturer)), strings.ToLower(strings.TrimSpace(d.Model)), strings.ToLower(strings.TrimSpace(d.Variant))}, "\x00")
		if prior, exists := identities[identity]; exists {
			return nil, fmt.Errorf("duplicate device identity %q in %s and %s", d.ID, prior, p)
		}
		identities[identity] = p
		defs = append(defs, d)
	}
	return defs, nil
}

func finite(x float64) bool       { return !math.IsNaN(x) && !math.IsInf(x, 0) }
func finiteVec(v Vector3) bool    { return finite(v.X) && finite(v.Y) && finite(v.Z) }
func vecLength(v Vector3) float64 { return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z) }
func contains(outer, inner Rect) bool {
	return inner.X >= outer.X && inner.Y >= outer.Y && inner.X+inner.Width <= outer.X+outer.Width && inner.Y+inner.Height <= outer.Y+outer.Height
}
func overlaps(a, b Rect) bool {
	return a.X < b.X+b.Width && a.X+a.Width > b.X && a.Y < b.Y+b.Height && a.Y+a.Height > b.Y
}
