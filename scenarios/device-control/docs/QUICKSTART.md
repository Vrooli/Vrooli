# Quickstart — Device Control

Run the scenario, see your devices, and take a screenshot.

## Prerequisites

- The `vrooli` CLI on `PATH` (`vrooli help`); run `make setup` once from the workspace root if it is missing.
- `adb` (the `android-sdk` resource) for Android phones, tablets and Google TVs.

## Start

```bash
make setup   # once, and after dependency changes
make start   # API and UI; ports are allocated for you
make status
```

## Use it

```bash
device-control device list                       # your devices, one row each, with what each can do
device-control device screenshot "SmartTV 4K"    # wakes it, captures, prints the saved PNG path
device-control device open "SmartTV 4K" --app netflix
device-control device record moto_g_7_ --seconds 8
device-control device evidence <evidence-id> --out shot.png
```

Every verb accepts `--json`. A `capability_gap` or failure prints
`next_actions`, the exact commands that fix it.

## Connect a device

```bash
device-control device connect --kind android     # guided ladder for a phone (USB, then wireless)
device-control device connect --kind google-tv   # discovery, pairing, ADB network debugging
device-control device relations                  # transports that may be one device
device-control device relation-confirm <candidate-id>
```

[guides/connecting-a-device.md](guides/connecting-a-device.md) has the full
walkthrough.

## Test it

```bash
make test                                        # vrooli scenario test device-control
vrooli scenario test device-control --phases unit
```

Next: [START-HERE.md](START-HERE.md), [concepts/ARCHITECTURE.md](concepts/ARCHITECTURE.md),
[internal/TESTING.md](internal/TESTING.md).
