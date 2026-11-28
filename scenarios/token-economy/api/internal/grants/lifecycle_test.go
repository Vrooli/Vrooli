package grants

import "testing"

func TestGrantStatusTransitionMatrixRejectsInvalidEdgesWithoutMutation(t *testing.T) {
	states := []GrantStatus{
		GrantStatusDraft,
		GrantStatusLive,
		GrantStatusExhausted,
		GrantStatusExpired,
		GrantStatusRevoked,
		"unknown",
	}
	for _, from := range states {
		for _, to := range states {
			t.Run(string(from)+"_to_"+string(to), func(t *testing.T) {
				grant := Grant{Status: from}
				err := grant.TransitionTo(to)
				valid := from != GrantStatusRevoked && from != "unknown" && to == GrantStatusRevoked
				if valid {
					if err != nil {
						t.Fatalf("legal transition returned error: %v", err)
					}
					if grant.Status != to {
						t.Fatalf("status = %q, want %q", grant.Status, to)
					}
					return
				}
				if err == nil {
					t.Fatal("invalid transition succeeded")
				}
				if grant.Status != from {
					t.Fatalf("invalid transition changed status to %q", grant.Status)
				}
			})
		}
	}
}
