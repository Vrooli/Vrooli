package planning

import "context"

// seam: Repository persists planned scenarios and their proto file tree.
// Production wires SQLiteRepository; tests wire mocks.FakeRepository.
type Repository interface {
	CreateScenario(ctx context.Context, in CreateInput) (Scenario, error)
	ListScenarios(ctx context.Context, filter ListFilter) ([]Scenario, error)
	GetScenario(ctx context.Context, slug string) (Scenario, error)
	PutFile(ctx context.Context, in PutFileInput) (ProtoFile, error)
	DeleteFile(ctx context.Context, slug, path string) (bool, error)
}

// seam: ProtoValidator validates planned proto text against repository schemas.
// Production wires CompilerValidator; tests wire mocks.FakeValidator.
type ProtoValidator interface {
	Validate(ctx context.Context, scenario Scenario) ([]PlanFinding, error)
}

// seam: Materializer writes validated planned proto text into packages/proto.
// Production wires FilesystemMaterializer; tests wire mocks.FakeMaterializer.
type Materializer interface {
	Materialize(ctx context.Context, scenario Scenario) (MaterializeResult, error)
}
