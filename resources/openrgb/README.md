# OpenRGB resource

OpenRGB is an opt-in host resource. Its manifest is not a project dependency,
and default Vrooli setup does not install the OpenRGB artifact, launch its
process, inspect controllers, or query its SDK. Operators who choose to use
lighting must explicitly enable and install the resource, then start it through
the resource lifecycle.

This initial implementation declares only the Linux amd64 OpenRGB 1.1
AppImage. The managed process starts headless in Qt offscreen mode and binds
the SDK to `127.0.0.1:6742`. Other operating systems and Linux arm64 are
reported as unsupported until their service, permission, and startup paths are
qualified.

See [operations](docs/OPERATIONS.md) for lifecycle, compatibility, permissions,
update and recovery limits.
