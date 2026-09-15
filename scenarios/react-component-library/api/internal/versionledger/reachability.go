package versionledger

import (
	"context"
	"fmt"
	"strings"
)

// BuildReachability constructs the only graph used by retention. It unions
// source imports, generated locks, mirrored locks, adoption edges, and then
// computes a fixpoint from every live latest version. External exact pins are
// already represented by the workbench/adoption scanners; callers do not get
// to select a weaker edge source.
func (r *Repository) BuildReachability(ctx context.Context) (Reachability, error) {
	source, unreadable, err := r.sourceReferencesDetailed(ctx)
	if err != nil {
		return Reachability{}, fmt.Errorf("build source references: %w", err)
	}
	locks, err := r.lockReferences(ctx)
	if err != nil {
		return Reachability{}, fmt.Errorf("build lock references: %w", err)
	}
	refs := make(map[string][]VersionReference, len(source)+len(locks))
	for key, values := range source {
		refs[key] = appendUniqueReferences(refs[key], values)
	}
	for key, values := range locks {
		refs[key] = appendUniqueReferences(refs[key], values)
	}

	roots := map[string]struct{}{}
	if r.db != nil {
		rows, queryErr := r.db.QueryContext(ctx, `SELECT library_id, latest_version FROM components WHERE trim(latest_version) <> ''`)
		if queryErr == nil {
			defer rows.Close()
			for rows.Next() {
				var libraryID, version string
				if err := rows.Scan(&libraryID, &version); err != nil {
					return Reachability{}, err
				}
				roots[sourceReferenceKey(libraryID, version)] = struct{}{}
			}
			if err := rows.Err(); err != nil {
				return Reachability{}, err
			}
		} else if !strings.Contains(strings.ToLower(queryErr.Error()), "no such table") {
			return Reachability{}, queryErr
		}
	}
	for _, item := range unreadable {
		roots[sourceReferenceKey(item.LibraryID, item.Version)] = struct{}{}
	}
	// A direct adoption is an external consumer root even when the adoption
	// is package-linked and therefore has no adoption_files row to scan. Its
	// version lock still defines the transitive closure that must remain
	// available to that consumer.
	if r.db != nil {
		rows, queryErr := r.db.QueryContext(ctx, `
			SELECT library_id, adopted_version
			FROM adoption_records
			WHERE trim(COALESCE(library_id, '')) <> ''
			  AND trim(COALESCE(adopted_version, '')) <> ''
			  AND lower(COALESCE(mode, 'copied')) <> 'ejected'`)
		if queryErr == nil {
			defer rows.Close()
			for rows.Next() {
				var libraryID, version string
				if err := rows.Scan(&libraryID, &version); err != nil {
					return Reachability{}, err
				}
				roots[sourceReferenceKey(libraryID, version)] = struct{}{}
			}
			if err := rows.Err(); err != nil {
				return Reachability{}, err
			}
		} else if !strings.Contains(strings.ToLower(queryErr.Error()), "no such table") && !strings.Contains(strings.ToLower(queryErr.Error()), "no such column") {
			return Reachability{}, queryErr
		}
	}
	// References without an owning library version come from external
	// consumers (for example the workbench). They are exact-pin roots.
	for target, values := range refs {
		for _, ref := range values {
			if strings.TrimSpace(ref.OwnerLibraryID) == "" && strings.TrimSpace(ref.OwnerVersion) == "" {
				roots[target] = struct{}{}
				break
			}
		}
	}
	reachable := make(map[string]struct{}, len(roots))
	queue := make([]string, 0, len(roots))
	for root := range roots {
		reachable[root] = struct{}{}
		queue = append(queue, root)
	}
	for len(queue) > 0 {
		owner := queue[0]
		queue = queue[1:]
		for target, values := range refs {
			for _, ref := range values {
				if sourceReferenceKey(ref.OwnerLibraryID, ref.OwnerVersion) == owner {
					if _, ok := reachable[target]; !ok {
						reachable[target] = struct{}{}
						queue = append(queue, target)
					}
					break
				}
			}
		}
	}
	return Reachability{References: refs, Reachable: reachable, Unreadable: unreadable}, nil
}

func (r Reachability) IsReachable(libraryID, version string) bool {
	_, ok := r.Reachable[sourceReferenceKey(libraryID, version)]
	return ok
}
