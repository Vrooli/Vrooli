package targetmodel

import "testing"

func TestProjectDesktopPlatform(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		wantOS   string
		wantArch string
		wantErr  bool
	}{
		{name: "linux x64", target: "linux-x64", wantOS: DesktopOSLinux, wantArch: DesktopArchAMD64},
		{name: "linux amd64", target: "linux-amd64", wantOS: DesktopOSLinux, wantArch: DesktopArchAMD64},
		{name: "linux arm64", target: "linux-arm64", wantOS: DesktopOSLinux, wantArch: DesktopArchARM64},
		{name: "linux os only", target: "linux", wantOS: DesktopOSLinux},
		{name: "darwin arm64", target: "darwin-arm64", wantOS: DesktopOSMac, wantArch: DesktopArchARM64},
		{name: "darwin x64", target: "darwin-x64", wantOS: DesktopOSMac, wantArch: DesktopArchAMD64},
		{name: "mac arm64", target: "mac-arm64", wantOS: DesktopOSMac, wantArch: DesktopArchARM64},
		{name: "macos alias", target: "macos", wantOS: DesktopOSMac},
		{name: "win x64", target: "win-x64", wantOS: DesktopOSWindows, wantArch: DesktopArchAMD64},
		{name: "windows amd64", target: "windows-amd64", wantOS: DesktopOSWindows, wantArch: DesktopArchAMD64},
		{name: "windows os only", target: "windows", wantOS: DesktopOSWindows},
		{name: "case and whitespace insensitive", target: "  Linux-X64 ", wantOS: DesktopOSLinux, wantArch: DesktopArchAMD64},
		{name: "underscore separator", target: "linux_x86_64", wantOS: DesktopOSLinux, wantArch: DesktopArchAMD64},
		{name: "empty", target: "  ", wantErr: true},
		{name: "unknown os", target: "freebsd-x64", wantErr: true},
		{name: "unknown arch", target: "linux-mips", wantErr: true},
		{name: "too many segments", target: "linux-x64-extra", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			osPlatform, arch, err := ProjectDesktopPlatform(tt.target)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ProjectDesktopPlatform(%q) = (%q, %q), want error", tt.target, osPlatform, arch)
				}
				return
			}
			if err != nil {
				t.Fatalf("ProjectDesktopPlatform(%q) error: %v", tt.target, err)
			}
			if osPlatform != tt.wantOS || arch != tt.wantArch {
				t.Fatalf("ProjectDesktopPlatform(%q) = (%q, %q), want (%q, %q)", tt.target, osPlatform, arch, tt.wantOS, tt.wantArch)
			}
		})
	}
}
