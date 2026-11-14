# Exact wireless activation and recovery

This is the implemented control-plane route for W1/W2. `activation-preflight`
joins read-only evidence; `sign`, `publish-signed`, `record-recovery`, `stage`,
`activate`, `status`, `recover` and `rollback` expose the typed owners through
`vrooli host wireless`. Effects require explicit apply and the corresponding
fresh prerequisites. Signing reuses the existing module-only MOK through its
fixed native helper; it never creates or enrolls trust. The live adapter was
implemented and exercised through private/injected fixtures, not operated on
this host. Copied-root proof does not establish physical boot or device recovery.

## Implemented durable file owner

`durable_wireless_linux.go` consumes a fully authenticated/indexed generation,
public certificate and optional exact firmware closure. The Linux IO owner uses
descriptor-relative openat2 with no symlinks/magic links, rejects hardlinks and
qualifies every state ancestor and private transaction directory. An exclusive
cross-profile file lock protects a random immutable journal lineage. The private
constructor refuses the live host root and its aliases. The separate live
constructor requires an internally issued, short-lived root namespace capability,
actual elevated credentials, verified usr-merge/header paths and all fresh
operation prerequisites. The caller holds native package locks and the shared
wireless lock during publication. No arbitrary root/path or executable flag is
accepted by the normal command.

Before any target mutation the owner fsyncs original/candidate blobs, exact
hash/mode/UID/GID and absent/present states, public cert, immutable intent and
active lineage. Reopen authenticates signed payloads again, verifies root and
provider/ABI/recovery identities, classifies each target as exact original or
candidate and preserves foreign drift. Resume syncs already-completed entries
and directory ancestors before committing. Late rollback drift is checked again
per target and immediately before removal. Cancellation uses a fresh bounded
compensation context. Close retains journals and undo bytes.

A fresh combined native index closure can succeed a committed peer profile.
The child protects the peer's signed modules and firmware and records the
parent's exact intent digest. Reverse rollback restores the committed parent
selector, then permits exact stock rollback. It cannot replace combined indexes
with a stock-only peer selection. Unit fixtures cover both profiles together,
including the actual pinned Wi-Fi and ASUS Bluetooth firmware under private
roots. Native exact-bundle proof also interrupts a real generated index publish,
reopens, resumes and restores all original file bytes/mode/UID/GID while retaining
the actual Atlantic/NVIDIA providers. Boot data in that native fixture is
explicitly synthetic; it is not proof of a usable recovery boot.

Persistent signed bundles authenticate the exact unsigned receipt and every
signed payload before publication or use. Firmware admission evaluates actual
custom, versioned, compressed and initramfs precedence and refuses a conflicting
early image instead of regenerating it speculatively. Radio activation captures
fresh stock module parameters and boot identity before durable intent. Partial
candidate loading requires radio rollback; observation errors cannot authorize
file-only undo. A child remains the active lineage through interrupted stock
runtime restoration, including reopen after durable stock_restored but before
parent publication. The implementation never treats local-use
terms uncertainty as a prohibition or an extra approval gate. The source-only
effects approved for SDA constrain that acquisition operation, not every later
operator-authorized host operation.

## Owners and exact inputs

Reuse `prepare.go` for exact source/ABI/profile receipts, `signed_copy.go` for
authenticated appended signatures and exact stripped payloads, and
`indexed_activation.go` for full-stock-tree native depmod, alias/provider checks
and exact rollback. `hostinventory` owns portable observation and later function
evidence. `hostreqkit.WithSudo` owns elevation policy;
`RunPrivilegedCommandWithOutput` accepts a caller-owned bounded context executor.
`hostreqkit/modules` owns autoload/options and generic modprobe, but no signed
overlay or journal. The privilege broker has no wireless action. Never broaden
an unrelated broker subject to accept arbitrary paths or shell commands.

Ubuntu `shim-signed` owns `/usr/sbin/update-secureboot-policy` and
`/var/lib/shim-signed/mok/MOK.der` plus its protected private counterpart.
Installed `/usr/lib/shim/mok/openssl.cnf` specifies code signing and module-only
EKU `1.3.6.1.4.1.2312.16.1.2`. Reuse this identity rather than creating a key.
Observed public DER SHA-256:
`377ca8643eb2816c3996f1df1e0a052d9f1d6fa17bd39a157c8f516966351c48`.
No private key was inspected. Current mokutil reports not enrolled and cannot
access the trusted keyring. Exit 0 means **not enrolled** for this test command;
other nonzero results include enrolled, pending and blocked. Preflight parses
explicit bounded status lines. A positive keyring lookup compares SKID and is
reported as an observation, not cryptographic kernel acceptance.

| Profile | Unsigned generation | Receipt SHA-256 |
| --- | --- | --- |
| noble-mt7927 | 1890979824 | 1515aaeb5c54f7e38b70b971e8878632b02f22d50fb605e415dad4d31a00307f |
| noble-mt6639 | 362036278 | 26c3bbb37033fe91fd665066df83f2ab6682c397cd9f35aa71fc5cb155314f19 |

Wi-Fi uses firmware from governed `mt7927-source` generation
`5467fd1237391e8b76625c0fd6fad3285d7ff930`, not its rejected broad driver recipe.
Preflight reports both exact firmware hashes. Bluetooth uses the qualified local
ASUS artifact under `host-wireless-firmware/asus-mt6639/` at source generation
`fc088966f2357af65f16f48178e047fa553ed5eabad34bcf0c7ab4ce2033435a`:
`BT_RAM_CODE_MT6639_2_1_hdr.bin`, SHA-256
`2135f2c4220cfa6e8eb9fdf430517098c13b862a95844bbff0153240a768efa8`.
Package-specific Linux use remains uncertain; redistribution is unqualified.
That uncertainty does not establish a prohibition of owner-local research/use.

## Reviewable implementation sequence

1. **Snapshot:** require explicit profile, immutable bundle and distinct recovery
   release. Revalidate machine/device, ABI/symbol CRCs, full stock reverse
   dependencies, candidate/stock hashes, loaded radio consumers, default wired
   routes/provider, NVIDIA providers, Secure Boot and lockdown. Record boot ID
   and observation time. A snapshot is not an authorization token; drift, reboot
   or kernel upgrade invalidates it.
2. **Recovery:** with coordinated elevation, read/hash current and fallback
   kernel/initramfs, GRUB config/environment, depmod policy and boot symlinks.
   Extend the existing GRUB owner to resolve exact menu IDs; its current parser
   handles kernel command-line parameters, not generated boot menus. Verify fallback
   Atlantic Ethernet, NVIDIA modules and firmware. Require reachable console
   and a successfully selected stock boot before calling it known good. Current
   hidden zero-timeout GRUB and symlinks to -42, which lacks NVIDIA modules,
   prevent a generic reboot recommendation. File existence is insufficient.
3. **Sign copies:** after supervisor coordination, copy the exact reviewed files
   into a protected signing workspace. Recheck receipt and payload hashes, then
   invoke the installed exact-ABI `scripts/sign-file sha256` with the existing
   protected MOK key and exact public DER. The key path is an argument to the
   signer; key contents/passwords never enter logs, repository, receipts or
   command arguments. Never modify the canonical unsigned bundle. Authenticate
   every result through `VerifySignedCopy`. Publish the closed signed bundle
   atomically with capacity admission, artifact lock and signer identity.
4. **Trust:** inspect exact enrolled/pending/blocklist state first. The installed
   `update-secureboot-policy --enroll-key` calls `validate_actions`, which exits
   when no DKMS modules are installed; it is not an assured route here. Once
   recovery and console are proven, an explicit human
   `sudo mokutil --import /var/lib/shim-signed/mok/MOK.der` queues that exact
   public cert and asks interactively for an enrollment password. Complete
   MokManager enrollment at the coordinated reboot. Preserve Secure Boot; no
   disable-validation, reset, trust-mok, implicit key creation, automatic reboot
   or password argument. After the intended stock boot, reobserve exact ABI,
   cert, enrollment and running trust policy. Enrollment alone does not prove
   module acceptance; the later coordinated selected-module load observes that.
5. **Durable stage owner:** implement a typed internal transaction taking only
   closed profile/bundle identities, authenticated signatures and fresh recovery
   evidence. Privileged policy recomputes evidence instead of trusting JSON.
   Use hostreqkit elevation with bounded context execution; no private repair
   script. Hold one wireless mutation lock across Wi-Fi and BT and reject kernel
   package drift. A root-owned journal under
   `/var/lib/vrooli/host-wireless/transactions/<id>` records original absent/
   present states, hash/mode/owner, exact signed/cert/symbol identities, all ten
   original depmod indexes and alias/provider evidence. Fsync journal, backups
   and containing directories before the first effect. Verify all payloads first.
6. **Publication:** retain stock modules. Publish only the exact profile overlay
   under `updates/vrooli-mt7927` or `updates/vrooli-mt6639`. Resolve usr-merge at
   admission and pin directory descriptors; reject symlinks/foreign entries.
   Resolve actual firmware lookup precedence, including kernel-release/custom
   paths, compressed files and initramfs. A missing raw .bin does not prove no
   firmware exists. Journal existing firmware or absence. Initramfs change needs
   its own proven necessity and exact rollback; never regenerate speculatively.
   Generate native indexes in an exact copied stock tree, validate full provider/
   alias/reverse dependency closure, then journal each final atomic rename. Live
   kmod has no whole-index-directory selector protocol: the private-root symlink
   cannot make multiple live index replacements atomic. Retain that intermediate
   state limitation, prohibit intentional radio activation until completion and
   classify ambiguous rename/fsync as recovery_required. Recheck Ethernet/NVIDIA
   files, provider selections and routes immediately after publication.
7. **Reopen/abort/retry:** durable states are admitted, backups_durable,
   overlay_published, indexes_published, device_change_started, observed,
   rollback_required and rolled_back. Every effect has durable intent and a
   measured result. Reopen classifies exact original/candidate hashes rather
   than trusting the last journal field. Before device changes, abort restores
   exact original indexes, firmware and overlay states, preserving foreign data
   and failing closed on drift. Cancellation after an effect runs bounded
   compensation using a fresh cleanup context. Interruption retains evidence
   and an actionable recovery request. Retry resumes only unambiguous state.
8. **Activate/observe:** separate explicit coordinated radio effect from file
   publication. Never unload serving Ethernet/GPU, reboot automatically or
   globally change rfkill. Wi-Fi needs exact PCI function and reviewed PCIe loss
   containment. BT needs its exact HCI interface and all btusb consumers; a global
   btusb unload or USB reset may affect siblings. Preserve diagnostic logs before
   compensation. File rollback does not undo firmware downloaded into a device
   or restore a lost PCIe bus; native reset or coordinated stock boot must prove
   that recovery independently. Require association/traffic or discovery/pairing,
   plus resume and physical recovery evidence. Activation is not boot-safety proof.

## Qualification before exposing a live mutation command

Extend the existing copied/injected transaction. Interrupt before/after every
journal write, fsync and final rename and reopen using retained disk state.
Exercise absent/existing firmware, foreign overlay, concurrent profiles,
post-sign tamper, wrong cert/bundle, keyring ambiguity, kernel/provider drift,
compressed/override firmware precedence, canceled depmod, partial index publish
and process death. Require exact SHA/mode/owner rollback and unchanged unrelated
tree/provider/route identities. Keep effects injected until privileged policy
and durable reopen pass independent adversarial review. Do not advertise an
unproven live route to make an eligibility boolean true.

## Safe current entrypoint

```sh
go run ./cmd/vrooli host wireless activation-preflight \
  --profile noble-mt7927 --bundle 1890979824 \
  --recovery-kernel 7.0.0-29-generic \
  --cert /var/lib/shim-signed/mok/MOK.der --json
```

For BT use profile `noble-mt6639` and bundle `362036278`. Add `--signed-dir` only
for retained actual signed copies; ephemeral test signatures are not activation
inputs. `--apply` is rejected. No preflight field authorizes effects.

Primary references: [Ubuntu Secure Boot](https://documentation.ubuntu.com/security/security-features/platform-protections/secure-boot/),
[kernel module signing](https://www.kernel.org/doc/html/latest/admin-guide/module-signing.html),
[mokutil 0.6 test_key and print_skip_message](https://github.com/lcp/mokutil/blob/0.6.0/src/mokutil.c).
Installed shim policy/config and the public certificate were read directly.

## Coordinated operator sequence

These commands describe the implemented route, not authorization to operate the
current host from an agent turn. First adopt the exact independently qualified
root CLI candidate using the normal human rebaseline workflow. Review its source
contract and the two preparation receipt pins above. Select a real distinct
stock GRUB entry with working wired access and all current NVIDIA providers.
The installed partial -42 tree lacks NVIDIA and is not qualified by its presence
alone. Boot selection and physical console/reboot coordination remain with the
operator; none of the wireless commands writes GRUB or initiates a reboot.

At the active physical VT on that independently booted fallback, record it:

```sh
sudo vrooli host wireless record-recovery --profile noble-mt6639 \
  --recovery-kernel <exact-stock-release> --recovery-entry <exact-grub-selector> --json
sudo vrooli host wireless record-recovery --profile noble-mt6639 \
  --recovery-kernel <exact-stock-release> --recovery-entry <exact-grub-selector> --apply --json
```

Return to the exact target `7.0.0-30-generic` through the separately coordinated
boot. Enrollment, if needed, is a separate machine-owner decision about the exact
existing public MOK DER fingerprint above. No command here creates a key or
silently imports/enrolls trust. MOK enrollment alone is insufficient: the stage
owner cryptographically verifies the exact certificate against a trusted running
kernel key and checks revocation access.

For Bluetooth, review the sign preview before explicitly requesting signing:

```sh
vrooli host wireless sign --profile noble-mt6639 --bundle 362036278 \
  --cert /var/lib/shim-signed/mok/MOK.der \
  --receipt-sha256 26c3bbb37033fe91fd665066df83f2ab6682c397cd9f35aa71fc5cb155314f19 --json
sudo vrooli host wireless sign --profile noble-mt6639 --bundle 362036278 \
  --cert /var/lib/shim-signed/mok/MOK.der \
  --receipt-sha256 26c3bbb37033fe91fd665066df83f2ab6682c397cd9f35aa71fc5cb155314f19 --apply --json
```

Use the returned immutable signed-bundle digest, never an ephemeral fixture:

```sh
sudo vrooli host wireless stage --profile noble-mt6639 --bundle 362036278 \
  --signed-bundle <returned-signed-receipt-sha256> \
  --receipt-sha256 26c3bbb37033fe91fd665066df83f2ab6682c397cd9f35aa71fc5cb155314f19 \
  --recovery-kernel <exact-stock-release> --recovery-entry <exact-grub-selector> --json
```

Only after every measured prerequisite passes, repeat that exact stage request
with `--apply`. It publishes exact files/indexes without loading radio code and
returns a random transaction ID. Previews remain read-only. In the coordinated
radio window, inspect activation before adding `--apply`:

```sh
sudo vrooli host wireless activate --profile noble-mt6639 \
  --transaction <returned-transaction-id> --maintenance-window --json
sudo vrooli host wireless activate --profile noble-mt6639 \
  --transaction <returned-transaction-id> --maintenance-window --apply --json
sudo vrooli host wireless status --profile noble-mt6639 \
  --transaction <returned-transaction-id> --json
```

Activation checks fresh exact stock/runtime/boot and shared driver consumers,
then journals intent before changing only that radio cohort. It observes binding
and controller/interface readiness; a paired connected peer or Wi-Fi traffic
remains separate functional evidence. Passive Bluetooth observation uses bounded
DBus calls with autoactivation and interactive authorization disabled. Resume,
device-loss and physical recovery are not inferred from a successful load.

On the target kernel, `rollback` with the exact profile/transaction and explicit
`--maintenance-window --apply` performs bounded radio compensation and exact file
undo. `recover --apply` permits file-only undo only when no candidate is loaded
and no unfinished radio intent exists. On the independently proven fallback,
that same `recover --apply` instead rechecks its exact proof, boot image, wired
and GPU providers and restores only the non-running target's journaled files.
It never observes, unloads or loads the fallback's radio modules. Unknown file
drift is preserved and reported. Child lineage is retained until durable stock
restoration, so reopen/retry can complete before returning to its parent.

For Wi-Fi substitute profile `noble-mt7927`, bundle `1890979824` and its exact
receipt pin above. Its PCIe bus-loss containment and physical recovery evidence
remain open; the command route does not claim that successful binding proves
those behaviors or a boot-safe result. Qualification must preserve the wired
session and independently usable fallback throughout. No live signing, staging,
enrollment or radio operation was performed as part of these source proofs.
