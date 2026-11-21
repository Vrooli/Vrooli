//go:build !linux

package hostwireless

import "reflect"

func wirelessJournalPersistentTypes() []reflect.Type { return nil }
