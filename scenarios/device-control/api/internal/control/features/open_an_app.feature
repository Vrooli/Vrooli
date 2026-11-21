Feature: Open an app by name
  One call opens Netflix, YouTube or any package, proves it reached the
  foreground, and never launches what is not installed.

  Scenario Outline: App names resolve to candidate packages
    When the app name <name> is resolved
    Then the candidate packages are <packages>

    Examples:
      | name          | packages                                                   |
      | Netflix       | com.netflix.ninja,com.netflix.mediaclient                  |
      | youtube       | com.google.android.youtube.tv,com.google.android.youtube   |
      | com.acme.tool | com.acme.tool                                              |

  Scenario: An unknown app name is refused with the known names
    When the app name "nonexistent app" is resolved
    Then resolution fails with "unknown app name"

  Scenario: The TV build of an app is opened and verified
    Given a device with "com.netflix.ninja" installed
    When "netflix" is opened
    Then the launch status is "foreground_verified"
    And the launched package is "com.netflix.ninja"
    And no lease remains held

  Scenario: Only the phone build is installed
    Given a device with "com.netflix.mediaclient" installed
    When "netflix" is opened
    Then the launched package is "com.netflix.mediaclient"
    And the launch status is "foreground_verified"

  Scenario: A composed TV opens the app through the transport that can launch it
    Given a TV whose default Cast transport cannot launch apps but whose ADB transport can, with "com.netflix.ninja" installed
    When "netflix" is opened
    Then the launch status is "foreground_verified"
    And the launched package is "com.netflix.ninja"

  Scenario: An app that is not installed is never launched
    Given a device with nothing installed
    When "netflix" is opened
    Then the launch status is "not_installed"
    And no launch was issued

  Scenario Outline: The foreground is read from the device's independent signals
    When the device reports resumed activity <resumed> and focused window <focused>
    Then the observed foreground package is <package>

    Examples:
      | resumed                          | focused                        | package          |
      | com.netflix.ninja/.MainActivity  | com.netflix.ninja/.MainActivity | com.netflix.ninja |
      | com.netflix.ninja/.MainActivity  | none                           | com.netflix.ninja |
      | none                             | com.android.tv.settings/.Main  | com.android.tv.settings |
      | com.netflix.ninja/.MainActivity  | com.other/.Main                | nothing           |
      | none                             | none                           | nothing           |
