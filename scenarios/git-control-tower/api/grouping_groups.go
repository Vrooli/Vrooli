package main

import (
	"sort"
	"strings"
)

const (
	groupSourceManual   = "manual"
	groupSourceContract = "contract"
	groupSourceBuiltin  = "builtin"
)

// ChangeGroup is one resolved bucket in the repository change list.
type ChangeGroup struct {
	Key    string   `json:"key"`
	Kind   string   `json:"kind,omitempty"`
	ID     string   `json:"id,omitempty"`
	Label  string   `json:"label"`
	Root   string   `json:"root,omitempty"`
	Source string   `json:"source"`
	Files  []string `json:"files"`
}

// RepoGroupsResponse is the read-only response for the repository groups endpoint.
type RepoGroupsResponse struct {
	Groups []ChangeGroup `json:"groups"`
}

type resolvedChangeGroup struct {
	group      ChangeGroup
	order      int
	sourceRank int
}

// ResolveChangeGroups resolves manual rules first, then contract targets, then
// the builtin Other group. Manual rule order is precedence order.
func ResolveChangeGroups(repoDir string, files RepoFilesStatus, config GroupingRulesConfig) []ChangeGroup {
	index := targetIndexForRepo(repoDir)
	groups := map[string]*resolvedChangeGroup{}
	for _, path := range changedPaths(files) {
		if rule, prefix, segment, ruleIndex, ok := resolveManualGroup(path, config.Rules); ok {
			key := rule.ID
			label := rule.Label
			root := prefix
			if segment != "" {
				key += ":" + segment
				label = segment
				root = prefix + segment + "/"
			}
			addResolvedGroup(groups, resolvedChangeGroup{
				group: ChangeGroup{
					Key: key, Label: label, Root: root, Source: groupSourceManual,
				},
				order:      ruleIndex,
				sourceRank: 0,
			}, path)
			continue
		}
		if index != nil {
			if target, ok := index.Lookup(path); ok {
				addResolvedGroup(groups, resolvedChangeGroup{
					group: ChangeGroup{
						Key:    "contract:" + string(target.Kind) + ":" + target.ID,
						Kind:   string(target.Kind),
						ID:     target.ID,
						Label:  target.ID,
						Root:   target.Root,
						Source: groupSourceContract,
					},
					order:      0,
					sourceRank: 1,
				}, path)
				continue
			}
		}
		addResolvedGroup(groups, resolvedChangeGroup{
			group: ChangeGroup{Key: "other", Label: "Other", Source: groupSourceBuiltin},
			order: 0, sourceRank: 2,
		}, path)
	}

	resolved := make([]resolvedChangeGroup, 0, len(groups))
	for _, group := range groups {
		resolved = append(resolved, *group)
	}
	sort.SliceStable(resolved, func(i, j int) bool {
		if resolved[i].sourceRank != resolved[j].sourceRank {
			return resolved[i].sourceRank < resolved[j].sourceRank
		}
		if resolved[i].order != resolved[j].order {
			return resolved[i].order < resolved[j].order
		}
		return resolved[i].group.Label < resolved[j].group.Label
	})

	result := make([]ChangeGroup, 0, len(resolved))
	for _, group := range resolved {
		result = append(result, group.group)
	}
	return result
}

func changedPaths(files RepoFilesStatus) []string {
	paths := make([]string, 0, len(files.Conflicts)+len(files.Staged)+len(files.Unstaged)+len(files.Untracked))
	seen := make(map[string]struct{})
	for _, category := range [][]string{files.Conflicts, files.Staged, files.Unstaged, files.Untracked} {
		for _, path := range category {
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
		}
	}
	return paths
}

func resolveManualGroup(path string, rules []GroupingRule) (GroupingRule, string, string, int, bool) {
	for ruleIndex, rule := range rules {
		for _, rawPrefix := range rule.Prefixes {
			prefix := normalizeGroupingPrefix(rawPrefix)
			if prefix == "" || !strings.HasPrefix(path, prefix) {
				continue
			}
			if rule.Mode != "segment" {
				return rule, prefix, "", ruleIndex, true
			}
			rest := strings.TrimPrefix(path, prefix)
			segment := strings.SplitN(rest, "/", 2)[0]
			return rule, prefix, segment, ruleIndex, true
		}
	}
	return GroupingRule{}, "", "", 0, false
}

func normalizeGroupingPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return ""
	}
	if prefix == "/" || strings.HasSuffix(prefix, "/") {
		return prefix
	}
	return prefix + "/"
}

func addResolvedGroup(groups map[string]*resolvedChangeGroup, group resolvedChangeGroup, path string) {
	existing, ok := groups[group.group.Key]
	if !ok {
		groups[group.group.Key] = &group
		existing = &group
	}
	for _, existingPath := range existing.group.Files {
		if existingPath == path {
			return
		}
	}
	existing.group.Files = append(existing.group.Files, path)
}
