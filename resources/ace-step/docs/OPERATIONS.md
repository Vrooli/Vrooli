# ACE-Step Operations

`ace-step` is a `managed-service` resource. The control plane owns the
standalone Python runtime, hash-pinned wheels, source archive, and model data;
the service owns only model loading and one-take generation.

The pinned ACE-Step model uses Transformers remote code. The reviewed source
and custom model modules are pinned to source commit
`ca1e85fe9430179831e6bc6be790c332190a3866` and model revision
`19671f406d603126926c1b7e2adc169acbcade22`. The remote-code review is recorded
in `scenarios/music-tools/docs/internal/SECURITY.md`. Moving either pin requires
a fresh review and a new composed-tree digest.

The service runs with `HF_HUB_OFFLINE=1`; it must not fetch code or weights at
startup. `full` keeps the DiT resident on the GPU. `offload-dit` restarts the
managed service with `ACE_STEP_OFFLOAD_DIT=1`. Neither rung evicts another
tenant, and there is no CPU fallback.

Use the control plane for lifecycle:

```bash
vrooli resource validate ace-step
vrooli resource start ace-step
resource-ace-step capacity degrade --to offload-dit
resource-ace-step capacity upshift --to full
vrooli resource status ace-step --json
```
