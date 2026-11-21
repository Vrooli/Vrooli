package composition

import "sort"

func (p *Pool) List(styleID string) []Take {
	return p.ListFiltered(styleID, "")
}

func (p *Pool) ListFiltered(styleID, jobID string) []Take {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Take, 0, len(p.takes))
	for _, take := range p.takes {
		if (styleID == "" || take.StyleID == styleID) && (jobID == "" || take.JobID == jobID) {
			out = append(out, take)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (p *Pool) Get(id string) (Take, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	take, ok := p.takes[id]
	return take, ok
}
