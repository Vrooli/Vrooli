Feature: J17 composed funding and reservation lifecycle
  # Traces REQ-P5, REQ-F3 and REQ-C4 to the real LPBS Postgres/Door evidence.

  Scenario: unavailable funding refuses without creating money effects
    Given a real Postgres funding lot is unavailable to the configured Door mode
    When the request passes through the LPBS Door and embedded Gateway
    Then the request is refused
    And no reservation, allocation, usage record, usage event, or ledger row is created

  Scenario: a spendable lot settles once and exact replay has no extra effect
    Given a settled spendable funding lot in the matching account and mode
    When the request passes through the LPBS Door and embedded Gateway
    Then the reservation is finalized within its limit
    And allocations, charged credits, usage, and wallet remain consistent
    When the same request is replayed
    Then the replay receives the documented HTTP 409 settled-operation response
    And the persisted reservation still has the original operation identity
    And reservation, allocation, usage record, usage event, ledger, and wallet snapshots remain unchanged
