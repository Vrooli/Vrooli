package discovery

import "context"

// CompositeScanner fans a single TargetSourceScanner seam out over several
// concrete scanners, concatenating their candidates. Downstream dedup in the
// service (by owner+name and by locator) makes plain concatenation safe even if
// two scanners surface the same path. The first scanner error aborts the scan,
// so a real failure is never silently hidden.
type CompositeScanner struct {
	scanners []TargetSourceScanner
}

// NewCompositeScanner builds a composite over the given scanners, in order.
func NewCompositeScanner(scanners ...TargetSourceScanner) *CompositeScanner {
	return &CompositeScanner{scanners: scanners}
}

// Compile-time guarantee.
var _ TargetSourceScanner = (*CompositeScanner)(nil)

func (c *CompositeScanner) Scan(ctx context.Context) ([]TargetCandidate, error) {
	var out []TargetCandidate
	for _, s := range c.scanners {
		if s == nil {
			continue
		}
		got, err := s.Scan(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, got...)
	}
	return out, nil
}
