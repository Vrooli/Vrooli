package recordings

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"connectrpc.com/connect"

	autodriver "github.com/vrooli/browser-automation-studio/automation/driver"
	autosession "github.com/vrooli/browser-automation-studio/automation/session"
	sessionprofilepersistence "github.com/vrooli/browser-automation-studio/services/session-profile/persistence"
	recordingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/recordings"
)

type service struct {
	deps Deps
}

// =============================================================================
// Storage state — playwright storage_state shape used for load/save.
// =============================================================================

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
	return s.mutateStorage(req.Msg.GetProfileId(), "cookies", func(json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`[]`), nil
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
	return s.deleteCookies(req.Msg.GetProfileId(), domain, "")
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
	return s.deleteCookies(req.Msg.GetProfileId(), domain, name)
}

func (s *service) ClearAllLocalStorage(
	_ context.Context,
	req *connect.Request[recordingsv1.ClearAllLocalStorageRequest],
) (*connect.Response[recordingsv1.StorageMutationResponse], error) {
	return s.deleteLocalStorage(req.Msg.GetProfileId(), "", "")
}

func (s *service) DeleteLocalStorageByOrigin(
	_ context.Context,
	req *connect.Request[recordingsv1.DeleteLocalStorageByOriginRequest],
) (*connect.Response[recordingsv1.StorageMutationResponse], error) {
	origin := strings.TrimSpace(req.Msg.GetOrigin())
	if origin == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errOriginRequired)
	}
	return s.deleteLocalStorage(req.Msg.GetProfileId(), origin, "")
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
	return s.deleteLocalStorage(req.Msg.GetProfileId(), origin, name)
}

// mutateStorage replaces only the selected top-level field. Other fields never
// pass through a partial schema or floating-point JSON representation.
func (s *service) mutateStorage(rawProfileID, field string, modify func(json.RawMessage) (json.RawMessage, error)) (*connect.Response[recordingsv1.StorageMutationResponse], error) {
	profileID, err := requireProfileID(rawProfileID)
	if err != nil {
		return nil, err
	}
	_, err = s.deps.Repo.UpdateStorageState(profileID, func(current json.RawMessage) (json.RawMessage, error) {
		state := make(map[string]json.RawMessage)
		if len(current) > 0 {
			if err := json.Unmarshal(current, &state); err != nil {
				return nil, err
			}
		}
		if state == nil {
			state = make(map[string]json.RawMessage)
		}
		changed, err := modify(state[field])
		if err != nil {
			return nil, err
		}
		state[field] = changed
		return json.Marshal(state)
	})
	if err != nil {
		return nil, mapStoreError(err)
	}
	return connect.NewResponse(&recordingsv1.StorageMutationResponse{Status: "deleted"}), nil
}

func (s *service) deleteCookies(profileID, domain, name string) (*connect.Response[recordingsv1.StorageMutationResponse], error) {
	return s.mutateStorage(profileID, "cookies", func(raw json.RawMessage) (json.RawMessage, error) {
		cookies, err := filterStorageItems(raw, func(cookie struct {
			Name   string `json:"name"`
			Domain string `json:"domain"`
		},
		) bool {
			return cookie.Domain == domain && (name == "" || cookie.Name == name)
		})
		if err != nil {
			return nil, err
		}
		return json.Marshal(cookies)
	})
}

func (s *service) deleteLocalStorage(profileID, origin, name string) (*connect.Response[recordingsv1.StorageMutationResponse], error) {
	return s.mutateStorage(profileID, "origins", func(raw json.RawMessage) (json.RawMessage, error) {
		var origins []json.RawMessage
		if len(raw) > 0 {
			if err := json.Unmarshal(raw, &origins); err != nil {
				return nil, err
			}
		}
		kept := make([]json.RawMessage, 0, len(origins))
		for _, rawOrigin := range origins {
			changed, err := editOriginLocalStorage(rawOrigin, origin, name)
			if err != nil {
				return nil, err
			}
			if changed != nil {
				kept = append(kept, changed)
			}
		}
		return json.Marshal(kept)
	})
}

// editOriginLocalStorage retains all non-localStorage fields. An origin carrying
// opaque state must survive removal of its final localStorage item.
func editOriginLocalStorage(raw json.RawMessage, selectedOrigin, selectedName string) (json.RawMessage, error) {
	var origin map[string]json.RawMessage
	if err := json.Unmarshal(raw, &origin); err != nil {
		return nil, err
	}
	var address string
	if err := json.Unmarshal(origin["origin"], &address); err != nil {
		return nil, err
	}
	if selectedOrigin != "" && selectedOrigin != address {
		return raw, nil
	}
	if _, exists := origin["localStorage"]; !exists {
		return raw, nil
	}
	items, err := filterStorageItems(origin["localStorage"], func(item struct {
		Name string `json:"name"`
	},
	) bool {
		return selectedName == "" || item.Name == selectedName
	})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 && len(origin) <= 2 {
		return nil, nil
	}
	origin["localStorage"], err = json.Marshal(items)
	if err != nil {
		return nil, err
	}
	return json.Marshal(origin)
}

func filterStorageItems[T any](raw json.RawMessage, remove func(T) bool) ([]json.RawMessage, error) {
	var items []json.RawMessage
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, err
		}
	}
	kept := make([]json.RawMessage, 0, len(items))
	for _, item := range items {
		var identity *T
		if err := json.Unmarshal(item, &identity); err != nil {
			return nil, err
		}
		if identity == nil {
			return nil, errStorageObjectRequired
		}
		if !remove(*identity) {
			kept = append(kept, item)
		}
	}
	return kept, nil
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
	owner, ok := s.deps.RecordMode.GetSession(sessionID)
	if !ok || owner == nil {
		return nil, connect.NewError(connect.CodeNotFound, errNoActiveSession)
	}
	resp, err := owner.Navigate(ctx, url, autosession.WithWaitUntil(req.Msg.GetWaitUntil()), autosession.WithNavigateTimeout(int(req.Msg.GetTimeoutMs())))
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
	_, err = s.deps.Repo.UpdateProfile(profileID, func(profile *sessionprofilepersistence.SessionProfile) error {
		found := false
		tabs := make([]sessionprofilepersistence.TabState, 0, len(profile.OpenTabs))
		for _, tab := range profile.OpenTabs {
			if tab.Order == order {
				found = true
				continue
			}
			tab.Order = len(tabs)
			tabs = append(tabs, tab)
		}
		if !found {
			return errTabNotFound
		}
		profile.OpenTabs = tabs
		return nil
	})
	if err != nil {
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
	if errors.Is(err, sessionprofilepersistence.ErrInvalidProfileID) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if errors.Is(err, errTabNotFound) {
		return connect.NewError(connect.CodeNotFound, errTabNotFound)
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
