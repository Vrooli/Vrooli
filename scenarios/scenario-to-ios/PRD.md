# Product Requirements Document (PRD)

## 🎯 Capability Definition

### Core Capability
**What permanent capability does this scenario add to Vrooli?**
This scenario provides universal iOS app generation for any Vrooli scenario, enabling instant deployment to iPhones and iPads. It transforms web-based scenarios into native iOS applications with full access to Apple ecosystem features (Face ID, Apple Pay, HealthKit, ARKit, Core ML) while providing seamless iCloud sync, App Store distribution, and the premium user experience iOS users expect.

### Intelligence Amplification
**How does this capability make future agents smarter?**
- **Apple Ecosystem Integration**: Agents can leverage exclusive iOS capabilities like Siri Shortcuts, Handoff, and Universal Clipboard for deeper system integration
- **Premium Monetization**: iOS users spend 2.5x more than Android users, enabling higher-revenue business models
- **Privacy-First Architecture**: Apple's privacy frameworks enable secure on-device processing, making agents trustworthy for sensitive data
- **Neural Engine Access**: Direct access to Apple Silicon's ML accelerators for blazing-fast on-device AI inference
- **Continuity Features**: Scenarios can seamlessly transition between iPhone, iPad, Mac, and Apple Watch
- **Enterprise Distribution**: TestFlight and Enterprise certificates enable controlled rollouts to organizations

### Recursive Value
**What new scenarios become possible after this exists?**
1. **Health & Wellness Ecosystem**: Scenarios that integrate with HealthKit, workout tracking, and Apple Watch for comprehensive health management
2. **AR-Powered Business Tools**: Scenarios using ARKit for interior design, real estate visualization, and industrial training
3. **Premium SaaS Products**: High-value B2B scenarios distributed through Apple Business Manager
4. **Creative Professional Suite**: Scenarios leveraging ProRAW, ProRes, and Pencil support for creative workflows
5. **Privacy-Centric Intelligence**: Scenarios that process sensitive data entirely on-device using Core ML
6. **Cross-Apple-Device Workflows**: Scenarios that start on iPhone, continue on iPad, and finish on Mac

## 📊 Success Metrics

### Functional Requirements
- **Must Have (P0)**
  - [ ] Convert any scenario UI to native iOS app (SwiftUI/UIKit hybrid)
  - [ ] Generate signed IPA ready for TestFlight/App Store
  - [ ] JavaScript bridge for native iOS APIs
  - [ ] Support iOS 15+ (covering 95% of devices)
  - [ ] Universal app support (iPhone + iPad)
  - [ ] Xcode project generation with proper provisioning
  
- **Should Have (P1)**
  - [ ] Push notifications via APNs
  - [ ] Face ID/Touch ID authentication
  - [ ] Core Data integration for offline storage
  - [ ] iCloud sync for cross-device data
  - [ ] App Store Connect API integration
  - [ ] Widget extensions for home screen
  
- **Nice to Have (P2)**
  - [ ] Apple Watch companion app
  - [ ] Siri Shortcuts integration
  - [ ] App Clips for instant experiences
  - [ ] StoreKit 2 for in-app purchases
  - [ ] ARKit scene generation
  - [ ] Core ML model deployment

### Performance Criteria
| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| Build Time | < 3 minutes for IPA generation | Xcode build timing |
| App Size | < 50MB base (before scenario assets) | IPA analysis |
| Startup Time | < 1.5 seconds cold start | Instruments profiling |
| Memory Usage | < 80MB baseline | Xcode memory gauge |
| Battery Impact | < 1% per hour active use | Energy impact analysis |

### Quality Gates
- [ ] All P0 requirements implemented and tested
- [ ] Generated IPAs install and run on iOS 15+ devices
- [ ] JavaScript bridge functional for all exposed APIs
- [ ] Apps pass App Store review guidelines
- [ ] Human Interface Guidelines compliance verified
- [ ] Accessibility standards met (VoiceOver, Dynamic Type)

## 🏗️ Technical Architecture

### Resource Dependencies
```yaml
required:
  - resource_name: n8n
    purpose: Orchestrate build pipeline and workflow management
    integration_pattern: Shared workflows for build automation
    access_method: initialization/n8n/ios-builder.json
    
  - resource_name: browserless
    purpose: Validate iOS web views and UI testing
    integration_pattern: Screenshot validation
    access_method: resource-browserless CLI
    
optional:
  - resource_name: minio
    purpose: Store IPAs and build artifacts
    fallback: Local filesystem storage
    access_method: resource-minio CLI
    
  - resource_name: redis
    purpose: Cache build configurations and certificates
    fallback: Rebuild from scratch each time
    access_method: resource-redis CLI
    
  - resource_name: postgres
    purpose: Track app versions and TestFlight builds
    fallback: File-based version tracking
    access_method: resource-postgres CLI
```

### Resource Integration Standards
```yaml
integration_priorities:
  1_shared_workflows:
    - workflow: ios-builder.json
      location: initialization/n8n/
      purpose: Orchestrates the entire iOS build pipeline
    - workflow: app-store-deploy.json
      location: initialization/n8n/
      purpose: Automates TestFlight and App Store submission
  
  2_resource_cli:
    - command: resource-minio upload
      purpose: Store generated IPAs and dSYMs
    - command: resource-redis get/set
      purpose: Cache provisioning profiles
    - command: resource-postgres query
      purpose: Track build history
  
  3_direct_api:
    - justification: Only for Xcode/Swift operations
      endpoint: Local filesystem and build tools
    - justification: App Store Connect API
      endpoint: https://api.appstoreconnect.apple.com
```

### Data Models
```yaml
primary_entities:
  - name: iOSBuildConfig
    storage: filesystem/redis
    schema: |
      {
        id: UUID
        scenario_name: string
        app_name: string
        bundle_identifier: string (com.vrooli.*)
        version: string (1.0.0)
        build_number: integer
        minimum_ios_version: string (15.0)
        capabilities: {
          push_notifications: boolean
          icloud: boolean
          healthkit: boolean
          siri: boolean
          app_groups: boolean
        }
        signing: {
          team_id: string
          provisioning_profile: string
          certificate_id: string
        }
        app_store: {
          app_id: string
          sku: string
          primary_category: string
        }
      }
    relationships: Links to scenario configuration and App Store metadata
```

### API Contract
```yaml
endpoints:
  - method: POST
    path: /api/v1/ios/build
    purpose: Generate iOS app from scenario
    input_schema: |
      {
        scenario_name: string (required)
        config_overrides?: {
          app_name?: string
          version?: string
          capabilities?: string[]
          target_device?: "iphone" | "ipad" | "universal"
        }
      }
    output_schema: |
      {
        success: boolean
        ipa_path: string
        build_id: UUID
        metadata: {
          size_mb: number
          supported_devices: string[]
          minimum_ios_version: string
          app_store_ready: boolean
        }
      }
    sla:
      response_time: 180000ms (3 min)
      availability: 99%
      
  - method: POST
    path: /api/v1/ios/testflight
    purpose: Submit to TestFlight
    input_schema: |
      {
        build_id: UUID
        release_notes: string
        test_groups?: string[]
      }
    output_schema: |
      {
        success: boolean
        testflight_url: string
        build_number: integer
      }
      
  - method: GET
    path: /api/v1/ios/status/{build_id}
    purpose: Check build status
    output_schema: |
      {
        status: "pending" | "building" | "signing" | "complete" | "failed"
        progress: number (0-100)
        logs?: string[]
        errors?: string[]
      }
```

### Event Interface
```yaml
published_events:
  - name: ios.build.started
    payload: { scenario_name, build_id, target_devices }
    subscribers: [deployment-manager, notification-hub]
    
  - name: ios.build.completed
    payload: { scenario_name, build_id, ipa_path, testflight_ready }
    subscribers: [app-store-manager, distribution-service]
    
  - name: ios.testflight.uploaded
    payload: { scenario_name, build_number, testflight_url }
    subscribers: [notification-hub, beta-testing-manager]
    
consumed_events:
  - name: scenario.updated
    action: Invalidate cached builds and provisioning
  - name: certificate.expiring
    action: Alert and prepare renewal
```

## 🖥️ CLI Interface Contract

### Command Structure
```yaml
cli_binary: scenario-to-ios
install_script: cli/install.sh

required_commands:
  - name: status
    description: Show build system status and Xcode info
    flags: [--json, --verbose, --check-signing]
    
  - name: help
    description: Display command help and usage
    flags: [--all, --command <name>]
    
  - name: version
    description: Show CLI, Xcode, and Swift versions
    flags: [--json, --check-updates]

custom_commands:
  - name: build
    description: Build iOS app from scenario
    api_endpoint: /api/v1/ios/build
    arguments:
      - name: scenario
        type: string
        required: true
        description: Name of scenario to convert
    flags:
      - name: --output
        description: Output path for IPA
      - name: --config
        description: Path to custom build config
      - name: --device
        description: Target device (iphone/ipad/universal)
      - name: --sign
        description: Code signing identity
      - name: --provision
        description: Provisioning profile path
    output: Path to generated IPA and dSYM
    
  - name: testflight
    description: Submit to TestFlight
    api_endpoint: /api/v1/ios/testflight
    arguments:
      - name: ipa_path
        type: string
        required: true
    flags:
      - name: --notes
        description: Release notes for testers
      - name: --groups
        description: TestFlight groups (comma-separated)
      - name: --auto-release
        description: Auto-release after testing
    
  - name: validate
    description: Validate against App Store requirements
    arguments:
      - name: ipa_path
        type: string
        required: true
    flags:
      - name: --fix
        description: Attempt to fix validation issues
      - name: --screenshots
        description: Generate App Store screenshots
    
  - name: simulator
    description: Test on iOS Simulator
    arguments:
      - name: scenario
        type: string
        required: true
    flags:
      - name: --device
        description: Simulator device type
      - name: --ios-version
        description: iOS version to simulate
      - name: --record
        description: Record simulator session
```

### CLI-API Parity Requirements
- Every API endpoint has corresponding CLI command
- CLI uses kebab-case (build-ios vs /build)
- All API parameters available as CLI flags
- JSON output with --json flag
- Keychain integration for certificates

### Implementation Standards
```yaml
implementation_requirements:
  - architecture: Thin wrapper over lib/ functions
  - language: Go with Swift tooling integration
  - dependencies: Minimal, reuse API client
  - error_handling: Standard exit codes
  - configuration: 
      - ~/.vrooli/scenario-to-ios/config.yaml
      - Keychain for sensitive data
      - Environment variables override
      - Command flags highest priority
  
installation:
  - install_script: Creates symlink in ~/.vrooli/bin/
  - path_update: Adds to PATH if needed
  - permissions: 755 executable
  - xcode_check: Verifies Xcode installation
  - certificate_setup: Guides through signing setup
```

## 🔄 Integration Requirements

### Upstream Dependencies
**What capabilities must exist before this can function?**
- **Xcode**: Full Xcode installation with command line tools
- **Swift**: Swift compiler and runtime
- **Apple Developer Account**: For code signing and distribution
- **macOS**: Requires macOS for iOS development (or cloud Mac service)

### Downstream Enablement
**What future capabilities does this unlock?**
- **app-to-app-store**: Automated App Store submission and release
- **ios-analytics**: App analytics and crash reporting
- **watch-companion**: Apple Watch app generation
- **ios-widgets**: Home screen widget creation
- **siri-shortcuts**: Voice command integration

### Cross-Scenario Interactions
```yaml
provides_to:
  - scenario: deployment-manager
    capability: iOS app generation and distribution
    interface: API/CLI
    
  - scenario: app-store-optimizer
    capability: App Store ready packages
    interface: API/Events
    
  - scenario: ANY
    capability: iOS deployment of their UI
    interface: CLI

consumes_from:
  - scenario: scenario-to-android
    capability: Shared mobile patterns
    fallback: Use own templates
    
  - scenario: secrets-manager
    capability: Certificate and key management
    fallback: Local keychain storage
    
  - scenario: seo-optimizer
    capability: App Store keyword optimization
    fallback: Basic metadata generation
```

## 🎨 Style and Branding Requirements

### UI/UX Style Guidelines
```yaml
style_profile:
  category: premium/technical
  inspiration: Xcode, Swift Playgrounds
  
  visual_style:
    design_system: Human Interface Guidelines
    typography: SF Pro (Apple system font)
    layout: iOS native patterns
    animations: UIKit/SwiftUI defaults
    dark_mode: Automatic with system
  
  personality:
    tone: premium, polished
    mood: professional
    target_feeling: "It just works" Apple experience

style_references:
  technical: 
    - "Xcode build interface aesthetic"
    - "TestFlight beta experience"
    - "App Store Connect dashboard"
  premium:
    - "Apple's attention to detail"
    - "Smooth, refined interactions"
```

### Target Audience Alignment
- **Primary Users**: Scenario developers targeting iOS users
- **User Expectations**: Premium, polished Apple experience
- **Accessibility**: Full VoiceOver, Dynamic Type support
- **Responsive Design**: Adaptive layouts for all iOS devices

### Brand Consistency Rules
- Follows Human Interface Guidelines strictly
- Maintains Apple platform conventions
- Premium feel matching iOS ecosystem expectations

## 💰 Value Proposition

### Business Value
- **Primary Value**: Access to premium iOS market (1B+ devices)
- **Revenue Potential**: $50K - $500K per app (iOS users spend more)
- **Cost Savings**: Eliminates need for iOS development team
- **Market Differentiator**: Instant scenario to premium iOS app

### Technical Value
- **Reusability Score**: 100% - every scenario can target iOS
- **Complexity Reduction**: iOS development becomes one command
- **Innovation Enablement**: Access to cutting-edge Apple features

## 🧬 Evolution Path

### Version 1.0 (Current)
- WebView-based iOS apps with native wrapper
- Basic iOS API bridges
- IPA generation and TestFlight submission

### Version 2.0 (Planned)
- SwiftUI native components option
- Advanced features (ARKit, Core ML)
- App Store optimization tools
- Widget and App Clip support

### Long-term Vision
- Full native SwiftUI generation from scenarios
- Apple Watch and tvOS support
- Mac Catalyst for desktop apps
- Vision Pro support for spatial computing

## 🔄 Scenario Lifecycle Integration

### Direct Scenario Deployment
```yaml
direct_execution:
  supported: true
  structure_compliance:
    - service.json with iOS configuration
    - SwiftUI/UIKit templates
    - Xcode project structure
    - Info.plist configuration
    
  deployment_targets:
    - local: IPA for development testing
    - testflight: Beta testing distribution
    - app_store: Public App Store release
    - enterprise: In-house enterprise distribution
    
  revenue_model:
    - type: freemium (free with IAP)
    - pricing_tiers: $0.99 - $99.99
    - subscription_options: Monthly/Annual
```

### Capability Discovery
```yaml
discovery:
  registry_entry:
    name: scenario-to-ios
    category: deployment
    capabilities: [ios-deployment, ipa-generation, app-store-ready]
    interfaces:
      - api: http://localhost:{API_PORT}/api/v1/ios
      - cli: scenario-to-ios
      - events: ios.*
      
  metadata:
    description: Convert any scenario to native iOS app
    keywords: [ios, iphone, ipad, app-store, mobile, apple]
    dependencies: [xcode, swift, macos]
    enhances: [ALL_SCENARIOS]
```

### Version Management
```yaml
versioning:
  current: 1.0.0
  minimum_compatible: 1.0.0
  ios_minimum: 15.0
  xcode_minimum: 14.0
  
  breaking_changes: []
  deprecations: []
```

## 🚨 Risk Mitigation

### Technical Risks
| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| App Store rejection | Medium | High | Automated guideline checking |
| Certificate expiration | Medium | High | Proactive renewal alerts |
| iOS version fragmentation | Low | Medium | Support iOS 15+ (95% coverage) |
| Build environment issues | Medium | Medium | Containerized Mac runners |

### Operational Risks
- **Signing Failures**: Comprehensive keychain management
- **API Changes**: Version detection and compatibility layer
- **Resource Usage**: Build queue with cloud Mac fallback
- **Security**: Secure certificate storage and rotation

## ✅ Validation Criteria

### Declarative Test Specification
```yaml
version: 1.0
scenario: scenario-to-ios

structure:
  required_files:
    - .vrooli/service.json
    - PRD.md
    - api/main.go
    - api/go.mod
    - cli/scenario-to-ios
    - cli/install.sh
    - initialization/templates/ios/
    - scenario-test.yaml
    
  required_dirs:
    - api
    - cli
    - initialization
    - initialization/templates
    - templates/ios

resources:
  required: [n8n, browserless]
  optional: [minio, redis, postgres]
  health_timeout: 60

tests:
  - name: "Xcode accessible"
    type: exec
    command: xcodebuild -version
    expect:
      exit_code: 0
      output_contains: ["Xcode"]
      
  - name: "Swift compiler available"
    type: exec
    command: swift --version
    expect:
      exit_code: 0
      output_contains: ["Swift"]
      
  - name: "Build simple scenario"
    type: exec
    command: ./cli/scenario-to-ios build hello-world --output /tmp/test.ipa
    expect:
      exit_code: 0
      file_exists: /tmp/test.ipa
      
  - name: "IPA is valid"
    type: exec
    command: unzip -t /tmp/test.ipa
    expect:
      exit_code: 0
      output_contains: ["Payload/"]
```

### Test Execution Gates
```bash
./test.sh --scenario scenario-to-ios --validation complete
./test.sh --structure
./test.sh --resources
./test.sh --integration
./test.sh --performance
```

### Performance Validation
- [ ] Build completes in under 3 minutes
- [ ] IPA size under 50MB base
- [ ] Memory usage during build under 4GB
- [ ] Supports parallel builds

### Integration Validation
- [ ] Converts sample scenarios successfully
- [ ] IPAs install on test devices
- [ ] JavaScript bridge functional
- [ ] TestFlight submission works

### Capability Verification
- [ ] Any scenario becomes iOS app
- [ ] Native features accessible
- [ ] App Store submission ready
- [ ] Performance acceptable on iPhone 12 and newer

## 📝 Implementation Notes

### Design Decisions
**Native Wrapper vs Full Native**: Hybrid approach with native wrapper
- Alternative considered: Full SwiftUI generation
- Decision driver: Compatibility with web-based scenarios
- Trade-offs: Some performance for universal compatibility

**Build System**: Xcode with Swift Package Manager
- Alternative considered: React Native, Flutter
- Decision driver: Native iOS quality and features
- Trade-offs: macOS requirement for true iOS quality

### Known Limitations
- **macOS Requirement**: Need Mac for building
  - Workaround: Cloud Mac services (MacStadium, AWS)
  - Future fix: Cross-platform build system
  
- **App Store Approval**: Subject to Apple review
  - Workaround: Automated guideline checking
  - Future fix: Pre-submission validation suite

### Security Considerations
- **Certificate Protection**: Keychain integration, never in code
- **API Keys**: Secure environment variables
- **App Transport Security**: Enforce HTTPS
- **Privacy**: App Tracking Transparency compliance
- **Entitlements**: Minimal required capabilities

## 🔗 References

### Documentation
- README.md - Quick start guide
- docs/api.md - Full API specification
- docs/ios-setup.md - Xcode and signing setup
- docs/app-store.md - App Store submission guide

### Related PRDs
- scenarios/scenario-to-android/PRD.md
- scenarios/scenario-to-desktop/PRD.md
- scenarios/scenario-to-extension/PRD.md

### External Resources
- [Apple Developer Documentation](https://developer.apple.com)
- [Human Interface Guidelines](https://developer.apple.com/design/human-interface-guidelines/)
- [App Store Review Guidelines](https://developer.apple.com/app-store/review/guidelines/)
- [TestFlight Documentation](https://developer.apple.com/testflight/)

---

**Last Updated**: 2025-09-04  
**Status**: Draft  
**Owner**: AI Agent  
**Review Cycle**: After each major iOS/Xcode release