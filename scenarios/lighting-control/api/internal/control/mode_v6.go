package control

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
)

const (
	updateModePacketV6     uint32 = 1101
	updateZoneModePacketV6 uint32 = 1103

	sdkModeHasSpeed             uint32 = 1 << 0
	sdkModeHasBrightness        uint32 = 1 << 4
	sdkModeHasPerLEDColor       uint32 = 1 << 5
	sdkModeHasModeSpecificColor uint32 = 1 << 6
	sdkModeAutomaticallySaves   uint32 = 1 << 9
	sdkModeRequiresEntireDevice uint32 = 1 << 10

	sdkModeColorsNone         uint32 = 0
	sdkModeColorsPerLED       uint32 = 1
	sdkModeColorsModeSpecific uint32 = 2

	maxSDKModeColors = 4096
)

// SDKModeParameters follows the pinned v6 GetModeDescriptionData field order.
// Some drivers use reversed range endpoints. Preserve their advertised order.
type SDKModeParameters struct {
	Flags         uint32
	SpeedMin      uint32
	SpeedMax      uint32
	BrightnessMin uint32
	BrightnessMax uint32
	ColorsMin     uint32
	ColorsMax     uint32
	Speed         uint32
	Brightness    uint32
	Direction     uint32
	ColorMode     uint32
}

type SDKMode struct {
	Name string
	SDKModeParameters
	Colors []uint32 // OpenRGB packed RGBColor values
}

type SDKModeTarget struct {
	ModeIndex uint32
	ZoneIndex *uint32 // nil selects a controller mode
}

func decodeSDKModeV6(c *cursor) (SDKMode, error) {
	var mode SDKMode
	var err error
	mode.Name, err = c.str16()
	if err != nil {
		return SDKMode{}, err
	}
	data, err := c.take(uint64(binary.Size(mode.SDKModeParameters)))
	if err != nil {
		return SDKMode{}, err
	}
	if _, err = binary.Decode(data, binary.LittleEndian, &mode.SDKModeParameters); err != nil {
		return SDKMode{}, err
	}
	count, err := c.u16()
	if err != nil {
		return SDKMode{}, err
	}
	if count > maxSDKModeColors {
		return SDKMode{}, fmt.Errorf("invalid mode color count %d", count)
	}
	data, err = c.take(uint64(count) * 4)
	if err != nil {
		return SDKMode{}, err
	}
	if count > 0 {
		mode.Colors = make([]uint32, count)
		if _, err = binary.Decode(data, binary.LittleEndian, mode.Colors); err != nil {
			return SDKMode{}, err
		}
	}
	return mode, nil
}

// encodeSDKModeUpdateV6 builds the transient UpdateMode or UpdateZoneMode body.
// The caller resolves scope and mode from the current controller snapshot.
func encodeSDKModeUpdateV6(target SDKModeTarget, mode SDKMode) (uint32, []byte, error) {
	if target.ModeIndex > math.MaxInt32 || target.ZoneIndex != nil && *target.ZoneIndex > math.MaxInt32 {
		return 0, nil, errors.New("OpenRGB mode or zone index is out of range")
	}
	if len(mode.Name) == 0 || len(mode.Name) >= math.MaxUint16 || strings.ContainsRune(mode.Name, 0) {
		return 0, nil, errors.New("invalid OpenRGB mode name")
	}
	if len(mode.Colors) > maxSDKModeColors {
		return 0, nil, fmt.Errorf("invalid mode color count %d", len(mode.Colors))
	}
	packetID := updateModePacketV6
	data := make([]byte, 4, 16+len(mode.Name)+binary.Size(mode.SDKModeParameters)+4*len(mode.Colors))
	if target.ZoneIndex != nil {
		if mode.Flags&sdkModeRequiresEntireDevice != 0 {
			return 0, nil, errors.New("OpenRGB mode requires the entire controller")
		}
		packetID = updateZoneModePacketV6
		data = binary.LittleEndian.AppendUint32(data, *target.ZoneIndex)
	}
	data = binary.LittleEndian.AppendUint32(data, target.ModeIndex)
	data = binary.LittleEndian.AppendUint16(data, uint16(len(mode.Name)+1))
	data = append(data, mode.Name...)
	data = append(data, 0)
	data, err := binary.Append(data, binary.LittleEndian, mode.SDKModeParameters)
	if err != nil {
		return 0, nil, err
	}
	data = binary.LittleEndian.AppendUint16(data, uint16(len(mode.Colors)))
	for _, color := range mode.Colors {
		data = binary.LittleEndian.AppendUint32(data, color)
	}
	binary.LittleEndian.PutUint32(data, uint32(len(data)))
	return packetID, data, nil
}

// sdkModeMatchesReadback checks controller data after the mode command ACK.
// OpenRGB 1.1 may ACK status OK after rejecting descriptor values. This
// verifies cached metadata only. Compare the LED buffer separately for per-LED
// colors because it is not part of a per-LED mode descriptor.
func sdkModeMatchesReadback(activeIndex int32, active SDKMode, requestedIndex uint32, requested SDKMode) bool {
	return activeIndex >= 0 && uint32(activeIndex) == requestedIndex &&
		active.Name == requested.Name &&
		active.SDKModeParameters == requested.SDKModeParameters &&
		slices.Equal(active.Colors, requested.Colors)
}

// sdkControllerModeCoversZone checks whether a one-zone request covers all LEDs
// before considering a controller-wide mode. Multiple zones need an explicit
// controller-wide intent even if all but one currently report zero LEDs.
func sdkControllerModeCoversZone(controllerLEDCount uint32, zoneLEDCounts []uint32, zoneIndex uint32) bool {
	return zoneIndex == 0 && len(zoneLEDCounts) == 1 &&
		controllerLEDCount > 0 && zoneLEDCounts[0] == controllerLEDCount
}
