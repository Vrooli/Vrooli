# Bundle integration

Browser Automation Studio uses the shared monetization contract for paid
features. Its declaration is validated by LPBS monetization conformance and
its desktop delivery carries the bundle and application identity from the
build artifact.

Before a release, provide both seams required for paid applications:

- an account surface that can resolve the shared authenticated session and
  present subscription state;
- a journey probe that exercises the paid boundary and records evidence with
  an identifier and checksum.

Class A enforcement remains server-owned. Local declarations, cached leases,
and desktop UI state are not authorities.
