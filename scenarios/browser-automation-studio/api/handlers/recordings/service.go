package recordings

import (
	"context"
	"encoding/json"
	"strings"

	"connectrpc.com/connect"

	autodriver "github.com/vrooli/browser-automation-studio/automation/driver"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	recordingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/recordings"
)

type service struct {
	deps Deps
}

// =============================================================================
// Storage state — playwright storage_state shape used for load/save.
// =============================================================================

// playwrightStorageState matches the on-disk Playwright storage_state format.
type playwrightStorageState struct {
	Cookies []playwrightCookie `json:"cookies"`
	Origins []playwrightOrigin `json:"origins"`
}

type playwrightCookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HttpOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	SameSite string  `json:"sameSite"`
}

type playwrightOrigin struct {
	Origin       string                  `json:"origin"`
	LocalStorage []playwrightLocalStItem `json:"localStorage"`
}

type playwrightLocalStItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// =============================================================================
// GetStorageState
// =============================================================================

func (s *service) GetStorageState(
	_ context.Context,
	req *connect.Request[recordingsv1.GetStorageStateRequest],
) (*connect.Response[recordingsv1.GetStorageStateResponse], error) {
	profileID, err := requireProfileID(req.Msg.GetProfileId())
	if err != nil {
		return nil, err
	}
	profile, err := s.deps.Repo.GetProfile(profileID)
	if err != nil {
		return nil, mapStoreError(err)
	}
	masked, err := s.deps.Repo.MaskStorageState(profile.StorageState)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	cookies := make([]*recordingsv1.Cookie, 0, len(masked.Cookies))
	for _, c := range masked.Cookies {
		cookies = append(cookies, &recordingsv1.Cookie{
			Name:        c.Name,
			Value:       c.Value,
			ValueMasked: c.ValueMasked,
			Domain:      c.Domain,
			Path:        c.Path,
			Expires:     c.Expires,
			HttpOnly:    c.HttpOnly,
			Secure:      c.Secure,
			SameSite:    c.SameSite,
		})
	}
	origins := make([]*recordingsv1.Origin, 0, len(masked.Origins))
	for _, o := range masked.Origins {
		items := make([]*recordingsv1.LocalStorageItem, 0, len(o.LocalStorage))
		for _, item := range o.LocalStorage {
			items = append(items, &recordingsv1.LocalStorageItem{Name: item.Name, Value: item.Value})
		}
		origins = append(origins, &recordingsv1.Origin{Origin: o.Origin, LocalStorage: items})
	}
	return connect.NewResponse(&recordingsv1.GetStorageStateResponse{
		Cookies: cookies,
		Origins: origins,
		Stats: &recordingsv1.StorageStats{
			CookieCount:       int32(masked.Stats.CookieCount),
			LocalStorageCount: int32(masked.Stats.LocalStorageCount),
			OriginCount:       int32(masked.Stats.OriginCount),
		},
	}), nil
}

// =============================================================================
// Storage mutations
// =============================================================================

func (s *service) ClearAllStorage(
	_ context.Context,
	req *connect.Request[recordingsv1.ClearAllStorageRequest],
) (*connect.Response[recordingsv1.StorageMutationResponse], error) {
	profileID, err := requireProfileID(req.Msg.GetProfileId())
	if err != nil {
		return nil, err
	}
	empty := json.RawMessage(`{"cookies":[],"origins":[]}`)
	if _, err := s.deps.Repo.SaveStorageState(profileID, empty); err != nil {
		return nil, mapStoreError(err)
	}
	return connect.NewResponse(&recordingsv1.StorageMutationResponse{Status: "cleared"}), nil
}

func (s *service) ClearAllCookies(
	_ context.Context,
	req *connect.Request[recordingsv1.ClearAllCookiesRequest],
) (*connect.Response[recordingsv1.StorageMutationResponse], error) {
	return s.mutateStorage(req.Msg.GetProfileId(), func(state *playwrightStorageState) {
		state.Cookies = nil
	})
}

func (s *service) DeleteCookiesByDomain(
	_ context.Context,
	req *connect.Request[recordingsv1.DeleteCookiesByDomainRequest],
) (*connect.Response[recordingsv1.StorageMutationResponse], error) {
	domain := strings.TrimSpace(req.Msg.GetDomain())
	if domain == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errDomainRequired)
	}
	return s.mutateStorage(req.Msg.GetProfileId(), func(state *playwrightStorageState) {
		filtered := make([]playwrightCookie, 0, len(state.Cookies))
		for _, c := range state.Cookies {
			if c.Domain != domain {
				filtered = append(filtered, c)
			}
		}
		state.Cookies = filtered
	})
}

func (s *service) DeleteCookie(
	_ context.Context,
	req *connect.Request[recordingsv1.DeleteCookieRequest],
) (*connect.Response[recordingsv1.StorageMutationResponse], error) {
	domain := strings.TrimSpace(req.Msg.GetDomain())
	if domain == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errDomainRequired)
	}
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNameRequired)
	}
	return s.mutateStorage(req.Msg.GetProfileId(), func(state *playwrightStorageState) {
		filtered := make([]playwrightCookie, 0, len(state.Cookies))
		for _, c := range state.Cookies {
			if !(c.Domain == domain && c.Name == name) {
				filtered = append(filtered, c)
			}
		}
		state.Cookies = filtered
	})
}

func (s *service) ClearAllLocalStorage(
	_ context.Context,
	req *connect.Request[recordingsv1.ClearAllLocalStorageRequest],
) (*connect.Response[recordingsv1.StorageMutationResponse], error) {
	return s.mutateStorage(req.Msg.GetProfileId(), func(state *playwrightStorageState) {
		state.Origins = nil
	})
}

func (s *service) DeleteLocalStorageByOrigin(
	_ context.Context,
	req *connect.Request[recordingsv1.DeleteLocalStorageByOriginRequest],
) (*connect.Response[recordingsv1.StorageMutationResponse], error) {
	origin := strings.TrimSpace(req.Msg.GetOrigin())
	if origin == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errOriginRequired)
	}
	return s.mutateStorage(req.Msg.GetProfileId(), func(state *playwrightStorageState) {
		filtered := make([]playwrightOrigin, 0, len(state.Origins))
		for _, o := range state.Origins {
			if o.Origin != origin {
				filtered = append(filtered, o)
			}
		}
		state.Origins = filtered
	})
}

func (s *service) DeleteLocalStorageItem(
	_ context.Context,
	req *connect.Request[recordingsv1.DeleteLocalStorageItemRequest],
) (*connect.Response[recordingsv1.StorageMutationResponse], error) {
	origin := strings.TrimSpace(req.Msg.GetOrigin())
	if origin == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errOriginRequired)
	}
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errNameRequired)
	}
	return s.mutateStorage(req.Msg.GetProfileId(), func(state *playwrightStorageState) {
		for i := range state.Origins {
			if state.Origins[i].Origin != origin {
				continue
			}
			filtered := make([]playwrightLocalStItem, 0, len(state.Origins[i].LocalStorage))
			for _, item := range state.Origins[i].LocalStorage {
				if item.Name != name {
					filtered = append(filtered, item)
				}
			}
			state.Origins[i].LocalStorage = filtered
			if len(filtered) == 0 {
				state.Origins = append(state.Origins[:i], state.Origins[i+1:]...)
			}
			return
		}
	})
}

// mutateStorage loads, modifies, and persists storage state for a profile.
func (s *service) mutateStorage(rawProfileID string, modify func(*playwrightStorageState)) (*connect.Response[recordingsv1.StorageMutationResponse], error) {
	profileID, err := requireProfileID(rawProfileID)
	if err != nil {
		return nil, err
	}
	profile, err := s.deps.Repo.GetProfile(profileID)
	if err != nil {
		return nil, mapStoreError(err)
	}
	var state playwrightStorageState
	if len(profile.StorageState) > 0 {
		if err := json.Unmarshal(profile.StorageState, &state); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	modify(&state)
	newState, err := json.Marshal(state)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if _, err := s.deps.Repo.SaveStorageState(profileID, newState); err != nil {
		return nil, mapStoreError(err)
	}
	return connect.NewResponse(&recordingsv1.StorageMutationResponse{Status: "deleted"}), nil
}

// =============================================================================
// Service workers
// =============================================================================

func (s *service) GetServiceWorkers(
	ctx context.Context,
	req *connect.Request[recordingsv1.GetServiceWorkersRequest],
) (*connect.Response[recordingsv1.GetServiceWorkersResponse], error) {
	rawProfileID := req.Msg.GetProfileId()
	if strings.TrimSpace(rawProfileID) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errProfileIDRequired)
	}
	sessionID := s.deps.Repo.GetSessionForProfile(rawProfileID)
	if sessionID == "" {
		return connect.NewResponse(&recordingsv1.GetServiceWorkersResponse{
			Control: &recordingsv1.ServiceWorkerControl{Mode: "allow"},
			Message: "No active session for this profile",
		}), nil
	}
	swResp, err := s.deps.RecordMode.GetServiceWorkers(ctx, sessionID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&recordingsv1.GetServiceWorkersResponse{
		SessionId: swResp.SessionID,
		Workers:   serviceWorkersToProto(swResp.Workers),
		Control:   serviceWorkerControlToProto(swResp.Control),
		Message:   swResp.Message,
	}), nil
}

func (s *service) ClearAllServiceWorkers(
	ctx context.Context,
	req *connect.Request[recordingsv1.ClearAllServiceWorkersRequest],
) (*connect.Response[recordingsv1.ClearAllServiceWorkersResponse], error) {
	rawProfileID := req.Msg.GetProfileId()
	if strings.TrimSpace(rawProfileID) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errProfileIDRequired)
	}
	sessionID := s.deps.Repo.GetSessionForProfile(rawProfileID)
	if sessionID == "" {
		return connect.NewResponse(&recordingsv1.ClearAllServiceWorkersResponse{
			Message: "No active session for this profile",
		}), nil
	}
	resp, err := s.deps.RecordMode.UnregisterAllServiceWorkers(ctx, sessionID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&recordingsv1.ClearAllServiceWorkersResponse{
		SessionId:         resp.SessionID,
		UnregisteredCount: int32(resp.UnregisteredCount),
		Message:           resp.Message,
	}), nil
}

func (s *service) DeleteServiceWorker(
	ctx context.Context,
	req *connect.Request[recordingsv1.DeleteServiceWorkerRequest],
) (*connect.Response[recordingsv1.DeleteServiceWorkerResponse], error) {
	rawProfileID := req.Msg.GetProfileId()
	if strings.TrimSpace(rawProfileID) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errProfileIDRequired)
	}
	scopeURL := strings.TrimSpace(req.Msg.GetScopeUrl())
	if scopeURL == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errScopeURLRequired)
	}
	sessionID := s.deps.Repo.GetSessionForProfile(rawProfileID)
	if sessionID == "" {
		return nil, connect.NewError(connect.CodeNotFound, errNoActiveSession)
	}
	resp, err := s.deps.RecordMode.UnregisterServiceWorker(ctx, sessionID, scopeURL)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if resp.Error != "" {
		return nil, connect.NewError(connect.CodeNotFound, &stringError{msg: resp.Error})
	}
	return connect.NewResponse(&recordingsv1.DeleteServiceWorkerResponse{
		SessionId:    resp.SessionID,
		Unregistered: resp.Unregistered,
	}), nil
}

// =============================================================================
// History
// =============================================================================

func (s *service) GetHistory(
	_ context.Context,
	req *connect.Request[recordingsv1.GetHistoryRequest],
) (*connect.Response[recordingsv1.GetHistoryResponse], error) {
	profileID, err := requireProfileID(req.Msg.GetProfileId())
	if err != nil {
		return nil, err
	}
	entries, settings, gErr := s.deps.Repo.GetHistoryWithPruning(profileID)
	if gErr != nil {
		return nil, mapStoreError(gErr)
	}
	out := make([]*recordingsv1.HistoryEntry, 0, len(entries))
	for _, e := range entries {
		out = append(out, &recordingsv1.HistoryEntry{
			Id:        e.ID,
			Url:       e.URL,
			Title:     e.Title,
			Timestamp: e.Timestamp,
			Thumbnail: e.Thumbnail,
		})
	}
	stats := &recordingsv1.HistoryStats{TotalEntries: int32(len(entries))}
	if len(entries) > 0 {
		stats.NewestEntry = entries[0].Timestamp
		stats.OldestEntry = entries[len(entries)-1].Timestamp
	}
	return connect.NewResponse(&recordingsv1.GetHistoryResponse{
		Entries:  out,
		Settings: historySettingsToProto(settings),
		Stats:    stats,
	}), nil
}

func (s *service) ClearHistory(
	_ context.Context,
	req *connect.Request[recordingsv1.ClearHistoryRequest],
) (*connect.Response[recordingsv1.HistoryMutationResponse], error) {
	profileID, err := requireProfileID(req.Msg.GetProfileId())
	if err != nil {
		return nil, err
	}
	if _, err := s.deps.Repo.ClearHistory(profileID); err != nil {
		return nil, mapStoreError(err)
	}
	return connect.NewResponse(&recordingsv1.HistoryMutationResponse{Status: "cleared"}), nil
}

func (s *service) DeleteHistoryEntry(
	_ context.Context,
	req *connect.Request[recordingsv1.DeleteHistoryEntryRequest],
) (*connect.Response[recordingsv1.HistoryMutationResponse], error) {
	profileID, err := requireProfileID(req.Msg.GetProfileId())
	if err != nil {
		return nil, err
	}
	entryID := strings.TrimSpace(req.Msg.GetEntryId())
	if entryID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errEntryIDRequired)
	}
	if _, err := s.deps.Repo.DeleteHistoryEntry(profileID, entryID); err != nil {
		return nil, mapStoreError(err)
	}
	return connect.NewResponse(&recordingsv1.HistoryMutationResponse{Status: "deleted", Id: entryID}), nil
}

func (s *service) UpdateHistorySettings(
	_ context.Context,
	req *connect.Request[recordingsv1.UpdateHistorySettingsRequest],
) (*connect.Response[recordingsv1.UpdateHistorySettingsResponse], error) {
	profileID, err := requireProfileID(req.Msg.GetProfileId())
	if err != nil {
		return nil, err
	}
	in := req.Msg.GetSettings()
	if in == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, &stringError{msg: "settings is required"})
	}
	settings := &sessionprofilepersistence.HistorySettings{
		MaxEntries:        int(in.GetMaxEntries()),
		RetentionDays:     int(in.GetRetentionDays()),
		CaptureThumbnails: in.GetCaptureThumbnails(),
	}
	profile, uErr := s.deps.Repo.UpdateHistorySettings(profileID, settings)
	if uErr != nil {
		// Persistence layer validates ranges with "must be between" messages.
		if strings.Contains(uErr.Error(), "must be between") {
			return nil, connect.NewError(connect.CodeInvalidArgument, uErr)
		}
		return nil, mapStoreError(uErr)
	}
	return connect.NewResponse(&recordingsv1.UpdateHistorySettingsResponse{
		Settings:     historySettingsToProto(profile.HistorySettings),
		HistoryCount: int32(len(profile.History)),
	}), nil
}

func (s *service) NavigateToHistoryURL(
	ctx context.Context,
	req *connect.Request[recordingsv1.NavigateToHistoryURLRequest],
) (*connect.Response[recordingsv1.NavigateToHistoryURLResponse], error) {
	rawProfileID := req.Msg.GetProfileId()
	if strings.TrimSpace(rawProfileID) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errProfileIDRequired)
	}
	url := strings.TrimSpace(req.Msg.GetUrl())
	if url == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errURLRequired)
	}
	sessionID := s.deps.Repo.GetSessionForProfile(rawProfileID)
	if sessionID == "" {
		return nil, connect.NewError(connect.CodeNotFound, errNoActiveSession)
	}
	resp, err := s.deps.RecordMode.DriverClient().Navigate(ctx, sessionID, &autodriver.NavigateRequest{
		URL:       url,
		WaitUntil: req.Msg.GetWaitUntil(),
		TimeoutMs: int(req.Msg.GetTimeoutMs()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&recordingsv1.NavigateToHistoryURLResponse{
		Url:          resp.URL,
		Title:        resp.Title,
		CanGoBack:    resp.CanGoBack,
		CanGoForward: resp.CanGoForward,
	}), nil
}

// =============================================================================
// Tabs
// =============================================================================

func (s *service) GetSessionTabs(
	_ context.Context,
	req *connect.Request[recordingsv1.GetSessionTabsRequest],
) (*connect.Response[recordingsv1.GetSessionTabsResponse], error) {
	profileID, err := requireProfileID(req.Msg.GetProfileId())
	if err != nil {
		return nil, err
	}
	profile, gErr := s.deps.Repo.GetProfile(profileID)
	if gErr != nil {
		return nil, mapStoreError(gErr)
	}
	tabs := make([]*recordingsv1.TabInfo, 0, len(profile.OpenTabs))
	for _, t := range profile.OpenTabs {
		tabs = append(tabs, &recordingsv1.TabInfo{
			Url:      t.URL,
			Title:    t.Title,
			IsActive: t.IsActive,
			Order:    int32(t.Order),
		})
	}
	return connect.NewResponse(&recordingsv1.GetSessionTabsResponse{Tabs: tabs}), nil
}

func (s *service) ClearSessionTabs(
	_ context.Context,
	req *connect.Request[recordingsv1.ClearSessionTabsRequest],
) (*connect.Response[recordingsv1.ClearSessionTabsResponse], error) {
	profileID, err := requireProfileID(req.Msg.GetProfileId())
	if err != nil {
		return nil, err
	}
	if _, err := s.deps.Repo.SaveOpenTabs(profileID, nil); err != nil {
		return nil, mapStoreError(err)
	}
	return connect.NewResponse(&recordingsv1.ClearSessionTabsResponse{
		Status:    "cleared",
		ProfileId: string(profileID),
	}), nil
}

func (s *service) DeleteSessionTab(
	_ context.Context,
	req *connect.Request[recordingsv1.DeleteSessionTabRequest],
) (*connect.Response[recordingsv1.DeleteSessionTabResponse], error) {
	profileID, err := requireProfileID(req.Msg.GetProfileId())
	if err != nil {
		return nil, err
	}
	order := int(req.Msg.GetOrder())
	profile, gErr := s.deps.Repo.GetProfile(profileID)
	if gErr != nil {
		return nil, mapStoreError(gErr)
	}
	var found bool
	newTabs := make([]sessionprofilepersistence.TabState, 0, len(profile.OpenTabs))
	for _, tab := range profile.OpenTabs {
		if tab.Order == order {
			found = true
			continue
		}
		newTabs = append(newTabs, tab)
	}
	if !found {
		return nil, connect.NewError(connect.CodeNotFound, errTabNotFound)
	}
	for i := range newTabs {
		newTabs[i].Order = i
	}
	if _, err := s.deps.Repo.SaveOpenTabs(profileID, newTabs); err != nil {
		return nil, mapStoreError(err)
	}
	return connect.NewResponse(&recordingsv1.DeleteSessionTabResponse{
		Status:    "deleted",
		ProfileId: string(profileID),
	}), nil
}

// =============================================================================
// Helpers
// =============================================================================

// stringError lets us return an error with a runtime-built message without
// pulling fmt.Errorf for trivial cases.
type stringError struct{ msg string }

func (e *stringError) Error() string { return e.msg }

func requireProfileID(raw string) (sessionprofilepersistence.ProfileID, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errProfileIDRequired)
	}
	return sessionprofilepersistence.ProfileID(v), nil
}

func mapStoreError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "not found") {
		return connect.NewError(connect.CodeNotFound, errProfileNotFound)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func serviceWorkersToProto(in []autodriver.ServiceWorkerInfo) []*recordingsv1.ServiceWorkerInfo {
	out := make([]*recordingsv1.ServiceWorkerInfo, 0, len(in))
	for _, w := range in {
		out = append(out, &recordingsv1.ServiceWorkerInfo{
			RegistrationId: w.RegistrationID,
			ScopeUrl:       w.ScopeURL,
			ScriptUrl:      w.ScriptURL,
			Status:         w.Status,
			VersionId:      w.VersionID,
		})
	}
	return out
}

func serviceWorkerControlToProto(c autodriver.ServiceWorkerControl) *recordingsv1.ServiceWorkerControl {
	overrides := make([]*recordingsv1.ServiceWorkerDomainOverride, 0, len(c.DomainOverrides))
	for _, o := range c.DomainOverrides {
		overrides = append(overrides, &recordingsv1.ServiceWorkerDomainOverride{
			Domain: o.Domain,
			Mode:   o.Mode,
		})
	}
	return &recordingsv1.ServiceWorkerControl{
		Mode:            c.Mode,
		DomainOverrides: overrides,
		BlockedDomains:  c.BlockedDomains,
	}
}

func historySettingsToProto(s *sessionprofilepersistence.HistorySettings) *recordingsv1.HistorySettings {
	if s == nil {
		return nil
	}
	return &recordingsv1.HistorySettings{
		MaxEntries:        int32(s.MaxEntries),
		RetentionDays:     int32(s.RetentionDays),
		CaptureThumbnails: s.CaptureThumbnails,
	}
}
