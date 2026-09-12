package main

import (
	"encoding/json"
	"os"
	"testing"

	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type attributionManifest struct {
	Groups []struct {
		Commands []struct {
			Name  string `json:"name"`
			Flags []struct {
				Name string `json:"name"`
			} `json:"flags"`
			Binding struct {
				Service string `json:"service"`
				Method  string `json:"method"`
			} `json:"binding"`
		} `json:"commands"`
	} `json:"groups"`
}

func TestEveryAuditedWriteExposesActorAndReason(t *testing.T) {
	data, err := os.ReadFile("manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest attributionManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	services := map[string]protoreflect.ServiceDescriptor{
		"CatalogService": offerspb.File_offer_desk_v1_offers_offers_proto.Services().ByName("CatalogService"),
	}
	for _, group := range manifest.Groups {
		for _, command := range group.Commands {
			service := services[command.Binding.Service]
			if service == nil {
				continue
			}
			method := service.Methods().ByName(protoreflect.Name(command.Binding.Method))
			if method == nil {
				continue
			}
			fields := method.Input().Fields()
			actor, reason := fields.ByName("actor"), fields.ByName("reason")
			if actor == nil && reason == nil {
				continue
			}
			flags := map[string]bool{}
			for _, flag := range command.Flags {
				flags[flag.Name] = true
			}
			if actor != nil && !flags["actor"] {
				t.Errorf("%s binds %s but exposes no --actor flag", command.Name, command.Binding.Method)
			}
			if reason != nil && !flags["reason"] {
				t.Errorf("%s binds %s but exposes no --reason flag", command.Name, command.Binding.Method)
			}
		}
	}
}
