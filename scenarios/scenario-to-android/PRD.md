# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
This scenario provides universal Android app generation for any Vrooli scenario, enabling instant mobile deployment. It transforms web-based scenarios into native Android applications with full access to device capabilities (camera, GPS, notifications, etc.) while maintaining offline-first architecture and seamless synchronization.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- Enables agents to deploy solutions directly to billions of mobile devices worldwide
- Provides access to mobile-specific sensors and capabilities for richer data collection
- Creates persistent user touchpoints through push notifications and widgets
- Allows agents to learn from mobile usage patterns and optimize for mobile contexts
- Establishes distribution channel for monetization through Play Store

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Mobile-First Businesses**: Any agent can now create and deploy mobile apps that generate revenue through app stores
2. **Field Data Collection**: Scenarios can gather location-based, camera-based, and sensor-based data from mobile users
3. **Offline-First Workflows**: Scenarios can operate without internet, syncing when connectivity returns
4. **Push-Based Engagement**: Scenarios can re-engage users proactively through notifications
5. **Cross-Platform Sync**: Desktop/web scenarios can sync with mobile companions for seamless workflows

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Convert any scenario UI to Android WebView app
  - [ ] Generate signed APK ready for installation
  - [ ] Provide JavaScript bridge for native Android APIs
  - [ ] Support offline mode with local asset serving
  - [ ] Include build configuration for multiple architectures (arm64, x86)
  
- **Should Have (P1)**
  - [ ] Push notification support
  - [ ] Camera and photo gallery access
  - [ ] GPS/location services integration
  - [ ] Local storage and database persistence
  - [ ] Google Play Store preparation scripts
  
- **Nice to Have (P2)**
  - [ ] Widget support for home screen
  - [ ] Share intent handling (receive data from other apps)
  - [ ] Background service capabilities
  - [ ] Biometric authentication
  - [ ] In-app purchases integration

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Build Time | < 5 minutes for APK generation | CLI timing |
| App Size | < 30MB base (before scenario assets) | APK analysis |
| Startup Time | < 3 seconds cold start | Device testing |
| Memory Usage | < 100MB baseline | Android profiler |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Generated APKs install and run on Android 7+ (API 24+)
- [ ] JavaScript bridge functional for all exposed APIs
- [ ] Offline mode works without any network connectivity
- [ ] Build process handles signing certificates correctly

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: n8n
    purpose: Orchestrate build pipeline and workflow management
    integration_pattern: Shared workflows for build automation
    access_method: initialization/n8n/android-builder.json
    
optional:
  - resource_name: minio
    purpose: Store APKs and build artifacts
    fallback: Local filesystem storage
    access_method: resource-minio CLI
    
  - resource_name: redis
    purpose: Cache build configurations and templates
    fallback: Rebuild from scratch each time
    access_method: resource-redis CLI
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: android-builder.json
      location: initialization/automation/n8n/
      purpose: Orchestrates the entire Android build pipeline
  
  2_resource_cli:
    - command: resource-minio upload
      purpose: Store generated APKs
    - command: resource-redis get/set
      purpose: Cache build configurations
  
  3_direct_api:
    - justification: Only for Android SDK/Gradle operations
      endpoint: Local filesystem and build tools
```

### Data Models
```yaml
primary_entities:
  - name: AndroidBuildConfig
    storage: filesystem/redis
    schema: |
      {
        id: UUID
        scenario_name: string
        app_name: string
        package_name: string (com.vrooli.scenario.*)
        version_code: integer
        version_name: string
        min_sdk: integer
        target_sdk: integer
        permissions: string[]
        features: {
          offline: boolean
          notifications: boolean
          camera: boolean
          location: boolean
        }
        signing: {
          keystore_path: string
          key_alias: string
        }
      }
    relationships: Links to scenario configuration
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/android/build
    purpose: Generate Android APK from scenario
    input_schema: |
      {
        scenario_name: string (required)
        config_overrides?: {
          app_name?: string
          version?: string
          permissions?: string[]
        }
      }
    output_schema: |
      {
        success: boolean
        apk_path: string
        build_id: UUID
        metadata: {
          size_mb: number
          architecture: string[]
          min_android_version: string
        }
      }
    sla:
      response_time: 300000ms (5 min)
      availability: 99%
      
  - method: GET
    path: /api/v1/android/status/{build_id}
    purpose: Check build status
    output_schema: |
      {
        status: "pending" | "building" | "complete" | "failed"
        progress: number (0-100)
        logs?: string[]
      }
```

### Event Interface
```yaml
published_events:
  - name: android.build.started
    payload: { scenario_name, build_id }
    subscribers: [deployment-manager, notification-service]
    
  - name: android.build.completed
    payload: { scenario_name, build_id, apk_path }
    subscribers: [deployment-manager, distribution-service]
    
consumed_events:
  - name: scenario.updated
    action: Invalidate cached builds for scenario
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: scenario-to-android
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show build system status and Android SDK info
    flags: [--json, --verbose]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI and Android SDK versions
    flags: [--json]

custom_commands:
  - name: build
    description: Build Android APK from scenario
    api_endpoint: /api/v1/android/build
    arguments:
      - name: scenario
        type: string
        required: true
        description: Name of scenario to convert
    flags:
      - name: --output
        description: Output path for APK
      - name: --config
        description: Path to custom build config
      - name: --sign
        description: Sign APK with provided keystore
    output: Path to generated APK
    
  - name: test
    description: Test APK on emulator or device
    arguments:
      - name: apk_path
        type: string
        required: true
    flags:
      - name: --device
        description: Target device ID
      - name: --emulator
        description: Use Android emulator
    
  - name: prepare-store
    description: Prepare for Google Play Store submission
    arguments:
      - name: apk_path
        type: string
        required: true
    flags:
      - name: --screenshots
        description: Generate store screenshots
      - name: --listing
        description: Generate store listing metadata
```

### CLI-API Parity Requirements
- Every API endpoint has corresponding CLI command
- CLI uses kebab-case (build-apk vs /build)
- All API parameters available as CLI flags
- JSON output with --json flag
- Environment variables for configuration

### Implementation Standards
```yaml
implementation_requirements:
  - architecture: Thin wrapper over lib/ functions
  - language: Go (consistency with other scenarios)
  - dependencies: Minimal, reuse API client
  - error_handling: Standard exit codes
  - configuration: 
      - ~/.vrooli/scenario-to-android/config.yaml
      - Environment variables override
      - Command flags highest priority
  
installation:
  - install_script: Creates symlink in ~/.vrooli/bin/
  - path_update: Adds to PATH if needed
  - permissions: 755 executable
  - documentation: Comprehensive --help
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **Android SDK**: Build tools, platform tools, and build-tools
- **Java/Kotlin**: Required for Android compilation
- **Gradle**: Android build system
- **Node.js**: For processing scenario UIs

### Downstream Enablement
**What future capabilities does this unlock?**
- **app-to-play-store**: Automated Play Store deployment
- **mobile-analytics**: Track app usage and performance
- **cross-platform-sync**: Sync between web/desktop/mobile
- **mobile-monetization**: In-app purchases and ads

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: deployment-manager
    capability: Android APK generation and signing
    interface: API/CLI
    
  - scenario: distribution-hub
    capability: Mobile app distribution channel
    interface: API/Events
    
  - scenario: ANY
    capability: Mobile deployment of their UI
    interface: CLI

consumes_from:
  - scenario: scenario-to-desktop
    capability: Shared build patterns and templates
    fallback: Use own templates
    
  - scenario: code-signer
    capability: Certificate management
    fallback: Local keystore generation
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: technical
  inspiration: Android Studio, Flutter tooling
  
  visual_style:
    color_scheme: Material Design adherence
    typography: Roboto (Android standard)
    layout: Native Android patterns
    animations: Material motion principles
  
  personality:
    tone: technical
    mood: focused
    target_feeling: Professional mobile development

style_references:
  technical: 
    - "Android Studio build output aesthetic"
    - "Clean, informative progress indicators"
    - "Professional developer tooling"
```

### Target Audience Alignment
- **Primary Users**: Scenario developers wanting mobile deployment
- **User Expectations**: Professional build tools experience
- **Accessibility**: CLI output readable, progress clear
- **Responsive Design**: N/A (build tool, not UI)

### Brand Consistency Rules
- Follows Android platform guidelines
- Integrates seamlessly with Vrooli ecosystem
- Professional developer tool aesthetic

## 💰 Value Proposition

### Business Value
- **Primary Value**: Instant mobile deployment for any scenario
- **Revenue Potential**: $20K - $100K per app (Play Store revenue)
- **Cost Savings**: Eliminates need for separate mobile development
- **Market Differentiator**: One-click scenario to mobile app

### Technical Value
- **Reusability Score**: 100% - every scenario can use this
- **Complexity Reduction**: Mobile deployment becomes trivial
- **Innovation Enablement**: Opens mobile-first business models

## 🧬 Evolution Path

### Version 1.0 (Current)
- WebView-based Android apps
- Basic native API bridges
- APK generation and signing

### Version 2.0 (Planned)
- React Native integration for better performance
- Advanced native features (AR, ML on-device)
- Play Store automation

### Long-term Vision
- Cross-platform (iOS via scenario-to-ios)
- Native UI components option
- App store optimization tools

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with Android configuration
    - Templates for Android project structure
    - Build scripts (gradle configs)
    - Signing configuration
    
  deployment_targets:
    - local: APK for sideloading
    - play_store: AAB for Google Play
    - enterprise: MDM distribution
    
  revenue_model:
    - type: one-time (app purchase)
    - pricing_tiers: Free with IAP
    - trial_period: N/A
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: scenario-to-android
    category: deployment
    capabilities: [mobile-deployment, apk-generation, android-conversion]
    interfaces:
      - api: http://localhost:{API_PORT}/api/v1/android
      - cli: scenario-to-android
      - events: android.*
      
  metadata:
    description: Convert any scenario to native Android app
    keywords: [android, mobile, apk, deployment, app]
    dependencies: [android-sdk, java]
    enhances: [ALL_SCENARIOS]
```

### Version Management
```yaml
versioning:
  current: 1.0.0
  minimum_compatible: 1.0.0
  
  breaking_changes: []
  deprecations: []
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| Android SDK issues | Medium | High | Containerized build environment |
| Large APK sizes | Medium | Medium | ProGuard/R8 optimization |
| Signing key loss | Low | High | Backup in secure storage |

### Operational Risks
- **Build Failures**: Comprehensive error logging and recovery
- **Version Conflicts**: Isolated build environments
- **Resource Usage**: Build queue management
- **Security**: Secure keystore management

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: scenario-to-android

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/scenario-to-android
    - cli/install.sh
    - initialization/templates/android/
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/templates
    - templates/android

resources:
  required: [n8n]
  optional: [minio, redis]
  health_timeout: 60

tests:
  - name: "Android SDK accessible"
    type: exec
    command: which android || which sdkmanager
    expect:
      exit_code: 0
      
  - name: "Build simple scenario"
    type: exec
    command: ./cli/scenario-to-android build hello-world --output /tmp/test.apk
    expect:
      exit_code: 0
      file_exists: /tmp/test.apk
      
  - name: "APK is valid"
    type: exec
    command: aapt dump badging /tmp/test.apk
    expect:
      exit_code: 0
      output_contains: ["package:", "application-label:"]
```

### Test Execution Gates
```bash
./test.sh --scenario scenario-to-android --validation complete
./test.sh --structure
./test.sh --resources
./test.sh --integration
./test.sh --performance
```

### Performance Validation
- [ ] Build completes in under 5 minutes
- [ ] APK size under 30MB base
- [ ] Memory usage during build under 2GB
- [ ] Parallel builds supported

### Integration Validation
- [ ] Converts sample scenarios successfully
- [ ] APKs install on test devices
- [ ] JavaScript bridge functional
- [ ] Offline mode operational

### Capability Verification
- [ ] Any scenario becomes Android app
- [ ] Native features accessible
- [ ] Signing and distribution ready
- [ ] Performance acceptable on mid-range devices

## 📝 Implementation Notes

### Design Decisions
**WebView vs Native**: Chose WebView for compatibility with existing web UIs
- Alternative considered: React Native conversion
- Decision driver: Simplicity and universal compatibility
- Trade-offs: Some performance for massive compatibility gain

**Build System**: Gradle-based with Android SDK
- Alternative considered: Bazel, Buck
- Decision driver: Android standard, best documentation
- Trade-offs: Build speed for ecosystem compatibility

### Known Limitations
- **WebView Performance**: Not as fast as native
  - Workaround: Optimize web assets, use service workers
  - Future fix: React Native option in v2.0
  
- **iOS Support**: Android only
  - Workaround: scenario-to-ios for iOS
  - Future fix: Unified cross-platform builder

### Security Considerations
- **Keystore Protection**: Encrypted storage, secure access
- **API Keys**: Never embedded in APK, use secure config
- **Permissions**: Minimal required, user-controlled
- **Code Signing**: Mandatory for distribution

## 🔗 References

### Documentation
- README.md - Quick start guide
- docs/api.md - Full API specification
- docs/android-setup.md - Android SDK setup
- docs/signing.md - Keystore and signing guide

### Related PRDs
- scenarios/scenario-to-desktop/PRD.md
- scenarios/scenario-to-extension/PRD.md
- scenarios/scenario-to-ios/PRD.md (future)

### External Resources
- [Android Developer Documentation](https://developer.android.com)
- [Material Design Guidelines](https://material.io/design)
- [Google Play Console Help](https://support.google.com/googleplay/android-developer)

---

**Last Updated**: 2024-09-04  
**Status**: Draft  
**Owner**: AI Agent  
**Review Cycle**: After each major Android SDK update