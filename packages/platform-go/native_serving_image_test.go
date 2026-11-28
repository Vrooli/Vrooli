package platform

import (
	"context"
	"errors"
	"testing"
)

func TestNativeServingImageRejectsMissingAuthority(t *testing.T) {
	if f, err := OpenCurrentNativeSystemSupervisorImage(nil); !errors.Is(err, ErrOperationIdentity) || f != nil {
		t.Fatalf("nil context: %v %v", f, err)
	}
	if f, err := OpenOwnedServiceRootImage(context.Background(), ServiceRef{}); !errors.Is(err, ErrOperationIdentity) || f != nil {
		t.Fatalf("missing root: %v %v", f, err)
	}
	if f, err := OpenOwnedServiceChildImage(context.Background(), ServiceRef{}, OwnedImageChildIdentity{}); !errors.Is(err, ErrOperationIdentity) || f != nil {
		t.Fatalf("missing child: %v %v", f, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if f, err := OpenCurrentNativeSystemSupervisorImage(ctx); !errors.Is(err, context.Canceled) || f != nil {
		t.Fatalf("canceled: %v %v", f, err)
	}
}
func TestOwnedImageChildIdentityRetainsBirthHandle(t *testing.T) {
	child := &OwnedImageChild{PID: 42, ImageID: "id", ImageHandle: 92, Incarnation: "inc", NativeAccount: "account", ContainmentID: "job"}
	got := child.Identity()
	if got.ImageHandle != 92 || got.ImageID != "id" || got.PID != 42 {
		t.Fatalf("lost birth evidence: %+v", got)
	}
}
