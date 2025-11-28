# Product Requirements Document (PRD)

> **Template Version**: 2.0.0
> **Canonical Reference**: scenarios/prd-control-tower/docs/CANONICAL_PRD_TEMPLATE.md
> **Last Updated**: 2025-11-27

## 🎯 Overview

**Purpose**: This scenario provides universal Android app generation for any Vrooli scenario, enabling instant mobile deployment. It transforms web-based scenarios into native Android applications with full access to device capabilities (camera, GPS, notifications, etc.) while maintaining offline-first architecture and seamless synchronization.

**Primary Users**: Scenario developers wanting mobile deployment capabilities

**Deployment Surfaces**: CLI, API, UI

Every scenario built for Vrooli can instantly become an Android app through this conversion system, opening mobile-first business models and enabling agents to deploy solutions directly to billions of mobile devices worldwide.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | API Health Check | Health endpoint operational at /health and /api/v1/health, schema-compliant with readiness field
- [x] OT-P0-002 | UI Server | UI server running with schema-compliant health checks including API connectivity monitoring
- [x] OT-P0-003 | CLI Tool | scenario-to-android CLI installed and functional with help/status commands
- [x] OT-P0-004 | Android Templates | Base Android project templates in initialization/templates/
- [x] OT-P0-005 | Build API Endpoint | /api/v1/android/build endpoint for APK generation
- [x] OT-P0-006 | Build Status Endpoint | /api/v1/android/status/{build_id} for tracking builds
- [x] OT-P0-007 | JavaScript Bridge | Full VrooliJSInterface with device info, permissions, storage, vibration
- [x] OT-P0-008 | Offline Mode | WebView with local asset serving and service worker support
- [x] OT-P0-009 | Convert Scenario UI to Android | Templates and CLI convert any scenario to Android project
- [ ] OT-P0-010 | Generate Signed APK | Generate signed APK ready for installation (requires Android SDK on host)
- [ ] OT-P0-011 | Multi-architecture Builds | Include build configuration for multiple architectures (gradle configs exist, needs SDK)

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Push Notification Support | Enable push notifications in generated Android apps
- [ ] OT-P1-002 | Camera and Photo Gallery Access | Provide camera and photo gallery integration
- [ ] OT-P1-003 | GPS/Location Services | Integrate GPS and location services
- [ ] OT-P1-004 | Local Storage and Database | Provide local storage and database persistence
- [ ] OT-P1-005 | Play Store Preparation Scripts | Scripts for preparing apps for Google Play Store submission

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Widget Support | Support for home screen widgets
- [ ] OT-P2-002 | Share Intent Handling | Handle share intents to receive data from other apps
- [ ] OT-P2-003 | Background Service Capabilities | Enable background services for apps
- [ ] OT-P2-004 | Biometric Authentication | Integrate biometric authentication support
- [ ] OT-P2-005 | In-app Purchases Integration | Support for in-app purchases

## 🧱 Tech Direction Snapshot

- **Preferred Stack**: Go API, Node.js UI, WebView-based Android apps
- **Data Storage**: In-memory build state with optional Redis caching, APK artifacts in local filesystem or MinIO
- **Integration Strategy**: Shared workflows for build automation via n8n, resource CLIs for MinIO and Redis
- **Non-goals**: iOS support (separate scenario-to-ios), React Native conversion (v2.0 future)

Key architectural choices:
- WebView-based for universal compatibility with existing web UIs
- Gradle build system for Android standard ecosystem compatibility
- JavaScript bridge for native feature access from web code
- Offline-first with service worker support

## 🤝 Dependencies & Launch Plan

**Required Local Resources**:
- n8n: Orchestrate build pipeline and workflow management (shared workflows in initialization/n8n/)

**Optional Local Resources**:
- MinIO: Store APKs and build artifacts (fallback: local filesystem)
- Redis: Cache build configurations and templates (fallback: rebuild from scratch)

**Scenario Dependencies**: None (enables ALL scenarios)

**External Dependencies**:
- Android SDK: Required for actual APK compilation and signing
- Java/Kotlin: Required for Android compilation
- Gradle: Android build system
- Node.js: For processing scenario UIs

**Launch Sequencing**:
1. Core project generation (templates, CLI, API endpoints) - ✅ Complete
2. Android SDK environment setup - ⏳ Blocked (external dependency)
3. APK signing and multi-architecture builds - ⏳ Waiting on SDK
4. P1 native features (camera, GPS, notifications) - Future

**Risks**:
- Android SDK installation complexity may reduce adoption
- Large APK sizes if not optimized with ProGuard/R8
- Signing key management requires secure backup strategy

## 🎨 UX & Branding

**Visual Palette**: Material Design adherence with Roboto typography and Android standard UI patterns

**Accessibility Targets**: Follow Android accessibility guidelines, support TalkBack screen reader, minimum touch target sizes (48dp)

**Voice/Personality**: Technical and professional, similar to Android Studio build tooling. Focused on developer productivity with clear progress indicators and informative error messages.

**Brand Consistency**: Integrates seamlessly with Vrooli ecosystem while following Android platform guidelines for generated apps.

## 📎 Appendix

**Related Documentation**: README.md (quick start guide), PROBLEMS.md (known issues and standards compliance)

**Related PRDs**: scenario-to-desktop (desktop app generation), scenario-to-extension (browser extension generation), scenario-to-ios (iOS app generation - future)

**External References**: [Android Developer Documentation](https://developer.android.com), [Material Design Guidelines](https://material.io/design), [Google Play Console Help](https://support.google.com/googleplay/android-developer)
