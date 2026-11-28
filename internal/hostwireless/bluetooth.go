package hostwireless

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/vrooli/vrooli/internal/hostinventory"
)

func selectBluetoothDevice(r *Receipt, inventory hostinventory.HostInventory) {
	for _, device := range inventory.Devices {
		if device.BusType != "usb" || device.VendorID != "0489" || device.DeviceID != "e13a" {
			continue
		}
		r.USBDevices = append(r.USBDevices, device.ID)
		for _, function := range device.Functions {
			alias := strings.ToLower(function.Modalias)
			if strings.HasPrefix(alias, "usb:v0489pe13ad") && strings.HasSuffix(alias, "ice0isc01ip01in00") {
				r.Modaliases = append(r.Modaliases, function.Modalias)
			}
		}
	}
	slices.Sort(r.USBDevices)
	slices.Sort(r.Modaliases)
	r.Modaliases = slices.Compact(r.Modaliases)
	switch {
	case len(r.Modaliases) > 0:
		r.State, r.Reason = "eligible", ""
	case len(r.USBDevices) > 0 || inventory.ProbeStatus["usbDevices"] != hostinventory.IntegrityProbeOK:
		r.State, r.Reason = "unverified", "MT6639 USB HCI interface modalias could not be verified"
	default:
		r.State, r.Reason = "unsupported", "USB device 0489:e13a was not observed"
	}
}

func verifyBluetoothQuirk(work, bundle string) error {
	stock, err := elfSymbol(filepath.Join(work, "stock", "btusb.ko"), "quirks_table", elf.STB_LOCAL)
	if err != nil {
		return err
	}
	candidate, err := elfSymbol(filepath.Join(bundle, "btusb.ko"), "quirks_table", elf.STB_LOCAL)
	if err != nil {
		return err
	}
	if err := compareBluetoothQuirk(stock, candidate); err != nil {
		return err
	}
	if _, err := elfSymbol(filepath.Join(bundle, "btmtk.ko"), "__btmtk_setup_firmware_79xx", elf.STB_LOCAL); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(bundle, "quirk-proof.txt"), []byte("Only 0489:e13a added to btusb quirks_table with vendor/product match and MediaTek plus wideband flags. The btmtk implementation remains local.\n"), 0600)
}

func compareBluetoothQuirk(stock, candidate []byte) error {
	const entrySize = 32
	if len(stock)%entrySize != 0 || len(candidate) != len(stock)+entrySize {
		return fmt.Errorf("unexpected btusb quirk table size")
	}
	found := -1
	for i := 0; i < len(candidate); i += entrySize {
		entry := candidate[i : i+entrySize]
		if binary.LittleEndian.Uint16(entry[2:]) != 0x0489 || binary.LittleEndian.Uint16(entry[4:]) != 0xe13a {
			continue
		}
		expected := [entrySize]byte{0x03, 0x00, 0x89, 0x04, 0x3a, 0xe1, 26: 0x30}
		if found != -1 || !bytes.Equal(entry, expected[:]) {
			return fmt.Errorf("unexpected flags or duplicate MT6639 quirk")
		}
		found = i
	}
	if found == -1 || !bytes.Equal(stock[:found], candidate[:found]) || !bytes.Equal(stock[found:], candidate[found+entrySize:]) {
		return fmt.Errorf("btusb quirks differ outside the scoped MT6639 entry")
	}
	return nil
}

func elfSymbol(path, name string, binding elf.SymBind) ([]byte, error) {
	file, err := elf.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if file.Type != elf.ET_REL || file.Machine != elf.EM_X86_64 || file.Data != elf.ELFDATA2LSB {
		return nil, fmt.Errorf("unexpected module ELF format in %s", path)
	}
	symbols, err := file.Symbols()
	if err != nil {
		return nil, err
	}
	for _, symbol := range symbols {
		if symbol.Name != name || elf.ST_BIND(symbol.Info) != binding || symbol.Size == 0 ||
			symbol.Section == elf.SHN_UNDEF || int(symbol.Section) >= len(file.Sections) {
			continue
		}
		data, err := file.Sections[symbol.Section].Data()
		if err != nil {
			return nil, err
		}
		if symbol.Value <= uint64(len(data)) && symbol.Size <= uint64(len(data))-symbol.Value {
			return data[symbol.Value : symbol.Value+symbol.Size], nil
		}
	}
	return nil, fmt.Errorf("verified symbol %s unavailable in %s", name, path)
}
