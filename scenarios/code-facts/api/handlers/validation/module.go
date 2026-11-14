package validation

import (
	"github.com/vrooli/api-core/module"
	"github.com/vrooli/api-core/validationstub"
)

func Module() module.Module { return validationstub.Module("code-facts") }
