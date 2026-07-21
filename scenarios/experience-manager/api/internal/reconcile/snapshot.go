package reconcile

// Snapshot is the normalized BAS accessibility snapshot contract.
type Snapshot struct {
	Contract      string        `json:"contract"`
	URL           string        `json:"url"`
	ScreenshotRef string        `json:"-"`
	Timing        CaptureTiming `json:"-"`
	Root          AXNode        `json:"root"`
}

// CaptureTiming is the timing reported by Browser Automation Studio for one
// target capture. It deliberately excludes artifact data so historical timing
// records remain small and queryable.
type CaptureTiming struct {
	TotalMilliseconds         int64
	NavigationMilliseconds    int64
	ReadinessWaitMilliseconds int64
	Strategy                  string
	Outcome                   string
}

// AXNode is the subset of bas-accessibility-snapshot/v1 used by structure
// reconciliation.
type AXNode struct {
	Role        string   `json:"role"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Value       string   `json:"value"`
	States      []string `json:"states"`
	Bounds      *Bounds  `json:"bounds"`
	DOM         DOMNode  `json:"dom"`
	Children    []AXNode `json:"children"`
}

type Bounds struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type DOMNode struct {
	TestID string `json:"testid"`
	Tag    string `json:"tag"`
}

// Flatten returns the accessibility tree in reading order.
func (s Snapshot) Flatten() []*AXNode {
	var out []*AXNode
	var walk func(*AXNode)
	walk = func(node *AXNode) {
		if node == nil {
			return
		}
		out = append(out, node)
		for i := range node.Children {
			walk(&node.Children[i])
		}
	}
	walk(&s.Root)
	return out
}

// KeyboardReachable is a conservative Tier 0 approximation from AX state/role.
func (n *AXNode) KeyboardReachable() bool {
	if n == nil {
		return false
	}
	for _, state := range n.States {
		if state == "focusable" || state == "focused" {
			return true
		}
	}
	switch n.Role {
	case "button", "textbox", "link", "tab", "checkbox", "combobox", "menuitem", "switch":
		return true
	default:
		return false
	}
}
