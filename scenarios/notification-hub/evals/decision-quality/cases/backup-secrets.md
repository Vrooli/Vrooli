# Case: backup-secrets

Taken from Hub ask ba03832a (15 Nov 2026). You are Data Backup Manager's agent
(`--from "Data Backup Manager"`).

## Facts you know

- Data Backup Manager scanned the machine and found 45 places that hold secrets
  (sign-ins, keys, private chat history) and have no backup. Because they hold
  secrets, it never adds them to the backup plan without the operator's answer.
- Backups are encrypted. Adding all 45 adds 3.1 GiB to each backup.
- This is the third time the operator has been asked about this list: 44 items on
  13 Nov, 45 on 14 Nov, 45 again today. Each time new secret-holding places
  appear, the same question comes back. There is no standing rule for secrets.
- Risk of backing up: after a restore, a sign-in or token may be out of date, so
  the operator signs in again once. The data itself comes back intact.
- Risk of not backing up: if the disk fails, everything below that exists nowhere
  else is gone (AI coding history and memory, Vault's encrypted secrets, app keys).
- No answer within 7 days: nothing is added.
- The Hub can offer a form question (`--form`) with a multi-select, so the operator
  could pick groups without typing.

## The 45 items, grouped (sizes are real)

AI tools and local services (12 items, 2.9 GiB) — history and sign-ins that exist nowhere else:
- Claude Code: transcripts, memory, settings, sign-in — 2.1 GiB
- Grok: sessions, memory, settings, sign-in — 364.7 MiB
- Antigravity: conversations, memory, settings, sign-in — 327.7 MiB
- Codex: sessions, memories, settings, sign-in — 85.7 MiB
- Speaker verification profiles (cannot be re-made without re-enrolling) — 34.6 KiB
- Vault encrypted secret store — 17.4 KiB; Vault config — 224 B
- Kopia config 10.0 KiB, Kopia state 3.2 KiB, AdGuard Home config 4.0 KiB,
  Twilio config 2.1 KiB, Ollama registry credential 387 B

Vrooli apps' keys and records (10 items, 222.5 MiB) — losing them breaks sign-ins or loses history:
- Workspace sandbox saved patches and file bodies — 218.8 MiB
- Swarm Manager event and approval history — 2.9 MiB (plus its live database
  sidecar file, 32 KiB, which is useless without the database and should travel with it)
- Browser Automation Studio session profiles (cookies, sign-ins) — 626.8 KiB
- Notification Hub decision attachments — 141.1 KiB; Notification Hub push key — 44 B
- Web Console control-plane signing key 32 B, Web Console agent identity secret 32 B
- Test Genie seed processes 384 B, Test Genie seed sessions 29 B

Host settings that Vrooli setup wrote (23 tiny files, 10.9 KiB total) — sudo,
kernel, AppArmor, systemd, journald, logrotate, GRUB and DNS settings. Setup can
write them again; a copy records exactly how they were set.

## What the sender would recommend

Back up all 45 now, and back up anything new that holds secrets automatically from
now on, with a weekly note listing what was added (the operator can exclude any item).
Reason: losing this would cost far more than 3.1 GiB, and it stops this question coming back.

## What carrying out each answer means

- Standing rule: register the 45 today; future secret-holding places are added
  automatically and listed in a weekly note.
- Groups only: register only the groups the operator selects; new secrets still wait for an answer.
- Not now: nothing is added.
