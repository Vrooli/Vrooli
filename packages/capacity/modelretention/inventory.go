package modelretention

import "time"

// InventorySchema identifies the document an owner's `models retention
// --json` prints.
const InventorySchema = "vrooli.model-store-retention/v1"

// InventoryModel is one installed tag as its owner sees it.
type InventoryModel struct {
	Name        string    `json:"name"`
	Digest      string    `json:"digest"`
	Bytes       int64     `json:"bytes"`
	UniqueBytes int64     `json:"unique_bytes"`
	ModifiedAt  time.Time `json:"modified_at"`
	Loaded      bool      `json:"loaded"`
	// Required models are never eviction candidates; RequiredReason says why
	// (a role's primary model, an explicit pin, a default pull).
	Required       bool   `json:"required"`
	RequiredReason string `json:"required_reason,omitempty"`
	// Reacquire is the owner command that brings the model back.
	Reacquire      []string `json:"reacquire"`
	ReacquireExact bool     `json:"reacquire_exact"`
	// Blocked explains why the owner would refuse to evict this model even
	// when it is not required (for example an unpinned digest).
	Blocked string `json:"blocked,omitempty"`
}

// Inventory is an owner's whole-store retention view.
type Inventory struct {
	Schema     string           `json:"schema"`
	Resource   string           `json:"resource"`
	ModelRoot  string           `json:"model_root"`
	StoreBytes int64            `json:"store_bytes"`
	ObservedAt time.Time        `json:"observed_at"`
	Models     []InventoryModel `json:"models"`
}

// EvictResult is what an owner's `models evict --json` prints.
type EvictResult struct {
	Resource       string   `json:"resource"`
	Model          string   `json:"model"`
	Digest         string   `json:"digest"`
	Evicted        bool     `json:"evicted"`
	FreedBytes     int64    `json:"freed_bytes"`
	Reacquire      []string `json:"reacquire"`
	ReacquireExact bool     `json:"reacquire_exact"`
	ReceiptsFile   string   `json:"receipts_file"`
	// Refused is the fence or precondition that stopped the eviction.
	Refused string `json:"refused,omitempty"`
}
