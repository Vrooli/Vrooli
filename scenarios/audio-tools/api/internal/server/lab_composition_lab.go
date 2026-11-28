//go:build audio_lab

package server

import (
	corpusH "audio-tools/handlers/corpus"
	evalH "audio-tools/handlers/eval"
	experimentH "audio-tools/handlers/experiment"
	soakH "audio-tools/handlers/soak"
	validationH "audio-tools/handlers/validation"
	intcorpus "audio-tools/internal/corpus"
	inteval "audio-tools/internal/eval"
	intexp "audio-tools/internal/experiment"
	"audio-tools/internal/experiment/report"
	"audio-tools/internal/modulekit"
	"github.com/vrooli/api-core/schedule"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/eval"
)

type LabRuntimeDeps struct {
	Corpus            *intcorpus.Service
	ExperimentManager *intexp.Manager
	Experiments       *intexp.Service
}

func ComposeWithLab(d CompositionDeps, lab LabRuntimeDeps) *Server {
	return compose(d, labModules(d, lab))
}

func ComposeForBuild(d CompositionDeps) *Server { return ComposeWithLab(d, LabRuntimeDeps{}) }

func ReportToProto(report inteval.EvalReport) *evalv1.EvalReport { return evalH.ReportToProto(report) }

func MetricsOnlyReportToProto(report inteval.EvalReport) *evalv1.EvalReport {
	return evalH.MetricsOnlyReportToProto(report)
}

func labModules(d CompositionDeps, lab LabRuntimeDeps) []modulekit.Module {
	return []modulekit.Module{
		corpusH.Module(corpusH.Deps{Logger: d.Logger, Clock: schedule.System(), Service: lab.Corpus}),
		experimentH.Module(experimentH.Deps{Logger: d.Logger, Manager: lab.ExperimentManager, Service: lab.Experiments, EstimateClipSeconds: report.EstimateClipSeconds(lab.Corpus)}),
		validationH.Module(validationH.Deps{ScenarioDir: "", StreamLedgers: d.StreamLedgers}),
		soakH.Module(soakH.Deps{Ledgers: d.StreamLedgers, Experiments: lab.Experiments, TestIsolationActive: d.Env.TestIsolationActive}),
	}
}
