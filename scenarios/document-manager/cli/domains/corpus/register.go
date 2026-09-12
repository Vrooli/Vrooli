package corpus

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "corpus"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]cliapp.PrimitiveHandler{
		"CorpusService.CreateCollection": cliapp.ProtoMutation(h.createCall, h.createReport),
		"CorpusService.GetCollection":    cliapp.ProtoList(h.getCall, h.getReport),
		"CorpusService.ListCollections":  cliapp.ProtoList(h.listCall, h.listReport),
		"CorpusService.AddDocument":      cliapp.ProtoMutation(h.addDocumentCall, h.addDocumentReport),
		"CorpusService.ListDocuments":    cliapp.ProtoList(h.listDocumentsCall, h.listDocumentsReport),
		"CorpusService.Export":           cliapp.ProtoList(h.exportCall, h.exportReport),
		"CorpusService.Import":           cliapp.ProtoMutation(h.importCall, h.importReport),
		"CorpusService.Prune":            cliapp.ProtoMutation(h.pruneCall, h.pruneReport),
	}
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("corpus: load from manifest: %w", err)
	}
	return group, nil
}
