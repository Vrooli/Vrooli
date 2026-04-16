package lint

func evaluatePolicy(component Component, matched bool, settings *Settings) []PolicyFinding {
	if matched {
		return nil
	}

	severity, ok := settings.Policy.UnconfiguredCommonComponents[component.Name]
	if ok {
		return []PolicyFinding{{
			Component: component.Name,
			Path:      component.RelativePath,
			Severity:  severity,
			Message:   "common component is present without a supported lint contract",
		}}
	}

	if component.CodeBearing {
		return []PolicyFinding{{
			Component: component.Name,
			Path:      component.RelativePath,
			Severity:  settings.Policy.UnmatchedCodeComponents,
			Message:   "code-bearing component has no supported lint contract",
		}}
	}

	return nil
}

func observationForPolicyFinding(finding PolicyFinding) Observation {
	message := finding.Message
	if finding.Component != "." {
		message = finding.Component + ": " + message
	}

	switch finding.Severity {
	case PolicySeverityInfo:
		return NewInfoObservation(message)
	case PolicySeverityWarning:
		return NewWarningObservation(message)
	case PolicySeverityError:
		return NewErrorObservation(message)
	default:
		return Observation{}
	}
}
