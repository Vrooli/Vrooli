# hello-mobile fixture

This is the shared Android fixture named by the device-control physical
conformance contract. It is intentionally dependency-light: the app uses
platform APIs only and exercises the contract surfaces directly:

- `hello-mobile-input` persists text through backgrounding and process death;
- `hello-mobile-notification` creates a semantic notification target;
- `vrooli-hello://home` is a cold/warm deep link;
- notification permission denial is safe;
- connectivity state is explicit and bounded;
- rotation uses the same activity state and input layout.

Build the debug APK with a JDK, Gradle 8.10.2, and an Android SDK with platform
35 and build-tools installed:

```bash
bash build-fixture.sh
```

The copied APK is a generated validation artifact and should not be committed.
The fixture contract remains at
`scenarios/device-control/fixtures/hello-mobile.contract.json`.
