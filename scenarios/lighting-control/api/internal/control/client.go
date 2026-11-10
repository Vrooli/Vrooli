package control

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"syscall"
	"time"
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
	Name        string
	Vendor      string
	Description string
	Zones       []SDKZone
}

type SDKZone struct {
	Name     string
	LEDCount uint32
	Index    uint32 // transient SDK zone index; never persisted as identity
	Static   bool
}

type RGB struct{ R, G, B uint8 }

// SDKCommandResult separates a successful local write from the OpenRGB v6
// acknowledgement that the backend processed the packet.
type SDKCommandResult struct {
	LocalWriteCompleted bool
	BackendAcknowledged bool
	AckStatus           uint32
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

func (c SDKClient) negotiate(ctx context.Context, conn net.Conn) (uint32, error) {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, openRGBProtocolVersion)
	if err := writePacket(conn, 0, requestProtocolVersion, data); err != nil {
		return 0, fmt.Errorf("%w: negotiate SDK protocol: %v", ErrTransport, err)
	}
	var version uint32
	var gotServerName bool
	for version == 0 || !gotServerName {
		packet, err := readPacket(conn)
		if err != nil {
			return 0, transportReadError("negotiate SDK protocol", err)
		}
		switch packet.packetID {
		case requestProtocolVersion:
			if packet.deviceID != 0 || len(packet.data) != 4 || version != 0 {
				return 0, fmt.Errorf("%w: invalid protocol-version reply", ErrProtocolMismatch)
			}
			version = binary.LittleEndian.Uint32(packet.data)
			if version < openRGBProtocolVersion {
				return version, fmt.Errorf("%w: server supports SDK protocol %d; version %d is required for stable IDs and acknowledgements", ErrProtocolMismatch, version, openRGBProtocolVersion)
			}
		case serverNamePacket:
			gotServerName = true
		default:
			// OpenRGB may publish asynchronous state notifications on this
			// connection. They are not replies to the request being negotiated.
		}
	}
	return version, nil
}

func transportReadError(action string, err error) error {
	var netErr net.Error
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, context.DeadlineExceeded) || errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("%w: %s: %v", ErrTransport, action, err)
	}
	return err
}

func validateAck(packet sdkPacket, deviceID, packetID uint32) error {
	if packet.deviceID != deviceID || len(packet.data) != 8 {
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

func (c SDKClient) request(ctx context.Context, deviceID, packetID uint32, data []byte) (sdkPacket, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return sdkPacket{}, err
	}
	defer conn.Close()
	if _, err = c.negotiate(ctx, conn); err != nil {
		return sdkPacket{}, err
	}
	if err = writePacket(conn, deviceID, packetID, data); err != nil {
		return sdkPacket{}, fmt.Errorf("%w: write packet: %v", ErrTransport, err)
	}
	return readOperation(conn, deviceID, packetID, true, packetID != requestControllerCount)
}

func readOperation(conn net.Conn, deviceID, packetID uint32, hasReply, hasAck bool) (sdkPacket, error) {
	var response sdkPacket
	gotReply, gotAck := false, !hasAck
	for !gotAck || hasReply && !gotReply {
		packet, err := readPacket(conn)
		if err != nil {
			return sdkPacket{}, transportReadError("read SDK operation result", err)
		}
		if packet.packetID == ackPacket {
			if err := validateAck(packet, deviceID, packetID); err != nil {
				return sdkPacket{}, err
			}
			gotAck = true
			continue
		}
		if hasReply && packet.packetID == packetID {
			if packet.deviceID != deviceID || gotReply {
				return sdkPacket{}, fmt.Errorf("%w: unexpected duplicate or device ID in response to packet %d", ErrProtocolMismatch, packetID)
			}
			response = packet
			gotReply = true
		}
		// Unrelated server notifications are drained, never mistaken for the
		// expected reply or the next command's acknowledgement.
	}
	return response, nil
}

func (c SDKClient) ProtocolVersion(ctx context.Context) (uint32, error) {
	conn, err := c.dial(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	return c.negotiate(ctx, conn)
}

// Discover negotiates each short-lived operation connection. OpenRGB stores
// negotiated protocol version per connection, so controller-count and
// controller-data requests each repeat the v6 handshake before sending their
// operation. Protocol v6 unique controller IDs are used as identity.
func (c SDKClient) Discover(ctx context.Context) ([]SDKController, error) {
	countPacket, err := c.request(ctx, 0, requestControllerCount, nil)
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
	ids := make([]uint32, count)
	for i := range ids {
		ids[i] = binary.LittleEndian.Uint32(countPacket.data[4+i*4:])
	}
	controllers := make([]SDKController, 0, count)
	for _, id := range ids {
		packet, requestErr := c.request(ctx, id, requestControllerData, nil)
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

// SendZoneColors performs a v6 handshake on this operation's connection and
// waits for the correlated backend ACK. A successful ACK reports OpenRGB's
// processing status, not controller state or visible output.
func (c SDKClient) SendZoneColors(ctx context.Context, controllerID, zoneIndex uint32, colors []RGB) (SDKCommandResult, error) {
	if len(colors) == 0 || len(colors) > 65535 {
		return SDKCommandResult{}, fmt.Errorf("invalid LED color count %d", len(colors))
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
	conn, err := c.dial(ctx)
	if err != nil {
		return SDKCommandResult{}, err
	}
	defer conn.Close()
	if _, err := c.negotiate(ctx, conn); err != nil {
		return SDKCommandResult{}, err
	}
	if err := writePacket(conn, controllerID, updateZoneLEDs, data); err != nil {
		return SDKCommandResult{}, fmt.Errorf("%w: write UpdateZoneLEDs: %v", ErrTransport, err)
	}
	result := SDKCommandResult{LocalWriteCompleted: true}
	_, err = readOperation(conn, controllerID, updateZoneLEDs, false, true)
	var ackErr *SDKAckError
	if errors.As(err, &ackErr) {
		result.BackendAcknowledged = true
		result.AckStatus = ackErr.Status
	} else if err == nil {
		result.BackendAcknowledged = true
	}
	return result, err
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
	if err = c.skip(4); err != nil {
		return SDKController{}, err
	} // controller type
	name, err := c.str16()
	if err != nil {
		return SDKController{}, err
	}
	vendor, err := c.str16()
	if err != nil {
		return SDKController{}, err
	}
	description, err := c.str16()
	if err != nil {
		return SDKController{}, err
	}
	for i := 0; i < 3; i++ {
		if _, err = c.str16(); err != nil {
			return SDKController{}, err
		}
	} // version, serial, location
	modes, err := c.u16()
	if err != nil || modes > 4096 {
		return SDKController{}, errors.New("invalid mode count")
	}
	if err = c.skip(4); err != nil {
		return SDKController{}, err
	} // active mode
	for i := uint16(0); i < modes; i++ {
		if err = skipModeV6(&c); err != nil {
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
	return SDKController{ID: id, Name: name, Vendor: vendor, Description: description, Zones: zones}, nil
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
	if err = c.skip(8); err != nil {
		return SDKZone{}, err
	} // flags and active mode
	modes, err := c.u16()
	if err != nil || modes > 4096 {
		return SDKZone{}, errors.New("invalid zone mode count")
	}
	static := false
	for i := uint16(0); i < modes; i++ {
		var mode string
		mode, err = readModeV6(c)
		if err != nil {
			return SDKZone{}, err
		}
		if strings.EqualFold(strings.TrimSpace(mode), "static") {
			static = true
		}
	}
	if _, err = c.str16(); err != nil {
		return SDKZone{}, err
	} // display name
	return SDKZone{Name: name, LEDCount: count, Static: static}, nil
}
