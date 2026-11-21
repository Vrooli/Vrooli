package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqkit"
)

// The owner descriptor lives outside operator HOME. Its asserted owner must
// equal the actual OS file owner. The runtime principal is the same ordinary
// operator. Private filesystem ACLs protect that operator's namespace and do
// not claim isolation between processes using the same principal.
type openRGBNativeOwner struct {
	resourceRoot     string
	SchemaVersion    int    `json:"version"`
	OwnerPrincipal   string `json:"owner_principal"`
	RuntimePrincipal string `json:"runtime_principal"`
	NativeAccount    string `json:"native_account"`
}

func (o openRGBNativeOwner) validate() error {
	if o.SchemaVersion != openRGBNativeOwnerVersion || o.OwnerPrincipal == "" || o.RuntimePrincipal == "" || o.OwnerPrincipal != o.RuntimePrincipal || o.NativeAccount != o.OwnerPrincipal {
		return errors.New("native OpenRGB owner descriptor is incomplete")
	}
	goos := "windows"
	if strings.HasPrefix(o.OwnerPrincipal, "uid:") {
		goos = "darwin"
	}
	if err := hostreqkit.ValidateNativeOpenRGBAccount(goos, o.OwnerPrincipal); err != nil {
		return err
	}
	for _, s := range []string{o.OwnerPrincipal, o.RuntimePrincipal, o.NativeAccount} {
		if len(s) > 256 || strings.ContainsAny(s, "\x00\r\n") {
			return errors.New("native OpenRGB owner descriptor is invalid")
		}
	}
	return nil
}
func (o openRGBNativeOwner) policyFor(path string) (config.PrivatePathPolicy, error) {
	if err := o.validate(); err != nil {
		return config.PrivatePathPolicy{}, err
	}
	root := o.resourceRoot
	if root == "" {
		var err error
		root, err = OpenRGBNativeResourceRoot()
		if err != nil {
			return config.PrivatePathPolicy{}, err
		}
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.Clean(path) != path {
		return config.PrivatePathPolicy{}, errors.New("native private path escapes the exact OpenRGB resource root")
	}
	p := config.PrivatePathPolicy{OwnerPrincipal: o.OwnerPrincipal}
	if rel == "." {
		p.NoInherit = true
	}
	return p, nil
}
func readOpenRGBNativeOwner() (openRGBNativeOwner, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return readOpenRGBNativeOwnerContext(ctx)
}
func readOpenRGBNativeOwnerContext(ctx context.Context) (openRGBNativeOwner, error) {
	var o openRGBNativeOwner
	root, err := OpenRGBNativeResourceRoot()
	if err != nil {
		return o, err
	}
	return readOpenRGBNativeOwnerAt(ctx, root)
}
func readOpenRGBNativeOwnerAt(ctx context.Context, root string) (openRGBNativeOwner, error) {
	var o openRGBNativeOwner
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || strings.ContainsAny(root, "\x00\r\n") {
		return o, errors.New("native owner requires canonical frozen resource root")
	}
	path := filepath.Join(root, "owner.json")
	info, err := os.Lstat(path)
	if err != nil {
		return o, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 4096 {
		return o, errors.New("native OpenRGB owner descriptor has invalid type or size")
	}
	actualOwner, err := config.PrivatePathOwner(path, info)
	if err != nil {
		return o, err
	}
	file, err := os.Open(path)
	if err != nil {
		return o, err
	}
	defer file.Close()
	actual, err := file.Stat()
	if err != nil || !os.SameFile(info, actual) {
		return o, errors.Join(errors.New("native OpenRGB owner descriptor changed while opening"), err)
	}
	data, err := io.ReadAll(io.LimitReader(file, 4097))
	if err != nil {
		return o, err
	}
	if len(data) > 4096 {
		return o, errors.New("native owner descriptor exceeds size bound")
	}
	if err := validateManagedUpdateJSON(data); err != nil {
		return o, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&o); err != nil {
		return o, err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return o, errors.New("native OpenRGB owner descriptor has trailing bytes")
	}
	if err := o.validate(); err != nil {
		return o, err
	}
	o.resourceRoot = root
	if actualOwner != o.OwnerPrincipal {
		return o, errors.New("native OpenRGB descriptor owner differs from OS ownership")
	}
	if err := ctx.Err(); err != nil {
		return o, err
	}
	policy, err := o.policyFor(path)
	if err != nil {
		return o, err
	}
	if err := config.VerifyPrivatePath(path, actual, policy); err != nil {
		return o, err
	}
	return o, nil
}

// prepareOpenRGBNativeOwner is called only by explicit native setup/apply.
// Existing descriptors are never rebound to a different operator or account.
func prepareOpenRGBNativeOwner(ctx context.Context, account platform.OwnedServiceAccount) (openRGBNativeOwner, error) {
	var o openRGBNativeOwner
	if err := ctx.Err(); err != nil {
		return o, err
	}
	principal, err := openRGBNativeOperatorPrincipal()
	if err != nil {
		return o, err
	}
	root, err := OpenRGBNativeResourceRoot()
	if err != nil {
		return o, err
	}
	o = openRGBNativeOwner{SchemaVersion: openRGBNativeOwnerVersion, OwnerPrincipal: principal, RuntimePrincipal: account.Principal, NativeAccount: account.Name, resourceRoot: root}
	if err := o.validate(); err != nil {
		return o, err
	}
	existing, err := readOpenRGBNativeOwner()
	if err == nil {
		if existing != o {
			return o, errors.New("native OpenRGB resource belongs to a different enrolled owner")
		}
		return existing, nil
	}
	if !os.IsNotExist(err) {
		return o, err
	}
	policy, err := o.policyFor(root)
	if err != nil {
		return o, err
	}
	if err := config.ConfigurePrivateDirectoryAccess(root, policy); err != nil {
		return o, err
	}
	data, err := json.Marshal(o)
	if err != nil {
		return o, err
	}
	if err := config.WritePrivateFileDurable(filepath.Join(root, "owner.json"), data, 0600, policy); err != nil {
		return o, err
	}
	return readOpenRGBNativeOwner()
}
