package hostinventory

import (
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

// LinuxBluetoothController is passive BlueZ evidence. Registration and power
// do not establish peer function; only an existing paired, connected peer does.
type LinuxBluetoothController struct {
	Powered       bool
	PowerKnown    bool
	PeerConnected bool
}

var blueZPeerPath = regexp.MustCompile(`^/org/bluez/hci[0-9]+/dev_(?:[A-Fa-f0-9]{2}_){5}[A-Fa-f0-9]{2}$`)

// ParseLinuxBluetoothControllers decodes the bounded lossless busctl object
// manager shapes shared by inventory and governed wireless verification.
func ParseLinuxBluetoothControllers(data []byte) (map[string]LinuxBluetoothController, error) {
	if len(data) > 64<<10 {
		return nil, errors.New("BlueZ message exceeds bound")
	}
	var message struct {
		Type string            `json:"type"`
		Data []json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &message); err != nil {
		return nil, err
	}
	if message.Type != "a{oa{sa{sv}}}" || len(message.Data) != 1 {
		return nil, errors.New("unexpected BlueZ object-manager signature")
	}
	objects, err := blueZDictionary(message.Data[0])
	if err != nil {
		return nil, err
	}
	controllers := map[string]LinuxBluetoothController{}
	for path, object := range objects {
		name := strings.TrimPrefix(path, "/org/bluez/")
		if !isLinuxHCIName(name) {
			continue
		}
		interfaces, err := blueZDictionary(object)
		if err != nil {
			return nil, err
		}
		adapter, exists := interfaces["org.bluez.Adapter1"]
		if !exists {
			continue
		}
		properties, err := blueZDictionary(adapter)
		if err != nil {
			return nil, err
		}
		powered, known := blueZBoolean(properties["Powered"])
		controller := LinuxBluetoothController{Powered: powered, PowerKnown: known}
		for peerPath, peerObject := range objects {
			if !strings.HasPrefix(peerPath, path+"/dev_") || !blueZPeerPath.MatchString(peerPath) {
				continue
			}
			peerInterfaces, err := blueZDictionary(peerObject)
			if err != nil {
				continue
			}
			peerProperties, err := blueZDictionary(peerInterfaces["org.bluez.Device1"])
			if err != nil {
				continue
			}
			paired, _ := blueZBoolean(peerProperties["Paired"])
			connected, _ := blueZBoolean(peerProperties["Connected"])
			controller.PeerConnected = controller.PeerConnected || (paired && connected)
		}
		controllers[name] = controller
	}
	return controllers, nil
}

// busctl has emitted both JSON dictionaries and alternating key/value arrays
// across supported systemd releases. Decode those exact lossless shapes only.
func blueZDictionary(data json.RawMessage) (map[string]json.RawMessage, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return nil, errors.New("missing BlueZ dictionary")
	}
	var object map[string]json.RawMessage
	if len(data) > 0 && data[0] == '{' {
		decoder := json.NewDecoder(bytes.NewReader(data))
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		object = map[string]json.RawMessage{}
		for decoder.More() {
			token, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			key, ok := token.(string)
			if !ok {
				return nil, errors.New("invalid BlueZ dictionary key")
			}
			if _, exists := object[key]; exists {
				return nil, errors.New("duplicate BlueZ dictionary key")
			}
			var value json.RawMessage
			if err := decoder.Decode(&value); err != nil {
				return nil, err
			}
			object[key] = value
			if len(object) > 256 {
				return nil, errors.New("BlueZ dictionary exceeds bound")
			}
		}
		if _, err := decoder.Token(); err != nil {
			return nil, err
		}
		return object, nil
	}
	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	if len(items)%2 != 0 || len(items) > 512 {
		return nil, errors.New("invalid bounded BlueZ dictionary")
	}
	object = map[string]json.RawMessage{}
	for i := 0; i < len(items); i += 2 {
		var key string
		if err := json.Unmarshal(items[i], &key); err != nil {
			return nil, err
		}
		if _, ok := object[key]; ok {
			return nil, errors.New("duplicate BlueZ dictionary key")
		}
		object[key] = items[i+1]
	}
	return object, nil
}

func blueZBoolean(data json.RawMessage) (bool, bool) {
	var value struct {
		Type string `json:"type"`
		Data *bool  `json:"data"`
	}
	if json.Unmarshal(data, &value) != nil || value.Type != "b" || value.Data == nil {
		return false, false
	}
	return *value.Data, true
}
