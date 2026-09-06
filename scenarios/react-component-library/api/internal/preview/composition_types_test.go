package preview

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCompositionBindingsCompileWithoutDuplicatingOwnedSlots(t *testing.T) {
	ui, err := filepath.Abs("../../../ui")
	if err != nil {
		t.Fatal(err)
	}
	compiler := filepath.Join(ui, "node_modules", ".bin", "tsc")
	if _, err := os.Stat(compiler); err != nil {
		t.Fatalf("TypeScript validation requires governed UI dependencies: %v", err)
	}
	c := Composition{Revision: "typed-fixture", Template: CompositionAsset{CatalogID: "frame", Version: "1.0.0", Export: "Frame"}, Regions: []CompositionRegion{{ID: "detail", Slot: []string{"data", "inspector"}}, {ID: "action", Parent: "detail", Slot: []string{"children"}}, {ID: "header", Slot: []string{"toolbar"}}, {ID: "footer", Slot: []string{"footer", "primary"}}, {ID: "options", Slot: []string{"options", "toolbar"}}}}
	imports := map[string]compositionImport{}
	for key, slug := range map[string]string{"$template": "Frame", "detail": "Card", "action": "Action", "header": "Header", "footer": "Header", "options": "Header"} {
		imports[key] = compositionImport{Slug: slug, Asset: CompositionAsset{Export: slug, Version: "1.0.0"}}
	}
	source, err := lowerComposition(c, imports)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	files := map[string]string{"composition.tsx": source, "assets.d.ts": `
 declare module '@vrooli/react-component-library/Frame/1.0.0' { export function Frame(props:{data:{inspector:import('react').ReactNode;businessCount:number};toolbar:import('react').ReactNode;footer:{primary:import('react').ReactNode};options?:{toolbar:import('react').ReactNode;value:number}}):import('react').ReactElement; }
 declare module '@vrooli/react-component-library/Card/1.0.0' { export function Card(props:{label:string;children:import('react').ReactNode}):import('react').ReactElement; }
 declare module '@vrooli/react-component-library/Action/1.0.0' { export function Action(props:{onClick:()=>void}):import('react').ReactElement; }
 declare module '@vrooli/react-component-library/Header/1.0.0' { export function Header(props:{title:string}):import('react').ReactElement; }
 `, "consumer.ts": `import type {CompositionBindings} from './composition';
 const valid:CompositionBindings = {$labels:{missing:'Missing',failed:'Failed'},$template:{data:{businessCount:2},options:{value:1}},detail:{label:'Details'},action:{onClick(){}},header:{title:'Header'},footer:{title:'Footer'},options:{title:'Options'}};
 // @ts-expect-error required business sibling must remain required
 const badTemplate:CompositionBindings['$template'] = {data:{}};
 // @ts-expect-error a generated child instantiates its optional parent, so sibling business data is required
 const missingSibling:CompositionBindings['$template'] = {data:{businessCount:2}};
 // @ts-expect-error required leaf business property must remain required
 const badAction:CompositionBindings['action'] = {};
 // @ts-expect-error generated child must not be duplicated by consumer
 const duplicate:CompositionBindings['detail'] = {label:'Details',children:'duplicate'};
 void valid; void badTemplate; void badAction; void duplicate;`}
	config := map[string]any{"compilerOptions": map[string]any{"strict": true, "noEmit": true, "skipLibCheck": true, "jsx": "react", "target": "ES2020", "moduleResolution": "node", "baseUrl": dir, "paths": map[string]any{"react": []string{filepath.Join(ui, "node_modules", "@types", "react")}}}, "include": []string{"*.ts", "*.tsx"}}
	raw, _ := json.Marshal(config)
	files["tsconfig.json"] = string(raw)
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	out, err := exec.Command(compiler, "--project", filepath.Join(dir, "tsconfig.json")).CombinedOutput()
	if err != nil {
		t.Fatalf("generated adoption contract does not typecheck: %v\n%s", err, out)
	}
}
