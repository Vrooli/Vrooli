package hostwireless

import "testing"

func TestMOKStatusDoesNotConfuseEnrollmentPendingOrKernelTrust(t *testing.T) {
	for _, tc := range []struct{ output, enrolled, keyring string }{
		{"/proc/self/fd/3 is not enrolled\n", "not_enrolled", "unknown"},
		{"/proc/self/fd/3 is already enrolled\n", "enrolled", "unknown"},
		{"/proc/self/fd/3 is already in the enrollment request\n", "pending", "unknown"},
		{"/proc/self/fd/3 is blocked in dbx\n", "blocked", "unknown"},
		{"/proc/self/fd/3 is already in the built-in trusted keyring\n", "unknown", "subject_key_identifier_found"},
		{"Failed to accesss kernel trusted keyring: Required key not available\n/proc/self/fd/3 is not enrolled\n", "not_enrolled", "unknown"},
		{"/proc/self/fd/3 is already enrolled\n/proc/self/fd/3 is not enrolled", "not_enrolled", "unknown"},
		{"Failed to accesss kernel trusted keyring\n/proc/self/fd/3 is already enrolled", "unknown", "unknown"},
		{"CA of /proc/self/fd/3 is already enrolled", "unknown", "unknown"},
		{"", "unknown", "unknown"},
	} {
		enrollment, keyring := parseMOKCertificate(tc.output)
		if enrollment != tc.enrolled || keyring != tc.keyring {
			t.Fatalf("%q: got %s/%s", tc.output, enrollment, keyring)
		}
	}
}
