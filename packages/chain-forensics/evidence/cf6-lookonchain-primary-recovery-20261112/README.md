# CF6: Lookonchain primary-source recovery

## Finding

The cited primary Lookonchain public report is recoverable from free public material. ChainCatcher's 2023-08-14 article links its “Source Link” directly to `https://twitter.com/lookonchain/status/1690965940937334784`; this redirects to the public X status URL below. An unauthenticated read of X's public syndication endpoint returns the same status ID, author, post timestamp, and text. This closes the source-recovery question; it does not independently validate the reported transactions.

- Primary status: https://x.com/lookonchain/status/1690965940937334784
- ChainCatcher citing page: https://www.chaincatcher.com/en/article/2099200
- Status ID: `1690965940937334784`
- Author in returned primary-host data: `@lookonchain` / Lookonchain
- Published: `2023-08-14T05:57:51Z`
- Report text: “Bald Deployer has successfully bridged 7,000 $ETH ($12.9M) from the #Base network back to #Ethereum and deposited 2,100 $ETH($3.87M) to #Kraken. There are currently 2,000 $ETH ($3.7M) in bridging and 2,237 $ETH ($4.13M) in the #Base network.” The post includes links to an image (`https://t.co/BdB5QIob0s`) and a detail link (`https://t.co/mtko220gTR`).

## Captured bytes and provenance

These are direct HTTPS GET responses captured on `2026-11-12T17:17:10Z` without credentials or cookies. `capture-manifest.json` records requested and final URLs, response status/content type, byte counts, and SHA-256 hashes. The raw X status HTML is 149,623 bytes (`c60e7a0d05cea7274c05aebca413a31e1b4d5fbf0528b116994565d2f7c69330`). The raw public X syndication JSON for the same status is 3,926 bytes (`cf8b4fe0f5e0914387c9c80fe3ee27632bc16487bfe0937663db95811f005ced`). The JSON is a retrieval representation served by X's public CDN; it is retained as corroborating machine-readable source material, not as a separate independent publisher. Files in this directory are read-only after capture.

The attribution of this status ID is corroborated by ChainCatcher's page linking the `/status/1690965940937334784` URL and by contemporaneous secondary reproductions that quote the same text and identify Lookonchain. The primary source itself directly establishes only what the account reported and when.

## Limits

The post is an analyst report, not transaction-level proof. Its text does not supply transaction hashes, a bridge protocol, an Ethereum recipient address, or a Kraken deposit address. It cannot by itself establish that funds were credited to Kraken custody, traded or sold; cannot establish identity, intent, wrongdoing, or venue account ownership; and should be attached as attribution to the report only. Retain the existing graded packet claims and public-chain evidence separately. This capture does not modify the served BALD case packet or ATTENTION.

## Reproduction

Verify raw captures with `sha256sum *`. The retained capture manifest records the exact unauthenticated source URLs and byte hashes. No paid service, credential, or contact was used.

## Work record

- Changed: added this isolated evidence directory with the X primary-status HTML, X CDN syndication JSON, capture manifest, and interpretation note.
- Verified: both captured response hashes match the manifest; syndication JSON reports status ID `1690965940937334784`, account `lookonchain`, timestamp `2023-08-14T05:57:51Z`, and the quoted 7,000/2,100 ETH wording; ChainCatcher's Source Link resolves to this status ID.
- Remaining: the primary post contains no tx hashes or deposit address; any on-chain/venue correlation must remain a separate graded claim.
- Unverified: whether Kraken credited any specific deposit, whether any ETH was sold, and real-world identity or intent.
