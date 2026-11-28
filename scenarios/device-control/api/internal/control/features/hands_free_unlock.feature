Feature: A locked phone unlocks itself through its own profile
  The one-call verbs wake a phone and, when a swipe does not unlock it, use the
  device's active unlock profile. The PIN lives only in the credential
  authority. A profile whose last attempt was wrong is never retried, so the
  phone's lockout counter is safe, and the owner is told the exact setup step.

  Background:
    Given a locked phone "target"

  Scenario: An active profile unlocks the phone without being named
    Given an active unlock profile whose last outcome was "already_unlocked"
    When the phone is made ready
    Then the phone is unlocked
    And the result never contains the PIN

  Scenario: A profile whose last attempt was wrong is not retried
    Given an active unlock profile whose last outcome was "wrong_credential"
    When the phone is made ready
    Then making it ready is refused as still locked
    And the unlock guidance contains "saved PIN last failed"
    And the unlock guidance contains "auth provision"

  Scenario: A phone with no profile is told the one-time setup
    When the phone is made ready
    Then making it ready is refused as still locked
    And the unlock guidance contains "auth create --device target --method pin"
    And the unlock guidance contains "auth provision"

  Scenario Outline: Which profile a verb may use
    Given profiles for "target" with last outcomes <outcomes>
    Then the usable profile outcome is <chosen>

    Examples:
      | outcomes                              | chosen          |
      | wrong_credential                      | none            |
      | wrong_credential,unlocked             | unlocked        |
      | verification_failed,already_unlocked  | already_unlocked |
      | none                                  | untried         |
