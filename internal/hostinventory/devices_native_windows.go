package hostinventory

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sys/windows"
)

var (
	// DEVPKEY_Device_Parent from devpkey.h.
	pnpParentProperty = windows.DEVPROPKEY{
		FmtID: windows.DEVPROPGUID{Data1: 0x4340a6c5, Data2: 0x93fa, Data3: 0x4706, Data4: [8]byte{0x97, 0x2c, 0x7b, 0x64, 0x80, 0x08, 0xa5, 0xa7}},
		PID:   8,
	}
	// GUID_DEVINTERFACE_NET from Ndisguid.h.
	pnpNetworkInterface = windows.GUID{Data1: 0xcac88484, Data2: 0x7515, Data3: 0x4c03, Data4: [8]byte{0x82, 0xe6, 0x71, 0xa8, 0x7a, 0xba, 0xc3, 0x61}}
)

func collectNativeDevices(ctx context.Context, _ CommandRunner) NativeDeviceObservation {
	if err := ctx.Err(); err != nil {
		return windowsPnPObservation(nil, err)
	}
	set, err := windows.SetupDiGetClassDevsEx(nil, "", 0, windows.DIGCF_PRESENT|windows.DIGCF_ALLCLASSES, 0, "")
	if err != nil {
		return windowsPnPObservation(nil, fmt.Errorf("open the present PnP device set: %w", err))
	}
	defer set.Close()

	var nodes []windowsDeviceNode
	var walkErr error
	for index := 0; ; index++ {
		if err := ctx.Err(); err != nil {
			walkErr = errors.Join(walkErr, err)
			break
		}
		info, err := set.EnumDeviceInfo(index)
		if errors.Is(err, windows.ERROR_NO_MORE_ITEMS) {
			break
		}
		if err != nil {
			walkErr = errors.Join(walkErr, fmt.Errorf("enumerate PnP device %d: %w", index, err))
			break
		}
		id, err := set.DeviceInstanceID(info)
		if err != nil || id == "" {
			walkErr = errors.Join(walkErr, fmt.Errorf("PnP device %d has no readable instance ID: %v", index, err))
			continue
		}
		bus, _, _ := strings.Cut(strings.ToUpper(id), "\\")
		switch bus {
		case "PCI", "USB", "ACPI", "SD":
			nodes = append(nodes, readWindowsDeviceNode(set, info, id))
		}
	}
	return windowsPnPObservation(nodes, walkErr)
}

func readWindowsDeviceNode(set windows.DevInfo, info *windows.DevInfoData, id string) windowsDeviceNode {
	node := windowsDeviceNode{id: id}
	var classErr, hardwareErr, compatibleErr, serviceErr error
	node.class, classErr = pnpRegistryProperty[string](set, info, windows.SPDRP_CLASS)
	node.hardwareIDs, hardwareErr = pnpRegistryProperty[[]string](set, info, windows.SPDRP_HARDWAREID)
	node.compatibleIDs, compatibleErr = pnpRegistryProperty[[]string](set, info, windows.SPDRP_COMPATIBLEIDS)
	node.identityErr = errors.Join(classErr, hardwareErr, compatibleErr)
	node.description, _ = pnpRegistryProperty[string](set, info, windows.SPDRP_FRIENDLYNAME)
	if node.description == "" {
		node.description, _ = pnpRegistryProperty[string](set, info, windows.SPDRP_DEVICEDESC)
	}
	node.vendor, _ = pnpRegistryProperty[string](set, info, windows.SPDRP_MFG)
	node.service, serviceErr = pnpRegistryProperty[string](set, info, windows.SPDRP_SERVICE)
	if strings.HasPrefix(strings.ToUpper(id), `USB\`) && windowsIDField(id, "MI_", 2) != "" {
		parent, err := windows.SetupDiGetDeviceProperty(set, info, &pnpParentProperty)
		if err != nil {
			node.identityErr = errors.Join(node.identityErr, fmt.Errorf("read USB function parent: %w", err))
		}
		node.parent, _ = parent.(string)
	}
	var status, problem uint32
	statusErr := windows.CM_Get_DevNode_Status(&status, &problem, info.DevInst, 0)
	if statusErr == nil {
		node.started = status&windows.DN_STARTED != 0
		if status&windows.DN_HAS_PROBLEM != 0 {
			node.problem = problem
		}
	} else {
		statusErr = fmt.Errorf("read PnP devnode status: %w", statusErr)
	}
	node.bindingErr = errors.Join(serviceErr, statusErr)

	if _, class := windowsDeviceClass(node); class == "network controller" {
		interfaces, err := windows.CM_Get_Device_Interface_List(id, &pnpNetworkInterface, windows.CM_GET_DEVICE_INTERFACE_LIST_PRESENT)
		switch {
		case err == nil, errors.Is(err, windows.ERROR_NO_SUCH_DEVICE_INTERFACE):
			node.networkObserved = true
			node.networkPresent = len(interfaces) > 0
		default:
			node.networkErr = fmt.Errorf("enumerate active PnP network device interfaces: %w", err)
		}
	}
	return node
}

func pnpRegistryProperty[T string | []string](set windows.DevInfo, info *windows.DevInfoData, property windows.SPDRP) (T, error) {
	var zero T
	value, err := set.DeviceRegistryProperty(info, property)
	if errors.Is(err, windows.ERROR_INVALID_DATA) {
		// SetupDiGetDeviceRegistryProperty reports this for an unset property.
		return zero, nil
	}
	if err != nil {
		return zero, fmt.Errorf("read PnP device property %d: %w", property, err)
	}
	result, ok := value.(T)
	if !ok {
		return zero, fmt.Errorf("PnP device property %d has unexpected type %T", property, value)
	}
	return result, nil
}
