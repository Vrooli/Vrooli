# Wireless driver preparation on Ubuntu Noble

`host wireless activation-preflight` joins exact prepared/signed-copy, public
certificate status, firmware and recovery observations without host effects.
The privileged transaction and exact operator sequence are described
in [ACTIVATION-DESIGN.md](ACTIVATION-DESIGN.md). Enrollment, kernel acceptance,
functional radio activation and boot recovery remain separate evidence.

The durable file engine now retains exact original/candidate bytes and
hash/mode/UID/GID, resumes partial publication from an authenticated disk journal,
and reverses committed Wi-Fi/Bluetooth generations in order. Private fixtures
exercise the same journal and radio transaction owners used by the explicitly
elevated normal CLI. Live admission remeasures exact kernel trust, recovery,
firmware precedence, provider identities and namespace ownership. No command
creates or enrolls a key, changes boot selection, or reboots. See the design's
operator sequence for the remaining physical prerequisites.

Focused durable recovery proof is reproducible with:

```sh
go test ./internal/hostwireless -run TestDurableWireless -count=1
```

Set `VROOLI_WIRELESS_DURABLE_FIRMWARE_TEST=1` to include already-governed local
firmware bytes in the private presence/absence and peer-profile fixtures. It
acquires nothing. The existing `TestNativeSignedIndexedActivation` accepts
explicit canonical bundle paths in `VROOLI_WIRELESS_NATIVE_BUNDLES`; adding
`VROOLI_WIRELESS_DURABLE_NATIVE_TEST=1` exercises the durable owner with actual
signed copies/native indexes and fixed synthetic boot fixtures. Optional
`VROOLI_WIRELESS_NATIVE_EVIDENCE_DIR` exports exact public journal intents. Run
native copied-tree proof with ordinary capacity admission for its temporary
working allocation. It never uses the host signing key or live destination.

## MT7927 PCIe Wi-Fi

Run `vrooli host wireless prepare --profile noble-mt7927 --json` to preview the
profile. The host must report Ubuntu Noble, amd64, running kernel
`7.0.0-30-generic`, and PCI device `14c3:7927` with its observed modalias.
Other environments receive an explicit unsupported or unverified result.

The profile consumes the governed SDA host-tool source closure
`source/mt7927-noble-source@35a5dcc71735eb42606be34e313f9eb8148b6a42`.
After staging that closure through SDA, add `--apply` to verify its bytes,
reserve disk capacity, and build in a private directory as an unprivileged
user. The recipe uses the installed matching Ubuntu kernel headers and
configuration, GCC 13, GNU Make, tar, gzip, Git, and kmod.

The Ubuntu source archive is the published Noble HWE
`7.0.0-30.30~24.04.1` source. The embedded patch adapts the 20 ordered upstream
MT7927 PCIe series ending at `35a5dcc71735eb42606be34e313f9eb8148b6a42` to that
source. Preparation reconstructs only mt76 and verifies its 231-file digest
manifest against the independently unpacked full Noble source. The adapted
patch omits the USB-specific change, preserves the existing Connac2
conditions and firmware declarations, and uses the stock Noble DMA and
station-link APIs where their behavior already matches the series.

The consolidated patch also carries the reviewed PCIe recovery delta
`e01bef004cdef6b9efead9f66e3a3f06f1b9fe55242439bcce9b2dcfc49e71d6`.
It checks the initial driver-own handshake before changing DMA or IRQ state,
restores the reset state and TX worker when the later firmware-own handshake
fails, and uses the wiphy-locked interface iterator in suspend and resume.
PCIe unregister keeps MCU completion interrupts available through mac80211
unregister and reset drain. Noble's `disable_work_sync()` then prevents further
reset enqueues before the removal flag is set and NAPI is disabled.
This adapts the upstream cancellation boundary to the exact Noble workqueue
API, including a caller already racing with teardown. The source provenance
is the original PCIe series above and the pinned follow-ups below.
[Driver-own error check](https://github.com/torvalds/linux/commit/d12a54be2604ecd23a1e9ac863583e34034aadde)
[Firmware-own cleanup](https://patch.msgid.link/20260827162115.1449757-1-nbd@nbd.name)
[Suspend/resume iterator](https://patch.msgid.link/20260830064201.92285-3-lucid_duck@justthetip.ca)
[Unregister ordering correction](https://lore.kernel.org/linux-wireless/20260817052840.99478-1-eason.lai@mediatek.com/)

The prepared patch SHA-256 is
`526881cd4d9f6c92b56ee9e29113533c785e9299eefc7c60d029e09b43c88094`.
The build recipe digest is recorded independently in each preparation receipt.

The owner builds four stock modules and verifies their symbol versions before
applying the delta. The candidate must retain the existing exports and match
every imported symbol CRC. It must select the observed PCI aliases with the
expected Wi-Fi firmware declarations in a copied module tree. The copied
tree must index without warnings, select the same providers for every other
module, and recover its original file digests, links, and selected paths after
overlay removal.

A successful run publishes an owner-only bundle under
`~/.vrooli/artifacts/host-wireless/noble-mt7927/` with four unsigned modules,
their SHA-256 digests, ABI evidence, and copied-tree logs in `receipt.json`.
Each run has a distinct directory.

## Activation qualification

The shared mt7925 init and reset follow-ups check `mt76.bus_hung` before
retrying hardware operations. Noble already has that field for SDIO. The
reviewed MT7927 PCIe path has no corresponding producer of the hung state.
The upstream USB timeout producer and reset hunk apply to USB. Applying
only the shared guards would leave the MT7927 PCIe failure path unqualified.
The upstream MT7921 PCIe AER change demonstrates why PCIe detection and
shared DMA safety need review together. It targets MT7921. Qualify the
MT7927 PCIe path on a maintained distro build or verify its own containment
and recovery before live use. [Init guard](https://github.com/torvalds/linux/commit/7e87dc3b2157967b14ded97129e677a764573ec7)
[Reset guards](https://github.com/torvalds/linux/commit/c781b74c3fd0c24db33ca7cfc0b9a3857d623543)
[MT7925 PCIe driver](https://github.com/torvalds/linux/blob/master/drivers/net/wireless/mediatek/mt76/mt7925/pci.c)
[MT7921 PCIe AER scope](https://kernel.googlesource.com/pub/scm/linux/security/vulns/+/ffaa1b4ec1ea8813a90a5a6f9f577958c848a33c/cve/published/2026/CVE-2026-90379.mbox)

Activation also requires the separate governed Wi-Fi firmware, module
signing and trust, a host transaction with selectable boot recovery, and a
coordinated test of binding, firmware initialization, association, traffic,
and wired access. Bluetooth qualification remains separate.

## MT6639 USB Bluetooth

Run `vrooli host wireless prepare --profile noble-mt6639 --json` for the
Bluetooth preview. This profile requires the same Ubuntu ABI and an observed
USB `0489:e13a` Bluetooth HCI interface modalias. With `--apply` it uses the
same governed Noble source closure and publishes the unsigned `btusb.ko`
and `btmtk.ko` bundle under
`~/.vrooli/artifacts/host-wireless/noble-mt6639/`.

The recipe reconstructs the exact 51-file stock Bluetooth subtree and builds
both stock modules before applying the three-file Bluetooth delta. It verifies
the stock exports and import sets, checks candidate CRCs and vermagic, and
inspects the compiled USB quirk. The candidate must add only the exact
`0489:e13a` quirk, with MediaTek and wideband flags, while retaining the
shared exported firmware helper and stock dependency relationships.
The copied tree must select both overrides for the observed HCI modalias,
index without diagnostics, and recover its original hashes and lookups
after removing the overlay. [MT6639 upstream support](https://github.com/torvalds/linux/commit/28b7c5a6db74e9305c6cbcbe52f259ff1cf85158)
[USB ID upstream](https://github.com/torvalds/linux/commit/bfe11e17b3005e7458b46530e9d27e1d36924541)
[ISO alternate-setting fix](https://github.com/torvalds/linux/commit/e4f6bc7f8222a9e40283d3fc6c4c4c656d79a48e)

Bluetooth preparation verifies the declared firmware path
`mediatek/mt7927/BT_RAM_CODE_MT6639_2_1_hdr.bin`. The firmware bytes and
their eligibility remain open. Activation requires authenticated eligible
firmware, review of later transfer and reset changes, signing and trust,
selectable boot recovery, controller initialization, BlueZ peer discovery
and pairing, and post-resume operation.

## Activation and recovery preview

Run `vrooli host wireless activation-plan --profile noble-mt7927 --json`
or select `noble-mt6639` for Bluetooth. Use `--bundle <generation>` to
verify the exact prepared receipt and artifact digests. Use
`--recovery-kernel <release>` to inspect one distinct installed recovery
candidate identified by the preview. This is a read-only action.

The plan records the current ABI, Secure Boot state, exact selected stock
module paths and digests, wired IPv4 default-route evidence, NVIDIA module
providers, dependency indexes, and readable boot files. An installed kernel,
initramfs, and module tree identify a recovery candidate. A selectable
known-good stock boot entry and console access require their own evidence.
An alternate menu entry that uses the same ABI also uses the same module
tree. The recovery candidate must have a distinct ABI.

The copied-root rehearsal verifies the original module and protected
boot, wired and NVIDIA files before publishing the exact candidate
directory. It verifies an existing exact overlay on retry, discards a
partial staging attempt, and removes only an exact owned overlay during
rollback. An unrelated overlay remains in place and causes a conflict.
The prepare recipes separately exercise native dependency index generation
and rollback in a copied module tree. The private transaction exercises
the file boundary and abort/retry behavior. The privileged owner journals exact
live overlay, firmware and native index publication separately from radio
activation; it refuses effects until its real prerequisites pass.

Before activation, review firmware and driver recovery for the selected
profile. Verify the recovery boot path independently. Bind signed module
digests to the reviewed unsigned bundle, verify the signing key against
the running kernel's trusted keys, and preserve the existing boot selection.
Signing is separate from enrollment. On Ubuntu, the Machine Owner Key
enrollment decision belongs to the system owner at the console. Boot
selection is separate from module overlay publication. A one-time GRUB
selection changes the next boot choice and does not prove that the selected
kernel can boot and recover access. [Ubuntu Secure Boot](https://documentation.ubuntu.com/security/security-features/platform-protections/secure-boot/)
[Kernel module signing](https://www.kernel.org/doc/html/latest/admin-guide/module-signing.html)
[GRUB next entry](https://www.gnu.org/software/grub/manual/grub/html_node/next_005fentry.html)

## Verify signed copies

Given an existing unsigned preparation and a directory containing exactly
its signed module names, run:

```bash
vrooli host wireless verify-signed-copy --profile noble-mt7927 \
  --bundle <generation> --signed-dir <directory> --cert <public-cert> --json
```

The verifier accepts one public X.509 certificate in PEM or DER. It checks
the reviewed SHA-256 `sign-file` trailer and detached CMS signature format,
compares the unsigned payload of every copy against its preparation receipt,
and asks OpenSSL to verify the signature using the supplied certificate as
the sole signer source. The receipt records the preparation receipt digest,
certificate DER digest and each signed and unsigned module digest. Kernel
trust and activation remain separate fields and both remain false in this
read-only action. [Linux signature format](https://github.com/torvalds/linux/blob/v7.0/scripts/sign-file.c)
[OpenSSL CMS verification](https://docs.openssl.org/3.0/man1/openssl-cms/)

Before a live trial, prove that the exact certificate is trusted by the
running kernel and verify a distinct selectable stock recovery entry with
console access. If enrollment is required, the machine owner confirms that
specific public certificate at the console after reviewing the recovery path.
The Wi-Fi and Bluetooth activation prerequisites above still apply.

## Private signed and indexed transaction

The control-plane rehearsal authenticates each signed copy again, binds its
payload to the reviewed preparation, and verifies that the copied kernel
symbol identity matches the preparation. It builds an unpublished private
generation containing the complete stock module tree and exact signed
overlay. Native `depmod` recreates the closed index set in that generation.
The owner verifies all observed aliases, candidate providers, Ethernet and
NVIDIA providers, unchanged modules' dependency closures, and every file and
mode outside the index set.

A single atomic private selector publishes the completed generation.
Cancellation, partial index output, unknown index files, changed protected
bytes or modes, and signature drift leave the stock generation selected.
Retry verifies the selected generation. Rollback selects the original
immutable stock generation, recovering its module and index identities
together. An ambiguous selector or changed selected index fails validation.

The native fixture uses explicitly supplied prepared bundles and synthetic
boot recovery files. It runs through `TestNativeSignedIndexedActivation`
with `VROOLI_WIRELESS_NATIVE_BUNDLES` set to a JSON map of profile names
to canonical bundle directories. Its exact signing keys and copied roots
are temporary. Kernel trust, real stock boot recovery, firmware eligibility,
and radio operation retain their independent proof requirements. Live
system-path publication and durable crash recovery use the root-admitted
journal owner described in [ACTIVATION-DESIGN.md](ACTIVATION-DESIGN.md).

## ASUS MT6639 local firmware qualification

The closed SDA source profile is
`source/asus-mt6639-local-source@fc088966f2357af65f16f48178e047fa553ed5eabad34bcf0c7ab4ce2033435a`.
It stages only the exact official ASUS ZIP in the operator's artifact store.
The source descriptor records the vendor terms evidence and the unresolved
package-specific Linux-use and redistribution questions.

Run `vrooli host wireless qualify-firmware --profile noble-mt6639 --json`
after governed staging. The owner requires the observed ASUS board and USB
HCI identity, verifies the official archive pin, and reads only its exact
Bluetooth container. It validates the versioned offset table,
complete payload coverage, MT6639 header and section bounds, and additive
section checks. The resulting opaque image retains all nine sections.
The reviewed driver selects five Bluetooth sections by the low mode byte.

With `--apply`, the owner publishes the exact 571,349-byte image and a
qualification receipt under `~/.vrooli/artifacts/host-wireless-firmware/`.
The image SHA-256 is
`2135f2c4220cfa6e8eb9fdf430517098c13b862a95844bbff0153240a768efa8`.
This is a machine-local data artifact. Static qualification leaves
activation and redistribution eligibility false.

The official package targets this exact board and USB interface, and ASUS
documents downloaded archive extraction. The inspected archive and setup
resources supplied no standalone EULA. Package-specific Linux loading
entitlement remains uncertain. Local static qualification follows the
operator's hardware-support request while recording that uncertainty.
Firmware installation and device transfer still need their own qualified
activation and recovery transaction. [ASUS package](https://www.asus.com/motherboards-components/motherboards/proart/proart-x870e-creator-wifi/helpdesk_download?model2Name=ProArt-X870E-CREATOR-WIFI)
[ASUS extraction instructions](https://www.asus.com/support/faq/1045127/)
[ASUS site terms](https://www.asus.com/terms_of_use_notice_privacy_policy/official-site/)
