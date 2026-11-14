package hub

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func shapeCodes(warnings []PacketWarning) []string {
	var codes []string
	for _, warning := range warnings {
		if warning.Code == WarnExplainerMayHelp {
			codes = append(codes, warning.Code)
		}
	}
	return codes
}

func optionsOf(labels ...string) []AskOption {
	options := make([]AskOption, 0, len(labels)+1)
	for index, label := range labels {
		options = append(options, AskOption{Key: string(rune('a' + index)), Label: label})
	}
	return append(options, AskOption{Key: "other", Label: "Other"})
}

func TestShapeAdviceSuggestsAnExplainerForAComparison(t *testing.T) {
	long := "Option with a long description of cost, time and who has to do the work"
	manyOptions := coverSpec()
	manyOptions.Options = optionsOf(long, long, long)
	longBackground := coverSpec()
	longBackground.Options = optionsOf("Yes", "No")
	longBackground.BigPicture = strings.Repeat("A sentence about the situation. ", 40)
	images := coverSpec()
	images.Options = optionsOf("Yes", "No")
	pictures := []PacketAttachment{{Name: "a-cards.png"}, {Name: "b-table.PNG"}}

	for name, tc := range map[string]struct {
		spec        AskSpec
		attachments []PacketAttachment
	}{
		"three options with long descriptions": {manyOptions, nil},
		"a long background":                    {longBackground, nil},
		"several images and no comparison":     {images, pictures},
	} {
		t.Run(name, func(t *testing.T) {
			warnings := CheckPacket(tc.spec, tc.attachments, nil)

			require.Equal(t, []string{WarnExplainerMayHelp}, shapeCodes(warnings))
			for _, warning := range warnings {
				if warning.Code == WarnExplainerMayHelp {
					require.Equal(t, WarningAdvice, warning.Severity)
					require.Contains(t, warning.Message, "--explainer")
					require.Contains(t, warning.Message, "examples/plan-tradeoffs.html")
				}
			}
		})
	}
}

func TestShapeAdviceStaysSilentOnOrdinaryDecisions(t *testing.T) {
	yesNo := coverSpec()
	yesNo.Options = optionsOf("Yes", "No")
	shortOptions := coverSpec()
	shortOptions.Options = optionsOf("Retry now", "Skip tonight", "Ask me later")
	twoLong := coverSpec()
	twoLong.Options = optionsOf(strings.Repeat("A long option label. ", 5), strings.Repeat("Another long option label. ", 5))
	oneImage := coverSpec()
	oneImage.Options = optionsOf("Yes", "No")
	variant := coverSpec()
	variant.Options = optionsOf("Yes", "No")
	variant.Questions = []AskQuestion{{ID: "layout", Type: "variant"}}
	reserved := coverSpec()
	reserved.Options = []AskOption{{Key: "other", Label: strings.Repeat("x", 80)}, {Key: "other", Label: strings.Repeat("x", 80)}, {Key: "other", Label: strings.Repeat("x", 80)}}

	for name, tc := range map[string]struct {
		spec        AskSpec
		attachments []PacketAttachment
	}{
		"yes or no":                   {yesNo, nil},
		"three short options":         {shortOptions, nil},
		"two long options":            {twoLong, nil},
		"one image":                   {oneImage, []PacketAttachment{{Name: "a.png"}}},
		"images in a variant form":    {variant, []PacketAttachment{{Name: "a.png"}, {Name: "b.png"}}},
		"only the reserved other key": {reserved, nil},
		"two non-image files":         {yesNo, []PacketAttachment{{Name: "a.md"}, {Name: "b.pdf"}}},
	} {
		t.Run(name, func(t *testing.T) {
			require.Empty(t, shapeCodes(CheckPacket(tc.spec, tc.attachments, nil)))
		})
	}
}

func TestShapeAdviceStepsAsideWhenTheAskHasAnExplainer(t *testing.T) {
	spec := coverSpec()
	long := "Option with a long description of cost, time and who has to do the work"
	spec.Options = optionsOf(long, long, long)
	page := &PacketExplainer{Field: "attachments[0]", Name: "explainer.html", SourcePath: "explainer.html", HTML: []byte(`<!doctype html><meta name="viewport" content="width=device-width"><p>x</p>`), Size: 80}

	require.Empty(t, shapeCodes(CheckPacket(spec, []PacketAttachment{{Name: "explainer.html"}}, page)))
}
