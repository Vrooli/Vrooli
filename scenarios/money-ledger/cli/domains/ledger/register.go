package ledger

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "ledger"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	g, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, ledgerBindings(h))
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("ledger: load from manifest: %w", err)
	}
	return g, nil
}

func RegisterIngest(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	g, err := cliapp.LoadFromManifestPrimitives(manifest, "ingest", ingestBindings(h))
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("ingest: load from manifest: %w", err)
	}
	return g, nil
}

func ledgerBindings(h *handlers) map[string]cliapp.PrimitiveHandler {
	return map[string]cliapp.PrimitiveHandler{
		"BooksService.ListBooks":        cliapp.ProtoList(h.booksList, booksListReport),
		"BooksService.CreateBook":       cliapp.ProtoMutation(h.booksCreate, booksCreateReport),
		"BooksService.ListAccounts":     cliapp.ProtoList(h.accountsList, accountsListReport),
		"BooksService.CreateAccount":    cliapp.ProtoMutation(h.accountsCreate, accountsCreateReport),
		"JournalService.ListPostings":   cliapp.ProtoList(h.postingsList, postingsListReport),
		"JournalService.GetPosting":     cliapp.ProtoList(h.postingGet, postingGetReport),
		"JournalService.ReversePosting": cliapp.ProtoMutation(h.reverse, reverseReport),
		"JournalService.Transfer":       cliapp.ProtoMutation(h.transfer, transferReport),
		"PositionService.GetPosition":   cliapp.ProtoList(h.position, positionReport),
		"PositionService.GetStatement":  cliapp.ProtoList(h.statement, statementReport),
		"PositionService.ListGoals":     cliapp.ProtoList(h.goals, goalsReport),
		"PositionService.DeclareGoal":   cliapp.ProtoMutation(h.goalDeclare, goalDeclareReport),
	}
}

func ingestBindings(h *handlers) map[string]cliapp.PrimitiveHandler {
	return map[string]cliapp.PrimitiveHandler{
		"IngestService.ListAdapters":         cliapp.ProtoList(h.adaptersList, adaptersReport),
		"IngestService.RegisterAdapter":      cliapp.ProtoMutation(h.adapterRegister, adapterRegisterReport),
		"IngestService.IngestEvent":          cliapp.ProtoMutation(h.ingestEvent, ingestReport),
		"IngestService.RunAdapter":           cliapp.ProtoList(h.adapterRun, runReport),
		"IngestService.ImportFile":           cliapp.ProtoMutation(h.fileImport, fileImportReport),
		"IngestService.ImportOperatorInputs": cliapp.ProtoMutation(h.operatorImport, operatorImportReport),
	}
}
