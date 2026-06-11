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
		meta := fileMeta(scenario, path)
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
	l.applyAdoptionSignals(&s)
	return s, nil
}

type meta struct {
	version string
	domain  string
}

func fileMeta(scenario, path string) meta {
	parts := strings.Split(path, "/")
	m := meta{}
	if len(parts) >= 2 && parts[0] == scenario {
		m.version = parts[1]
	}
	if len(parts) >= 3 && parts[0] == scenario {
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
	for i := 0; i < imports.Len(); i++ {
		imp := imports.Get(i)
		toPath := imp.Path()
		entry := Import{
			FromFile:   fd.Path(),
			ToFile:     toPath,
			FromDomain: fromDomain,
			ToDomain:   fileMeta(scenario, toPath).domain,
		}
		if strings.HasPrefix(toPath, scenario+"/") {
			intra = append(intra, entry)
			continue
		}
		if isExternalWellKnown(toPath) {
			continue
		}
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
	paths, err := endpointPaths(filepath.Join(l.repoRoot, "scenarios", s.Scenario, ".vrooli", "endpoints.json"))
	if err != nil {
		return
	}
	hasHandRolled := hasHandRolledTransport(filepath.Join(l.repoRoot, "scenarios", s.Scenario, "api"))
	connectCount := 0
	handRolledCount := 0
	servedCount := 0
	for i := range s.Services {
		for j := range s.Services[i].RPCs {
			procedure := "/" + s.Services[i].FullName + "/" + s.Services[i].RPCs[j].Name
			if paths[procedure] {
				s.Services[i].RPCs[j].Transport = TransportKindConnect
				connectCount++
				servedCount++
				continue
			}
			if hasHandRolled {
				s.Services[i].RPCs[j].Transport = TransportKindHandRolled
				handRolledCount++
				servedCount++
			}
		}
	}
	switch {
	case servedCount == 0:
		s.TransportWorld = TransportWorldNone
	case handRolledCount > 0 && connectCount == 0:
		s.TransportWorld = TransportWorldHandRolled
	case handRolledCount > 0 && connectCount > 0:
		s.TransportWorld = TransportWorldMixed
	case connectCount == servedCount:
		s.TransportWorld = TransportWorldConnect
	default:
		s.TransportWorld = TransportWorldMixed
	}
}

func hasHandRolledTransport(apiRoot string) bool {
	found := false
	_ = filepath.WalkDir(apiRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := string(b)
		if strings.Contains(text, "protojson.") ||
			strings.Contains(text, "RegisterRoutes") ||
			strings.Contains(text, ".HandleFunc(") && strings.Contains(text, "/api/") {
			found = true
		}
		return nil
	})
	return found
}

func endpointPaths(path string) (map[string]bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Endpoints []struct {
			Path          string          `json:"path"`
			RESTException json.RawMessage `json:"rest_exception"`
		} `json:"endpoints"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(doc.Endpoints))
	for _, e := range doc.Endpoints {
		if strings.HasPrefix(e.Path, "/") && len(e.RESTException) == 0 {
			out[e.Path] = true
		}
	}
	return out, nil
}

func (l *DescriptorLoader) applyAdoptionSignals(s *Surface) {
	if l.repoRoot == "" {
		return
	}
	genImport := "github.com/vrooli/vrooli/packages/proto/gen/go/" + s.Scenario
	apiRoot := filepath.Join(l.repoRoot, "scenarios", s.Scenario, "api")
	s.AdoptionSignals = append(s.AdoptionSignals, AdoptionSignal{
		Name:    "api_go_mod_replace",
		Present: fileContains(filepath.Join(apiRoot, "go.mod"), "github.com/vrooli/vrooli/packages/proto"),
		Detail:  "api/go.mod references the shared packages/proto module",
	})
	s.AdoptionSignals = append(s.AdoptionSignals, AdoptionSignal{
		Name:    "api_generated_go_import",
		Present: dirContains(apiRoot, genImport),
		Detail:  "api code imports this scenario's generated Go proto package",
	})
}

func fileContains(path, needle string) bool {
	b, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(b), needle)
}

func dirContains(root, needle string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found || d.IsDir() {
			if d != nil && d.IsDir() && (d.Name() == ".git" || d.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if fileContains(path, needle) {
			found = true
		}
		return nil
	})
	return found
}

func sortSurface(s *Surface) {
	sort.Slice(s.Files, func(i, j int) bool { return s.Files[i].Path < s.Files[j].Path })
	sort.Slice(s.Services, func(i, j int) bool { return s.Services[i].FullName < s.Services[j].FullName })
	sort.Slice(s.Messages, func(i, j int) bool { return s.Messages[i].FullName < s.Messages[j].FullName })
	sort.Slice(s.IntraScenarioImports, lessImport(s.IntraScenarioImports))
	sort.Slice(s.CrossScenarioImports, lessImport(s.CrossScenarioImports))
}

func lessImport(imports []Import) func(i, j int) bool {
	return func(i, j int) bool {
		if imports[i].FromFile != imports[j].FromFile {
			return imports[i].FromFile < imports[j].FromFile
		}
		return imports[i].ToFile < imports[j].ToFile
	}
}
