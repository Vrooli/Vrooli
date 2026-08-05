package releases

import "testing"

func TestSchemaIsTheReleaseDomainSchema(t *testing.T) {
	if len(Schema()) == 0 {
		t.Fatal("release schema is empty")
	}
}
