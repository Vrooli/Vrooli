package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSafeCodeSupportsBoundedPublicLoopExpressions(t *testing.T) {
	created, err := executeSafeCode(`return Array.from({ length: 25 }, () => ({ json: { value: 1 } }));`, nil, time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("execute bounded Array.from generator: %v", err)
	}
	if len(created) != 25 {
		t.Fatalf("Array.from emitted %d items, want 25", len(created))
	}
	for index, item := range created {
		if got := item["json"].(map[string]any)["value"]; got != float64(1) {
			t.Fatalf("item %d value = %#v, want 1", index, got)
		}
	}

	code := `return $input.all().map(item => {
  return {
    json: {
      ...item.json,
      seconds: Math.floor(Math.random() * (60 - 20 + 1)) + 20
    }
  };
});`
	input := []map[string]any{{"value": float64(1)}, {"value": float64(2)}}
	now := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	first, err := executeSafeCode(code, input, now)
	if err != nil {
		t.Fatalf("execute bounded item transform: %v", err)
	}
	second, err := executeSafeCode(code, input, now)
	if err != nil {
		t.Fatalf("repeat bounded item transform: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same seeded Code input was not deterministic: %#v != %#v", first, second)
	}
	if len(first) != len(input) {
		t.Fatalf("transform emitted %d items, want %d", len(first), len(input))
	}
	for index, item := range first {
		jsonItem := item["json"].(map[string]any)
		if jsonItem["value"] != input[index]["value"] {
			t.Fatalf("item %d did not preserve the spread input: %#v", index, jsonItem)
		}
		seconds, ok := jsonItem["seconds"].(float64)
		if !ok || seconds < 20 || seconds > 60 || seconds != math.Trunc(seconds) {
			t.Fatalf("item %d random delay is outside the integer range 20..60: %#v", index, jsonItem["seconds"])
		}
	}
}

func TestSafeCodeDateLocaleStringUsesDeterministicUTCSubset(t *testing.T) {
	now := time.Date(2026, 11, 12, 15, 4, 5, 0, time.UTC)
	for _, tc := range []struct{ locale, want string }{
		{"id-ID", "12/11/2026 15.04.05"},
		{"ru-RU", "12.11.2026, 15:04:05"},
		{"en-US", "11/12/2026, 3:04:05 PM"},
	} {
		source := `return [{json:{formatted:new Date('2026-11-12T15:04:05Z').toLocaleString('` + tc.locale + `')}}];`
		if _, err := parseSafeCode(source); err != nil {
			t.Fatalf("parse locale %s: %v", tc.locale, err)
		}
		got, err := executeSafeCode(source, nil, now)
		if err != nil {
			t.Fatalf("evaluate locale %s: %v", tc.locale, err)
		}
		jsonValue := got[0]["json"].(map[string]any)["formatted"]
		if jsonValue != tc.want {
			t.Fatalf("locale %s output = %#v, want %q", tc.locale, jsonValue, tc.want)
		}
	}
	if _, err := executeSafeCode(`return [{json:{formatted:new Date('2026-11-12T15:04:05Z').toLocaleString('xx-YY')}}];`, nil, now); err == nil {
		t.Fatal("unsupported locale must fail closed")
	}
}

func TestSafeCodeArraySpreadDeduplicatesSetInInsertionOrder(t *testing.T) {
	source := `const unique=[...new Set(['a','b','a'])]; return [{json:{unique}}];`
	if _, err := parseSafeCode(source); err != nil {
		t.Fatalf("parse bounded Set spread: %v", err)
	}
	got, err := executeSafeCode(source, nil, time.Time{})
	if err != nil {
		t.Fatalf("evaluate bounded Set spread: %v", err)
	}
	values := got[0]["json"].(map[string]any)["unique"].(*safeArray).values
	if !reflect.DeepEqual(values, []any{"a", "b"}) {
		t.Fatalf("Set spread values = %#v, want insertion-ordered unique values", values)
	}
}

func TestSafeCodeArrayDestructuringBindsBoundedArrayValues(t *testing.T) {
	source := `const [month, day, year]=['12','3','2026']; return [{json:{month,day,year}}];`
	if _, err := parseSafeCode(source); err != nil {
		t.Fatalf("parse bounded array destructuring: %v", err)
	}
	got, err := executeSafeCode(source, nil, time.Time{})
	if err != nil {
		t.Fatalf("evaluate bounded array destructuring: %v", err)
	}
	data := got[0]["json"].(map[string]any)
	if data["month"] != "12" || data["day"] != "3" || data["year"] != "2026" {
		t.Fatalf("array destructuring bindings = %#v", data)
	}
	short, err := executeSafeCode(`const [first, second]=['only']; return [{json:{first,second}}];`, nil, time.Time{})
	if err != nil {
		t.Fatalf("short array uses undefined-like nil for missing binding: %v", err)
	}
	shortData := short[0]["json"].(map[string]any)
	if shortData["first"] != "only" || shortData["second"] != nil {
		t.Fatalf("short array bindings = %#v", shortData)
	}
	for _, unsupported := range []string{
		`const [,second]=['a','b']; return [];`,
		`const [first,...rest]=['a','b']; return [];`,
		`const [[first]]= [['a']]; return [];`,
		`const [first]={0:'a'}; return [];`,
	} {
		if _, err := parseSafeCode(unsupported); err == nil {
			t.Errorf("unsupported array destructuring accepted: %s", unsupported)
		}
	}
	if _, err := executeSafeCode(`const [first]='abc'; return [];`, nil, time.Time{}); err == nil {
		t.Fatal("array destructuring must reject non-array values")
	}
}

func TestSafeCodeStringPadStartUsesBoundedASCIIArguments(t *testing.T) {
	source := `return [{json:{padded:'7'.padStart(3,'0'),unchanged:'123'.padStart(2,'0'),space:'x'.padStart(3)}}];`
	got, err := executeSafeCode(source, nil, time.Time{})
	if err != nil {
		t.Fatalf("evaluate bounded string padStart: %v", err)
	}
	data := got[0]["json"].(map[string]any)
	if data["padded"] != "007" || data["unchanged"] != "123" || data["space"] != "  x" {
		t.Fatalf("padStart results = %#v", data)
	}
	for _, unsupported := range []string{
		`return [{json:{bad:'x'.padStart(1000,'0')}}];`,
		`return [{json:{bad:'x'.padStart(2,'é')}}];`,
		`return [{json:{bad:'x'.padStart(-1,'0')}}];`,
	} {
		if _, err := executeSafeCode(unsupported, nil, time.Time{}); err == nil {
			t.Errorf("unbounded padStart accepted: %s", unsupported)
		}
	}
}

func TestNormalizeSortDefaultsOmittedDirectionToAscending(t *testing.T) {
	fields, err := normalizeSort(map[string]any{
		"options": map[string]any{},
		"sortFieldsUi": map[string]any{"sortField": []any{
			map[string]any{"fieldName": "priorityRank"},
		}},
	})
	if err != nil {
		t.Fatalf("normalize static Sort field with omitted direction: %v", err)
	}
	if len(fields) != 1 || fields[0].Path != "priorityRank" || fields[0].Descending {
		t.Fatalf("normalized fields = %#v, want priorityRank ascending", fields)
	}
	for _, order := range []string{"ascending", "descending", "sideways"} {
		_, err := normalizeSort(map[string]any{
			"options": map[string]any{},
			"sortFieldsUi": map[string]any{"sortField": []any{
				map[string]any{"fieldName": "priorityRank", "order": order},
			}},
		})
		if (order == "sideways") != (err != nil) {
			t.Errorf("order %q error = %v", order, err)
		}
	}
}

func TestStopAndErrorNodeHaltsSyntheticExecution(t *testing.T) {
	node, err := normalizeWorkflowNodeVersion("stop", "Stop", "n8n-nodes-base.stopAndError", map[string]any{}, 1)
	if err != nil {
		t.Fatalf("normalize pinned Stop And Error node: %v", err)
	}
	if _, err := normalizeWorkflowNodeVersion("unsupported", "Stop", "n8n-nodes-base.stopAndError", map[string]any{"errorMessage": "dynamic"}, 1); err == nil {
		t.Fatal("Stop And Error must reject unmodeled parameters")
	}
	workflow := Workflow{
		ID: "synthetic-test",
		Nodes: []WorkflowNode{
			{ID: "entry", Name: "Manual", Type: "n8n-nodes-base.manualTrigger", Trigger: TriggerManual},
			node,
		},
		Connections: []Connection{{From: "entry", To: "stop", OutputPort: 0}},
	}
	var trace []TraceEvent
	_, err = executeWorkflowEntry(context.Background(), workflow, "entry", map[string]any{}, nil, false, time.Time{}, &trace, map[string]bool{}, 0)
	if err == nil || !strings.Contains(err.Error(), "Stop And Error halted") {
		t.Fatalf("Stop And Error execution error = %v", err)
	}
}

func TestGoogleTasksCreateDefaultsAreBoundedAndGeneric(t *testing.T) {
	node, err := normalizeWorkflowNodeVersion("task", "Create task", "n8n-nodes-base.googleTasks", map[string]any{
		"task":             "synthetic-list",
		"title":            "={{ $json.taskName }}",
		"additionalFields": map[string]any{"notes": "=Priority: {{ $json.priority }}"},
	}, 1)
	if err != nil {
		t.Fatalf("normalize pinned Google Tasks create defaults: %v", err)
	}
	if node.Request == nil || node.Request.Resource != "task" || node.Request.Action != "create" {
		t.Fatalf("normalized Google Tasks request = %#v, want task.create", node.Request)
	}
	for _, params := range []map[string]any{
		{"task": "synthetic-list", "title": "task", "additionalFields": map[string]any{"unknown": "value"}},
		{"task": "synthetic-list", "title": strings.Repeat("x", 4097), "additionalFields": map[string]any{}},
		{"task": "synthetic-list", "title": "task", "additionalFields": map[string]any{"notes": 3}},
	} {
		if _, err := normalizeWorkflowNodeVersion("task", "Create task", "n8n-nodes-base.googleTasks", params, 1); err == nil {
			t.Errorf("unsafe Google Tasks parameters accepted: %#v", params)
		}
	}
}

func TestSafeCodeSetAddPreservesBoundedInsertionOrderAndUniqueness(t *testing.T) {
	source := `const seen=new Set(); seen.add('a'); seen.add('b'); seen.add('a'); return [{json:{values:[...seen],size:seen.size,hasB:seen.has('b')}}];`
	if _, err := parseSafeCode(source); err != nil {
		t.Fatalf("parse bounded Set.add: %v", err)
	}
	got, err := executeSafeCode(source, nil, time.Time{})
	if err != nil {
		t.Fatalf("evaluate bounded Set.add: %v", err)
	}
	data := got[0]["json"].(map[string]any)
	values := data["values"].(*safeArray).values
	if !reflect.DeepEqual(values, []any{"a", "b"}) || data["size"] != float64(2) || data["hasB"] != true {
		t.Fatalf("Set.add result = %#v, want ordered unique values [a b], size 2 and hasB true", data)
	}
}

func TestSafeCodeSetAddRejectsUnboundedAndWrongArity(t *testing.T) {
	for _, source := range []string{
		`const seen=new Set(); seen.add({value:'x'}); return [];`,
		`const seen=new Set(); seen.add(); return [];`,
		`const seen=new Set(); seen.add('x','y'); return [];`,
	} {
		if _, err := executeSafeCode(source, nil, time.Time{}); err == nil {
			t.Errorf("expected bounded Set.add rejection for %s", source)
		}
	}
}

func TestSafeCodeStringIncludesUsesBoundedLiteralSemantics(t *testing.T) {
	source := `const message='temporary synthetic decline'; return [{json:{matches:message.includes('temporary'),misses:message.includes('fraud')}}];`
	if _, err := parseSafeCode(source); err != nil {
		t.Fatalf("parse bounded string includes: %v", err)
	}
	got, err := executeSafeCode(source, nil, time.Time{})
	if err != nil {
		t.Fatalf("evaluate bounded string includes: %v", err)
	}
	data := got[0]["json"].(map[string]any)
	if data["matches"] != true || data["misses"] != false {
		t.Fatalf("string includes results = %#v, want true/false", data)
	}
	if _, err := executeSafeCode(`return [{json:{bad:'x'.includes(1)}}];`, nil, time.Time{}); err == nil {
		t.Fatal("string includes must reject non-string search values")
	}
}

func TestSafeCodeSubsetRejectsUnboundedAndEffectfulSyntax(t *testing.T) {
	for name, source := range map[string]string{
		"unsafe property write":          `const a = {}; a.__proto__ = 1; return [];`,
		"global access":                  `return process.env;`,
		"network call":                   `return fetch('https://example.com');`,
		"dynamic eval":                   `return eval('1+1');`,
		"while loop":                     `while (true) { } return [];`,
		"unsupported regex":              `const r = /(?=x)/; return [];`,
		"input method on other receiver": `return [{json:{items: [].first()}}];`,
		"arbitrary property call":        `return globalThis.require('fs');`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseSafeCode(source); err == nil {
				t.Fatalf("unsafe or unsupported source was accepted: %s", source)
			}
		})
	}
}

func TestSafeCodeSliceRejectsRuntimeBoundsBeyondLimit(t *testing.T) {
	for _, source := range []string{
		`const text='x'; return [{json:{text:text.slice(0, 5000)}}];`,
		`const n=5000; return [{json:{text:'x'.slice(0,n)}}];`,
	} {
		if _, err := executeSafeCode(source, nil, time.Time{}); err == nil {
			t.Fatalf("expected bounded slice check to reject source: %s", source)
		}
	}
}

func TestSafeCodeSubsetInterpretsBoundedFirstFallbackRegexAndSlice(t *testing.T) {
	source := `const b=$input.first().json.body||{};
const email=String(b.email||'').trim().toLowerCase();
const valid=/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email);
return [{json:{email,valid,message:String(b.message||'').slice(0,2000)}}];`
	if _, err := parseSafeCode(source); err != nil {
		t.Fatalf("expected public 21098 Code form in bounded subset: %v", err)
	}
	message := strings.Repeat("x", 2100)
	got, err := executeSafeCode(source, []map[string]any{{"body": map[string]any{"email": " LEAD@EXAMPLE.INVALID ", "message": message}}}, time.Date(2026, 11, 10, 12, 13, 14, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	data := got[0]["json"].(map[string]any)
	if data["email"] != "lead@example.invalid" || data["valid"] != true || len(data["message"].(string)) != 2000 {
		t.Fatalf("bounded 21098 Code semantics differ: %#v", data)
	}
	if _, err := executeSafeCode(source, nil, time.Time{}); err == nil {
		t.Fatal("$input.first must fail closed with no input item")
	}
	if _, err := executeSafeCode(source, []map[string]any{{"body": map[string]any{"email": "lead@example.invalid", "message": strings.Repeat("x", 1999) + "😀"}}}, time.Time{}); err == nil {
		t.Fatal("slice must fail closed if the output boundary would split a UTF-16 surrogate pair")
	}
}

func TestSafeCodeRegexTestSupportsBoundedStaticRE2PatternsAndCaseFlag(t *testing.T) {
	source := `const mime=$input.first().json.mime||'';
const kind=/pdf|jpeg|png|tiff/.test(mime);
const unclear=/uncertain|unclear|illegible/i.test($input.first().json.quality||'');
return [{json:{kind,unclear}}];`
	input := []map[string]any{{"mime": "image/jpeg", "quality": "UnClear"}}
	got, err := executeSafeCode(source, input, time.Time{})
	if err != nil {
		t.Fatalf("execute bounded RE2 predicates: %v", err)
	}
	data := got[0]["json"].(map[string]any)
	if data["kind"] != true || data["unclear"] != true {
		t.Fatalf("unexpected bounded regex results: %#v", data)
	}
	for _, pattern := range []string{`/(?=unsafe)/`, `/\\1/`} {
		code := "const matched=" + pattern + ".test('safe'); return [{json:{matched}}];"
		if _, err := parseSafeCode(code); err == nil {
			t.Errorf("accepted unsupported regex pattern %s", pattern)
		}
	}
	linear := `const matched=/(a+)+$/.test('aaaaaaaaaaaaaaaa'); return [{json:{matched}}];`
	if _, err := executeSafeCode(linear, nil, time.Time{}); err != nil {
		t.Fatalf("RE2-compatible nested quantifier should remain bounded: %v", err)
	}
	stateful := `const matched=/pdf/g.test('pdf'); return [{json:{matched}}];`
	if _, err := executeSafeCode(stateful, nil, time.Time{}); err == nil {
		t.Fatal("global regex test must fail closed because JavaScript lastIndex is stateful")
	}
}

func TestSafeCodeSubsetInterpretsBoundedFormNormalization(t *testing.T) {
	source := `const out = [];
for (const item of $input.all()) {
 const b = item.json.body ?? item.json;
 const name = String(b.name ?? '').trim();
 const email = String(b.email ?? '').trim().toLowerCase();
 const errors = [];
 if (!name) errors.push('name is missing');
 if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) errors.push('email looks invalid');
 out.push({ json: { name, email, valid: errors.length === 0, errors: errors.join('; ') } });
}

return out;`
	input := []map[string]any{{"body": map[string]any{"name": " Ada ", "email": " ADA@EXAMPLE.INVALID "}}}
	got, err := executeSafeCode(source, input, time.Date(2026, 11, 10, 12, 13, 14, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	jsonData := got[0]["json"].(map[string]any)
	if jsonData["name"] != "Ada" || jsonData["email"] != "ada@example.invalid" || jsonData["valid"] != true || jsonData["errors"] != "" {
		t.Fatalf("unexpected interpreted output: %#v", jsonData)
	}
}

func TestSafeCodeSubsetRejectsOversizedItemsAndSource(t *testing.T) {
	if _, err := parseSafeCode(strings.Repeat(" ", maxSafeCodeBytes+1)); err == nil {
		t.Fatal("oversized source was accepted")
	}
	large := strings.Repeat("x", maxSafeCodeItemBytes+1)
	if _, err := executeSafeCode(`return [];`, []map[string]any{{"large": large}}, time.Unix(0, 0)); err == nil {
		t.Fatal("oversized input item was accepted")
	}
}

func TestPublic21013SanitizedExportRunsThroughSafeInterpreterAndSyntheticServices(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/21013.json")
	fixtureHash := fmt.Sprintf("%x", sha256.Sum256([]byte(workflowJSON)))
	if fixtureHash != "f863ec05b5a2011622b1451b56f492c2282ef2c21a80523715fa12954104245c" {
		t.Fatalf("sanitized public fixture changed without provenance update: %s", fixtureHash)
	}
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("normalize sanitized public workflow: %v", err)
	}
	var codeNode, webhookID string
	for _, node := range workflow.Nodes {
		if node.Type == "n8n-nodes-base.code" {
			codeNode = node.SafeCode
		}
		if node.Trigger == TriggerWebhook {
			webhookID = node.ID
		}
	}
	if codeNode == "" || webhookID == "" {
		t.Fatal("expected actual Code node and webhook trigger in sanitized export")
	}
	if _, err := parseSafeCode(codeNode); err != nil {
		t.Fatalf("actual public Code source must parse in bounded subset: %v", err)
	}

	profile := readFixture(t, "native_service_catalog.json")
	simulator, err := NewDescriptionSimulator(profile, false)
	if err != nil {
		t.Fatalf("load synthetic Gmail/Sheets service descriptions: %v", err)
	}
	validInput := map[string]any{"body": map[string]any{"name": "Synthetic Lead", "email": "LEAD@EXAMPLE.INVALID", "phone": "5550100", "message": "Synthetic form submission", "source": "synthetic website form"}}
	run, err := executeDetailedWithInput(context.Background(), workflow, webhookID, simulator, false, time.Date(2026, 11, 10, 12, 13, 14, 0, time.UTC), validInput)
	if err != nil {
		t.Fatalf("run sanitized public workflow through synthetic simulator: %v", err)
	}
	if !traceHasNode(run.Trace, "Clean & Validate") || !traceHasOperation(run.Trace, "googleSheets:sheet.append") || !traceHasOperation(run.Trace, "gmail:message.send") {
		t.Fatalf("expected code, synthetic Sheets and synthetic Gmail steps: %#v", run.Trace)
	}
	if strings.Contains(strings.ToLower(workflowJSON), "credential") || strings.Contains(workflowJSON, "YOUR BUSINESS NAME") {
		t.Fatal("sanitized fixture retained credential labels or client-specific mail template data")
	}
	if run.Output["valid"] != true || run.Output["email"] != "lead@example.invalid" {
		t.Fatalf("safe Code node output did not preserve the synthetic valid lead: %#v", run.Output)
	}
	if run.Output["received_at"] != "2026-11-10T12:13:14.000Z" {
		t.Fatalf("expected deterministic synthetic date, got %#v", run.Output["received_at"])
	}

	invalid, err := executeDetailedWithInput(context.Background(), workflow, webhookID, simulator, false, time.Date(2026, 11, 10, 12, 13, 14, 0, time.UTC), map[string]any{"body": map[string]any{"name": "Synthetic Invalid", "email": "not-an-email", "message": "Made-up invalid address"}})
	if err != nil {
		t.Fatalf("run invalid synthetic form branch: %v", err)
	}
	if !traceHasNode(invalid.Trace, "Respond Error") || traceHasNode(invalid.Trace, "Save to Google Sheet") || traceHasNode(invalid.Trace, "Auto-reply (Gmail)") {
		t.Fatalf("invalid form must take the error path without synthetic Sheets/Gmail effects: %#v", invalid.Trace)
	}
}

func TestPublic21098SanitizedExportRunsThroughGenericSafeCodeFilterAndServices(t *testing.T) {
	workflowJSON := readFixture(t, "public-workflows/21098.json")
	fixtureHash := fmt.Sprintf("%x", sha256.Sum256([]byte(workflowJSON)))
	if fixtureHash != "ef2efa5c177289695c072f0880b93ee7816f565214922cab821f001d95d52f88" {
		t.Fatalf("sanitized public fixture changed without provenance update: %s", fixtureHash)
	}
	workflow, _, _, err := normalizeWorkflow(workflowJSON)
	if err != nil {
		t.Fatalf("normalize sanitized public workflow: %v", err)
	}
	var codeSource, webhookID string
	for _, node := range workflow.Nodes {
		if node.Type == "n8n-nodes-base.code" {
			codeSource = node.SafeCode
		}
		if node.Trigger == TriggerWebhook {
			webhookID = node.ID
		}
	}
	if codeSource == "" || webhookID == "" {
		t.Fatal("expected actual Code node and webhook trigger in sanitized export")
	}
	if _, err := parseSafeCode(codeSource); err != nil {
		t.Fatalf("actual public Code source must parse in bounded subset: %v", err)
	}
	profile := readFixture(t, "native_service_catalog.json")
	simulator, err := NewDescriptionSimulator(profile, false)
	if err != nil {
		t.Fatalf("load synthetic Sheets/Slack/Gmail service descriptions: %v", err)
	}
	now := time.Date(2026, 11, 10, 12, 13, 14, 0, time.UTC)
	input := map[string]any{"body": map[string]any{"name": "Synthetic Lead", "email": " LEAD@EXAMPLE.INVALID ", "message": strings.Repeat("made-up ", 320), "source": "synthetic form"}}
	run, err := executeDetailedWithInput(context.Background(), workflow, webhookID, simulator, false, now, input)
	if err != nil {
		t.Fatalf("run unchanged sanitized public graph through synthetic simulator: %v", err)
	}
	for _, op := range []string{"Safe Code subset", "Filter valid true", "googleSheets:sheet.append", "slack:message.post", "gmail:message.send"} {
		if !traceHasOperation(run.Trace, op) {
			t.Fatalf("expected operation %q in generic simulator trace: %#v", op, run.Trace)
		}
	}
	if run.Output["valid"] != true || run.Output["email"] != "lead@example.invalid" || run.Output["received_at"] != "2026-11-10T12:13:14.000Z" {
		t.Fatalf("unexpected normalized public Code output: %#v", run.Output)
	}
	if len(run.Output["message"].(string)) != 2000 {
		t.Fatalf("bounded public message truncation was not applied: %d", len(run.Output["message"].(string)))
	}

	invalid, err := executeDetailedWithInput(context.Background(), workflow, webhookID, simulator, false, now, map[string]any{"body": map[string]any{"name": "Synthetic Invalid", "email": "not-an-email", "message": "Made-up invalid form"}})
	if err != nil {
		t.Fatalf("run invalid synthetic form through Filter: %v", err)
	}
	if traceHasOperation(invalid.Trace, "googleSheets:sheet.append") || traceHasOperation(invalid.Trace, "slack:message.post") || traceHasOperation(invalid.Trace, "gmail:message.send") {
		t.Fatalf("invalid item must be dropped before synthetic service operations: %#v", invalid.Trace)
	}
}

func TestCodeNodeUsesPinnedRunOnceForAllItemsDefault(t *testing.T) {
	raw := `{"id":"code-batch","nodes":[
		{"id":"manual","name":"Manual","type":"n8n-nodes-base.manualTrigger","parameters":{}},
		{"id":"split","name":"Split records","type":"n8n-nodes-base.splitOut","parameters":{"fieldToSplitOut":"records"}},
		{"id":"code","name":"Copy bounded input items","type":"n8n-nodes-base.code","typeVersion":2,"parameters":{"jsCode":"return $input.all();"}},
		{"id":"done","name":"Done","type":"n8n-nodes-base.noOp","parameters":{}}
	],"connections":{"Manual":{"main":[[{"node":"Split records","type":"main","index":0}]]},"Split records":{"main":[[{"node":"Copy bounded input items","type":"main","index":0}]]},"Copy bounded input items":{"main":[[{"node":"Done","type":"main","index":0}]]}}}`
	workflow, _, _, err := normalizeWorkflow(raw)
	if err != nil {
		t.Fatalf("normalize generic Code batch workflow: %v", err)
	}
	run, err := executeDetailedWithInput(context.Background(), workflow, workflow.EntryNodeIDs[0], nil, false, time.Unix(0, 0), map[string]any{"records": []any{map[string]any{"id": "synthetic-1"}, map[string]any{"id": "synthetic-2"}}})
	if err != nil {
		t.Fatal(err)
	}
	codeEvents, doneEvents := 0, 0
	for _, event := range run.Trace {
		if event.Operation == "code-batch/Safe Code subset" {
			codeEvents++
			if event.ResponseSummary != "safe Code interpreter emitted 2 synthetic items" {
				t.Fatalf("unexpected bounded Code batch result: %#v", event)
			}
		}
		if event.Operation == "code-batch/No Operation" {
			doneEvents++
		}
	}
	if codeEvents != 1 || doneEvents != 2 {
		t.Fatalf("Code default mode must execute once on all inputs and forward both outputs: %#v", run.Trace)
	}
	perItem := strings.Replace(raw, `"parameters":{"jsCode":"return $input.all();"}`, `"parameters":{"mode":"runOnceForEachItem","jsCode":"return {json:{...$json, seen:true}};"}`, 1)
	perWorkflow, _, _, err := normalizeWorkflow(perItem)
	if err != nil {
		t.Fatalf("normalize public per-item Code mode: %v", err)
	}
	perRun, err := executeDetailedWithInput(context.Background(), perWorkflow, perWorkflow.EntryNodeIDs[0], nil, false, time.Unix(0, 0), map[string]any{"records": []any{map[string]any{"id": "synthetic-1"}, map[string]any{"id": "synthetic-2"}}})
	if err != nil {
		t.Fatalf("execute per-item Code mode: %v", err)
	}
	perItemEvents := 0
	for _, event := range perRun.Trace {
		if event.Operation == "code-batch/Safe Code subset" && event.ResponseSummary == "safe Code interpreter emitted 2 synthetic items" {
			perItemEvents++
		}
	}
	if perItemEvents != 1 {
		t.Fatalf("expected per-item Code to transform both split records, trace=%#v", perRun.Trace)
	}
}

func TestSafeCodeSupportsBoundedPublicRoutingAndOptionalChainPatterns(t *testing.T) {
	source := `const item=$input.first().json;
let score=0;
const sourceScores={paid:30,referral:25,webinar:20,organic:15};
score += sourceScores[item.leadSource] || 5;
if (item.companySize >= 500) score += 30;
else if (item.companySize >= 100) score += 20;
const domain=item.email.split('@')[1] || '';
const freeEmailDomains=['gmail.com','yahoo.com'];
if (!freeEmailDomains.includes(domain)) score += 20;
const repMap={north:{hot:'rep@example.invalid',warm:'rep@example.invalid',cold:'rep@example.invalid'}};
const region=item.region in repMap ? item.region : 'unassigned';
const tier=score>=60?'hot':score>=35?'warm':'cold';
const assignedRep=repMap[region]?.[tier] || 'fallback@example.invalid';
return [{json:{score,region,tier,assignedRep,domain}}];`
	if _, err := parseSafeCode(source); err != nil {
		t.Fatalf("parse public routing Code subset: %v", err)
	}
	input := []map[string]any{{"leadSource": "referral", "companySize": float64(500), "email": "lead@example.invalid", "region": "north"}}
	got, err := executeSafeCode(source, input, time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("execute public routing Code subset: %v", err)
	}
	data := got[0]["json"].(map[string]any)
	if data["score"] != float64(75) || data["region"] != "north" || data["tier"] != "hot" || data["assignedRep"] != "rep@example.invalid" || data["domain"] != "example.invalid" {
		t.Fatalf("public routing result = %#v", data)
	}
	optional, err := executeSafeCode(`const ai=$input.first().json; const draft=ai.message?.content || ai.choices?.[0]?.message?.content || 'fallback'; return [{json:{draft}}];`, []map[string]any{{"choices": []any{map[string]any{"message": map[string]any{"content": "draft text"}}}}}, time.Time{})
	if err != nil {
		t.Fatalf("execute optional-chain fallback: %v", err)
	}
	if got := optional[0]["json"].(map[string]any)["draft"]; got != "draft text" {
		t.Fatalf("optional-chain result = %#v", got)
	}
}

func TestSafeSetExpressionEvaluatesBoundedN8nInputExpression(t *testing.T) {
	for _, input := range []struct {
		expression string
		want       any
	}{
		{"={{ $json.body.first_name?.trim() || $json.body.firstName?.trim() || '' }}", "Synthetic"},
		{"={{ $json.body.email?.toLowerCase().trim() }}", "lead@example.invalid"},
		{"={{ parseInt($json.body.company_size || $json.body.companySize || '0') }}", float64(500)},
	} {
		expression, err := normalizeSafeExpression(input.expression)
		if err != nil {
			t.Fatalf("normalize %q: %v", input.expression, err)
		}
		got, err := executeSafeExpression(expression, map[string]any{"body": map[string]any{"first_name": " Synthetic ", "email": " LEAD@EXAMPLE.INVALID ", "company_size": "500"}}, time.Time{})
		if got == nil {
			direct, directErr := executeSafeCode(`return [{json:{value: `+expression+`}}];`, []map[string]any{{"body": map[string]any{"first_name": " Synthetic ", "email": " LEAD@EXAMPLE.INVALID ", "company_size": "500"}}}, time.Time{})
			t.Logf("direct expression: %#v err=%v expression=%q", direct, directErr, expression)
		}
		if err != nil {
			t.Fatalf("evaluate %q: %v", input.expression, err)
		}
		if !reflect.DeepEqual(got, input.want) {
			t.Fatalf("expression %q = %#v, want %#v", input.expression, got, input.want)
		}
	}
}

func TestSafeCodeReadsBoundedPriorNodeOutputsButCannotReassignConst(t *testing.T) {
	for _, source := range []string{
		`const score=1; score += 1; return [];`,
		`const score=1; score=2; return [];`,
	} {
		if _, err := parseSafeCode(source); err == nil {
			t.Fatalf("const reassignment was accepted: %s", source)
		}
	}
	source := `const lead=$('Score & Route Lead').first().json; return [{json:{email:lead.email,score:lead.leadScore}}];`
	if _, err := parseSafeCode(source); err != nil {
		t.Fatalf("parse named-node lookup: %v", err)
	}
	prior := map[string][]map[string]any{"Score & Route Lead": {{"email": "lead@example.invalid", "leadScore": float64(75)}}}
	got, err := executeSafeCodeWithNodeOutputs(source, nil, prior, time.Time{})
	if err != nil {
		t.Fatalf("read bounded prior node item: %v", err)
	}
	data := got[0]["json"].(map[string]any)
	if data["email"] != "lead@example.invalid" || data["score"] != float64(75) {
		t.Fatalf("named node projection = %#v", data)
	}
}

func TestSafeCodeAllowsFunctionParameterForNamedNodeLookup(t *testing.T) {
	source := `function firstFrom(nodeName) { return $(nodeName).first().json; } const page=firstFrom('Fetch Static Pages'); return [{json:{id:page.id}}];`
	prior := map[string][]map[string]any{"Fetch Static Pages": {{"id": float64(17), "title": "Synthetic page"}}}
	got, err := executeSafeCodeWithNodeOutputs(source, nil, prior, time.Time{})
	if err != nil {
		t.Fatalf("execute bounded function parameter lookup: %v", err)
	}
	if len(got) != 1 || got[0]["json"].(map[string]any)["id"] != float64(17) {
		t.Fatalf("dynamic named-node lookup = %#v", got)
	}
}

func TestSafeCodeBoundsArrayIsArrayAndConcat(t *testing.T) {
	source := `const isList=Array.isArray([1]); const joined=[1].concat([2]); return [{json:{isList:isList,joined:joined}}];`
	got, err := executeSafeCode(source, nil, time.Time{})
	if err != nil {
		t.Fatalf("execute bounded array helpers: %v", err)
	}
	output := got[0]["json"].(map[string]any)
	if output["isList"] != true {
		t.Fatalf("Array.isArray = %#v", output["isList"])
	}
	joined := output["joined"].(*safeArray).values
	if len(joined) != 2 || joined[0] != float64(1) || joined[1] != float64(2) {
		t.Fatalf("bounded concat result = %#v", joined)
	}
}

func TestSafeCodeHTMLRegexSubsetHandlesBreakAndTags(t *testing.T) {
	source := `const clean="<P>A<br/>B</P>".replace(/<br\s*\/?>/gi," ").replace(/<[^>]+>/g,""); return [{json:{clean:clean}}];`
	got, err := executeSafeCode(source, nil, time.Time{})
	if err != nil {
		t.Fatalf("execute bounded HTML regexes: %v", err)
	}
	if got[0]["json"].(map[string]any)["clean"] != "A B" {
		t.Fatalf("HTML cleanup result = %#v", got)
	}
}

func TestSafeCodeRegexReplacementSupportsBoundedCaptureCallback(t *testing.T) {
	source := `const clean="Code: &#65;".replace(/&#(\d+);/g, (match, digits) => String.fromCodePoint(Number(digits))); return [{json:{clean:clean}}];`
	got, err := executeSafeCode(source, nil, time.Time{})
	if err != nil {
		t.Fatalf("execute bounded numeric-entity callback: %v", err)
	}
	if got[0]["json"].(map[string]any)["clean"] != "Code: A" {
		t.Fatalf("decoded entity = %#v", got)
	}
}

func TestSafeCodeBoundedSetSupportsMembership(t *testing.T) {
	source := `const seen=new Set(["page_1","post_2"]); return [{json:{page:seen.has("page_1"),other:seen.has("page_3")}}];`
	got, err := executeSafeCode(source, nil, time.Time{})
	if err != nil {
		t.Fatalf("execute bounded Set membership: %v", err)
	}
	output := got[0]["json"].(map[string]any)
	if output["page"] != true || output["other"] != false {
		t.Fatalf("Set.has results = %#v", output)
	}
}

func TestSafeCodeStringIndexOfSupportsPrefixGuards(t *testing.T) {
	source := `const id="wc_page_42"; return [{json:{prefix:id.indexOf("wc_page_")===0,other:id.indexOf("wc_post_")}}];`
	got, err := executeSafeCode(source, nil, time.Time{})
	if err != nil {
		t.Fatalf("execute bounded string indexOf: %v", err)
	}
	output := got[0]["json"].(map[string]any)
	if output["prefix"] != true || output["other"] != float64(-1) {
		t.Fatalf("indexOf results = %#v", output)
	}
}

func TestSyntheticEngineCarriesNamedNodeItemsIntoLaterCodeNodes(t *testing.T) {
	workflow := Workflow{
		ID:           "named-node-contract",
		EntryNodeIDs: []string{"trigger"},
		Nodes: []WorkflowNode{
			{ID: "trigger", Name: "Manual Trigger", Type: "n8n-nodes-base.manualTrigger", Trigger: TriggerManual},
			{ID: "make", Name: "Make first", Type: "n8n-nodes-base.code", SafeCode: `const item=$input.first().json; return [{json:{value:item.value+1}}];`},
			{ID: "read", Name: "Read prior", Type: "n8n-nodes-base.code", SafeCode: `const prior=$('Make first').first().json; return [{json:{copied:prior.value}}];`},
		},
		Connections: []Connection{{From: "trigger", To: "make"}, {From: "make", To: "read"}},
	}
	result, err := executeDetailedWithInput(context.Background(), workflow, "trigger", nil, false, time.Time{}, map[string]any{"value": float64(41)})
	if err != nil {
		t.Fatalf("run named-node Code path: %v", err)
	}
	if result.Output["copied"] != float64(42) {
		t.Fatalf("named prior-node output = %#v", result.Output)
	}
}

func TestSafeSetExpressionNowToISOUsesInjectedClock(t *testing.T) {
	now := time.Date(2026, 11, 11, 1, 2, 3, 0, time.UTC)
	expression, err := normalizeSafeExpression("={{ $now.toISO() }}")
	if err != nil {
		t.Fatalf("normalize injected clock expression: %v", err)
	}
	got, err := executeSafeExpression(expression, map[string]any{}, now)
	if err != nil {
		t.Fatalf("evaluate injected clock expression: %v", err)
	}
	if got != now.Format(time.RFC3339Nano) {
		t.Fatalf("$now.toISO() = %#v, want %q", got, now.Format(time.RFC3339Nano))
	}
}

func TestSafeCodeObjectKeysIsBoundedAndDeterministic(t *testing.T) {
	source := `const keys = Object.keys($json); return {json:{...$json, keys}};`
	if _, err := parseSafeCodeMode(source, false, true); err != nil {
		t.Fatalf("parse bounded Object.keys: %v", err)
	}
	got, err := executeSafeCodePerItemWithNodeOutputs(source, map[string]any{"z": 1, "a": 2}, nil, time.Time{})
	if err != nil {
		t.Fatalf("execute bounded Object.keys: %v", err)
	}
	keys := got["json"].(map[string]any)["keys"].(*safeArray).values
	if !reflect.DeepEqual(keys, []any{"a", "z"}) {
		t.Fatalf("Object.keys order = %#v, want sorted keys", keys)
	}
}

func TestSafeCodeBoundsRecursiveLocalFunctions(t *testing.T) {
	source := `function descend(n) { if (n > 0) { return descend(n - 1); } return n; } return [{json:{value:descend(100)}}];`
	if _, err := parseSafeCode(source); err != nil {
		t.Fatalf("parse bounded local recursion: %v", err)
	}
	if _, err := executeSafeCode(source, nil, time.Time{}); err == nil || !strings.Contains(err.Error(), "call depth exceeds") {
		t.Fatalf("deep recursion should stop at the interpreter bound, got %v", err)
	}
}
