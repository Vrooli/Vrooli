package kopiaregistry

import (
	"reflect"

	"github.com/vrooli/vrooli/internal/stateformat"
)

// kopiaRegistrySchemaDigest pins the stored shape of this store's format. The digest is
// derived from the types, so a stored field change moves it. A deliberate change
// must bump the store's write version and re-pin.
const kopiaRegistrySchemaDigest = "f94f2c7fc29aa361da949f52b04f5468b3d9f87f4a3ec7787df443b8882b303a"

// StateFormat describes the Kopia registry file the resource plane reads. It lives under the platform state directory, outside the runtime home, so this store declares the contract only.
func StateFormat() (stateformat.Store, error) {
	digest, err := stateformat.DigestTypes(
		reflect.TypeFor[Entry](),
		reflect.TypeFor[fileShape](),
	)
	if err != nil {
		return stateformat.Store{}, err
	}
	return stateformat.Store{
		Name:  "kopia-registry",
		Write: 1,
		Reads: []stateformat.Version{1},
		Schemas: map[stateformat.Version]string{
			1: digest,
		},
		Vocabulary: stateformat.CanonicalVocabulary(
			BackendFilesystem,
			BackendS3,
			BackendSFTP,
			PassphraseField,
			S3AccessKeyIDField,
			S3SecretAccessKeyField,
			SFTPPrivateKeyField,
		),
		ContractOnly: true,
	}, nil
}
