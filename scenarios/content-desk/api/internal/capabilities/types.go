// Package capabilities owns the persisted marketing-capability catalog:
// capability records, their observed qualifications, and typed links to other
// owner records. It is the single source of truth for capability readiness; it
// never copies campaign, draft, claim, publish, or metric state.
package capabilities

import (
	"errors"
	"time"
)

// Unknown is the explicit sentinel for absent or unobserved facts. An absent
// qualification reads as Unknown rather than an empty string or zero value, so
// "unknown" is never silently reported as "zero".
const Unknown = "unknown"

// UnknownMaxAgeSeconds is the sentinel max-age for a qualification that has no
// observed freshness bound. It is negative, not zero, to preserve unknown.
const UnknownMaxAgeSeconds int64 = -1

// Capability media. Each capability declares exactly one medium.
const (
	MediumText         = "text"
	MediumImage        = "image"
	MediumVideo        = "video"
	MediumAudio        = "audio"
	MediumDistribution = "distribution"
	MediumMeasurement  = "measurement"
	MediumGovernance   = "governance"
)

// Definition status: whether the capability is documented.
const (
	DefinitionDocumented = "documented"
	DefinitionAbsent     = "absent"
)

// Implementation status: how far the producing operation is built.
const (
	ImplementationNotStarted  = "not-started"
	ImplementationSkeleton    = "skeleton"
	ImplementationImplemented = "implemented"
)

// Operational readiness: whether the producing operation works in an observed
// environment. It is independent of implementation status.
const (
	ReadinessQualified   = "qualified-in-environment"
	ReadinessUnverified  = "unverified"
	ReadinessUnavailable = "unavailable"
)

// Output quality: whether produced output has been accepted by review.
const (
	QualityAccepted   = "accepted-by-review"
	QualityUnassessed = "unassessed"
)

// Distribution connectivity: whether the capability is wired to a live
// distribution surface.
const (
	ConnectivityConnected     = "connected"
	ConnectivityDisconnected  = "disconnected"
	ConnectivityNotApplicable = "not-applicable"
)

// Freshness basis names how a qualification's evidence is bound.
const (
	FreshnessCandidateIdentity = "candidate_identity"
	FreshnessMaxAge            = "max_age"
)

// Capability link relations. Relations are a closed set so an unknown or empty
// relation is rejected at the domain boundary.
const (
	RelationProduces     = "produces"
	RelationPrerequisite = "prerequisite"
	RelationEvidence     = "evidence"
	RelationDistribution = "distribution"
	RelationOwner        = "owner"
	RelationRelated      = "related"
)

var (
	// ErrNotFound is returned when no capability matches the reference.
	ErrNotFound = errors.New("capability not found")
	// ErrReferenceRequired is returned when neither an id nor an alias is given.
	ErrReferenceRequired = errors.New("capability id or alias is required")
	// ErrInvalidMedium is returned for a medium outside the closed set.
	ErrInvalidMedium = errors.New("invalid capability medium")
	// ErrInvalidStatus is returned for a status outside its closed set.
	ErrInvalidStatus = errors.New("invalid capability status")
	// ErrInvalidFreshness is returned for a freshness basis outside its set.
	ErrInvalidFreshness = errors.New("invalid freshness basis")
	// ErrInvalidRelation is returned for a link relation outside the closed set.
	ErrInvalidRelation = errors.New("invalid capability link relation")
	// ErrInvalidTarget is returned when a link target id is empty.
	ErrInvalidTarget = errors.New("capability link target id is required")
	// ErrInvalidName is returned when a capability name is empty.
	ErrInvalidName = errors.New("capability name is required")
)

// Capability is one marketing capability and its five separate readiness
// dimensions. The dimensions are stored and returned independently: an
// implementation or healthy service does not establish usable output.
type Capability struct {
	ID                       string
	Name                     string
	Medium                   string
	Aliases                  []string
	Channels                 []string
	AudienceApplicability    string
	DeliveryApplicability    string
	ProducingOperation       string
	Prerequisites            []string
	Priority                 int32
	PriorityReason           string
	PriorityScope            string
	DefinitionStatus         string
	ImplementationStatus     string
	OperationalReadiness     string
	OutputQuality            string
	DistributionConnectivity string
	Owner                    string
	SourceRefs               []string
	LatestQualification      *CapabilityQualification
	NextAction               string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

// HasQualification reports whether the capability carries observed
// qualification evidence rather than the unknown sentinel.
func (c Capability) HasQualification() bool {
	return c.LatestQualification != nil && !c.LatestQualification.IsUnknown()
}

// CapabilityQualification is observed evidence for one capability.
type CapabilityQualification struct {
	ID                string
	CapabilityID      string
	LatestArtifactID  string
	LatestRunID       string
	Environment       string
	ValidatedAt       string
	ObservedAt        string
	FreshnessBasis    string
	CandidateIdentity string
	MaxAgeSeconds     int64
	Limitation        string
	NextAction        string
	CreatedAt         time.Time
}

// IsUnknown reports whether the qualification is the explicit unknown sentinel
// (absent or unobserved data) rather than a recorded observation.
func (q CapabilityQualification) IsUnknown() bool {
	return q.ObservedAt == Unknown || q.ObservedAt == ""
}

// UnknownQualification returns an all-unknown qualification for capabilityID.
// Callers use it to make absence explicit instead of emitting an empty message.
func UnknownQualification(capabilityID string) CapabilityQualification {
	return CapabilityQualification{
		CapabilityID:      capabilityID,
		LatestArtifactID:  Unknown,
		LatestRunID:       Unknown,
		Environment:       Unknown,
		ValidatedAt:       Unknown,
		ObservedAt:        Unknown,
		FreshnessBasis:    Unknown,
		CandidateIdentity: Unknown,
		MaxAgeSeconds:     UnknownMaxAgeSeconds,
		Limitation:        Unknown,
		NextAction:        Unknown,
	}
}

// CapabilityLink relates a capability to another owner record by id without
// copying that record's state.
type CapabilityLink struct {
	ID           string
	CapabilityID string
	Relation     string
	TargetID     string
	CreatedAt    time.Time
}

// IsValidMedium reports whether medium is in the closed set.
func IsValidMedium(medium string) bool {
	switch medium {
	case MediumText, MediumImage, MediumVideo, MediumAudio, MediumDistribution, MediumMeasurement, MediumGovernance:
		return true
	}
	return false
}

// IsValidDefinitionStatus reports whether status is in the closed set.
func IsValidDefinitionStatus(status string) bool {
	return status == DefinitionDocumented || status == DefinitionAbsent
}

// IsValidImplementationStatus reports whether status is in the closed set.
func IsValidImplementationStatus(status string) bool {
	switch status {
	case ImplementationNotStarted, ImplementationSkeleton, ImplementationImplemented:
		return true
	}
	return false
}

// IsValidOperationalReadiness reports whether readiness is in the closed set.
func IsValidOperationalReadiness(readiness string) bool {
	switch readiness {
	case ReadinessQualified, ReadinessUnverified, ReadinessUnavailable:
		return true
	}
	return false
}

// IsValidOutputQuality reports whether quality is in the closed set.
func IsValidOutputQuality(quality string) bool {
	return quality == QualityAccepted || quality == QualityUnassessed
}

// IsValidDistributionConnectivity reports whether connectivity is in the closed
// set.
func IsValidDistributionConnectivity(connectivity string) bool {
	switch connectivity {
	case ConnectivityConnected, ConnectivityDisconnected, ConnectivityNotApplicable:
		return true
	}
	return false
}

// IsValidFreshnessBasis reports whether basis is in the closed set.
func IsValidFreshnessBasis(basis string) bool {
	return basis == FreshnessCandidateIdentity || basis == FreshnessMaxAge
}

// IsValidRelation reports whether relation is in the closed set.
func IsValidRelation(relation string) bool {
	switch relation {
	case RelationProduces, RelationPrerequisite, RelationEvidence, RelationDistribution, RelationOwner, RelationRelated:
		return true
	}
	return false
}
