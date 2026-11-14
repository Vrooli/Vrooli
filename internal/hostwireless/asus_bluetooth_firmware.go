package hostwireless

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/vrooli/internal/hostinventory"
	mt7927source "github.com/vrooli/vrooli/internal/tools/mt7927-source"
)

const (
	asusBTMember             = "BT/mtkbt_v2.dat"
	bluetoothFirmwareName    = "BT_RAM_CODE_MT6639_2_1_hdr.bin"
	bluetoothFirmwareSHA     = "2135f2c4220cfa6e8eb9fdf430517098c13b862a95844bbff0153240a768efa8"
	bluetoothFirmwareSize    = 571349
	bluetoothPublicationPeak = 2 << 20
)

type BluetoothFirmwareSection struct {
	Index             int    `json:"index"`
	Offset            uint32 `json:"offset"`
	Bytes             uint32 `json:"bytes"`
	DownloadBytes     uint32 `json:"download_bytes"`
	Mode              uint32 `json:"mode"`
	SelectedBluetooth bool   `json:"selected_bluetooth"`
}

type bluetoothFirmwareLayout struct {
	ContainerSHA256       string                     `json:"container_sha256,omitempty"`
	BuildTime             string                     `json:"build_time"`
	SelectedDownloadBytes uint64                     `json:"selected_download_bytes"`
	Sections              []BluetoothFirmwareSection `json:"sections"`
}

// BluetoothFirmwareQualification records local static format and provenance.
// It carries no assertion of controller initialization or runtime compatibility.
type BluetoothFirmwareQualification struct {
	SchemaVersion           int                        `json:"schema_version"`
	Profile                 string                     `json:"profile"`
	State                   string                     `json:"state"`
	Reason                  string                     `json:"reason,omitempty"`
	ArchiveSHA256           string                     `json:"archive_sha256"`
	ArchiveProfileSHA256    string                     `json:"archive_profile_sha256,omitempty"`
	ContainerSHA256         string                     `json:"container_sha256,omitempty"`
	FirmwareSHA256          string                     `json:"firmware_sha256,omitempty"`
	FirmwareName            string                     `json:"firmware_name"`
	FirmwareRelativePath    string                     `json:"firmware_relative_path"`
	FirmwareBytes           int                        `json:"firmware_bytes,omitempty"`
	BuildTime               string                     `json:"build_time,omitempty"`
	SelectedDownloadBytes   uint64                     `json:"selected_download_bytes,omitempty"`
	Sections                []BluetoothFirmwareSection `json:"sections,omitempty"`
	LocalUseBasis           string                     `json:"local_use_basis"`
	LinuxUseTerms           string                     `json:"linux_use_terms"`
	RedistributionQualified bool                       `json:"redistribution_qualified"`
	ActivationEligible      bool                       `json:"activation_eligible"`
	ArtifactDirectory       string                     `json:"artifact_directory,omitempty"`
	PublicationPeakBytes    int64                      `json:"publication_peak_bytes"`
}

// QualifyBluetoothFirmware consumes the closed SDA archive profile. Apply
// publishes an exact opaque image and its receipt under the local artifact root.
func QualifyBluetoothFirmware(ctx context.Context, apply bool) (BluetoothFirmwareQualification, error) {
	r := BluetoothFirmwareQualification{
		SchemaVersion: 1, Profile: NobleBluetoothProfile, State: "unverified", ArchiveSHA256: mt7927source.ASUSBluetoothVersion,
		FirmwareName:         bluetoothFirmwareName,
		FirmwareRelativePath: "mediatek/mt7927/" + bluetoothFirmwareName,
		PublicationPeakBytes: bluetoothPublicationPeak,
		LocalUseBasis:        "Official ASUS package for this board and USB device, requested by the hardware owner. Vendor documentation supports archive extraction.",
		LinuxUseTerms:        "Package-specific Linux-use entitlement remains uncertain. No EULA or local-extraction prohibition was found in the inspected package and setup resources.",
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		r.State, r.Reason = "unsupported", "this operator-local vendor archive profile requires Linux amd64"
		return firmwareInspection(r, apply)
	}
	inventory, err := hostinventory.NewIntegrityCollector(hostinventory.IntegrityCollectorOptions{}).Collect(ctx)
	if err != nil {
		return r, err
	}
	observed := Receipt{Profile: NobleBluetoothProfile}
	selectBluetoothDevice(&observed, inventory)
	if observed.State != "eligible" {
		r.State, r.Reason = observed.State, observed.Reason
		return firmwareInspection(r, apply)
	}
	vendor, vendorErr := os.ReadFile("/sys/devices/virtual/dmi/id/board_vendor")
	board, boardErr := os.ReadFile("/sys/devices/virtual/dmi/id/board_name")
	if vendorErr != nil || boardErr != nil {
		r.Reason = "ASUS board identity unavailable"
		return firmwareInspection(r, apply)
	}
	if string(bytes.TrimSpace(vendor)) != "ASUSTeK COMPUTER INC." || string(bytes.TrimSpace(board)) != "ProArt X870E-CREATOR WIFI" {
		r.State, r.Reason = "unsupported", "this vendor archive profile targets the observed ASUS ProArt X870E-CREATOR WIFI"
		return firmwareInspection(r, apply)
	}
	profile, _ := mt7927source.Lookup(mt7927source.ASUSBluetoothName)
	stage, err := profile.StagedPath()
	if err != nil {
		return r, err
	}
	if err := profile.VerifyStaged(ctx, stage); err != nil {
		r.State, r.Reason = "source_required", fmt.Sprintf("stage SDA source/%s@%s: %v", mt7927source.ASUSBluetoothName, profile.Version(), err)
		return firmwareInspection(r, apply)
	}
	r.ArchiveProfileSHA256, err = binaryfetch.FileDigestContext(ctx, filepath.Join(stage, mt7927source.Manifest))
	if err != nil {
		return r, err
	}
	payload, metadata, err := inspectASUSArchive(filepath.Join(stage, mt7927source.ASUSBluetoothArchive))
	if err != nil {
		return r, err
	}
	r.ContainerSHA256, r.FirmwareSHA256, r.FirmwareBytes = metadata.ContainerSHA256, digest(payload), len(payload)
	r.BuildTime, r.SelectedDownloadBytes, r.Sections = metadata.BuildTime, metadata.SelectedDownloadBytes, metadata.Sections
	if r.FirmwareSHA256 != bluetoothFirmwareSHA || len(payload) != bluetoothFirmwareSize {
		return r, errors.New("extracted MT6639 firmware differs from the reviewed archive member")
	}
	r.State = "statically_qualified"
	if !apply {
		return r, nil
	}
	if os.Geteuid() == 0 {
		return r, errors.New("local firmware qualification publishes artifacts as an unprivileged user")
	}
	published, err := publishBluetoothFirmware(ctx, r, payload, stage)
	if err != nil {
		if published.State == localFirmwareArtifactState {
			published.State, published.Reason = "staged_local_cleanup_unverified", err.Error()
			return published, err
		}
		r.State, r.Reason = "publication_failed", err.Error()
		return r, err
	}
	return published, nil
}

func firmwareInspection(r BluetoothFirmwareQualification, apply bool) (BluetoothFirmwareQualification, error) {
	if apply {
		return r, errors.New(r.Reason)
	}
	return r, nil
}

func inspectASUSArchive(file string) ([]byte, bluetoothFirmwareLayout, error) {
	var r bluetoothFirmwareLayout
	root, err := os.OpenRoot(filepath.Dir(file))
	if err != nil {
		return nil, r, err
	}
	defer root.Close()
	data, err := readSmallRegular(bundleFiles{root}, filepath.Base(file), 8<<20)
	if err != nil {
		return nil, r, err
	}
	if len(data) != 7142068 || digest(data) != mt7927source.ASUSBluetoothVersion {
		return nil, r, errors.New("ASUS archive bytes differ from the official pin")
	}
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, r, err
	}
	if len(archive.File) > 64 {
		return nil, r, errors.New("ASUS archive exceeds member bound")
	}
	var container []byte
	for _, entry := range archive.File {
		if entry.Name != asusBTMember {
			continue
		}
		if container != nil || !entry.Mode().IsRegular() || entry.UncompressedSize64 > 8<<20 {
			return nil, r, errors.New("ambiguous or oversized ASUS Bluetooth container")
		}
		reader, err := entry.Open()
		if err != nil {
			return nil, r, err
		}
		container, err = io.ReadAll(io.LimitReader(reader, (8<<20)+1))
		err = errors.Join(err, reader.Close())
		if err != nil || len(container) > 8<<20 {
			return nil, r, errors.Join(errors.New("ASUS Bluetooth container read failed"), err)
		}
	}
	if container == nil {
		return nil, r, errors.New("ASUS archive lacks the exact Bluetooth container")
	}
	payload, err := parseMTKContainer(container)
	if err != nil {
		return nil, r, err
	}
	r, err = inspectMT6639Firmware(payload)
	r.ContainerSHA256 = digest(container)
	return payload, r, err
}

func parseMTKContainer(data []byte) ([]byte, error) {
	if len(data) < 20 || string(data[:4]) != "MTK-" || string(data[len(data)-4:]) != "BIGM" ||
		binary.LittleEndian.Uint16(data[6:]) != 1 || uint64(binary.LittleEndian.Uint32(data[8:])) != uint64(len(data)) ||
		binary.LittleEndian.Uint32(data[12:]) != 0 {
		return nil, errors.New("unsupported MTK container header or footer")
	}
	count := int(binary.LittleEndian.Uint16(data[4:]))
	if count < 1 || count > 64 || 16+count*76 > len(data)-4 {
		return nil, errors.New("MTK container entry count is out of bounds")
	}
	expectedOffset := 16 + count*76
	seen := map[string]bool{}
	var firmware []byte
	for i := 0; i < count; i++ {
		entry := data[16+i*76 : 16+(i+1)*76]
		nul := bytes.IndexByte(entry[:48], 0)
		if nul < 1 || !zeroBytes(entry[nul:48]) {
			return nil, errors.New("invalid MTK entry name")
		}
		name := string(entry[:nul])
		if seen[name] {
			return nil, errors.New("duplicate MTK entry name")
		}
		seen[name] = true
		stamp := bytes.TrimRight(entry[48:64], "\x00")
		if len(stamp) != 0 && !numericTimestamp(stamp) {
			return nil, errors.New("invalid MTK entry timestamp")
		}
		offset, size := uint64(binary.LittleEndian.Uint32(entry[64:])), uint64(binary.LittleEndian.Uint32(entry[68:]))
		if offset != uint64(expectedOffset) || size == 0 || offset > uint64(len(data)-4) || size > uint64(len(data)-4)-offset ||
			binary.LittleEndian.Uint32(entry[72:]) != 0 {
			return nil, errors.New("MTK entry payload bounds or ordering mismatch")
		}
		expectedOffset = int(offset + size)
		if name == bluetoothFirmwareName {
			firmware = data[int(offset):expectedOffset]
		}
	}
	if expectedOffset != len(data)-4 || firmware == nil {
		return nil, errors.New("MTK container coverage or requested firmware entry mismatch")
	}
	return firmware, nil
}

func inspectMT6639Firmware(data []byte) (bluetoothFirmwareLayout, error) {
	var r bluetoothFirmwareLayout
	if len(data) < 96 || !numericTimestamp(bytes.TrimRight(data[:16], "\x00")) || string(data[16:20]) != "ALPS" {
		return r, errors.New("invalid MT6639 firmware header")
	}
	count := uint64(binary.LittleEndian.Uint32(data[44:]))
	if count == 0 || count > 64 || 96+count*64 > uint64(len(data)) {
		return r, errors.New("MT6639 section table is out of bounds")
	}
	previous := uint64(96) + count*64
	for i := uint64(0); i < count; i++ {
		entry := data[96+i*64 : 96+(i+1)*64]
		offset, size, download := uint64(binary.LittleEndian.Uint32(entry[4:])), uint64(binary.LittleEndian.Uint32(entry[8:])), uint64(binary.LittleEndian.Uint32(entry[16:]))
		mode, checksum := binary.LittleEndian.Uint32(entry[32:]), binary.LittleEndian.Uint32(entry[36:])
		if offset != previous || offset > uint64(len(data)) || size > uint64(len(data))-offset || download > size {
			return r, errors.New("MT6639 section payload bounds or ordering mismatch")
		}
		previous = offset + size
		if download > 0 {
			var sum uint32
			for _, value := range data[offset : offset+download] {
				sum += uint32(value)
			}
			if sum != checksum {
				return r, errors.New("MT6639 section additive check differs")
			}
		}
		selected := download > 0 && mode&0xff == 1
		r.Sections = append(r.Sections, BluetoothFirmwareSection{Index: int(i), Offset: uint32(offset), Bytes: uint32(size), DownloadBytes: uint32(download), Mode: mode, SelectedBluetooth: selected})
		if selected {
			r.SelectedDownloadBytes += download
		}
	}
	if previous != uint64(len(data)) || r.SelectedDownloadBytes == 0 {
		return r, errors.New("MT6639 section coverage or Bluetooth section selection mismatch")
	}
	r.BuildTime = string(bytes.TrimRight(data[:16], "\x00"))
	return r, nil
}

func zeroBytes(data []byte) bool {
	for _, value := range data {
		if value != 0 {
			return false
		}
	}
	return true
}

func numericTimestamp(data []byte) bool {
	if len(data) != 14 {
		return false
	}
	for _, value := range data {
		if value < '0' || value > '9' {
			return false
		}
	}
	_, err := time.Parse("20060102150405", string(data))
	return err == nil
}
