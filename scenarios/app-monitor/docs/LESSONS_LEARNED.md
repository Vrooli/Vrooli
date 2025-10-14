# Lessons Learned

### App Monitor iOS Preview Bounce (Resolved)

<!-- EMBED:APP_MONITOR_IOS_BOUNCE:START -->
**Status**: Resolved as of 2025-10-13 (`f71e8311c`, `018b53fb7`).

**Root cause**:
- iOS Safari fired a stray `popstate` about 2–3 seconds after entering the preview, before hydration finished. The router interpreted this as a user back navigation and returned to the apps list.
- Our preview mount mutated history during guard setup, inadvertently leaving the router with a stale entry that iOS reused.

**Telemetry breakthroughs**:
- Added `history-*` and `preview-*` beacons (`ui/src/main.tsx`) to capture push/replace/pop flows with timestamps, confirming the pop originated client-side.
- Logged guard state transitions so we could see the premature bounce without losing user context.

**Fix**:
- Introduced a preview guard on window history that suppresses unexpected `popstate` events while the preview hydrates and immediately replays the correct state (`ui/src/main.tsx`, `AppPreviewView.tsx`).
- Guard tracks TTL, originating route, and recovery state so once hydration completes the normal back button still works.
- Added complementary layout hooks to re-arm the guard only when navigating from the apps list and to disarm once the user interacts, ensuring we do not mask legitimate navigation (`Layout.tsx`, `AppsView.tsx`).

**Outcome**:
- iOS preview no longer bounces back to the app list during load, while manual back gestures still behave as expected.
- The telemetry remains in place to alert us if Safari regresses or similar flows appear on other platforms.
<!-- EMBED:APP_MONITOR_IOS_BOUNCE:END -->
