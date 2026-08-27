package cli

func bindFixtureOne() {}
func bindFixtureTwo() {}

func useFixtureBinders() {
	bindFixtureOne()
	bindFixtureTwo()
}

var binderComment = "bindNotADeclaration"
