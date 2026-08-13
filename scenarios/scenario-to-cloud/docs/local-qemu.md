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

Example CLI flow:

```text
scenario-to-cloud vps instance plan --name lane-a --image /var/lib/vm/base.qcow2 \
  --workdir /var/lib/vm/lane-a --profile headless-linux \
  --authorized-key "$(cat ~/.ssh/id_ed25519.pub)"
scenario-to-cloud vps instance create --name lane-a --image /var/lib/vm/base.qcow2 \
  --workdir /var/lib/vm/lane-a --profile headless-linux \
  --authorized-key "$(cat ~/.ssh/id_ed25519.pub)"
scenario-to-cloud vps instance wait-for-ssh <id>
scenario-to-cloud vps instance snapshot <id> clean
scenario-to-cloud vps instance reset <id> clean
scenario-to-cloud vps instance destroy <id>
```

The provider reports a typed readiness error when QEMU or the provisioning
inputs are missing. This is intentional: a container is not an equivalent
fresh-host proof because it cannot validate systemd, host-bound credential
wraps, or the privilege broker.
