package templatecontracts

import "io"

func writeScenarioDesignListJSON(w io.Writer, kits []DesignKitInfo) error {
	return marshalScenarioStatus(w, struct {
		Success    bool            `json:"success"`
		DesignKits []DesignKitInfo `json:"design_kits"`
	}{Success: true, DesignKits: kits})
}

func writeScenarioDesignShowJSON(w io.Writer, info DesignKitInfo) error {
	return marshalScenarioStatus(w, struct {
		Success   bool          `json:"success"`
		DesignKit DesignKitInfo `json:"design_kit"`
	}{Success: true, DesignKit: info})
}

func writeScenarioDesignValidateJSON(w io.Writer, report DesignValidationReport) error {
	return marshalScenarioStatus(w, struct {
		Success          bool                   `json:"success"`
		DesignValidation DesignValidationReport `json:"design_validation"`
	}{Success: true, DesignValidation: report})
}

func writeScenarioOrientationJSON(w io.Writer, report OrientationReport) error {
	return marshalScenarioStatus(w, struct {
		Success     bool              `json:"success"`
		Orientation OrientationReport `json:"orientation"`
	}{Success: true, Orientation: report})
}

func writeScenarioDetemplateJSON(w io.Writer, result DetemplateResult) error {
	return marshalScenarioStatus(w, struct {
		Success    bool             `json:"success"`
		Detemplate DetemplateResult `json:"detemplate"`
	}{Success: true, Detemplate: result})
}
