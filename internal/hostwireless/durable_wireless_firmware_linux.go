//go:build linux

package hostwireless

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
)

func durableFirmwarePins(profile string) []FileIdentity {
	if profile == NobleBluetoothProfile {
		return []FileIdentity{{Path: "lib/firmware/mediatek/mt7927/" + bluetoothFirmwareName, SHA256: bluetoothFirmwareSHA}}
	}
	return []FileIdentity{
		{Path: "lib/firmware/mediatek/mt7927/WIFI_MT6639_PATCH_MCU_2_1_hdr.bin", SHA256: "7a5aacf68ed8526c1d94ba962f84046496b3909562decf8bb690cfbd8de5b65e"},
		{Path: "lib/firmware/mediatek/mt7927/WIFI_RAM_CODE_MT6639_2_1.bin", SHA256: "e841f5ed4fe547ff2f15275a2f330bb19a253db92e3e59cbdfd8581d72b2582d"},
	}
}

func readDurableFirmware(ctx context.Context, profile string, images []FirmwarePreflight) ([]FileIdentity, map[string][]byte, error) {
	pins := durableFirmwarePins(profile)
	if len(images) == 0 {
		return nil, nil, nil
	} // Module-only proof does not imply firmware readiness.
	if len(images) != len(pins) {
		return nil, nil, errors.New("incomplete wireless firmware closure")
	}
	contents := map[string][]byte{}
	for i, pin := range pins {
		image := images[i]
		if image.ExpectedSHA256 != pin.SHA256 || image.Candidate.SHA256 != pin.SHA256 || filepath.Base(image.Candidate.Path) != path.Base(pin.Path) || !image.CandidateVerified {
			return nil, nil, errors.New("wireless firmware candidate identity differs")
		}
		root, err := os.OpenRoot(filepath.Dir(image.Candidate.Path))
		if err != nil {
			return nil, nil, err
		}
		data, err := readSmallRegular(bundleFiles{root}, filepath.Base(image.Candidate.Path), maxArtifactBytes)
		root.Close()
		if err != nil || digest(data) != pin.SHA256 {
			return nil, nil, errors.New("wireless firmware source changed")
		}
		if err = ctx.Err(); err != nil {
			return nil, nil, err
		}
		contents[pin.Path] = data
	}
	return pins, contents, nil
}
