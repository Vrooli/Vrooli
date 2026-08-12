package modeltest

import "fmt"

// ValidateFormalArtifactFresh checks that a formal artifact is complete, fresh
// (its recorded source hashes still match the files on disk), and matches the
// caller's expectations. It returns every problem found rather than failing
// fast, so callers see the full picture in one shot.
func ValidateFormalArtifactFresh(artifact FormalArtifact, expected FormalArtifactExpectation) []error {
	var errs []error
	errs = append(errs, validateArtifactIdentity(artifact, expected)...)
	errs = append(errs, validateArtifactSourceFreshness(artifact, expected)...)
	errs = append(errs, validateArtifactChecks(artifact.Checks)...)
	errs = append(errs, validateArtifactExpectations(artifact, expected)...)
	errs = append(errs, validateArtifactCoverage(artifact.Coverage)...)
	errs = append(errs, validateArtifactCollections(artifact)...)
	return errs
}

func validateArtifactIdentity(artifact FormalArtifact, expected FormalArtifactExpectation) []error {
	var errs []error
	if artifact.SchemaVersion != formalArtifactSchemaVersion {
		errs = append(errs, fmt.Errorf("formal artifact schemaVersion=%d, want %d", artifact.SchemaVersion, formalArtifactSchemaVersion))
	}
	if artifact.FlowID == "" {
		errs = append(errs, fmt.Errorf("formal artifact flowId is required"))
	}
	if artifact.Source.ContractPath != expected.ContractPath {
		errs = append(errs, fmt.Errorf("formal artifact contractPath=%s, want %s", artifact.Source.ContractPath, expected.ContractPath))
	}
	if artifact.Source.ModelPath != expected.ModelPath {
		errs = append(errs, fmt.Errorf("formal artifact modelPath=%s, want %s", artifact.Source.ModelPath, expected.ModelPath))
	}
	if expected.GeneratorPath != "" && artifact.Source.GeneratorPath != expected.GeneratorPath {
		errs = append(errs, fmt.Errorf("formal artifact generatorPath=%s, want %s", artifact.Source.GeneratorPath, expected.GeneratorPath))
	}
	if artifact.Source.GeneratorPath == "" {
		errs = append(errs, fmt.Errorf("formal artifact generatorPath is required"))
	}
	if artifact.Source.GeneratorVersion < 1 {
		errs = append(errs, fmt.Errorf("formal artifact generatorVersion is required"))
	}
	if artifact.Source.VerificationBackend == "" {
		errs = append(errs, fmt.Errorf("formal artifact verificationBackend is required"))
	}
	if artifact.Source.QuintVersion == "" {
		errs = append(errs, fmt.Errorf("formal artifact quintVersion is required"))
	}
	return errs
}

func validateArtifactSourceFreshness(artifact FormalArtifact, expected FormalArtifactExpectation) []error {
	var errs []error
	if err := validateFreshHash(artifact.Source.ContractPath, artifact.Source.ContractSHA256, "contractSha256"); err != nil {
		errs = append(errs, err)
	}
	if expected.ContractSHA256 != "" && artifact.Source.ContractSHA256 != expected.ContractSHA256 {
		errs = append(errs, fmt.Errorf("formal artifact contractSha256=%s, want %s", artifact.Source.ContractSHA256, expected.ContractSHA256))
	}
	if expected.GeneratorSHA256 != "" && artifact.Source.GeneratorSHA256 != expected.GeneratorSHA256 {
		errs = append(errs, fmt.Errorf("formal artifact generatorSha256=%s, want %s", artifact.Source.GeneratorSHA256, expected.GeneratorSHA256))
	}
	if err := validateFreshHash(expected.ModelPath, artifact.Source.ModelSHA256, "modelSha256"); err != nil {
		errs = append(errs, err)
	}
	if expected.ModelSHA256 != "" && artifact.Source.ModelSHA256 != expected.ModelSHA256 {
		errs = append(errs, fmt.Errorf("formal artifact modelSha256=%s, want %s", artifact.Source.ModelSHA256, expected.ModelSHA256))
	}
	return errs
}

func validateArtifactChecks(checks FormalArtifactChecks) []error {
	required := []struct {
		ok     bool
		reason string
	}{
		{checks.Typechecked, "was not typechecked"},
		{checks.Tested, "was not tested"},
		{checks.Verified, "was not verified"},
		{checks.GeneratedFromContract, "was not generated from contract"},
		{checks.GeneratedFromModel, "was not generated from model"},
	}
	var errs []error
	for _, c := range required {
		if !c.ok {
			errs = append(errs, fmt.Errorf("formal artifact %s", c.reason))
		}
	}
	return errs
}

func validateArtifactExpectations(artifact FormalArtifact, expected FormalArtifactExpectation) []error {
	var errs []error
	for _, invariant := range expected.Invariants {
		if !containsString(artifact.Invariants, invariant) {
			errs = append(errs, fmt.Errorf("formal artifact missing invariant %s", invariant))
		}
	}
	for _, check := range expected.GeneratedChecks {
		if !containsString(artifact.GeneratedChecks, check) {
			errs = append(errs, fmt.Errorf("formal artifact missing generated check %s", check))
		}
	}
	return errs
}

func validateArtifactCoverage(coverage FormalArtifactCoverage) []error {
	var errs []error
	if !coverage.TransitionMatrixComplete {
		errs = append(errs, fmt.Errorf("formal artifact transition matrix is incomplete"))
	}
	if !coverage.TerminalTransitionsChecked {
		errs = append(errs, fmt.Errorf("formal artifact does not check terminal transitions"))
	}
	if !coverage.NamedTraces.AllStatesCovered {
		errs = append(errs, fmt.Errorf("formal artifact named traces do not cover all states"))
	}
	if !coverage.NamedTraces.AllEventsCovered {
		errs = append(errs, fmt.Errorf("formal artifact named traces do not cover all events"))
	}
	if len(coverage.GeneratedTraces.CoveredStates) == 0 {
		errs = append(errs, fmt.Errorf("formal artifact generated traces do not report covered states"))
	}
	if len(coverage.GeneratedTraces.CoveredEvents) == 0 {
		errs = append(errs, fmt.Errorf("formal artifact generated traces do not report covered events"))
	}
	if coverage.GeneratedTraces.CoveredPairs == nil {
		errs = append(errs, fmt.Errorf("formal artifact generated traces do not report covered pairs"))
	}
	return errs
}

func validateArtifactCollections(artifact FormalArtifact) []error {
	var errs []error
	if len(artifact.Transitions) == 0 {
		errs = append(errs, fmt.Errorf("formal artifact transitions must not be empty"))
	}
	if len(artifact.NamedTraces) == 0 {
		errs = append(errs, fmt.Errorf("formal artifact namedTraces must not be empty"))
	}
	if len(artifact.GeneratedTraces) == 0 {
		errs = append(errs, fmt.Errorf("formal artifact generatedTraces must not be empty"))
	}
	return errs
}
