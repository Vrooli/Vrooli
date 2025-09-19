package rules

/*
Rule: Exit Code Testing
Description: Tests exit code handling in test runner
Reason: Ensures proper test evaluation with exit codes
Category: test
Severity: medium
Standard: testing-v1

<test-case id="clean-code-passes" should-fail="false">
  <description>Clean code that compiles and runs successfully</description>
  <input language="go">
func simpleFunction() string {
    return "Hello World"
}
  </input>
</test-case>

<test-case id="compilation-error" should-fail="false">
  <description>Code with compilation error should fail even with no violations</description>
  <input language="go">
func brokenCode() {
    this is not valid Go code
    return undefined_variable
}
  </input>
</test-case>

<test-case id="runtime-panic" should-fail="false">
  <description>Code that panics at runtime should fail</description>
  <input language="go">
func panicCode() {
    panic("deliberate panic")
}
  </input>
</test-case>
*/

// CheckExitCode is a dummy check function for testing
func CheckExitCode(content []byte, filePath string) []Violation {
    // This rule doesn't actually check for violations
    // It's used to test exit code handling
    return []Violation{}
}