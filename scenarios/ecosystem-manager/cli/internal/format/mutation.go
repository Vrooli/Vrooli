package format

import "fmt"

// MutationResult prints a mutation result with optional details and next-step commands.
func MutationResult(result string, details string, nextSteps []string) {
	fmt.Println(result)
	if details != "" {
		fmt.Printf("  %s\n", details)
	}
	if len(nextSteps) > 0 {
		fmt.Println("\nNext steps:")
		for _, step := range nextSteps {
			fmt.Printf("  $ %s\n", step)
		}
	}
}
