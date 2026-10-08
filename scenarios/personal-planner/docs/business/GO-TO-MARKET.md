# Go-To-Market — Personal Planner

This document records how Personal Planner would reach users and validate
demand. Like [`MONETIZATION.md`](MONETIZATION.md), it is honest about
stage: the R1 product is implemented for local validation, so this is a
launch plan and a set of hypotheses, not a market or revenue claim.

The strategy follows directly from the monetization bet — in a saturated
market, **beauty and honest planning are the story**, and the go-to-market
must *show* that beauty rather than describe features. The Observatory
experience is itself the marketing asset.

## Purpose Of This Document

Use this document to answer:

- Who is this for, and how is it positioned against the crowded field?
- Which channels reach them?
- What is the launch motion and sequence?
- What message lands, and what experiments prove it?

## Audience And Positioning

- **Audience:** individuals who repeatedly over-commit and feel the pain
  of it — founders, makers, freelancers, knowledge workers, and students
  who already tried a calendar and a to-do app and still miss deadlines
  because neither models *feasibility*. They value calm and craft (they
  notice and care about beautiful software) and they are burned by
  automation that moves their plans without asking.
- **Positioning statement:** *"A planner as calm and beautiful as the best
  of them — that tells you the truth about your time. See what actually
  fits, why a deadline slipped, and what one more 'yes' will cost, without
  anything silently rearranging your day."*
- **Category:** the calm/editorial personal-planner tier (alongside
  Sunsama, Amie, Fantastical), differentiated by an honest-accounting
  substrate the calm tier lacks and a refusal to auto-move accepted work
  that the automation tier (Motion, Reclaim) does.
- **Anti-positioning:** not a team project-management platform, not a bare
  calendar, not a "10x your productivity" hustle app. The voice is calm and
  truthful, never gamified guilt.

## Channels

Ordered by fit with a craft-first, visually-led product:

- **Visual/design-led social** (the primary fit): short screen recordings
  of the Day↔Night transition, the truthful timeline, and the
  "what-moves-if-I-accept-this" moment. The product *is* the ad.
- **Product-discovery communities:** Product Hunt, Hacker News (Show HN),
  and relevant subreddits/forums for productivity and calm software.
- **Content / SEO:** essays on the *ideas* — capacity vs a to-do list,
  why forecasts should be honest not confident, remaining-effort vs
  time-tracking — that attract the audience who feels the pain.
- **Creator / word-of-mouth:** productivity and "tools for thought"
  creators who value craft; the selective read-only commitment-share link
  is a natural, privacy-safe viral surface (a shared promise page carries
  the brand without exposing a private calendar).
- **Vrooli ecosystem (internal channel):** users of Daily and Cadence
  encounter Personal Planner as their scheduling backbone — a built-in
  distribution path unique to this product.

## Launch Motion

Sequenced to the release plan (R0 → R1); do not launch before the honest
loop and the visual gate are real:

1. **Private dogfood (R0–R1 build):** the operator and a small group use
   it daily; the retention question is asked of ourselves first.
2. **Closed beta (R1 candidate):** invite the over-committers described
   above; instrument the retention and honesty-value signals from
   [`MONETIZATION.md`](MONETIZATION.md). Fix trust-breakers before any
   public moment.
3. **Public launch (R1 validated):** a visually-led launch (Product Hunt /
   Show HN / social) leading with the Observatory experience and one
   honest-planning story, not a feature list. Free personal product only.
4. **Monetization test (post-retention, R2/R3):** introduce the paid
   assistance layer to retained cohorts via the channel experiments below.

## Messaging

- **Lead with the feeling, prove with the truth.** Hero moment: the calm
  Observatory Today and the Day↔Night transition. Immediately follow with
  a truthful moment: "3h planned / 4h available / 1h breathing room," and
  "Promised Thursday → Forecast Friday. Here's why."
- **Three message pillars:**
  1. *Calm you return to* — a place you want to open each morning.
  2. *The truth about your time* — capacity, remaining effort, and
     explainable forecasts that never lie green.
  3. *Nothing moves behind your back* — suggestions and previews, applied
     by you (decision D01).
- **Proof, not adjectives:** show the timeline where block width equals
  real duration, the "what moves if I accept this," and the honest empty/
  over-capacity states. Never claim the software guarantees punctuality or
  maximizes productivity (a boundary from the plan, §1.1).
- **Voice:** calm, precise, non-hustle; distinguishes facts from forecasts
  in the copy itself.

## Validation Experiments

- **Desirability (pre-launch):** show the Observatory Today (both
  appearances) to target users; measure "would open daily" and unprompted
  reactions to the beauty. Gate: passes the visual + accessibility release
  gates (plan §6.8).
- **Message resonance:** A/B the three pillars as landing-page hero
  variants; measure sign-up intent per message. Hypothesis: "calm you
  return to" + "truth about your time" beats any feature-list framing.
- **Retention cohort (the decisive experiment):** track weekly return to
  Today and completion of the morning planning loop over 4–8 weeks.
  Monetization is gated on this, not on traffic.
- **Honesty-value:** measure fewer unrecognized overloads and better
  estimate calibration over time from users' own records.
- **Willingness-to-pay (post-retention):** offer the assistance layer to a
  retained cohort; measure conversion at the reference price band from
  [`MONETIZATION.md`](MONETIZATION.md).
- **Share-link virality:** measure whether recipients of a shared
  commitment page convert — the privacy-safe organic loop unique to this
  product.

## Cross-References

- [`MONETIZATION.md`](MONETIZATION.md) — revenue role and pricing hypothesis
- [`../concepts/EXPERIENCE.md`](../concepts/EXPERIENCE.md) — the comparison and the primary surface
- [`../../DESIGN.md`](../../DESIGN.md) — the Observatory design contract (the core marketing asset)
- [`../../PRD.md`](../../PRD.md) — product intent and release boundaries
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — success signals to instrument
- Project-level marketing strategy: `path:docs/marketing/README.md`.
