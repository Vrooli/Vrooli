# Secret Service consumer inventory

Measured 2026-08-24 on the control-plane host. This inventory contains no
credential values.

| Consumer | Owner | Observation | Move to Vrooli encrypted authority? |
|---|---|---|---|
| Vrooli credential authority | Vrooli | The selected backend is `encrypted-file`; it does not require Secret Service for normal resolution. | Already moved; keep as the authority. |
| GNOME Remote Desktop credentials | Desktop service / operator | The pre-migration user-session provider used Secret Service; the safeguard now selects system-mode GNOME Remote Desktop for the automatic path and resolves declared values through Vrooli's credential authority. The vrooli-autoheal RDP check still *reads* the user-session credential state through `grdctl status` (a libsecret client) every 60 s via `hostinventory.ProbeRemoteDesktopCredentials`, so the read path remains a Secret Service consumer even where provisioning has moved. | Provisioning is migrated; the autoheal read path is not. Live provisioning and reboot qualification are recorded in the C2 evidence. |
| GNOME login keyring | Desktop stack | `login.keyring` and `user.keystore` exist under the user keyring directory. The previously unreferenced `login.keyring.corrupt-backup` was retired through the Vrooli control plane after inspection. | No: the keyring file itself is desktop-owned. Migrate individual Vrooli-owned consumers instead. |
| `secret-tool` / Secret Service clients | Desktop and third-party applications | `/usr/bin/secret-tool` and `/usr/bin/gdbus` are present; no value read was attempted. | Only when the owning consumer offers a supported alternative. |

The Vrooli store and the login keyring are therefore separate. A locked or
unreachable login keyring is not evidence that the Vrooli authority is
unavailable, and the Vrooli authority must not unlock or rewrite desktop-owned
entries. The live remote-desktop migration and reboot evidence remain operator
gates because they require the current RDP credential and an announced reboot.
