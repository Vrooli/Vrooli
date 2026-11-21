package envx

// Env reads environment variables behind an injectable seam.
type Env interface {
	Getenv(key string) string
}

// OS reads from the process environment.
type OS struct{}

func (OS) Getenv(key string) string {
	return getenv(key)
}
