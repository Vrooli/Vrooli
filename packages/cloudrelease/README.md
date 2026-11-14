# cloudrelease

The one definition of a cloud release identity.

scenario-to-cloud builds and publishes a release manifest; `vrooli cloud-target release
verify|stage` (`internal/cloudtarget`) verifies it before extraction. Both sides import the
manifest shape, canonical JSON form and digest rule from here, so there is exactly one way
to compute a release digest.

Governed by `.vrooli/package.json`; see [package governance](../../docs/package-governance.md).
