package memberflow

import "strings"

type OperatingModelReferenceKind string

const (
	OperatingModelReferenceKindTopic    OperatingModelReferenceKind = "topic"
	OperatingModelReferenceKindMember   OperatingModelReferenceKind = "member"
	OperatingModelReferenceKindTeam     OperatingModelReferenceKind = "team"
	OperatingModelReferenceKindExternal OperatingModelReferenceKind = "external"
	OperatingModelReferenceKindProcess  OperatingModelReferenceKind = "process"
	OperatingModelReferenceKindPOR      OperatingModelReferenceKind = "por"
	OperatingModelReferenceKindPath     OperatingModelReferenceKind = "path"
	OperatingModelReferenceKindCommand  OperatingModelReferenceKind = "command"
)

type OperatingModelReference struct {
	Kind      OperatingModelReferenceKind
	Qualifier OperatingGraphQualifier
	Value     string
	Raw       string
	Line      int
	Surface   string
}

type OperatingModelReferenceIndex struct {
	Model          OperatingModelDocument
	Runtime        OperatingGraphRuntime
	References     []OperatingModelReference
	ExternalInputs []OperatingExternalInputAssurance
	Outputs        []OperatingOutputAssurance
	Feedback       []OperatingFeedbackStepAssurance
}

func NewOperatingModelRuleContext(model OperatingModelDocument, runtime OperatingGraphRuntime) OperatingModelRuleContext {
	return OperatingModelRuleContext{
		Model:          model,
		Runtime:        runtime,
		ReferenceIndex: NewOperatingModelReferenceIndex(model, runtime),
	}
}

func NewOperatingModelReferenceIndex(model OperatingModelDocument, runtime OperatingGraphRuntime) OperatingModelReferenceIndex {
	idx := OperatingModelReferenceIndex{Model: model, Runtime: runtime}
	idx.References = idx.collectReferences()
	idx.ExternalInputs = idx.collectExternalInputAssurances()
	idx.Outputs = idx.collectOutputAssurances()
	idx.Feedback = idx.collectFeedbackAssurances()
	return idx
}

func (idx OperatingModelReferenceIndex) ExternalInputAssurance(row OperatingExternalInputRow) OperatingExternalInputAssurance {
	for _, assurance := range idx.ExternalInputs {
		if operatingExternalInputRowsEqual(assurance.Row, row) {
			return assurance
		}
	}
	return idx.externalInputAssurance(row)
}

func (idx OperatingModelReferenceIndex) OutputAssurance(row OperatingOutputRow) OperatingOutputAssurance {
	for _, assurance := range idx.Outputs {
		if operatingOutputRowsEqual(assurance.Row, row) {
			return assurance
		}
	}
	return idx.outputAssurance(row)
}

func (idx OperatingModelReferenceIndex) FeedbackReferenceAssurance(ref string) OperatingFeedbackReferenceAssurance {
	return OperatingFeedbackReferenceAssurance{Reference: ref, Backed: idx.feedbackReferenceBacked(ref)}
}

func (idx OperatingModelReferenceIndex) SurfaceReferences(raw string) []operatingSurfaceReference {
	return operatingSurfaceReferences(raw)
}

func (idx OperatingModelReferenceIndex) ActorReferences(raw string) []OperatingActorReference {
	return idx.actorRefs(raw)
}

type OperatingExternalInputAssurance struct {
	Row      OperatingExternalInputRow
	Producer bool
	Entry    bool
	Drainer  bool
}

func (assurance OperatingExternalInputAssurance) Backed() bool {
	return assurance.Producer && assurance.Entry && assurance.Drainer
}

type OperatingOutputAssurance struct {
	Row      OperatingOutputRow
	Surface  bool
	Consumer bool
}

func (assurance OperatingOutputAssurance) Backed() bool {
	return assurance.Surface && assurance.Consumer
}

type OperatingFeedbackStepAssurance struct {
	Step       OperatingFeedbackStep
	References []OperatingFeedbackReferenceAssurance
	Anchored   bool
}

type OperatingFeedbackReferenceAssurance struct {
	Reference string
	Backed    bool
}

func (assurance OperatingFeedbackStepAssurance) UnbackedReferences() int {
	var unbacked int
	for _, ref := range assurance.References {
		if !ref.Backed {
			unbacked++
		}
	}
	return unbacked
}

type operatingSurfaceReference struct {
	Kind      OperatingGraphNodeKind
	Qualifier OperatingGraphQualifier
	Value     string
}

func (idx OperatingModelReferenceIndex) collectExternalInputAssurances() []OperatingExternalInputAssurance {
	out := make([]OperatingExternalInputAssurance, 0, len(idx.Model.Sections.ExternalInputs.Rows))
	for _, row := range idx.Model.Sections.ExternalInputs.Rows {
		out = append(out, idx.externalInputAssurance(row))
	}
	return out
}

func (idx OperatingModelReferenceIndex) collectOutputAssurances() []OperatingOutputAssurance {
	out := make([]OperatingOutputAssurance, 0, len(idx.Model.Sections.Outputs.Rows))
	for _, row := range idx.Model.Sections.Outputs.Rows {
		out = append(out, idx.outputAssurance(row))
	}
	return out
}

func (idx OperatingModelReferenceIndex) collectFeedbackAssurances() []OperatingFeedbackStepAssurance {
	out := make([]OperatingFeedbackStepAssurance, 0, len(idx.Model.Sections.FeedbackLoop.Steps))
	for _, step := range idx.Model.Sections.FeedbackLoop.Steps {
		assurance := OperatingFeedbackStepAssurance{Step: step}
		for _, ref := range step.References {
			refAssurance := idx.FeedbackReferenceAssurance(ref)
			if refAssurance.Backed {
				assurance.Anchored = true
			}
			assurance.References = append(assurance.References, refAssurance)
		}
		out = append(out, assurance)
	}
	return out
}

func (idx OperatingModelReferenceIndex) externalInputAssurance(row OperatingExternalInputRow) OperatingExternalInputAssurance {
	if operatingModelRowIsTargetState(row.ProducerTrigger, row.EntrySurface, row.Drainer, row.RoutingRule) {
		return OperatingExternalInputAssurance{Row: row, Producer: true, Entry: true, Drainer: true}
	}
	producers := idx.actorRefs(row.ProducerTrigger)
	surfaces := operatingSurfaceReferences(row.EntrySurface)
	return OperatingExternalInputAssurance{
		Row:      row,
		Producer: idx.externalInputProducerBacked(producers),
		Entry:    idx.entrySurfaceBacked(row.EntrySurface, surfaces),
		Drainer:  idx.externalInputDrainerBacked(producers, surfaces, idx.actorRefs(row.Drainer), row.Drainer),
	}
}

func (idx OperatingModelReferenceIndex) outputAssurance(row OperatingOutputRow) OperatingOutputAssurance {
	if operatingModelRowIsTargetState(row.Output, row.Surface, row.Consumer, row.Purpose) {
		return OperatingOutputAssurance{Row: row, Surface: true, Consumer: true}
	}
	surfaces := operatingSurfaceReferences(row.Output + " " + row.Surface)
	surfaceBacked := idx.outputSurfaceBacked(row.Surface, surfaces)
	return OperatingOutputAssurance{
		Row:      row,
		Surface:  surfaceBacked,
		Consumer: idx.outputConsumerBacked(surfaces, idx.actorRefs(row.Consumer), row.Consumer) || surfaceBacked && operatingOutputConsumerTextIsDownstream(row.Consumer),
	}
}

func (idx OperatingModelReferenceIndex) feedbackReferenceBacked(ref string) bool {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return false
	}
	normalized := strings.TrimPrefix(ref, "topic:")
	normalized = strings.TrimPrefix(normalized, "path:")
	if idx.feedbackGraphNodeBacked(ref, normalized) {
		return true
	}
	if idx.topicCatalogHasTopic(normalized) {
		return true
	}
	if idx.feedbackExternalInputBacked(ref, normalized) {
		return true
	}
	if idx.feedbackOutputBacked(ref, normalized) {
		return true
	}
	return false
}

func (idx OperatingModelReferenceIndex) feedbackGraphNodeBacked(raw, normalized string) bool {
	for _, block := range idx.Model.Graphs {
		for _, node := range block.Graph.Nodes {
			if node.Value != normalized && string(node.Kind)+":"+node.Value != raw {
				continue
			}
			switch node.Kind {
			case OperatingGraphNodeKindTopic, OperatingGraphNodeKindMember, OperatingGraphNodeKindTeam, OperatingGraphNodeKindPOR, OperatingGraphNodeKindExternal:
				return true
			}
		}
	}
	return false
}

func (idx OperatingModelReferenceIndex) feedbackExternalInputBacked(raw, normalized string) bool {
	for _, row := range idx.Model.Sections.ExternalInputs.Rows {
		if operatingFeedbackTextMentions(row.ProducerTrigger, raw, normalized) ||
			operatingFeedbackTextMentions(row.EntrySurface, raw, normalized) ||
			operatingFeedbackTextMentions(row.Drainer, raw, normalized) ||
			operatingFeedbackTextMentions(row.RoutingRule, raw, normalized) {
			return true
		}
	}
	return false
}

func (idx OperatingModelReferenceIndex) feedbackOutputBacked(raw, normalized string) bool {
	for _, row := range idx.Model.Sections.Outputs.Rows {
		if operatingFeedbackTextMentions(row.Output, raw, normalized) ||
			operatingFeedbackTextMentions(row.Surface, raw, normalized) ||
			operatingFeedbackTextMentions(row.Consumer, raw, normalized) ||
			operatingFeedbackTextMentions(row.Purpose, raw, normalized) {
			return true
		}
	}
	return false
}

func operatingFeedbackTextMentions(text, raw, normalized string) bool {
	text = strings.ToLower(text)
	raw = strings.ToLower(raw)
	normalized = strings.ToLower(normalized)
	return strings.Contains(text, raw) || strings.Contains(text, normalized)
}

func (idx OperatingModelReferenceIndex) actorRefs(raw string) []OperatingActorReference {
	if len(idx.Model.Graphs) == 0 {
		return nil
	}
	resolver := NewOperatingActorResolver(idx.Model.Graphs[0].Metadata, idx.graphs()...)
	var refs []OperatingActorReference
	for _, token := range extractInlineCodeTokens(raw) {
		refs = append(refs, idx.expandActorRefs(resolver, resolver.Resolve(idx.Model.Team, idx.Runtime, token))...)
	}
	refs = append(refs, idx.expandActorRefs(resolver, resolver.Resolve(idx.Model.Team, idx.Runtime, raw))...)
	if strings.Contains(strings.ToLower(raw), "operator") {
		refs = append(refs, idx.expandActorRefs(resolver, resolver.Resolve(idx.Model.Team, idx.Runtime, "operator"))...)
	}
	return dedupeOperatingActorRefs(refs)
}

func (idx OperatingModelReferenceIndex) expandActorRefs(resolver DefaultOperatingActorResolver, refs []OperatingActorReference) []OperatingActorReference {
	var out []OperatingActorReference
	for _, ref := range refs {
		out = append(out, resolver.Expand(idx.Model.Team, idx.Runtime, ref)...)
	}
	return out
}

func (idx OperatingModelReferenceIndex) graphs() []OperatingGraph {
	graphs := make([]OperatingGraph, 0, len(idx.Model.Graphs))
	for _, block := range idx.Model.Graphs {
		graphs = append(graphs, block.Graph)
	}
	return graphs
}

func dedupeOperatingActorRefs(refs []OperatingActorReference) []OperatingActorReference {
	seen := map[string]bool{}
	var out []OperatingActorReference
	for _, ref := range refs {
		if ref.Kind == "" || ref.Value == "" {
			continue
		}
		key := string(ref.Kind) + "\x00" + ref.Value
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, ref)
	}
	return out
}

func (idx OperatingModelReferenceIndex) externalInputProducerBacked(producers []OperatingActorReference) bool {
	var externalSeen bool
	for _, producer := range producers {
		if producer.Kind != OperatingActorKindExternal {
			continue
		}
		externalSeen = true
		if idx.graphHasNode(OperatingGraphNodeKindExternal, producer.Value) ||
			idx.graphHasRelationship(func(rel OperatingRelationship) bool {
				return (rel.Kind == operatingRelExternalProducer || rel.Kind == operatingRelExternalProducerIntake) && rel.External == producer.Value
			}) {
			return true
		}
	}
	return !externalSeen
}

func (idx OperatingModelReferenceIndex) entrySurfaceBacked(raw string, surfaces []operatingSurfaceReference) bool {
	if len(surfaces) == 0 {
		normalized := strings.ToLower(raw)
		return strings.Contains(normalized, "direct member context") || strings.Contains(normalized, "heartbeat trigger")
	}
	for _, surface := range surfaces {
		if operatingSurfaceReferenceIsTargetState(surface) {
			continue
		}
		if idx.surfaceReferenceBacked(surface) {
			return true
		}
	}
	return false
}

func (idx OperatingModelReferenceIndex) outputSurfaceBacked(raw string, surfaces []operatingSurfaceReference) bool {
	if len(surfaces) == 0 {
		return idx.surfaceTextMentionsBackedPath(raw)
	}
	for _, surface := range surfaces {
		if operatingSurfaceReferenceIsTargetState(surface) {
			continue
		}
		if idx.surfaceReferenceBacked(surface) {
			return true
		}
	}
	return false
}

func (idx OperatingModelReferenceIndex) externalInputDrainerBacked(producers []OperatingActorReference, surfaces []operatingSurfaceReference, drainers []OperatingActorReference, raw string) bool {
	if len(drainers) == 0 {
		return false
	}
	if strings.Contains(strings.ToLower(raw), "work owner") && idx.surfacesContainBackedTopic(surfaces) {
		return true
	}
	for _, drainer := range drainers {
		if idx.actorExistsInGraph(drainer) {
			if len(surfaces) == 0 && strings.Contains(strings.ToLower(raw), "direct member context") {
				return true
			}
		}
		switch drainer.Kind {
		case OperatingActorKindMember:
			if idx.memberDrainerBacked(producers, surfaces, drainer.Value) {
				return true
			}
		case OperatingActorKindTeam:
			if idx.teamConsumerBacked(surfaces, drainer.Value) {
				return true
			}
		case OperatingActorKindExternal:
			if idx.graphHasNode(OperatingGraphNodeKindExternal, drainer.Value) {
				return true
			}
		}
	}
	return false
}

func (idx OperatingModelReferenceIndex) outputConsumerBacked(surfaces []operatingSurfaceReference, consumers []OperatingActorReference, raw string) bool {
	if len(consumers) == 0 {
		return false
	}
	for _, consumer := range consumers {
		if idx.actorExistsInGraph(consumer) {
			return true
		}
		switch consumer.Kind {
		case OperatingActorKindMember:
			if idx.memberSurfaceConsumerBacked(surfaces, consumer.Value) {
				return true
			}
		case OperatingActorKindTeam:
			if idx.teamConsumerBacked(surfaces, consumer.Value) {
				return true
			}
		case OperatingActorKindExternal:
			if idx.graphHasNode(OperatingGraphNodeKindExternal, consumer.Value) {
				return true
			}
		}
	}
	return strings.Contains(strings.ToLower(raw), "work owner") && idx.surfacesContainBackedTopic(surfaces)
}

func (idx OperatingModelReferenceIndex) memberDrainerBacked(producers []OperatingActorReference, surfaces []operatingSurfaceReference, member string) bool {
	for _, producer := range producers {
		if producer.Kind == OperatingActorKindExternal && idx.graphHasRelationship(func(rel OperatingRelationship) bool {
			return rel.Kind == operatingRelExternalProducer && rel.External == producer.Value && rel.Member == member
		}) {
			return true
		}
	}
	return idx.memberSurfaceConsumerBacked(surfaces, member)
}

func (idx OperatingModelReferenceIndex) memberSurfaceConsumerBacked(surfaces []operatingSurfaceReference, member string) bool {
	for _, surface := range surfaces {
		switch surface.Kind {
		case OperatingGraphNodeKindTopic:
			if idx.graphHasRelationship(func(rel OperatingRelationship) bool {
				return rel.Kind == operatingRelTopicRead && rel.Member == member && topicsOverlap(rel.Topic, surface.Value)
			}) {
				return true
			}
		}
	}
	return false
}

func (idx OperatingModelReferenceIndex) teamConsumerBacked(surfaces []operatingSurfaceReference, team string) bool {
	if idx.graphHasNode(OperatingGraphNodeKindTeam, team) {
		return true
	}
	for _, surface := range surfaces {
		if surface.Kind != OperatingGraphNodeKindTopic {
			continue
		}
		if idx.graphHasRelationship(func(rel OperatingRelationship) bool {
			return rel.Kind == operatingRelCrossTeamOutput && rel.TargetTeam == team && topicsOverlap(rel.Topic, surface.Value)
		}) {
			return true
		}
	}
	return false
}

func (idx OperatingModelReferenceIndex) surfaceReferenceBacked(ref operatingSurfaceReference) bool {
	switch ref.Kind {
	case OperatingGraphNodeKindTopic:
		return idx.graphHasNode(OperatingGraphNodeKindTopic, ref.Value) &&
			idx.topicCatalogHasTopic(ref.Value)
	case OperatingGraphNodeKindPOR:
		return idx.graphHasNode(OperatingGraphNodeKindPOR, ref.Value)
	case OperatingGraphNodeKindMember, OperatingGraphNodeKindTeam, OperatingGraphNodeKindExternal:
		return idx.graphHasNode(ref.Kind, ref.Value)
	default:
		return false
	}
}

func (idx OperatingModelReferenceIndex) surfaceTextMentionsBackedPath(raw string) bool {
	for _, token := range extractInlineCodeTokens(raw) {
		if strings.HasPrefix(token, "path:") || strings.HasPrefix(token, "docs/") {
			if idx.graphHasNode(OperatingGraphNodeKindPOR, strings.TrimPrefix(token, "path:")) {
				return true
			}
		}
	}
	return false
}

func (idx OperatingModelReferenceIndex) surfacesContainBackedTopic(surfaces []operatingSurfaceReference) bool {
	for _, surface := range surfaces {
		if surface.Kind == OperatingGraphNodeKindTopic && idx.surfaceReferenceBacked(surface) {
			return true
		}
	}
	return false
}

func (idx OperatingModelReferenceIndex) actorExistsInGraph(actor OperatingActorReference) bool {
	switch actor.Kind {
	case OperatingActorKindMember:
		return idx.graphHasNode(OperatingGraphNodeKindMember, actor.Value)
	case OperatingActorKindTeam:
		return idx.graphHasNode(OperatingGraphNodeKindTeam, actor.Value)
	case OperatingActorKindExternal:
		return idx.graphHasNode(OperatingGraphNodeKindExternal, actor.Value)
	case OperatingActorKindProcess:
		return idx.graphHasNode(OperatingGraphNodeKindProcess, actor.Value)
	default:
		return false
	}
}

func (idx OperatingModelReferenceIndex) graphHasNode(kind OperatingGraphNodeKind, value string) bool {
	for _, block := range idx.Model.Graphs {
		for _, node := range block.Graph.Nodes {
			if node.Kind == kind && operatingModelSurfaceValuesOverlap(node.Value, value) {
				return true
			}
		}
	}
	return false
}

func (idx OperatingModelReferenceIndex) graphHasRelationship(match func(OperatingRelationship) bool) bool {
	for _, block := range idx.Model.Graphs {
		for _, rel := range BuildGraphOperatingRelationships(block) {
			if match(rel) {
				return true
			}
		}
	}
	return false
}

func (idx OperatingModelReferenceIndex) topicCatalogHasTopic(topic string) bool {
	for _, row := range idx.Model.Sections.TopicCatalog.Rows {
		if operatingModelSurfaceValuesOverlap(row.Topic, topic) {
			return true
		}
	}
	return false
}

func (idx OperatingModelReferenceIndex) collectReferences() []OperatingModelReference {
	var refs []OperatingModelReference
	seen := map[string]bool{}
	add := func(ref OperatingModelReference) {
		if ref.Kind == "" || ref.Value == "" {
			return
		}
		key := string(ref.Kind) + "\x00" + string(ref.Qualifier) + "\x00" + ref.Value + "\x00" + ref.Surface + "\x00" + ref.Raw
		if seen[key] {
			return
		}
		seen[key] = true
		refs = append(refs, ref)
	}
	for _, row := range idx.Model.Sections.TopicCatalog.Rows {
		add(OperatingModelReference{Kind: OperatingModelReferenceKindTopic, Qualifier: OperatingGraphQualifier(row.Qualifier), Value: row.Topic, Raw: row.RawTopic, Line: row.SourceLine, Surface: "topic_catalog"})
		for _, actor := range append(append([]OperatingActorReference{}, row.Writers...), row.Readers...) {
			add(operatingModelReferenceFromActor(actor, row.SourceLine, "topic_catalog"))
		}
	}
	for _, row := range idx.Model.Sections.ExternalInputs.Rows {
		for _, ref := range idx.referencesFromText(row.ProducerTrigger+" "+row.EntrySurface+" "+row.Drainer+" "+row.RoutingRule, row.SourceLine, "external_inputs") {
			add(ref)
		}
	}
	for _, row := range idx.Model.Sections.Outputs.Rows {
		for _, ref := range idx.referencesFromText(row.Output+" "+row.Surface+" "+row.Consumer+" "+row.Purpose, row.SourceLine, "outputs") {
			add(ref)
		}
	}
	for _, step := range idx.Model.Sections.FeedbackLoop.Steps {
		for _, ref := range step.References {
			add(operatingModelReferenceFromToken(ref, step.SourceLine, "feedback"))
		}
	}
	for _, item := range idx.Model.Sections.Gaps.Items {
		for _, ref := range item.References {
			add(operatingModelReferenceFromToken(ref, item.SourceLine, "gaps"))
		}
	}
	for _, command := range idx.Model.Sections.Adoption.Commands {
		add(OperatingModelReference{Kind: OperatingModelReferenceKindCommand, Value: command.Command, Raw: command.Command, Line: command.SourceLine, Surface: "adoption"})
	}
	return refs
}

func (idx OperatingModelReferenceIndex) referencesFromText(text string, line int, surface string) []OperatingModelReference {
	var refs []OperatingModelReference
	for _, token := range extractInlineCodeTokens(text) {
		if ref := operatingModelReferenceFromToken(token, line, surface); ref.Kind != "" {
			refs = append(refs, ref)
		}
	}
	for _, actor := range idx.ActorReferences(text) {
		if ref := operatingModelReferenceFromActor(actor, line, surface); ref.Kind != "" {
			refs = append(refs, ref)
		}
	}
	return refs
}

func operatingModelReferenceFromToken(token string, line int, surface string) OperatingModelReference {
	raw := strings.TrimSpace(token)
	token = parseInlineCodeToken(token)
	if kind, qualifier, value, ok := parseOperatingGraphTypedToken(token); ok {
		return OperatingModelReference{Kind: operatingModelReferenceKindFromGraphNode(kind), Qualifier: qualifier, Value: value, Raw: raw, Line: line, Surface: surface}
	}
	if actor := parseTypedOperatingActorReference(token); actor.Kind != "" {
		ref := operatingModelReferenceFromActor(actor, line, surface)
		ref.Raw = raw
		return ref
	}
	switch {
	case strings.HasPrefix(token, "path:"):
		return OperatingModelReference{Kind: OperatingModelReferenceKindPath, Value: strings.TrimPrefix(token, "path:"), Raw: raw, Line: line, Surface: surface}
	case strings.HasPrefix(token, "docs/"):
		return OperatingModelReference{Kind: OperatingModelReferenceKindPath, Value: token, Raw: raw, Line: line, Surface: surface}
	case strings.Contains(token, "/") || strings.Contains(token, "*"):
		return OperatingModelReference{Kind: OperatingModelReferenceKindTopic, Value: strings.TrimPrefix(token, "topic:"), Raw: raw, Line: line, Surface: surface}
	default:
		return OperatingModelReference{Raw: raw, Line: line, Surface: surface}
	}
}

func operatingModelReferenceFromActor(actor OperatingActorReference, line int, surface string) OperatingModelReference {
	return OperatingModelReference{
		Kind:    operatingModelReferenceKindFromActor(actor.Kind),
		Value:   actor.Value,
		Raw:     actor.Raw,
		Line:    line,
		Surface: surface,
	}
}

func operatingExternalInputRowsEqual(a, b OperatingExternalInputRow) bool {
	return a.SourceLine == b.SourceLine &&
		a.ProducerTrigger == b.ProducerTrigger &&
		a.EntrySurface == b.EntrySurface &&
		a.Drainer == b.Drainer &&
		a.RoutingRule == b.RoutingRule
}

func operatingOutputRowsEqual(a, b OperatingOutputRow) bool {
	return a.SourceLine == b.SourceLine &&
		a.Output == b.Output &&
		a.Surface == b.Surface &&
		a.Consumer == b.Consumer &&
		a.Purpose == b.Purpose
}

func operatingModelReferenceKindFromGraphNode(kind OperatingGraphNodeKind) OperatingModelReferenceKind {
	switch kind {
	case OperatingGraphNodeKindTopic:
		return OperatingModelReferenceKindTopic
	case OperatingGraphNodeKindMember:
		return OperatingModelReferenceKindMember
	case OperatingGraphNodeKindTeam:
		return OperatingModelReferenceKindTeam
	case OperatingGraphNodeKindExternal:
		return OperatingModelReferenceKindExternal
	case OperatingGraphNodeKindProcess:
		return OperatingModelReferenceKindProcess
	case OperatingGraphNodeKindPOR:
		return OperatingModelReferenceKindPOR
	default:
		return ""
	}
}

func operatingModelReferenceKindFromActor(kind OperatingActorKind) OperatingModelReferenceKind {
	switch kind {
	case OperatingActorKindMember:
		return OperatingModelReferenceKindMember
	case OperatingActorKindTeam:
		return OperatingModelReferenceKindTeam
	case OperatingActorKindExternal:
		return OperatingModelReferenceKindExternal
	case OperatingActorKindProcess:
		return OperatingModelReferenceKindProcess
	default:
		return ""
	}
}

func operatingSurfaceReferences(raw string) []operatingSurfaceReference {
	var refs []operatingSurfaceReference
	seen := map[string]bool{}
	add := func(ref operatingSurfaceReference) {
		if ref.Kind == "" || ref.Value == "" {
			return
		}
		key := string(ref.Kind) + "\x00" + string(ref.Qualifier) + "\x00" + ref.Value
		if seen[key] {
			return
		}
		seen[key] = true
		refs = append(refs, ref)
	}
	for _, token := range extractInlineCodeTokens(raw) {
		add(operatingSurfaceReferenceFromToken(token, raw))
	}
	if len(refs) == 0 {
		for _, token := range operatingSurfaceLooseTokens(raw) {
			add(operatingSurfaceReferenceFromToken(token, raw))
		}
	}
	return refs
}

func operatingSurfaceLooseTokens(raw string) []string {
	raw = strings.ReplaceAll(raw, " and ", ",")
	raw = strings.ReplaceAll(raw, " or ", ",")
	parts := strings.Split(raw, ",")
	var out []string
	for _, part := range parts {
		token := strings.TrimSpace(part)
		token = strings.TrimSuffix(token, " topics")
		token = strings.TrimSuffix(token, " topic")
		token = strings.TrimSuffix(token, " work items")
		token = strings.TrimSuffix(token, " work item")
		if token != "" {
			out = append(out, token)
		}
	}
	return out
}

func operatingSurfaceReferenceFromToken(token, context string) operatingSurfaceReference {
	token = parseInlineCodeToken(token)
	if kind, qualifier, value, ok := parseOperatingGraphTypedToken(token); ok {
		return operatingSurfaceReference{Kind: kind, Qualifier: qualifier, Value: value}
	}
	switch {
	case strings.HasPrefix(token, "docs/"):
		return operatingSurfaceReference{Kind: OperatingGraphNodeKindPOR, Value: token}
	case strings.Contains(token, "/") || strings.Contains(token, "*"):
		return operatingSurfaceReference{Kind: OperatingGraphNodeKindTopic, Value: strings.TrimPrefix(token, "topic:")}
	default:
		return operatingSurfaceReference{}
	}
}

func operatingSurfaceReferenceIsTargetState(ref operatingSurfaceReference) bool {
	return ref.Qualifier == OperatingGraphQualifierFuture || ref.Kind == OperatingGraphNodeKindFuture
}

func operatingModelRowIsTargetState(parts ...string) bool {
	normalized := strings.ToLower(strings.Join(parts, " "))
	return strings.Contains(normalized, "target-state") ||
		strings.Contains(normalized, "target state") ||
		strings.Contains(normalized, "future") ||
		strings.Contains(normalized, "not yet") ||
		strings.Contains(normalized, "until ")
}

func operatingModelSurfaceValuesOverlap(a, b string) bool {
	if a == b {
		return true
	}
	if strings.Contains(a, "/") || strings.Contains(b, "/") {
		return topicsOverlap(a, b)
	}
	return strings.TrimSpace(a) != "" && strings.TrimSpace(a) == strings.TrimSpace(b)
}

func operatingOutputConsumerTextIsDownstream(raw string) bool {
	normalized := strings.ToLower(raw)
	for _, token := range []string{"operator", "downstream", "cross-team", "implementation", "owner", "team", "member", "validator", "consumer"} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}
