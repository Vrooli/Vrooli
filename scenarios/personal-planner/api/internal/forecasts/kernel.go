package forecasts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func Build(snapshot Snapshot, generatedAt time.Time) (Forecast, error) {
	start, err := time.ParseInLocation("2006-01-02", snapshot.StartDate, location(snapshot.Timezone))
	if err != nil {
		return Forecast{}, fmt.Errorf("start date: %w", err)
	}
	if snapshot.HorizonDays <= 0 || snapshot.HorizonDays > 90 {
		return Forecast{}, fmt.Errorf("horizon_days must be between 1 and 90")
	}
	if snapshot.KnownWork < 0 || snapshot.OccupiedMinutes < 0 || snapshot.UnresolvedWork < 0 {
		return Forecast{}, fmt.Errorf("snapshot values must be non-negative")
	}
	reservePerDay := int64(60)
	usablePerDay := int64(420)
	conservativePerDay := int64(300)
	available := int64(snapshot.HorizonDays) * usablePerDay
	occupied := snapshot.OccupiedMinutes
	if occupied > available {
		occupied = available
	}
	forecastable := available - occupied
	horizonEnd := start.AddDate(0, 0, snapshot.HorizonDays-1)
	base := Forecast{GeneratedAt: generatedAt.UTC().Format(time.RFC3339), Freshness: FreshnessCurrent, HorizonStart: snapshot.StartDate, HorizonEnd: horizonEnd.Format("2006-01-02"), AlgorithmVersion: AlgorithmVersion, KnownWorkMinutes: snapshot.KnownWork, AvailableMinutes: forecastable, ReserveMinutes: int64(snapshot.HorizonDays) * reservePerDay, UnresolvedWorkCount: snapshot.UnresolvedWork, InputFingerprint: fingerprint(snapshot)}
	if snapshot.KnownWork == 0 {
		base.ResultState, base.RiskState, base.Explanation = StateNoKnownWork, RiskUnknown, "No remaining work with a known effort is in the shared forecast input; add or estimate work before interpreting a finish date."
		base.CommitmentOutlooks = commitmentOutlooks(snapshot.Commitments, "", "", false)
		return base, nil
	}
	base.CentralFinish = finishDate(start, snapshot.KnownWork, forecastable, usablePerDay)
	base.CautiousFinish = finishDate(start, snapshot.KnownWork, forecastable, conservativePerDay)
	centralWithin := snapshot.KnownWork <= forecastable
	cautiousWithin := snapshot.KnownWork <= int64(snapshot.HorizonDays)*conservativePerDay
	if centralWithin {
		base.ResultState = StateFeasible
	} else {
		base.ResultState = StateBeyondHorizon
	}
	if snapshot.UnresolvedWork > 0 {
		base.RiskState = RiskUnknown
		base.Explanation = fmt.Sprintf("%d minutes of known remaining work are modeled across one shared resource; %d work item(s) still need an explicit effort estimate.", snapshot.KnownWork, snapshot.UnresolvedWork)
	} else if cautiousWithin {
		base.RiskState = RiskOnTrack
		base.Explanation = fmt.Sprintf("%d minutes of known remaining work fit inside the %d-day horizon after a protected %d-minute daily reserve; the cautious scenario also fits.", snapshot.KnownWork, snapshot.HorizonDays, reservePerDay)
	} else {
		base.RiskState = RiskElevated
		base.Explanation = fmt.Sprintf("The central scenario uses %d minutes per day, while the cautious scenario uses %d; the cautious finish extends beyond the current horizon.", usablePerDay, conservativePerDay)
	}
	base.CommitmentOutlooks = commitmentOutlooks(snapshot.Commitments, base.CentralFinish, base.CautiousFinish, true)
	return base, nil
}

func commitmentOutlooks(inputs []CommitmentInput, central, cautious string, hasForecast bool) []CommitmentOutlook {
	out := make([]CommitmentOutlook, 0, len(inputs))
	for _, input := range inputs {
		x := CommitmentOutlook{ID: input.ID, Result: input.Result, PromisedBoundary: input.PromisedBoundary, ForecastFinish: central}
		boundary := input.PromisedBoundary
		if len(boundary) >= 10 {
			boundary = boundary[:10]
		}
		if !hasForecast || central == "" {
			x.RiskState, x.Explanation = RiskUnknown, "No known-work finish is available to compare with this promise."
		} else if len(boundary) != 10 {
			x.RiskState, x.Explanation = RiskUnknown, "The promise boundary is not a comparable local date."
		} else if central > boundary {
			x.RiskState, x.Explanation = "at_risk", fmt.Sprintf("The central scenario finishes after the promised boundary of %s.", boundary)
		} else if cautious > boundary {
			x.RiskState, x.Explanation = RiskElevated, fmt.Sprintf("The central scenario fits by %s, but the cautious scenario reaches beyond %s.", central, boundary)
		} else {
			x.RiskState, x.Explanation = RiskOnTrack, fmt.Sprintf("Both labeled scenarios finish by the promised boundary of %s.", boundary)
		}
		out = append(out, x)
	}
	return out
}

func location(zone string) *time.Location {
	if zone == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return time.UTC
	}
	return loc
}

func finishDate(start time.Time, effort, available, perDay int64) string {
	if perDay <= 0 {
		return ""
	}
	days := (effort + perDay - 1) / perDay
	if days < 1 {
		days = 1
	}
	return start.AddDate(0, 0, int(days)-1).Format("2006-01-02")
}

func fingerprint(snapshot Snapshot) string {
	commitmentValue := ""
	for _, commitment := range snapshot.Commitments {
		commitmentValue += "|" + commitment.ID + "|" + commitment.PromisedBoundary
	}
	value := strings.Join([]string{snapshot.StartDate, snapshot.Timezone, strconv.Itoa(snapshot.HorizonDays), strconv.FormatInt(snapshot.KnownWork, 10), strconv.FormatInt(snapshot.UnresolvedWork, 10), strconv.FormatInt(snapshot.OccupiedMinutes, 10), commitmentValue}, "|")
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
