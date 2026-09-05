# macOS first-run audit — 2026-08-14

## Target and access

- Target: `minimouse.local`, Mac mini Intel (`darwin/amd64`)
- Bridge machine: `451ea636-a80f-4080-82b7-fa65d0e3289a`
- Current Bridge node: `ee0d0bf6-7534-49e6-9601-5ec6701a28f7`
- Direct SSH from this control-plane session was not available: the local
  private keys did not match the machine's Bridge-managed key. Bridge retained
  the existing SSH trust and node connection.

## What was proven

Bridge operation `4f7e31ba-f8ca-4e07-b5bc-b62172c6c648` completed with durable
state `ONBOARDING_STATE_SUCCEEDED`. It shipped the working tree, received
prebuilt `darwin/amd64` artifacts, recovered the Go toolchain, built the native
CGO-enabled CLI, completed setup-finalize through the native Keychain-enabled
CLI, paired the node, installed the service, enabled launchd KeepAlive, and
verified the node online.

## Fresh-host blocker

The required fresh reset could not be performed safely through the available
authority. The existing `/Users/matthalloran8/Vrooli` directory is not a Git
checkout. Bridge's repair operation `415c17f5-fd36-4c3a-a2fd-7c56a7a7b3ab`
therefore stopped at clone with:

```text
checkout directory /Users/matthalloran8/Vrooli already exists and is not a Git checkout;
use --source-dir for an intentional working-tree ship or choose an empty --checkout-dir
```

The successful working-tree operation used the isolated checkout
`/Users/matthalloran8/Vrooli-plan-audit`; it did not remove `~/.vrooli` or the
original checkout. Bridge's recorded cleanup tombstone explicitly says that
no remote cleanup was executed. No claim is made here for fresh macOS backend
selection, passphrase count, Keychain reachability after a clean reset, or
unattended credential read after reboot.

This is an external access/reset blocker, not a passing macOS first-run proof.
