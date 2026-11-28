package destinations

import "testing"

func TestParseSFTPLocation(t *testing.T) {
	got, err := parseSFTPLocation("sftp://vrooli@backup.example:2222/backups/vrooli?known_hosts=%2Fhome%2Fuser%2F.ssh%2Fknown_hosts")
	if err != nil {
		t.Fatal(err)
	}
	want := sftpLocation{Host: "backup.example", Port: 2222, Username: "vrooli", RemotePath: "/backups/vrooli", KnownHosts: "/home/user/.ssh/known_hosts"}
	if got != want {
		t.Fatalf("parseSFTPLocation = %#v, want %#v", got, want)
	}

	defaultPort, err := parseSFTPLocation("sftp://backup@backup.example/archive?known_hosts=%2Ftmp%2Fknown_hosts")
	if err != nil {
		t.Fatal(err)
	}
	if defaultPort.Port != 22 {
		t.Fatalf("default SFTP port = %d, want 22", defaultPort.Port)
	}
}

func TestParseSFTPLocationRejectsSecretOrUnsafeMetadata(t *testing.T) {
	for _, input := range []string{
		"sftp://user:password@host/path?known_hosts=%2Ftmp%2Fkh",
		"sftp://user@host/path",
		"sftp://user@host/../path?known_hosts=%2Ftmp%2Fkh",
		"sftp://user@host/?known_hosts=%2Ftmp%2Fkh",
		"sftp://user@host/path?known_hosts=%2Ftmp%2Fkh&token=bad",
		"sftp://host/path?known_hosts=%2Ftmp%2Fkh",
	} {
		t.Run(input, func(t *testing.T) {
			if _, err := parseSFTPLocation(input); err == nil {
				t.Fatalf("parseSFTPLocation(%q) unexpectedly succeeded", input)
			}
		})
	}
}
