package bootstrap

import (
	"io"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/project"
	"github.com/vrooli/vrooli/internal/resources"
)

type Services struct {
	Root   string
	Home   string
	Stdout io.Writer
	Stderr io.Writer
	Logger *slog.Logger

	once         sync.Once
	orchestrator *orchestrator.Service
	resources    *resources.Controller
	maintenance  *maintenance.Controller
	project      *project.Controller
}

func New(root, home string, stdout, stderr io.Writer, logger *slog.Logger) *Services {
	baseLogger := logger
	if baseLogger == nil {
		baseLogger = slog.Default()
	}
	return &Services{
		Root:   filepath.Clean(root),
		Home:   filepath.Clean(home),
		Stdout: stdout,
		Stderr: stderr,
		Logger: logx.WithSubsystem(baseLogger, "bootstrap"),
	}
}

func (s *Services) Orchestrator() *orchestrator.Service {
	s.init()
	return s.orchestrator
}

func (s *Services) Resources() *resources.Controller {
	s.init()
	return s.resources
}

func (s *Services) Maintenance() *maintenance.Controller {
	s.init()
	return s.maintenance
}

func (s *Services) Project() *project.Controller {
	s.init()
	return s.project
}

func (s *Services) LifecycleRunner() (*lifecycle.Runner, error) {
	return lifecycle.NewRunner(s.Root, s.Home, s.Stdout, s.Stderr, s.Logger)
}

func (s *Services) init() {
	s.once.Do(func() {
		s.resources = resources.NewController(s.Root, s.Home)
		s.orchestrator = orchestrator.New(s.Root, s.Home, s.Stdout, s.Stderr, s.Logger)
		s.maintenance = maintenance.NewController(s.Root, s.Home)
		s.project = project.NewWithDependencies(s.Root, s.Home, s.Stdout, s.Stderr, project.Dependencies{
			Resources:   s.resources,
			Scenarios:   s.orchestrator,
			Maintenance: s.maintenance,
		})
	})
}
