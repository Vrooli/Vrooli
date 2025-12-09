# Mobile Deployment Workflow (Tier 3)

Deploy a Vrooli scenario as an iOS or Android mobile application.

## Implementation Status

| Platform | Status | Packager |
|----------|--------|----------|
| iOS | Not Started | `scenario-to-ios` (planned) |
| Android | Not Started | `scenario-to-android` (planned) |

> **Current Reality**: Mobile deployment is in the planning stage. This document outlines the intended workflow for when implementation begins.

---

## Planned Workflow Overview

Mobile deployment follows the same 8-phase structure as [Desktop Deployment](desktop-deployment.md), with mobile-specific considerations:

### Phase 1: Check Compatibility

```bash
# Check mobile fitness (tier 3)
deployment-manager fitness <scenario-name> --tier 3
# OR
deployment-manager fitness <scenario-name> --tier mobile
deployment-manager fitness <scenario-name> --tier ios
deployment-manager fitness <scenario-name> --tier android
```

### Phase 2: Address Blocking Issues

Mobile has stricter requirements than desktop:

| Blocker | Desktop Solution | Mobile Solution |
|---------|------------------|-----------------|
| PostgreSQL | SQLite | SQLite or cloud API |
| Redis | In-process cache | In-process or cloud API |
| Ollama | Bundled models | Cloud inference API (OpenRouter) |
| browserless | Playwright driver | Not applicable |
| Heavy binaries | Cross-compile | Cloud API proxies |

**Expected swap impact:**
- postgres → sqlite: +30-40 fitness points
- ollama → openrouter: +20-30 fitness points
- Local compute → cloud APIs: Variable based on use case

### Phase 3: Create Deployment Profile

```bash
# Create iOS profile
deployment-manager profile create <name> <scenario> --tier ios

# Create Android profile
deployment-manager profile create <name> <scenario> --tier android

# Or target both
deployment-manager profile create <name> <scenario> --tier 3
```

### Phase 4: Configure Secrets

Mobile apps have additional secret considerations:

| Secret Type | iOS | Android |
|-------------|-----|---------|
| API keys | Keychain | EncryptedSharedPreferences |
| User tokens | Secure Enclave | Android Keystore |
| Generated secrets | On first launch | On first launch |

### Phase 5: Generate Bundle Manifest

The mobile manifest extends the desktop schema with:

```json
{
  "schema_version": "v0.1",
  "target": "mobile",
  "platforms": ["ios", "android"],
  "app": {
    "name": "picker-wheel",
    "bundle_id": "com.vrooli.pickerwheel",
    "version": "1.0.0"
  },
  "capabilities": {
    "camera": false,
    "location": false,
    "notifications": true
  }
}
```

### Phase 6: Build Mobile Apps

**iOS (planned):**
```bash
# Requires macOS with Xcode
deployment-manager package <profile-id> --packager scenario-to-ios
```

**Android (planned):**
```bash
# Requires Android SDK
deployment-manager package <profile-id> --packager scenario-to-android
```

### Phase 7: Distribute

| Platform | Distribution Method |
|----------|---------------------|
| iOS | App Store, TestFlight, Enterprise distribution |
| Android | Play Store, APK sideload, Enterprise distribution |

### Phase 8: Monitor

Mobile telemetry includes:
- App launches and session duration
- Crash reports
- Performance metrics (startup time, memory usage)
- Update adoption rates

---

## Technical Requirements (Planned)

### iOS Requirements

- macOS development machine
- Xcode 15+
- Apple Developer account ($99/year)
- Code signing certificates and provisioning profiles
- React Native or similar cross-platform framework

### Android Requirements

- Android SDK (any OS)
- Java 17+
- Google Play Developer account ($25 one-time)
- Signing keystore
- React Native or similar cross-platform framework

### Shared Requirements

- Cloud API proxy for heavy compute (Ollama → OpenRouter)
- SQLite for local storage
- Secure secret storage using platform APIs
- Push notification service integration

---

## Architecture Decisions (Pending)

### Framework Choice

| Option | Pros | Cons |
|--------|------|------|
| React Native | Reuse React UI code | Bridge overhead |
| Flutter | High performance | Dart rewrite needed |
| Capacitor | Web-first, minimal changes | Limited native access |
| Native (Swift/Kotlin) | Best performance | Duplicate UI code |

**Recommendation**: Capacitor for scenarios with simple UIs; React Native for complex interactions.

### API Strategy

| Approach | Use Case |
|----------|----------|
| Bundled API | Simple, offline-first apps |
| Cloud proxy | Heavy compute, always-connected |
| Hybrid | Offline cache + cloud sync |

---

## Fitness Scoring for Mobile

Mobile baseline fitness is lower than desktop due to stricter constraints:

| Tier | Baseline | Notes |
|------|----------|-------|
| Desktop (2) | 75/100 | Lightweight deps required |
| Mobile (3) | 40/100 | Heavy swaps often needed |

### Common Blockers

1. **Database size** - SQLite DBs over 50MB are problematic
2. **Binary size** - App store limits (200MB iOS OTA, 150MB Android)
3. **Compute requirements** - No GPU, limited CPU
4. **Network dependency** - Offline capability expectations
5. **Platform APIs** - Some resources have no mobile equivalent

---

## Roadmap

1. **Define mobile manifest schema** - Extend v0.1 with mobile-specific fields
2. **Build scenario-to-ios** - Capacitor/React Native wrapper generator
3. **Build scenario-to-android** - Android-specific packaging
4. **Integrate with deployment-manager** - Fitness scoring, swap suggestions
5. **Add mobile telemetry** - Crash reporting, analytics
6. **Document store submission** - App Store, Play Store requirements

---

## Related Documentation

- [Tier 3 Mobile Reference](../tiers/tier-3-mobile.md)
- [Desktop Deployment Workflow](desktop-deployment.md) - Similar workflow structure
- [Dependency Swapping Guide](../guides/dependency-swapping.md)
- [Fitness Scoring Guide](../guides/fitness-scoring.md)

---

## Contributing

Mobile deployment is a significant undertaking. If you're interested in contributing:

1. Review the [Tier 3 Mobile Reference](../tiers/tier-3-mobile.md) for context
2. Evaluate framework options (React Native, Capacitor, Flutter)
3. Prototype with a simple scenario (e.g., picker-wheel)
4. Document findings in this workflow guide
5. Create scenario-to-ios/android packager scenarios
