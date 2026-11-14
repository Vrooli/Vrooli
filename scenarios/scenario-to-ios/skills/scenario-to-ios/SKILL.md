---
name: "scenario-to-ios"
description: "Turn a scenario into an iOS app: generate the iOS project, build it on a Mac, test it on the iOS simulator or a device, check Apple readiness (TestFlight, App Store, ad-hoc), and compare validation runs."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  status: "active"
  modes: ["tools"]
  tags: ["scenario-to-ios", "ios", "iphone", "ipad", "apple", "testflight", "app-store", "simulator", "xcode", "mobile-app"]
  requires:
    scenarios: ["scenario-to-ios"]
    commands: ["scenario-to-ios ios targets"]
  origin:
    kind: "authored"
---

## scenario-to-ios

Builds happen on a macOS bridge host; start by probing what is ready.

| You want | Command |
|---|---|
| What can build and test iOS here | `scenario-to-ios ios targets`, `scenario-to-ios ios readiness` |
| Generate the iOS project | `scenario-to-ios ios generate` |
| Build an artifact | `scenario-to-ios ios build` |
| Test on simulators and devices | `scenario-to-ios ios matrix-create`, `scenario-to-ios ios matrix-start`, `scenario-to-ios ios matrix-wait` |
| Distribution channels | `scenario-to-ios ios distribution` |
| Compare runs | `scenario-to-ios ios compare`, `scenario-to-ios ios matrix-list` |

Run `scenario-to-ios <group> <command> --help` for flags.
