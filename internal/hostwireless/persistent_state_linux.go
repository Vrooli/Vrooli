//go:build linux

package hostwireless

import "reflect"

func wirelessJournalPersistentTypes() []reflect.Type {
	return []reflect.Type{reflect.TypeFor[durableWirelessIntent](), reflect.TypeFor[durableActive](), reflect.TypeFor[wirelessRadioRecord]()}
}
