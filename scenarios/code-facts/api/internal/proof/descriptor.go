package proof

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/packages/proto/descriptorimage"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

type ProtoReader interface {
	ReadFile(context.Context, string) ([]byte, error)
}

type OSProtoReader struct{}

func (OSProtoReader) ReadFile(ctx context.Context, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

type DescriptorSource struct {
	Source   *descriptorimage.Source
	RepoRoot string
	Reader   ProtoReader
}

func (s DescriptorSource) Snapshot(ctx context.Context) (ContractSnapshot, error) {
	if s.Source == nil || s.Reader == nil || strings.TrimSpace(s.RepoRoot) == "" {
		return ContractSnapshot{}, fmt.Errorf("descriptor contract source requires source, repository root, and proto reader")
	}
	if err := ctx.Err(); err != nil {
		return ContractSnapshot{}, err
	}
	descriptorSnapshot, reloadErr := s.Source.Snapshot()
	if descriptorSnapshot == nil {
		return ContractSnapshot{}, reloadErr
	}
	result := ContractSnapshot{
		Digest:               descriptorSnapshot.Digest,
		DescriptorGeneration: descriptorSnapshot.Generation,
	}
	if reloadErr != nil {
		result.LastReloadFailure = reloadErr.Error()
		result.LastReloadFailureAt = s.Source.LastReloadFailureAt()
	}
	files := append([]*descriptorpb.FileDescriptorProto(nil), descriptorSnapshot.Descriptor.GetFile()...)
	sort.Slice(files, func(i, j int) bool { return files[i].GetName() < files[j].GetName() })
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return ContractSnapshot{}, err
		}
		path := authoritativeProtoPath(s.RepoRoot, file.GetName())
		payload, err := s.Reader.ReadFile(ctx, path)
		sourceHash := ""
		if err != nil {
			result.ProvenanceFailures = append(result.ProvenanceFailures, fmt.Sprintf("%s: %v", filepath.ToSlash(path), err))
		} else {
			digest := sha256.Sum256(payload)
			sourceHash = "sha256:" + hex.EncodeToString(digest[:])
		}
		result.Contracts = append(result.Contracts, extractFileContracts(file, filepath.ToSlash(path), sourceHash, descriptorSnapshot.Digest)...)
	}
	sort.Slice(result.Contracts, func(i, j int) bool { return result.Contracts[i].ID < result.Contracts[j].ID })
	sort.Strings(result.ProvenanceFailures)
	return result, nil
}

func authoritativeProtoPath(repoRoot, descriptorName string) string {
	name := filepath.FromSlash(strings.TrimPrefix(descriptorName, "packages/proto/schemas/"))
	return filepath.Join(filepath.Clean(repoRoot), "packages", "proto", "schemas", name)
}

func extractFileContracts(file *descriptorpb.FileDescriptorProto, path, sourceHash, digest string) []Contract {
	locations := sourceLocations(file.GetSourceCodeInfo())
	packageName := file.GetPackage()
	fileID := contractID("proto_file", file.GetName())
	out := []Contract{newContract(fileID, "proto_file", filepath.Base(file.GetName()), file.GetName(), "", path, sourceHash, digest, locations[""], map[string]string{
		"package": packageName,
		"syntax":  file.GetSyntax(),
	})}
	for index, dependency := range file.GetDependency() {
		fullName := file.GetName() + "->" + dependency
		out = append(out, newContract(contractID("import", fullName), "import", dependency, fullName, fileID, path, sourceHash, digest, locations[pathKey(3, index)], map[string]string{"target": dependency}))
	}
	for index, service := range file.GetService() {
		fullName := qualify(packageName, service.GetName())
		serviceID := contractID("service", fullName)
		out = append(out, newContract(serviceID, "service", service.GetName(), fullName, fileID, path, sourceHash, digest, locations[pathKey(6, index)], optionAttributes(service.GetOptions(), service.GetOptions().GetDeprecated())))
		for methodIndex, method := range service.GetMethod() {
			methodName := fullName + "." + method.GetName()
			attributes := optionAttributes(method.GetOptions(), method.GetOptions().GetDeprecated())
			attributes["input_type"] = method.GetInputType()
			attributes["output_type"] = method.GetOutputType()
			attributes["client_streaming"] = strconv.FormatBool(method.GetClientStreaming())
			attributes["server_streaming"] = strconv.FormatBool(method.GetServerStreaming())
			methodID := contractID("method", methodName)
			out = append(out, newContract(methodID, "method", method.GetName(), methodName, serviceID, path, sourceHash, digest, locations[pathKey(6, index, 2, methodIndex)], attributes))
			out = append(out, generatedAliases(methodID, methodName, path, sourceHash, digest)...)
		}
	}
	for index, message := range file.GetMessageType() {
		out = append(out, extractMessage(message, packageName, fileID, path, sourceHash, digest, locations, []int{4, index})...)
	}
	for index, enum := range file.GetEnumType() {
		out = append(out, extractEnum(enum, packageName, fileID, path, sourceHash, digest, locations, []int{5, index})...)
	}
	return out
}

func extractMessage(message *descriptorpb.DescriptorProto, parentName, parentID, path, sourceHash, digest string, locations map[string]sourceLocation, descriptorPath []int) []Contract {
	fullName := qualify(parentName, message.GetName())
	id := contractID("message", fullName)
	out := []Contract{newContract(id, "message", message.GetName(), fullName, parentID, path, sourceHash, digest, locations[pathKey(descriptorPath...)], optionAttributes(message.GetOptions(), message.GetOptions().GetDeprecated()))}
	for index, field := range message.GetField() {
		fieldName := fullName + "." + field.GetName()
		attributes := optionAttributes(field.GetOptions(), field.GetOptions().GetDeprecated())
		attributes["number"] = strconv.Itoa(int(field.GetNumber()))
		attributes["label"] = field.GetLabel().String()
		attributes["type"] = field.GetType().String()
		if field.GetTypeName() != "" {
			attributes["type_name"] = field.GetTypeName()
		}
		fieldPath := appendPath(descriptorPath, 2, index)
		out = append(out, newContract(contractID("field", fieldName), "field", field.GetName(), fieldName, id, path, sourceHash, digest, locations[pathKey(fieldPath...)], attributes))
	}
	for index, nested := range message.GetNestedType() {
		out = append(out, extractMessage(nested, fullName, id, path, sourceHash, digest, locations, appendPath(descriptorPath, 3, index))...)
	}
	for index, enum := range message.GetEnumType() {
		out = append(out, extractEnum(enum, fullName, id, path, sourceHash, digest, locations, appendPath(descriptorPath, 4, index))...)
	}
	return out
}

func extractEnum(enum *descriptorpb.EnumDescriptorProto, parentName, parentID, path, sourceHash, digest string, locations map[string]sourceLocation, descriptorPath []int) []Contract {
	fullName := qualify(parentName, enum.GetName())
	id := contractID("enum", fullName)
	out := []Contract{newContract(id, "enum", enum.GetName(), fullName, parentID, path, sourceHash, digest, locations[pathKey(descriptorPath...)], optionAttributes(enum.GetOptions(), enum.GetOptions().GetDeprecated()))}
	for index, value := range enum.GetValue() {
		valueName := fullName + "." + value.GetName()
		attributes := optionAttributes(value.GetOptions(), value.GetOptions().GetDeprecated())
		attributes["number"] = strconv.Itoa(int(value.GetNumber()))
		valuePath := appendPath(descriptorPath, 2, index)
		out = append(out, newContract(contractID("enum_value", valueName), "enum_value", value.GetName(), valueName, id, path, sourceHash, digest, locations[pathKey(valuePath...)], attributes))
	}
	return out
}

func generatedAliases(parentID, fullName, path, sourceHash, digest string) []Contract {
	aliases := make([]Contract, 0, 2)
	for _, language := range []string{"go", "typescript"} {
		aliasName := language + ":" + fullName
		aliases = append(aliases, Contract{
			ID: contractID("generated_alias", aliasName), Kind: "generated_alias", Name: fullName,
			FullName: aliasName, ParentID: parentID, Path: path, SourceHash: sourceHash, Digest: digest,
			Attributes: map[string]string{"language": language, "authority": "resolved_contract", "target_contract_id": parentID},
		})
	}
	return aliases
}

func optionAttributes(options proto.Message, deprecated bool) map[string]string {
	attributes := map[string]string{"deprecated": strconv.FormatBool(deprecated)}
	if options == nil {
		return attributes
	}
	if encoded := strings.TrimSpace(prototext.MarshalOptions{Multiline: false}.Format(options)); encoded != "" {
		attributes["options"] = encoded
	}
	return attributes
}

type sourceLocation struct {
	startLine int
	endLine   int
	comment   string
}

func sourceLocations(info *descriptorpb.SourceCodeInfo) map[string]sourceLocation {
	out := map[string]sourceLocation{}
	if info == nil {
		return out
	}
	for _, location := range info.GetLocation() {
		span := location.GetSpan()
		startLine, endLine := 0, 0
		if len(span) >= 3 {
			startLine = int(span[0]) + 1
			endLine = startLine
			if len(span) >= 4 {
				endLine = int(span[2]) + 1
			}
		}
		comment := strings.TrimSpace(strings.Join([]string{location.GetLeadingComments(), location.GetTrailingComments()}, "\n"))
		out[pathKey32(location.GetPath())] = sourceLocation{startLine: startLine, endLine: endLine, comment: comment}
	}
	return out
}

func newContract(id, kind, name, fullName, parentID, path, sourceHash, digest string, location sourceLocation, attributes map[string]string) Contract {
	if attributes == nil {
		attributes = map[string]string{}
	}
	if location.startLine == 0 {
		attributes["provenance"] = "range_missing"
	} else {
		attributes["provenance"] = "source_info"
	}
	return Contract{
		ID: id, Kind: kind, Name: name, FullName: fullName, ParentID: parentID,
		Path: path, StartLine: location.startLine, EndLine: location.endLine, Comment: location.comment,
		SourceHash: sourceHash, Digest: digest, Attributes: attributes,
	}
}

func contractID(kind, fullName string) string {
	return "contract:" + kind + ":" + strings.TrimPrefix(fullName, ".")
}

func qualify(parent, name string) string {
	if parent == "" {
		return name
	}
	return strings.TrimPrefix(parent, ".") + "." + name
}

func pathKey(path ...int) string {
	parts := make([]string, len(path))
	for index, value := range path {
		parts[index] = strconv.Itoa(value)
	}
	return strings.Join(parts, ".")
}

func pathKey32(path []int32) string {
	parts := make([]string, len(path))
	for index, value := range path {
		parts[index] = strconv.Itoa(int(value))
	}
	return strings.Join(parts, ".")
}

func appendPath(path []int, values ...int) []int {
	out := make([]int, 0, len(path)+len(values))
	out = append(out, path...)
	out = append(out, values...)
	return out
}

var (
	_ ContractSource = DescriptorSource{}
	_ ProtoReader    = OSProtoReader{}
)
