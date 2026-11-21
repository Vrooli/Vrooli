package gatewaycatalog

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"ai-gateway/internal/providers"
	"ai-gateway/internal/rolecatalog"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// LoadRoles reads and validates the inference role catalog. The canonical
// role-catalog type and validation live in internal/rolecatalog; this package
// only projects the dispatchable roles into operator documentation, so it must
// not keep a second schema that can drift from the runtime loader.
func LoadRoles(path string) (*rolecatalog.RoleCatalog, error) {
	catalog, err := rolecatalog.LoadCatalog(path)
	if err != nil {
		return nil, err
	}
	if len(catalog.Roles) == 0 {
		return nil, fmt.Errorf("role catalog %s declares no roles", path)
	}
	return &catalog, nil
}

// sortedRoleNames returns the provider-role names in stable alphabetical order.
// Code roles are not provider candidates and are omitted from this projection.
func sortedRoleNames(c *rolecatalog.RoleCatalog) []string {
	names := make([]string, 0, len(c.Roles))
	for name := range c.Roles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RenderReference renders docs/reference/capability-catalog.md.
func RenderReference(c *ProfileCatalog, roles *rolecatalog.RoleCatalog, specs []providers.ProviderSpec) ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	if roles == nil || len(roles.Roles) == 0 {
		return nil, fmt.Errorf("role catalog is empty")
	}
	if len(specs) == 0 {
		return nil, fmt.Errorf("provider specs are empty")
	}

	var out bytes.Buffer
	writeReferenceHeader(&out)
	writeProfileTable(&out, c)
	writePrivacyClassTable(&out)
	writeRequestKindTable(&out)
	writeRoleTable(&out, roles)
	if err := writeProviderTable(&out, specs); err != nil {
		return nil, err
	}
	writeCandidateMatrix(&out, roles)
	writeEscalationCandidates(&out, roles)
	writeQualificationCandidates(&out, roles)
	return out.Bytes(), nil
}

func codeList(values []string) string {
	items := make([]string, len(values))
	for i, value := range values {
		items[i] = "`" + mdCell(value) + "`"
	}
	return strings.Join(items, ", ")
}

func modalityNames(values []sharedv1.Modality) []string {
	names := make([]string, len(values))
	for i, value := range values {
		name := strings.ToLower(value.String())
		names[i] = strings.TrimPrefix(name, "modality_")
	}
	return names
}

type privacyClassEntry struct {
	value string
	proto string
}

func privacyClassCatalog() []privacyClassEntry {
	values := make([]int, 0, len(sharedv1.PrivacyClass_name))
	for value := range sharedv1.PrivacyClass_name {
		if sharedv1.PrivacyClass(value) != sharedv1.PrivacyClass_PRIVACY_CLASS_UNSPECIFIED {
			values = append(values, int(value))
		}
	}
	sort.Ints(values)
	entries := make([]privacyClassEntry, 0, len(values))
	for _, value := range values {
		proto := sharedv1.PrivacyClass_name[int32(value)]
		entries = append(entries, privacyClassEntry{
			value: strings.ToLower(strings.TrimPrefix(proto, "PRIVACY_CLASS_")),
			proto: proto,
		})
	}
	return entries
}

func mdCell(value string) string {
	value = strings.ReplaceAll(value, "|", `\|`)
	value = strings.ReplaceAll(value, "\r\n", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}
