package catalogcoverage

import "sort"

// Bucket is which side of the join an entry landed on.
type Bucket string

const (
	// BucketPlannedBuilt is a catalog asset with an implementation.
	BucketPlannedBuilt Bucket = "planned-built"
	// BucketPlannedUnbuilt is a catalog asset with no implementation for a
	// declared target. The normal state for most of the catalog.
	BucketPlannedUnbuilt Bucket = "planned-unbuilt"
	// BucketSupplemental is an implementation with no catalog entry. Legitimate
	// — the library may exceed the target list — and worth counting separately
	// so it is a deliberate choice rather than an accident.
	BucketSupplemental Bucket = "supplemental"
)

// Row is one joined entry.
type Row struct {
	AssetID  string
	Name     string
	Domain   string
	Kind     string
	Priority string
	Bucket   Bucket
	// Implementation is the on-disk directory name when one exists.
	Implementation string
	// BlocksDownstream is how many other catalog assets transitively require
	// this one. It is the work-ordering signal: an unbuilt asset blocking forty
	// others is worth more than one blocking none.
	BlocksDownstream int
}

// Report is the joined view.
type Report struct {
	Rows []Row
	// Totals count rows per bucket.
	Totals map[Bucket]int
	// ByDomain and ByPriority count only planned assets, since supplemental
	// entries have no declared domain or priority to roll up under.
	ByDomain   map[string]DomainCount
	ByPriority map[string]DomainCount
}

// DomainCount is a planned/built pair for one grouping.
type DomainCount struct {
	Planned int
	Built   int
}

// Compute joins catalog assets against implementations by catalogId.
func Compute(assets []Asset, impls []Implementation) Report {
	rep := Report{
		Totals:     map[Bucket]int{},
		ByDomain:   map[string]DomainCount{},
		ByPriority: map[string]DomainCount{},
	}
	byCatalogID := map[string]Implementation{}
	claimed := map[string]bool{}
	for _, impl := range impls {
		if impl.CatalogID != "" {
			byCatalogID[impl.CatalogID] = impl
		}
	}
	downstream := blockedCounts(assets)

	for _, asset := range assets {
		row := Row{
			AssetID: asset.ID, Name: asset.Name, Domain: asset.Domain,
			Kind: asset.Kind, Priority: asset.Priority,
			BlocksDownstream: downstream[asset.ID],
		}
		if impl, ok := byCatalogID[asset.ID]; ok {
			row.Bucket = BucketPlannedBuilt
			row.Implementation = impl.Name
			claimed[impl.Name] = true
		} else {
			row.Bucket = BucketPlannedUnbuilt
		}
		rep.Rows = append(rep.Rows, row)
		rep.Totals[row.Bucket]++

		d := rep.ByDomain[asset.Domain]
		d.Planned++
		if row.Bucket == BucketPlannedBuilt {
			d.Built++
		}
		rep.ByDomain[asset.Domain] = d

		p := rep.ByPriority[asset.Priority]
		p.Planned++
		if row.Bucket == BucketPlannedBuilt {
			p.Built++
		}
		rep.ByPriority[asset.Priority] = p
	}

	for _, impl := range impls {
		if impl.CatalogID != "" && claimed[impl.Name] {
			continue
		}
		if impl.CatalogID != "" {
			// Points at a catalog id that does not exist. A dangling reference
			// is a real defect rather than a supplemental asset, so it is
			// surfaced with its intended id still attached.
			rep.Rows = append(rep.Rows, Row{
				AssetID: impl.CatalogID, Name: impl.Name, Bucket: BucketSupplemental,
				Implementation: impl.Name, Domain: "(dangling catalogId)",
			})
			rep.Totals[BucketSupplemental]++
			continue
		}
		rep.Rows = append(rep.Rows, Row{
			Name: impl.Name, Bucket: BucketSupplemental,
			Implementation: impl.Name, Domain: "(" + impl.Root + ")",
		})
		rep.Totals[BucketSupplemental]++
	}
	return rep
}

// NextWork returns unbuilt assets ordered by leverage: how many other assets
// they block, then priority, then id. This is what makes an agent loop
// deterministic — there is always exactly one correct next target.
func NextWork(rep Report, limit int) []Row {
	rank := map[string]int{"P0": 0, "P1": 1, "P2": 2}
	var out []Row
	for _, row := range rep.Rows {
		if row.Bucket == BucketPlannedUnbuilt {
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].BlocksDownstream != out[j].BlocksDownstream {
			return out[i].BlocksDownstream > out[j].BlocksDownstream
		}
		ri, rj := rank[out[i].Priority], rank[out[j].Priority]
		if ri != rj {
			return ri < rj
		}
		return out[i].AssetID < out[j].AssetID
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// blockedCounts computes, for each asset, how many distinct other assets
// transitively require it. Only `requires` edges count: a `suggests` edge is
// never copied and therefore never blocks anything.
func blockedCounts(assets []Asset) map[string]int {
	requires := make(map[string][]string, len(assets))
	for _, a := range assets {
		requires[a.ID] = a.Requires
	}
	counts := map[string]int{}
	for _, a := range assets {
		seen := map[string]bool{}
		var walk func(string)
		walk = func(id string) {
			for _, dep := range requires[id] {
				if seen[dep] {
					continue
				}
				seen[dep] = true
				walk(dep)
			}
		}
		walk(a.ID)
		for dep := range seen {
			counts[dep]++
		}
	}
	return counts
}
