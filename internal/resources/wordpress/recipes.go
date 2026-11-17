package wordpress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const recipeSchemaVersion = 1

var recipeIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type Recipe struct {
	ID              string            `json:"id"`
	Version         int               `json:"version"`
	Kind            string            `json:"kind"`
	Title           string            `json:"title"`
	Plugins         []string          `json:"plugins"`
	PackageVersions map[string]string `json:"-"`
	Site            map[string]any    `json:"site"`
	Settings        map[string]any    `json:"settings,omitempty"`
	Content         map[string]any    `json:"content"`
	ExpectedLevels  []string          `json:"expectedLevels"`
}

type recipeFile struct {
	SchemaVersion int      `json:"schemaVersion"`
	Recipes       []Recipe `json:"recipes"`
}

type pluginLock struct {
	SchemaVersion int    `json:"schemaVersion"`
	Source        string `json:"source"`
	ObservedAt    string `json:"observedAt"`
	Packages      map[string]struct {
		Version      string  `json:"version"`
		LicenseClass string  `json:"licenseClass"`
		SourceURL    string  `json:"sourceUrl"`
		SHA256       *string `json:"sha256"`
		Staging      string  `json:"staging"`
	} `json:"packages"`
}

// LoadRecipe returns one validated, immutable recipe description and its exact
// plugin versions. The caller still needs approved staged archives to apply it.
func LoadRecipe(path, recipeID string) (Recipe, error) {
	if !recipeIDPattern.MatchString(recipeID) {
		return Recipe{}, fmt.Errorf("invalid WordPress recipe ID %q", recipeID)
	}
	file, err := os.Open(path)
	if err != nil {
		return Recipe{}, fmt.Errorf("open WordPress recipes: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 1<<20))
	decoder.DisallowUnknownFields()
	var catalog recipeFile
	if err := decoder.Decode(&catalog); err != nil {
		return Recipe{}, fmt.Errorf("decode WordPress recipes: %w", err)
	}
	if catalog.SchemaVersion != recipeSchemaVersion || len(catalog.Recipes) == 0 {
		return Recipe{}, fmt.Errorf("unsupported or empty WordPress recipe catalog")
	}
	lock, err := loadPluginLock(filepath.Join(filepath.Dir(path), "plugins.lock.json"))
	if err != nil {
		return Recipe{}, err
	}
	var selected *Recipe
	seen := map[string]bool{}
	for index := range catalog.Recipes {
		recipe := &catalog.Recipes[index]
		if !recipeIDPattern.MatchString(recipe.ID) || seen[recipe.ID] {
			return Recipe{}, fmt.Errorf("invalid or duplicate WordPress recipe ID %q", recipe.ID)
		}
		seen[recipe.ID] = true
		if recipe.ID == recipeID {
			selected = recipe
		}
	}
	if selected == nil {
		return Recipe{}, fmt.Errorf("unknown WordPress recipe %q", recipeID)
	}
	if err := validateRecipe(selected, lock); err != nil {
		return Recipe{}, err
	}
	return *selected, nil
}

func loadPluginLock(path string) (pluginLock, error) {
	file, err := os.Open(path)
	if err != nil {
		return pluginLock{}, fmt.Errorf("open WordPress plugin lock: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 256<<10))
	decoder.DisallowUnknownFields()
	var lock pluginLock
	if err := decoder.Decode(&lock); err != nil {
		return pluginLock{}, fmt.Errorf("decode WordPress plugin lock: %w", err)
	}
	if lock.SchemaVersion != 1 {
		return pluginLock{}, fmt.Errorf("unsupported WordPress plugin lock schema %d", lock.SchemaVersion)
	}
	if len(lock.Packages) == 0 {
		return pluginLock{}, fmt.Errorf("WordPress plugin lock is empty")
	}
	return lock, nil
}

func validateRecipe(recipe *Recipe, lock pluginLock) error {
	if recipe.Version < 1 || strings.TrimSpace(recipe.Title) == "" || len(recipe.Site) == 0 || len(recipe.Content) == 0 || len(recipe.ExpectedLevels) == 0 {
		return fmt.Errorf("WordPress recipe %q is incomplete", recipe.ID)
	}
	if recipe.Kind != "forms" && recipe.Kind != "shop" && recipe.Kind != "content" {
		return fmt.Errorf("WordPress recipe %q has unsupported kind %q", recipe.ID, recipe.Kind)
	}
	if len(recipe.Plugins) == 0 {
		provider, _ := recipe.Settings["fieldProvider"].(string)
		if recipe.Kind != "content" || provider != "wordpress-core-post-meta" {
			return fmt.Errorf("WordPress recipe %q has no plugins and does not declare the supported WordPress core content provider", recipe.ID)
		}
	}
	versions := make(map[string]string, len(recipe.Plugins))
	for _, slug := range recipe.Plugins {
		pkg, ok := lock.Packages[slug]
		if !ok || !recipeIDPattern.MatchString(slug) {
			return fmt.Errorf("WordPress recipe %q references an unapproved plugin %q", recipe.ID, slug)
		}
		if pkg.Version == "" || !strings.HasPrefix(pkg.SourceURL, "https://downloads.wordpress.org/plugin/") || pkg.LicenseClass == "" || pkg.SHA256 != nil || !strings.Contains(pkg.Staging, "blocked-until-scenario-dependency-analyzer-wordpress-support") {
			return fmt.Errorf("WordPress plugin lock entry %q lacks the required official pinned release and approved staging state", slug)
		}
		versions[slug] = pkg.Version
	}
	levels := map[string]bool{}
	for _, level := range recipe.ExpectedLevels {
		if strings.TrimSpace(level) == "" || levels[level] {
			return fmt.Errorf("WordPress recipe %q has an empty or duplicate expected level", recipe.ID)
		}
		levels[level] = true
	}
	if recipe.Kind == "forms" && (len(recipe.Plugins) != 1 || len(levels) < 5) {
		return fmt.Errorf("form recipe %q requires one form plugin and at least five graded levels", recipe.ID)
	}
	if recipe.Kind == "shop" {
		if len(recipe.Plugins) != 1 || recipe.Plugins[0] != "woocommerce" {
			return fmt.Errorf("shop recipe must use WooCommerce only")
		}
		payments, ok := recipe.Settings["paymentsEnabled"].(bool)
		if !ok || payments {
			return fmt.Errorf("shop recipe must explicitly disable payments")
		}
		products, orders := nestedSlice(recipe.Content, "products"), nestedSlice(recipe.Content, "orders")
		if len(products) < 3 || len(orders) < 3 {
			return fmt.Errorf("shop recipe requires at least three synthetic products and orders")
		}
		for _, item := range orders {
			order, ok := item.(map[string]any)
			if !ok || !strings.HasSuffix(fmt.Sprint(order["customerEmail"]), ".invalid") {
				return fmt.Errorf("shop recipe orders must use reserved .invalid synthetic email addresses")
			}
		}
	}
	if recipe.Kind == "content" && len(levels) < 3 {
		return fmt.Errorf("content recipe requires at least three graded levels")
	}
	recipe.PackageVersions = versions
	return nil
}

func nestedSlice(value map[string]any, key string) []any {
	items, _ := value[key].([]any)
	return items
}

// DecodeStrictRecipeCatalog is exposed to package tests to verify the catalog
// remains a closed JSON document with no trailing content.
func DecodeStrictRecipeCatalog(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var catalog recipeFile
	if err := decoder.Decode(&catalog); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("trailing JSON after WordPress recipe catalog")
	}
	return nil
}
