package control

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
	"google.golang.org/protobuf/types/known/emptypb"
)

func resizableFixture(count uint32) []byte {
	data := controllerV6Fixture("Static")
	offset := bytes.Index(data, []byte{6, 0, 'S', 't', 'r', 'i', 'p', 0}) + 8
	binary.LittleEndian.PutUint32(data[offset+4:], 0)
	binary.LittleEndian.PutUint32(data[offset+8:], 500)
	binary.LittleEndian.PutUint32(data[offset+12:], count)
	binary.LittleEndian.PutUint32(data[offset+20:], 2)
	return data
}
func TestSDKResizeReadbackPrecedesRenderAndUsesFreshHandle(t *testing.T) {
	for _, readback := range []uint32{24, 0} {
		t.Run(fmt.Sprint(readback), func(t *testing.T) {
			server := listenSDK(t)
			done := make(chan error, 1)
			go func() {
				conn, err := server.Accept()
				if err != nil {
					done <- err
					return
				}
				defer conn.Close()
				negotiateFixtureConnection(conn, t, 6)
				discovery := func(id, count uint32) {
					readSDKRequest(conn, nil, 0)
					writeSDKPacket(conn, 0, 0, append(uint32Bytes(1), uint32Bytes(id)...))
					writeSDKAck(conn, 0, 0, 0)
					readSDKRequest(conn, nil, 1)
					writeSDKPacket(conn, id, 1, resizableFixture(count))
					writeSDKAck(conn, id, 1, 0)
				}
				discovery(7, 0)
				packet, err := readPacket(conn)
				if err != nil {
					done <- err
					return
				}
				if packet.packetID != 1000 || packet.deviceID != 7 || !bytes.Equal(packet.data, append(uint32Bytes(0), uint32Bytes(24)...)) {
					done <- fmt.Errorf("invalid resize packet %+v", packet)
					return
				}
				writeSDKAck(conn, 7, 1000, 0)
				discovery(88, readback)
				if readback != 24 {
					packet, err = readPacket(conn)
					if err == nil {
						done <- fmt.Errorf("render after mismatched readback: %+v", packet)
						return
					}
					done <- nil
					return
				}
				packet, err = readPacket(conn)
				if err != nil {
					done <- err
					return
				}
				if packet.packetID != 1051 || packet.deviceID != 88 || binary.LittleEndian.Uint16(packet.data[8:10]) != 24 {
					done <- fmt.Errorf("invalid post-resize colors %+v", packet)
					return
				}
				writeSDKAck(conn, 88, 1051, 0)
				done <- nil
			}()
			c, err := parseController(7, resizableFixture(0))
			require.NoError(t, err)
			require.True(t, configurableZone(c.Zones[0]))
			id, _ := controllerIdentity(c)
			d := &controlv1.ZoneState{ControllerId: id, ZoneId: "opaque", DesiredMode: &controlv1.ZoneState_StaticColor{StaticColor: &controlv1.StaticZoneState{Color: &controlv1.Color{Red: 25}, BrightnessPercent: 100}}}
			result, err := (SDKClient{Address: server.Addr().String()}).ApplyConfiguredStableZone(context.Background(), id, zoneFingerprint(c.Zones[0]), 24, nil, d)
			if readback == 24 {
				require.NoError(t, err)
				require.True(t, result.BackendAcknowledged)
			} else {
				require.ErrorContains(t, err, "length readback")
				require.False(t, result.BackendAcknowledged)
			}
			require.NoError(t, <-done)
		})
	}
}
func TestZoneConfigurationRejectsUnknownAndEffectOnlyRanges(t *testing.T) {
	for _, z := range []SDKZone{{LEDMax: 500, Flags: 1}, {LEDMin: 500, LEDMax: 0, Flags: 2}, {LEDMin: 0, LEDMax: 0xffffffff, Flags: 2}, {LEDMin: 3, LEDMax: 3, Flags: 2}} {
		require.False(t, configurableZone(z))
		require.ErrorIs(t, validateZoneLength(z, 5), ErrUnsupported)
	}
}

func TestControllerDirectReuseRequiresFreshModeOrCompleteOffScope(t *testing.T) {
	direct := SDKMode{Name: "Direct", SDKModeParameters: SDKModeParameters{Flags: sdkModeHasPerLEDColor, ColorMode: sdkModeColorsPerLED}}
	c := SDKController{ActiveMode: 1, Modes: []SDKMode{direct, {Name: "Breathing"}}, Zones: []SDKZone{{Index: 0, LEDCount: 3}, {Index: 1, LEDCount: 5}}}
	d := &controlv1.ZoneState{DesiredMode: &controlv1.ZoneState_StaticColor{StaticColor: &controlv1.StaticZoneState{Color: &controlv1.Color{Red: 1}}}}
	_, _, ok := requestedModeWithControllerOff(d, c, c.Zones[0], false)
	require.False(t, ok, "one-zone intent cannot switch a whole controller")
	_, _, ok = requestedModeWithControllerOff(d, c, c.Zones[0], true)
	require.False(t, ok, "whole-controller permission cannot authorize a non-Off global mutation")
	d.DesiredMode = &controlv1.ZoneState_Off{Off: &emptypb.Empty{}}
	_, _, ok = requestedModeWithControllerOff(d, c, c.Zones[0], true)
	require.True(t, ok, "explicit whole-controller Off can initialize known zones")
	c.Zones[1].LEDCount = 0
	_, _, ok = requestedModeWithControllerOff(d, c, c.Zones[0], true)
	require.False(t, ok, "unknown empty peer cannot authorize global mode mutation")
	c.ActiveMode = 0
	_, _, ok = requestedModeWithControllerOff(d, c, c.Zones[0], false)
	require.True(t, ok, "already-active per-LED mode requires no global mutation")
}

func TestWholeControllerOffScopeRejectsUnmappedChangedAndAmbiguousPeers(t *testing.T) {
	c := SDKController{Zones: []SDKZone{{Name: "Fixed", LEDCount: 1}, {Name: "Header", LEDCount: 12, LEDMax: 500, Flags: 2}}}
	expected := map[string]uint32{zoneFingerprint(c.Zones[0]): 1, zoneFingerprint(c.Zones[1]): 12}
	require.True(t, completeControllerOffScope(c, expected))
	c.Zones[1].LEDCount = 500
	require.False(t, completeControllerOffScope(c, expected), "changed physical count invalidates global authority")
	c.Zones[1].LEDCount = 12
	require.False(t, completeControllerOffScope(c, map[string]uint32{zoneFingerprint(c.Zones[0]): 1}), "subset cannot authorize global mode mutation")
	c.Zones = append(c.Zones, SDKZone{Name: "New", LEDCount: 5})
	require.False(t, completeControllerOffScope(c, expected), "new physical zone invalidates scope")
	c.Zones = c.Zones[:2]
	c.Zones[1].LEDCount = 0
	require.False(t, completeControllerOffScope(c, expected), "unknown length prevents initialization")
	c.Zones[1] = c.Zones[0]
	require.False(t, completeControllerOffScope(c, expected), "duplicate fingerprints prevent initialization")
}

func TestAdversarialResizeWriteSurvivesReadbackDisconnect(t *testing.T) {
	server := listenSDK(t)
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := server.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		negotiateFixtureConnection(conn, t, 6)
		readSDKRequest(conn, nil, 0)
		writeSDKPacket(conn, 0, 0, append(uint32Bytes(1), uint32Bytes(7)...))
		writeSDKAck(conn, 0, 0, 0)
		readSDKRequest(conn, nil, 1)
		writeSDKPacket(conn, 7, 1, resizableFixture(0))
		writeSDKAck(conn, 7, 1, 0)
		readSDKRequest(conn, nil, 1000)
		writeSDKAck(conn, 7, 1000, 0)
	}()
	c, err := parseController(7, resizableFixture(0))
	require.NoError(t, err)
	id, _ := controllerIdentity(c)
	d := &controlv1.ZoneState{ControllerId: id, DesiredMode: &controlv1.ZoneState_StaticColor{StaticColor: &controlv1.StaticZoneState{Color: &controlv1.Color{Red: 1}, BrightnessPercent: 100}}}
	result, err := (SDKClient{Address: server.Addr().String()}).ApplyConfiguredStableZone(context.Background(), id, zoneFingerprint(c.Zones[0]), 24, nil, d)
	require.Error(t, err)
	require.True(t, result.LocalWriteCompleted, "resize bytes were sent and acknowledged before connection loss; do not erase partial mutation evidence")
	require.False(t, result.BackendAcknowledged, "complete render was not acknowledged")
	<-done
}

func TestResizeNegativeAckPreservesStatusWithoutAcknowledgingRender(t *testing.T) {
	server := listenSDK(t)
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := server.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		negotiateFixtureConnection(conn, t, 6)
		readSDKRequest(conn, nil, 0)
		writeSDKPacket(conn, 0, 0, append(uint32Bytes(1), uint32Bytes(7)...))
		writeSDKAck(conn, 0, 0, 0)
		readSDKRequest(conn, nil, 1)
		writeSDKPacket(conn, 7, 1, resizableFixture(0))
		writeSDKAck(conn, 7, 1, 0)
		readSDKRequest(conn, nil, 1000)
		writeSDKAck(conn, 7, 1000, 3)
	}()
	c, err := parseController(7, resizableFixture(0))
	require.NoError(t, err)
	id, _ := controllerIdentity(c)
	d := &controlv1.ZoneState{ControllerId: id, DesiredMode: &controlv1.ZoneState_Off{Off: &emptypb.Empty{}}}
	result, err := (SDKClient{Address: server.Addr().String()}).ApplyConfiguredStableZone(context.Background(), id, zoneFingerprint(c.Zones[0]), 24, nil, d)
	require.Error(t, err)
	require.True(t, result.LocalWriteCompleted)
	require.False(t, result.BackendAcknowledged)
	require.Equal(t, uint32(3), result.AckStatus)
	<-done
}
