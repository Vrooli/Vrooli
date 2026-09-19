# Provider Survey — Resale Terms And Billing Facts

**Nothing in this scenario is implemented.** This document is a survey carried
out during design. No provider account has been opened, no credential has been
configured, and no capacity has been provisioned. The survey exists because the
choice of first adapter was made on contract terms rather than on API quality,
and that reasoning is worthless if it cannot be re-checked.

**This is not legal advice.** It is a record of what specific clauses in
specific documents said on specific dates, so that a human can re-read them
before money moves. Every operative claim below carries the document, its own
version or effective date, the section number, the retrieval date, and enough
verbatim text to be falsified by reading the source.

## Purpose Of This Document

Use this document to answer:

- May Vrooli buy capacity from this provider and let a customer use it?
- Which document actually governs that question, and which one wins on conflict?
- What obligations does permission carry, and can it be withdrawn?
- What was read, when, and how, so a stale row is visible as stale?

Two consumers depend on it. [`../business/GO-TO-MARKET.md`](../business/GO-TO-MARKET.md)
uses it as the positioning constraint on the paid path, and
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) cites it for the decision
to write the Hetzner adapter first.

## Method

Three method rules produced different answers than a casual reading would have.

**A provider's general terms are not the whole contract.** Scaleway's General
Terms of Service are silent on reselling, which reads as permission. The
Specific Conditions applicable to Instance Services forbid it outright and
declare themselves to take precedence on conflict. Reading only the headline
agreement produced the opposite answer to the correct one. Every row below
records which documents were read and which one wins.

**Permission is a spectrum, not a boolean.** Four distinct shapes appear:
permitted by the standard terms with no further step; permitted only after
acceptance into a partner programme; permitted only under a separate policy
that is revocable at will; and prohibited outright. Collapsing these into
"allowed" and "not allowed" loses the fact that two of the three permissive
providers can withdraw permission unilaterally.

**The scope is the specific product.** The survey covered AWS Lightsail rather
than EC2, and Scaleway Instances rather than Elastic Metal. Where a clause
governs "the Services" as a whole this does not matter, and where it is
product-specific it matters a great deal. Each row says which.

**Vendor documentation carries no contractual weight.** Fly.io's own
documentation recommends the per-customer application pattern that its Terms of
Service prohibit. The entire-agreement clause settles that conflict against the
documentation. Marketing pages and how-to guides were not treated as evidence.

## Retrieval Provenance

| Provider | Documents read | Document's own date | Retrieved | Method |
|---|---|---|---|---|
| Hetzner Online GmbH | Terms and Conditions | Version 2.0.0, 27 October 2021 | 2026-09-03 | Direct fetch, two independent fetchers, identical text |
| DigitalOcean | Terms of Service; Partner Terms and Conditions | Partner Terms effective 15 April 2026 | 2026-09-03 | Direct fetch |
| Amazon Web Services | AWS Customer Agreement; AWS Service Terms | Customer Agreement last updated 14 August 2026; Service Terms last updated 1 September 2026 | 2026-09-03 | Direct fetch, two independent fetchers, identical text |
| Scaleway | General Terms of Service; Specific Conditions applicable to Instance Services; five further Specific Conditions | General Terms v17/07/2024 and v2026; Instance Specific Conditions v07/04/2026 | 2026-09-03 | Direct fetch of the published PDFs |
| Fly.io | Terms of Service | Effective 29 April 2026 | 2026-09-03 | Direct fetch |
| Linode / Akamai Connected Cloud | Akamai Master Services Agreement; Akamai Reseller Policy | MSA v1.4.0, updated 4 January 2023, effective 1 April 2020; Reseller Policy v1.0, updated 15 December 2019, effective 1 January 2020 | 2026-09-03 | Direct fetch |
| Vultr | Terms of Service; Acceptable Use Policy; Partner Program Terms | ToS 28 March 2024; AUP 11 March 2026; Partner Terms 20 September 2021 | Archived captures, not live: ToS and Partner Terms captured 2026-06-14, AUP captured 2026-05-17 | Common Crawl WARC byte-range fetches. See the caveat below |

**The Vultr rows are archived captures and must be re-read before use.**
`vultr.com` returns HTTP 403 to every fetcher tried, and the Internet Archive
was rate-limiting throughout. The quotations come from Common Crawl WARC
records, which are genuine HTTP 200 crawler captures rather than renderings,
and the documents' own last-modified lines were byte-identical across four
independent captures spanning March 2025 to June 2026. Newer snapshots exist
that could not be opened. Treat the Vultr rows as up to three months stale.

**The Linode documents predate the Akamai rebrand.** The MSA has not been
revised since January 2023 and the Reseller Policy not since December 2019.
`www.linode.com/legal-tos/` now returns 404, so the Akamai MSA is the operative
agreement.

The billing facts in the second half of this document come from a separate pass
over public product documentation, also on 2026-09-03: the Hetzner Cloud billing
FAQ, the Hetzner pressroom price-adjustment announcement and its per-plan price
table, the DigitalOcean Droplet pricing page, the AWS Cost Explorer user guide,
the DigitalOcean billing API reference and the Vultr server-billing support
page. Those pages are public and unauthenticated, so they are cheap to re-check.

## Resale And Third-Party Use

| Provider | May a customer let a third party use the capacity? | Governing clause | Programme required? | Per-end-customer acceptance? | Revocable? |
|---|---|---|---|---|---|
| **Hetzner** | **Yes, by the standard terms** | T&C §7.1 | No | No | Only by terminating the contract |
| DigitalOcean | Yes, but only inside the Partner Program | Partner Terms §1.1, §1.3 | Yes | **Yes, each End Customer individually** | Yes, on programme termination |
| Linode / Akamai | Prohibited by default; granted on consent to the Reseller Policy | MSA §4.4; Reseller Policy §2 | Yes, consent to the reseller SUP or SPA | Written contract required with every end user | **Yes, "revocable by Linode at any time"** |
| Scaleway | **No, for Instances**; the other product lines are silent | Instance Specific Conditions, resale undertaking | n/a | n/a | n/a |
| AWS (Lightsail) | No | Customer Agreement §6.4(c) | Solution Provider or Distribution Program, neither self-serve | n/a | n/a |
| Fly.io | No | ToS §1.4(d) | No programme exists | n/a | n/a |
| Vultr | No, without written permission | AUP; ToS §5(b)(ii) | Signed Partner Agreement; programme acceptance alone is **not** sufficient | n/a | n/a |

Counted the way the design decisions count it: **three of seven permit resale in
some form** (Hetzner, DigitalOcean, Linode) and **four of seven forbid it or gate
it behind written permission** (AWS, Fly.io, Vultr, Scaleway Instances). Of the
three that permit it, **Hetzner is the only one whose standard terms suffice**,
with no programme to join, no per-customer acceptance step, and no separate
revocable policy.

## Per-Provider Detail

### Hetzner Online GmbH

Terms and Conditions, Version 2.0.0, last updated 27 October 2021, retrieved
2026-09-03.

§7 is headed *Use by third parties*. §7.1, verbatim:

> The Customer is entitled to grant third parties a contractual term of use to
> any services the Customer orders from Hetzner. In this case, the Customer
> nevertheless remains the sole contractual partner. The Customer continues to be
> solely and fully liable for compliance with the contractual agreements between
> us and the Customer.

§7.2 requires the customer to ensure legal and contractual provisions are
followed at the time of transfer. §7.3 makes the customer fully liable for the
third party's breaches and indemnifies Hetzner against claims.

The words *resell*, *reseller* and *sublet* appear zero times in the document.
Hetzner grants third-party use rights without ever using resale vocabulary,
which is why a keyword search for "resell" finds nothing and a reader who stops
there concludes wrongly that the question is unaddressed.

**The acceptable-use constraint that matters commercially** is §8.3. Its first
paragraph prohibits spam and sender-identity falsification. Its second
paragraph, verbatim:

> The operation of applications for mining cryptocurrencies remains prohibited.
> These include, but are not limited to, mining, farming and plotting of
> cryptocurrencies.

followed by:

> We are entitled to lock the Customer's access to their Hetzner services or
> account in the event of non-compliance.

Read together with §7.1's "the Customer remains the sole contractual partner",
this is the account-termination risk in its precise form: a customer mining
crypto on capacity Vrooli bought is Vrooli's breach, and the remedy Hetzner
reserves is locking the account rather than the individual server. §5.2 grants
the same remedy for (d)DoS and open relays, "without prior notice".

Two further clauses bear on that risk. §9.2 makes the customer liable for all
direct and indirect damages arising from a §8 breach, including legal defence
costs. §1.3 reserves Hetzner's right to change the terms, the System Policies
and the prices on notice through the customer account or by email.

### DigitalOcean

Terms of Service and Partner Terms and Conditions, retrieved 2026-09-03.
Partner Terms effective 15 April 2026.

The general Terms of Service do contemplate third parties: §4.4 defines "End
Users" to include parties who access your Services Content "including via
resale", and §4.5 makes you responsible for their compliance. Read alone this
looks permissive.

The Partner Terms are the operative document. §1.1, verbatim in part:

> Subject to the terms and conditions of the Partner Terms and the other Partner
> Program Documents, we authorize you to promote and/or resell the Services to
> end customers ("End Customers") in the Territory in accordance with the Program
> Track(s) for which you are approved.

Authorization is a grant, and it is conditional on approval for a Program Track.
The clause that blocks a white-label subscription is §1.3, verbatim in part:

> If you are participating in any Program Track that involves the reselling of
> Services, you are responsible for ensuring that each End Customer agrees to the
> End Customer Terms in a manner that is legally binding upon the End Customer.
> We may refuse to enable or provide Services to an End Customer until we have
> confirmed (either through an End Customer's acceptance of the End Customer
> Terms through our site or through written documentation from you) that the End
> Customer has accepted the End Customer Terms.

§1.3 also forbids adding to or varying the End Customer Terms, allowing the
partner to set only pricing and payment terms.

### Amazon Web Services

AWS Customer Agreement, last updated 14 August 2026, retrieved 2026-09-03. The
surveyed product was Lightsail, but §6.4 governs "the Services" generally.

§6.4 *Restrictions*, verbatim:

> Neither you nor any End User will use the AWS Content or Services in any manner
> or for any purpose other than as expressly permitted by this Agreement. Neither
> you nor any End User will, or will attempt to (a) reverse engineer, disassemble,
> or decompile the Services or AWS Content or apply any other process or procedure
> to derive the source code of any software included in the Services or AWS
> Content (except to the extent applicable law doesn't allow this restriction),
> (b) access or use the Services or AWS Content in a way intended to avoid
> incurring fees or exceeding usage limits or quotas, or (c) resell the Services
> or AWS Content.

Clause (c) is a flat prohibition rather than a gate. §2.5 makes the customer
responsible for End Users' compliance and requires immediate suspension on
becoming aware of a violation.

That 6.4(c) is the default rather than an edge case is corroborated by AWS
Service Terms §8.3.2 (last updated 1 September 2026), which carves out an
exception "unless you have been authorized as an AWS reseller". AWS's own
drafting treats authorized-reseller status as the exception to a standing ban.

The gated paths are the AWS Solution Provider Program and the AWS Distribution
Program. Neither is self-serve; entry runs through an AWS Distributor. **The
text of the AWS Distribution Seller Agreement was not retrieved**, so nothing is
claimed about its contents.

### Scaleway

General Terms of Service (v17/07/2024 and the 2026 revision) and six Specific
Conditions PDFs, all retrieved 2026-09-03.

The General Terms of Service are silent on resale in both the 2024 and 2026
versions. The Specific Conditions applicable to Instance Services, version of
07/04/2026, state their own precedence, verbatim:

> The Specific Conditions herein supplement the provisions of the General Terms
> of Service. The provisions of the Specific Conditions shall take precedence
> over those of the General Terms of Service, in the event of any conflict
> between the two.

and then, verbatim:

> The Client undertakes not to resell the subscribed Instances to third parties.

**The prohibition is Instance-specific and does not generalise.** The Specific
Conditions for Bare Metal, Dedibox VPS, Containers, Storage, AI Services and Web
Hosting are all silent on resale. Bare Metal goes further and contemplates the
opposite, verbatim:

> The Client acknowledges that it is the administrator of the Dedicated Server
> made available to it as part of the Services, and might therefore be considered
> as a "hosting service provider" within the meaning of the Digital Services Act
> Regulation 2022 (EU) 2022/2065.

Elastic Metal is therefore a path that remains open where Instances are closed.
A row that reads "Scaleway prohibits reselling" without the product qualifier is
wrong, and this is the clearest illustration of why the per-service annex rule
exists.

### Fly.io

Terms of Service, effective 29 April 2026, retrieved 2026-09-03.

§1.4 *Restrictions*, verbatim:

> Customer will not, and will not permit any Authorized User or other party to:
> (a) knowingly interfere with or disrupt the integrity or performance of the
> Fly.io Services or the data contained therein; (b) reverse engineer, disassemble
> or decompile any component of the Fly.io Services; (c) interfere in any manner
> with the operation of the Fly.io Services or the hardware and network used to
> operate the Fly.io Services; (d) sublicense any of Customer's rights under this
> Agreement, or otherwise use the Fly.io Services for the benefit of a third
> party; (e) modify, copy or make derivative works based on any part of the
> Fly.io Services; or (f) otherwise use the Fly.io Services in any manner that
> exceeds the scope of use permitted under this Agreement.

Clause (d) is the operative one, and the framing matters: **the agreement never
uses the word "resell" at all.** The prohibition is on the underlying conduct,
"use the Fly.io Services for the benefit of a third party", which is broader
than resale and would also capture free hosting and agency work.

Two clauses make (d) unambiguous. §1.1 grants "a non-sublicensable,
non-transferable, non-exclusive subscription to, solely for Customer's internal
use". The §13 definition of "Authorized User" covers only "Customer's employees,
agents, and independent contractors". There is no End User concept anywhere in
the agreement, and no reseller programme: `/legal/reseller` and `/legal/partner`
both return 404.

Fly.io's own documentation recommends the opposite. Its page "One App Per
Customer" says "We recommend that you create one app, with one or more machines,
for each customer you have", and its Machines guide describes running user code
"All while charging users per request!". The entire-agreement clause settles the
conflict against the documentation. This is noted only so that a later reader
who finds those pages does not conclude the row above is an error.

### Linode / Akamai Connected Cloud

Akamai Master Services Agreement (v1.4.0, updated 4 January 2023, effective
1 April 2020) and Akamai Reseller Policy (v1.0, updated 15 December 2019,
effective 1 January 2020), both retrieved 2026-09-03.

MSA §4.4 is headed *Resale Prohibited*, verbatim:

> Covered Users are prohibited from selling or reselling any Service unless you,
> on behalf of yourself and all Covered Users, consent to the appropriate
> reseller SUP or SPA without exception.

*Low-confidence detail:* two independent fetches returned this sentence with and
without the trailing phrase "without exception". The operative rule, prohibited
unless consent is given to the reseller SUP or SPA, was identical in both.

"Covered Users" means "you, your Representatives, your End Users, and the End
Users of your Representatives". MSA §2.3 makes the account holder responsible
and liable for all activities associated with the account, and §4.2 requires
ensuring all Covered Users comply with the Terms of Service.

The Reseller Policy is the SUP that §4.4 points at. §1 lists the Eligible
Services, which include Standard, Dedicated, High Memory and Graphics Cloud
Compute. §2, verbatim in part:

> Linode grants Reseller a non-exclusive, revocable right to resell the Eligible
> Services solely to Qualified End Users ... this right is non-exclusive and
> revocable by Linode at any time

§3 requires a written contract with every end user carrying the data-processing
and supplemental-use-policy flow-down. §4.1 makes the reseller solely
responsible for end-user use and disputes. §5 is the channel prohibition,
verbatim:

> Reseller is strictly prohibited from marketing, soliciting, or selling any
> Eligible Service to any current customer of Linode, provided that Reseller is
> not prohibited from responding to any inbound inquiry from any customer,
> including customers of Linode.

Akamai's own technical documentation states the accountability plainly: "If for
any reason a customer of a reseller breaks our ToS, it is the reseller who will
be held accountable".

The correct one-line summary is therefore *prohibited by default, granted on
consent, revocable at any time, and carrying both a non-solicitation bar and a
written-contract flow-down*. A summary that reads "permits resale but bars
soliciting" has the polarity backwards and understates the revocability.

### Vultr

Terms of Service (28 March 2024), Acceptable Use Policy (11 March 2026) and
Partner Program Terms (20 September 2021). Archived captures rather than live
retrievals; see the provenance caveat above.

The operative written-permission requirement is in the **Acceptable Use Policy**,
not the Terms of Service, verbatim:

> You agree NOT to attempt to resell Vultr's products and/or access to the
> Services without our written permission.

The Terms of Service §5(b)(ii) restriction is broader but weaker as a resale
rule, verbatim in part:

> You may not: ... (ii) reproduce, modify, prepare derivative works based upon,
> distribute, license, lease, sell, resell, transfer, publicly display, publicly
> perform, transmit, stream, broadcast or otherwise exploit the Services except
> as expressly permitted by Vultr

The Partner Program Terms explicitly refuse to be the written permission,
verbatim:

> Acceptance into the Program alone does not authorize you to resell or
> sublicense Vultr services.

Authorization requires a separately signed Partner Agreement. A row that cites
only the Terms of Service understates the position, because the ToS restriction
reads as a software-licence boilerplate clause while the AUP sentence is
specifically about reselling infrastructure.

## Billing Facts

Terms decide whether capacity may be sold. These facts decide whether selling
it can be profitable. They were retrieved 2026-09-03 from the providers' own
public documentation.

### Hetzner rounding and the monthly cap

From the Hetzner Cloud billing FAQ, verbatim:

> We always round up the hourly usage of a server. If you create a server just
> for a few minutes, we will still bill you for one whole hour.

> Your server's bill will never exceed its monthly price cap... we will bill you
> for the minimum amount, whether that is the monthly price cap OR the hourly
> price multiplied by the number of hours you used the server.

> We will bill you for each cloud server until you choose to delete them... If
> you delete your cloud server before the end of the billing month, we will only
> bill you for the hourly rate.

The rule is `min(monthly_cap, ceil(hours) * hourly)`. **The cap is per server, so
it gives no protection against churn.** Fifty ten-minute servers cost fifty
hours, and no cap applies to any of them. This is the single most load-bearing
cost fact in the pricing hypothesis: it is what makes a minimum billable unit
mandatory rather than optional, and it is why warm pooling is a plausible
mitigation rather than an optimisation.

### The June 2026 Hetzner price increase

Announced 27 May 2026, effective 15 June 2026 at 08:00 CEST. Scope is "all
dedicated servers at all locations" and "all cloud plans at all locations".
Explicitly unaffected: web hosting, managed servers, Server Auction servers,
IPs, storage products, Load Balancers, Volumes, Snapshots and Object Storage.
The stated reason is "the ongoing challenges in the hardware procurement
market".

Two scope details matter for an adapter. First, "the changes apply exclusively
to new orders and rescales of existing servers", so existing contracts keep
their pricing and **a resize codepath is a repricing event**. Second, the
separate dedicated root server line (AX/EX/SX/GEX/DX) is also covered by the
adjustment and moves differently, because it comes with a significant reduction
in one-time setup fees. Its figures were not extracted.

**Hetzner published no percentages.** Neither the press release nor the
price-adjustment documentation states one. Every percentage below is arithmetic
on Hetzner's own published old and new monthly prices, and must be attributed as
derived rather than quoted.

| Line | What it is | Region | Derived increase |
|---|---|---|---|
| CX | Cloud shared vCPU, Intel/AMD | Germany, Finland (EU only) | +30.8% to +37.6% |
| CAX | Cloud shared vCPU, Arm | Germany, Finland (EU only) | +30.2% to +33.4% |
| CPX | Cloud shared vCPU, AMD | Germany, Finland | +143.9% to +175.4% |
| CPX | Cloud shared vCPU, AMD | USA | +166.8% to +209.0% |
| CPX | Cloud shared vCPU, AMD | Singapore | +50.8% to +93.9% |
| CCX | Cloud dedicated vCPU | Germany, Finland | +113.4% to +173.1% |
| CCX | Cloud dedicated vCPU | USA | +107.1% to +157.4% |
| CCX | Cloud dedicated vCPU | Singapore | +65.3% to +110.7% |

**The common summary "shared rose by about a third, dedicated by up to 175
percent" is wrong and understates the risk.** CPX is a shared vCPU line and it
rose by up to 209 percent, which is six times the "about a third" figure. The
honest one-line version is: the EU CX and CAX shared lines rose 30 to 38
percent, while the CPX shared line and the CCX dedicated-vCPU line rose between
51 and 209 percent depending on line and region.

### DigitalOcean per-second billing

Per-second billing took effect 1 January 2026, with "a minimum charge of 60
seconds or $0.01, whichever is higher". That is a dual floor, both time and
money, and on the cheapest Droplets **the money floor binds before the time
floor does**. Bundled CPU Droplets cap at 672 hours per month; v5-configuration
Droplets have no monthly cap.

Per-second billing does not remove the need for a minimum billable unit. It
changes which provider sets the worst case, and the minimum billable unit must
cover the worst case across every provider the router may select.

### Billing-data latency: no provider publishes a bound

**No provider in scope publishes a billing-data latency SLA.** This was checked
directly against the billing documentation of AWS, DigitalOcean, Vultr and
Hetzner, and by search for Linode, Scaleway and Fly.io.

AWS is the only one with a published number and it explicitly declines to bound
the tail. Cost Explorer documentation, verbatim in part:

> The current month's data is available for viewing in about 24 hours... Cost
> Explorer refreshes your cost data at least once every 24 hours. However, this
> depends on your upstream data from your billing applications, and some data
> might be updated later than 24 hours.

Cost and Usage Reports are written to the bucket once a day. DigitalOcean's
billing API reference states no update frequency and no freshness commitment;
its balance object carries a `generated_at` timestamp, which reports the age of
the data at read time and promises nothing about it. Vultr generates invoices on
the first of the month with charges accruing hourly from deployment, and
documents nothing about how quickly accrued usage becomes visible. Hetzner
documents no latency and ships consumption detail as a separate individual
consumption statement rather than on the invoice.

**The consequence for reconciliation is structural, not a tuning parameter.**
A cadence cannot be sized against a documented worst case because no worst case
is documented. Reconciliation must tolerate unbounded lateness and correct
retroactively, rather than assume any fixed settlement window. Any figure with a
specific upper bound in hours would be a number without provenance.

## Claims Not Sourced By This Document

These appear in other documents in this scenario and are **not** supported by
anything retrieved here. They are listed so that a reader does not mistake this
survey for their evidence.

| Claim | Where it appears | Status |
|---|---|---|
| A stopped instance still bills at the full rate on five of the seven providers surveyed | [`../internal/DECISIONS.md`](../internal/DECISIONS.md), [`../business/GO-TO-MARKET.md`](../business/GO-TO-MARKET.md) | **Unsourced.** No per-provider pricing page was retrieved for this claim. It is load-bearing for `OT-P0-007` and it must be sourced per provider before it appears in any public asset. |
| Provider billing data lags by 7 to 33 hours | Previously in [`../business/MONETIZATION.md`](../business/MONETIZATION.md) | **Withdrawn.** Actively checked and no provider publishes such a bound. See the latency section above for the sourced replacement. Do not reintroduce a figure in hours unless it is an in-house observation carrying its measurement method and date. |
| Hetzner bills outbound traffic only; AWS small-instance products count inbound | [`../internal/DECISIONS.md`](../internal/DECISIONS.md) | **Unsourced.** Traffic-pricing pages were not part of this survey. |
| Hetzner offers twenty times DigitalOcean's included traffic | Design brief | **Unsourced.** Same gap. |

Sourcing the remaining rows is a prerequisite for any published price, and it is
cheaper than the terms survey above because pricing pages are public and
unauthenticated.

## Revisit Triggers

Machine-evaluable, per the discipline in `path:../../docs/monetization/catalogs/CATALOG.md`.

- **Before the first real provider credential is configured for any provider**,
  re-read that provider's rows. Nothing here is older than one retrieval cycle
  at the moment and all of it will be.
- **Before the first paid customer instance is created**, re-read the Hetzner
  T&C and confirm §7.1 and §8.3 are unchanged, and record the new version number
  and date in the provenance table. Hetzner reserves the right to change terms on
  notice under §1.3.
- **When a second adapter is proposed** (`OT-P1-004`), the provider's terms
  review is part of the work and this document gains a row before any code is
  written.
- **The Vultr rows expire the moment a live retrieval becomes possible.**
  Re-fetch and replace the archived captures.
- **The Linode rows should be re-read whenever Akamai revises the MSA**, because
  the current version is from January 2023 and the Reseller Policy from December
  2019.

## Cross-References

- [`../internal/DECISIONS.md`](../internal/DECISIONS.md): the first-adapter decision and the per-service annex method rule, both of which cite this survey
- [`../internal/SECURITY.md`](../internal/SECURITY.md): account termination as a fleet-wide availability failure
- [`../business/GO-TO-MARKET.md`](../business/GO-TO-MARKET.md): positioning, the single-supplier risk, and the launch-gating checklist
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md): the pricing hypothesis these terms constrain
- [`../../PRD.md`](../../PRD.md): `OT-P1-004`, the second-provider target that triggers the next survey row
