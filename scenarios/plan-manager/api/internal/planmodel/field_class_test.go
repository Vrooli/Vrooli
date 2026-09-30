package planmodel

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlanFieldClassesCoverPlan(t *testing.T) {
	typ := reflect.TypeOf(Plan{})
	seen := make(map[string]struct{}, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		class, ok := PlanFieldClasses[field.Name]
		require.Truef(t, ok, "Plan.%s is not classified", field.Name)
		require.Contains(t, []FieldClass{FieldClassAuthored, FieldClassIdentity, FieldClassComputed, FieldClassGovernance, FieldClassGraph}, class, "Plan.%s has invalid class", field.Name)
		seen[field.Name] = struct{}{}
	}
	for name := range PlanFieldClasses {
		_, ok := seen[name]
		require.Truef(t, ok, "classification %q has no Plan field", name)
	}
}
