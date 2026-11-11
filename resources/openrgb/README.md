# OpenRGB resource

OpenRGB is an opt-in host resource. Its manifest is not a project dependency,
and default Vrooli setup does not install the OpenRGB artifact, launch its
process, inspect controllers, or query its SDK. Operators who choose to use
lighting explicitly choose a source and prepare the resource, then enable it
through the resource lifecycle.

Exact OpenRGB 1.1 artifact profiles cover Linux amd64/arm64, macOS Intel/Apple
Silicon and Windows x64. The managed process is configured headlessly with a
loopback SDK endpoint. macOS and Windows remain `unqualified` until native
service, permissions, startup and hardware behavior is observed; Linux arm64
runtime proof is separate from the existing amd64 evidence. Windows ARM64 and
desktop bundling remain unsupported.

See [operations](docs/OPERATIONS.md) for lifecycle, compatibility, permissions,
update and recovery limits.

Native artifact and prerequisite contracts: [native release profiles](docs/NATIVE-RELEASE-PROFILES.md). Artifact/ABI proof does not imply native runtime or hardware qualification.
