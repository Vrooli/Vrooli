package dochealth

import "fmt"

func scanAbsolutePath(file, line, trim string, lineNum int, cfg effective, findings *[]Finding, summary *fileMetrics) {
	scanLine := lineForAbsolutePathScan(line, lineNum)
	var absMatch string
	switch {
	case absUnixPathPattern.MatchString(scanLine):
		absMatch = absUnixPathPattern.FindString(scanLine)
	case absWindowsPathPattern.MatchString(scanLine):
		absMatch = absWindowsPathPattern.FindString(scanLine)
	}
	if absMatch == "" {
		return
	}
	summary.AbsoluteHits++
	if allowedPrefix(absMatch, cfg.pathAllow) {
		return
	}
	summary.AbsoluteFailures++
	*findings = append(*findings, Finding{
		Code:     "absolute_path",
		Severity: SeverityFailure,
		Message:  fmt.Sprintf("%s:%d contains absolute filesystem path", file, lineNum),
		Path:     file,
		Line:     lineNum,
		Target:   absMatch,
	})
	_ = trim
}
