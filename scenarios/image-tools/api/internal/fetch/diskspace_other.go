//go:build !unix

package fetch

// diskAvail is a no-disk-check fallback on platforms without statfs. It reports a
// large value so installs are never blocked by an unknown free-space figure; the
// download itself will still fail on a genuinely full disk.
func diskAvail(string) (int64, error) {
	const large = int64(1) << 60
	return large, nil
}
