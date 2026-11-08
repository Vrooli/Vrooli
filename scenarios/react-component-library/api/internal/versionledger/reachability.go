package versionledger

import (
	"context"
	"fmt"
	"strings"
)

// BuildReachability constructs the only graph used by retention. It unions
// source imports, generated locks, mirrored locks, consumer imports and
// adoption edges, and then computes a fixpoint from its roots:
//
//   - every latest version and retained draft;
//   - every direct adoption;
//   - every pin held by something that is not a cold catalog version: an
//     external consumer (scenario or template source, the workbench, a
//     harness lock), an owner the index has not recorded, or a release that
//     is still materialized.
//
// The last rule keeps the materialized set closed under its own pins. A
// release on disk can be imported, previewed and type-checked, so evicting
// what it pins breaks it whether or not anything reaches the release itself.
// Historical chains therefore leave one link per pass, owner first: once the
// owner is evicted its pins live only in the cold mirror and become ordinary
// edges, honoured only while that owner is reachable. Callers do not get to
// select a weaker edge source.
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
		rows, queryErr := r.db.QueryContext(ctx, `
			SELECT library_id, latest_version
			FROM components
			WHERE trim(latest_version) <> ''
			UNION
			SELECT c.library_id, v.version
			FROM components c
			JOIN component_versions v ON v.component_id = c.id
			WHERE lower(v.status) <> 'retired'
			  AND (v.version = c.draft_version OR lower(v.status) LIKE '%draft%')`)
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
	cold, err := r.coldOwners(ctx)
	if err != nil {
		return Reachability{}, err
	}
	for target, values := range refs {
		for _, ref := range values {
			if !cold[sourceReferenceKey(strings.TrimSpace(ref.OwnerLibraryID), strings.TrimSpace(ref.OwnerVersion))] {
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

// coldOwners lists the indexed catalog versions whose pins are ordinary graph
// edges rather than roots: evicted, non-retired versions, whose bytes live
// only in the mirror. Every other owner is either retained on disk, unknown to
// the index, or external, and retention cannot prove that any of those will
// leave with this plan. A store without the catalog tables has no cold
// versions, which keeps every pin a root.
func (r *Repository) coldOwners(ctx context.Context) (map[string]bool, error) {
	cold := map[string]bool{}
	if r.db == nil {
		return cold, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.library_id, v.version
		FROM components c
		JOIN component_versions v ON v.component_id = c.id
		WHERE lower(COALESCE(v.presence, 'materialized')) = 'evicted'
		  AND lower(v.status) <> 'retired'`)
	if err != nil {
		lowered := strings.ToLower(err.Error())
		if strings.Contains(lowered, "no such table") || strings.Contains(lowered, "no such column") {
			return cold, nil
		}
		return nil, fmt.Errorf("read cold catalog versions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var libraryID, version string
		if err := rows.Scan(&libraryID, &version); err != nil {
			return nil, err
		}
		cold[sourceReferenceKey(libraryID, version)] = true
	}
	return cold, rows.Err()
}
