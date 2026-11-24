package config

import (
	"os"
	"testing"

	"github.com/vrooli/api-core/storage/storagetest"
)

func TestMain(m *testing.M) { os.Exit(storagetest.RunWithPrivateHome(m)) }
