# Professional integrations and monetization contract examples

These examples are illustrative design anchors for the Plan Manager plan
`professional-integrations-monetization-ux`. They are not production contracts
and must be reconciled with existing generated APIs, ownership rules, and the
first concrete Integration Hub connector before implementation.

## Consumer-facing connection view

```ts
export type IntegrationConnectionView = {
  id: string;
  connectorId: string;
  providerName: string;
  displayName: string;
  identityLabel?: string;
  status:
    | "healthy"
    | "checking"
    | "needs_attention"
    | "disconnected"
    | "revoked"
    | "expired"
    | "insufficient_scope"
    | "provider_unavailable";
  scopes: Array<{ id: string; label: string; granted: boolean }>;
  boundScenarios: Array<{ id: string; label: string }>;
  lastCheckedAt?: string;
  secretState: "configured" | "missing";
  supportedActions: Array<
    "test" | "reconnect" | "refresh" | "rotate" | "bind" | "unbind" | "revoke" | "delete"
  >;
  nextStep?: { label: string; action: string };
};
```

The browser receives safe metadata and action capability. It never receives an
API key, access token, refresh token, credential-authority value, or provider
secret. `supportedActions` is derived from backend capability, not guessed by
the client.

## Commercial-context contract sketch

```proto
message GetCommercialContextRequest {
  repeated string placements = 1;
  string scenario_id = 2;
}

message CommercialContext {
  AccountFacts account = 1;                 // authoritative account facts
  repeated CommercialNotice notices = 2;    // presentation-only
  repeated CommercialOffer offers = 3;      // presentation-only
  google.protobuf.Timestamp generated_at = 4;
  google.protobuf.Timestamp expires_at = 5;
  string freshness_token = 6;
}

message CommercialOffer {
  string id = 1;                            // stable dismissal/analytics key
  string placement = 2;                     // server-approved placement
  string relevance_key = 3;                 // explains why it is shown
  bool eligible = 4;                        // server-evaluated only
  string cta_uri = 5;                       // owned destination
  google.protobuf.Timestamp expires_at = 6;
  bool dismissible = 7;
}
```

The exact field names and package are implementation decisions. The hard
invariant is that cached notices/offers can influence presentation only; they
cannot grant an entitlement or replace signed-lease and trusted-service
enforcement.

## Optional-content cache policy

```json
{
  "placements": ["integrations"],
  "cache": {
    "ttlSeconds": 900,
    "staleWhileRevalidateSeconds": 3600,
    "deduplicateInFlightRequests": true,
    "failOpenForCoreSettings": true
  },
  "rendering": {
    "showOnlyServerEligible": true,
    "hideExpired": true,
    "allowGenericMarketplaceOffers": false
  }
}
```

This policy is a starting point for tests and documentation, not permission to
invent a new client-side commercial authority. The final values must follow
LPBS operational constraints and the scenario's privacy/security review.
