package planscli

import (
	"io"

	planapp "github.com/vrooli/vrooli/internal/app/plans"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

type CommandID string

const (
	CommandAdd     CommandID = "add"
	CommandList    CommandID = "list"
	CommandShow    CommandID = "show"
	CommandPath    CommandID = "path"
	CommandArchive CommandID = "archive"
	CommandImport  CommandID = "import"
	CommandExport  CommandID = "export"
)

type AddRequest struct {
	Title string
	Slug  string
	Repo  string
	Stdin bool
}

type ListRequest struct {
	Repo            string
	AllRepos        bool
	IncludeArchived bool
}

type RefRequest struct {
	Ref  string
	Repo string
}

type ImportRequest struct {
	Path         string
	Title        string
	Slug         string
	Repo         string
	DeleteSource bool
}

type ExportRequest struct {
	Ref  string
	Repo string
	To   string
}

func CommandSpecs() []commandtree.Spec[CommandID] {
	return []commandtree.Spec[CommandID]{
		{
			Name:    string(CommandAdd),
			Summary: "Add a scratch implementation plan to user plan storage",
			Group:   "Plan Lifecycle",
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{
					{Name: "--title", ValueName: "title", Description: "Readable plan title"},
					{Name: "--slug", ValueName: "slug", Description: "Optional readable filename slug"},
					{Name: "--repo", ValueName: "path", Description: "Associate the plan with a repository path"},
					{Name: "--stdin", Description: "Read markdown content from stdin"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandAdd,
		},
		{
			Name:    string(CommandList),
			Summary: "List scratch implementation plans",
			Group:   "Plan Lifecycle",
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{
					{Name: "--repo", ValueName: "path", Description: "List plans for a specific repository path"},
					{Name: "--all", Description: "List plans across all repositories"},
					{Name: "--archived", Description: "Include archived plans"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandList,
		},
		{
			Name:    string(CommandShow),
			Summary: "Show plan metadata and content",
			Group:   "Plan Lifecycle",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "id-or-slug", Required: true}},
				Options: []commandtree.OptionArg{
					{Name: "--repo", ValueName: "path", Description: "Repository association to search"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandShow,
		},
		{
			Name:    string(CommandPath),
			Summary: "Print the saved path for a plan",
			Group:   "Plan Lifecycle",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "id-or-slug", Required: true}},
				Options: []commandtree.OptionArg{
					{Name: "--repo", ValueName: "path", Description: "Repository association to search"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandPath,
		},
		{
			Name:    string(CommandArchive),
			Summary: "Archive a scratch implementation plan",
			Group:   "Plan Lifecycle",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "id-or-slug", Required: true}},
				Options: []commandtree.OptionArg{
					{Name: "--repo", ValueName: "path", Description: "Repository association to search"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandArchive,
		},
		{
			Name:    string(CommandImport),
			Summary: "Import an existing markdown plan into user plan storage",
			Group:   "Plan Lifecycle",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "path", Required: true}},
				Options: []commandtree.OptionArg{
					{Name: "--title", ValueName: "title", Description: "Override imported plan title"},
					{Name: "--slug", ValueName: "slug", Description: "Optional stable slug"},
					{Name: "--repo", ValueName: "path", Description: "Associate the plan with a repository path"},
					{Name: "--delete-source", Description: "Delete the original file after import"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandImport,
		},
		{
			Name:    string(CommandExport),
			Summary: "Export a scratch implementation plan into a repository path",
			Group:   "Plan Lifecycle",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "id-or-slug", Required: true}},
				Options: []commandtree.OptionArg{
					{Name: "--repo", ValueName: "path", Description: "Repository association to search"},
					{Name: "--to", ValueName: "path", Description: "Destination markdown path"},
					commandtree.JSONOption(),
				},
			},
			Handler: CommandExport,
		},
	}
}

func RenderCommandHelp(w io.Writer) {
	commandtree.RenderHelp(w, commandtree.Help{
		Title:        "Vrooli Plan Commands",
		Usage:        "vrooli plans <subcommand> [options]",
		DefaultGroup: "Plan Lifecycle",
	}, CommandSpecs())
}

func ParseAddRequest(args []string) (AddRequest, error) {
	parsed, err := commandtree.ParseArgs("plans add", commandHelpText(CommandAdd), commandSpec(CommandAdd).Args, args)
	if err != nil {
		return AddRequest{}, err
	}
	return AddRequest{
		Title: parsed.FlagValue("--title"),
		Slug:  parsed.FlagValue("--slug"),
		Repo:  parsed.FlagValue("--repo"),
		Stdin: parsed.HasFlag("--stdin"),
	}, nil
}

func ParseListRequest(args []string) (ListRequest, error) {
	parsed, err := commandtree.ParseArgs("plans list", commandHelpText(CommandList), commandSpec(CommandList).Args, args)
	if err != nil {
		return ListRequest{}, err
	}
	return ListRequest{
		Repo:            parsed.FlagValue("--repo"),
		AllRepos:        parsed.HasFlag("--all"),
		IncludeArchived: parsed.HasFlag("--archived"),
	}, nil
}

func ParseRefRequest(command string, id CommandID, args []string) (RefRequest, error) {
	parsed, err := commandtree.ParseArgs(command, commandHelpText(id), commandSpec(id).Args, args)
	if err != nil {
		return RefRequest{}, err
	}
	return RefRequest{Ref: parsed.Positionals[0], Repo: parsed.FlagValue("--repo")}, nil
}

func ParseImportRequest(args []string) (ImportRequest, error) {
	parsed, err := commandtree.ParseArgs("plans import", commandHelpText(CommandImport), commandSpec(CommandImport).Args, args)
	if err != nil {
		return ImportRequest{}, err
	}
	return ImportRequest{
		Path:         parsed.Positionals[0],
		Title:        parsed.FlagValue("--title"),
		Slug:         parsed.FlagValue("--slug"),
		Repo:         parsed.FlagValue("--repo"),
		DeleteSource: parsed.HasFlag("--delete-source"),
	}, nil
}

func ParseExportRequest(args []string) (ExportRequest, error) {
	parsed, err := commandtree.ParseArgs("plans export", commandHelpText(CommandExport), commandSpec(CommandExport).Args, args)
	if err != nil {
		return ExportRequest{}, err
	}
	return ExportRequest{
		Ref:  parsed.Positionals[0],
		Repo: parsed.FlagValue("--repo"),
		To:   parsed.FlagValue("--to"),
	}, nil
}

func ToAppAdd(req AddRequest, content string) planapp.AddRequest {
	return planapp.AddRequest{Title: req.Title, Slug: req.Slug, Repo: req.Repo, Content: content}
}

func commandSpec(id CommandID) commandtree.Spec[CommandID] {
	for _, spec := range CommandSpecs() {
		if spec.Handler == id {
			return spec
		}
	}
	panic("unknown plans command spec: " + string(id))
}

func commandHelpText(id CommandID) string {
	spec := commandSpec(id)
	return commandtree.SpecHelpText("", "vrooli plans "+spec.Name, spec)
}
