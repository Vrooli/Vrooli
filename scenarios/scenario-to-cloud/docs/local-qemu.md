# Disposable local-QEMU hosts

`scenario-to-cloud vps instance` owns the disposable Linux validation lane. It
uses the same Postgres repository as deployments, so an instance record and
its disk lifecycle cannot silently diverge into a second state store.

The host tools are declared as `qemu` and `cloud-localds` and must be installed
through Vrooli host requirements. Do not install them with ad-hoc
package-manager commands. They are separate declarations because
distributions ship the emulator and the cloud-init image utility in different
packages. The provider also requires a readable Linux image, QEMU/KVM, a
non-root VM username, and an authorized SSH key. The two profiles are:

- `headless-linux`: serial console and no desktop display.
- `desktop-linux`: a graphical display for desktop-session validation.

Inspect and install the scenario-owned host requirements through the selected
scenario setup path. Global setup status does not include these declarations:

```text
vrooli setup --scenarios scenario-to-cloud --dry-run
vrooli setup --scenarios scenario-to-cloud --sudo-mode ask
```

The dry run is read-only. The apply command is the only supported elevation
boundary for these tools; it may require operator authentication.

Example CLI flow:

```text
scenario-to-cloud vps instance plan --name lane-a --image /var/lib/vm/base.qcow2 --workdir /var/lib/vm/lane-a --profile headless-linux --authorized-key "$(cat ~/.ssh/id_ed25519.pub)"
scenario-to-cloud vps instance create --name lane-a --image /var/lib/vm/base.qcow2 --workdir /var/lib/vm/lane-a --profile headless-linux --authorized-key "$(cat ~/.ssh/id_ed25519.pub)"
scenario-to-cloud vps instance wait-for-ssh "$INSTANCE_ID"
scenario-to-cloud vps instance snapshot "$INSTANCE_ID" clean
scenario-to-cloud vps instance reset "$INSTANCE_ID" clean
scenario-to-cloud vps instance destroy "$INSTANCE_ID"
```

The provider reports a typed readiness error when QEMU or the provisioning
inputs are missing. This is intentional: a container is not an equivalent
fresh-host proof because it cannot validate systemd, host-bound credential
wraps, or the privilege broker.

`instance create` creates a copy-on-write `disk.qcow2` inside the requested
work directory and boots that owned disk. The source image is never mutated by
snapshot, reset, or destroy. Destroy stops the VM and removes the owned disk
and cloud-init artifacts; the source image remains available for the next
fresh-host run.

## Readiness report

`GET /api/v1/instances/readiness` (`api/instance.Readiness`) reports the lane
without running anything: each declared tool with its `hostTools` owner, KVM
usability, the recorded base images from `certification/lanes/qemu-images.json`
(add `?verify=1` to hash them), the `amd64`/`arm64` execution state
(`kvm`, `tcg` or `unavailable`), every limitation with its external-input row,
and one `next_action`. An unready lane is a 200 report with `state:
"unavailable"`; the qualification program refuses on it. Missing tools name
`vrooli setup` as the next action; the provider never installs them.

## Certification lane

The QEMU certification lane (matrix cases whose lanes include `qemu`) is
declared in `certification/lanes/qemu.json` and driven by the governed program
`scenario-to-cloud.cloud-qemu-qualification`. Operator steps, admission
refusals and receipt reading are in `docs/guides/qemu-qualification.md`.
