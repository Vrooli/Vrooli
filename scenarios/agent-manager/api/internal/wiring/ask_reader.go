package wiring

// askReaderSource is agent-manager's name on the decisions it raises as events,
// and so the asks Notification Hub lets its ask-reader principal read back. The
// resolution webhook reads each answer back as that principal before applying it,
// and tells the operator when one is refused (see hubidentity).
const askReaderSource = "agent-manager"
