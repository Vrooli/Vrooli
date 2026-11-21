package resources

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
)

func (p openRGBReleaseProfile) verifyVendorStartupMode(path string) error {
	if p.OS != "windows" {
		return nil
	}
	image, err := pe.Open(path)
	if err != nil {
		return err
	}
	defer image.Close()
	header, ok := image.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || header.NumberOfRvaAndSizes < 3 {
		return errors.New("native Windows release lacks its startup manifest")
	}
	rootRVA := header.DataDirectory[2].VirtualAddress
	table, err := openRGBPEResourceBytes(image, rootRVA, 16)
	if err != nil {
		return err
	}
	typeEntry, err := openRGBPEResourceEntry(table, 0, 24)
	if err != nil {
		return err
	}
	if typeEntry&0x80000000 == 0 {
		return errors.New("Windows manifest resource has no type directory")
	}
	nameEntry, err := openRGBPEResourceEntry(table, typeEntry&0x7fffffff, 1)
	if err != nil {
		return err
	}
	if nameEntry&0x80000000 == 0 {
		return errors.New("Windows process manifest resource has no name directory")
	}
	offset := nameEntry & 0x7fffffff
	if uint64(offset)+16 > uint64(len(table)) {
		return errors.New("truncated Windows manifest language directory")
	}
	count := uint32(binary.LittleEndian.Uint16(table[offset+12:])) + uint32(binary.LittleEndian.Uint16(table[offset+14:]))
	if count == 0 || count > 256 || uint64(offset)+16+uint64(count)*8 > uint64(len(table)) {
		return errors.New("invalid Windows manifest language bound")
	}
	for i := uint32(0); i < count; i++ {
		entry := table[offset+16+i*8:]
		location := binary.LittleEndian.Uint32(entry[4:])
		if location&0x80000000 != 0 || uint64(location)+16 > uint64(len(table)) {
			return errors.New("invalid Windows process manifest data entry")
		}
		rva := binary.LittleEndian.Uint32(table[location:])
		size := binary.LittleEndian.Uint32(table[location+4:])
		if size == 0 || size > 1<<20 {
			return errors.New("Windows process manifest exceeds bound")
		}
		data, err := openRGBPEResourceBytes(image, rva, size)
		if err != nil {
			return err
		}
		if err := verifyOpenRGBAsInvokerXML(data[:size]); err != nil {
			return err
		}
	}
	return nil
}
func openRGBPEResourceEntry(table []byte, offset, id uint32) (uint32, error) {
	if uint64(offset)+16 > uint64(len(table)) {
		return 0, errors.New("truncated native resource directory")
	}
	count := uint32(binary.LittleEndian.Uint16(table[offset+12:])) + uint32(binary.LittleEndian.Uint16(table[offset+14:]))
	if count > 4096 || uint64(offset)+16+uint64(count)*8 > uint64(len(table)) {
		return 0, errors.New("invalid native resource directory bound")
	}
	for i := uint32(0); i < count; i++ {
		entry := table[offset+16+i*8:]
		name := binary.LittleEndian.Uint32(entry)
		if name == id {
			return binary.LittleEndian.Uint32(entry[4:]), nil
		}
	}
	return 0, fmt.Errorf("native Windows manifest resource %d is missing", id)
}
func openRGBPEResourceBytes(image *pe.File, rva, size uint32) ([]byte, error) {
	for _, section := range image.Sections {
		if rva < section.VirtualAddress || rva-section.VirtualAddress >= section.Size {
			continue
		}
		if section.Size > 16<<20 {
			return nil, errors.New("native resource section exceeds bound")
		}
		data, err := section.Data()
		if err != nil {
			return nil, err
		}
		offset := rva - section.VirtualAddress
		if uint64(offset)+uint64(size) > uint64(len(data)) {
			return nil, errors.New("native resource data lies outside section")
		}
		return data[offset:], nil
	}
	return nil, errors.New("native process manifest is outside the image")
}
func verifyOpenRGBAsInvokerXML(data []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	found := 0
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "requestedExecutionLevel" {
			continue
		}
		if start.Name.Space != "urn:schemas-microsoft-com:asm.v3" {
			return errors.New("unqualified Windows execution-level namespace")
		}
		found++
		level, access := "", ""
		for _, attribute := range start.Attr {
			switch attribute.Name.Local {
			case "level":
				level = attribute.Value
			case "uiAccess":
				access = attribute.Value
			}
		}
		if level != "asInvoker" || access != "false" {
			return errors.New("the pinned native Windows release does not permit a narrow noninteractive asInvoker launch")
		}
	}
	if found != 1 {
		return errors.New("native Windows startup execution level is ambiguous")
	}
	return nil
}
