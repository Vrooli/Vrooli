package format

import (
	"fmt"
	"io"
)

func FormatComparisonContext(w io.Writer, analysis ValidationQualityAnalysis, score float64) {
	fmt.Fprintln(w, sectionSep)
	fmt.Fprintln(w)
	switch {
	case analysis.TotalPenalty > 50:
		fmt.Fprintln(w, "🎓 Study browser-automation-studio as reference for proper test structure:")
		fmt.Fprintln(w, "   • Has API tests: api/**/*_test.go")
		fmt.Fprintln(w, "   • Has UI tests: ui/src/**/*.test.tsx")
		fmt.Fprintln(w, "   • Has e2e playbooks: bas/cases/**/ui/*.json")
		fmt.Fprintln(w, "   • Requirements reference appropriate test types")
	case score >= 80 && analysis.TotalPenalty < 10:
		fmt.Fprintln(w, "🌟 Excellent work! This scenario demonstrates:")
		fmt.Fprintln(w, "   ✓ Comprehensive multi-layer testing")
		fmt.Fprintln(w, "   ✓ Proper test organization")
		fmt.Fprintln(w, "   ✓ High pass rates across all metrics")
		fmt.Fprintln(w, "   ✓ Minimal gaming patterns detected")
	case score >= 40 && analysis.TotalPenalty < 30:
		fmt.Fprintln(w, "✨ This scenario has good test structure - continue improving:")
		fmt.Fprintln(w, "   • Proper use of bas/ for e2e testing")
		fmt.Fprintln(w, "   • Good mix of test types where present")
		fmt.Fprintln(w, "   • Focus on increasing test coverage and pass rates")
	}
	fmt.Fprintln(w)
}
