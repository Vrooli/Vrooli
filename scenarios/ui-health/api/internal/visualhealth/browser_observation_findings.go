package visualhealth

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/visualhealth"
)

// These are client-measured geometry/style judgments, never native pixel claims.
func browserObservationFindings(row *pb.BrowserObservation) []*pb.VisualFinding {
	var layout map[string]any
	if json.Unmarshal([]byte(row.GetLayoutJson()), &layout) != nil {
		return nil
	}
	var out []*pb.VisualFinding
	add := func(code, message, evidence, remediation string) {
		out = append(out, &pb.VisualFinding{Code: code, Severity: severityError, Category: categoryLayout, Message: message, Evidence: evidence, Remediation: remediation, Location: row.Scenario, StepId: row.Id})
	}
	viewport, screen := firstMap(layout, "viewport"), firstMap(layout, "screen")
	canvas, shell := firstMap(layout, "cssViewport"), firstMap(layout, "shell")
	wants := firstMap(layout, "expectations")
	keyboard := boolValue(layout, "keyboardVisible")
	if row.Standalone && !row.Embedded && !keyboard && boolValue(wants, "fullScreenCanvas") &&
		math.Abs(firstNumber(viewport, "width")-firstNumber(screen, "width")) <= 2 &&
		firstNumber(canvas, "largeHeight") > 0 && math.Abs(firstNumber(canvas, "largeHeight")-firstNumber(screen, "height")) <= 2 {
		unused := firstNumber(canvas, "largeHeight") - firstNumber(shell, "bottom")
		if unused > 2 {
			add("visual_drawable_canvas_unused", "installed app leaves measured drawable canvas unused", fmt.Sprintf("CSS canvas %.1fpx; shell bottom %.1fpx; unused %.1fpx", firstNumber(canvas, "largeHeight"), firstNumber(shell, "bottom"), unused), "Use the shared measured drawable frame for the app roots, shell and overlays.")
		}
	}
	header := firstMap(layout, "header")
	if maximum := firstNumber(wants, "headerMaxHeight"); maximum > 0 && firstNumber(header, "height") > maximum+1 {
		add("visual_mobile_toolbar_oversized", "mobile toolbar exceeds its declared compact height", fmt.Sprintf("header %.1fpx; limit %.1fpx", firstNumber(header, "height"), maximum), "Remove duplicate toolbar padding while preserving the separate status inset and tap targets.")
	}
	overlays := arrayValue(layout, "overlays")
	if len(overlays) == 0 {
		for _, item := range arrayValue(layout, "controls") {
			control, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if hit, present := control["centerHit"].(bool); present && !hit {
				add("visual_nav_control_unreachable", "a visible navigation control misses its client hit test", fmt.Sprintf("control top %.1fpx; bottom %.1fpx", firstNumber(control, "top"), firstNumber(control, "bottom")), "Keep the full control inside the drawable, hit-testable app frame.")
				break
			}
		}
	} else {
		// The last semantic dialog is the frontmost owner in these client reports.
		overlay, _ := overlays[len(overlays)-1].(map[string]any)
		panel := firstMap(overlay, "panel")
		color := firstString(panel, "background")
		body := firstString(firstMap(layout, "body"), "background")
		if color != "" && body != "" && color != "rgba(0, 0, 0, 0)" && strings.ReplaceAll(color, " ", "") != strings.ReplaceAll(body, " ", "") {
			add("visual_overlay_canvas_color_mismatch", "open mobile overlay and page canvas have different computed colors", "panel "+color+"; body "+body, "Use the shared overlay surface and canvas ownership; remove consumer background overrides.")
		}
	}
	return out
}
