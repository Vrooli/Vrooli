package consumerdeclaration

import "testing"

func TestValidateAcceptsPublicDeclaration(t *testing.T) {
	_, got := Validate([]byte(`{"schemaVersion":"browser-automation-studio.consumer-declaration/v1","profiles":[{"key":"operator-browser","workflowRef":"bas/workflows/publish.json","allowedVariables":["POST_ID"],"preferences":{"locale":"en-US","headless":false}}]}`))
	if !got.Valid() {
		t.Fatalf("issues = %#v", got.Issues)
	}
}
func TestValidateRejectsSecretsRuntimeProfilesAndDuplicates(t *testing.T) {
	_, got := Validate([]byte(`{"schemaVersion":"browser-automation-studio.consumer-declaration/v1","profiles":[{"key":"dup","workflowRef":"a","preferences":{"password":"nope"}},{"key":"dup","workflowRef":"b","runtimeProfile":"abc"}]}`))
	if got.Valid() {
		t.Fatal("expected invalid declaration")
	}
}
