package autosteer

import (
	"fmt"
	"strings"
)

// ModeInstructions provides detailed instructions for each steer mode
type ModeInstructions struct{}

// NewModeInstructions creates a new mode instructions provider
func NewModeInstructions() *ModeInstructions {
	return &ModeInstructions{}
}

// GetInstructions returns the detailed instructions for a given mode
func (m *ModeInstructions) GetInstructions(mode SteerMode) string {
	instructions := map[SteerMode]string{
		ModeProgress:    m.getProgressInstructions(),
		ModeUX:          m.getUXInstructions(),
		ModeRefactor:    m.getRefactorInstructions(),
		ModeTest:        m.getTestInstructions(),
		ModeExplore:     m.getExploreInstructions(),
		ModePolish:      m.getPolishInstructions(),
		ModeIntegration: m.getIntegrationInstructions(),
		ModePerformance: m.getPerformanceInstructions(),
		ModeSecurity:    m.getSecurityInstructions(),
	}

	if instr, ok := instructions[mode]; ok {
		return instr
	}

	return fmt.Sprintf("Unknown mode: %s. Proceed with general improvements.", mode)
}

// GetToolRecommendations returns recommended tools for a mode
func (m *ModeInstructions) GetToolRecommendations(mode SteerMode) []string {
	tools := map[SteerMode][]string{
		ModeProgress: {
			"scenario-dependency-analyzer (find related capabilities)",
			"All standard scenario resources",
		},
		ModeUX: {
			"browser-automation-studio (test user flows, validate accessibility)",
			"react-component-library (use consistent, accessible components)",
			"tidiness-manager (ensure code organization doesn't hinder UX work)",
		},
		ModeRefactor: {
			"tidiness-manager (identify issues, validate improvements)",
			"golangci-lint / eslint (automated quality checks)",
			"ast-grep (structural code analysis and refactoring)",
		},
		ModeTest: {
			"test-genie (generate high-quality tests, identify gaps)",
			"browser-automation-studio (UI automation and testing)",
			"Existing test frameworks (vitest, bats, go test)",
		},
		ModeExplore: {
			"All available resources and scenarios",
			"Experimentation encouraged - try novel approaches",
		},
		ModePolish: {
			"browser-automation-studio (verify edge cases)",
			"User flow validation tools",
		},
		ModeIntegration: {
			"scenario-dependency-analyzer (find integration opportunities)",
			"All available resources and scenarios",
			"deployment-manager (understand dependencies)",
		},
		ModePerformance: {
			"Browser DevTools (profiling, network analysis)",
			"Backend profiling tools",
			"Database query analyzers",
		},
		ModeSecurity: {
			"Security scanning tools",
			"secrets-manager integration",
			"Dependency audit tools",
		},
	}

	if recommendations, ok := tools[mode]; ok {
		return recommendations
	}

	return []string{}
}

// GetSuccessCriteria returns success criteria for a mode
func (m *ModeInstructions) GetSuccessCriteria(mode SteerMode) []string {
	criteria := map[SteerMode][]string{
		ModeProgress: {
			"Operational target completion percentage increases",
			"All implemented features have basic UI, API, and database components",
			"No regression in existing operational targets",
		},
		ModeUX: {
			"Accessibility score > 90%",
			"All operational targets have complete user flows",
			"Responsive across mobile, tablet, desktop",
			"Zero UX-breaking errors",
		},
		ModeRefactor: {
			"Tidiness score improves",
			"Standards violations decrease",
			"No regression in operational targets or tests",
			"Code is more maintainable",
		},
		ModeTest: {
			"Unit coverage > 80%",
			"Integration coverage > 70%",
			"UI coverage > 60%",
			"Zero flaky tests",
			"All tests have clear failure messages",
		},
		ModeExplore: {
			"At least one novel/interesting addition per iteration",
			"Experiments are documented",
			"Build remains stable",
		},
		ModePolish: {
			"Zero edge cases without proper handling",
			"All user actions have clear feedback",
			"Application feels complete and professional",
		},
		ModeIntegration: {
			"At least one meaningful integration per iteration",
			"Integrations are documented",
			"APIs are designed for external use",
		},
		ModePerformance: {
			"Initial load time < 3s",
			"Core Web Vitals in Good range",
			"Database queries optimized (no N+1 queries)",
			"Appropriate caching strategy implemented",
		},
		ModeSecurity: {
			"Zero high-severity vulnerabilities",
			"All inputs validated",
			"Auth/authz properly implemented",
			"Dependencies up to date with no known CVEs",
		},
	}

	if c, ok := criteria[mode]; ok {
		return c
	}

	return []string{}
}

// Mode-specific instruction implementations

func (m *ModeInstructions) getProgressInstructions() string {
	return `Focus on implementing missing operational targets and progressing incomplete features.
Prioritize breadth over depth - get features working end-to-end.

Key Areas:
- Implement missing operational targets from requirements
- Complete partial feature implementations
- Ensure basic UI, API, and database components exist for all features
- Maintain existing functionality - no regressions allowed

Approach:
- Review requirements/index.json for incomplete operational targets
- Focus on making tests pass rather than perfect implementation
- Build complete vertical slices (UI -> API -> Database)
- Validate that operational target status updates correctly`
}

func (m *ModeInstructions) getUXInstructions() string {
	return `Focus on user experience improvements and accessibility. Make the application
delightful to use, accessible to all users, and visually polished.

Key Areas:
- Accessibility: ARIA labels, keyboard navigation, screen reader support
- Responsive design: Test and optimize for all breakpoints
- User flows: Ensure smooth, intuitive paths for all operational targets
- Visual polish: Consistent spacing, typography, color usage
- Loading states: All async operations show appropriate feedback
- Error handling: Clear, helpful error messages with recovery paths
- Performance UX: Perceived performance, optimistic updates

Follow F-layout principles and modern UX best practices.

Guidelines:
- Every interactive element must be keyboard accessible
- Use semantic HTML and proper ARIA attributes
- Test with screen readers if possible
- Ensure color contrast meets WCAG AA standards
- Add loading indicators for operations > 300ms
- Provide clear feedback for all user actions`
}

func (m *ModeInstructions) getRefactorInstructions() string {
	return `Focus on code quality and maintainability. Reduce technical debt, improve
code organization, and ensure adherence to project standards.

Key Areas:
- Reduce cyclomatic complexity (target: < 10 per function)
- Eliminate code duplication (target: < 5%)
- Enforce naming conventions and project standards
- Improve code organization and module boundaries
- Add/improve code documentation
- Remove dead code and unused dependencies

Approach:
- Run tidiness-manager to identify issues
- Use golangci-lint (Go) or eslint (TypeScript) for automated checks
- Extract complex functions into smaller, focused units
- Consolidate duplicate code into shared utilities
- Add clear comments for complex logic
- Ensure consistent code style across the codebase

Remember: All refactoring must preserve functionality - tests must continue to pass.`
}

func (m *ModeInstructions) getTestInstructions() string {
	return `Focus on comprehensive testing across all layers. Write high-quality tests
that catch real issues and provide confidence in changes.

Key Areas:
- Unit tests: Cover edge cases, error paths, boundary conditions
- Integration tests: Validate component interactions
- UI tests: Verify user flows and accessibility
- Test quality: Clear, maintainable tests that fail meaningfully
- Reduce flakiness: Eliminate non-deterministic test failures

Testing Strategy:
- Start with unit tests for business logic
- Add integration tests for API endpoints
- Create UI tests for critical user flows
- Test error handling and edge cases
- Ensure tests have descriptive names and clear assertions
- Use test fixtures and helpers to reduce duplication

Quality Checks:
- All tests must have clear failure messages
- No hardcoded timeouts or sleeps (causes flakiness)
- Tests should be fast and isolated
- Mock external dependencies appropriately`
}

func (m *ModeInstructions) getExploreInstructions() string {
	return `This is your creative sandbox. Experiment with interesting approaches,
novel UIs, innovative features, or alternative implementations.

Key Areas:
- Try unconventional UI patterns that might improve UX
- Explore alternative architectural approaches
- Experiment with new libraries or techniques
- Implement "nice to have" features that add delight
- Think outside the box on problem-solving

Guidelines:
- Don't break existing functionality
- Document experiments clearly (comments, commit messages)
- If an experiment succeeds, integrate it properly
- If it fails, revert cleanly with git
- Be creative but maintain code quality

Ideas to Consider:
- Novel interaction patterns (drag-and-drop, gestures, etc.)
- Animations and transitions for better UX
- Alternative data visualizations
- Keyboard shortcuts and power user features
- Easter eggs or delightful micro-interactions

Remember: Innovation is encouraged, but not at the cost of stability.`
}

func (m *ModeInstructions) getPolishInstructions() string {
	return `Focus on the details that make a production-ready application. This is
the final 10% that takes 90% of the time.

Key Areas:
- Error messages: Clear, actionable, user-friendly
- Loading states: Every async operation has feedback
- Empty states: Helpful guidance when no data exists
- Edge cases: Handle all boundary conditions gracefully
- Validation messages: Clear, specific, helpful
- Tooltips and help text: Guide users proactively
- Confirmation dialogs: Prevent destructive actions
- Success feedback: Confirm actions completed

Checklist:
- All forms have proper validation with helpful messages
- All buttons show loading state during async operations
- Empty states provide guidance on next actions
- Destructive actions require confirmation
- Success/error toasts for all user actions
- No raw error messages from backend (translate to user-friendly text)
- All edge cases have graceful fallbacks
- Consistent messaging tone throughout the app

Think like a user: What would feel incomplete or frustrating?`
}

func (m *ModeInstructions) getIntegrationInstructions() string {
	return `Focus on integrating with other scenarios and resources in the ecosystem.
Expand the capability surface by composing existing tools.

Key Areas:
- Identify relevant scenarios that could enhance functionality
- Integrate with additional resources (databases, AI services, etc.)
- Enable this scenario to be used BY other scenarios (API design)
- Cross-scenario workflows and data sharing
- Resource composition for emergent capabilities

Integration Strategy:
- Use scenario-dependency-analyzer to find integration opportunities
- Design clean APIs that other scenarios can consume
- Document integration points clearly
- Ensure backward compatibility for published APIs
- Test integrations thoroughly

Examples:
- Use tidiness-manager for code quality scanning
- Integrate with secrets-manager for credential storage
- Leverage test-genie for automated test generation
- Connect to deployment-manager for deployment workflows

Remember: Good integrations are loosely coupled and well-documented.`
}

func (m *ModeInstructions) getPerformanceInstructions() string {
	return `Focus on performance optimization across frontend and backend.

Key Areas:
Frontend:
- Bundle size reduction (code splitting, tree shaking)
- Lazy loading for routes and components
- Render optimization (memoization, virtualization)
- Asset optimization (image compression, lazy loading)

Backend:
- Query optimization (indexes, N+1 prevention)
- Caching strategies (Redis, in-memory caching)
- Connection pooling
- Response compression

Measurement:
- Use Lighthouse for frontend metrics
- Profile backend with appropriate tools
- Monitor database query performance
- Measure before and after optimizations

Targets:
- Initial load time < 3 seconds
- Time to Interactive < 5 seconds
- Largest Contentful Paint < 2.5 seconds
- First Input Delay < 100ms
- Cumulative Layout Shift < 0.1

Remember: Measure first, optimize second. Don't optimize prematurely.`
}

func (m *ModeInstructions) getSecurityInstructions() string {
	return `Focus on security best practices and vulnerability remediation.

Key Areas:
- Input validation: All user inputs sanitized and validated
- Authentication: Secure implementation, proper session handling
- Authorization: Proper access controls for all operations
- SQL injection prevention: Use parameterized queries
- XSS prevention: Sanitize output, use Content Security Policy
- CSRF protection: Use tokens for state-changing operations
- Dependency vulnerabilities: Keep dependencies updated
- Secrets management: Never commit secrets, use secrets-manager

Security Checklist:
- All inputs validated on both client and server
- All database queries use parameterized statements
- Authentication tokens properly secured (httpOnly, secure flags)
- Authorization checked for every protected operation
- Sensitive data encrypted at rest and in transit
- No secrets in code or environment files
- Dependencies scanned for known vulnerabilities
- Error messages don't leak sensitive information

Tools:
- npm audit / go mod verify for dependency scanning
- secrets-manager for credential management
- Static analysis tools for code vulnerabilities

Remember: Security is not optional. Validate everything, trust nothing.`
}

// FormatConditionProgress formats stop conditions with current values
func (m *ModeInstructions) FormatConditionProgress(conditions []StopCondition, metrics MetricsSnapshot, evaluator *ConditionEvaluator) string {
	if len(conditions) == 0 {
		return "No stop conditions defined for this phase."
	}

	var output strings.Builder
	for i, condition := range conditions {
		status := "✗"
		if result, err := evaluator.Evaluate(condition, metrics); err == nil && result {
			status = "✓"
		}

		conditionStr := m.formatConditionHuman(condition, metrics)
		output.WriteString(fmt.Sprintf("%s %s\n", status, conditionStr))

		if i < len(conditions)-1 {
			output.WriteString("\n")
		}
	}

	return output.String()
}

// formatConditionHuman formats a condition in human-readable form with current values
func (m *ModeInstructions) formatConditionHuman(condition StopCondition, metrics MetricsSnapshot) string {
	if condition.Type == ConditionTypeSimple {
		currentValue := m.getMetricDisplayValue(condition.Metric, metrics)
		return fmt.Sprintf("%s %s %.1f (current: %s)",
			condition.Metric,
			condition.CompareOperator,
			condition.Value,
			currentValue,
		)
	}

	// Compound condition
	var parts []string
	for _, subCondition := range condition.Conditions {
		parts = append(parts, m.formatConditionHuman(subCondition, metrics))
	}

	operator := strings.ToUpper(string(condition.Operator))
	return fmt.Sprintf("(%s)", strings.Join(parts, " "+operator+" "))
}

// getMetricDisplayValue gets a metric value as a display string
func (m *ModeInstructions) getMetricDisplayValue(metricName string, metrics MetricsSnapshot) string {
	evaluator := NewConditionEvaluator()
	value, err := evaluator.GetMetricValue(metricName, metrics)
	if err != nil {
		return "N/A"
	}

	return fmt.Sprintf("%.1f", value)
}
