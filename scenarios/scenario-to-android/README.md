# Scenario to Android 🤖

Transform any Vrooli scenario into a native Android application with full device capabilities.

## 🎯 Overview

**scenario-to-android** provides instant mobile deployment for Vrooli scenarios, enabling them to reach billions of Android devices worldwide. Each scenario becomes a fully-functional Android app with offline support, native features, and Play Store readiness.

## ✨ Features

- **📱 Universal Conversion**: Any scenario → Android app in minutes
- **🔌 Native Bridge**: Access camera, GPS, notifications, storage
- **📴 Offline-First**: Works without internet, syncs when connected
- **🔐 APK Signing**: Ready for distribution and Play Store
- **⚡ Performance**: Optimized with ProGuard/R8
- **🎨 Customizable**: Maintains scenario branding and identity

## 🚀 Quick Start

### Installation

```bash
# Install the CLI
./cli/install.sh

# Verify installation
scenario-to-android --version
```

### Basic Usage

```bash
# Convert a scenario to Android app
scenario-to-android build study-buddy

# Build with custom name and version
scenario-to-android build notes --app-name "My Notes" --version 2.0.0

# Build and sign for release
scenario-to-android build my-app --sign --keystore mykey.jks --key-alias myalias
```

## 📋 Prerequisites

### Required
- Node.js 18+
- Java 11+ (for Android builds)
- Android SDK (for building APKs)

### Optional
- Android Studio (for advanced customization)
- ADB (for device testing)
- Gradle (uses wrapper if not installed)

### Setup Android SDK

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install android-sdk

# Set ANDROID_HOME
export ANDROID_HOME=/usr/lib/android-sdk
export PATH=$PATH:$ANDROID_HOME/tools:$ANDROID_HOME/platform-tools
```

Or download Android Studio: https://developer.android.com/studio

## 🛠️ Commands

### Build Commands

```bash
# Build APK from scenario
scenario-to-android build <scenario> [options]
  --output <dir>       Output directory
  --app-name <name>    App display name
  --version <ver>      App version
  --sign              Sign APK
  --optimize          Enable ProGuard/R8

# Test APK on device
scenario-to-android test <apk>
  --device            Use physical device
  --emulator          Use emulator

# Prepare for Play Store
scenario-to-android prepare-store <apk>
  --screenshots       Generate screenshots
  --listing          Create store listing
```

### Utility Commands

```bash
# Check system status
scenario-to-android status

# List templates
scenario-to-android templates

# Show help
scenario-to-android help
```

## 🏗️ How It Works

1. **Analyzes scenario** structure and requirements
2. **Generates Android project** from templates
3. **Wraps UI in WebView** with native bridge
4. **Copies assets** for offline support
5. **Builds APK** with Gradle
6. **Signs and optimizes** for distribution

## 🔌 Native Features

### JavaScript Bridge

The app provides a `VrooliNative` interface to JavaScript:

```javascript
// Check if running in Android app
if (window.isAndroidApp) {
    // Get device info
    const info = JSON.parse(VrooliNative.getDeviceInfo());
    
    // Request permissions
    VrooliNative.requestPermission('camera');
    
    // Save/load data
    VrooliNative.saveData('key', 'value');
    const data = VrooliNative.getData('key');
    
    // Show native toast
    VrooliNative.showToast('Hello Android!');
    
    // Vibrate device
    VrooliNative.vibrate(100);
}
```

### Supported Permissions

- 📷 **Camera**: Photo/video capture
- 📍 **Location**: GPS and network location
- 💾 **Storage**: Read/write files
- 🔔 **Notifications**: Push notifications
- 🔒 **Biometric**: Fingerprint/face authentication

## 📦 Project Structure

Generated Android projects follow standard structure:

```
scenario-android/
├── app/
│   ├── src/main/
│   │   ├── java/          # Kotlin/Java code
│   │   ├── assets/         # Scenario UI files
│   │   ├── res/            # Android resources
│   │   └── AndroidManifest.xml
│   └── build.gradle
├── gradle/
├── build.gradle
└── settings.gradle
```

## 🎨 Customization

### Modify Templates

Edit templates in `initialization/templates/android/` to customize:
- MainActivity behavior
- JavaScript bridge functions
- UI wrapper styling
- Build configurations

### Add Native Features

Extend `VrooliJSInterface` in MainActivity.kt:

```kotlin
@JavascriptInterface
fun myCustomFunction(param: String): String {
    // Your native code here
    return "result"
}
```

## 🚢 Distribution

### Google Play Store

1. Build release APK:
   ```bash
   scenario-to-android build my-app --sign --optimize
   ```

2. Prepare store assets:
   ```bash
   scenario-to-android prepare-store my-app.apk
   ```

3. Requirements:
   - Google Play Console account ($25 one-time)
   - App icon (512x512)
   - Feature graphic (1024x500)
   - Screenshots (min 2)
   - Privacy policy URL

### Direct Distribution

Share APK directly for sideloading:
```bash
# Users install with:
adb install my-app.apk
```

## 🧪 Testing

### On Physical Device

```bash
# Enable developer mode on Android device
# Connect via USB
scenario-to-android test my-app.apk --device
```

### On Emulator

```bash
# Start emulator
emulator -avd Pixel_6_API_34

# Install and test
scenario-to-android test my-app.apk --emulator
```

### Debugging

```bash
# View logs
adb logcat | grep -i vrooli

# Chrome DevTools (for WebView)
# Navigate to chrome://inspect in Chrome
```

## 🔧 Troubleshooting

### Build Fails

```bash
# Check Android SDK
scenario-to-android status

# Set ANDROID_HOME
export ANDROID_HOME=$HOME/Android/Sdk

# Install missing tools
sdkmanager "build-tools;34.0.0" "platforms;android-34"
```

### APK Won't Install

```bash
# Check device API level
adb shell getprop ro.build.version.sdk

# Enable installation from unknown sources
# Settings → Security → Unknown sources
```

### WebView Issues

- Check INTERNET permission in manifest
- Enable JavaScript in WebView settings
- Check Content Security Policy
- Review console logs with Chrome DevTools

## 🤝 Integration

### With Other Scenarios

- **deployment-manager**: Automated deployment pipeline
- **saas-billing-hub**: In-app purchases
- **notification-hub**: Push notification campaigns
- **analytics-dashboard**: Track app usage

### CI/CD Pipeline

```yaml
# GitHub Actions example
- name: Build Android App
  run: |
    scenario-to-android build ${{ env.SCENARIO }} \
      --version ${{ github.run_number }} \
      --sign --optimize
```

## 📊 Performance

- **Build time**: < 5 minutes
- **Base APK size**: < 30MB
- **Cold start**: < 3 seconds
- **Memory usage**: < 100MB baseline

## 🔮 Future Enhancements

- [ ] React Native conversion option
- [ ] iOS support (via scenario-to-ios)
- [ ] Play Store automation
- [ ] A/B testing framework
- [ ] Remote configuration
- [ ] Crash reporting integration

## 📚 Resources

- [Android Developer Docs](https://developer.android.com)
- [Material Design Guidelines](https://material.io)
- [Play Console Help](https://support.google.com/googleplay/android-developer)

## 🆘 Support

- **Issues**: [GitHub Issues](https://github.com/vrooli/vrooli/issues)
- **Docs**: [Vrooli Documentation](https://docs.vrooli.com)
- **Community**: [Discord Server](https://discord.gg/vrooli)

## 📄 License

MIT - See LICENSE file

---

**Remember**: Every Android app you create becomes a permanent mobile capability for Vrooli, enabling scenarios to reach billions of users worldwide! 🌍📱