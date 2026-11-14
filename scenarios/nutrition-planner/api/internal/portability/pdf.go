package portability

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"nutrition-planner/internal/planning"
	"nutrition-planner/internal/recipe"
	"nutrition-planner/internal/shopping"
)

const unicodeFontPath = "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf"

// RecipePDF renders a print-friendly A4 recipe document.
func RecipePDF(item recipe.Recipe) ([]byte, error) {
	return RecipePDFWithPageSize(item, "A4")
}

// RecipePDFWithPageSize renders a recipe using A4 or US Letter paper.
func RecipePDFWithPageSize(item recipe.Recipe, pageSize string) ([]byte, error) {
	pdf, err := newPDF(pageSize)
	if err != nil {
		return nil, err
	}
	pdf.SetTitle(item.Name, false)
	pdf.SetFont("DejaVu", "", 18)
	pdf.CellFormat(0, 12, item.Name, "", 1, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(0, 7, "Status: "+unknownIfEmpty(item.Status), "", 1, "L", false, 0, "")
	if item.CanonicalYield != "" || item.ServingUnit != "" {
		section(pdf, "Yield", unknownIfEmpty(item.CanonicalYield)+" "+unknownIfEmpty(item.ServingUnit))
	}
	if len(item.Ingredients) > 0 {
		lines := make([]string, 0, len(item.Ingredients))
		for _, ingredient := range item.Ingredients {
			amount := unknownIfEmpty(ingredient.Amount + " " + ingredient.Unit)
			label := amount + " — " + unknownIfEmpty(ingredient.Name)
			if ingredient.Preparation != "" {
				label += " (" + ingredient.Preparation + ")"
			}
			lines = append(lines, label)
		}
		section(pdf, "Ingredients", strings.Join(lines, "\n"))
	}
	if item.Notes != "" {
		section(pdf, "Notes", item.Notes)
	}
	for _, method := range item.Methods {
		mapLines := make([]string, 0, len(method.Steps))
		for _, step := range method.Steps {
			mapLines = append(mapLines, fmt.Sprintf("%s: %s → %s; after %s", unknownIfEmpty(step.ID), unknownIfEmpty(strings.Join(step.Inputs, ", ")), unknownIfEmpty(strings.Join(step.Outputs, ", ")), unknownIfEmpty(strings.Join(step.DependsOn, ", "))))
		}
		if len(mapLines) > 0 {
			section(pdf, "Preparation map — "+unknownIfEmpty(method.Name), strings.Join(mapLines, "\n"))
		}
		pdf.SetFont("DejaVu", "B", 13)
		pdf.CellFormat(0, 9, unknownIfEmpty(method.Name), "", 1, "L", false, 0, "")
		pdf.SetFont("DejaVu", "", 10)
		for index, step := range method.Steps {
			text := fmt.Sprintf("%d. %s", index+1, unknownIfEmpty(step.Instruction))
			pdf.MultiCell(0, 5, text, "", "L", false)
			pdf.Ln(1)
		}
	}
	return finishPDF(pdf)
}

// WeeklyPDF renders the generated week and its shopping context. Open/social
// slots and unknown values are written literally, never silently omitted.
func WeeklyPDF(draft planning.Draft, lines []shopping.Line) ([]byte, error) {
	return WeeklyPDFWithPageSize(draft, lines, "A4")
}

// WeeklyPDFWithPageSize renders the generated week and shopping context using
// A4 or US Letter paper.
func WeeklyPDFWithPageSize(draft planning.Draft, lines []shopping.Line, pageSize string) ([]byte, error) {
	pdf, err := newPDF(pageSize)
	if err != nil {
		return nil, err
	}
	pdf.SetTitle("Weekly nutrition plan", false)
	pdf.SetFont("DejaVu", "B", 18)
	pdf.CellFormat(0, 12, "Weekly nutrition plan", "", 1, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(0, 7, "Objective: "+unknownIfEmpty(draft.ObjectiveVersion), "", 1, "L", false, 0, "")
	for _, occurrence := range draft.Occurrences {
		label := occurrence.Date + "  " + unknownIfEmpty(occurrence.SlotName)
		if occurrence.Mode != "" {
			label += " (" + occurrence.Mode + ")"
		}
		if occurrence.RecipeName != "" {
			label += ": " + occurrence.RecipeName
		} else {
			label += ": open slot"
		}
		pdf.MultiCell(0, 5, label, "", "L", false)
		if occurrence.Reason != "" {
			pdf.SetTextColor(90, 90, 90)
			pdf.MultiCell(0, 5, "  "+occurrence.Reason, "", "L", false)
			pdf.SetTextColor(0, 0, 0)
		}
	}
	if len(draft.Unresolved) > 0 {
		section(pdf, "Unresolved", unresolvedText(draft.Unresolved))
	}
	pdf.AddPage()
	pdf.SetFont("DejaVu", "B", 15)
	pdf.CellFormat(0, 9, "Shopping", "", 1, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	for _, line := range lines {
		pdf.MultiCell(0, 5, fmt.Sprintf("%s — need %s; stock %s; price %s", line.Label, unknownIfEmpty(line.Need), unknownIfEmpty(line.Stock), unknownIfEmpty(line.Price)), "", "L", false)
	}
	return finishPDF(pdf)
}

func newPDF(pageSize string) (*gofpdf.Fpdf, error) {
	if _, err := os.Stat(unicodeFontPath); err != nil {
		return nil, errors.New("unicode PDF font is unavailable: " + unicodeFontPath)
	}
	format := strings.ToUpper(strings.TrimSpace(pageSize))
	if format == "" {
		format = "A4"
	}
	if format != "A4" && format != "LETTER" {
		return nil, fmt.Errorf("unsupported PDF page size %q; use A4 or LETTER", pageSize)
	}
	pdf := gofpdf.New("P", "mm", format, "")
	pdf.SetMargins(16, 16, 16)
	pdf.SetFooterFunc(func() {
		pdf.SetY(-12)
		pdf.SetFont("DejaVu", "", 8)
		pdf.SetTextColor(90, 90, 90)
		pdf.CellFormat(0, 6, fmt.Sprintf("Page %d of {nb}", pdf.PageNo()), "", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	})
	pdf.AliasNbPages("")
	pdf.SetFontLocation("/usr/share/fonts/truetype/dejavu")
	pdf.AddUTF8Font("DejaVu", "", "DejaVuSans.ttf")
	pdf.AddUTF8Font("DejaVu", "B", "DejaVuSans.ttf")
	pdf.AddPage()
	pdf.SetFont("DejaVu", "", 10)
	return pdf, nil
}

func finishPDF(pdf *gofpdf.Fpdf) ([]byte, error) {
	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func section(pdf *gofpdf.Fpdf, title, text string) {
	pdf.SetFont("DejaVu", "B", 12)
	pdf.CellFormat(0, 8, title, "", 1, "L", false, 0, "")
	pdf.SetFont("DejaVu", "", 10)
	pdf.MultiCell(0, 5, text, "", "L", false)
	pdf.Ln(2)
}

func unresolvedText(items []planning.Unresolved) string {
	values := make([]string, 0, len(items))
	for _, item := range items {
		values = append(values, fmt.Sprintf("%s: %s", unknownIfEmpty(item.Date), unknownIfEmpty(item.Message)))
	}
	return strings.Join(values, "\n")
}

func unknownIfEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}
