# Kubo CAR fixture

`site.car` and `index.html` are copied from the prior crypto-infrastructure synthetic site at `runs/research-renewal-20261026T131100Z/evidence/E4/build-a/`. They are used only as local offline Kubo import fixtures. The source receipt records the 661-byte HTML hash as `7e9ea89af53aa20b368536238d019caefa10231eaaa9e40cac26619b71bc6659`; this fixture is synthetic and not a public pin or deployment.

`public-sandbox-type4.item` is the public 817-byte synthetic type-4 item retained from `runs/research-renewal-20261027T043008Z/evidence/SB7-public-handoff`. The test validates its signature, deep hash, framing, item ID and exact bytes against `index.html`. It contains no private key or account state and is used only for local protocol regression tests.
