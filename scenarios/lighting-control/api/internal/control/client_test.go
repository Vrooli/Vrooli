package control

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
)

func TestSDKClientRejectsNonLoopbackEndpoint(t *testing.T) {
	_, err := (SDKClient{Address: "192.0.2.10:6742"}).ProtocolVersion(context.Background())
	require.ErrorContains(t, err, "literal loopback")
}

func TestReadPacketRejectsTruncatedAndOversizedFrames(t *testing.T) {
	truncated := append([]byte("ORGB"), make([]byte, 11)...)
	_, err := readPacket(strings.NewReader(string(truncated)))
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	bodyTruncated := make([]byte, 18)
	copy(bodyTruncated, "ORGB")
	binary.LittleEndian.PutUint32(bodyTruncated[12:], 4)
	bodyTruncated[16] = 1
	bodyTruncated[17] = 2
	_, err = readPacket(strings.NewReader(string(bodyTruncated)))
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	oversized := make([]byte, 16)
	copy(oversized, "ORGB")
	binary.LittleEndian.PutUint32(oversized[12:], 4<<20+1)
	_, err = readPacket(strings.NewReader(string(oversized)))
	require.ErrorContains(t, err, "exceeds")
}

func TestProtocolVersionFakeServer(t *testing.T) {
	server := listenSDK(t)
	go func() {
		conn, e := server.Accept()
		if e != nil {
			return
		}
		defer conn.Close()
		negotiateFixtureConnection(conn, t, 6)
	}()
	version, err := (SDKClient{Address: server.Addr().String()}).ProtocolVersion(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint32(6), version)
}

func TestProtocolVersionRejectsOldServer(t *testing.T) {
	server := listenSDK(t)
	go func() {
		conn, e := server.Accept()
		if e != nil {
			return
		}
		defer conn.Close()
		negotiateFixtureConnection(conn, t, 5)
	}()
	_, err := (SDKClient{Address: server.Addr().String()}).ProtocolVersion(context.Background())
	require.ErrorIs(t, err, ErrProtocolMismatch)
}

func TestV6RepliesRequireExactSequenceAndCorrelatedAcks(t *testing.T) {
	version := sdkPacket{packetID: 40, data: uint32Bytes(6)}
	name := sdkPacket{packetID: 51, data: []byte("OpenRGB 1.1\x00")}
	ack := func(device, packet uint32) sdkPacket {
		return sdkPacket{deviceID: device, packetID: 10, data: append(uint32Bytes(packet), uint32Bytes(0)...)}
	}
	count := sdkPacket{packetID: 0, data: uint32Bytes(0)}
	for _, tc := range []struct {
		name        string
		negotiation []sdkPacket
		discovery   bool
		count       []sdkPacket
		want        error
	}{
		{name: "server name before version", negotiation: []sdkPacket{name, version}, want: ErrProtocolMismatch},
		{name: "duplicate version", negotiation: []sdkPacket{version, version, name}, want: ErrProtocolMismatch},
		{name: "malformed server name", negotiation: []sdkPacket{version, {packetID: 51, data: []byte("OpenRGB")}}, want: ErrProtocolMismatch},
		{name: "wrong server name device", negotiation: []sdkPacket{version, {deviceID: 7, packetID: 51, data: name.data}}, want: ErrProtocolMismatch},
		{name: "negotiation ack before server name", negotiation: []sdkPacket{version, ack(0, 40), name}, want: ErrProtocolMismatch},
		{name: "wrong negotiation ack", negotiation: []sdkPacket{version, name, ack(0, 0)}, want: ErrProtocolMismatch},
		{name: "missing negotiation ack", negotiation: []sdkPacket{version, name}, want: ErrTransport},
		{name: "empty pinned roster", discovery: true, count: []sdkPacket{count, ack(0, 0)}, want: ErrNoDevice},
		{name: "duplicate negotiation ack", negotiation: []sdkPacket{version, name, ack(0, 40), ack(0, 40)}, discovery: true, count: []sdkPacket{count, ack(0, 0)}, want: ErrProtocolMismatch},
		{name: "count ack before reply", discovery: true, count: []sdkPacket{ack(0, 0), count}, want: ErrProtocolMismatch},
		{name: "duplicate count reply", discovery: true, count: []sdkPacket{count, count, ack(0, 0)}, want: ErrProtocolMismatch},
		{name: "wrong count ack device", discovery: true, count: []sdkPacket{count, ack(7, 0)}, want: ErrProtocolMismatch},
		{name: "malformed count ack", discovery: true, count: []sdkPacket{count, {packetID: 10, data: uint32Bytes(0)}}, want: ErrProtocolMismatch},
		{name: "missing count ack", discovery: true, count: []sdkPacket{count}, want: ErrTransport},
		{name: "malformed notification", discovery: true, count: []sdkPacket{{packetID: 100, data: []byte{0}}, count, ack(0, 0)}, want: ErrProtocolMismatch},
		{name: "unknown notification", discovery: true, count: []sdkPacket{{packetID: 999}, count, ack(0, 0)}, want: ErrProtocolMismatch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := listenSDK(t)
			go func() {
				conn, err := server.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				readSDKRequest(conn, nil, 40)
				frames := tc.negotiation
				if frames == nil {
					frames = []sdkPacket{version, name, ack(0, 40)}
				}
				for _, frame := range frames {
					writeSDKPacket(conn, frame.deviceID, frame.packetID, frame.data)
				}
				if tc.discovery {
					readSDKRequest(conn, nil, 0)
					for _, frame := range tc.count {
						writeSDKPacket(conn, frame.deviceID, frame.packetID, frame.data)
					}
				}
			}()
			client := SDKClient{Address: server.Addr().String(), Timeout: time.Second}
			var err error
			if tc.discovery {
				_, err = client.Discover(context.Background())
			} else {
				_, err = client.ProtocolVersion(context.Background())
			}
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestPinnedOpenRGBReleaseNoDevices(t *testing.T) {
	if os.Getenv("VROOLI_OPENRGB_PINNED_FIXTURE") != "1" {
		t.Skip("requires the isolated pinned OpenRGB 1.1 fixture at 127.0.0.1:47642")
	}
	client := SDKClient{Address: "127.0.0.1:47642"}
	version, err := client.ProtocolVersion(context.Background())
	require.NoError(t, err)
	require.Equal(t, uint32(6), version)
	devices, err := client.Discover(context.Background())
	require.ErrorIs(t, err, ErrNoDevice)
	require.Empty(t, devices)
}

func TestProtocolReadTimeoutIsTransportFailure(t *testing.T) {
	server := listenSDK(t)
	go func() {
		conn, e := server.Accept()
		if e == nil {
			defer conn.Close()
			_, _ = io.Copy(io.Discard, conn)
		}
	}()
	_, err := (SDKClient{Address: server.Addr().String(), Timeout: 40 * time.Millisecond}).ProtocolVersion(context.Background())
	require.ErrorIs(t, err, ErrTransport)
}

func TestDiscoverV6RetainsTransientHandlesAndParsesControllerIdentityAndCapability(t *testing.T) {
	server := listenSDK(t)
	done := make(chan error, 1)
	go func() {
		conn, err := server.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		negotiateFixtureConnection(conn, nil, 6)
		for _, step := range []struct {
			id       uint32
			deviceID uint32
			response []byte
		}{
			{0, 0, append(uint32Bytes(1), uint32Bytes(4242)...)},
			{1, 4242, controllerV6Fixture("Static")},
		} {
			p := readSDKRequest(conn, nil, step.id)
			if p.packetID != step.id || p.deviceID != step.deviceID || len(p.data) != 0 {
				done <- fmt.Errorf("request %d device=%d payload=%x, want packet %d, device=%d and empty payload", p.packetID, p.deviceID, p.data, step.id, step.deviceID)
				return
			}
			writeSDKPacket(conn, p.deviceID, step.id, step.response)
			writeSDKAck(conn, p.deviceID, step.id, 0)
		}
		done <- nil
	}()
	controllers, err := (SDKClient{Address: server.Addr().String()}).Discover(context.Background())
	require.NoError(t, err)
	require.NoError(t, <-done)
	require.Len(t, controllers, 1)
	require.Equal(t, uint32(4242), controllers[0].ID)
	require.Equal(t, "Board", controllers[0].Name)
	require.Equal(t, "serial", controllers[0].Serial)
	require.Equal(t, "local", controllers[0].Location)
	require.Len(t, controllers[0].Zones, 1)
	require.True(t, controllers[0].Zones[0].Static)
	require.Equal(t, uint32(2), controllers[0].Zones[0].LEDCount)
	parsed, err := parseController(4242, controllerV6Fixture("Rainbow"))
	require.NoError(t, err)
	require.False(t, parsed.Zones[0].Static, "unsupported modes must not be reported as static-color capability")
	truncatedSize := controllerV6Fixture("Static")
	binary.LittleEndian.PutUint32(truncatedSize[:4], uint32(len(truncatedSize)-4))
	_, err = parseController(4242, truncatedSize)
	require.ErrorContains(t, err, "invalid device data size", "release_1.1 prefixes the full serialized reply size")
}

func TestStableMutationUsesFreshIdentityAndMutationOnOneSDKConnection(t *testing.T) {
	server := listenSDK(t)
	done := make(chan error, 1)
	go func() {
		conn, err := server.Accept()
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()
		negotiateFixtureConnection(conn, nil, 6)
		count := readSDKRequest(conn, nil, requestControllerCount)
		if count.deviceID != 0 {
			done <- fmt.Errorf("controller count request device = %d", count.deviceID)
			return
		}
		writeSDKPacket(conn, 0, requestControllerCount, append(uint32Bytes(1), uint32Bytes(4242)...))
		writeSDKAck(conn, 0, requestControllerCount, 0)
		data := readSDKRequest(conn, nil, requestControllerData)
		if data.deviceID != 4242 {
			done <- fmt.Errorf("controller data request device = %d", data.deviceID)
			return
		}
		writeSDKPacket(conn, 4242, requestControllerData, controllerV6FixtureWithActiveMode("Static", math.MaxUint32))
		writeSDKAck(conn, 4242, requestControllerData, 0)
		mode := readSDKRequest(conn, nil, 1103)
		if mode.packetID != 1103 || mode.deviceID != 4242 || len(mode.data) < 12 ||
			binary.LittleEndian.Uint32(mode.data[4:8]) != 0 || binary.LittleEndian.Uint32(mode.data[8:12]) != 0 {
			done <- fmt.Errorf("zone mode request targeted packet=%d device=%d body=%x", mode.packetID, mode.deviceID, mode.data)
			return
		}
		writeSDKPacket(conn, 0, 100, nil)
		writeSDKAck(conn, 4242, 1103, 0)
		count = readSDKRequest(conn, nil, 0)
		if count.packetID != 0 || count.deviceID != 0 || len(count.data) != 0 {
			done <- fmt.Errorf("readback count request = %+v", count)
			return
		}
		writeSDKPacket(conn, 0, 0, append(uint32Bytes(1), uint32Bytes(4243)...))
		writeSDKAck(conn, 0, 0, 0)
		data = readSDKRequest(conn, nil, 1)
		if data.packetID != 1 || data.deviceID != 4243 || len(data.data) != 0 {
			done <- fmt.Errorf("readback controller-data request = %+v", data)
			return
		}
		writeSDKPacket(conn, 4243, 1, controllerV6Fixture("Static"))
		writeSDKAck(conn, 4243, 1, 0)
		colors := readSDKRequest(conn, nil, updateZoneLEDs)
		if colors.packetID != 1051 || colors.deviceID != 4243 || len(colors.data) < 8 || binary.LittleEndian.Uint32(colors.data[4:8]) != 0 {
			done <- fmt.Errorf("zone color request targeted device=%d body=%x", colors.deviceID, colors.data)
			return
		}
		writeSDKAck(conn, 4243, updateZoneLEDs, 0)
		done <- nil
	}()

	controller, err := parseController(4242, controllerV6Fixture("Static"))
	require.NoError(t, err)
	id, source := controllerIdentity(controller)
	require.True(t, stableIdentitySource(source))
	zone := controller.Zones[0]
	desired := &controlv1.ZoneState{ControllerId: id, ZoneId: "unused-persisted-zone-id", DesiredMode: &controlv1.ZoneState_StaticColor{StaticColor: &controlv1.StaticZoneState{Color: &controlv1.Color{Red: 1, Green: 2, Blue: 3}, BrightnessPercent: 100}}}
	result, err := (SDKClient{Address: server.Addr().String()}).ApplyStableZone(context.Background(), id, zoneFingerprint(zone), desired)
	require.NoError(t, err)
	require.True(t, result.BackendAcknowledged)
	require.Equal(t, controlv1.CachedModeReadback_CACHED_MODE_READBACK_MATCHES, result.CachedModeReadback)
	require.NoError(t, <-done, "discovery and the scoped mutation must share one negotiated SDK connection")
}

func TestStableMutationStopsBeforeColorsWhenReadbackFails(t *testing.T) {
	for _, tc := range []struct {
		name     string
		ackOnly  bool
		want     error
		readback controlv1.CachedModeReadback
	}{
		{name: "unchanged cached mode", want: ErrModeReadbackMismatch, readback: controlv1.CachedModeReadback_CACHED_MODE_READBACK_MISMATCH},
		{name: "missing transient controller replies with ack only", ackOnly: true, want: ErrProtocolMismatch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := listenSDK(t)
			done := make(chan error, 1)
			go func() {
				conn, err := server.Accept()
				if err != nil {
					done <- err
					return
				}
				defer conn.Close()
				negotiateFixtureConnection(conn, nil, 6)
				for index, command := range []uint32{0, 1, 1103, 0, 1} {
					packet := readSDKRequest(conn, nil, command)
					device := uint32(4242)
					if command == 0 {
						device = 0
					}
					if packet.packetID != command || packet.deviceID != device {
						done <- fmt.Errorf("readback sequence received %+v, want packet=%d device=%d", packet, command, device)
						return
					}
					switch command {
					case 0:
						writeSDKPacket(conn, 0, 0, append(uint32Bytes(1), uint32Bytes(4242)...))
					case 1:
						if !tc.ackOnly || index == 1 {
							writeSDKPacket(conn, 4242, 1, controllerV6FixtureWithActiveMode("Static", math.MaxUint32))
						}
					}
					writeSDKAck(conn, device, command, 0)
				}
				_ = conn.SetReadDeadline(time.Now().Add(time.Second))
				packet, err := readPacket(conn)
				if err == nil {
					done <- fmt.Errorf("received packet %d after failed readback", packet.packetID)
					return
				}
				done <- nil
			}()
			controller, err := parseController(4242, controllerV6Fixture("Static"))
			require.NoError(t, err)
			id, _ := controllerIdentity(controller)
			desired := &controlv1.ZoneState{DesiredMode: &controlv1.ZoneState_StaticColor{StaticColor: &controlv1.StaticZoneState{Color: &controlv1.Color{}, BrightnessPercent: 100}}}
			result, err := (SDKClient{Address: server.Addr().String()}).ApplyStableZone(context.Background(), id, zoneFingerprint(controller.Zones[0]), desired)
			require.ErrorIs(t, err, tc.want)
			require.True(t, result.BackendAcknowledged)
			require.Equal(t, tc.readback, result.CachedModeReadback)
			require.NoError(t, <-done)
		})
	}
}

func TestUpdateZoneLEDsReportsCorrelatedBackendAcknowledgement(t *testing.T) {
	server := listenSDK(t)
	got := make(chan []byte, 1)
	go func() {
		conn, e := server.Accept()
		if e != nil {
			return
		}
		defer conn.Close()
		negotiateFixtureConnection(conn, t, 6)
		p := readSDKRequest(conn, t, 1051)
		got <- p.data
		writeSDKAck(conn, p.deviceID, 1051, 0)
	}()
	result, err := (SDKClient{Address: server.Addr().String()}).SendZoneColors(context.Background(), 77, 3, []RGB{{R: 4, G: 5, B: 6}, {R: 8, G: 9, B: 10}})
	require.NoError(t, err)
	require.True(t, result.BackendAcknowledged)
	require.Zero(t, result.AckStatus)
	data := <-got
	require.Equal(t, uint32(len(data)), binary.LittleEndian.Uint32(data[:4]))
	require.Equal(t, []byte{
		18, 0, 0, 0, 3, 0, 0, 0, 2, 0, 4, 5, 6, 0, 8, 9, 10, 0,
	}, data, "release_1.1 UpdateZoneLEDs body includes the size word in its prefix")
	require.Equal(t, uint32(3), binary.LittleEndian.Uint32(data[4:8]))
	require.Equal(t, uint16(2), binary.LittleEndian.Uint16(data[8:10]))
	require.Equal(t, []byte{4, 5, 6, 0, 8, 9, 10, 0}, data[10:])
}

func TestUpdateZoneLEDsRejectsEmptyOrOversizedColorLists(t *testing.T) {
	client := SDKClient{Address: "127.0.0.1:1"}
	_, err := client.SendZoneColors(context.Background(), 1, 0, nil)
	require.ErrorContains(t, err, "invalid LED color count")
	_, err = client.SendZoneColors(context.Background(), 1, 0, make([]RGB, 65536))
	require.ErrorContains(t, err, "invalid LED color count")
}

func listenSDK(t *testing.T) *net.TCPListener {
	t.Helper()
	l, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP("127.0.0.1")})
	require.NoError(t, err)
	t.Cleanup(func() { _ = l.Close() })
	return l
}

type receivedPacket struct {
	deviceID, packetID uint32
	data               []byte
}

func readSDKRequest(conn net.Conn, t *testing.T, expected uint32) receivedPacket {
	if t != nil {
		t.Helper()
	}
	var h [16]byte
	_, err := io.ReadFull(conn, h[:])
	if t != nil {
		require.NoError(t, err)
		require.Equal(t, "ORGB", string(h[:4]))
	} else if err != nil {
		return receivedPacket{}
	}
	size := binary.LittleEndian.Uint32(h[12:])
	data := make([]byte, size)
	_, err = io.ReadFull(conn, data)
	if t != nil {
		require.NoError(t, err)
	} else if err != nil {
		return receivedPacket{}
	}
	p := receivedPacket{binary.LittleEndian.Uint32(h[4:]), binary.LittleEndian.Uint32(h[8:]), data}
	if t != nil {
		require.Equal(t, expected, p.packetID)
	}
	return p
}

func writeSDKPacket(conn net.Conn, device, packet uint32, data []byte) {
	var h [16]byte
	copy(h[:], "ORGB")
	binary.LittleEndian.PutUint32(h[4:], device)
	binary.LittleEndian.PutUint32(h[8:], packet)
	binary.LittleEndian.PutUint32(h[12:], uint32(len(data)))
	_, _ = conn.Write(h[:])
	_, _ = conn.Write(data)
}

// negotiateFixtureConnection mirrors the release_1.1 listener and the
// recorded Linux amd64 frames: version, server string, then ACK for 40.
func negotiateFixtureConnection(conn net.Conn, t *testing.T, version uint32) {
	if t != nil {
		t.Helper()
	}
	request := readSDKRequest(conn, t, 40)
	if len(request.data) != 4 || binary.LittleEndian.Uint32(request.data) != 6 {
		if t != nil {
			t.Fatalf("protocol negotiation payload=%x, want little-endian uint32 6", request.data)
		}
		return
	}
	writeSDKPacket(conn, 0, 40, uint32Bytes(version))
	if version < 6 {
		return
	}
	writeSDKPacket(conn, 0, 51, []byte("OpenRGB 1.1\x00"))
	writeSDKAck(conn, 0, 40, 0)
}

func writeSDKAck(conn net.Conn, device, ackedPacketID, status uint32) {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint32(data[:4], ackedPacketID)
	binary.LittleEndian.PutUint32(data[4:], status)
	writeSDKPacket(conn, device, 10, data)
}

func TestUpdateZoneLEDsRejectsAckForDifferentPacket(t *testing.T) {
	server := listenSDK(t)
	go func() {
		conn, err := server.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		negotiateFixtureConnection(conn, nil, 6)
		packet := readSDKRequest(conn, nil, 1051)
		writeSDKAck(conn, packet.deviceID, 0, 0)
	}()
	_, err := (SDKClient{Address: server.Addr().String()}).SendZoneColors(context.Background(), 77, 3, []RGB{{R: 1}})
	require.ErrorIs(t, err, ErrProtocolMismatch)
}

func TestUpdateZoneLEDsReturnsNonzeroBackendAckStatus(t *testing.T) {
	server := listenSDK(t)
	go func() {
		conn, err := server.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		negotiateFixtureConnection(conn, nil, 6)
		packet := readSDKRequest(conn, nil, 1051)
		writeSDKAck(conn, packet.deviceID, 1051, 2)
	}()
	result, err := (SDKClient{Address: server.Addr().String()}).SendZoneColors(context.Background(), 77, 3, []RGB{{R: 1}})
	var ackErr *SDKAckError
	require.ErrorAs(t, err, &ackErr)
	require.Equal(t, uint32(2), ackErr.Status)
	require.True(t, result.BackendAcknowledged)
	require.Equal(t, uint32(2), result.AckStatus)
}

func TestUpdateZoneLEDsSeparatesCompletedWriteFromMissingAck(t *testing.T) {
	server := listenSDK(t)
	go func() {
		conn, err := server.Accept()
		if err != nil {
			return
		}
		negotiateFixtureConnection(conn, nil, 6)
		_ = readSDKRequest(conn, nil, 1051)
		_ = conn.Close()
	}()
	result, err := (SDKClient{Address: server.Addr().String(), Timeout: 100 * time.Millisecond}).SendZoneColors(context.Background(), 77, 3, []RGB{{R: 1}})
	require.ErrorIs(t, err, ErrTransport)
	require.True(t, result.LocalWriteCompleted)
	require.False(t, result.BackendAcknowledged)
}

func uint32Bytes(n uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, n)
	return b
}

func controllerV6Fixture(zoneMode string) []byte {
	return controllerV6FixtureWithActiveMode(zoneMode, 0)
}

func controllerV6FixtureWithActiveMode(zoneMode string, activeMode uint32) []byte {
	data := make([]byte, 4)
	putU32 := func(v uint32) { b := make([]byte, 4); binary.LittleEndian.PutUint32(b, v); data = append(data, b...) }
	putU16 := func(v uint16) { b := make([]byte, 2); binary.LittleEndian.PutUint16(b, v); data = append(data, b...) }
	putString := func(s string) { putU16(uint16(len(s) + 1)); data = append(data, []byte(s)...); data = append(data, 0) }
	putU32(0)
	putString("Board")
	putString("Vendor")
	putString("Fixture")
	putString("v1")
	putString("serial")
	putString("local")
	putU16(0)
	putU32(0)
	putU16(1)
	putString("Strip")
	putU32(0)
	putU32(2)
	putU32(2)
	putU32(2)
	putU16(0)
	putU16(0)
	putU32(0)
	putU32(activeMode)
	putU16(1)
	putString(zoneMode)
	for i := 0; i < 11; i++ {
		putU32(0)
	}
	putU16(0)
	putString("Strip")
	putU16(2)
	putString("LED 1")
	putString("LED 2")
	putU16(2)
	data = append(data, make([]byte, 8)...)
	putU16(0)
	putU32(0)
	putString("Board")
	putU32(0)
	binary.LittleEndian.PutUint32(data[:4], uint32(len(data)))
	return data
}
