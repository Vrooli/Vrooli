package smoke

import "time"

// DefaultTimeout is the overall smoke test timeout.
const DefaultTimeout = 90 * time.Second

// DefaultHandshakeTimeout is the maximum wait for the iframe-bridge handshake.
const DefaultHandshakeTimeout = 15 * time.Second

// DefaultViewportWidth is the default browser viewport width.
const DefaultViewportWidth = 1280

// DefaultViewportHeight is the default browser viewport height.
const DefaultViewportHeight = 720
