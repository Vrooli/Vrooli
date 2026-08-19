package packagegov

import (
	"fmt"
	"sort"
)

// ConsumerClassViolation records a discovered dependent whose consumer class
// is outside the package's declared adoption boundary.
type ConsumerClassViolation struct {
	PackageName      string          `json:"package_name"`
	ConsumerName     string          `json:"consumer_name"`
	ConsumerPath     string          `json:"consumer_path"`
	ConsumerClass    ConsumerClass   `json:"consumer_class"`
	DependencyFile   string          `json:"dependency_file"`
	AllowedConsumers []ConsumerClass `json:"allowed_consumers,omitempty"`
}

// ValidateConsumerClassBoundary compares discovered consumers with the
// package manifest's allowed consumer classes.
func ValidateConsumerClassBoundary(pkg Package, dependents []Dependent) []ConsumerClassViolation {
	allowed := append([]ConsumerClass(nil), pkg.Manifest.Package.Adoption.AllowedConsumers...)
	sort.Slice(allowed, func(i, j int) bool { return allowed[i] < allowed[j] })

	violations := make([]ConsumerClassViolation, 0)
	for _, dependent := range dependents {
		if containsConsumerClass(allowed, dependent.ConsumerClass) {
			continue
		}
		violations = append(violations, ConsumerClassViolation{
			PackageName:      pkg.Name,
			ConsumerName:     dependent.ConsumerName,
			ConsumerPath:     dependent.ConsumerPath,
			ConsumerClass:    dependent.ConsumerClass,
			DependencyFile:   dependent.DependencyFile,
			AllowedConsumers: append([]ConsumerClass(nil), allowed...),
		})
	}

	sort.Slice(violations, func(i, j int) bool {
		if violations[i].ConsumerName != violations[j].ConsumerName {
			return violations[i].ConsumerName < violations[j].ConsumerName
		}
		return violations[i].DependencyFile < violations[j].DependencyFile
	})
	return violations
}

func (v ConsumerClassViolation) ValidationIssue() ValidationIssue {
	return ValidationIssue{
		Severity:    "error",
		Code:        "PACKAGE_CONSUMER_CLASS_VIOLATION",
		Message:     fmt.Sprintf("package %q is consumed by %q as %s, but that consumer class is not allowed", v.PackageName, v.ConsumerName, v.ConsumerClass),
		Path:        v.DependencyFile,
		PackageName: v.PackageName,
	}
}
