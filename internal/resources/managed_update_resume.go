package resources

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
)

// ResumeManagedServiceUpdates reads the resource-state authority, independently
// of today's source manifest or installed resource CLI. One corrupt ledger is
// reported without hiding another operation. Each pass and each owner attempt
// are bounded; passes rotate across resources so an unresolved owner cannot
// starve another operation forever.
func (c *Controller) ResumeManagedServiceUpdates(ctx context.Context) error {
	resolver, err := resourceStorageResolver()
	if err != nil {
		return err
	}
	root, err := resolver.ResourcesRoot(runtimestorage.ClassState, "")
	if err != nil {
		return err
	}
	dir, err := os.Open(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer dir.Close()
	entries, err := dir.ReadDir(513)
	if err != nil && err != io.EOF {
		return err
	}
	if len(entries) > 512 {
		return fmt.Errorf("resource-state inventory exceeds the 512-directory update recovery bound")
	}
	var active []ManagedServiceUpdate
	var failures []error
	var cleanupResources []string
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !managedUpdateResource.MatchString(entry.Name()) || !entry.IsDir() {
			continue
		}
		store, e := newManagedUpdateStore(entry.Name())
		if e != nil {
			failures = append(failures, e)
			continue
		}
		ledger, e := store.read()
		if e != nil {
			failures = append(failures, fmt.Errorf("resource %s update ledger: %w", entry.Name(), e))
			continue
		}
		for _, record := range ledger.Records {
			u := record.State
			if u.Phase.Terminal() {
				if e := c.managedUpdateEffectOwner().release(ctx, u); e != nil {
					failures = append(failures, fmt.Errorf("resource %s terminal capacity release: %w", u.Resource, e))
				}
				continue
			}
			active = append(active, u)
			if e := c.managedUpdateEffectOwner().reserve(ctx, u); e != nil {
				failures = append(failures, fmt.Errorf("resource %s update capacity: %w", u.Resource, e))
			}
		}
		if len(ledger.Records) > 0 && entry.Name() != openRGBResourceName {
			cleanupResources = append(cleanupResources, entry.Name())
		}
	}
	c.updateResumeMu.Lock()
	defer c.updateResumeMu.Unlock()
	sort.Slice(active, func(i, j int) bool { return active[i].Resource < active[j].Resource })
	first := sort.Search(len(active), func(i int) bool { return active[i].Resource > c.updateResumeAfter })
	active = append(active[first:], active[:first]...)
	if len(active) > 4 {
		active = active[:4]
	}
	for _, u := range active {
		c.updateResumeAfter = u.Resource
		operation, cancel := context.WithTimeout(ctx, 45*time.Second)
		_, e := c.AdvanceManagedServiceUpdate(operation, u.Resource, u.RequestID, false)
		cancel()
		if e != nil {
			failures = append(failures, fmt.Errorf("resource %s update %s: %w", u.Resource, u.ID, e))
		}
	}
	// Cleanup is secondary to lease renewal and active recovery. A foreground
	// owner may hold the lifecycle lock; a busy resource must not stall this
	// worker or another resource's continuation.
	sort.Strings(cleanupResources)
	cleanupFirst := sort.SearchStrings(cleanupResources, c.updateCleanupAfter)
	if cleanupFirst < len(cleanupResources) && cleanupResources[cleanupFirst] == c.updateCleanupAfter {
		cleanupFirst++
	}
	cleanupResources = append(cleanupResources[cleanupFirst:], cleanupResources[:cleanupFirst]...)
	if len(cleanupResources) > 4 {
		cleanupResources = cleanupResources[:4]
	}
	for _, resource := range cleanupResources {
		c.updateCleanupAfter = resource
		lockCtx, cancelLock := context.WithTimeout(ctx, 250*time.Millisecond)
		release, err := acquireManagedServiceLifecycleLock(lockCtx, resource)
		cancelLock()
		if err == nil {
			// Give exact native quiescence proofs time to finish after the
			// short lock attempt; busy foreground owners remain cheap to skip.
			cleanup, cancel := context.WithTimeout(ctx, 10*time.Second)
			err = c.pruneManagedServiceUpdateInputsLocked(cleanup, resource)
			cancel()
			release()
		}
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			failures = append(failures, fmt.Errorf("resource %s private update input cleanup: %w", resource, err))
		}
	}
	return errors.Join(failures...)
}
