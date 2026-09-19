package workspace

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Service interface {
	Get(context.Context) (Profile, error)
	Update(context.Context, UpdateInput) (Profile, error)
	ListAvailability(context.Context) (Availability, error)
	ReplaceAvailability(context.Context, AvailabilityInput) (Availability, error)
}

func (s *service) ListAvailability(ctx context.Context) (Availability, error) {
	return s.repo.ListAvailability(ctx)
}

func (s *service) ReplaceAvailability(ctx context.Context, in AvailabilityInput) (Availability, error) {
	if len(in.Windows) == 0 && len(in.Exceptions) == 0 {
		return Availability{}, ErrInvalidAvailability{"availability", "configure at least one window or exception"}
	}
	for i, w := range in.Windows {
		if w.Weekday < 1 || w.Weekday > 7 {
			return Availability{}, ErrInvalidAvailability{fmt.Sprintf("windows[%d].weekday", i), "must be between 1 and 7"}
		}
		if w.StartMinute < 0 || w.EndMinute > 1440 || w.StartMinute >= w.EndMinute {
			return Availability{}, ErrInvalidAvailability{fmt.Sprintf("windows[%d].interval", i), "must be within 00:00-24:00 and start before end"}
		}
		if _, err := timeLoadLocation(w.Timezone); err != nil {
			return Availability{}, ErrInvalidAvailability{fmt.Sprintf("windows[%d].timezone", i), "must be an IANA timezone"}
		}
	}
	for i, e := range in.Exceptions {
		if e.StartMinute < 0 || e.EndMinute > 1440 || e.StartMinute >= e.EndMinute {
			return Availability{}, ErrInvalidAvailability{fmt.Sprintf("exceptions[%d].interval", i), "must be within 00:00-24:00 and start before end"}
		}
		if e.Kind != "protected" && e.Kind != "extra" {
			return Availability{}, ErrInvalidAvailability{fmt.Sprintf("exceptions[%d].kind", i), "must be protected or extra"}
		}
		if _, err := time.Parse("2006-01-02", e.Date); err != nil {
			return Availability{}, ErrInvalidAvailability{fmt.Sprintf("exceptions[%d].date", i), "must be YYYY-MM-DD"}
		}
	}
	for i := range in.Windows {
		for j := i + 1; j < len(in.Windows); j++ {
			a, b := in.Windows[i], in.Windows[j]
			if a.Weekday == b.Weekday && a.EffectiveStartDate == b.EffectiveStartDate && a.EffectiveEndDate == b.EffectiveEndDate && a.StartMinute < b.EndMinute && b.StartMinute < a.EndMinute {
				return Availability{}, ErrInvalidAvailability{"windows", "overlapping intervals need distinct effective dates or priority"}
			}
		}
	}
	return s.repo.ReplaceAvailability(ctx, in)
}

type service struct{ repo Repository }

func NewService(repo Repository) Service { return &service{repo: repo} }

func (s *service) Get(ctx context.Context) (Profile, error) { return s.repo.Get(ctx) }

func (s *service) Update(ctx context.Context, in UpdateInput) (Profile, error) {
	if strings.TrimSpace(in.Timezone) == "" {
		return Profile{}, ErrInvalidProfile{"timezone", "required"}
	}
	if _, err := timeLoadLocation(in.Timezone); err != nil {
		return Profile{}, ErrInvalidProfile{"timezone", "must be an IANA timezone"}
	}
	if in.WeekStart != "monday" && in.WeekStart != "sunday" {
		return Profile{}, ErrInvalidProfile{"week_start", "must be monday or sunday"}
	}
	if in.DailyCapacityMinutes < 0 || in.DailyCapacityMinutes > 1440 {
		return Profile{}, ErrInvalidProfile{"daily_capacity_minutes", "must be between 0 and 1440"}
	}
	if in.ReserveMinutes < 0 || in.ReserveMinutes > in.DailyCapacityMinutes {
		return Profile{}, ErrInvalidProfile{"reserve_minutes", "must not exceed daily capacity"}
	}
	if in.FocusSessionMinutes < 5 || in.FocusSessionMinutes > 240 {
		return Profile{}, ErrInvalidProfile{"focus_session_minutes", "must be between 5 and 240"}
	}
	return s.repo.Update(ctx, in)
}

var timeLoadLocation = func(name string) (*time.Location, error) { return time.LoadLocation(name) }
