# Platform Deployments

This directory contains platform-specific deployment configurations and implementations for Vrooli, enabling the application to run natively on various operating systems and devices beyond the standard web browser.

## 📋 Overview

Vrooli's core application is built as a Progressive Web App (PWA) using React, which provides excellent cross-platform compatibility out of the box. However, certain use cases require deeper system integration, offline capabilities, or distribution through platform-specific app stores. This directory houses the necessary configurations and native wrappers to deploy Vrooli as a native application on different platforms.

## 🖥️ Current Implementations

### Desktop (Electron)
**Status:** ✅ Implemented  
**Directory:** `platforms/desktop/`

The desktop implementation uses Electron to wrap the Vrooli web application in a native desktop container, providing:
- Native desktop application experience for Windows, macOS, and Linux
- System tray integration
- Native file system access
- Desktop notifications
- Auto-update capabilities
- Offline functionality with local resource management

**Key Files:**
- `desktop/src/main.ts` - Main Electron process handling window management and system integration
- `desktop/src/preload.ts` - Preload script for secure context bridging between web and native
- `desktop/src/splash.html` - Splash screen shown during application startup
- `desktop/assets/icon.ico` - Application icon for Windows/Linux

## 🚀 Planned Implementations

### Mobile Platforms

#### Android
**Status:** 📝 Planned  
**Proposed Directory:** `platforms/android/`  
**Technology:** React Native or Capacitor

**Expected Features:**
- Native Android app distributed via Google Play Store
- Push notifications via Firebase Cloud Messaging
- Background task execution for agent operations
- Deep linking with Android Intent system
- Biometric authentication support
- Access to device sensors and cameras

#### iOS
**Status:** 📝 Planned  
**Proposed Directory:** `platforms/ios/`  
**Technology:** React Native or Capacitor

**Expected Features:**
- Native iOS app distributed via Apple App Store
- Push notifications via Apple Push Notification Service
- Background task execution within iOS limitations
- Deep linking with Universal Links
- Face ID/Touch ID authentication
- Integration with iOS Shortcuts

### Native Desktop Platforms

#### macOS (Native)
**Status:** 🔮 Future Consideration  
**Proposed Directory:** `platforms/macos/`  
**Technology:** Swift/SwiftUI with WKWebView or Tauri

**Advantages over Electron:**
- Smaller bundle size (~10MB vs ~150MB)
- Better battery efficiency
- Native macOS UI elements
- Seamless integration with macOS features (Spotlight, Handoff, etc.)

#### Windows (Native)
**Status:** 🔮 Future Consideration  
**Proposed Directory:** `platforms/windows/`  
**Technology:** WinUI 3 with WebView2 or Tauri

**Advantages over Electron:**
- Smaller memory footprint
- Windows Hello authentication
- Native Windows 11 design language
- Better integration with Windows features

### Alternative Platforms

#### Browser Extension
**Status:** 💡 Under Consideration  
**Proposed Directory:** `platforms/extension/`  
**Technology:** WebExtensions API

**Use Cases:**
- Quick access to Vrooli agents from any webpage
- Web scraping and automation directly from browser
- Context-aware AI assistance while browsing

#### Command Line Interface
**Status:** 💡 Under Consideration  
**Proposed Directory:** `platforms/cli/`  
**Technology:** Node.js with Commander.js or Rust

**Use Cases:**
- Server administration without GUI
- Scripting and automation
- CI/CD pipeline integration
- Resource-constrained environments

## 🏗️ Architecture Principles

### Shared Core
All platform implementations should:
1. Share the same core business logic from `packages/shared`
2. Reuse UI components from `packages/ui` where possible
3. Connect to the same backend APIs
4. Maintain feature parity with the web version

### Platform-Specific Optimizations
Each platform should leverage its unique capabilities:
- **Mobile:** Touch gestures, camera, GPS, biometrics
- **Desktop:** File system access, system tray, keyboard shortcuts
- **Extension:** Browser APIs, tab management, cookie access
- **CLI:** Pipe support, scripting, headless operation

### Progressive Enhancement
Start with the PWA as baseline and progressively enhance with native features:
```
PWA (baseline) → Platform Wrapper → Native APIs → Platform-Specific Features
```

## 🛠️ Development Guidelines

### Adding a New Platform

1. **Create Platform Directory**
   ```bash
   mkdir platforms/<platform-name>
   ```

2. **Define Configuration**
   Create a `config.json` specifying:
   - Build requirements
   - Target OS/runtime versions
   - Required permissions
   - Distribution channels

3. **Implement Bridge Layer**
   Create abstraction for platform-specific APIs:
   ```typescript
   interface PlatformBridge {
     storage: StorageAPI
     notifications: NotificationAPI
     filesystem?: FileSystemAPI
     // ... other platform capabilities
   }
   ```

4. **Setup Build Pipeline**
   Add platform-specific build scripts to `package.json`:
   ```json
   {
     "scripts": {
       "build:<platform>": "...",
       "dev:<platform>": "...",
       "test:<platform>": "..."
     }
   }
   ```

5. **Document Platform Requirements**
   Create `platforms/<platform>/README.md` with:
   - Setup instructions
   - Build requirements
   - Testing procedures
   - Distribution process

### Code Sharing Strategy

```
packages/ui (React components)
     ↓
platforms/shared (Platform abstraction layer)
     ↓
platforms/<specific> (Platform implementation)
```

### Testing Requirements

Each platform must include:
- Unit tests for platform-specific code
- Integration tests with core application
- Platform-specific E2E tests
- Performance benchmarks
- Accessibility testing

## 📦 Build & Distribution

### Current Build Commands
```bash
# Desktop (Electron)
cd platforms/desktop
pnpm run build        # Build for current platform
pnpm run build:all    # Build for all platforms
pnpm run dev          # Development mode with hot reload
```

### Future Build Matrix
| Platform | Development | Production | Distribution |
|----------|------------|------------|--------------|
| Desktop | `pnpm dev:desktop` | `pnpm build:desktop` | GitHub Releases, Snap Store, Microsoft Store, Mac App Store |
| Android | `pnpm dev:android` | `pnpm build:android` | Google Play Store, F-Droid |
| iOS | `pnpm dev:ios` | `pnpm build:ios` | Apple App Store, TestFlight |
| Extension | `pnpm dev:extension` | `pnpm build:extension` | Chrome Web Store, Firefox Add-ons |
| CLI | `pnpm dev:cli` | `pnpm build:cli` | npm, Homebrew, APT/YUM |

## 🔒 Security Considerations

### Platform-Specific Security
- **Desktop:** Code signing certificates for distribution
- **Mobile:** App store review and sandboxing
- **Extension:** Limited permissions and content security policies
- **CLI:** Secure credential storage and command injection prevention

### Common Security Practices
1. Never expose sensitive APIs directly to platform code
2. Use secure communication channels (IPC, encrypted storage)
3. Implement platform-specific authentication where available
4. Regular security audits for each platform
5. Follow platform-specific security guidelines

## 📊 Platform Comparison

| Feature | PWA | Desktop | Mobile | Extension | CLI |
|---------|-----|---------|--------|-----------|-----|
| Offline Support | ✅ | ✅ | ✅ | ⚠️ | ✅ |
| Push Notifications | ✅ | ✅ | ✅ | ✅ | ❌ |
| File System Access | ⚠️ | ✅ | ⚠️ | ❌ | ✅ |
| App Store Distribution | ❌ | ✅ | ✅ | ✅ | ✅ |
| System Integration | ❌ | ✅ | ✅ | ⚠️ | ✅ |
| Auto Updates | ✅ | ✅ | ✅ | ✅ | ⚠️ |
| Resource Access | ⚠️ | ✅ | ⚠️ | ❌ | ✅ |
| Background Tasks | ⚠️ | ✅ | ✅ | ✅ | ✅ |

Legend: ✅ Full Support | ⚠️ Limited Support | ❌ No Support

## 🚦 Getting Started

### For Contributors
1. Choose a platform from the "Planned Implementations" section
2. Create an issue discussing the implementation approach
3. Follow the "Adding a New Platform" guidelines
4. Submit a PR with the initial implementation

### For Users
Currently, only the desktop platform is available:
```bash
# Clone the repository
git clone https://github.com/Vrooli/Vrooli.git
cd Vrooli

# Setup and build
./scripts/manage.sh setup --target native-linux --yes yes
cd platforms/desktop
pnpm run build

# The built application will be in platforms/desktop/dist/
```

## 📚 Resources

### Platform Documentation
- [Electron Documentation](https://www.electronjs.org/docs)
- [React Native Documentation](https://reactnative.dev/docs/getting-started)
- [Capacitor Documentation](https://capacitorjs.com/docs)
- [Tauri Documentation](https://tauri.app/v1/guides/)
- [WebExtensions API](https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions)

### Design Guidelines
- [Apple Human Interface Guidelines](https://developer.apple.com/design/human-interface-guidelines/)
- [Material Design](https://material.io/design)
- [Fluent Design System](https://www.microsoft.com/design/fluent/)

## 📮 Support & Contribution

For questions about platform deployments or to contribute a new platform implementation:
- Open an issue with the `platform` label
- Join our Discord community
- Check the [main project documentation](/docs/README.md)

---

**Note:** This directory is part of Vrooli's strategy to maximize accessibility and reach by meeting users where they are, whether that's in a web browser, on their phone, or through their terminal.