package format

const (
	sectionSep    = "===================================================================="
	subsectionSep = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
)

var (
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorBold   = "\033[1m"
	colorReset  = "\033[0m"
)

func SetColorEnabled(enabled bool) {
	if enabled {
		colorRed = "\033[0;31m"
		colorGreen = "\033[0;32m"
		colorYellow = "\033[1;33m"
		colorBold = "\033[1m"
		colorReset = "\033[0m"
		return
	}
	colorRed = ""
	colorGreen = ""
	colorYellow = ""
	colorBold = ""
	colorReset = ""
}

func ColorEnabled() bool {
	return colorRed != "" || colorGreen != "" || colorYellow != "" || colorBold != "" || colorReset != ""
}
