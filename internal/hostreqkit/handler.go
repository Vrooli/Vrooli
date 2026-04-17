package hostreqkit

import "github.com/vrooli/vrooli/internal/hostreqspec"

type Handler interface {
	Name() string
	Kind() hostreqspec.Kind
	Inspect(host Host, requirement hostreqspec.ResolvedRequirement) ItemStatus
	Apply(host Host, status ItemStatus, opts EnsureOptions) (ItemStatus, error)
}
