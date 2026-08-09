# Run Task: Opportunity Scout

## Reasoning Framework
1. Scan external signal: market trends, competitor moves, and capability arrivals.
2. Scan internal signal: recent operator vision-walk knowledge and new scenario surfaces.
3. Identify capability x market combinations.
4. Classify each idea by SKU, add-on, services line, or discovery channel.
5. Attach acquisition and retention hypotheses.
6. Attach a concrete revisit trigger.
7. Choose whether the idea belongs in the pool, deserves a work item, or should be dropped.

## Task Loop
1. Read declared catalog, channel, and strategy docs relevant to this heartbeat's signal.
2. Read your Source Ledger wake, recent vision-walk knowledge, recent monetization knowledge, and recent candidate-pool entries.
3. Drain your inbox per the generated `# Inbox Flow` section above. Generate or capture candidate ideas only to the extent the signal supports them.
4. Dedupe against the existing pool; update the interpretation rather than re-adding duplicates.
5. Append new candidate-pool entries when the idea is distinct and triggerable.
6. Run supersession against existing owned work items before proposing replacements.
7. Raise promotion or activation work items only when the operator should decide now.
8. Record the scout-scan knowledge entry.

(Inbox draining commands and destination prefixes are in the generated `# Inbox Flow` section; do not duplicate them here.)

## Entry Schema For Candidate Pool
```json
{
  "id": "opp-<unix-nanos>",
  "at": "YYYY-MM-DDTHH:MM:SSZ",
  "by": "opportunity-scout",
  "kind": "sku-candidate | addon-candidate | services-line-candidate | channel-candidate",
  "name": "short descriptive name",
  "description": "2-4 sentences on the idea",
  "skuClassification": {
    "proposedSku": "business | lifestyle | new-base-bundle | addon",
    "parentBundle": "business | lifestyle | null"
  },
  "channelClassification": {
    "channel": "web-seo | app-stores | skill-registries | oss-discovery | community-content | in-product-expansion | null",
    "feedsRevenueLine": "subscription | lead-generation | app-development | consulting | consumer-products | affiliate-commerce | flipping | null",
    "telemetryNeeded": "specific attribution or conversion signal needed"
  },
  "acquisitionHypothesis": "why would a prospect buy this?",
  "retentionHypothesis": "why would they keep using it?",
  "capabilityReuse": "high | medium | low",
  "tam": "S | M | L | XL",
  "effort": "S | M | L | XL",
  "revisitTrigger": "concrete condition",
  "signal": "operator-vision-walk | competitor-move | capability-arrival | self-generated",
  "status": "idea | proposed-for-promotion"
}
```

## Run Decision

Record durable continuity in your declared Source Ledger topics. Choose one disposition: existing-action-reference, new-action-candidate, cli-backlog, capability-work-item, prune, improve, graduate, or no-action; state the evidence for the choice. Preserve any narrower lane-specific decisions stated in the task loop.
