package notes

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// CensusOwnerAlias returns the complete durable Notes generation inventory
// for one explicit owner/alias. It reports stored recipe and lifecycle facts
// only; it does not resolve an embedding policy, inspect Qdrant, or promote a
// generation. Callers must keep an empty result distinct from a live vector
// inventory because no physical-store probe is part of this contract.
func (c FleetCatalog) CensusOwnerAlias(ctx context.Context, ownerID, alias string) ([]CatalogIdentity, error) {
	ownerID, alias = strings.TrimSpace(ownerID), strings.TrimSpace(alias)
	if c.DB == nil || ownerID == "" || alias == "" {
		return nil, errors.New("Notes generation census requires its owner database, owner ID and alias")
	}
	rows, err := c.DB.QueryContext(ctx, `SELECT arm_id, owner_id, alias, recipe_id, snapshot_id, snapshot_watermark, catchup_watermark, dimensions, collection, COALESCE(rollback_to,''), state, cursor, version, protected, updated_at
FROM notes_fleet_generations WHERE owner_id=$1 AND alias=$2 ORDER BY updated_at DESC, arm_id`, ownerID, alias)
	if err != nil {
		return nil, fmt.Errorf("list Notes generations for census: %w", err)
	}
	defer rows.Close()
	items := make([]CatalogIdentity, 0)
	active := 0
	for rows.Next() {
		var item CatalogIdentity
		if err := rows.Scan(&item.ArmID, &item.OwnerID, &item.Alias, &item.RecipeID, &item.SnapshotID, &item.SnapshotWatermark, &item.CatchupWatermark, &item.Dimensions, &item.Collection, &item.RollbackTo, &item.State, &item.Cursor, &item.Version, &item.Protected, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("read Notes generation census row: %w", err)
		}
		if item.OwnerID != ownerID || item.Alias != alias || strings.TrimSpace(item.ArmID) == "" || strings.TrimSpace(item.Collection) == "" || strings.TrimSpace(item.RecipeID) == "" || item.Dimensions <= 0 {
			return nil, errors.New("Notes generation census found incomplete or mis-scoped catalog identity")
		}
		if item.State == "active" {
			active++
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Notes generation census: %w", err)
	}
	if active > 1 {
		return nil, errors.New("Notes generation census found multiple active generations for one owner alias")
	}
	return items, nil
}

// CensusAllGenerations reads only the durable Notes-owned generation catalog.
// It does not infer owner/alias values from defaults or inspect Qdrant. Every
// returned recipe ID and collection is exactly what Notes has recorded.
func (c FleetCatalog) CensusAllGenerations(ctx context.Context) ([]CatalogIdentity, error) {
	if c.DB == nil {
		return nil, errors.New("Notes generation census requires its owner database")
	}
	rows, err := c.DB.QueryContext(ctx, `SELECT arm_id, owner_id, alias, recipe_id, snapshot_id, snapshot_watermark, catchup_watermark, dimensions, collection, COALESCE(rollback_to,''), state, cursor, version, protected, updated_at
FROM notes_fleet_generations ORDER BY owner_id, alias, updated_at DESC, arm_id`)
	if err != nil {
		return nil, fmt.Errorf("list all Notes generations for census: %w", err)
	}
	defer rows.Close()
	items := make([]CatalogIdentity, 0)
	activeByAlias := make(map[string]int)
	for rows.Next() {
		var item CatalogIdentity
		if err := rows.Scan(&item.ArmID, &item.OwnerID, &item.Alias, &item.RecipeID, &item.SnapshotID, &item.SnapshotWatermark, &item.CatchupWatermark, &item.Dimensions, &item.Collection, &item.RollbackTo, &item.State, &item.Cursor, &item.Version, &item.Protected, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("read Notes generation census row: %w", err)
		}
		if strings.TrimSpace(item.ArmID) == "" || strings.TrimSpace(item.OwnerID) == "" || strings.TrimSpace(item.Alias) == "" || strings.TrimSpace(item.Collection) == "" || strings.TrimSpace(item.RecipeID) == "" || item.Dimensions <= 0 {
			return nil, errors.New("Notes generation census found incomplete catalog identity")
		}
		if item.State == "active" {
			activeByAlias[item.OwnerID+"\x00"+item.Alias]++
			if activeByAlias[item.OwnerID+"\x00"+item.Alias] > 1 {
				return nil, fmt.Errorf("Notes generation census found multiple active generations for owner %q alias %q", item.OwnerID, item.Alias)
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Notes generation census: %w", err)
	}
	return items, nil
}
