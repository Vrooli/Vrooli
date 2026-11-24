# Bahmni Lite resource

This resource runs the official Bahmni Lite 1.0.0 evaluation stack from the pinned `Bahmni/bahmni-docker` source commit recorded in `UPSTREAM.md` and the digest lock at `images.json`. It uses the `bahmni-lite` Compose profile so its comparison includes the Crater billing module; the upstream `emr` default omits that module.

Use the owner CLI:

```sh
vrooli resource bahmni-lite install
vrooli resource bahmni-lite start --instance-id m3-bahmni-proof
vrooli resource bahmni-lite status --instance-id m3-bahmni-proof
vrooli resource bahmni-lite logs --instance-id m3-bahmni-proof --service openmrs --tail 100
vrooli resource bahmni-lite stop --instance-id m3-bahmni-proof
vrooli resource bahmni-lite remove --instance-id m3-bahmni-proof
```

The owner creates a unique Compose project and generated 0600 environment file for each instance. Containers, the Compose network, and named volumes carry resource and instance labels. The default network is internal, and upstream fixed host port publications are removed. The app URL is the owner's private bridge address; it is reachable from this host and is not a published host port.

Credentials are synthetic evaluation credentials. OpenMRS bootstrap credentials follow the upstream evaluation image defaults; Crater receives a generated per-instance admin password. Read them only with `vrooli resource bahmni-lite credentials --instance-id <id>`. No real patient records, clinic credentials, production access, client contact, vendor contact or spending is authorized. Bahmni demo data and this locally seeded CIEL evaluation database are not the M1 dataset; a same-journey matrix must use the frozen synthetic recipes and report each case as pass, unsupported or blocked.

This resource has start, stop, status, health, logs, image acquisition and explicit removal. Snapshot/restore and a data-preserving upgrade to a later Bahmni release are not yet implemented or proven. This evaluation path does not establish clinical adequacy, vendor support, security acceptance or production readiness.
