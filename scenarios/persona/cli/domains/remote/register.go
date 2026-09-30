// Package remote exposes persona's Connect-RPC contracts through the typed
// cli-core dispatcher. Request construction remains descriptor-driven; this
// package only selects the safe, operator-facing methods for each group.
package remote

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/access"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/accounts"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/channels"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/documents"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/handoffs"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/journal"
	_ "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/personas"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type groupSpec struct {
	service protoreflect.FullName
	read    map[string]bool
}

var groups = map[string]groupSpec{
	"personas":  {service: "vrooli.persona.v1.personas.PersonasService", read: map[string]bool{"GetPersona": true, "ListPersonas": true, "CheckHealth": true}},
	"access":    {service: "vrooli.persona.v1.access.AccessService", read: map[string]bool{"ListGrants": true, "ResolvePersona": true}},
	"channels":  {service: "vrooli.persona.v1.channels.ChannelsService", read: map[string]bool{"ListChannels": true}},
	"handoffs":  {service: "vrooli.persona.v1.handoffs.HandoffsService", read: map[string]bool{"GetHandoff": true, "ListHandoffs": true}},
	"documents": {service: "vrooli.persona.v1.documents.DocumentsService", read: map[string]bool{"ListBindings": true}},
	"accounts":  {service: "vrooli.persona.v1.accounts.AccountsService", read: map[string]bool{"ListAccounts": true, "ListAddresses": true, "ListObligations": true}},
	"journal":   {service: "vrooli.persona.v1.journal.JournalService", read: map[string]bool{"List": true}},
}

func Register(core *cliapp.ScenarioApp, manifest []byte, groupName string) (cliapp.SubcommandGroup, error) {
	spec, ok := groups[groupName]
	if !ok {
		return cliapp.SubcommandGroup{}, fmt.Errorf("unknown persona CLI group %q", groupName)
	}
	bindings, err := cliapp.ProtoPrimitiveBindings(core, spec.service, cliapp.ProtoBindingOptions{}, spec.read)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: build primitive bindings: %w", groupName, err)
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, groupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: load proto group: %w", groupName, err)
	}
	return group, nil
}
