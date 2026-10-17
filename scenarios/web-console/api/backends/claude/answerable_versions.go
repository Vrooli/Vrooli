package claude

// answerVerifiedVersions are the Claude Code versions whose prompt boxes are
// captured in testdata/prompts and whose answers typed from Messages (the
// option's number, then Enter while the box is still up) were verified live.
// A version not listed stays read-only (level 2) even with answering on.
var answerVerifiedVersions = map[string]bool{
	"2.1.268": true,
}

// AnswerVerified reports whether prompts from this Claude Code version may be
// answered with keystrokes.
func AnswerVerified(version string) bool {
	return answerVerifiedVersions[version]
}
