package hub

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// The skill's worked examples are the reference for a good explainer, so the
// packet check must accept every one of them. Each documents the command that
// sends it; the files that command attaches (an attachment's name is its file
// name, and --packet attaches the packet's BRIEF.md) are the ones the page may
// refer to.
var (
	attachFlagPattern = regexp.MustCompile(`--attach\s+(\S+)`)
	packetFlagPattern = regexp.MustCompile(`--packet\s+\S+`)
)

func TestSkillExampleExplainersPassThePacketCheck(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "skills", "notification-hub", "examples")
	files, err := filepath.Glob(filepath.Join(dir, "*.html"))
	require.NoError(t, err)
	require.NotEmpty(t, files, "the skill ships example explainers")
	for _, file := range files {
		name := filepath.Base(file)
		t.Run(name, func(t *testing.T) {
			page, err := os.ReadFile(file)
			require.NoError(t, err)
			attachments := []PacketAttachment{{Name: name}}
			for _, match := range attachFlagPattern.FindAllStringSubmatch(string(page), -1) {
				attachments = append(attachments, PacketAttachment{Name: filepath.Base(match[1])})
			}
			if packetFlagPattern.Match(page) {
				attachments = append(attachments, PacketAttachment{Name: "BRIEF.md"})
			}
			explainer := &PacketExplainer{Field: "attachments[0]", Name: name, SourcePath: "/tmp/" + name, Size: int64(len(page)), HTML: page}
			for _, warning := range checkExplainer(*explainer, namesOf(attachments)) {
				t.Errorf("%s: %s: %s", name, warning.Code, warning.Message)
			}
		})
	}
}

func namesOf(attachments []PacketAttachment) map[string]bool {
	names := make(map[string]bool, len(attachments))
	for _, attachment := range attachments {
		names[attachment.Name] = true
	}
	return names
}
