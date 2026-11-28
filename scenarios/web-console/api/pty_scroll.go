package main

import "errors"

// errScrollNoHistory reports that a back-scroll had nothing to reveal: a
// normal-screen pane that holds zero lines above its viewport (a freshly
// spawned shell). It is distinct from a genuine backend failure so the client
// can show a "no scrollback" hint without latching scroll off for the whole
// session — scroll re-enables the moment the pane accumulates real history.
var errScrollNoHistory = errors.New("scroll: pane has no history to reveal")

// errScrollForwardedToApp reports that a back-scroll could not use tmux history
// because the pane runs a full-screen program on the alternate screen (every
// agent TUI we host), but the gesture was instead forwarded to that program as
// wheel input so it scrolls its own transcript. The pane did move; the client
// surfaces a one-time pointer to the Messages view, which holds the complete
// transcript the in-program scrollback may not. It is distinct from
// errScrollNoHistory so the client does not claim "nothing happened" when the
// terminal actually scrolled.
var errScrollForwardedToApp = errors.New("scroll: forwarded to full-screen program")
