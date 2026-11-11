package control

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The pinned 1.1 server returned these bytes over SDK v6 from the four-LED
// ASUS TUF RTX 4070 Ti SUPER. The original retained evidence is
// openrgb-gpu-baseline-2026-11-10.json. This fixture is the controller-data
// payload, excluding the 16-byte transport header.
func TestOpenRGBV6CapturedGPUMode(t *testing.T) {
	capture, err := os.ReadFile("testdata/openrgb-1.1-gpu-controller-data.hex")
	require.NoError(t, err)
	payload := decodeModeFixtureHex(t, string(capture))
	require.Equal(t, "1438a21a69d7a504c894a84dd925526ec272264a3fc182735ee00eea9001f784", fmt.Sprintf("%x", sha256.Sum256(payload)))
	parsed, err := parseController(17, payload)
	require.NoError(t, err)
	parsedID, parsedSource := controllerIdentity(parsed)
	require.NotEmpty(t, parsedID)
	require.Equal(t, "openrgb_linux_pci_i2c_location", parsedSource)
	require.EqualValues(t, 6, parsed.ActiveMode)
	require.Len(t, parsed.Modes, 10)
	require.Equal(t, "Breathing", parsed.Modes[3].Name)
	require.Len(t, parsed.Zones, 1)
	require.EqualValues(t, 4, parsed.Zones[0].LEDCount)

	c := cursor{data: payload}
	size, err := c.u32()
	require.NoError(t, err)
	require.EqualValues(t, len(payload), size)
	identity, err := decodeSDKHardwareIdentityV6(&c)
	require.NoError(t, err)
	require.Equal(t, SDKHardwareIdentity{
		Type:        2,
		Name:        "ASUS TUF GeForce RTX 4070 Ti SUPER Gaming OC",
		Vendor:      "ASUS",
		Description: "ENE SMBus Device",
		Version:     "AUMA0-E6K5-1107",
		Location:    "I2C: NVIDIA i2c adapter 1 at 1:00.0\n (/dev/i2c-4), address 0x67",
	}, identity)
	id, source := stableControllerIdentity(identity)
	require.NotEmpty(t, id)
	require.Equal(t, "openrgb_linux_pci_i2c_location", source)

	modeCount, err := c.u16()
	require.NoError(t, err)
	require.EqualValues(t, 10, modeCount)
	activeIndex, err := c.u32()
	require.NoError(t, err)
	require.EqualValues(t, 6, activeIndex)
	modes := make([]SDKMode, modeCount)
	for i := range modes {
		modes[i], err = decodeSDKModeV6(&c)
		require.NoError(t, err)
	}
	require.Equal(t, "Off", modes[1].Name)
	require.Equal(t, "Static", modes[2].Name)
	require.Equal(t, "Rainbow", modes[activeIndex].Name)
	zoneCount, err := c.u16()
	require.NoError(t, err)
	require.EqualValues(t, 1, zoneCount)
	zone, err := parseZoneV6(&c)
	require.NoError(t, err)
	require.Equal(t, "Unknown", zone.Name)
	require.EqualValues(t, 4, zone.LEDCount)
	// The captured zone ends with active mode, mode count, and its empty
	// display-name string. Check the recorded absence of zone modes.
	require.Equal(t, []byte{0xff, 0xff, 0xff, 0xff, 0, 0, 1, 0, 0}, payload[c.at-9:c.at])
	ledCount, err := c.u16()
	require.NoError(t, err)
	require.True(t, sdkControllerModeCoversZone(uint32(ledCount), []uint32{zone.LEDCount}, 0))

	breathing := modes[3]
	require.Equal(t, SDKMode{
		Name: "Breathing",
		SDKModeParameters: SDKModeParameters{
			Flags:     0xa1,
			SpeedMin:  4,
			SpeedMax:  0,
			Speed:     2,
			ColorMode: sdkModeColorsPerLED,
		},
	}, breathing)
	breathing.Speed = breathing.SpeedMin
	packetID, body, err := encodeSDKModeUpdateV6(SDKModeTarget{ModeIndex: 3}, breathing)
	require.NoError(t, err)
	require.EqualValues(t, 1101, packetID)
	require.Equal(t, decodeModeFixtureHex(t, `
		42000000 03000000 0a00 427265617468696e6700
		a1000000 04000000 00000000 00000000 00000000 00000000 00000000
		04000000 00000000 00000000 01000000 0000
	`), body)
}

func TestOpenRGBV6ZoneModeWireAndReadback(t *testing.T) {
	// Zone 0 uses an independent mode-specific palette. An explicit zero
	// zone index must select the zone packet and preserve its signed field.
	zone := uint32(0)
	mode := SDKMode{
		Name: "Breath",
		SDKModeParameters: SDKModeParameters{
			Flags:         sdkModeHasSpeed | sdkModeHasBrightness | sdkModeHasModeSpecificColor,
			SpeedMin:      9,
			SpeedMax:      1,
			BrightnessMin: 2,
			BrightnessMax: 8,
			ColorsMin:     1,
			ColorsMax:     2,
			Speed:         9,
			Brightness:    3,
			ColorMode:     sdkModeColorsModeSpecific,
		},
		Colors: []uint32{0x00030201, 0x00060504},
	}
	target := SDKModeTarget{ModeIndex: 2, ZoneIndex: &zone}
	packetID, body, err := encodeSDKModeUpdateV6(target, mode)
	require.NoError(t, err)
	require.EqualValues(t, 1103, packetID)
	require.Equal(t, decodeModeFixtureHex(t, `
		4b000000 00000000 02000000 0700 42726561746800
		51000000 09000000 01000000 02000000 08000000 01000000 02000000
		09000000 03000000 00000000 02000000 0200 01020300 04050600
	`), body)
	for end := 12; end < len(body); end++ {
		c := cursor{data: body[12:end]}
		_, err := decodeSDKModeV6(&c)
		require.Error(t, err, "truncated at byte %d", end)
	}
	c := cursor{data: body[12:]}
	readback, err := decodeSDKModeV6(&c)
	require.NoError(t, err)
	require.Equal(t, len(c.data), c.at)

	// OpenRGB can report ACK status OK after rejecting the requested
	// values. The same active index or display name is insufficient.
	require.NoError(t, validateAck(sdkPacket{
		packetID: 10,
		deviceID: 17,
		data:     []byte{0x4f, 0x04, 0, 0, 0, 0, 0, 0},
	}, 17, packetID))
	require.False(t, sdkModeMatchesReadback(-1, readback, target.ModeIndex, mode))
	require.False(t, sdkModeMatchesReadback(1, readback, target.ModeIndex, mode))
	readback.Brightness = 6
	require.False(t, sdkModeMatchesReadback(2, readback, target.ModeIndex, mode))
	readback.Brightness = mode.Brightness
	readback.Colors[0] = 0
	require.False(t, sdkModeMatchesReadback(2, readback, target.ModeIndex, mode))
	readback.Colors[0] = mode.Colors[0]
	require.True(t, sdkModeMatchesReadback(2, readback, target.ModeIndex, mode))
}

func TestOpenRGBV6ModeScopeGuards(t *testing.T) {
	require.False(t, sdkControllerModeCoversZone(4, []uint32{4, 0}, 0), "a sibling's mode can change even when it has no LEDs")
	require.False(t, sdkControllerModeCoversZone(4, []uint32{2}, 0), "the requested zone does not cover the LED buffer")
	require.False(t, sdkControllerModeCoversZone(4, []uint32{4}, 1), "the requested index is absent")
	require.False(t, sdkControllerModeCoversZone(0, []uint32{0}, 0), "the device has no controllable LEDs")

	zone := uint32(0)
	wholeDevice := SDKMode{Name: "Static", SDKModeParameters: SDKModeParameters{Flags: sdkModeRequiresEntireDevice}}
	_, _, err := encodeSDKModeUpdateV6(SDKModeTarget{ZoneIndex: &zone}, wholeDevice)
	require.ErrorContains(t, err, "entire controller")
	_, _, err = encodeSDKModeUpdateV6(SDKModeTarget{ModeIndex: math.MaxUint32}, SDKMode{Name: "Static"})
	require.ErrorContains(t, err, "out of range")
}

func TestOpenRGBV6IdentityRejectsTransientLocation(t *testing.T) {
	identity := SDKHardwareIdentity{
		Type:     2,
		Name:     "ASUS GPU",
		Vendor:   "ASUS",
		Location: "I2C: NVIDIA i2c adapter 1 at 01:00.0\n (/dev/i2c-4), address 0x67",
	}
	originalID, source := stableControllerIdentity(identity)
	require.NotEmpty(t, originalID)
	require.Equal(t, "openrgb_linux_pci_i2c_location", source)

	identity.Location = strings.Replace(identity.Location, "/dev/i2c-4", "/dev/i2c-18", 1)
	identity.Version = "updated firmware"
	id, _ := stableControllerIdentity(identity)
	require.Equal(t, originalID, id)
	identity.Location = strings.Replace(identity.Location, "adapter 1", "adapter 2", 1)
	id, _ = stableControllerIdentity(identity)
	require.NotEqual(t, originalID, id)

	identity.Location = "I2C: /dev/i2c-4, address 0x67"
	for _, placeholder := range []string{"Unknown", "00000000", "n/a"} {
		identity.Serial = placeholder
		id, source = stableControllerIdentity(identity)
		require.Empty(t, id)
		require.Empty(t, source)
	}

	identity.Serial = "actual-serial"
	id, source = stableControllerIdentity(identity)
	require.NotEmpty(t, id)
	require.Equal(t, "openrgb_sdk_serial", source)
	identity.Location = "a different transport or port"
	sameDevice, _ := stableControllerIdentity(identity)
	require.Equal(t, id, sameDevice)
}

func decodeModeFixtureHex(t *testing.T, source string) []byte {
	t.Helper()
	data, err := hex.DecodeString(strings.Join(strings.Fields(source), ""))
	require.NoError(t, err)
	return data
}
