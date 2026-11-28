package profiles

import "testing"

func TestSchemaIsTheProfileDomainSchema(t *testing.T) {
	if len(Schema()) == 0 {
		t.Fatal("profile schema is empty")
	}
}
