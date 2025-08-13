# Platform Transformation Templates

This directory contains reference implementations and templates that AI agents use to add platform deployment capabilities to ANY application built through Vrooli's scenario system. These are not platform-specific deployments of Vrooli itself, but rather **reusable patterns** that agents adapt intelligently to transform apps for different deployment targets.

## 📋 Overview

In Vrooli's self-improving intelligence system, platform deployment is not a static property but a **composable capability** that can be added to any app through agent-driven transformations. When a scenario needs to deploy to Windows, mobile, or as a browser extension, agents use these templates as starting points - adapting the patterns intelligently to each app's unique structure rather than following rigid rules.

### The Transformation Pipeline

1. **Base scenarios** create functional apps (e.g., lead-generation + account-manager)
2. **Transformer scenarios** enhance these apps with new capabilities:
   - `brand-manager` scenarios customize UI/theming for specific clients
   - `app-to-desktop` scenarios add Electron wrappers for native desktop deployment
   - `app-to-mobile` scenarios enable iOS/Android distribution
   - `app-to-extension` scenarios package as browser extensions
3. **AI agents** use these templates as reference implementations, adapting them intelligently to each app's structure

### Why Templates, Not Rigid Frameworks

- **No standardized structure required**: Apps only need a `service.json` file - agents figure out the rest
- **Intelligent adaptation**: Agents read code, understand intent, and apply patterns contextually
- **Recursive learning**: Each successful transformation becomes a new reference pattern for future agents
- **Emergent compatibility**: Platform deployment patterns evolve and improve organically through use

## 🖥️ Available Templates

### Desktop (Electron)
**Status:** ✅ Template Available  
**Directory:** `platforms/desktop/`
**Use Case:** When agents need to transform web apps into native desktop applications

This template demonstrates Electron wrapping patterns that agents can adapt:
- Native desktop application patterns for Windows, macOS, and Linux
- System tray integration examples
- Native file system access patterns
- Desktop notification implementations
- Auto-update capability structures
- Offline functionality approaches

**Reference Files for Agents:**
- `desktop/src/main.ts` - Window management and system integration patterns
- `desktop/src/preload.ts` - Secure context bridging examples
- `desktop/src/splash.html` - Application startup UI patterns
- `desktop/assets/icon.ico` - Asset organization examples

### Browser Extension
**Status:** ✅ Template Available  
**Directory:** `platforms/extension/`
**Use Case:** When agents need to package apps as browser extensions

This comprehensive template provides WebExtension API patterns that agents can study:
- Manifest v3 structure with extensive documentation and examples
- Background service worker patterns for API communication and state management
- Popup UI implementation with React (easily replaceable with vanilla JS)
- Optional content script patterns for page interaction and data extraction
- Chrome storage API usage for persistence
- Message passing architecture between extension components
- Common extension patterns: context menus, keyboard shortcuts, tab management
- Security best practices and CSP configuration
- Complete TypeScript type definitions for maintainability
- Build system using Vite for fast development

### Edge Worker (Cloudflare)
**Status:** ✅ Template Available  
**Directory:** `platforms/og-worker/`
**Use Case:** When agents need to add edge computing capabilities or dynamic meta tags

This template shows serverless edge deployment patterns:
- Cloudflare Worker request handling
- Dynamic Open Graph metadata generation
- Edge caching strategies
- Origin proxying patterns

## 🚀 Future Template Opportunities

These represent platform patterns that would be valuable for agents to learn, expanding the types of transformations possible:

### Mobile Templates

#### Android
**Proposed Directory:** `platforms/android/`  
**Reference Technologies:** React Native, Capacitor, or native Kotlin

**Patterns agents could learn:**
- Google Play Store packaging and distribution
- Firebase Cloud Messaging integration
- Android Intent system handling
- Biometric authentication implementation
- Device sensor and camera access

#### iOS
**Proposed Directory:** `platforms/ios/`  
**Reference Technologies:** React Native, Capacitor, or native Swift

**Patterns agents could learn:**
- App Store submission preparation
- Apple Push Notification Service setup
- Universal Links configuration
- Face ID/Touch ID integration
- iOS Shortcuts compatibility

### Native Desktop Templates

#### macOS (Native)
**Proposed Directory:** `platforms/macos/`  
**Reference Technologies:** Swift/SwiftUI with WKWebView or Tauri

**Benefits over Electron patterns:**
- Minimal bundle size techniques (~10MB vs ~150MB)
- Battery-efficient architectures
- Native UI element integration
- macOS-specific feature usage (Spotlight, Handoff, etc.)

#### Windows (Native)
**Proposed Directory:** `platforms/windows/`  
**Reference Technologies:** WinUI 3 with WebView2 or Tauri

**Benefits over Electron patterns:**
- Memory optimization strategies
- Windows Hello authentication patterns
- Fluent Design System implementation
- Windows-specific integrations

### Specialized Templates

#### CLI (Command Line Interface)
**Proposed Directory:** `platforms/cli/`  
**Reference Technologies:** Node.js, Go, or Rust

**Patterns for headless operation:**
- Argument parsing and command structures
- Pipeline integration approaches
- Output formatting (JSON, tables, etc.)
- Interactive prompt handling

## 🏗️ How Agents Use These Templates

### Pattern Recognition & Adaptation
When an agent receives a platform transformation task:
1. **Analyze the target app's structure** - understand entry points, dependencies, build outputs
2. **Study relevant templates** - learn patterns from existing implementations
3. **Adapt intelligently** - apply patterns contextually, not through rigid copying
4. **Test and iterate** - verify the transformation works for the specific app

### No Rigid Requirements
Unlike traditional frameworks, agents working with these templates:
- **Don't need standardized app structures** - only `service.json` is required
- **Can mix and match patterns** - combine ideas from multiple templates
- **Learn from failures** - unsuccessful attempts become learning data
- **Improve over time** - each transformation enriches the pattern library

### Platform-Specific Capabilities Agents Learn
- **Mobile:** Touch gesture handling, device API access, app store requirements
- **Desktop:** Native menu integration, file system access, OS-specific features
- **Extension:** Permission models, content script injection, browser API usage
- **CLI:** Argument parsing, output formatting, pipeline integration
- **Edge:** Request routing, caching strategies, serverless constraints

## 🛠️ Contributing New Templates

### Adding a Template for Agents

1. **Create Template Directory**
   ```bash
   mkdir platforms/<platform-name>
   ```

2. **Include Working Examples**
   Provide complete, functional code that agents can study:
   - Entry point patterns
   - Configuration examples
   - Build scripts
   - Common pitfalls and solutions

3. **Document Key Patterns**
   Create a README explaining:
   - What makes this platform unique
   - Critical implementation patterns
   - Platform-specific constraints
   - Distribution requirements

4. **Provide Multiple Approaches**
   When possible, show different implementation strategies:
   ```
   platforms/desktop/
     ├── electron-approach/     # Heavy but feature-rich
     ├── tauri-approach/        # Lightweight alternative
     └── patterns.md            # When to use each
   ```

5. **Include Integration Points**
   Show how apps typically connect to platform features:
   - Native API access patterns
   - Permission handling
   - Platform-specific UI adaptations
   - Performance optimizations

### What Makes a Good Template

**DO:**
- Include complete, working code
- Show real-world patterns agents will need
- Document gotchas and edge cases
- Provide multiple implementation options
- Keep templates generic (not app-specific)

**DON'T:**
- Create rigid frameworks agents must follow
- Assume specific app structures
- Over-engineer the template
- Include app-specific business logic

## 📦 Example Patterns Agents Can Learn

### Build & Distribution Strategies

Templates demonstrate various build and distribution patterns that agents adapt:

| Platform | Build Patterns | Distribution Channels | Key Learnings |
|----------|---------------|----------------------|---------------|
| Desktop | Electron Builder, Tauri bundling | GitHub Releases, App Stores | Code signing, auto-updates |
| Android | Gradle, React Native Metro | Play Store, F-Droid, APK | Manifest permissions, signing |
| iOS | Xcode, Metro bundler | App Store, TestFlight | Provisioning profiles, entitlements |
| Extension | Webpack, Vite | Web Stores, Self-hosted | Manifest versions, CSP rules |
| CLI | pkg, nexe, native binaries | npm, Homebrew, package managers | Cross-platform binaries |
| Edge | Wrangler, esbuild | Cloudflare, Vercel, Netlify | Bundle size limits, runtime APIs |

### How Agents Apply These Patterns

When transforming an app, agents:
1. **Identify the app's build system** (Vite, Webpack, etc.)
2. **Select compatible platform patterns** from templates
3. **Merge platform requirements** with existing build pipeline
4. **Generate distribution artifacts** following platform conventions
5. **Create deployment instructions** specific to the transformed app

## 🔒 Security Patterns for Agents to Consider

### Platform-Specific Security Challenges
Templates demonstrate how to handle security requirements that agents must understand:
- **Desktop:** Code signing, certificate management, privileged system access
- **Mobile:** App sandboxing, permission models, secure storage APIs
- **Extension:** Content Security Policies, cross-origin restrictions, permission scopes
- **CLI:** Credential management, shell injection prevention, secure piping
- **Edge:** Secret management, origin validation, DDoS protection

### Security Patterns Agents Learn
1. **Least privilege principle** - Request only necessary permissions
2. **Secure defaults** - Start restrictive, expand carefully
3. **Platform authentication** - Use native auth when available (Touch ID, Windows Hello)
4. **Data isolation** - Separate user data from app code
5. **Update mechanisms** - Secure, verified update channels

## 📊 Platform Capability Matrix

Agents use this matrix to understand what's possible on each platform:

| Capability | Web | Desktop | Mobile | Extension | CLI | Edge |
|------------|-----|---------|--------|-----------|-----|------|
| Local File Access | Limited | ✅ | Limited | ❌ | ✅ | ❌ |
| Native APIs | ❌ | ✅ | ✅ | Browser | ✅ | ❌ |
| Background Processing | Limited | ✅ | ✅ | ✅ | ✅ | Per-request |
| Push Notifications | ✅ | ✅ | ✅ | ✅ | ❌ | ❌ |
| Offline Operation | ✅ | ✅ | ✅ | Limited | ✅ | ❌ |
| App Store Distribution | ❌ | ✅ | ✅ | ✅ | Package Managers | N/A |
| System Integration | ❌ | ✅ | ✅ | Limited | ✅ | ❌ |
| Resource Constraints | Browser | OS | OS/Battery | Browser | OS | Time/Memory |

## 🚦 Getting Started

### For Scenario Developers
1. Identify platform transformation needs for your apps
2. Create `app-to-<platform>` scenarios using these templates as reference
3. Let agents handle the intelligent adaptation
4. Each successful transformation enriches the pattern library

### For Template Contributors
1. Choose a platform not yet represented
2. Create a working example implementation
3. Document the key patterns and gotchas
4. Submit PR with the new template

### Example: Using Templates in Scenarios
```javascript
// In an app-to-desktop scenario
scenario.transform = async (app, agent) => {
  // Agent studies the Electron template
  const template = await agent.study('platforms/desktop');
  
  // Agent analyzes the app structure
  const appStructure = await agent.analyze(app);
  
  // Agent adapts patterns intelligently
  return agent.adaptTemplate(template, appStructure);
};
```

## 📚 Resources

### Platform References
- [Electron Documentation](https://www.electronjs.org/docs) - Desktop app patterns
- [React Native Documentation](https://reactnative.dev/docs/getting-started) - Mobile patterns
- [Capacitor Documentation](https://capacitorjs.com/docs) - Cross-platform mobile
- [Tauri Documentation](https://tauri.app/v1/guides/) - Lightweight desktop
- [WebExtensions API](https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions) - Browser extensions

### Platform Design Systems
- [Apple Human Interface Guidelines](https://developer.apple.com/design/human-interface-guidelines/) - iOS/macOS patterns
- [Material Design](https://material.io/design) - Android patterns
- [Fluent Design System](https://www.microsoft.com/design/fluent/) - Windows patterns

## 📮 The Recursive Improvement Loop

Remember: Every successful platform transformation becomes part of the knowledge base. When an agent successfully transforms an app to run on Windows, that solution becomes a new template others can learn from. This is how Vrooli's intelligence compounds - not through rigid frameworks, but through accumulated pattern recognition.

---

**Note:** This directory embodies Vrooli's core philosophy - intelligence emerges from patterns, not prescriptions. These templates are seeds of knowledge that agents cultivate into platform transformation capabilities, growing more sophisticated with each use.