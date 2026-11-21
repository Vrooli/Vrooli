package preflight

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	configv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config"
	configconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config/config_v1connect"
	"scenario-to-cloud/cli/internal/manifest"
)

type mailDNSRequirement struct {
	Provider     string
	SPFMechanism string
	SPFHost      string
	DKIM         []mailDKIMRequirement
}

type mailDKIMRequirement struct {
	Selector string
	Type     string
	Expected string
}

type mailDNSObservation struct {
	Provider  string   `json:"provider"`
	Hostname  string   `json:"hostname"`
	Type      string   `json:"type"`
	Desired   string   `json:"desired"`
	Published []string `json:"published,omitempty"`
	Status    string   `json:"status"`
}

func readMailDNSManifest(path string) (string, []mailDNSRequirement, error) {
	raw, err := manifest.ReadJSONFile(path)
	if err != nil {
		return "", nil, err
	}
	edge, ok := raw["edge"].(map[string]interface{})
	if !ok {
		return "", nil, errors.New("manifest has no edge object")
	}
	domain, _ := edge["domain"].(string)
	domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
	if domain == "" {
		return "", nil, errors.New("manifest edge.domain is required")
	}
	items, _ := edge["mail_dns"].([]interface{})
	requirements := make([]mailDNSRequirement, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		req := mailDNSRequirement{Provider: stringValue(entry["name"]), SPFMechanism: stringValue(entry["spf_mechanism"]), SPFHost: stringValue(entry["spf_host"])}
		if rawDKIM, ok := entry["dkim"].([]interface{}); ok {
			for _, raw := range rawDKIM {
				value, _ := raw.(map[string]interface{})
				req.DKIM = append(req.DKIM, mailDKIMRequirement{Selector: stringValue(value["selector"]), Type: strings.ToUpper(stringValue(value["type"])), Expected: stringValue(value["expected"])})
			}
		}
		requirements = append(requirements, req)
	}
	sort.SliceStable(requirements, func(i, j int) bool { return requirements[i].Provider < requirements[j].Provider })
	return domain, requirements, nil
}

func stringValue(value interface{}) string {
	text, _ := value.(string)
	return strings.TrimSpace(text)
}

func observeMailDNS(ctx context.Context, domain string, requirements []mailDNSRequirement) []mailDNSObservation {
	observations := make([]mailDNSObservation, 0)
	for _, req := range requirements {
		if req.SPFMechanism != "" {
			published, _ := net.LookupTXT(domain)
			status := "missing"
			for _, value := range published {
				if strings.Contains(strings.ToLower(value), strings.ToLower(req.SPFMechanism)) {
					status = "present"
				}
			}
			observations = append(observations, mailDNSObservation{Provider: req.Provider, Hostname: domain, Type: "SPF", Desired: "merge " + req.SPFMechanism, Published: published, Status: status})
		}
		for _, dkim := range req.DKIM {
			host := dkim.Selector + "._domainkey." + domain
			var published []string
			if dkim.Type == "CNAME" {
				if value, err := net.LookupCNAME(host); err == nil {
					published = []string{strings.TrimSuffix(value, ".")}
				}
			} else {
				published, _ = net.LookupTXT(host)
			}
			status := "missing"
			for _, value := range published {
				if dkim.Expected == "" || strings.Contains(strings.ToLower(value), strings.ToLower(dkim.Expected)) {
					status = "present"
				}
			}
			observations = append(observations, mailDNSObservation{Provider: req.Provider, Hostname: host, Type: dkim.Type, Desired: dkim.Expected, Published: published, Status: status})
		}
	}
	return observations
}

func runMailDNSDiff(args []string) error {
	fs := flag.NewFlagSet("mail-dns-diff", flag.ContinueOnError)
	jsonOutput := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: scenario-to-cloud preflight mail-dns-diff <manifest.json>")
	}
	path := fs.Arg(0)
	domain, requirements, err := readMailDNSManifest(path)
	if err != nil {
		return err
	}
	observations := observeMailDNS(context.Background(), domain, requirements)
	if *jsonOutput {
		body, _ := json.MarshalIndent(map[string]interface{}{"domain": domain, "observations": observations}, "", "  ")
		fmt.Println(string(body))
		return nil
	}
	printMailDNSObservations(domain, observations)
	return nil
}

func runMailDNSApply(args []string) error {
	profile := "cloudflare-default"
	owner := "scenario-to-cloud:mail-dns"
	confirm := false
	jsonOutput := false
	fs := flag.NewFlagSet("mail-dns-apply", flag.ContinueOnError)
	fs.StringVar(&profile, "provider-profile", profile, "tunnel-manager managed DNS provider profile")
	fs.StringVar(&owner, "owner", owner, "stable DNS ownership tag")
	fs.BoolVar(&confirm, "confirm", false, "explicitly authorize live DNS writes")
	fs.BoolVar(&jsonOutput, "json", jsonOutput, "output JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !confirm {
		return errors.New("mail-dns-apply refuses without --confirm; run mail-dns-diff first")
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: scenario-to-cloud preflight mail-dns-apply <manifest.json> --confirm")
	}
	path := fs.Arg(0)
	domain, requirements, err := readMailDNSManifest(path)
	if err != nil {
		return err
	}
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("VROOLI_TUNNEL_MANAGER_URL")), "/")
	if baseURL == "" {
		return errors.New("VROOLI_TUNNEL_MANAGER_URL is required for mail DNS apply")
	}
	client := configconnect.NewConfigServiceClient(httpClient(), baseURL)
	results := make([]map[string]interface{}, 0)
	for _, req := range requirements {
		if req.SPFMechanism != "" {
			resp, callErr := client.EnsureSPFRecord(context.Background(), connect.NewRequest(&configv1.EnsureSPFRecordRequest{ProviderProfile: profile, Hostname: domain, Mechanism: req.SPFMechanism, Owner: owner, Ttl: 300}))
			if callErr != nil {
				return callErr
			}
			results = append(results, map[string]interface{}{"provider": req.Provider, "type": "SPF", "hostname": domain, "merged": resp.Msg.MergedValue, "changed": resp.Msg.Changed})
		}
		for _, dkim := range req.DKIM {
			if dkim.Expected == "" {
				continue
			}
			host := dkim.Selector + "._domainkey." + domain
			resp, callErr := client.UpdateDNSRecord(context.Background(), connect.NewRequest(&configv1.UpdateDNSRecordRequest{ProviderProfile: profile, Hostname: host, Type: dkim.Type, Content: dkim.Expected, Owner: owner, Ttl: 300}))
			if callErr != nil {
				return callErr
			}
			results = append(results, map[string]interface{}{"provider": req.Provider, "type": dkim.Type, "hostname": host, "record_id": resp.Msg.RecordId, "changed": resp.Msg.Changed})
		}
	}
	if jsonOutput {
		body, _ := json.MarshalIndent(map[string]interface{}{"domain": domain, "applied": results}, "", "  ")
		fmt.Println(string(body))
	} else {
		fmt.Printf("Applied reviewed mail DNS changes for %s (%d operations).\n", domain, len(results))
	}
	return nil
}

func printMailDNSObservations(domain string, observations []mailDNSObservation) {
	fmt.Printf("Mail DNS review for %s\n", domain)
	for _, observation := range observations {
		published := "(none)"
		if len(observation.Published) > 0 {
			published = strings.Join(observation.Published, " | ")
		}
		fmt.Printf("  %-6s %-40s %-8s desired=%s published=%s\n", observation.Type, observation.Hostname, observation.Status, observation.Desired, published)
	}
}

func httpClient() *http.Client { return &http.Client{Timeout: 30 * time.Second} }
