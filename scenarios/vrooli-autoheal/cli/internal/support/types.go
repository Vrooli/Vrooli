package support

type Dependencies struct {
	RunLoop      func(args []string) error
	DiagnosePort func(args []string) error
}
