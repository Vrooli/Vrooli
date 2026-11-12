package grub

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// BootEntry is a generated GRUB menu leaf with an explicit stable selector.
// Selector includes enclosing submenu IDs, separated by > as GRUB expects.
type BootEntry struct {
	Selector string
	Kernel   string
	Initrd   string
}

var bootDeclaration = regexp.MustCompile(`^\s*(menuentry|submenu)\s+(['"])[^'"]+['"].*\s(?:--id|\$menuentry_id_option)\s+['"]([A-Za-z0-9_.:-]+)['"]\s*\{\s*$`)

// BootEntries parses the bounded static subset emitted by Ubuntu grub-mkconfig.
// Unknown/computed paths and malformed blocks fail closed. This is intentionally
// separate from the command-line settings parser: it never evaluates GRUB code.
func BootEntries(config []byte) ([]BootEntry, error) {
	if len(config) > 4<<20 {
		return nil, errors.New("GRUB config exceeds bound")
	}
	type block struct {
		id    string
		entry *BootEntry
	}
	var stack []block
	var entries []BootEntry
	seen := map[string]bool{}
	for _, raw := range strings.Split(string(config), "\n") {
		line := strings.TrimSpace(raw)
		if m := bootDeclaration.FindStringSubmatch(line); m != nil {
			ids := []string{}
			for _, b := range stack {
				if b.id != "" {
					ids = append(ids, b.id)
				}
			}
			ids = append(ids, m[3])
			selector := strings.Join(ids, ">")
			if seen[selector] {
				return nil, errors.New("duplicate GRUB menu selector")
			}
			seen[selector] = true
			b := block{id: m[3]}
			if m[1] == "menuentry" {
				b.entry = &BootEntry{Selector: selector}
			}
			stack = append(stack, b)
			continue
		}
		if strings.HasPrefix(line, "menuentry ") || strings.HasPrefix(line, "submenu ") {
			return nil, errors.New("unsupported GRUB menu declaration")
		}
		if line == "}" {
			if len(stack) == 0 {
				continue
			}
			b := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if b.entry != nil {
				if b.entry.Kernel != "" && b.entry.Initrd != "" {
					entries = append(entries, *b.entry)
				}
			}
			continue
		}
		if len(stack) == 0 {
			continue
		}
		b := &stack[len(stack)-1]
		if b.entry == nil {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		switch f[0] {
		case "linux", "linuxefi":
			if b.entry.Kernel != "" {
				return nil, errors.New("multiple kernel commands in GRUB entry")
			}
			b.entry.Kernel = f[1]
		case "initrd", "initrdefi":
			if len(f) != 2 || b.entry.Initrd != "" {
				return nil, errors.New("ambiguous initrd command in GRUB entry")
			}
			b.entry.Initrd = f[1]
		}
	}
	if len(stack) != 0 {
		return nil, fmt.Errorf("unclosed GRUB menu block")
	}
	return entries, nil
}
