package protosurface

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

type Loader interface {
	LoadScenario(scenario string) (Surface, error)
}

type DescriptorLoader struct {
	files    *protoregistry.Files
	repoRoot string
}

func NewDescriptorLoaderFromFile(repoRoot, descriptorPath string) (*DescriptorLoader, error) {
	b, err := os.ReadFile(descriptorPath)
	if err != nil {
		return nil, fmt.Errorf("read descriptor image %q: %w", descriptorPath, err)
	}
	return NewDescriptorLoaderFromBytes(repoRoot, b)
}

func NewDescriptorLoaderFromBytes(repoRoot string, b []byte) (*DescriptorLoader, error) {
	fdset := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(b, fdset); err != nil {
		return nil, fmt.Errorf("unmarshal descriptor image: %w", err)
	}
	files, err := protodesc.NewFiles(fdset)
	if err != nil {
		return nil, fmt.Errorf("build descriptor registry: %w", err)
	}
	return &DescriptorLoader{files: files, repoRoot: repoRoot}, nil
}

func (l *DescriptorLoader) LoadScenario(scenario string) (Surface, error) {
	scenario = strings.TrimSpace(scenario)
	if scenario == "" {
		return Surface{}, fmt.Errorf("scenario name is required")
	}
	prefix := scenario + "/"
	s := Surface{Scenario: scenario, TransportWorld: TransportWorldNone}
	l.files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		path := fd.Path()
		if !strings.HasPrefix(path, prefix) {
			return true
		}
		meta := fileMeta(path)
		annotations := annotationsForFile(fd)
		s.Files = append(s.Files, File{
			Path:        path,
			Package:     string(fd.Package()),
			Version:     meta.version,
			Domain:      meta.domain,
			Stability:   annotationValue(annotations, "stability"),
			Annotations: annotations,
		})
		s.Services = append(s.Services, servicesForFile(fd, meta.domain)...)
		s.Messages = append(s.Messages, messagesForFile(fd, meta.domain)...)
		intra, cross := importsForFile(scenario, fd, meta.domain)
		s.IntraScenarioImports = append(s.IntraScenarioImports, intra...)
		s.CrossScenarioImports = append(s.CrossScenarioImports, cross...)
		return true
	})
	sortSurface(&s)
	if len(s.Files) == 0 {
		return s, fmt.Errorf("no proto files found for scenario %q", scenario)
	}
	l.applyTransportFacts(&s)
	return s, nil
}

type meta struct {
	scenario string
	version  string
	domain   string
}

func fileMeta(path string) meta {
	parts := strings.Split(path, "/")
	m := meta{}
	if len(parts) >= 1 {
		m.scenario = parts[0]
	}
	if len(parts) >= 2 {
		m.version = parts[1]
	}
	if len(parts) == 3 && strings.HasSuffix(parts[2], ".proto") {
		m.domain = strings.TrimSuffix(parts[2], ".proto")
	} else if len(parts) >= 3 {
		m.domain = parts[2]
	}
	return m
}

func annotationsForFile(fd protoreflect.FileDescriptor) []Annotation {
	var out []Annotation
	srcLocs := fd.SourceLocations()
	appendLocationAnnotations(&out, srcLocs.ByDescriptor(fd))
	for i := 0; i < fd.Services().Len(); i++ {
		appendLocationAnnotations(&out, srcLocs.ByDescriptor(fd.Services().Get(i)))
	}
	for i := 0; i < fd.Messages().Len(); i++ {
		appendLocationAnnotations(&out, srcLocs.ByDescriptor(fd.Messages().Get(i)))
	}
	for i := 0; i < fd.Enums().Len(); i++ {
		appendLocationAnnotations(&out, srcLocs.ByDescriptor(fd.Enums().Get(i)))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name != out[j].Name {
			return out[i].Name < out[j].Name
		}
		return out[i].Value < out[j].Value
	})
	return out
}

func appendLocationAnnotations(out *[]Annotation, loc protoreflect.SourceLocation) {
	for _, comments := range loc.LeadingDetachedComments {
		appendAnnotations(out, comments)
	}
	appendAnnotations(out, loc.LeadingComments)
}

func appendAnnotations(out *[]Annotation, comments string) {
	for _, line := range strings.Split(comments, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimSpace(strings.TrimPrefix(line, "//"))
		if !strings.HasPrefix(line, "@") {
			continue
		}
		line = strings.TrimPrefix(line, "@")
		name, value, _ := strings.Cut(line, " ")
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		*out = append(*out, Annotation{Name: name, Value: strings.TrimSpace(value)})
	}
}

func annotationValue(in []Annotation, name string) string {
	for _, a := range in {
		if a.Name == name {
			return a.Value
		}
	}
	return ""
}

func servicesForFile(fd protoreflect.FileDescriptor, domain string) []Service {
	services := fd.Services()
	out := make([]Service, 0, services.Len())
	for i := 0; i < services.Len(); i++ {
		svc := services.Get(i)
		methods := svc.Methods()
		rpcs := make([]RPC, 0, methods.Len())
		for j := 0; j < methods.Len(); j++ {
			m := methods.Get(j)
			rpcs = append(rpcs, RPC{
				Name:      string(m.Name()),
				Input:     string(m.Input().FullName()),
				Output:    string(m.Output().FullName()),
				Transport: TransportKindNotServed,
			})
		}
		out = append(out, Service{
			FilePath: fd.Path(),
			Package:  string(fd.Package()),
			Name:     string(svc.Name()),
			FullName: string(svc.FullName()),
			Domain:   domain,
			RPCs:     rpcs,
		})
	}
	return out
}

func messagesForFile(fd protoreflect.FileDescriptor, domain string) []Message {
	var out []Message
	appendMessages(fd.Path(), string(fd.Package()), domain, fd.Messages(), &out)
	return out
}

func appendMessages(path, pkg, domain string, messages protoreflect.MessageDescriptors, out *[]Message) {
	for i := 0; i < messages.Len(); i++ {
		msg := messages.Get(i)
		fields := msg.Fields()
		m := Message{
			FilePath: path,
			Package:  pkg,
			Name:     string(msg.Name()),
			FullName: string(msg.FullName()),
			Domain:   domain,
			Fields:   make([]Field, 0, fields.Len()),
		}
		for j := 0; j < fields.Len(); j++ {
			fd := fields.Get(j)
			f := Field{
				Name:     string(fd.Name()),
				Type:     fieldType(fd),
				Repeated: fd.Cardinality() == protoreflect.Repeated,
				Optional: fd.HasOptionalKeyword(),
				Number:   int32(fd.Number()),
			}
			if fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind {
				f.MessageType = string(fd.Message().FullName())
			}
			if fd.Kind() == protoreflect.EnumKind {
				f.EnumType = string(fd.Enum().FullName())
			}
			m.Fields = append(m.Fields, f)
		}
		*out = append(*out, m)
		appendMessages(path, pkg, domain, msg.Messages(), out)
	}
}

func fieldType(fd protoreflect.FieldDescriptor) string {
	if fd.IsMap() {
		return "map"
	}
	return fd.Kind().String()
}

func importsForFile(scenario string, fd protoreflect.FileDescriptor, fromDomain string) ([]Import, []Import) {
	var intra []Import
	var cross []Import
	imports := fd.Imports()
	fromMeta := fileMeta(fd.Path())
	for i := 0; i < imports.Len(); i++ {
		imp := imports.Get(i)
		toPath := imp.Path()
		if isExternalWellKnown(toPath) {
			continue
		}
		toMeta := fileMeta(toPath)
		entry := Import{
			FromFile:     fd.Path(),
			ToFile:       toPath,
			FromScenario: fromMeta.scenario,
			ToScenario:   toMeta.scenario,
			FromPackage:  string(fd.Package()),
			ToPackage:    string(imp.FileDescriptor.Package()),
			FromVersion:  fromMeta.version,
			ToVersion:    toMeta.version,
			FromDomain:   fromDomain,
			ToDomain:     toMeta.domain,
		}
		if strings.HasPrefix(toPath, scenario+"/") {
			entry.Kind = ImportKindScenarioLocal
			intra = append(intra, entry)
			continue
		}
		entry.Kind = ImportKindCrossScenario
		cross = append(cross, entry)
	}
	return intra, cross
}

func isExternalWellKnown(path string) bool {
	return strings.HasPrefix(path, "google/") ||
		strings.HasPrefix(path, "buf/") ||
		strings.HasPrefix(path, "validate/")
}

func (l *DescriptorLoader) applyTransportFacts(s *Surface) {
	if l.repoRoot == "" {
		return
	}
	facts, err := endpointFacts(filepath.Join(l.repoRoot, "scenarios", s.Scenario, ".vrooli", "endpoints.json"))
	if err != nil {
		return
	}
	messageIndex := messageIndex(s.Messages)
	connectCount := 0
	servedCount := 0
	for i := range s.Services {
		for j := range s.Services[i].RPCs {
			procedure := "/" + s.Services[i].FullName + "/" + s.Services[i].RPCs[j].Name
			if facts.connectPaths[procedure] {
				s.Services[i].RPCs[j].Transport = TransportKindConnect
				connectCount++
				servedCount++
				continue
			}
		}
	}
	switch {
	case servedCount == 0:
		s.TransportWorld = TransportWorldNone
	case connectCount == servedCount:
		s.TransportWorld = TransportWorldConnect
	default:
		s.TransportWorld = TransportWorldMixed
	}
	for _, endpoint := range facts.restExceptions {
		s.RESTExceptions = append(s.RESTExceptions, RESTExceptionEndpoint{
			EndpointID:             endpoint.ID,
			Path:                   endpoint.Path,
			Method:                 endpoint.Method,
			Domain:                 endpoint.Category,
			Reason:                 endpoint.RESTException.Reason,
			HasPayloadDeclarations: endpoint.RESTException.ProtoPayloads != nil,
		})
		appendRESTExceptionPayloads(s, endpoint, messageIndex)
	}
	sortRESTFacts(s)
}

func appendRESTExceptionPayloads(s *Surface, endpoint endpointFact, index map[string]Message) {
	if endpoint.RESTException.ProtoPayloads == nil {
		return
	}
	payloads := []struct {
		role RESTPayloadRole
		ref  endpointPayloadDeclaration
	}{
		{role: RESTPayloadRoleRequest, ref: endpoint.RESTException.ProtoPayloads.Request},
		{role: RESTPayloadRoleResponse, ref: endpoint.RESTException.ProtoPayloads.Response},
		{role: RESTPayloadRoleError, ref: endpoint.RESTException.ProtoPayloads.Error},
	}
	for _, payload := range payloads {
		s.RESTExceptionPayloads = append(s.RESTExceptionPayloads, RESTExceptionPayloadRef{
			EndpointID:    endpoint.ID,
			Path:          endpoint.Path,
			Method:        endpoint.Method,
			Domain:        endpoint.Category,
			Reason:        endpoint.RESTException.Reason,
			Role:          payload.role,
			ProtoFullName: strings.TrimSpace(payload.ref.ProtoFullName),
			Transport:     strings.TrimSpace(payload.ref.Transport),
			Conformance:   strings.TrimSpace(payload.ref.Conformance),
			ProofStatus:   RESTPayloadProofNotEvaluated,
		})
		if payload.ref.ProtoFullName == "" {
			continue
		}
		if msg, ok := index[payload.ref.ProtoFullName]; ok {
			appendRESTExceptionRef(s, endpoint, msg)
		}
	}
}

func appendRESTExceptionRef(s *Surface, endpoint endpointFact, msg Message) {
	for _, ref := range s.RESTExceptionRefs {
		if ref.Path == endpoint.Path && ref.FullName == msg.FullName {
			return
		}
	}
	s.RESTExceptionRefs = append(s.RESTExceptionRefs, RESTExceptionRef{
		EndpointID: endpoint.ID,
		Path:       endpoint.Path,
		Method:     endpoint.Method,
		Domain:     endpoint.Category,
		Message:    msg.Name,
		FullName:   msg.FullName,
	})
}

type endpointsFacts struct {
	connectPaths   map[string]bool
	restExceptions []endpointFact
}

type endpointFact struct {
	ID            string            `json:"id"`
	Path          string            `json:"path"`
	Method        string            `json:"method"`
	Category      string            `json:"category"`
	Response      endpointSchema    `json:"response"`
	RESTException restExceptionFact `json:"rest_exception"`
}

type endpointSchema struct {
	Type       string            `json:"type"`
	Properties map[string]string `json:"properties"`
}

type restExceptionFact struct {
	Reason        string                 `json:"reason"`
	Note          string                 `json:"note"`
	ProtoPayloads *endpointProtoPayloads `json:"proto_payloads"`
}

type endpointProtoPayloads struct {
	Request  endpointPayloadDeclaration `json:"request"`
	Response endpointPayloadDeclaration `json:"response"`
	Error    endpointPayloadDeclaration `json:"error"`
}

type endpointPayloadDeclaration struct {
	ProtoFullName string `json:"proto_full_name"`
	Transport     string `json:"transport"`
	Conformance   string `json:"conformance"`
}

func endpointFacts(path string) (endpointsFacts, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return endpointsFacts{}, err
	}
	var doc struct {
		Endpoints []endpointFact `json:"endpoints"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return endpointsFacts{}, err
	}
	out := endpointsFacts{connectPaths: make(map[string]bool, len(doc.Endpoints))}
	for _, e := range doc.Endpoints {
		if strings.HasPrefix(e.Path, "/") && e.RESTException.Reason == "" {
			out.connectPaths[e.Path] = true
			continue
		}
		if e.RESTException.Reason != "" {
			out.restExceptions = append(out.restExceptions, e)
		}
	}
	return out, nil
}

func messageIndex(messages []Message) map[string]Message {
	out := map[string]Message{}
	for _, m := range messages {
		out[m.FullName] = m
	}
	return out
}

func sortSurface(s *Surface) {
	sort.Slice(s.Files, func(i, j int) bool { return s.Files[i].Path < s.Files[j].Path })
	sort.Slice(s.Services, func(i, j int) bool { return s.Services[i].FullName < s.Services[j].FullName })
	sort.Slice(s.Messages, func(i, j int) bool { return s.Messages[i].FullName < s.Messages[j].FullName })
	sort.Slice(s.IntraScenarioImports, lessImport(s.IntraScenarioImports))
	sort.Slice(s.CrossScenarioImports, lessImport(s.CrossScenarioImports))
	sortRESTFacts(s)
}

func sortRESTFacts(s *Surface) {
	sort.Slice(s.RESTExceptions, func(i, j int) bool {
		if s.RESTExceptions[i].Path != s.RESTExceptions[j].Path {
			return s.RESTExceptions[i].Path < s.RESTExceptions[j].Path
		}
		return s.RESTExceptions[i].EndpointID < s.RESTExceptions[j].EndpointID
	})
	sort.Slice(s.RESTExceptionPayloads, func(i, j int) bool {
		if s.RESTExceptionPayloads[i].Path != s.RESTExceptionPayloads[j].Path {
			return s.RESTExceptionPayloads[i].Path < s.RESTExceptionPayloads[j].Path
		}
		if s.RESTExceptionPayloads[i].Role != s.RESTExceptionPayloads[j].Role {
			return s.RESTExceptionPayloads[i].Role < s.RESTExceptionPayloads[j].Role
		}
		return s.RESTExceptionPayloads[i].ProtoFullName < s.RESTExceptionPayloads[j].ProtoFullName
	})
	sort.Slice(s.RESTExceptionRefs, func(i, j int) bool {
		if s.RESTExceptionRefs[i].Path != s.RESTExceptionRefs[j].Path {
			return s.RESTExceptionRefs[i].Path < s.RESTExceptionRefs[j].Path
		}
		return s.RESTExceptionRefs[i].FullName < s.RESTExceptionRefs[j].FullName
	})
}

func lessImport(imports []Import) func(i, j int) bool {
	return func(i, j int) bool {
		if imports[i].FromFile != imports[j].FromFile {
			return imports[i].FromFile < imports[j].FromFile
		}
		return imports[i].ToFile < imports[j].ToFile
	}
}
