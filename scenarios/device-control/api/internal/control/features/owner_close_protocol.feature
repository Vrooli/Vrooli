Feature: Closing an owner-held resource
  Pairing handshakes and native recordings close through one protocol. A close
  that fails leaves the resource uncertain; it is never replayed blindly.

  Scenario Outline: Close protocol transitions
    Given a resource that is <start>
    When a close begins
    Then the close is <admission>
    And the resource is <after>

    Examples:
      | start     | admission | after     |
      | open      | admitted  | closing   |
      | closing   | refused   | closing   |
      | uncertain | refused   | uncertain |

  Scenario Outline: Close results
    Given a resource that is closing
    When the close finishes with <outcome>
    Then the resource is <after>

    Examples:
      | outcome | after     |
      | success | removed   |
      | failure | uncertain |

  Scenario: A result for a resource that is not closing changes nothing
    Given a resource that is open
    When the close finishes with success
    Then the resource is open
