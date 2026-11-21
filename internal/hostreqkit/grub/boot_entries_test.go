package grub

import "testing"

func TestBootEntriesExplicitNestedSelectors(t *testing.T) {
	cfg := []byte("submenu 'Advanced options' $menuentry_id_option 'advanced-root' {\nmenuentry 'Ubuntu stock' --class ubuntu $menuentry_id_option 'stock-29' {\n linux /boot/vmlinuz-7.0.0-29-generic root=UUID=x ro\n initrd /boot/initrd.img-7.0.0-29-generic\n}\n}\n")
	got, e := BootEntries(cfg)
	if e != nil || len(got) != 1 || got[0].Selector != "advanced-root>stock-29" || got[0].Kernel != "/boot/vmlinuz-7.0.0-29-generic" {
		t.Fatalf("%+v %v", got, e)
	}
	for _, cfg := range []string{"menuentry 'computed' $menuentry_id_option $generated {\n}", "menuentry 'broken' --id 'a' {\n", "menuentry 'x' --id 'a' {\nlinux /a\nlinux /b\n}"} {
		if _, e := BootEntries([]byte(cfg)); e == nil {
			t.Fatalf("accepted unsupported/ambiguous config %q", cfg)
		}
	}
}
