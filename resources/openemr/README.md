# OpenEMR resource

`openemr` owns isolated, synthetic OpenEMR instances and their MariaDB databases. It pins OpenEMR 8.4.1 and 8.5.0 plus MariaDB 12.3.3. It supports instance lifecycle, status, logs, quiesced snapshots, restore, export, and image upgrade through the resource-specific owner command family.

```sh
vrooli resource openemr install
vrooli resource openemr start --instance-id demo
vrooli resource openemr status --instance-id demo --json
vrooli resource openemr health --instance-id demo --json
vrooli resource openemr snapshot --instance-id demo
vrooli resource openemr backup --instance-id demo --snapshot /path/to/verified-snapshot --destination /mounted/backup/openemr-demo
vrooli resource openemr upgrade --instance-id demo --to 8.5.0
vrooli resource openemr stop --instance-id demo
vrooli resource openemr remove --instance-id demo
```

Each instance gets its own private, non-egress Docker network, MariaDB and OpenEMR containers, database/site directories, and a 0600 credential file outside snapshot targets. The host can reach the UI through the instance's internal bridge IP; no host port is published. All containers and networks are verified against owner labels before an operation. The site returns synthetic credentials only when explicitly requested with `credentials`; ordinary status and bounded Docker logs redact secrets.

`snapshot` stops the site and database before archiving their on-disk trees, writes a hash-bound manifest with restrictive permissions, then restarts the instance. Database archives are read by the pinned MariaDB image under its mapped data UID; restore writes each tree under the mapped resource owner so host user namespaces preserve data access. `restore` verifies the manifest, replaces only that instance's two data trees, and restarts it. Export writes the same snapshot bundle to a caller-selected directory after a strict path check. `backup` accepts either an existing DBM destination ID for a manually triggered backup, or a verified local snapshot plus a caller-selected filesystem destination for encrypted Kopia. Kopia is an optional host tool; an attached filesystem is not an off-host or independent-failure-domain copy. Neither path sets an unattended schedule. `remove` retains Kopia's private recovery password/config until the external repository is retired, so stored backups do not become unrecoverable.

This resource is supported only on a Linux host with Docker for the commissioned synthetic research path. A successful start or readiness check does not establish clinical adequacy, security certification, vendor support, production readiness, or compatibility with any particular practice.
