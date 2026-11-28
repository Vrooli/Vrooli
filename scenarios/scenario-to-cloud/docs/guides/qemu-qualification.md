# QEMU qualification

The QEMU lane is a real-host proving ground for all 54 applicable case/lane cells. It validates systemd, host-bound credential wrapping, the privilege broker, reset behavior and failure recovery. Containers do not replace this proof.

## Governed artifacts

| Artifact | Path | Role |
|---|---|---|
| Lane manifest | `certification/lanes/qemu.json` | The applicable cells, fixtures, faults, architectures, budgets and acceptance rules. |
| Image manifest | `certification/lanes/qemu-images.json` | Approved base images, paths and recorded digests. |
| Readiness observation | `GET /api/v1/instances/readiness` | Read-only host tools, KVM, image and architecture observations. |
| Qualification program | `.vrooli/program-runtime/cloud-qemu-qualification.{json,py}` | Admits the QEMU-applicable matrix cells and records target-bound evidence. |
| QEMU Connect service | `api/qemusvc/` | Acquires, resolves and releases Compute Manager-owned local-QEMU instances. |
| Receipts | `certification/evidence/` | Immutable evidence by case, lane and target ground. |

## Ownership and run contract

Compute Manager is the only local-QEMU provider owner. Its shared provider admission stores a quote and capacity receipt before the local adapter can create a disk or process. A QEMU instance reference carries the Compute Manager ID independently from the Bridge machine ID and the SSH locator.

When the program acquires a guest, it calls the generated `scenario-to-cloud/qemu/acquire` binding and uses the returned manager-owned reference for deployment creation. Before deployment reads or mutations, it resolves `scenario-to-cloud/qemu/get` and checks that the deployment carries the same manager instance ID. Release is delegated to Compute Manager. The previous Scenario-to-Cloud `vps instance` lifecycle and direct provider implementation have been retired.

Snapshot and reset remain optional test-provider capabilities in Compute Manager. The shared cloud provider interface does not require them.

## Operator path and current gate

No terminal command sequence is required for this lane. QEMU host requirements and approved image readiness must be provided through the normal `vrooli setup` path. Clean-host setup closure is still open because plain `vrooli setup` does not yet expand scenario asset plans.

The 54-cell lane must wait for the capacity owner to review real broker-backed RAM/CPU and disk receipts, process-bound recovery, lease heartbeat, restart reattachment and release after process reaping. The current code and fake-admission tests do not satisfy that review. No QEMU guest or lease has been started for the current CD5 slice, and no showcase claim is made.

After that gate closes, qualification must run against a real guest and record fresh target-bound receipts. Test Genie phases and all 54 cells remain outstanding.
