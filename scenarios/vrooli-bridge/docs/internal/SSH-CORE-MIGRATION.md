# Shared SSH core decision record

Phase 8 removed the duplicated `hostkey.go` implementations from Bridge and
scenario-to-cloud. Both callers now use `github.com/vrooli/ssh-core` and pass
an explicit state path, so host-key trust never falls back to an unrelated
operator `~/.ssh/known_hosts` file.

The comparison was resolved as follows:

| Concern | Decision | Reason |
| --- | --- | --- |
| Unknown host | Keep accept-new/TOFU and persist the normalized host:port | Both existing clients depended on first-touch enrollment. |
| Changed host key | Reject and never append a second entry | Prevents a changed server key from being silently trusted. |
| Hashed known-host entries | Keep Bridge's HMAC-SHA1 matching | OpenSSH commonly writes hashed entries; cloud's copy did not support them. |
| RSA host-key algorithms | Keep Bridge's rsa-sha2 expansion | It avoids false mismatches when OpenSSH and x/crypto negotiate different RSA signature names. |
| Host-key review | Keep Bridge's targeted removal operation | Recovery must remove only the reviewed host:port, preserving unrelated trust records. |
| State ownership | Use the caller-supplied path | Bridge needs its state directory; scenario-to-cloud retains its existing SSH directory policy. |

The shared package has its own TOFU/change-key tests. The scenario consumers
retain only compatibility adapters and their existing connection/key APIs; new
host-key behavior must be added to `packages/ssh-core`, never copied into a
scenario.
