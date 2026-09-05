// parser.go: A small, focused VT/xterm escape-sequence parser.
//
// This is NOT a complete VT500-series implementation. It handles the
// subset the scenario actually emits in production:
//
//   - Printable UTF-8 (decoded incrementally; partial-codepoint splits
//     across Feed calls are stitched).
//   - C0 controls (BEL, BS, HT, LF, CR).
//   - CSI sequences: parameters then a final byte (intermediates are
//     consumed but ignored).
//   - OSC sequences: consumed up to BEL or ST and dropped.
//   - DCS / SOS / PM / APC: consumed up to ST and dropped.
//   - Single-byte ESC sequences (e.g. \x1bc, \x1b7, \x1b8) and two-byte
//     charset designators (ESC ( B, etc., dropped).
//
// Anything else is skipped. The parser surfaces semantic events to the
// Emulator via a small handler interface so tests can drive it directly.

package terminal

import (
	"unicode/utf8"
)

// handler receives semantic events from the parser. The Emulator
// implements this interface; tests can supply a recorder.
type handler interface {
	onRune(r rune)
	onC0(b byte) // BEL/BS/HT/LF/CR
	onCSI(private bool, params []int, final byte)
	onESC(final byte) // single-byte ESC sequence (e.g. 'c' for RIS)
}

type parserState int

const (
	psGround parserState = iota
	psEsc
	psEscIntermediate // consumed one charset-designator intro; drop next byte
	psCSIEntry
	psCSIParam
	psCSIIntermediate
	psOSC // wait for ST or BEL
	psStr // DCS/SOS/PM/APC — wait for ST
	psUTF8
)

const maxParams = 16

type parser struct {
	state parserState
	h     handler

	private  bool
	params   [maxParams]int
	nparams  int
	hasParam bool

	utf8Buf  [4]byte
	utf8Len  int
	utf8Need int
}

func newParser(h handler) *parser { return &parser{h: h} }

// feed pushes raw bytes through the parser. Always consumes len(p) bytes.
func (p *parser) feed(buf []byte) {
	for i := 0; i < len(buf); i++ {
		b := buf[i]
		switch p.state {
		case psGround:
			p.handleGround(b)
		case psEsc:
			p.handleEsc(b)
		case psEscIntermediate:
			// Drop the second byte of a charset designator.
			p.state = psGround
		case psCSIEntry, psCSIParam, psCSIIntermediate:
			p.handleCSI(b)
		case psOSC:
			p.handleOSC(b)
		case psStr:
			p.handleStr(b)
		case psUTF8:
			p.handleUTF8(b)
		}
	}
}

func (p *parser) handleGround(b byte) {
	switch {
	case b == 0x1b:
		p.state = psEsc
	case b == 0x07, b == 0x08, b == 0x09, b == 0x0a, b == 0x0d:
		p.h.onC0(b)
	case b < 0x20:
		// Other C0: ignore.
	case b == 0x7f:
		// DEL: ignore.
	case b < 0x80:
		p.h.onRune(rune(b))
	default:
		need := 0
		switch {
		case b&0xe0 == 0xc0:
			need = 1
		case b&0xf0 == 0xe0:
			need = 2
		case b&0xf8 == 0xf0:
			need = 3
		default:
			p.h.onRune(utf8.RuneError)
			return
		}
		p.utf8Buf[0] = b
		p.utf8Len = 1
		p.utf8Need = need
		p.state = psUTF8
	}
}

func (p *parser) handleUTF8(b byte) {
	if b&0xc0 != 0x80 {
		p.h.onRune(utf8.RuneError)
		p.utf8Len, p.utf8Need = 0, 0
		p.state = psGround
		p.handleGround(b)
		return
	}
	p.utf8Buf[p.utf8Len] = b
	p.utf8Len++
	p.utf8Need--
	if p.utf8Need == 0 {
		r, _ := utf8.DecodeRune(p.utf8Buf[:p.utf8Len])
		p.h.onRune(r)
		p.utf8Len = 0
		p.state = psGround
	}
}

func (p *parser) handleEsc(b byte) {
	switch b {
	case '[':
		p.resetCSI()
		p.state = psCSIEntry
	case ']':
		p.state = psOSC
	case 'P', 'X', '^', '_':
		p.state = psStr
	case '(', ')', '*', '+':
		p.state = psEscIntermediate
	default:
		p.h.onESC(b)
		p.state = psGround
	}
}

func (p *parser) resetCSI() {
	p.private = false
	p.nparams = 0
	p.hasParam = false
	for i := range p.params {
		p.params[i] = 0
	}
}

func (p *parser) handleCSI(b byte) {
	switch {
	case p.state == psCSIEntry && (b == '?' || b == '>' || b == '<' || b == '='):
		p.private = true
		p.state = psCSIParam
	case b >= '0' && b <= '9':
		if p.state == psCSIEntry {
			p.state = psCSIParam
		}
		if p.nparams < maxParams {
			p.params[p.nparams] = p.params[p.nparams]*10 + int(b-'0')
			p.hasParam = true
		}
	case b == ';':
		if p.state == psCSIEntry {
			p.state = psCSIParam
		}
		if p.nparams < maxParams-1 {
			p.nparams++
			p.params[p.nparams] = 0
		}
	case b >= 0x20 && b <= 0x2f:
		p.state = psCSIIntermediate
	case b >= 0x40 && b <= 0x7e:
		nparams := p.nparams
		if p.hasParam || nparams > 0 {
			nparams++
		}
		p.h.onCSI(p.private, p.params[:nparams], b)
		p.state = psGround
	default:
		p.state = psGround
	}
}

func (p *parser) handleOSC(b byte) {
	if b == 0x07 {
		p.state = psGround
		return
	}
	if b == 0x1b {
		p.state = psStr
		return
	}
}

func (p *parser) handleStr(b byte) {
	if b == 0x1b {
		return
	}
	if b == '\\' || b == 0x07 {
		p.state = psGround
		return
	}
}
