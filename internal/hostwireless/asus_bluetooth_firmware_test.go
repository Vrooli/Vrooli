package hostwireless

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"testing"
)

func firmwareFixture() []byte {
	const first = 96 + 2*64
	data := make([]byte, first+8)
	copy(data, "20260611041233")
	copy(data[16:], "ALPS")
	binary.LittleEndian.PutUint32(data[44:], 2)
	for i := 0; i < 2; i++ {
		entry := data[96+i*64 : 96+(i+1)*64]
		binary.LittleEndian.PutUint32(entry[4:], uint32(first+i*4))
		binary.LittleEndian.PutUint32(entry[8:], 4)
		binary.LittleEndian.PutUint32(entry[16:], 4)
		mode := uint32(0x101)
		if i == 1 {
			mode = 0x20100
		}
		binary.LittleEndian.PutUint32(entry[32:], mode)
		copy(data[first+i*4:], []byte{1, 2, 3, 4})
		binary.LittleEndian.PutUint32(entry[36:], 10)
	}
	return data
}

func containerFixture(firmware []byte, count int) []byte {
	start := 16 + count*76
	data := make([]byte, start+count*len(firmware)+4)
	copy(data, "MTK-")
	binary.LittleEndian.PutUint16(data[4:], uint16(count))
	binary.LittleEndian.PutUint16(data[6:], 1)
	binary.LittleEndian.PutUint32(data[8:], uint32(len(data)))
	for i := 0; i < count; i++ {
		entry := data[16+i*76 : 16+(i+1)*76]
		copy(entry, bluetoothFirmwareName)
		copy(entry[48:], "20260611041233")
		binary.LittleEndian.PutUint32(entry[64:], uint32(start+i*len(firmware)))
		binary.LittleEndian.PutUint32(entry[68:], uint32(len(firmware)))
		copy(data[start+i*len(firmware):], firmware)
	}
	copy(data[len(data)-4:], "BIGM")
	return data
}

func TestMTKContainerAndFirmwarePreserveOpaqueImage(t *testing.T) {
	firmware := firmwareFixture()
	extracted, err := parseMTKContainer(containerFixture(firmware, 1))
	if err != nil || !bytes.Equal(extracted, firmware) {
		t.Fatalf("opaque extraction %v", err)
	}
	r, err := inspectMT6639Firmware(extracted)
	if err != nil || r.SelectedDownloadBytes != 4 || len(r.Sections) != 2 ||
		!r.Sections[0].SelectedBluetooth || r.Sections[1].SelectedBluetooth {
		t.Fatalf("inspection %+v error %v", r, err)
	}
	for _, test := range []struct {
		name   string
		mutate func([]byte)
	}{
		{"truncated table", func(data []byte) { binary.LittleEndian.PutUint16(data[4:], 64) }},
		{"wrong size", func(data []byte) { binary.LittleEndian.PutUint32(data[8:], 0) }},
		{"payload outside file", func(data []byte) { binary.LittleEndian.PutUint32(data[80:], 0xffffffff) }},
		{"invalid timestamp", func(data []byte) { copy(data[64:], "20261311041233") }},
		{"footer", func(data []byte) { data[len(data)-1] ^= 1 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := containerFixture(firmware, 1)
			test.mutate(data)
			if _, err := parseMTKContainer(data); err == nil {
				t.Fatal("accepted malformed container")
			}
		})
	}
	if _, err := parseMTKContainer(containerFixture(firmware, 2)); err == nil {
		t.Fatal("accepted duplicate requested member")
	}
}

func TestMT6639FirmwareRejectsUnsafeSections(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func([]byte)
	}{
		{"unbounded count", func(data []byte) { binary.LittleEndian.PutUint32(data[44:], 0xffffffff) }},
		{"section overlap", func(data []byte) { binary.LittleEndian.PutUint32(data[96+4:], 0) }},
		{"oversized download", func(data []byte) { binary.LittleEndian.PutUint32(data[96+16:], 5) }},
		{"wrong additive check", func(data []byte) { data[len(data)-1] ^= 1 }},
		{"no Bluetooth section", func(data []byte) { binary.LittleEndian.PutUint32(data[96+32:], 0) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := firmwareFixture()
			test.mutate(data)
			if _, err := inspectMT6639Firmware(data); err == nil {
				t.Fatal("accepted unsafe firmware section")
			}
		})
	}
}

func TestNativeASUSBluetoothArchive(t *testing.T) {
	file := os.Getenv("VROOLI_ASUS_BT_ARCHIVE")
	if file == "" {
		t.Skip("set VROOLI_ASUS_BT_ARCHIVE to the exact official archive for read-only native qualification")
	}
	payload, r, err := inspectASUSArchive(file)
	if err != nil || len(payload) != bluetoothFirmwareSize || digest(payload) != bluetoothFirmwareSHA ||
		r.SelectedDownloadBytes != 486336 || len(r.Sections) != 9 {
		t.Fatalf("native archive qualification %+v error %v", r, err)
	}
	selected := 0
	for _, section := range r.Sections {
		if section.SelectedBluetooth {
			selected++
		}
	}
	if selected != 5 {
		t.Fatalf("selected %d Bluetooth sections", selected)
	}
	data, _ := json.Marshal(r)
	t.Logf("static_format_proof=%s firmware_sha256=%s bytes=%d", data, digest(payload), len(payload))
}
