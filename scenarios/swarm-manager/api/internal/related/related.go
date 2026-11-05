// Package related builds explainable related-work reports. Providers are kept
// behind a small interface so cross-scenario federation can join later without
// changing consumers of the report.
package related

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/depgraph"
	"swarm-manager/internal/goals"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/records"
)

type TargetKind string

const (
	TargetBacklog TargetKind = "backlog"
	TargetGoal    TargetKind = "goal"
)

type EntityKind string

const (
	EntityBacklog EntityKind = "backlog"
	EntityGoal    EntityKind = "goal"
	EntityRecord  EntityKind = "record"
)

type GroupName string

const (
	GroupLinked    GroupName = "linked"
	GroupSameScope GroupName = "same_scope"
	GroupSimilar   GroupName = "similar"
)

type TargetRef struct {
	Kind        TargetKind
	BacklogKind backlog.BacklogKind
	Name        string
}

func (t TargetRef) Key() string {
	if t.Kind == TargetBacklog {
		return string(t.BacklogKind) + "/" + t.Name
	}
	return t.Name
}

type Entity struct {
	Kind               EntityKind
	Key, Title, Status string
	Archived           bool
	Reasons            []string
	ScorePercent       int
}
type Group struct {
	Name     GroupName
	Entities []Entity
	Degraded bool
}
type Report struct {
	Linked    Group
	SameScope Group
	Similar   Group
}
type Provider interface {
	Group() GroupName
	Compute(context.Context, TargetRef) ([]Entity, bool, error)
}

type BacklogReader interface {
	LoadAll([]backlog.BacklogKind) ([]backlog.BacklogItem, error)
	LoadItem(backlog.BacklogKind, string) (backlog.BacklogItem, error)
}
type GoalReader interface {
	List() ([]goals.GoalWithScope, error)
}
type Similarity interface {
	Similar(context.Context, TargetRef, int) ([]Entity, bool, error)
}

type Engine struct {
	backlog BacklogReader
	goals   GoalReader
	records records.Store
	similar Similarity
}

func NewEngine(b BacklogReader, g GoalReader, r records.Store, s Similarity) *Engine {
	return &Engine{b, g, r, s}
}

func (e *Engine) Compute(ctx context.Context, target TargetRef, limit int) (Report, error) {
	linked, err := e.linked(ctx, target)
	if err != nil {
		return Report{}, err
	}
	scope, err := e.scope(target)
	if err != nil {
		return Report{}, err
	}
	var similar []Entity
	degraded := false
	if e.similar != nil {
		similar, degraded, err = e.similar.Similar(ctx, target, limit)
		if err != nil {
			degraded = true
			similar = nil
		}
	}
	e.hydrateRecordTitles(similar)
	linked, scope, similar = dedupe(linked, scope, similar)
	return Report{Group{GroupLinked, limitEntities(linked, limit), false}, Group{GroupSameScope, limitEntities(scope, limit), false}, Group{GroupSimilar, limitEntities(similar, limit), degraded}}, nil
}

// hydrateRecordTitles makes semantic results readable even when their stored
// vector payload predates the display-title field. Records are canonical in
// the file store, so use their trigger rather than exposing an opaque ID.
func (e *Engine) hydrateRecordTitles(entities []Entity) {
	for index := range entities {
		entity := &entities[index]
		if entity.Kind != EntityRecord {
			continue
		}
		record, err := e.records.Get(entity.Key)
		if err != nil || strings.TrimSpace(record.Trigger) == "" {
			continue
		}
		entity.Title = record.Trigger
		if entity.Status == "" {
			entity.Status = string(record.Outcome)
		}
	}
}

func limitEntities(in []Entity, limit int) []Entity {
	if limit > 0 && len(in) > limit {
		return in[:limit]
	}
	return in
}

func (e *Engine) allItems() ([]backlog.BacklogItem, error) { return e.backlog.LoadAll(nil) }
func itemMap(items []backlog.BacklogItem) map[string]backlog.BacklogItem {
	out := map[string]backlog.BacklogItem{}
	for _, i := range items {
		out[string(i.Kind)+"/"+i.Name] = i
	}
	return out
}

func backlogEntity(i backlog.BacklogItem, reasons ...string) Entity {
	return Entity{EntityBacklog, string(i.Kind) + "/" + i.Name, i.Title, string(i.Status), backlog.IsArchived(i), reasons, 0}
}

func goalEntity(g goals.GoalWithScope, reasons ...string) Entity {
	return Entity{EntityGoal, g.Goal.Name, g.Goal.Title, g.Goal.Status, g.Goal.ArchivedAt != nil && strings.TrimSpace(*g.Goal.ArchivedAt) != "", reasons, 0}
}

func recordEntity(r records.Record, reasons ...string) Entity {
	return Entity{EntityRecord, r.ID, r.Trigger, string(r.Outcome), true, reasons, 0}
}

func (e *Engine) linked(_ context.Context, target TargetRef) ([]Entity, error) {
	items, err := e.allItems()
	if err != nil {
		return nil, err
	}
	goalList, err := e.goals.List()
	if err != nil {
		return nil, err
	}
	byKey := itemMap(items)
	out := []Entity{}
	if target.Kind == TargetBacklog {
		item, ok := byKey[target.Key()]
		if !ok {
			return nil, fmt.Errorf("backlog target %q not found", target.Key())
		}
		g := depgraph.New()
		for _, x := range items {
			g.AddNode(string(x.Kind)+"/"+x.Name, x.DependsOn)
		}
		for _, key := range item.DependsOn {
			if x, ok := byKey[key]; ok {
				out = append(out, backlogEntity(x, "depends on this item"))
			}
		}
		for _, key := range g.Dependents(target.Key()) {
			if x, ok := byKey[key]; ok {
				out = append(out, backlogEntity(x, "depends on this item"))
			}
		}
		for _, goal := range goalList {
			if containsRef(goal.Scope.Closure, target.Key()) {
				for _, key := range goal.Scope.Closure {
					if key == target.Key() {
						continue
					}
					if byItem, ok := byKey[key]; ok {
						out = append(out, backlogEntity(byItem, "shares goal: "+goal.Goal.Name))
					}
				}
			}
		}
		rs, err := e.records.List(records.ListFilter{BacklogRef: target.Key(), IncludeStubs: true})
		if err != nil {
			return nil, err
		}
		for _, r := range rs {
			out = append(out, recordEntity(r, "record for this item"))
		}
	} else {
		goal, ok := findGoal(goalList, target.Name)
		if !ok {
			return nil, fmt.Errorf("goal target %q not found", target.Name)
		}
		for _, key := range goal.Scope.Closure {
			if x, ok := byKey[key]; ok {
				out = append(out, backlogEntity(x, "in this goal's derived scope"))
			}
		}
		rs, err := e.records.List(records.ListFilter{MilestoneID: target.Name, IncludeStubs: true})
		if err != nil {
			return nil, err
		}
		for _, r := range rs {
			out = append(out, recordEntity(r, "record for this goal"))
		}
	}
	return merge(out), nil
}

func findGoal(in []goals.GoalWithScope, name string) (goals.GoalWithScope, bool) {
	for _, g := range in {
		if g.Goal.Name == name {
			return g, true
		}
	}
	return goals.GoalWithScope{}, false
}

func containsRef(refs []string, ref string) bool {
	for _, candidate := range refs {
		if candidate == ref {
			return true
		}
	}
	return false
}

func (e *Engine) scope(target TargetRef) ([]Entity, error) {
	items, err := e.allItems()
	if err != nil {
		return nil, err
	}
	goalList, err := e.goals.List()
	if err != nil {
		return nil, err
	}
	targetScopes := scopesForTarget(target, items, goalList)
	out := []Entity{}
	for _, x := range items {
		if target.Kind == TargetBacklog && target.Key() == string(x.Kind)+"/"+x.Name {
			continue
		}
		common := intersection(targetScopes, scopesForItem(x))
		if len(common) > 0 {
			reasons := scopeReasons(common, targetScopes.globs, append(x.AcceptanceAllow, x.Creates...))
			out = append(out, backlogEntity(x, reasons...))
		}
	}
	for _, x := range goalList {
		if target.Kind == TargetGoal && target.Name == x.Goal.Name {
			continue
		}
		common := intersection(targetScopes, scopesForGoal(x, items))
		if len(common) > 0 {
			out = append(out, goalEntity(x, scopeReasons(common, targetScopes.globs, nil)...))
		}
	}
	return merge(out), nil
}

type scopeSet struct {
	slugs map[string]bool
	globs []string
}

func scopesForItem(i backlog.BacklogItem) scopeSet {
	g := append(append([]string{}, i.AcceptanceAllow...), i.Creates...)
	return scopeSet{set(pathutil.ScenariosFromGlobs(g)), g}
}

func scopesForGoal(goal goals.GoalWithScope, items []backlog.BacklogItem) scopeSet {
	o := scopeSet{set(mapKeys(nil)), nil}
	for _, key := range goal.Scope.Closure {
		for _, i := range items {
			if string(i.Kind)+"/"+i.Name == key {
				s := scopesForItem(i)
				for k := range s.slugs {
					o.slugs[k] = true
				}
				o.globs = append(o.globs, s.globs...)
			}
		}
	}
	return o
}

func scopesForTarget(t TargetRef, items []backlog.BacklogItem, goalList []goals.GoalWithScope) scopeSet {
	if t.Kind == TargetGoal {
		if goal, ok := findGoal(goalList, t.Name); ok {
			return scopesForGoal(goal, items)
		}
	}
	for _, i := range items {
		if string(i.Kind)+"/"+i.Name == t.Key() {
			return scopesForItem(i)
		}
	}
	return scopeSet{map[string]bool{}, nil}
}

func set(in []string) map[string]bool {
	o := map[string]bool{}
	for _, x := range in {
		o[x] = true
	}
	return o
}
func mapKeys(_ map[string]bool) []string { return nil }
func intersection(a, b scopeSet) []string {
	var out []string
	for x := range a.slugs {
		if b.slugs[x] {
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}

func scopeReasons(common, left, right []string) []string {
	out := []string{}
	for _, s := range common {
		out = append(out, "shares scope: scenarios/"+s)
		p := commonPrefix(left, right, s)
		if p != "" {
			out = append(out, "shares scope: "+p)
		}
	}
	return unique(out)
}

func commonPrefix(a, b []string, scenario string) string {
	best := ""
	for _, x := range a {
		for _, y := range b {
			if !strings.HasPrefix(x, "scenarios/"+scenario+"/") || !strings.HasPrefix(y, "scenarios/"+scenario+"/") {
				continue
			}
			p := strings.TrimSuffix(x, "/**")
			q := strings.TrimSuffix(y, "/**")
			for p != "." && !strings.HasPrefix(q, p) {
				p = path.Dir(p)
			}
			if len(p) > len(best) && p != "." {
				best = p
			}
		}
	}
	return best
}

func merge(in []Entity) []Entity {
	m := map[string]Entity{}
	for _, x := range in {
		k := string(x.Kind) + ":" + x.Key
		if old, ok := m[k]; ok {
			old.Reasons = unique(append(old.Reasons, x.Reasons...))
			m[k] = old
		} else {
			m[k] = x
		}
	}
	out := make([]Entity, 0, len(m))
	for _, x := range m {
		out = append(out, x)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

func unique(in []string) []string {
	m := map[string]bool{}
	var o []string
	for _, x := range in {
		if x != "" && !m[x] {
			m[x] = true
			o = append(o, x)
		}
	}
	return o
}

func dedupe(groups ...[]Entity) ([]Entity, []Entity, []Entity) {
	type position struct{ group, index int }
	seen := map[string]position{}
	outs := make([][]Entity, len(groups))
	for i, g := range groups {
		for _, x := range g {
			k := string(x.Kind) + ":" + x.Key
			if prior, ok := seen[k]; ok {
				entity := &outs[prior.group][prior.index]
				entity.Reasons = unique(append(entity.Reasons, x.Reasons...))
				continue
			}
			outs[i] = append(outs[i], x)
			seen[k] = position{i, len(outs[i]) - 1}
		}
	}
	return outs[0], outs[1], outs[2]
}
