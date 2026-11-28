package agentaccess

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const audioQualificationFixtureRef = "audio-tools:dictation-reference"

// AudioFixtureResolver maps the one named Audio Tools qualification fixture
// to an owner-local path. Agent Access never accepts a caller-supplied path.
type AudioFixtureResolver interface {
	ResolveAudioFixture(context.Context, string) (string, error)
}

type vrooliAudioFixtureResolver struct {
	root string
}

// NewVrooliAudioFixtureResolver resolves only the checked-in Audio Tools WAV.
// The root is deployment configuration, not caller input.
func NewVrooliAudioFixtureResolver(root string) AudioFixtureResolver {
	return vrooliAudioFixtureResolver{root: strings.TrimSpace(root)}
}

func (r vrooliAudioFixtureResolver) ResolveAudioFixture(_ context.Context, ref string) (string, error) {
	if ref != audioQualificationFixtureRef {
		return "", fmt.Errorf("unsupported audio fixture reference %q", ref)
	}
	if r.root == "" || !filepath.IsAbs(r.root) {
		return "", errors.New("Vrooli root is unavailable for the Audio Tools fixture")
	}
	path := filepath.Join(r.root, "scenarios", "audio-tools", "bas", "fixtures", "dictation-reference.wav")
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("owner-approved Audio Tools fixture is unavailable")
	}
	return path, nil
}

func (b *Broker) SetAudioFixtureResolver(resolver AudioFixtureResolver) {
	if b == nil {
		return
	}
	b.mu.Lock()
	b.audioFixtureResolver = resolver
	b.mu.Unlock()
}
