Feature: An expiring pairing handshake stays visible until its close is proven
  The lifecycle inventory must never report an idle owner while a pairing
  session is still being closed, and a close that fails is never replayed.

  Scenario: A close that returns cleanly clears the owner
    Given an expired pairing session whose close will succeed
    When the expiry sweep starts closing it
    Then the lifecycle inventory shows the pairing as "closing"
    When the close returns
    Then the lifecycle inventory shows no pairing owner

  Scenario: A close that fails leaves an explicit uncertain blocker
    Given an expired pairing session whose close will fail
    When the expiry sweep starts closing it
    Then the lifecycle inventory shows the pairing as "closing"
    And completing the pairing is refused
    And the session has been closed exactly once
    When the close returns
    Then the lifecycle inventory shows the pairing as "uncertain"
    And completing the pairing is refused
    And the session has been closed exactly once
