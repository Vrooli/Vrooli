Feature: Network ADB reconnects without hammering a sleeping TV
  An absent network ADB endpoint is reconnected during inventory, but a failed
  attempt backs off so a powered-off TV cannot tax every read.

  Scenario: One attempt at a time
    When a reconnect is requested twice at once
    Then only the first is admitted

  Scenario: A failed attempt backs off
    Given a failed reconnect just now
    When a reconnect is requested before the backoff ends
    Then it is refused
    When a reconnect is requested after the backoff ends
    Then it is admitted

  Scenario: Success clears the backoff
    Given a successful reconnect just now
    When a reconnect is requested before the backoff ends
    Then it is admitted

  Scenario: No gate means no reconnect
    Given no reconnect gate
    When a reconnect is requested before the backoff ends
    Then it is refused

  Scenario Outline: Display names resolve without inventory only when freshly observed
    Given a device "<device>" observed <age> ago and <health>
    When the name "<device>" is resolved without inventory
    Then the quick resolution <outcome>

    Examples:
      | device         | age     | health      | outcome  |
      | Living Room TV | 10s     | available   | succeeds |
      | Living Room TV | 1h      | available   | declines |
      | Living Room TV | 10s     | unreachable | declines |
