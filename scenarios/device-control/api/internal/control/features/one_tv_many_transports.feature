Feature: One TV, several transports
  A Cast TV and its ADB endpoint on the same host are one physical device once
  the owner confirms it. Until then they stay separate, and a capture that
  cannot be served says exactly what to run next.

  Background:
    Given a Cast TV named "SmartTV 4K" at 192.168.1.158

  Scenario: Confirming the relation yields one device with both transports
    Given an ADB endpoint on the same host reporting model "SmartTV-4K"
    When the owner confirms their same-host relation
    Then the Cast TV is the canonical device
    And the ADB identity no longer exists on its own
    And the canonical device has an "android-adb" transport
    And the asserted claim is "adb-serial=AE70A4D38B"

  Scenario: Different models are refused unless the owner forces it
    Given an ADB endpoint on the same host reporting model "Pixel 8"
    When the owner confirms their same-host relation
    Then the confirmation is refused with "different models"
    When the owner forces the confirmation
    Then the relation is confirmed

  Scenario: An unknown candidate points at the relations listing
    When the owner confirms the candidate "candidate:nope"
    Then the confirmation is refused with "device-control device relations"

  Scenario: A screenshot the Cast transport cannot serve names the confirm command
    Given an ADB endpoint on the same host reporting model "SmartTV-4K"
    When a screenshot of the Cast TV is requested
    Then the capture is a capability gap
    And the first next action starts with "device-control device relation-confirm candidate:"

  Scenario: A TV with no ADB identity is told how to onboard one on its own host
    When a screenshot of the Cast TV is requested
    Then the capture is a capability gap
    And the first next action contains "onboard-network --endpoint 192.168.1.158:5555 --attach-to google-cast:tv"

  Scenario Outline: A failed capture says how to recover from the reason it gave
    Given a phone named "moto"
    When a capture fails with the reason <reason> and failure class <class>
    Then the next action contains <command>

    Examples:
      | reason                                                      | class     | command                               |
      | visible surface unavailable: lock state is "locked"        | none      | UNLOCK-HINT                  |
      | screen is off                                               | app-state | device screenshot phone-1             |

  Scenario Outline: ADB owner actions
    When the ADB transport on "SmartTV 4K" over <transport> is <health> because "<reason>"
    Then the owner action contains <action>

    Examples:
      | transport | health       | reason             | action                                    |
      | wireless  | unauthorized | none               | accept the "Allow USB debugging" prompt   |
      | wireless  | unreachable  | device not present | device-control device reconnect tv        |
      | usb       | unreachable  | offline            | data cable                                |
      | wireless  | available    | none               | nothing                                   |
