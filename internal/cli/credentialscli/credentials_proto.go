package credentialscli

// Typed credential renderers are kept beside the command boundary. The
// application package delegates all JSON output through cliout and never
// includes secret values in a response.
