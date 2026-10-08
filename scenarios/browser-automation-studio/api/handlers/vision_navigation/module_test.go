package vision_navigation

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/browser-automation-studio/services/vision"
)

func TestModule_RequiresLogger(t *testing.T) {
	require.Panics(t, func() {
		Module(Deps{Registry: vision.NewNavigatorRegistry()})
	})
}

func TestModule_RequiresRegistry(t *testing.T) {
	require.Panics(t, func() {
		Module(Deps{Logger: logrus.New()})
	})
}

func TestModule_FullDepsReturnsMount(t *testing.T) {
	mount := Module(Deps{
		Logger:   logrus.New(),
		Registry: vision.NewNavigatorRegistry(),
	})
	require.NotEmpty(t, mount.Path)
	require.NotNil(t, mount.Handler)
}
