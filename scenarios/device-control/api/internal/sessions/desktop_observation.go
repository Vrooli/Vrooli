package sessions

import (
	"context"
	"image"
	"math"
	"time"

	"github.com/google/uuid"
)

// DesktopObservation is ephemeral native evidence, never part of the receipt
// journal. Consumers receive pixels only after current observe authorization.
type DesktopObservation struct {
	Image                       *image.RGBA
	DisplayID, GeometryRevision string
	CapturedAt                  time.Time
	Semantic                    *DesktopSemanticObservation
}

type DesktopSemanticElement struct {
	ID, Name           string
	ParentID, WindowID string
	Role               uint32
	Editable           bool
}
type DesktopSemanticObservation struct {
	Revision  string
	ExpiresAt time.Time
	ProcessID uint32
	Elements  []DesktopSemanticElement
}
type DesktopProcessObserver interface {
	ObserveProcess(context.Context, uint32, string) (DesktopObservation, error)
}

// Application identities are ephemeral helper-owned references, not PIDs or
// names to resolve again. Catalogs and selections never enter durable state.
type DesktopApplication struct {
	ID, Name  string
	ProcessID uint32
}
type DesktopApplications struct {
	Revision     string
	ExpiresAt    time.Time
	Applications []DesktopApplication
}
type DesktopApplicationObserver interface {
	Applications(context.Context, string) (DesktopApplications, error)
	ObserveApplication(context.Context, string, string, string) (DesktopObservation, error)
}

type DesktopObserver interface {
	Observe(context.Context) (DesktopObservation, error)
}

func (c *DesktopController) Observe(ctx context.Context, lease DesktopLease) (DesktopObservation, error) {
	return c.ObserveProcess(ctx, lease, 0)
}

func (c *DesktopController) ObserveProcess(ctx context.Context, lease DesktopLease, processID uint32) (DesktopObservation, error) {
	return c.observe(ctx, lease, processID, "", "")
}

func (c *DesktopController) ObserveApplication(ctx context.Context, lease DesktopLease, applicationID, revision string) (DesktopObservation, error) {
	if applicationID == "" || len(applicationID) > 128 || revision == "" || len(revision) > 128 {
		return DesktopObservation{}, ErrDesktopAdmission
	}
	return c.observe(ctx, lease, 0, applicationID, revision)
}

func (c *DesktopController) observe(ctx context.Context, lease DesktopLease, processID uint32, applicationID, revision string) (DesktopObservation, error) {
	var snapshot DesktopObservation
	if !c.valid(lease) || c.authority.Authorize(ctx, lease, "observe") != nil {
		return snapshot, ErrDesktopAdmission
	}
	observer, ok := c.native.(DesktopObserver)
	if !ok {
		return snapshot, ErrDesktopAdmission
	}
	ctx, cancel := c.nativeContext(ctx, lease)
	defer cancel()
	// Capture is serialized with takeover, expiry and input. Recheck admission
	// after capture so a mid-capture expiry/revocation cannot release pixels.
	err := c.repo.Update(ctx, func(s *DesktopState) error {
		if !c.admitted(s, lease) {
			return ErrDesktopAdmission
		}
		var err error
		if applicationID != "" {
			semantic, ok := c.native.(DesktopApplicationObserver)
			if !ok {
				return ErrDesktopAdmission
			}
			snapshot, err = semantic.ObserveApplication(ctx, applicationID, revision, lease.Ref.SessionID)
		} else if processID != 0 {
			semantic, ok := c.native.(DesktopProcessObserver)
			if !ok {
				return ErrDesktopAdmission
			}
			snapshot, err = semantic.ObserveProcess(ctx, processID, lease.Ref.SessionID)
		} else {
			snapshot, err = observer.Observe(ctx)
		}
		if err != nil {
			return err
		}
		if !c.admitted(s, lease) || c.authority.Authorize(ctx, lease, "observe") != nil {
			return ErrDesktopAdmission
		}
		if snapshot.Image == nil || snapshot.Image.Bounds().Dx() <= 0 || snapshot.Image.Bounds().Dy() <= 0 || int64(snapshot.Image.Bounds().Dx())*int64(snapshot.Image.Bounds().Dy()) > 16*1024*1024 || snapshot.DisplayID == "" || snapshot.GeometryRevision == "" || snapshot.CapturedAt.IsZero() || snapshot.CapturedAt.After(c.now()) {
			return ErrDesktopAdmission
		}
		if processID != 0 || applicationID != "" {
			semantic := snapshot.Semantic
			if semantic == nil || semantic.ProcessID == 0 || (processID != 0 && semantic.ProcessID != processID) || semantic.Revision == "" || len(semantic.Revision) > 128 || !c.now().Before(semantic.ExpiresAt) || semantic.ExpiresAt.After(c.now().Add(5*time.Second)) || len(semantic.Elements) > 128 {
				return ErrDesktopAdmission
			}
			seen := map[string]bool{}
			parents := map[string]string{}
			for _, element := range semantic.Elements {
				if element.ID == "" || len(element.ID) > 128 || len(element.Name) > 4096 || seen[element.ID] {
					return ErrDesktopAdmission
				}
				if element.ParentID != "" && !seen[element.ParentID] {
					return ErrDesktopAdmission
				}
				if element.WindowID != "" && element.WindowID != element.ID {
					ancestor := element.ParentID
					for ancestor != "" && ancestor != element.WindowID {
						ancestor = parents[ancestor]
					}
					if ancestor == "" {
						return ErrDesktopAdmission
					}
				}
				parents[element.ID] = element.ParentID
				seen[element.ID] = true
			}
		} else if snapshot.Semantic != nil {
			return ErrDesktopAdmission
		}
		return nil
	})
	if err != nil {
		return DesktopObservation{}, err
	}
	return snapshot, nil
}

func (c *DesktopController) Applications(ctx context.Context, lease DesktopLease) (DesktopApplications, error) {
	var catalog DesktopApplications
	if !c.valid(lease) || c.authority.Authorize(ctx, lease, "observe") != nil {
		return catalog, ErrDesktopAdmission
	}
	observer, ok := c.native.(DesktopApplicationObserver)
	if !ok {
		return catalog, ErrDesktopAdmission
	}
	ctx, cancel := c.nativeContext(ctx, lease)
	defer cancel()
	err := c.repo.Update(ctx, func(s *DesktopState) error {
		if !c.admitted(s, lease) {
			return ErrDesktopAdmission
		}
		var err error
		catalog, err = observer.Applications(ctx, lease.Ref.SessionID)
		if err != nil {
			return err
		}
		if !c.admitted(s, lease) || c.authority.Authorize(ctx, lease, "observe") != nil {
			return ErrDesktopAdmission
		}
		if catalog.Revision == "" || len(catalog.Revision) > 128 || !c.now().Before(catalog.ExpiresAt) || catalog.ExpiresAt.After(c.now().Add(30*time.Second)) || len(catalog.Applications) > 128 {
			return ErrDesktopAdmission
		}
		seen := map[string]bool{}
		bytes := 0
		for _, app := range catalog.Applications {
			bytes += len(app.Name)
			if app.ID == "" || len(app.ID) > 128 || app.ProcessID == 0 || len(app.Name) > 4096 || bytes > 64*1024 || seen[app.ID] {
				return ErrDesktopAdmission
			}
			seen[app.ID] = true
		}
		return nil
	})
	if err != nil {
		return DesktopApplications{}, err
	}
	return catalog, nil
}

// DesktopSelector matches exact names within one observed window. A missing or
// stale observation is an error, distinct from a valid observation with no match.
type DesktopSelector struct {
	Revision, WindowID, Name string
	EditableOnly             bool
}
type DesktopResolution struct {
	Disposition                string
	Revision, GeometryRevision string
	ExpiresAt                  time.Time
	ElementIDs                 []string
}
type DesktopSemanticResolver interface {
	Resolve(context.Context, DesktopSelector, string) (DesktopResolution, error)
}

func (c *DesktopController) Resolve(ctx context.Context, lease DesktopLease, selector DesktopSelector) (DesktopResolution, error) {
	var result DesktopResolution
	if selector.Revision == "" || len(selector.Revision) > 128 || selector.WindowID == "" || len(selector.WindowID) > 128 || selector.Name == "" || len(selector.Name) > 4096 || !c.valid(lease) || c.authority.Authorize(ctx, lease, "observe") != nil {
		return result, ErrDesktopAdmission
	}
	resolver, ok := c.native.(DesktopSemanticResolver)
	if !ok {
		return result, ErrDesktopAdmission
	}
	ctx, cancel := c.nativeContext(ctx, lease)
	defer cancel()
	err := c.repo.Update(ctx, func(s *DesktopState) error {
		if !c.admitted(s, lease) {
			return ErrDesktopAdmission
		}
		var err error
		result, err = resolver.Resolve(ctx, selector, lease.Ref.SessionID)
		if err != nil {
			return err
		}
		if !c.admitted(s, lease) || c.authority.Authorize(ctx, lease, "observe") != nil {
			return ErrDesktopAdmission
		}
		if result.Revision != selector.Revision || result.GeometryRevision == "" || !c.now().Before(result.ExpiresAt) || result.ExpiresAt.After(c.now().Add(5*time.Second)) || len(result.ElementIDs) > 128 {
			return ErrDesktopAdmission
		}
		seen := map[string]bool{}
		for _, id := range result.ElementIDs {
			if id == "" || len(id) > 128 || seen[id] {
				return ErrDesktopAdmission
			}
			seen[id] = true
		}
		switch result.Disposition {
		case "absent":
			if len(result.ElementIDs) != 0 {
				return ErrDesktopAdmission
			}
		case "unique":
			if len(result.ElementIDs) != 1 {
				return ErrDesktopAdmission
			}
		case "ambiguous":
			if len(result.ElementIDs) < 2 {
				return ErrDesktopAdmission
			}
		default:
			return ErrDesktopAdmission
		}
		return nil
	})
	if err != nil {
		return DesktopResolution{}, err
	}
	return result, nil
}

// DesktopActivationContext is ephemeral native metadata. Native window identities
// stay within the helper; they are not authority or stable application references.
// PointerWindow may identify decoration rather than a uniquely resolved app.
// DesktopBounds describes client-content pixels in the captured root coordinate
// frame. It is evidence, never permission to act at a future coordinate.
type DesktopBounds struct {
	X, Y          int32
	Width, Height uint32
}

func (b DesktopBounds) Valid() bool {
	return b.Width > 0 && b.Height > 0 && b.Width <= 65535 && b.Height <= 65535 && int64(b.X)+int64(b.Width) <= math.MaxInt32 && int64(b.Y)+int64(b.Height) <= math.MaxInt32
}

type DesktopActivationContext struct {
	SourceBounds                DesktopBounds
	ActiveWindow, PointerWindow uint64
	ProcessID                   uint32
	PointerX, PointerY          int32
	DisplayID, GeometryRevision string
	CapturedAt                  time.Time
}

// DesktopActivationImage is helper-local pixel evidence, captured in the same
// bounded native operation as its source metadata. It is never a receipt field.
type DesktopActivationImage struct {
	Context DesktopActivationContext
	Image   *image.RGBA
}

type DesktopActivationImageObserver interface {
	CaptureActivationImage(context.Context) (DesktopActivationImage, error)
}

type DesktopActivationObserver interface {
	CaptureActivation(context.Context) (DesktopActivationContext, error)
}

// CaptureActivation uses the same observation grant and exclusion as screenshots.
// It never persists context in the receipt journal or grants input authority.
func (c *DesktopController) CaptureActivation(ctx context.Context, lease DesktopLease) (DesktopActivationContext, error) {
	return c.captureActivation(ctx, lease, 0, 0)
}

type DesktopWindowVerifier interface {
	VerifyWindowProcess(context.Context, uint64, uint32) error
}

func (c *DesktopController) captureActivation(ctx context.Context, lease DesktopLease, window uint64, pid uint32) (DesktopActivationContext, error) {
	evidence, err := c.captureActivationEvidence(ctx, lease, window, pid, false)
	return evidence.Context, err
}

// cloneActivationPixels validates the native image before allocation. Copies at
// both cache boundaries keep backend and consumer mutations out of frozen context.
func cloneActivationPixels(pixels *image.RGBA, bounds DesktopBounds) (*image.RGBA, error) {
	if pixels == nil || !bounds.Valid() || uint64(bounds.Width)*uint64(bounds.Height) > 16*1024*1024 || pixels.Rect != image.Rect(0, 0, int(bounds.Width), int(bounds.Height)) || pixels.Stride < int(bounds.Width)*4 {
		return nil, ErrDesktopAdmission
	}
	required := uint64(bounds.Height-1)*uint64(pixels.Stride) + uint64(bounds.Width)*4
	if required > uint64(len(pixels.Pix)) {
		return nil, ErrDesktopAdmission
	}
	result := image.NewRGBA(pixels.Rect)
	for y := 0; y < int(bounds.Height); y++ {
		copy(result.Pix[y*result.Stride:(y+1)*result.Stride], pixels.Pix[y*pixels.Stride:y*pixels.Stride+result.Stride])
	}
	return result, nil
}

func (c *DesktopController) captureActivationEvidence(ctx context.Context, lease DesktopLease, window uint64, pid uint32, includeImage bool) (DesktopActivationImage, error) {
	var result DesktopActivationContext
	var pixels *image.RGBA
	if !c.valid(lease) || c.authority.Authorize(ctx, lease, "observe") != nil {
		return DesktopActivationImage{}, ErrDesktopAdmission
	}
	observer, ok := c.native.(DesktopActivationObserver)
	imageObserver, imageOK := c.native.(DesktopActivationImageObserver)
	if (!includeImage && !ok) || (includeImage && !imageOK) {
		return DesktopActivationImage{}, ErrDesktopAdmission
	}
	ctx, deadline := context.WithTimeout(ctx, 2*time.Second)
	defer deadline()
	ctx, cancel := c.nativeContext(ctx, lease)
	defer cancel()
	err := c.repo.Update(ctx, func(s *DesktopState) error {
		if !c.admitted(s, lease) || c.authority.Authorize(ctx, lease, "observe") != nil {
			return ErrDesktopAdmission
		}
		var err error
		if window != 0 {
			verifier, ok := c.native.(DesktopWindowVerifier)
			if !ok || pid == 0 || verifier.VerifyWindowProcess(ctx, window, pid) != nil {
				return ErrDesktopAdmission
			}
		}
		if includeImage {
			evidence, captureErr := imageObserver.CaptureActivationImage(ctx)
			result, pixels, err = evidence.Context, evidence.Image, captureErr
		} else {
			result, err = observer.CaptureActivation(ctx)
		}
		if window != 0 {
			verifier := c.native.(DesktopWindowVerifier)
			if verifier.VerifyWindowProcess(ctx, window, pid) != nil {
				return ErrDesktopAdmission
			}
		}

		if err != nil {
			return err
		}
		if ctx.Err() != nil || !c.admitted(s, lease) || c.authority.Authorize(ctx, lease, "observe") != nil {
			return ErrDesktopAdmission
		}
		now := c.now()
		if !result.SourceBounds.Valid() || result.ActiveWindow == 0 || result.ProcessID == 0 || result.DisplayID == "" || len(result.DisplayID) > 128 || result.GeometryRevision == "" || len(result.GeometryRevision) > 128 || result.CapturedAt.IsZero() || result.CapturedAt.After(now) || now.Sub(result.CapturedAt) > 2*time.Second {
			return ErrDesktopAdmission
		}
		if includeImage {
			pixels, err = cloneActivationPixels(pixels, result.SourceBounds)
			if err != nil {
				return err
			}
			if ctx.Err() != nil || !c.admitted(s, lease) || c.authority.Authorize(ctx, lease, "observe") != nil {
				return ErrDesktopAdmission
			}
		}
		return nil
	})
	if err != nil {
		return DesktopActivationImage{}, err
	}
	return DesktopActivationImage{Context: result, Image: pixels}, nil
}

// DesktopActivationReference contains no OS window identity or process ID. Its
// opaque ID only locates helper evidence; every read still requires observation
// authority for the original exact lease.
type DesktopActivationReference struct {
	HasImage                    bool
	SourceBounds                DesktopBounds
	ID                          string
	DisplayID, GeometryRevision string
	PointerX, PointerY          int32
	CapturedAt, ExpiresAt       time.Time
}
type desktopActivationEntry struct {
	image   *image.RGBA
	lease   DesktopLease
	context DesktopActivationContext
	ref     DesktopActivationReference
}

func (c *DesktopController) CaptureActivationReference(ctx context.Context, lease DesktopLease) (DesktopActivationReference, error) {
	return c.captureActivationReference(ctx, lease, 0, 0, false)
}

func (c *DesktopController) CaptureCompanionActivation(ctx context.Context, lease DesktopLease, window uint64, pid uint32) (DesktopActivationReference, error) {
	if window == 0 || pid == 0 {
		return DesktopActivationReference{}, ErrDesktopAdmission
	}
	return c.captureActivationReference(ctx, lease, window, pid, false)
}

// CaptureCompanionActivationImage is explicit pixel opt-in; ordinary companion
// activation remains metadata-only. Window/PID binding is checked in both paths.
func (c *DesktopController) CaptureCompanionActivationImage(ctx context.Context, lease DesktopLease, window uint64, pid uint32) (DesktopActivationReference, error) {
	if window == 0 || pid == 0 {
		return DesktopActivationReference{}, ErrDesktopAdmission
	}
	return c.captureActivationReference(ctx, lease, window, pid, true)
}

func (c *DesktopController) captureActivationReference(ctx context.Context, lease DesktopLease, window uint64, pid uint32, includeImage bool) (DesktopActivationReference, error) {
	evidence, err := c.captureActivationEvidence(ctx, lease, window, pid, includeImage)
	captured := evidence.Context
	if err != nil {
		return DesktopActivationReference{}, err
	}
	var ref DesktopActivationReference
	err = c.repo.Update(ctx, func(s *DesktopState) error {
		if ctx.Err() != nil || !c.admitted(s, lease) || c.authority.Authorize(ctx, lease, "observe") != nil {
			return ErrDesktopAdmission
		}
		now := c.now()
		expires := captured.CapturedAt.Add(30 * time.Second)
		if lease.ExpiresAt.Before(expires) {
			expires = lease.ExpiresAt
		}
		if !now.Before(expires) {
			return ErrDesktopAdmission
		}
		c.activationMu.Lock()
		defer c.activationMu.Unlock()
		// At most one context per destination. A delayed older capture cannot replace
		// newer evidence. New capture invalidates the previous opaque reference.
		if c.activation != nil && c.activation.context.CapturedAt.After(captured.CapturedAt) {
			return ErrDesktopAdmission
		}
		ref = DesktopActivationReference{HasImage: evidence.Image != nil, SourceBounds: captured.SourceBounds, ID: uuid.NewString(), DisplayID: captured.DisplayID, GeometryRevision: captured.GeometryRevision, PointerX: captured.PointerX, PointerY: captured.PointerY, CapturedAt: captured.CapturedAt, ExpiresAt: expires}
		c.activation = &desktopActivationEntry{image: evidence.Image, lease: lease, context: captured, ref: ref}
		return nil
	})
	if err != nil {
		return DesktopActivationReference{}, err
	}
	return ref, nil
}

func (c *DesktopController) ReadActivation(ctx context.Context, lease DesktopLease, id string) (DesktopActivationReference, error) {
	ref, _, err := c.readActivation(ctx, lease, id, false)
	return ref, err
}

// ReadActivationImage returns a caller-owned copy under current observation
// authority. It never re-captures pixels or extends the reference lifetime.
func (c *DesktopController) ReadActivationImage(ctx context.Context, lease DesktopLease, id string) (DesktopActivationReference, *image.RGBA, error) {
	return c.readActivation(ctx, lease, id, true)
}

func (c *DesktopController) readActivation(ctx context.Context, lease DesktopLease, id string, includeImage bool) (DesktopActivationReference, *image.RGBA, error) {
	var pixels *image.RGBA
	var result DesktopActivationReference
	if _, err := uuid.Parse(id); err != nil || len(id) != 36 {
		return result, nil, ErrDesktopAdmission
	}
	if !c.valid(lease) || c.authority.Authorize(ctx, lease, "observe") != nil {
		return result, nil, ErrDesktopAdmission
	}
	err := c.repo.Update(ctx, func(s *DesktopState) error {
		if ctx.Err() != nil || !c.admitted(s, lease) || c.authority.Authorize(ctx, lease, "observe") != nil {
			return ErrDesktopAdmission
		}
		c.activationMu.Lock()
		defer c.activationMu.Unlock()
		entry := c.activation
		if entry == nil {
			return ErrDesktopAdmission
		}
		if !c.now().Before(entry.ref.ExpiresAt) {
			c.activation = nil
			return ErrDesktopAdmission
		}
		if !sameDesktopLease(entry.lease, lease) || entry.ref.ID != id {
			return ErrDesktopAdmission
		}
		if includeImage {
			var err error
			pixels, err = cloneActivationPixels(entry.image, entry.ref.SourceBounds)
			if err != nil {
				return err
			}
			if ctx.Err() != nil || !c.now().Before(entry.ref.ExpiresAt) || !c.admitted(s, lease) || c.authority.Authorize(ctx, lease, "observe") != nil {
				return ErrDesktopAdmission
			}
		}
		result = entry.ref
		return nil
	})
	if err != nil {
		return DesktopActivationReference{}, nil, err
	}
	return result, pixels, nil
}

// pruneActivation runs under repository exclusion, like capture publication.
// Lifecycle cleanup must erase expired native identities even without a read.
// It never changes lease authority or invokes native input cleanup.
func (c *DesktopController) pruneActivation(s *DesktopState, force bool) {
	c.activationMu.Lock()
	defer c.activationMu.Unlock()
	entry := c.activation
	if entry == nil {
		return
	}
	_, admitted := c.admittedLease(s, entry.lease)
	if force || !admitted || !c.now().Before(entry.ref.ExpiresAt) {
		c.activation = nil
	}
}

// DeleteActivation erases only the exact reference in the admitted lease. An
// absent or superseded ID is already deleted; it must never erase its successor.
func (c *DesktopController) DeleteActivation(ctx context.Context, lease DesktopLease, id string) error {
	parsed, err := uuid.Parse(id)
	if err != nil || parsed == uuid.Nil || parsed.String() != id || !c.valid(lease) || c.authority.Authorize(ctx, lease, "observe") != nil {
		return ErrDesktopAdmission
	}
	return c.repo.Update(ctx, func(s *DesktopState) error {
		if ctx.Err() != nil || !c.admitted(s, lease) || c.authority.Authorize(ctx, lease, "observe") != nil {
			return ErrDesktopAdmission
		}
		c.activationMu.Lock()
		defer c.activationMu.Unlock()
		if c.activation != nil && sameDesktopLease(c.activation.lease, lease) && c.activation.ref.ID == id {
			c.activation = nil
		}
		return nil
	})
}
