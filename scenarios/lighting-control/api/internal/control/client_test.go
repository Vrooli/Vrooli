package control

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

func TestDiscoverV6UsesUniqueIDsAndParsesStaticZoneCapability(t *testing.T) {
	server := listenSDK(t)
	done := make(chan error, 1)
	go func() {
		for _, step := range []struct {
			id       uint32
			deviceID uint32
			response []byte
		}{
			{0, 0, append(uint32Bytes(1), uint32Bytes(4242)...)},
			{1, 4242, controllerV6Fixture("Static")},
		} {
			conn, err := server.Accept()
			if err != nil {
				done <- err
				return
			}
			negotiateFixtureConnection(conn, nil, 6)
			p := readSDKRequest(conn, nil, step.id)
			if p.deviceID != step.deviceID || step.id == 1 && len(p.data) != 0 {
				done <- fmt.Errorf("request %d device=%d payload=%x, want device=%d and no v6 controller-data payload", step.id, p.deviceID, p.data, step.deviceID)
				_ = conn.Close()
				return
			}
			writeSDKPacket(conn, p.deviceID, step.id, step.response)
			if step.id == 1 {
				writeSDKAck(conn, p.deviceID, step.id, 0)
			}
			_ = conn.Close()
		}
		done <- nil
	}()
	controllers, err := (SDKClient{Address: server.Addr().String()}).Discover(context.Background())
	require.NoError(t, err)
	require.NoError(t, <-done)
	require.Len(t, controllers, 1)
	require.Equal(t, uint32(4242), controllers[0].ID)
	require.Equal(t, "Board", controllers[0].Name)
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

// negotiateFixtureConnection is derived from the pinned OpenRGB release_1.1
// NetworkServer implementation (request 40 stores protocol state per client,
// then the listener sends reply 40 and server-name packet 51. It does not ACK
// the negotiation request; the controller worker ACKs controller operations.
// These fixture IDs and status values are literal release facts,
// not aliases of production constants.
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
	writeSDKPacket(conn, 0, 51, []byte("OpenRGB release fixture\x00"))
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
	putU32(0)
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
