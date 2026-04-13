package contextinfo

import "os"

type File struct {
	Path     string `json:"path"`
	Contents string `json:"contents,omitempty"`
}

type Output struct {
	Root  string `json:"root"`
	Files []File `json:"files"`
}

type Service struct {
	CollectSources func(root string) ([]string, []string, error)
	ResolvePath    func(root, path string) string
	ReadFile       func(path string) ([]byte, error)
}

func (s Service) List(root string) ([]string, []string, error) {
	sources, warnings, err := s.CollectSources(root)
	if err != nil {
		return nil, nil, err
	}
	paths := make([]string, 0, len(sources))
	for _, source := range sources {
		paths = append(paths, s.ResolvePath(root, source))
	}
	return paths, warnings, nil
}

func (s Service) Load(root string) (Output, []string, error) {
	sources, warnings, err := s.CollectSources(root)
	if err != nil {
		return Output{}, nil, err
	}
	output := Output{Root: root, Files: make([]File, 0, len(sources))}
	for _, source := range sources {
		resolved := s.ResolvePath(root, source)
		contents, readErr := s.ReadFile(resolved)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				warnings = append(warnings, "Skipping missing context file: "+source)
				continue
			}
			return Output{}, warnings, readErr
		}
		output.Files = append(output.Files, File{Path: resolved, Contents: string(contents)})
	}
	return output, warnings, nil
}
