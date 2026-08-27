package utilityclass

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmitsAnyRejectsFalsePositiveShapes(t *testing.T) {
	source := `
// className="md:inset-x-8"
const styles = ` + "`" + `[data-card] .bg-app-danger { display: grid; }` + "`" + `
export const Card = () => <div data-class="touch-target" />
`
	require.Empty(t, EmitsAny(source))
}

func TestEmitsAnyFindsClassBearingLiteralsAndVariables(t *testing.T) {
	source := `
const panel = cn("md:inset-x-8", condition && "bg-wc-backdrop", "touch-target")
export const Card = () => <div className={panel + " focus-visible:ring-app-primary/50"} />
`
	hits := EmitsAny(source)
	require.Equal(t, []string{"bg-wc-backdrop", "focus-visible:ring-app-primary/50", "md:inset-x-8", "touch-target"}, sortedClasses(hits))
	require.Equal(t, "variant", hits[0].Category)
}

func TestUndefinedInConsumerReportsOnlyMissingClasses(t *testing.T) {
	hits := UndefinedInConsumer(`<div className="flex gap-2" />`, map[string]struct{}{"flex": {}})
	require.Equal(t, []string{"gap-2"}, sortedClasses(hits))
}

func sortedClasses(hits []Hit) []string {
	classes := make([]string, 0, len(hits))
	for _, hit := range hits {
		classes = append(classes, hit.Class)
	}
	sort.Strings(classes)
	return classes
}
