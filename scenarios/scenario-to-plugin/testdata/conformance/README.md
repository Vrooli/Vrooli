# Permanent conformance fixtures

These fixtures are intentionally unsafe or drifted. They are kept as
regression inputs for the fail-closed conformance gate and must never be
packaged as release content.

| Fixture | Expected finding |
| --- | --- |
| `drifted-command/SKILL.md` | `PLG-CONF-DRIFT` |
| `angle-frontmatter/SKILL.md` | `PLG-CONF-ANGLE` |
| `hidden-unicode/SKILL.md` | `PLG-CONF-UNICODE` |
| `unrestricted-tools/SKILL.md` | `PLG-CONF-TOOLS` |
| `mutable-download/install.sh` | `PLG-CONF-INSTALL-PIN` |
| `missing-checksum/install.sh` | `PLG-CONF-INSTALL-SUM` |
| `privileged-install/install.sh` | `PLG-CONF-INSTALL-PRIV` |
| `outside-prefix/install.sh` | `PLG-CONF-INSTALL-PRIV` |
