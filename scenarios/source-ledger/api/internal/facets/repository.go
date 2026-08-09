package facets

import "context"

type Repository interface {
	Seed(context.Context) error
	List(context.Context) ([]Definition, error)
	SetPolicy(context.Context, FacetPolicy) (Definition, error)
	ListCorpus(context.Context, string) ([]CorpusEntry, error)
	Validate(context.Context, string) error
	CompactionEligible(context.Context, string) (bool, error)
	SetPin(context.Context, string, bool) error
	Pinned(context.Context, string) (bool, error)
	Assign(context.Context, Assignment) (Assignment, error)
	Assignments(context.Context, string) ([]Assignment, error)
	MarkSuperseded(context.Context, string, string) error
	ResolveThread(context.Context, string) error
	ListPinProposals(context.Context) ([]PinProposal, error)
	ResolvePinProposal(context.Context, string, bool) error
	RecordRecall(context.Context, []string) error
	ListPinCandidates(context.Context, int) ([]PinCandidate, error)
	CreateRule(context.Context, Rule) (Rule, error)
	ListRules(context.Context, string) ([]Rule, error)
	MatchRule(context.Context, string, RuleInput) (Rule, bool, error)
	DryRunRule(context.Context, string) (DryRun, error)
	MeasureDistribution(context.Context, string) (DistributionMeasurement, error)
	EnableRule(context.Context, string) error
	RevertRule(context.Context, string) (int, error)
	SeedExamples(context.Context) error
}
