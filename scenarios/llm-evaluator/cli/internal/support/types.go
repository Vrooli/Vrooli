package support

// This file is the home for shared response shapes once domain packages are
// introduced. Add typed structs here for endpoints with stable schemas; reach
// for map[string]interface{} in the domain package only for highly variable
// or diagnostic payloads.
//
// The LLM Evaluator API currently exposes only /health (covered by
// cli-core's built-in `status` command), so no scenario-specific types are
// defined yet.
