package remote

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const maxRecoverySecretBytes = 4096
const maxRecoveryBundleBytes = 1 << 20

// portableRecoveryRequestPreparer reads operator secrets only from an explicit
// stdin flow and loads the ciphertext from a regular local file. Neither value
// is accepted as an argv flag or copied into diagnostics.
func portableRecoveryRequestPreparer(base cliapp.RequestPreparer, stdin io.Reader) cliapp.RequestPreparer {
	return func(ctx cliapp.OperationContext, method string, request proto.Message) (map[string]string, error) {
		if method != "TreasuryAdmin.ExportPortableResearchWalletRecovery" && method != "TreasuryAdmin.RestorePortableResearchWalletRecovery" {
			if base != nil {
				return base(ctx, method, request)
			}
			return nil, nil
		}
		if !ctx.BoolFlag("secret-stdin") {
			return nil, fmt.Errorf("operator recovery secret must be supplied on stdin with --secret-stdin; secret values are never accepted as command-line arguments")
		}
		secret, err := readRecoverySecret(stdin)
		if err != nil {
			return nil, err
		}
		if err := setStringField(request, "operator_secret", secret); err != nil {
			return nil, err
		}
		if method == "TreasuryAdmin.RestorePortableResearchWalletRecovery" {
			path := strings.TrimSpace(ctx.Flag("bundle-file"))
			if path != "" {
				bundle, err := readRecoveryBundle(path)
				if err != nil {
					return nil, err
				}
				if err := setBytesField(request, "bundle", bundle); err != nil {
					return nil, err
				}
			}
		}
		if base != nil {
			return base(ctx, method, request)
		}
		return nil, nil
	}
}

func readRecoverySecret(stdin io.Reader) (string, error) {
	if stdin == nil {
		return "", fmt.Errorf("stdin is unavailable for operator recovery secret")
	}
	data, err := io.ReadAll(io.LimitReader(stdin, maxRecoverySecretBytes+1))
	if err != nil {
		return "", fmt.Errorf("read operator recovery secret from stdin: %w", err)
	}
	if len(data) > maxRecoverySecretBytes {
		return "", fmt.Errorf("operator recovery secret from stdin exceeds the %d-byte limit", maxRecoverySecretBytes)
	}
	secret := strings.TrimRight(string(data), "\r\n")
	if strings.TrimSpace(secret) == "" {
		return "", fmt.Errorf("operator recovery secret from stdin is empty")
	}
	return secret, nil
}

func readRecoveryBundle(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("inspect encrypted recovery bundle: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("encrypted recovery bundle must be a regular file")
	}
	if info.Size() <= 0 || info.Size() > maxRecoveryBundleBytes {
		return nil, fmt.Errorf("encrypted recovery bundle must be between 1 byte and %d bytes", maxRecoveryBundleBytes)
	}
	bundle, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read encrypted recovery bundle: %w", err)
	}
	if len(bundle) != int(info.Size()) {
		return nil, fmt.Errorf("encrypted recovery bundle changed while it was read")
	}
	return bundle, nil
}

func setStringField(message proto.Message, name protoreflect.Name, value string) error {
	field := message.ProtoReflect().Descriptor().Fields().ByName(name)
	if field == nil || field.Kind() != protoreflect.StringKind {
		return fmt.Errorf("request descriptor does not contain string field %q", name)
	}
	message.ProtoReflect().Set(field, protoreflect.ValueOfString(value))
	return nil
}

func setBytesField(message proto.Message, name protoreflect.Name, value []byte) error {
	field := message.ProtoReflect().Descriptor().Fields().ByName(name)
	if field == nil || field.Kind() != protoreflect.BytesKind {
		return fmt.Errorf("request descriptor does not contain bytes field %q", name)
	}
	message.ProtoReflect().Set(field, protoreflect.ValueOfBytes(value))
	return nil
}
