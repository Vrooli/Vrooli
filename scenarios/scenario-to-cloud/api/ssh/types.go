package ssh

// KeyType represents the type of SSH key.
type KeyType string

const (
	KeyTypeEd25519 KeyType = "ed25519"
	KeyTypeRSA     KeyType = "rsa"
	KeyTypeECDSA   KeyType = "ecdsa"
	KeyTypeDSA     KeyType = "dsa"
	KeyTypeUnknown KeyType = "unknown"
)

// KeyInfo represents information about an SSH key.
type KeyInfo struct {
	Path        string  `json:"path"`
	Type        KeyType `json:"type"`
	Bits        int     `json:"bits,omitempty"`
	Fingerprint string  `json:"fingerprint"`
	Comment     string  `json:"comment,omitempty"`
	CreatedAt   string  `json:"created_at,omitempty"`
}
