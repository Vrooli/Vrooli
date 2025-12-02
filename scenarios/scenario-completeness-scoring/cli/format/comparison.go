package format

import "fmt"

import "scenario-completeness-scoring/cli/models"

func FormatComparisonContext(analysis models.ValidationQualityAnalysis, score float64) {
	fmt.Println(sectionSep)
	fmt.Println()
	switch {
	case analysis.TotalPenalty > 50:
		fmt.Println("🎓 Study browser-automation-studio as reference for proper test structure:")
		fmt.Println("   • Has API tests: api/**/*_test.go")
		fmt.Println("   • Has UI tests: ui/src/**/*.test.tsx")
		fmt.Println("   • Has e2e playbooks: test/playbooks/capabilities/**/ui/*.json")
		fmt.Println("   • Requirements reference appropriate test types")
	case score >= 80 && analysis.TotalPenalty < 10:
		fmt.Println("🌟 Excellent work! This scenario demonstrates:")
		fmt.Println("   ✓ Comprehensive multi-layer testing")
		fmt.Println("   ✓ Proper test organization")
		fmt.Println("   ✓ High pass rates across all metrics")
		fmt.Println("   ✓ Minimal gaming patterns detected")
	case score >= 40 && analysis.TotalPenalty < 30:
		fmt.Println("✨ This scenario has good test structure - continue improving:")
		fmt.Println("   • Proper use of test/playbooks/ for e2e testing")
		fmt.Println("   • Good mix of test types where present")
		fmt.Println("   • Focus on increasing test coverage and pass rates")
	}
	fmt.Println()
}
