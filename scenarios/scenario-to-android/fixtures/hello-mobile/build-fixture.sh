#!/usr/bin/env bash
set -euo pipefail

fixture_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$fixture_root/../../../.." && pwd)"
gradle_bin="${GRADLE_BIN:-gradle}"
if [[ ! -x "$gradle_bin" && "$gradle_bin" != */* ]]; then
  gradle_bin="$(command -v "$gradle_bin" || true)"
fi
if [[ -z "$gradle_bin" || ! -x "$gradle_bin" ]]; then
  echo "hello-mobile build unavailable: Gradle 8.10.2 is required (set GRADLE_BIN)" >&2
  exit 1
fi

if [[ -n "${JAVA_HOME:-}" ]]; then
  if [[ ! -x "$JAVA_HOME/bin/javac" ]]; then
    echo "hello-mobile build unavailable: JAVA_HOME does not provide javac" >&2
    exit 1
  fi
else
  javac_bin="$(command -v javac || true)"
  if [[ -z "$javac_bin" ]]; then
    echo "hello-mobile build unavailable: a JDK with javac is required (set JAVA_HOME)" >&2
    exit 1
  fi
  export JAVA_HOME="$(cd "$(dirname "$javac_bin")/.." && pwd)"
fi

sdk_root="${ANDROID_SDK_ROOT:-${ANDROID_HOME:-}}"
if [[ -z "$sdk_root" || ! -d "$sdk_root/platforms/android-35" ]]; then
  echo "hello-mobile build unavailable: ANDROID_SDK_ROOT must provide platform android-35" >&2
  exit 1
fi

export ANDROID_SDK_ROOT="$sdk_root"
export ANDROID_HOME="$sdk_root"
"$gradle_bin" --no-daemon :app:assembleDebug

apk="$fixture_root/app/build/outputs/apk/debug/app-debug.apk"
if [[ ! -s "$apk" ]]; then
  echo "hello-mobile build failed: Gradle produced no debug APK at $apk" >&2
  exit 1
fi

output="${1:-$repo_root/scenarios/device-control/fixtures/hello-mobile.apk}"
mkdir -p "$(dirname "$output")"
cp "$apk" "$output"
printf 'hello-mobile APK: %s\n' "$output"
