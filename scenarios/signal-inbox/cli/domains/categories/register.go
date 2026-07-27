package categories

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "categories"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"CategoriesService.CreateCategory":        cliapp.ProtoMutation(h.createCall, h.createReport),
		"CategoriesService.ListCategories":        cliapp.ProtoList(h.listCall, h.listReport),
		"CategoriesService.RenameCategory":        cliapp.ProtoMutation(h.renameCall, h.renameReport),
		"CategoriesService.RetireCategory":        cliapp.ProtoMutation(h.retireCall, h.retireReport),
		"CategoriesService.GetClassification":     cliapp.ProtoList(h.getClassificationCall, h.getClassificationReport),
		"CategoriesService.ConfirmClassification": cliapp.ProtoMutation(h.confirmCall, h.confirmReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("categories: load manifest: %w", err)
	}
	return group, nil
}
