package catalogconformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"audio-tools/internal/operationcatalog"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/audio"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/summarize"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type cliBinding struct {
	Kind    string `json:"kind"`
	Service string `json:"service"`
	Method  string `json:"method"`
}

type cliCommand struct {
	Name    string     `json:"name"`
	Binding cliBinding `json:"binding"`
}

type cliManifest struct {
	Groups []struct {
		Name     string       `json:"name"`
		Commands []cliCommand `json:"commands"`
	} `json:"groups"`
}

// Each listed field path is the user-facing input/output represented by the
// operation's abstract port. Stream paths traverse the payload/event oneofs.
var operationWirePaths = map[string]struct {
	protoPackage string
	family       string
	input        string
	output       string
}{
	"transcribe":         {protoPackage: "stt", family: "stt", input: "audio", output: "text"},
	"transcribe_stream":  {protoPackage: "stt", family: "stt", input: "audio_chunk.audio", output: "partial.text"},
	"synthesize":         {protoPackage: "tts", family: "tts", input: "text", output: "audio"},
	"synthesize_stream":  {protoPackage: "tts", family: "tts", input: "text", output: "audio"},
	"summarize":          {protoPackage: "summarize", family: "summarize", input: "text", output: "text"},
	"transcode":          {protoPackage: "audio", family: "transform", input: "audio", output: "audio"},
	"trim":               {protoPackage: "audio", family: "transform", input: "audio", output: "audio"},
	"merge":              {protoPackage: "audio", family: "transform", input: "sources", output: "audio"},
	"split":              {protoPackage: "audio", family: "transform", input: "audio", output: "chunks"},
	"fade":               {protoPackage: "audio", family: "transform", input: "audio", output: "audio"},
	"volume":             {protoPackage: "audio", family: "transform", input: "audio", output: "audio"},
	"normalize":          {protoPackage: "audio", family: "transform", input: "audio", output: "audio"},
	"extract_metadata":   {protoPackage: "audio", family: "transform", input: "audio", output: "metadata"},
}

func TestOperationsMatchConnectWireAndCLIContracts(t *testing.T) {
	var manifest cliManifest
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "cli", "manifest.json"))
	if err != nil {
		t.Fatalf("read audio-tools CLI manifest: %v", err)
	}
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("decode audio-tools CLI manifest: %v", err)
	}

	catalog, err := operationcatalog.Default()
	if err != nil {
		t.Fatalf("load operation catalog: %v", err)
	}
	for _, op := range catalog.Operations {
		if op.Family == "stt" || op.Family == "tts" || op.Family == "summarize" || op.Family == "transform" {
			if _, covered := operationWirePaths[op.ID]; !covered {
				t.Errorf("operation %q has no Connect/CLI conformance mapping", op.ID)
			}
		}
	}
	for operationID, paths := range operationWirePaths {
		op, ok := catalog.Lookup(operationID)
		if !ok {
			t.Errorf("operation %q missing from operation catalog", operationID)
			continue
		}
		if op.Family != paths.family {
			t.Errorf("operation %q family %q disagrees with its wire mapping family %q", operationID, op.Family, paths.family)
			continue
		}
		serviceName := "vrooli.audio_tools.v1." + paths.protoPackage + "." + op.CLI.Service
		service, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(serviceName))
		if err != nil {
			t.Errorf("operation %q service %q missing from generated proto descriptors: %v", operationID, serviceName, err)
			continue
		}
		serviceDesc, ok := service.(protoreflect.ServiceDescriptor)
		if !ok {
			t.Errorf("descriptor %q is not a service", serviceName)
			continue
		}
		method := serviceDesc.Methods().ByName(protoreflect.Name(op.CLI.Method))
		if method == nil {
			t.Errorf("operation %q method %s.%s missing from generated proto descriptor", operationID, serviceName, op.CLI.Method)
			continue
		}
		wantClient, wantServer := op.Transport == "bidi", op.Transport == "server_stream" || op.Transport == "bidi"
		if method.IsStreamingClient() != wantClient || method.IsStreamingServer() != wantServer {
			t.Errorf("operation %q transport %q disagrees with Connect method streaming client=%t server=%t", operationID, op.Transport, method.IsStreamingClient(), method.IsStreamingServer())
		}
		if len(op.Ports.Inputs) != 1 || len(op.Ports.Outputs) != 1 {
			t.Errorf("operation %q expects one declared input and output port", operationID)
			continue
		}
		input := resolveWirePath(t, operationID, method.Input(), paths.input)
		output := resolveWirePath(t, operationID, method.Output(), paths.output)
		assertWirePort(t, operationID, "input", op.Ports.Inputs[0], input)
		assertWirePort(t, operationID, "output", op.Ports.Outputs[0], output)

		var cliCommand *cliCommand
		for i := range manifest.Groups {
			if manifest.Groups[i].Name != op.CLI.Group {
				continue
			}
			for j := range manifest.Groups[i].Commands {
				command := &manifest.Groups[i].Commands[j]
				if command.Name == op.CLI.Verb {
					cliCommand = command
				}
			}
		}
		if cliCommand == nil {
			t.Errorf("operation %q has no CLI manifest command %s %s", operationID, op.CLI.Group, op.CLI.Verb)
			continue
		}
		if cliCommand.Binding.Kind != "connect-rpc" || cliCommand.Binding.Service != op.CLI.Service || cliCommand.Binding.Method != op.CLI.Method {
			t.Errorf("operation %q CLI binding = %s %s.%s, want connect-rpc %s.%s", operationID, cliCommand.Binding.Kind, cliCommand.Binding.Service, cliCommand.Binding.Method, op.CLI.Service, op.CLI.Method)
		}
	}
}

func resolveWirePath(t *testing.T, operationID string, message protoreflect.MessageDescriptor, path string) protoreflect.FieldDescriptor {
	t.Helper()
	parts := strings.Split(path, ".")
	for i, part := range parts {
		field := message.Fields().ByName(protoreflect.Name(part))
		if field == nil {
			t.Errorf("operation %q wire path %q: %s has no field %q", operationID, path, message.FullName(), part)
			return nil
		}
		if i == len(parts)-1 {
			return field
		}
		if field.Kind() != protoreflect.MessageKind || field.IsList() {
			t.Errorf("operation %q wire path %q: %s is not a singular message", operationID, path, field.FullName())
			return nil
		}
		message = field.Message()
	}
	return nil
}

func assertWirePort(t *testing.T, operationID, direction string, port operationcatalog.Port, field protoreflect.FieldDescriptor) {
	t.Helper()
	if field == nil {
		return
	}
	if port.Name != string(field.Name()) {
		t.Errorf("operation %q %s port name %q disagrees with wire field %q", operationID, direction, port.Name, field.FullName())
	}
	wantType := strings.TrimSuffix(port.Type, "[]")
	wantRepeated := strings.HasSuffix(port.Type, "[]")
	if wantRepeated != field.IsList() {
		t.Errorf("operation %q %s port %q type %q repeated=%t disagrees with wire field repeated=%t", operationID, direction, port.Name, port.Type, wantRepeated, field.IsList())
		return
	}
	if wantKind, ok := map[string]protoreflect.Kind{"bytes": protoreflect.BytesKind, "string": protoreflect.StringKind}[wantType]; ok {
		if field.Kind() != wantKind {
			t.Errorf("operation %q %s port %q type %q disagrees with wire kind %s", operationID, direction, port.Name, port.Type, field.Kind())
		}
		return
	}
	if field.Kind() != protoreflect.MessageKind || field.Message().Name() != protoreflect.Name(wantType) {
		got := ""
		if field.Kind() == protoreflect.MessageKind {
			got = string(field.Message().Name())
		}
		t.Errorf("operation %q %s port %q type %q disagrees with wire message %q", operationID, direction, port.Name, port.Type, got)
	}
}
