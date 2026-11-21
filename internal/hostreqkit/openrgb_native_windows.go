//go:build windows

package hostreqkit

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var openRGBHIDDLL = windows.NewLazySystemDLL("hid.dll")
var openRGBGetAttributes = openRGBHIDDLL.NewProc("HidD_GetAttributes")
var openRGBGetPreparsed = openRGBHIDDLL.NewProc("HidD_GetPreparsedData")
var openRGBFreePreparsed = openRGBHIDDLL.NewProc("HidD_FreePreparsedData")
var openRGBGetCaps = openRGBHIDDLL.NewProc("HidP_GetCaps")
var openRGBGetProperty = windows.NewLazySystemDLL("setupapi.dll").NewProc("SetupDiGetDevicePropertyW")

func nativeOpenRGBPlatform() string { return "windows" }
func confineNativeOpenRGBTransport(ctx context.Context, root string) (string, error) {
	return verifyNativeOpenRGBBlockedModules(ctx, root, os.Lstat)
}
func nativeOpenRGBIdentity() (string, string, error) {
	token := windows.GetCurrentThreadEffectiveToken()
	// Query explicitly: Token.IsElevated cannot distinguish query failure from false.
	var elevated uint32
	var needed uint32
	if err := windows.GetTokenInformation(token, windows.TokenElevation, (*byte)(unsafe.Pointer(&elevated)), 4, &needed); err != nil {
		return "", "", err
	}
	if elevated != 0 {
		return "", "", errors.New("native OpenRGB HID access requires a non-elevated account; no administrator fallback")
	}
	groups, err := token.GetTokenGroups()
	if err != nil {
		return "", "", err
	}
	for _, group := range groups.AllGroups() {
		if group.Sid.String() == "S-1-5-32-544" && group.Attributes&windows.SE_GROUP_ENABLED != 0 && group.Attributes&windows.SE_GROUP_USE_FOR_DENY_ONLY == 0 {
			return "", "", errors.New("native OpenRGB HID account has enabled administrator group authority")
		}
	}
	user, err := token.GetTokenUser()
	if err != nil {
		return "", "", err
	}
	account := "sid:" + user.User.Sid.String()
	if !validNativeAccount("windows", account) {
		return "", "", errors.New("native OpenRGB refuses system or privileged service identity")
	}
	machine, err := nativeOpenRGBMachineID()
	if err != nil {
		return "", "", err
	}
	return machine, account, nil
}
func nativeOpenRGBMachineID() (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE|registry.WOW64_64KEY)
	if err != nil {
		return "", err
	}
	defer key.Close()
	guid, _, err := key.GetStringValue("MachineGuid")
	if err != nil || strings.TrimSpace(guid) == "" {
		return "", errors.Join(errors.New("read Windows native machine identity"), err)
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.ToLower(guid)))), nil
}

func openRGBWindowsInstance(path string) string {
	p := strings.TrimPrefix(strings.ToLower(path), `\\?\`)
	if end := strings.LastIndex(p, "#{"); end >= 0 {
		p = p[:end]
	}
	return strings.ReplaceAll(p, "#", `\`)
}

func nativeOpenRGBDiscover(ctx context.Context) ([]OpenRGBNativeNode, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	guid := windows.GUID{Data1: 0x4d1e55b2, Data2: 0xf16f, Data3: 0x11cf, Data4: [8]byte{0x88, 0xcb, 0x00, 0x11, 0x11, 0x00, 0x00, 0x30}}
	paths, err := windows.CM_Get_Device_Interface_List("", &guid, 0)
	if err != nil {
		return nil, err
	}
	if len(paths) > 4096 {
		return nil, errors.New("native HID interface inventory exceeds bound")
	}
	set, err := windows.SetupDiGetClassDevsEx(&guid, "", 0, windows.DIGCF_PRESENT|windows.DIGCF_DEVICEINTERFACE, 0, "")
	if err != nil {
		return nil, err
	}
	defer set.Close()
	type birth struct {
		devinst uint32
		arrival uint64
	}
	births := map[string]birth{}
	// LastArrivalDate is a FILETIME, not a stable instance identifier. Replugging
	// the same port changes it even when Windows reuses the symbolic HID path.
	key := windows.DEVPROPKEY{FmtID: windows.DEVPROPGUID{Data1: 0x83da6326, Data2: 0x97a6, Data3: 0x4088, Data4: [8]byte{0x94, 0x53, 0xa1, 0x92, 0x3f, 0x57, 0x3b, 0x29}}, PID: 102}
	for i := 0; i < 4096; i++ {
		dev, e := set.EnumDeviceInfo(i)
		if errors.Is(e, windows.ERROR_NO_MORE_ITEMS) {
			break
		}
		if e != nil {
			return nil, e
		}
		instance, e := set.DeviceInstanceID(dev)
		if e != nil {
			return nil, e
		}
		if !strings.Contains(strings.ToLower(instance), `vid_0b05&pid_`) {
			continue
		}
		var typ, size uint32
		var value [8]byte
		ok, _, callErr := openRGBGetProperty.Call(uintptr(set), uintptr(unsafe.Pointer(dev)), uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&value[0])), 8, uintptr(unsafe.Pointer(&size)), 0)
		if ok == 0 || typ != uint32(windows.DEVPROP_TYPE_FILETIME) || size != 8 {
			return nil, fmt.Errorf("native HID arrival generation unavailable: %v", callErr)
		}
		arrival := binary.LittleEndian.Uint64(value[:])
		if arrival == 0 {
			return nil, errors.New("native HID has no current arrival generation")
		}
		births[strings.ToLower(instance)] = birth{uint32(dev.DevInst), arrival}
	}
	var result []OpenRGBNativeNode
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		lower := strings.ToLower(path)
		if !strings.Contains(lower, "vid_0b05&pid_") {
			continue
		}
		// Metadata-only open. No reports, descriptor requests, or lighting writes.
		p, e := windows.UTF16PtrFromString(path)
		if e != nil {
			return nil, e
		}
		h, e := windows.CreateFile(p, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, 0, 0)
		if e != nil {
			return nil, fmt.Errorf("read native ASUS HID metadata: %w", e)
		}
		d, e := openRGBWindowsHIDMetadata(h)
		windows.CloseHandle(h)
		if e != nil {
			return nil, e
		}
		d.StableID = openRGBWindowsInstance(path)
		if !qualifiedNativeHID(d) {
			continue
		}
		born, ok := births[openRGBWindowsInstance(path)]
		if !ok {
			return nil, errors.New("native HID path has no exact present devnode birth")
		}
		result = append(result, OpenRGBNativeNode{Selector: d, Path: path, Generation: fmt.Sprintf("%d:%d", born.devinst, born.arrival)})
	}
	return result, nil
}

func openRGBWindowsHIDMetadata(h windows.Handle) (OpenRGBNativeDevice, error) {
	var attributes struct {
		Size                     uint32
		Vendor, Product, Version uint16
	}
	attributes.Size = uint32(unsafe.Sizeof(attributes))
	ok, _, err := openRGBGetAttributes.Call(uintptr(h), uintptr(unsafe.Pointer(&attributes)))
	if ok == 0 {
		return OpenRGBNativeDevice{}, fmt.Errorf("HID attributes: %v", err)
	}
	var data uintptr
	ok, _, err = openRGBGetPreparsed.Call(uintptr(h), uintptr(unsafe.Pointer(&data)))
	if ok == 0 || data == 0 {
		return OpenRGBNativeDevice{}, fmt.Errorf("HID capabilities: %v", err)
	}
	defer openRGBFreePreparsed.Call(data)
	var caps [64]byte
	status, _, _ := openRGBGetCaps.Call(data, uintptr(unsafe.Pointer(&caps[0])))
	if status != 0x00110000 {
		return OpenRGBNativeDevice{}, fmt.Errorf("HID capability status %#x", status)
	}
	return OpenRGBNativeDevice{Detector: nativeOpenRGBHIDDetector(fmt.Sprintf("%04x:%04x", attributes.Vendor, attributes.Product)), Kind: "hid", HardwareID: fmt.Sprintf("%04x:%04x", attributes.Vendor, attributes.Product), Usage: binary.LittleEndian.Uint16(caps[:2]), UsagePage: binary.LittleEndian.Uint16(caps[2:4])}, nil
}

func nativeOpenRGBAccess(ctx context.Context, node OpenRGBNativeNode) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !qualifiedNativeHID(node.Selector) || openRGBWindowsInstance(node.Path) != node.Selector.StableID {
		return errors.New("unqualified native Windows HID access request")
	}
	p, err := windows.UTF16PtrFromString(node.Path)
	if err != nil {
		return err
	}
	h, err := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OVERLAPPED, 0)
	if err != nil {
		return fmt.Errorf("selected Windows HID account read/write access unavailable; no privilege fallback: %w", err)
	}
	defer windows.CloseHandle(h)
	d, err := openRGBWindowsHIDMetadata(h)
	d.StableID = node.Selector.StableID
	if err != nil || d != node.Selector {
		return errors.Join(errors.New("selected Windows HID identity changed during open"), err)
	}
	return ctx.Err()
}
