package transcriptindex

import "testing"

func TestFileRowStatusTransitions(t *testing.T) {
	allowed := map[string][]string{
		StatusPending:    {StatusPending, StatusIndexed, StatusOwned, StatusExcluded, StatusUnreadable},
		StatusIndexed:    {StatusIndexed, StatusPending, StatusOwned, StatusExcluded, StatusUnreadable},
		StatusOwned:      {StatusOwned, StatusPending, StatusIndexed, StatusExcluded, StatusUnreadable},
		StatusExcluded:   {StatusExcluded, StatusPending},
		StatusUnreadable: {StatusUnreadable, StatusPending, StatusIndexed, StatusOwned, StatusExcluded},
	}
	statuses := []string{StatusPending, StatusIndexed, StatusOwned, StatusExcluded, StatusUnreadable}
	for _, from := range statuses {
		for _, to := range statuses {
			want := false
			for _, candidate := range allowed[from] {
				want = want || candidate == to
			}
			if got := (FileRow{Status: from}).CanTransitionTo(to); got != want {
				t.Errorf("%s -> %s = %t, want %t", from, to, got, want)
			}
		}
	}
	for _, invalid := range []string{"", "capturing", "future-state"} {
		if (FileRow{Status: StatusPending}).CanTransitionTo(invalid) {
			t.Errorf("pending -> %q unexpectedly allowed", invalid)
		}
		if (FileRow{Status: invalid}).CanTransitionTo(StatusPending) {
			t.Errorf("%q -> pending unexpectedly allowed", invalid)
		}
	}
}
