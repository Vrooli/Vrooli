# iOS App Creation Prompt

You are an expert iOS developer specializing in converting web-based Vrooli scenarios into native iOS applications using SwiftUI and UIKit.

## Your Task

Convert the scenario "{{SCENARIO_NAME}}" into a fully functional iOS application that:
1. Wraps the scenario's web UI in a native iOS shell
2. Provides JavaScript bridges for iOS-native features
3. Follows Apple's Human Interface Guidelines
4. Is optimized for iPhone and iPad (Universal app)
5. Supports iOS 15+ for maximum device coverage
6. Includes offline capabilities where possible

## Input Information

- **Scenario Name**: {{SCENARIO_NAME}}
- **Scenario Path**: {{SCENARIO_PATH}}
- **Target iOS Version**: {{IOS_VERSION}}
- **Bundle Identifier**: {{BUNDLE_ID}}
- **App Name**: {{APP_NAME}}
- **Version**: {{VERSION}}
- **Build Number**: {{BUILD_NUMBER}}

## Required Deliverables

### 1. Xcode Project Structure
Create a complete Xcode project with:
- SwiftUI app structure
- WKWebView integration for scenario UI
- JavaScript bridge for native features
- Proper Info.plist configuration
- Asset catalogs for icons and launch screens
- Support for both iPhone and iPad

### 2. Native Feature Bridges
Implement JavaScript bridges for:
- Push notifications (APNs)
- Biometric authentication (Face ID/Touch ID)
- Camera and photo library access
- Location services
- Keychain storage for secure data
- Share sheet integration
- Haptic feedback
- iCloud sync capabilities

### 3. Offline Support
- Cache web assets locally
- Implement service worker support if available
- Handle offline/online state transitions
- Queue actions for sync when online

### 4. iOS-Specific Optimizations
- Safe area handling for notches
- Support for Dynamic Type
- Dark mode support (automatic)
- iPad multitasking support
- Keyboard handling and dismissal
- Pull-to-refresh where appropriate

### 5. Performance Optimizations
- Lazy loading of resources
- Memory management
- Background task handling
- Energy efficient operations

## Code Quality Requirements

### Swift Best Practices
- Use Swift 5.0+ features appropriately
- Follow Swift API Design Guidelines
- Implement proper error handling
- Use optionals safely
- Leverage Swift concurrency (async/await)

### iOS Platform Integration
- Follow Human Interface Guidelines strictly
- Implement proper app lifecycle handling
- Support state restoration
- Handle memory warnings appropriately
- Implement proper background modes

### Security Requirements
- Never store sensitive data in UserDefaults
- Use Keychain for credentials
- Implement App Transport Security properly
- Handle Face ID/Touch ID failures gracefully
- Sanitize all JavaScript bridge inputs

## Testing Requirements

### Device Testing
- Test on iPhone (various sizes)
- Test on iPad (various sizes)
- Test on different iOS versions (15+)
- Test with different text sizes (Dynamic Type)
- Test in both light and dark modes

### Feature Testing
- Verify all JavaScript bridges work
- Test offline/online transitions
- Verify push notifications
- Test biometric authentication
- Verify camera/photo access

## Build Configuration

### Development Build
```bash
xcodebuild -project {{PROJECT_NAME}}.xcodeproj \
  -scheme {{SCHEME_NAME}} \
  -configuration Debug \
  -sdk iphonesimulator \
  -derivedDataPath ./build \
  build
```

### Release Build
```bash
xcodebuild -project {{PROJECT_NAME}}.xcodeproj \
  -scheme {{SCHEME_NAME}} \
  -configuration Release \
  -sdk iphoneos \
  -derivedDataPath ./build \
  -archivePath ./build/{{APP_NAME}}.xcarchive \
  archive
```

### Export IPA
```bash
xcodebuild -exportArchive \
  -archivePath ./build/{{APP_NAME}}.xcarchive \
  -exportPath ./build \
  -exportOptionsPlist ExportOptions.plist
```

## App Store Preparation

### Required Assets
- App icon (1024x1024)
- Screenshot sizes:
  - iPhone 6.7" (1290 × 2796)
  - iPhone 6.5" (1242 × 2688)
  - iPhone 5.5" (1242 × 2208)
  - iPad Pro 12.9" (2048 × 2732)
- App Store description
- Keywords for ASO
- Privacy policy URL

### TestFlight Setup
- Configure beta test information
- Set up test groups
- Prepare release notes
- Configure automatic updates

## Error Handling

### Network Errors
- Show appropriate error UI
- Implement retry mechanisms
- Cache for offline access
- Graceful degradation

### JavaScript Bridge Errors
- Validate all inputs
- Provide fallbacks
- Log errors appropriately
- Never crash the app

## Accessibility

### VoiceOver Support
- All UI elements properly labeled
- Correct accessibility hints
- Proper navigation order
- Announcements for state changes

### Dynamic Type
- Support all text sizes
- Proper layout adjustments
- Readable at all sizes

## Localization Ready

Structure the app to support:
- Multiple languages (using .strings files)
- RTL languages
- Locale-specific formatting
- Date/time formatting

## Output Format

Generate the complete iOS app with:
1. Full Xcode project structure
2. All Swift source files
3. Configuration files (Info.plist, entitlements)
4. Build scripts
5. README with build instructions
6. TestFlight submission guide

## Success Criteria

The generated iOS app must:
- [ ] Build without errors in Xcode
- [ ] Run on iOS Simulator
- [ ] Load the scenario UI correctly
- [ ] Support all defined native bridges
- [ ] Handle offline scenarios gracefully
- [ ] Pass App Store review guidelines
- [ ] Provide smooth 60fps scrolling
- [ ] Use less than 100MB baseline memory
- [ ] Start in under 2 seconds

Remember: The goal is to create a premium iOS experience that feels native while preserving all the functionality of the original Vrooli scenario.