// test-quality-calibrate compares adapter-produced observations with separately
// authored expectations. It never invents observations for missing adapters.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"unit-health/internal/testquality"
	"unit-health/internal/testquality/calibration"
)

func main() {
	root := flag.String("root", "internal/testquality/testdata", "fixture root")
	observations := flag.String("observations", "", "JSON object of case ID to adapter observations")
	nativePath := flag.String("native-observations", "", "native runner observation JSON")
	contextPath := flag.String("context-observations", "", "owner/reference observation JSON array")
	holdoutPath := flag.String("holdout", "", "independently authored holdout labels JSON")
	holdoutObservationsPath := flag.String("holdout-observations", "", "sampled observations JSON for holdout comparison")
	outputPath := flag.String("output", "", "write the complete report to this file instead of stdout")
	flag.Parse()
	cases, err := calibration.LoadCases(*root, "development.json", "development")
	if err != nil {
		fail(err)
	}
	inventory, err := calibration.Inventory(*root, cases)
	if err != nil {
		fail(err)
	}
	observed := map[string][]testquality.Result{}
	if *observations != "" {
		data, err := os.ReadFile(*observations)
		if err != nil {
			fail(err)
		}
		if err := json.Unmarshal(data, &observed); err != nil {
			fail(err)
		}
	}
	report, err := calibration.Run(context.Background(), cases, "development", func(_ context.Context, in calibration.Input) ([]testquality.Result, error) {
		rows, ok := observed[in.ID]
		if !ok {
			return nil, fmt.Errorf("no adapter observations supplied for %s", in.ID)
		}
		return rows, nil
	})
	if err != nil {
		fail(err)
	}
	pending := 0
	spec, err := calibration.LoadNativeSpecification(*root)
	if err != nil {
		fail(err)
	}
	var nativeObservation calibration.NativeObservation
	if *nativePath != "" {
		data, err := os.ReadFile(*nativePath)
		if err != nil {
			fail(err)
		}
		if err := json.Unmarshal(data, &nativeObservation); err != nil {
			fail(err)
		}
	}
	native := calibration.CompareNative(spec, nativeObservation)
	contextCases, err := calibration.LoadContextCases(*root)
	if err != nil {
		fail(err)
	}
	var contextObservations []calibration.ContextObservation
	if *contextPath != "" {
		data, err := os.ReadFile(*contextPath)
		if err != nil {
			fail(err)
		}
		if err := json.Unmarshal(data, &contextObservations); err != nil {
			fail(err)
		}
	}
	contextReport := calibration.CompareContext(contextCases, contextObservations)
	var holdout *calibration.HoldoutComparison
	if *holdoutPath != "" {
		labelsData, err := os.ReadFile(*holdoutPath)
		if err != nil {
			fail(err)
		}
		var labels []testquality.HoldoutLabel
		if err := json.Unmarshal(labelsData, &labels); err != nil {
			fail(err)
		}
		var observations []testquality.SampledObservation
		if *holdoutObservationsPath != "" {
			obsData, err := os.ReadFile(*holdoutObservationsPath)
			if err != nil {
				fail(err)
			}
			if err := json.Unmarshal(obsData, &observations); err != nil {
				fail(err)
			}
		}
		comparison, err := calibration.CompareHoldout(labels, observations, testquality.SchemaVersion)
		if err != nil {
			fail(err)
		}
		holdout = &comparison
	}
	for _, row := range inventory {
		if row.Disposition == "pending-fixture" {
			pending++
		}
	}
	out := struct {
		Inventory    []calibration.InventoryEntry   `json:"inventory"`
		PendingCases int                            `json:"pendingCases"`
		Calibration  calibration.Report             `json:"calibration"`
		Native       calibration.NativeReport       `json:"native"`
		Context      calibration.ContextReport      `json:"context"`
		Holdout      *calibration.HoldoutComparison `json:"holdout,omitempty"`
	}{inventory, pending, report, native, contextReport, holdout}
	output := os.Stdout
	if *outputPath != "" {
		file, err := os.Create(*outputPath)
		if err != nil {
			fail(err)
		}
		defer file.Close()
		output = file
	}
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(out); err != nil {
		fail(err)
	}
	if report.FailedCases > 0 || pending > 0 || len(native.Differences) > 0 || contextReport.MatchedCases != contextReport.TotalCases || len(contextReport.UnexpectedIDs) > 0 {
		os.Exit(1)
	}
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(1) }
