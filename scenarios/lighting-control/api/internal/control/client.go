package control

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"syscall"
	"time"

	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
)

const (
	openRGBProtocolVersion uint32 = 6
	maxSDKPacketSize              = 4 << 20
	requestControllerCount uint32 = 0
	requestControllerData  uint32 = 1
	ackPacket              uint32 = 10
	requestProtocolVersion uint32 = 40
	serverNamePacket       uint32 = 51
	updateZoneLEDs         uint32 = 1051
)

// SDKClient speaks only the OpenRGB SDK TCP protocol. It has no install,
// process, scan or lifecycle behavior; its caller supplies a loopback endpoint.
type SDKClient struct {
	Address string
	Timeout time.Duration
}

type SDKController struct {
	ID          uint32
	Type        uint32
	Name        string
	Vendor      string
	Description string
	Serial      string
	Location    string
	LEDCount    uint32
	ActiveMode  int32
	Modes       []SDKMode
	Zones       []SDKZone
}

type SDKZone struct {
	Name       string
	LEDCount   uint32
	Index      uint32 // transient SDK zone index; never persisted as identity
	Static     bool
	ActiveMode int32
	Modes      []SDKMode
}

type RGB struct{ R, G, B uint8 }

// SDKCommandResult separates a successful local write from the OpenRGB v6
// acknowledgement that the backend processed the packet.
type SDKCommandResult struct {
	LocalWriteCompleted bool
	BackendAcknowledged bool
	AckStatus           uint32
	CachedModeReadback  controlv1.CachedModeReadback
}

type SDKAckError struct {
	DeviceID uint32
	PacketID uint32
	Status   uint32
}

func (e *SDKAckError) Error() string {
	return fmt.Sprintf("OpenRGB acknowledged packet %d for device %d with status %d", e.PacketID, e.DeviceID, e.Status)
}

var (
	ErrBackendUnavailable = errors.New("OpenRGB backend unavailable")
	ErrProtocolMismatch   = errors.New("OpenRGB SDK protocol mismatch")
	ErrNoDevice           = errors.New("no OpenRGB controllers found")
	ErrPermission         = errors.New("OpenRGB SDK permission denied")
	ErrTransport          = errors.New("OpenRGB SDK transport failure")
)

func (c SDKClient) timeout() time.Duration {
	if c.Timeout <= 0 {
		return 3 * time.Second
	}
	return c.Timeout
}

func validateLoopbackAddress(address string) (string, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", fmt.Errorf("invalid OpenRGB SDK endpoint: %w", err)
	}
	ip, err := netip.ParseAddr(strings.Trim(host, "[]"))
	if err != nil || !ip.IsLoopback() {
		return "", errors.New("OpenRGB SDK endpoint must use a literal loopback address")
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return "", errors.New("OpenRGB SDK endpoint has an invalid port")
	}
	return net.JoinHostPort(ip.String(), port), nil
}

func (c SDKClient) dial(ctx context.Context) (net.Conn, error) {
	address, err := validateLoopbackAddress(c.Address)
	if err != nil {
		return nil, err
	}
	d := net.Dialer{Timeout: c.timeout()}
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) {
			return nil, fmt.Errorf("%w: %v", ErrPermission, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrBackendUnavailable, err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(c.timeout()))
	}
	return conn, nil
}

type sdkPacket struct {
	deviceID uint32
	packetID uint32
	data     []byte
}

func writePacket(w io.Writer, deviceID, packetID uint32, data []byte) error {
	if len(data) > maxSDKPacketSize {
		return fmt.Errorf("SDK packet is too large: %d bytes", len(data))
	}
	packet := make([]byte, 16+len(data))
	copy(packet, "ORGB")
	binary.LittleEndian.PutUint32(packet[4:8], deviceID)
	binary.LittleEndian.PutUint32(packet[8:12], packetID)
	binary.LittleEndian.PutUint32(packet[12:16], uint32(len(data)))
	copy(packet[16:], data)
	for len(packet) > 0 {
		n, err := w.Write(packet)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		packet = packet[n:]
	}
	return nil
}

func readPacket(r io.Reader) (sdkPacket, error) {
	var header [16]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return sdkPacket{}, fmt.Errorf("read SDK packet header: %w", err)
	}
	if string(header[:4]) != "ORGB" {
		return sdkPacket{}, fmt.Errorf("%w: invalid packet magic", ErrProtocolMismatch)
	}
	size := binary.LittleEndian.Uint32(header[12:16])
	if size > maxSDKPacketSize {
		return sdkPacket{}, fmt.Errorf("%w: packet length %d exceeds %d bytes", ErrProtocolMismatch, size, maxSDKPacketSize)
	}
	data := make([]byte, int(size))
	if _, err := io.ReadFull(r, data); err != nil {
		return sdkPacket{}, fmt.Errorf("read SDK packet body: %w", err)
	}
	return sdkPacket{
		deviceID: binary.LittleEndian.Uint32(header[4:8]),
		packetID: binary.LittleEndian.Uint32(header[8:12]),
		data:     data,
	}, nil
}

func negotiate(conn net.Conn) (uint32, error) {
	data := binary.LittleEndian.AppendUint32(nil, openRGBProtocolVersion)
	if err := writePacket(conn, 0, requestProtocolVersion, data); err != nil {
		return 0, fmt.Errorf("%w: negotiate SDK protocol: %v", ErrTransport, err)
	}
	reply, err := readExpectedPacket(conn, 0, requestProtocolVersion)
	if err != nil {
		return 0, err
	}
	if len(reply.data) != 4 {
		return 0, fmt.Errorf("%w: invalid protocol-version reply", ErrProtocolMismatch)
	}
	version := binary.LittleEndian.Uint32(reply.data)
	if version < openRGBProtocolVersion {
		return version, fmt.Errorf("%w: server supports SDK protocol %d; version %d is required for stable IDs and acknowledgements", ErrProtocolMismatch, version, openRGBProtocolVersion)
	}
	name, err := readExpectedPacket(conn, 0, serverNamePacket)
	if err != nil {
		return version, err
	}
	if !sdkCString(name.data) {
		return version, fmt.Errorf("%w: invalid server-name reply", ErrProtocolMismatch)
	}
	// OpenRGB 1.1 sends this ACK from its listener after both replies.
	return version, readAck(conn, 0, requestProtocolVersion)
}

func transportReadError(action string, err error) error {
	var netErr net.Error
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, context.DeadlineExceeded) || errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("%w: %s: %v", ErrTransport, action, err)
	}
	return err
}

func validateAck(packet sdkPacket, deviceID, packetID uint32) error {
	if packet.packetID != ackPacket || packet.deviceID != deviceID || len(packet.data) != 8 {
		return fmt.Errorf("%w: malformed ACK for device %d packet %d", ErrProtocolMismatch, deviceID, packetID)
	}
	ackedPacketID := binary.LittleEndian.Uint32(packet.data[:4])
	status := binary.LittleEndian.Uint32(packet.data[4:8])
	if ackedPacketID != packetID {
		return fmt.Errorf("%w: ACK correlates to packet %d, waiting for packet %d", ErrProtocolMismatch, ackedPacketID, packetID)
	}
	if status != 0 {
		return &SDKAckError{DeviceID: deviceID, PacketID: packetID, Status: status}
	}
	return nil
}

func requestOnConn(conn net.Conn, deviceID, packetID uint32, data []byte) (sdkPacket, error) {
	if err := writePacket(conn, deviceID, packetID, data); err != nil {
		return sdkPacket{}, fmt.Errorf("%w: write packet: %v", ErrTransport, err)
	}
	reply, err := readExpectedPacket(conn, deviceID, packetID)
	if err != nil {
		return sdkPacket{}, err
	}
	return reply, readAck(conn, deviceID, packetID)
}

func readAck(conn net.Conn, deviceID, packetID uint32) error {
	packet, err := readExpectedPacket(conn, deviceID, ackPacket)
	if err != nil {
		return err
	}
	return validateAck(packet, deviceID, packetID)
}

func readExpectedPacket(conn net.Conn, deviceID, packetID uint32) (sdkPacket, error) {
	for {
		packet, err := readPacket(conn)
		if err != nil {
			return sdkPacket{}, transportReadError("read SDK operation result", err)
		}
		if packet.packetID == packetID && packet.deviceID == deviceID {
			return packet, nil
		}
		if !sdkNotification(packet) {
			return sdkPacket{}, fmt.Errorf("%w: received packet %d for device %d, expected packet %d for device %d", ErrProtocolMismatch, packet.packetID, packet.deviceID, packetID, deviceID)
		}
	}
}

func sdkCString(data []byte) bool {
	return len(data) > 0 && data[len(data)-1] == 0 && bytes.IndexByte(data[:len(data)-1], 0) < 0
}

// The pinned server can interleave these unsolicited broadcasts with a
// request's replies and ACK. Only their native framing is accepted here.
func sdkNotification(packet sdkPacket) bool {
	data := packet.data
	switch packet.packetID {
	case 100, 101, 103: // Device list updated, detection started/completed.
		return packet.deviceID == 0 && len(data) == 0
	case 102: // Detection progress.
		return packet.deviceID == 0 && sdkSizedNotification(data, 11) &&
			int(binary.LittleEndian.Uint16(data[8:10])) == len(data)-10 && sdkCString(data[10:])
	case 160: // Profile list updated.
		return packet.deviceID == 0 && sdkProfileList(data)
	case 1150: // Controller state changed.
		return sdkSizedNotification(data, 10)
	default:
		return false
	}
}

func sdkProfileList(data []byte) bool {
	if !sdkSizedNotification(data, 6) {
		return false
	}
	count := binary.LittleEndian.Uint16(data[4:6])
	offset := 6
	for i := uint16(0); i < count; i++ {
		if offset+2 > len(data) {
			return false
		}
		size := int(binary.LittleEndian.Uint16(data[offset : offset+2]))
		offset += 2
		if offset+size > len(data) || !sdkCString(data[offset:offset+size]) {
			return false
		}
		offset += size
	}
	return offset == len(data)
}

func sdkSizedNotification(data []byte, minimum int) bool {
	return len(data) >= minimum && binary.LittleEndian.Uint32(data[:4]) == uint32(len(data))
}

func (c SDKClient) ProtocolVersion(ctx context.Context) (uint32, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return negotiate(conn)
}

// Discover reads the controller roster over one negotiated connection.
// SDK handles are transient; durable identity comes from serial or location.
func (c SDKClient) Discover(ctx context.Context) ([]SDKController, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if _, err := negotiate(conn); err != nil {
		return nil, err
	}
	return discoverOnConn(conn)
}

// SendMode applies one descriptor at the exact controller or zone scope
// already resolved from a fresh v6 controller snapshot. A successful result
// means the backend ACKed the mode packet; it does not prove cached or physical
// output.
func (c SDKClient) SendMode(ctx context.Context, controllerID uint32, target SDKModeTarget, mode SDKMode) (SDKCommandResult, error) {
	packetID, body, err := encodeSDKModeUpdateV6(target, mode)
	if err != nil {
		return SDKCommandResult{}, err
	}
	conn, err := c.dial(ctx)
	if err != nil {
		return SDKCommandResult{}, err
	}
	defer conn.Close()
	if _, err := negotiate(conn); err != nil {
		return SDKCommandResult{}, err
	}
	if err := writePacket(conn, controllerID, packetID, body); err != nil {
		return SDKCommandResult{}, fmt.Errorf("%w: write OpenRGB mode update: %v", ErrTransport, err)
	}
	result := SDKCommandResult{LocalWriteCompleted: true}
	err = readAck(conn, controllerID, packetID)
	var ackErr *SDKAckError
	if err == nil {
		result.BackendAcknowledged = true
	} else if errors.As(err, &ackErr) {
		result.BackendAcknowledged = true
		result.AckStatus = ackErr.Status
	}
	return result, err
}

// ApplyStableZone resolves the transient SDK controller and zone handles from
// a fresh controller-data snapshot immediately before issuing a mutation.
func (c SDKClient) ApplyStableZone(ctx context.Context, controllerID, fingerprint string, desired *controlv1.ZoneState) (SDKCommandResult, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return SDKCommandResult{}, err
	}
	defer conn.Close()
	if _, err = negotiate(conn); err != nil {
		return SDKCommandResult{}, err
	}
	found, err := discoverOnConn(conn)
	if err != nil {
		return SDKCommandResult{}, err
	}
	controller, zone, err := resolveStableZoneFrom(found, controllerID, fingerprint)
	if err != nil {
		return SDKCommandResult{}, err
	}
	mode, targetMode, ok := requestedMode(desired, *controller, *zone)
	if !ok {
		return SDKCommandResult{}, ErrUnsupported
	}
	_, wantsEffect := desired.DesiredMode.(*controlv1.ZoneState_Effect)
	if wantsEffect && !modeCanHonorBrightness(mode, desired) {
		return SDKCommandResult{}, fmt.Errorf("%w: %w", ErrUnsupported, ErrBrightnessLimit)
	}
	var combined SDKCommandResult
	modeSent := false
	if wantsEffect || !modeAlreadyActive(targetMode, *controller, *zone) {
		modeSent = true
		mode = configureMode(mode, desired)
		combined, err = sendModeOnConn(conn, controller.ID, targetMode, mode)
		if err != nil {
			return combined, err
		}
		// OpenRGB ACK and its subsequently returned controller cache are separate
		// facts. Check the exact cached scope after the command, then resolve fresh
		// transient handles again before any follow-up packet.
		found, err = discoverOnConn(conn)
		if err != nil {
			return combined, err
		}
		controller, zone, err = resolveStableZoneFrom(found, controllerID, fingerprint)
		if err != nil {
			return combined, err
		}
		if modeReadbackMatches(targetMode, *controller, *zone, mode) {
			combined.CachedModeReadback = controlv1.CachedModeReadback_CACHED_MODE_READBACK_MATCHES
		} else {
			combined.CachedModeReadback = controlv1.CachedModeReadback_CACHED_MODE_READBACK_MISMATCH
			return combined, ErrModeReadbackMismatch
		}
	} else {
		combined.CachedModeReadback = controlv1.CachedModeReadback_CACHED_MODE_READBACK_MATCHES
	}
	if wantsEffect && mode.ColorMode != sdkModeColorsPerLED {
		return combined, nil
	}
	color, ok := requestedColor(desired)
	if !ok {
		return combined, ErrUnsupported
	}
	brightness := requestedBrightness(desired)
	color.R = uint8(math.Round(float64(color.R) * brightness / 100))
	color.G = uint8(math.Round(float64(color.G) * brightness / 100))
	color.B = uint8(math.Round(float64(color.B) * brightness / 100))
	colors := make([]RGB, zone.LEDCount)
	for i := range colors {
		colors[i] = color
	}
	command, colorErr := sendZoneColorsOnConn(conn, controller.ID, zone.Index, colors)
	if colorErr != nil {
		return mergeSDKCommandResults(combined, command, modeSent), colorErr
	}
	return mergeSDKCommandResults(combined, command, modeSent), nil
}

func mergeSDKCommandResults(first, second SDKCommandResult, bothRequired bool) SDKCommandResult {
	out := second
	out.LocalWriteCompleted = first.LocalWriteCompleted || second.LocalWriteCompleted
	if bothRequired {
		out.BackendAcknowledged = first.BackendAcknowledged && second.BackendAcknowledged
	}
	if second.AckStatus == 0 {
		out.AckStatus = first.AckStatus
	}
	if out.CachedModeReadback == controlv1.CachedModeReadback_CACHED_MODE_READBACK_UNSPECIFIED {
		out.CachedModeReadback = first.CachedModeReadback
	}
	return out
}

func (c SDKClient) resolveStableZone(ctx context.Context, controllerID, fingerprint string) (*SDKController, *SDKZone, error) {
	found, err := c.Discover(ctx)
	if err != nil {
		return nil, nil, err
	}
	return resolveStableZoneFrom(found, controllerID, fingerprint)
}

func resolveStableZoneFrom(found []SDKController, controllerID, fingerprint string) (*SDKController, *SDKZone, error) {
	var controller *SDKController
	for i := range found {
		id, source := controllerIdentity(found[i])
		if id != controllerID || !stableIdentitySource(source) {
			continue
		}
		if controller != nil {
			return nil, nil, ErrUnstableIdentity
		}
		controller = &found[i]
	}
	if controller == nil {
		return nil, nil, ErrControllerNotFound
	}
	var zone *SDKZone
	for i := range controller.Zones {
		if zoneFingerprint(controller.Zones[i]) != fingerprint {
			continue
		}
		if zone != nil {
			return nil, nil, ErrAmbiguousZone
		}
		zone = &controller.Zones[i]
	}
	if zone == nil {
		return nil, nil, ErrControllerNotFound
	}
	if zone.LEDCount == 0 {
		return nil, nil, ErrUnsupported
	}
	return controller, zone, nil
}

// discoverOnConn refreshes controller count and descriptors over the same
// negotiated socket that will carry the following mutation. This narrows the
// transient-handle window and prevents a second SDK connection from applying
// an ID resolved against a different server snapshot.
func discoverOnConn(conn net.Conn) ([]SDKController, error) {
	countPacket, err := requestOnConn(conn, 0, requestControllerCount, nil)
	if err != nil {
		return nil, err
	}
	if len(countPacket.data) < 4 {
		return nil, fmt.Errorf("%w: controller count response truncated", ErrProtocolMismatch)
	}
	count := binary.LittleEndian.Uint32(countPacket.data[:4])
	if count > 1024 || uint64(len(countPacket.data)) != 4+uint64(count)*4 {
		return nil, fmt.Errorf("%w: invalid controller count payload", ErrProtocolMismatch)
	}
	if count == 0 {
		return []SDKController{}, ErrNoDevice
	}
	controllers := make([]SDKController, 0, count)
	for i := uint32(0); i < count; i++ {
		id := binary.LittleEndian.Uint32(countPacket.data[4+i*4:])
		packet, requestErr := requestOnConn(conn, id, requestControllerData, nil)
		if requestErr != nil {
			return nil, requestErr
		}
		controller, parseErr := parseController(id, packet.data)
		if parseErr != nil {
			return nil, fmt.Errorf("%w: controller %d: %v", ErrProtocolMismatch, id, parseErr)
		}
		controllers = append(controllers, controller)
	}
	return controllers, nil
}

func sendModeOnConn(conn net.Conn, controllerID uint32, target SDKModeTarget, mode SDKMode) (SDKCommandResult, error) {
	packetID, body, err := encodeSDKModeUpdateV6(target, mode)
	if err != nil {
		return SDKCommandResult{}, err
	}
	if err := writePacket(conn, controllerID, packetID, body); err != nil {
		return SDKCommandResult{}, fmt.Errorf("%w: write OpenRGB mode update: %v", ErrTransport, err)
	}
	result := SDKCommandResult{LocalWriteCompleted: true}
	err = readAck(conn, controllerID, packetID)
	var ackErr *SDKAckError
	if err == nil {
		result.BackendAcknowledged = true
	} else if errors.As(err, &ackErr) {
		result.BackendAcknowledged = true
		result.AckStatus = ackErr.Status
	}
	return result, err
}

func sendZoneColorsOnConn(conn net.Conn, controllerID, zoneIndex uint32, colors []RGB) (SDKCommandResult, error) {
	data, err := encodeZoneColors(zoneIndex, colors)
	if err != nil {
		return SDKCommandResult{}, err
	}
	return sendZoneColorsBodyOnConn(conn, controllerID, data)
}

func sendZoneColorsBodyOnConn(conn net.Conn, controllerID uint32, data []byte) (SDKCommandResult, error) {
	if err := writePacket(conn, controllerID, updateZoneLEDs, data); err != nil {
		return SDKCommandResult{}, fmt.Errorf("%w: write UpdateZoneLEDs: %v", ErrTransport, err)
	}
	result := SDKCommandResult{LocalWriteCompleted: true}
	err := readAck(conn, controllerID, updateZoneLEDs)
	var ackErr *SDKAckError
	if errors.As(err, &ackErr) {
		result.BackendAcknowledged = true
		result.AckStatus = ackErr.Status
	} else if err == nil {
		result.BackendAcknowledged = true
	}
	return result, err
}

// SendZoneColors performs a v6 handshake on this operation's connection and
// waits for the correlated backend ACK. A successful ACK reports OpenRGB's
// processing status, not controller state or visible output.
func (c SDKClient) SendZoneColors(ctx context.Context, controllerID, zoneIndex uint32, colors []RGB) (SDKCommandResult, error) {
	data, err := encodeZoneColors(zoneIndex, colors)
	if err != nil {
		return SDKCommandResult{}, err
	}
	conn, err := c.dial(ctx)
	if err != nil {
		return SDKCommandResult{}, err
	}
	defer conn.Close()
	if _, err := negotiate(conn); err != nil {
		return SDKCommandResult{}, err
	}
	return sendZoneColorsBodyOnConn(conn, controllerID, data)
}

func encodeZoneColors(zoneIndex uint32, colors []RGB) ([]byte, error) {
	if len(colors) == 0 || len(colors) > 65535 {
		return nil, fmt.Errorf("invalid LED color count %d", len(colors))
	}
	data := make([]byte, 10+4*len(colors))
	// OpenRGB's UpdateZoneLEDs parser compares this prefix with the full
	// command body size, including this four-byte prefix.
	binary.LittleEndian.PutUint32(data[:4], uint32(len(data)))
	binary.LittleEndian.PutUint32(data[4:8], zoneIndex)
	binary.LittleEndian.PutUint16(data[8:10], uint16(len(colors)))
	for i, color := range colors {
		data[10+i*4] = color.R
		data[11+i*4] = color.G
		data[12+i*4] = color.B
		data[13+i*4] = 0
	}
	return data, nil
}

type cursor struct {
	data []byte
	at   int
}

func (c *cursor) take(n uint64) ([]byte, error) {
	if n > uint64(len(c.data)-c.at) {
		return nil, io.ErrUnexpectedEOF
	}
	out := c.data[c.at : c.at+int(n)]
	c.at += int(n)
	return out, nil
}

func (c *cursor) u16() (uint16, error) {
	b, e := c.take(2)
	if e != nil {
		return 0, e
	}
	return binary.LittleEndian.Uint16(b), nil
}

func (c *cursor) u32() (uint32, error) {
	b, e := c.take(4)
	if e != nil {
		return 0, e
	}
	return binary.LittleEndian.Uint32(b), nil
}

func (c *cursor) str16() (string, error) {
	n, e := c.u16()
	if e != nil {
		return "", e
	}
	b, e := c.take(uint64(n))
	if e != nil {
		return "", e
	}
	return strings.TrimRight(string(b), "\x00"), nil
}

func (c *cursor) str32() (string, error) {
	n, e := c.u32()
	if e != nil {
		return "", e
	}
	b, e := c.take(uint64(n))
	if e != nil {
		return "", e
	}
	return strings.TrimRight(string(b), "\x00"), nil
}
func (c *cursor) skip(n uint64) error { _, e := c.take(n); return e }

func parseController(id uint32, payload []byte) (SDKController, error) {
	c := cursor{data: payload}
	blockSize, err := c.u32()
	if err != nil || uint64(blockSize) != uint64(len(payload)) {
		return SDKController{}, errors.New("invalid device data size")
	}
	identity, err := decodeSDKHardwareIdentityV6(&c)
	if err != nil {
		return SDKController{}, err
	}
	modes, err := c.u16()
	if err != nil || modes > 4096 {
		return SDKController{}, errors.New("invalid mode count")
	}
	activeMode, err := c.u32()
	if err != nil {
		return SDKController{}, err
	}
	controllerModes := make([]SDKMode, modes)
	for i := uint16(0); i < modes; i++ {
		controllerModes[i], err = decodeSDKModeV6(&c)
		if err != nil {
			return SDKController{}, err
		}
	}
	zoneCount, err := c.u16()
	if err != nil || zoneCount > 4096 {
		return SDKController{}, errors.New("invalid zone count")
	}
	zones := make([]SDKZone, 0, zoneCount)
	for i := uint16(0); i < zoneCount; i++ {
		z, zoneErr := parseZoneV6(&c)
		if zoneErr != nil {
			return SDKController{}, zoneErr
		}
		z.Index = uint32(i)
		zones = append(zones, z)
	}
	ledCount, err := c.u16()
	if err != nil || ledCount > 65535 {
		return SDKController{}, errors.New("invalid LED count")
	}
	for i := uint16(0); i < ledCount; i++ {
		if _, err = c.str16(); err != nil {
			return SDKController{}, err
		}
	}
	colors, err := c.u16()
	if err != nil || uint64(colors)*4 > uint64(len(c.data)-c.at) {
		return SDKController{}, errors.New("invalid color count")
	}
	if err = c.skip(uint64(colors) * 4); err != nil {
		return SDKController{}, err
	}
	// Protocol v5/v6 additions follow the v0 controller colors.
	displayNames, err := c.u16()
	if err != nil || displayNames > 4096 {
		return SDKController{}, errors.New("invalid LED display name count")
	}
	for i := uint16(0); i < displayNames; i++ {
		if _, err = c.str16(); err != nil {
			return SDKController{}, err
		}
	}
	if err = c.skip(4); err != nil {
		return SDKController{}, err
	} // controller flags
	if _, err = c.str16(); err != nil {
		return SDKController{}, err
	} // display name
	if _, err = c.str32(); err != nil {
		return SDKController{}, err
	} // configuration
	if c.at != len(c.data) {
		return SDKController{}, errors.New("unexpected trailing controller data")
	}
	return SDKController{ID: id, Type: identity.Type, Name: identity.Name, Vendor: identity.Vendor, Description: identity.Description, Serial: identity.Serial, Location: identity.Location, LEDCount: uint32(ledCount), ActiveMode: int32(activeMode), Modes: controllerModes, Zones: zones}, nil
}

func skipModeV6(c *cursor) error {
	_, err := readModeV6(c)
	return err
}

func readModeV6(c *cursor) (string, error) {
	name, err := c.str16()
	if err != nil {
		return "", err
	}
	// flags, speed min/max, brightness min/max, colors min/max, speed,
	// brightness, direction, color mode, followed by mode colors.
	if err := c.skip(11 * 4); err != nil {
		return "", err
	}
	n, err := c.u16()
	if err != nil || n > 4096 {
		return "", errors.New("invalid mode color count")
	}
	return name, c.skip(uint64(n) * 4)
}

func parseZoneV6(c *cursor) (SDKZone, error) {
	name, err := c.str16()
	if err != nil {
		return SDKZone{}, err
	}
	if err = c.skip(12); err != nil {
		return SDKZone{}, err
	} // type and LED limits
	count, err := c.u32()
	if err != nil {
		return SDKZone{}, err
	}
	matrixLen, err := c.u16()
	if err != nil || uint64(matrixLen) > uint64(maxSDKPacketSize) {
		return SDKZone{}, errors.New("invalid matrix map size")
	}
	if err = c.skip(uint64(matrixLen)); err != nil {
		return SDKZone{}, err
	}
	segments, err := c.u16()
	if err != nil || segments > 4096 {
		return SDKZone{}, errors.New("invalid segment count")
	}
	for i := uint16(0); i < segments; i++ {
		if _, err = c.str16(); err != nil {
			return SDKZone{}, err
		}
		if err = c.skip(12); err != nil {
			return SDKZone{}, err
		}
		segmentMatrix, e := c.u16()
		if e != nil || uint64(segmentMatrix) > uint64(maxSDKPacketSize) {
			return SDKZone{}, errors.New("invalid segment matrix size")
		}
		if err = c.skip(uint64(segmentMatrix) + 4); err != nil {
			return SDKZone{}, err
		}
	}
	flags, err := c.u32()
	if err != nil {
		return SDKZone{}, err
	}
	activeMode, err := c.u32()
	if err != nil {
		return SDKZone{}, err
	}
	modes, err := c.u16()
	if err != nil || modes > 4096 {
		return SDKZone{}, errors.New("invalid zone mode count")
	}
	static := false
	modeDescriptors := make([]SDKMode, modes)
	for i := uint16(0); i < modes; i++ {
		modeDescriptors[i], err = decodeSDKModeV6(c)
		if err != nil {
			return SDKZone{}, err
		}
		if strings.EqualFold(strings.TrimSpace(modeDescriptors[i].Name), "static") {
			static = true
		}
	}
	if _, err = c.str16(); err != nil {
		return SDKZone{}, err
	} // display name
	_ = flags
	return SDKZone{Name: name, LEDCount: count, Static: static, ActiveMode: int32(activeMode), Modes: modeDescriptors}, nil
}
