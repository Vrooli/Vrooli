package sysmounts

import "testing"

func TestMountEpisodeIDFromInfoUsesCurrentMountGeneration(t *testing.T) {
	for _, tc := range []struct {
		name       string
		data       string
		mountpoint string
		want       string
	}{
		{name: "exact mount", data: "36 25 8:1 / /media/Elements rw,relatime - fuseblk /dev/sda1 rw\n", mountpoint: "/media/Elements", want: "boot/ns/36"},
		{name: "escaped space", data: "37 25 8:1 / /media/Elements\\040Drive rw,relatime - fuseblk /dev/sda1 rw\n", mountpoint: "/media/Elements Drive", want: "boot/ns/37"},
		{name: "newest stacked mount", data: "36 25 8:1 / /media/Elements rw - fuseblk /dev/sda1 rw\n78 25 8:1 / /media/Elements rw - fuseblk /dev/sda1 rw\n", mountpoint: "/media/Elements", want: "boot/ns/78"},
		{name: "different mountpoint", data: "36 25 8:1 / /media/Other rw - fuseblk /dev/sda1 rw\n", mountpoint: "/media/Elements", want: ""},
		{name: "invalid id", data: "bad 25 8:1 / /media/Elements rw - fuseblk /dev/sda1 rw\n", mountpoint: "/media/Elements", want: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := mountEpisodeIDFromInfo([]byte(tc.data), tc.mountpoint, "boot", "ns"); got != tc.want {
				t.Fatalf("mount episode ID = %q, want %q", got, tc.want)
			}
		})
	}
	if got := mountEpisodeIDFromInfo([]byte("36 25 8:1 / /media/Elements rw - fuseblk /dev/sda1 rw\n"), "/media/Elements", "", "ns"); got != "" {
		t.Fatalf("missing boot ID must fail closed, got %q", got)
	}
}
