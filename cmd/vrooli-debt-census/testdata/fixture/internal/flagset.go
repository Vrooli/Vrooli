package sample

import "flag"

func flagFixture() { _ = flag.NewFlagSet("fixture", flag.ContinueOnError) }
