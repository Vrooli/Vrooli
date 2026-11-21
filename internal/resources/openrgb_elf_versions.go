package resources

import (
	"debug/elf"
	"encoding/binary"
	"errors"
)

func openRGBELFVersionNeeds(file *elf.File) (map[string][]string, error) {
	result := map[string][]string{}
	section := file.Section(".gnu.version_r")
	if section == nil {
		return result, nil
	}
	records, err := section.Data()
	if err != nil {
		return nil, err
	}
	stringsSection := file.Section(".dynstr")
	if stringsSection == nil {
		return nil, errors.New("ELF version references lack dynamic strings")
	}
	names, err := stringsSection.Data()
	if err != nil {
		return nil, err
	}
	if len(records) > 1<<20 || len(names) > 16<<20 {
		return nil, errors.New("ELF version metadata exceeds bound")
	}
	for offset, count := uint32(0), 0; ; count++ {
		if count > 4096 || uint64(offset)+16 > uint64(len(records)) {
			return nil, errors.New("invalid ELF version requirement chain")
		}
		r := records[offset:]
		library, err := openRGBELFString(names, binary.LittleEndian.Uint32(r[4:]))
		if err != nil {
			return nil, err
		}
		aux := offset + binary.LittleEndian.Uint32(r[8:])
		n := binary.LittleEndian.Uint16(r[2:])
		if n > 4096 {
			return nil, errors.New("excessive ELF version requirements")
		}
		for i := uint16(0); i < n; i++ {
			if uint64(aux)+16 > uint64(len(records)) {
				return nil, errors.New("truncated ELF auxiliary version")
			}
			a := records[aux:]
			version, err := openRGBELFString(names, binary.LittleEndian.Uint32(a[8:]))
			if err != nil {
				return nil, err
			}
			result[library] = append(result[library], version)
			next := binary.LittleEndian.Uint32(a[12:])
			if i+1 < n && next == 0 {
				return nil, errors.New("truncated ELF auxiliary version chain")
			}
			aux += next
		}
		next := binary.LittleEndian.Uint32(r[12:])
		if next == 0 {
			break
		}
		if next < 16 {
			return nil, errors.New("cyclic ELF version chain")
		}
		offset += next
	}
	return result, nil
}
func openRGBELFVersionDefinitions(file *elf.File) (map[string]bool, error) {
	result := map[string]bool{}
	section := file.Section(".gnu.version_d")
	if section == nil {
		return result, nil
	}
	records, err := section.Data()
	if err != nil {
		return nil, err
	}
	stringsSection := file.Section(".dynstr")
	if stringsSection == nil {
		return nil, errors.New("ELF version definitions lack dynamic strings")
	}
	names, err := stringsSection.Data()
	if err != nil {
		return nil, err
	}
	if len(records) > 1<<20 || len(names) > 16<<20 {
		return nil, errors.New("ELF version metadata exceeds bound")
	}
	for offset, count := uint32(0), 0; ; count++ {
		if count > 4096 || uint64(offset)+20 > uint64(len(records)) {
			return nil, errors.New("invalid ELF version definition chain")
		}
		r := records[offset:]
		aux := offset + binary.LittleEndian.Uint32(r[12:])
		if uint64(aux)+8 > uint64(len(records)) {
			return nil, errors.New("truncated ELF version definition")
		}
		name, err := openRGBELFString(names, binary.LittleEndian.Uint32(records[aux:]))
		if err != nil {
			return nil, err
		}
		result[name] = true
		next := binary.LittleEndian.Uint32(r[16:])
		if next == 0 {
			break
		}
		if next < 20 {
			return nil, errors.New("cyclic ELF version definitions")
		}
		offset += next
	}
	return result, nil
}
func openRGBELFString(table []byte, index uint32) (string, error) {
	if uint64(index) >= uint64(len(table)) {
		return "", errors.New("ELF version string offset is outside table")
	}
	data := table[index:]
	for i, b := range data {
		if i > 4096 {
			break
		}
		if b == 0 {
			return string(data[:i]), nil
		}
	}
	return "", errors.New("ELF version string is unbounded")
}
