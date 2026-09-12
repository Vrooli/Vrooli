# Desktop update governance

Desktop update implementation belongs to `scenario-to-desktop`. This page
records the deployment-manager boundary so update behavior is not duplicated
between governance and the Electron packager.

## Governance responsibilities

deployment-manager owns:

- release identity and promotion state;
- the source revision and target profile associated with a release;
- release-manifest trust and artifact checksums;
- channel promotion policy;
- evidence that an update was built and exercised.

The desktop ramp owns update-provider configuration, updater metadata,
download/install behavior, restart, migration, and rollback behavior.

## Required release properties

A promotable desktop update must:

- identify the target OS and architecture;
- verify the artifact against trusted release metadata;
- preserve user data;
- run compatible migrations;
- recover from interrupted download or installation;
- record update detection, download, replacement, relaunch, and failure events;
- avoid treating an unavailable update server as an application failure.

Update evidence must be kept separate from ordinary package/build evidence.

## Implementation reference

Read [scenario-to-desktop auto-updates](../../../scenario-to-desktop/docs/guides/AUTO_UPDATES.md)
for provider configuration, channels, metadata, and troubleshooting. Read the
[managed release authority](../../../../docs/configuration/release-authority.md)
for release-manifest trust.
