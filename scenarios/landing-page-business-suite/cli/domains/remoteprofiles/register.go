package remoteprofiles

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"landing-page-business-suite/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

func Register(deps support.Dependencies) cliapp.CommandGroup {
	return cliapp.CommandGroup{
		Title: "Admin Remote Profiles",
		Commands: []cliapp.Command{
			{Name: "remote-profiles-list", NeedsAPI: true, Description: "List remote profiles (admin)", Run: func(args []string) error { return runList(deps, args) }},
			{Name: "remote-profiles-create", NeedsAPI: true, Description: "Create remote profile (admin)", Run: func(args []string) error { return RunCreate(deps, args) }},
			{Name: "remote-profiles-update", NeedsAPI: true, Description: "Update remote profile (admin)", Run: func(args []string) error { return runUpdate(deps, args) }},
			{Name: "remote-profiles-delete", NeedsAPI: true, Description: "Delete remote profile (admin)", Run: func(args []string) error { return runDelete(deps, args) }},
			{Name: "remote-profiles-login", NeedsAPI: true, Description: "Login remote profile (admin)", Run: func(args []string) error { return runLogin(deps, args) }},
			{Name: "remote-profiles-logout", NeedsAPI: true, Description: "Logout remote profile (admin)", Run: func(args []string) error { return runLogout(deps, args) }},
			{Name: "remote-profiles-test", NeedsAPI: true, Description: "Test remote profile session (admin)", Run: func(args []string) error { return runTest(deps, args) }},
			{Name: "remote-profiles-download-storage-test", NeedsAPI: true, Description: "Test remote download storage via profile session (admin)", Run: func(args []string) error { return runDownloadStorageTest(deps, args) }},
			{Name: "remote-profiles-download-apps-list", NeedsAPI: true, Description: "List remote download apps via profile session (admin)", Run: func(args []string) error { return runDownloadAppsList(deps, args) }},
			{Name: "remote-profiles-proxy", NeedsAPI: true, Description: "Proxy remote admin request via profile session (admin)", Run: func(args []string) error { return RunProxy(deps, args) }},
		},
	}
}

func runList(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("remote-profiles-list", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return fmt.Errorf("usage: remote-profiles-list [--json]")
	}
	resp, err := deps.RequestAdmin("GET", "/admin/remote-profiles", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func RunCreate(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("remote-profiles-create", flag.ContinueOnError)
	tag := fs.String("tag", "", "Profile tag (unique)")
	label := fs.String("label", "", "Profile label")
	apiBase := fs.String("api-base", "", "Remote API base (must end with /api/v1)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 0 {
		return fmt.Errorf("usage: remote-profiles-create --tag <tag> --api-base <url> [--label <label>] [--json]")
	}

	tagValue := strings.TrimSpace(*tag)
	apiBaseValue := strings.TrimSpace(*apiBase)
	if tagValue == "" || apiBaseValue == "" {
		return fmt.Errorf("usage: remote-profiles-create --tag <tag> --api-base <url> [--label <label>] [--json]")
	}
	normalizedAPIBase, err := support.ValidateRemoteProfileAPIBase(apiBaseValue)
	if err != nil {
		return err
	}

	payload := map[string]string{
		"tag":      tagValue,
		"api_base": normalizedAPIBase,
	}
	if strings.TrimSpace(*label) != "" {
		payload["label"] = strings.TrimSpace(*label)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	resp, err := deps.RequestAdmin("POST", "/admin/remote-profiles", nil, body)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func runUpdate(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("remote-profiles-update", flag.ContinueOnError)
	var tag support.OptionalString
	var label support.OptionalString
	var apiBase support.OptionalString
	fs.Var(&tag, "tag", "Updated tag")
	fs.Var(&label, "label", "Updated label (use empty string to clear)")
	fs.Var(&apiBase, "api-base", "Updated API base")
	jsonOut := cliutil.JSONFlag(fs)
	if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: remote-profiles-update <id> [--tag <tag>] [--label <label>] [--api-base <url>] [--json]")
	}
	profileID := strings.TrimSpace(fs.Args()[0])
	if profileID == "" {
		return fmt.Errorf("usage: remote-profiles-update <id> [--tag <tag>] [--label <label>] [--api-base <url>] [--json]")
	}

	payload := map[string]string{}
	if tag.IsSet {
		payload["tag"] = strings.TrimSpace(tag.Value)
	}
	if label.IsSet {
		payload["label"] = strings.TrimSpace(label.Value)
	}
	if apiBase.IsSet {
		payload["api_base"] = strings.TrimSpace(apiBase.Value)
	}
	if len(payload) == 0 {
		return fmt.Errorf("at least one of --tag, --label, or --api-base is required")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	resp, err := deps.RequestAdmin("PUT", "/admin/remote-profiles/"+url.PathEscape(profileID), nil, body)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func runDelete(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("remote-profiles-delete", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: remote-profiles-delete <id> [--json]")
	}
	profileID := strings.TrimSpace(fs.Args()[0])
	if profileID == "" {
		return fmt.Errorf("usage: remote-profiles-delete <id> [--json]")
	}

	resp, err := deps.RequestAdmin("DELETE", "/admin/remote-profiles/"+url.PathEscape(profileID), nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func runLogin(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("remote-profiles-login", flag.ContinueOnError)
	profileIDFlag := fs.String("profile-id", "", "Remote profile id")
	tag := fs.String("tag", "", "Remote profile tag (resolves id automatically)")
	email := fs.String("email", "", "Remote admin email")
	password := fs.String("password", "", "Remote admin password or @file")
	jsonOut := cliutil.JSONFlag(fs)
	if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 1 {
		return fmt.Errorf("usage: remote-profiles-login <id> --email <email> --password <password> [--json]")
	}
	positionalProfileID := ""
	if len(fs.Args()) == 1 {
		positionalProfileID = strings.TrimSpace(fs.Args()[0])
	}
	flagProfileID := strings.TrimSpace(*profileIDFlag)
	tagValue := strings.TrimSpace(*tag)
	if positionalProfileID != "" && flagProfileID != "" {
		return fmt.Errorf("use either positional <id> or --profile-id, not both")
	}
	if tagValue != "" && (positionalProfileID != "" || flagProfileID != "") {
		return fmt.Errorf("use either --tag or an explicit profile id (--profile-id or positional <id>)")
	}

	profileID := positionalProfileID
	if profileID == "" {
		profileID = flagProfileID
	}
	if profileID == "" && tagValue != "" {
		resolvedProfileID, err := deps.ResolveRemoteProfileIDByTag(tagValue)
		if err != nil {
			return err
		}
		profileID = resolvedProfileID
	}
	if profileID == "" {
		return fmt.Errorf("usage: remote-profiles-login <id> --email <email> --password <password> [--json]")
	}
	emailValue := strings.TrimSpace(*email)
	if emailValue == "" {
		return fmt.Errorf("usage: remote-profiles-login <id> --email <email> --password <password> [--json]")
	}
	passwordValue, err := support.ResolveSecretArg(*password)
	if err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	if strings.TrimSpace(passwordValue) == "" {
		return fmt.Errorf("usage: remote-profiles-login <id> --email <email> --password <password> [--json]")
	}

	body, err := json.Marshal(map[string]string{"email": emailValue, "password": passwordValue})
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	resp, err := deps.RequestAdmin("POST", "/admin/remote-profiles/"+url.PathEscape(profileID)+"/login", nil, body)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func runLogout(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("remote-profiles-logout", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 1 {
		return fmt.Errorf("usage: remote-profiles-logout <id> [--json]")
	}
	profileID := strings.TrimSpace(fs.Args()[0])
	if profileID == "" {
		return fmt.Errorf("usage: remote-profiles-logout <id> [--json]")
	}

	resp, err := deps.RequestAdmin("POST", "/admin/remote-profiles/"+url.PathEscape(profileID)+"/logout", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func runTest(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("remote-profiles-test", flag.ContinueOnError)
	profileIDFlag := fs.String("profile-id", "", "Remote profile id")
	tag := fs.String("tag", "", "Remote profile tag (resolves id automatically)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 1 {
		return fmt.Errorf("usage: remote-profiles-test <id> [--json]")
	}
	positionalProfileID := ""
	if len(fs.Args()) == 1 {
		positionalProfileID = strings.TrimSpace(fs.Args()[0])
	}
	flagProfileID := strings.TrimSpace(*profileIDFlag)
	tagValue := strings.TrimSpace(*tag)
	if positionalProfileID != "" && flagProfileID != "" {
		return fmt.Errorf("use either positional <id> or --profile-id, not both")
	}
	if tagValue != "" && (positionalProfileID != "" || flagProfileID != "") {
		return fmt.Errorf("use either --tag or an explicit profile id (--profile-id or positional <id>)")
	}

	profileID := positionalProfileID
	if profileID == "" {
		profileID = flagProfileID
	}
	if profileID == "" && tagValue != "" {
		resolvedProfileID, err := deps.ResolveRemoteProfileIDByTag(tagValue)
		if err != nil {
			return err
		}
		profileID = resolvedProfileID
	}
	if profileID == "" {
		return fmt.Errorf("usage: remote-profiles-test <id> [--json]")
	}

	resp, err := deps.RequestAdmin("POST", "/admin/remote-profiles/"+url.PathEscape(profileID)+"/test", nil, nil)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func RunProxy(deps support.Dependencies, args []string) error {
	fs := flag.NewFlagSet("remote-profiles-proxy", flag.ContinueOnError)
	method := fs.String("method", "", "HTTP method (GET, POST, PUT, PATCH, DELETE)")
	pathValue := fs.String("path", "", "Admin path (e.g., /admin/download-artifacts)")
	var queries cliutil.StringList
	var headers cliutil.StringList
	body := fs.String("body", "", "JSON body payload or @file.json")
	profileIDFlag := fs.String("profile-id", "", "Remote profile id (alternative to positional <id>)")
	profileTag := fs.String("profile-tag", "", "Remote profile tag (resolves id automatically)")
	tagAlias := fs.String("tag", "", "Alias for --profile-tag")
	fs.Var(&queries, "query", "Query parameters (key=value or key=value&key2=value2). Repeatable.")
	fs.Var(&headers, "header", "Header override (key=value or key:value). Repeatable.")
	jsonOut := cliutil.JSONFlag(fs)
	if err := support.ParseFlagSetInterspersed(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) > 1 {
		return fmt.Errorf("usage: remote-profiles-proxy <id> --method <METHOD> --path <path> [--query k=v] [--header k=v] [--body @file.json] [--json]")
	}

	positionalProfileID := ""
	if len(fs.Args()) == 1 {
		positionalProfileID = strings.TrimSpace(fs.Args()[0])
	}
	flagProfileID := strings.TrimSpace(*profileIDFlag)
	tagValue := strings.TrimSpace(*profileTag)
	tagAliasValue := strings.TrimSpace(*tagAlias)
	if tagValue == "" && tagAliasValue != "" {
		tagValue = tagAliasValue
	}
	if tagValue != "" && tagAliasValue != "" && tagValue != tagAliasValue {
		return fmt.Errorf("use only one of --profile-tag or --tag (values differ)")
	}
	if positionalProfileID != "" && flagProfileID != "" {
		return fmt.Errorf("use either positional <id> or --profile-id, not both")
	}
	if tagValue != "" && (positionalProfileID != "" || flagProfileID != "") {
		return fmt.Errorf("use either --profile-tag/--tag or an explicit profile id (--profile-id or positional <id>)")
	}

	profileID := positionalProfileID
	if profileID == "" {
		profileID = flagProfileID
	}
	if profileID == "" && tagValue != "" {
		resolvedProfileID, err := deps.ResolveRemoteProfileIDByTag(tagValue)
		if err != nil {
			return err
		}
		profileID = resolvedProfileID
	}
	if profileID == "" {
		return fmt.Errorf("usage: remote-profiles-proxy <id> --method <METHOD> --path <path> [--query k=v] [--header k=v] [--body @file.json] [--json]")
	}
	methodValue := strings.ToUpper(strings.TrimSpace(*method))
	if methodValue == "" {
		return fmt.Errorf("method is required (use --method)")
	}
	pathValueTrimmed := strings.TrimSpace(*pathValue)
	if pathValueTrimmed == "" {
		return fmt.Errorf("path is required (use --path)")
	}
	if !strings.HasPrefix(pathValueTrimmed, "/admin") {
		return fmt.Errorf("path must start with /admin")
	}

	payloadBody, err := support.ParseBody(*body)
	if err != nil {
		return err
	}
	queryValues, err := support.ParseQueries(queries.Values())
	if err != nil {
		return err
	}
	headerValues, err := support.ParseKeyValuePairs(headers.Values())
	if err != nil {
		return err
	}

	resp, err := deps.RequestRemoteProxy(profileID, methodValue, pathValueTrimmed, queryValues, headerValues, payloadBody)
	if err != nil {
		return err
	}
	if *jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func runDownloadStorageTest(deps support.Dependencies, args []string) error {
	profileID, jsonOut, err := deps.ParseRemoteProfileSelector("remote-profiles-download-storage-test", args)
	if err != nil {
		return err
	}
	resp, err := deps.RequestRemoteProxy(profileID, http.MethodPost, "/admin/download-storage/test", nil, nil, nil)
	if err != nil {
		return err
	}
	if jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}

func runDownloadAppsList(deps support.Dependencies, args []string) error {
	profileID, jsonOut, err := deps.ParseRemoteProfileSelector("remote-profiles-download-apps-list", args)
	if err != nil {
		return err
	}
	resp, err := deps.RequestRemoteProxy(profileID, http.MethodGet, "/admin/download-apps", nil, nil, nil)
	if err != nil {
		return err
	}
	if jsonOut {
		cliutil.PrintJSON(resp)
		return nil
	}
	cliutil.PrintJSON(resp)
	return nil
}
