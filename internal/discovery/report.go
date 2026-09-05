package discovery

type Failure struct {
	Kind  string `json:"kind"`
	Name  string `json:"name"`
	Path  string `json:"path,omitempty"`
	Stage string `json:"stage,omitempty"`
	Error string `json:"error"`
}

type Report[T any] struct {
	Items    []T       `json:"items"`
	Failures []Failure `json:"failures,omitempty"`
}
