Feature: Try an app on a phone in one call
  Install an APK, prove it is installed by package state, open it, and screenshot
  it. Each step is verified, and a failed install never reaches open or screenshot.

  Background:
    Given a phone that can install and screenshot, reporting version code 3 after an install

  Scenario: Install, open and screenshot
    Given an APK for "com.acme.demo" version code 3
    When the APK is installed and opened and screenshotted
    Then the install status is "installed_verified"
    And the installed app is in the foreground
    And a screenshot was retained

  Scenario: Install alone does not open or screenshot
    Given an APK for "com.acme.demo" version code 3
    When the APK is installed
    Then the install status is "installed_verified"
    And no screenshot was taken

  Scenario: An accepted install that package state does not show is a failure
    Given an APK for "com.acme.demo" version code 3
    And the device accepts an install but installs nothing
    When the APK is installed and opened and screenshotted
    Then the install status is "failed"
    And the install detail contains "package-state does not show"
    And no screenshot was taken

  Scenario: A different installed version code is a failure
    Given an APK for "com.acme.demo" version code 9
    When the APK is installed
    Then the install status is "failed"
    And the install detail contains "differs from the APK"

  Scenario Outline: Only an absolute path to a real .apk file is accepted
    Given a file at "<path>" that is not an APK path
    Then installing it is refused with "<reason>"

    Examples:
      | path             | reason                 |
      | relative/app.apk | absolute path          |
      | /tmp/app.txt     | .apk file              |
      | /tmp/missing.apk | readable regular file  |
