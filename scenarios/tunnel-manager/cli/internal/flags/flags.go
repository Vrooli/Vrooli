package flags

func HasJSONOutput(args []string) bool {
	for _, arg := range args {
		if arg == "--json" || arg == "-j" {
			return true
		}
	}
	return false
}

func StringValue(args []string, name string) (string, bool) {
	for i, arg := range args {
		if arg == "--"+name && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

func BoolValue(args []string, name string) bool {
	for _, arg := range args {
		if arg == "--"+name {
			return true
		}
	}
	return false
}
