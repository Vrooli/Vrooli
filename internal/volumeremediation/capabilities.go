package volumeremediation

// ContractVersion identifies the JSON contract exposed by
// `vrooli host volume capabilities`. Callers use this read-only probe to
// detect an older installed control-plane binary before requesting a volume
// operation.
const ContractVersion = "host-volume-v1"

// Capabilities is the read-only host-volume contract advertised by the
// control-plane CLI. Keep this deliberately small: it describes the guarded
// verbs, not a promise that a particular device or filesystem is usable.
type Capabilities struct {
	Contract string   `json:"contract"`
	Actions  []string `json:"actions"`
}

// AdvertisedCapabilities returns a fresh value so callers cannot mutate the
// registry accidentally.
func AdvertisedCapabilities() Capabilities {
	return Capabilities{
		Contract: ContractVersion,
		Actions: []string{
			string(ActionInspect),
			string(ActionCheck),
			string(ActionRepair),
			string(ActionUnmount),
			string(ActionMountReadWrite),
		},
	}
}
