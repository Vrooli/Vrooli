# Upstream source and local resource changes

- Repository: https://github.com/Bahmni/bahmni-docker
- Tag: `1.0.0-lite`
- Commit: `8f900af235eaad4a0f7309f32767db8837b44265`
- License: AGPL-3.0; original license is preserved in `upstream/LICENSE`.

The selected upstream `bahmni-lite/docker-compose.yml` and its environment template are preserved under `upstream/`. The vendored Compose file drops the obsolete `version` key (Compose v2 ignores it and emitted a warning on every status call). The local override adds resource and instance labels to containers, networks and volumes; makes the default network internal; and removes upstream's fixed public host ports. The resource owner creates a private per-instance `.env` file from the scrubbed template and replaces every database/admin secret before start. The `images.json` profile pins the 14 image repositories resolved by the `bahmni-lite` profile to digest references observed from that exact release tag.

The upstream `COMPOSE_PROFILES=emr` default omits Crater billing; the maintained resource explicitly selects `bahmni-lite` to include its clinic billing UI. Optional SMS, Metabase/Mart, and Atomfeed Console profiles remain disabled.
