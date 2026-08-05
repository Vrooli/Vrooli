# scenario-to-ios

Transform any Vrooli scenario into a native iOS application with full access to Apple ecosystem features.

## 🎯 What It Does

This scenario provides the capability to convert any Vrooli scenario into a premium iOS application that can be distributed through TestFlight and the App Store. It creates native iOS apps with:

- **Native iOS wrapper** using SwiftUI/UIKit
- **JavaScript bridge** for accessing iOS features
- **Offline capability** with local asset caching
- **Full Apple ecosystem integration** (Face ID, Apple Pay, iCloud, etc.)
- **Universal app support** for iPhone and iPad
- **App Store ready** packaging and metadata

## 🚀 Quick Start

### Prerequisites

- **macOS** with Xcode 14+ installed
- **Apple Developer Account** (for App Store distribution)
- **Valid code signing certificates** (for device testing)

### Installation

```bash
# Install the CLI tool
cd scenarios/scenario-to-ios/cli
./install.sh

# Verify installation
scenario-to-ios version
```

### Basic Usage

```bash
# Build iOS app from a scenario
scenario-to-ios build my-scenario --output ./build/

# Test in iOS Simulator
scenario-to-ios simulator my-scenario --device "iPhone 14"

# Validate IPA for App Store
scenario-to-ios validate ./build/MyApp.ipa

# Submit to TestFlight
scenario-to-ios testflight ./build/MyApp.ipa --notes "Initial beta release"
```

## 📱 Features

### Native iOS Capabilities

The generated apps include comprehensive JavaScript bridges for native iOS features:

#### ✅ Fully Implemented (Production-Ready)
- **Biometric Authentication**: Face ID and Touch ID via LAContext
- **Secure Storage**: iOS Keychain for credentials and sensitive data
- **Local Notifications**: Schedule and display notifications with UNUserNotificationCenter
- **Haptic Feedback**: Tactical feedback with multiple intensity levels (light, medium, heavy, success, warning, error)
- **Native Sharing**: iOS share sheet for text and URLs
- **Device Info**: Access device model, iOS version, and identifiers

#### 🔄 Partially Implemented
- **Push Notifications**: APNs registration implemented, token handling needs completion

#### 🚧 Planned Features
- **Camera & Photos**: Photo library and camera access
- **Location Services**: GPS and geofencing
- **iCloud Sync**: Cross-device data synchronization
- **Core Data**: Offline database storage
- **Apple Pay**: In-app purchases and payments

### App Store Features

- **TestFlight** integration for beta testing
- **App Store Connect** API support
- **Automatic screenshot** generation
- **Metadata optimization** for ASO
- **Privacy label** generation

### Development Features

- **Hot reload** support during development
- **Safari Web Inspector** for debugging
- **Xcode integration** for profiling
- **Simulator testing** for all device sizes
- **Crash reporting** and analytics ready

## 🏗️ Architecture

### Project Structure

```
scenario-to-ios/
├── PRD.md                    # Product requirements
├── api/                      # Go API (future)
├── cli/                      # Command-line interface
│   ├── scenario-to-ios       # Main CLI executable
│   └── install.sh            # Installation script
├── api/internal/<domain>/
│   ├── prompts/              # AI prompts for app generation
│   │   ├── ios-app-creator.md
│   │   └── ios-app-debugger.md
│   └── templates/
│       └── ios/              # Xcode project template
│           ├── project.xcodeproj/
│           └── project/
│               ├── VrooliScenarioApp.swift
│               ├── ContentView.swift
│               ├── WebView.swift
│               ├── JSBridge.swift
│               └── www/      # Web assets
└── test/                     # Test scripts
```

### How It Works

1. **Template Expansion**: Takes the Xcode project template and replaces placeholders
2. **Asset Integration**: Copies scenario UI files into the iOS app bundle
3. **Native Bridging**: Injects JavaScript bridge for iOS feature access
4. **Code Signing**: Signs the app with developer certificates
5. **Package Generation**: Creates IPA for distribution

## 📱 JavaScript API Reference

All generated iOS apps expose the `window.vrooliNative` JavaScript API for accessing native iOS features:

### Biometric Authentication
```javascript
// Authenticate with Face ID or Touch ID
window.vrooliNative.authenticateWithBiometrics("Unlock premium features")
  .then(result => {
    if (result.success) {
      console.log("Authentication successful");
    }
  });
```

### Secure Storage (Keychain)
```javascript
// Save sensitive data securely
window.vrooliNative.saveToKeychain("auth_token", "secret_value");

// Retrieve from keychain
window.vrooliNative.loadFromKeychain("auth_token")
  .then(result => {
    console.log("Retrieved:", result.value);
  });
```

### Local Notifications
```javascript
// Schedule a notification (delay in seconds)
window.vrooliNative.scheduleNotification(
  "Reminder",
  "Don't forget to check your tasks!",
  60  // Show after 60 seconds
);
```

### Haptic Feedback
```javascript
// Trigger haptic feedback
window.vrooliNative.hapticFeedback("success");  // Options: light, medium, heavy, success, warning, error
```

### Native Sharing
```javascript
// Open iOS share sheet
window.vrooliNative.share("Check this out!", "https://example.com");
```

### Device Information
```javascript
// Get device details
window.vrooliNative.getDeviceInfo()
  .then(info => {
    console.log("Model:", info.model);
    console.log("iOS Version:", info.systemVersion);
  });
```

### Event Listener
```javascript
// Wait for native bridge to be ready
window.addEventListener('vrooliNativeReady', () => {
  console.log("iOS native features available");
});
```

## 🔧 Configuration

### Config File

Edit `~/.vrooli/scenario-to-ios/config.yaml`:

```yaml
developer_team: "YOUR_TEAM_ID"

build:
  configuration: Release
  sdk: iphoneos
  deployment_target: "15.0"

testflight:
  apple_id: "your@email.com"
  app_password: "xxxx-xxxx-xxxx-xxxx"

simulator:
  device: "iPhone 14"
  ios_version: "latest"
```

### Environment Variables

```bash
export DEVELOPER_TEAM="YOUR_TEAM_ID"
export APPLE_ID="your@email.com"
export APP_PASSWORD="xxxx-xxxx-xxxx-xxxx"
export XCODE_PATH="/Applications/Xcode.app"
```

## 🧪 Testing

### Unit Tests

```bash
# Run all tests
./test/test.sh

# Test specific scenario conversion
./test/test.sh --scenario hello-world
```

### Manual Testing

1. **Simulator Testing**:
   ```bash
   scenario-to-ios simulator my-scenario
   ```

2. **Device Testing**:
   ```bash
   scenario-to-ios build my-scenario --sign "iPhone Developer"
   # Install IPA using Xcode or Apple Configurator
   ```

3. **TestFlight Testing**:
   ```bash
   scenario-to-ios testflight ./build/MyApp.ipa
   # Invite testers through App Store Connect
   ```

## 📊 Performance

### Build Performance

- **Template generation**: < 5 seconds
- **Xcode build**: 1-3 minutes (depends on scenario size)
- **IPA packaging**: < 30 seconds
- **Total time**: < 5 minutes for typical scenario

### App Performance

- **Startup time**: < 1.5 seconds
- **Memory usage**: 50-80MB baseline
- **JavaScript bridge latency**: < 10ms
- **60fps scrolling** maintained

## 🔒 Security

### Best Practices

- **Keychain storage** for sensitive data
- **App Transport Security** enforced
- **Certificate pinning** available
- **Biometric authentication** for sensitive operations
- **Encrypted local storage** for offline data

### Privacy

- **App Tracking Transparency** compliance
- **Privacy manifest** generation
- **Data minimization** by default
- **User consent** flows built-in

## 🚢 Distribution

### TestFlight

```bash
# Prepare for TestFlight
scenario-to-ios build my-scenario --sign "iPhone Distribution"

# Upload to TestFlight
scenario-to-ios testflight ./build/MyApp.ipa \
  --notes "Version 1.0.0 - Initial release" \
  --groups "internal,beta"
```

### App Store

1. **Validate** the IPA:
   ```bash
   scenario-to-ios validate ./build/MyApp.ipa
   ```

2. **Generate** App Store assets:
   ```bash
   scenario-to-ios screenshots my-scenario
   ```

3. **Submit** through App Store Connect or use fastlane

## 🐛 Troubleshooting

### Common Issues

**Issue**: "No signing certificate found"
```bash
# Open Xcode and configure signing
open /Applications/Xcode.app
# Go to Preferences → Accounts → Add Apple ID
```

**Issue**: "Command not found: scenario-to-ios"
```bash
# Add to PATH
echo 'export PATH="$HOME/.vrooli/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

**Issue**: "Build failed with exit code 65"
```bash
# Clean build folder
rm -rf ~/Library/Developer/Xcode/DerivedData
# Retry build
scenario-to-ios build my-scenario
```

## 🤝 Integration with Other Scenarios

### Works Best With

- **scenario-to-android**: Share mobile patterns
- **secrets-manager**: Secure credential storage
- **app-store-optimizer**: Metadata optimization
- **deployment-manager**: Coordinated releases

### API Access

Future API endpoints will enable:
```javascript
// Build iOS app programmatically
POST /api/v1/ios/build
{
  "scenario_name": "my-scenario",
  "config_overrides": {
    "app_name": "My App",
    "version": "1.0.0"
  }
}
```

## 📈 Roadmap

### Version 1.0 (Current)
- ✅ WebView-based iOS apps
- ✅ Basic JavaScript bridge
- ✅ TestFlight submission
- ✅ Universal app support

### Version 2.0 (Planned)
- [ ] SwiftUI native components
- [ ] Apple Watch companion apps
- [ ] Widget extensions
- [ ] App Clips support
- [ ] ARKit integration

### Version 3.0 (Future)
- [ ] visionOS support
- [ ] Mac Catalyst apps
- [ ] tvOS applications
- [ ] Cross-device Handoff

## 📚 Resources

- [Apple Developer Documentation](https://developer.apple.com)
- [Human Interface Guidelines](https://developer.apple.com/design/human-interface-guidelines/)
- [App Store Review Guidelines](https://developer.apple.com/app-store/review/guidelines/)
- [TestFlight Documentation](https://developer.apple.com/testflight/)

## 🆘 Support

- **Documentation**: See [PRD.md](./PRD.md) for detailed requirements
- **Issues**: Report bugs in the Vrooli issue tracker
- **Community**: Join the Vrooli Discord for help

## 📄 License

Part of the Vrooli project. See main repository for license details.

---

**Remember**: Every scenario you convert to iOS becomes a permanent capability that makes Vrooli exponentially more powerful. You're not just building apps - you're expanding the intelligence of the entire system.
