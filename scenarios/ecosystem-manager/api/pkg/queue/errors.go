package queue

// This file previously contained custom error types that were never used in practice.
// They have been removed to reduce code complexity. The codebase uses standard
// fmt.Errorf with %w wrapping for error chains, which provides sufficient functionality.
//
// If structured errors become necessary in the future, consider using:
// - errors.Is() and errors.As() with sentinel errors
// - Custom error types with error wrapping via %w
// - Error fields in return structs for structured data
