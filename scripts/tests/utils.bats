#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
load '__testHelper.bash'

SCRIPT_PATH="$BATS_TEST_DIRNAME/../utils.sh"
. "$SCRIPT_PATH"

@test "get_color_code returns correct codes" {
    [ "$(get_color_code RED)" = "1" ]
    [ "$(get_color_code GREEN)" = "2" ]
    [ "$(get_color_code YELLOW)" = "3" ]
    [ "$(get_color_code BLUE)" = "4" ]
    [ "$(get_color_code MAGENTA)" = "5" ]
    [ "$(get_color_code CYAN)" = "6" ]
    [ "$(get_color_code WHITE)" = "7" ]
    [ "$(get_color_code INVALID)" = "0" ]
}

@test "prompt_confirm accepts 'y' input" {
    echo "y" | {
        run prompt_confirm "Do you want to continue?"
        [ "$status" -eq 0 ]
    }
}
@test "prompt_confirm accepts 'Y' input" {
    echo "Y" | {
        run prompt_confirm "Do you want to continue?"
        [ "$status" -eq 0 ]
    }
}
@test "prompt_confirm rejects 'n' input" {
    echo "n" | {
        run prompt_confirm "Do you want to continue?"
        [ "$status" -eq 1 ]
    }
}
@test "prompt_confirm rejects any other input" {
    echo "z" | {
        run prompt_confirm "Do you want to continue?"
        [ "$status" -eq 1 ]
    }
}

# Tests for exit_with_error function
@test "exit_with_error exits with provided message and default code" {
    run exit_with_error "Test error message"
    [ "$status" -eq 1 ]
    [ "${lines[0]}" = "[ERROR]   Test error message" ]
}
@test "exit_with_error exits with provided message and custom code" {
    run exit_with_error "Custom error message" 2
    [ "$status" -eq 2 ]
    [ "${lines[0]}" = "[ERROR]   Custom error message" ]
}

@test "run_step returns 0 if the command is successful" {
    run run_step "test success command" "true"
    # Because run_step calls exit on failure, if 'true' passes,
    # we expect an exit code of 0 from the function itself.
    [ "$status" -eq 0 ]

    # Bats captures all output in ${lines[@]}.
    # Check if it prints the success logs:
    [ "${lines[0]}" = "[INFO]    test success command..." ]
    [ "${lines[1]}" = "[SUCCESS] test success command - done!" ]
}
@test "run_step exits with code 1 if the command fails" {
    # 'false' returns non-zero => run_step should exit with error.
    run run_step "test failing command" "false"
    [ "$status" -eq 1 ]

    # Check the last line for the expected error message
    [ "${lines[-1]}" = "[ERROR]   Failed: test failing command" ]
}

@test "run_step_noncritical returns 0 if the command succeeds" {
    run run_step_noncritical "test noncritical success" "true"
    # If the command is successful, we expect status 0
    [ "$status" -eq 0 ]

    # Output checks
    [ "${lines[0]}" = "[INFO]    test noncritical success..." ]
    [ "${lines[1]}" = "[SUCCESS] test noncritical success - done!" ]
}
@test "run_step_noncritical returns 1 if the command fails but does not exit the script" {
    run run_step_noncritical "test noncritical fail" "false"
    [ "$status" -eq 1 ]

    # Should still have printed the warning, not an error
    [ "${lines[-1]}" = "[WARNING] Non-critical step failed: test noncritical fail" ]
}

@test "is_yes returns 0 for 'y' input" {
    run is_yes "y"
    [ "$status" -eq 0 ]
}
@test "is_yes returns 0 for 'Y' input" {
    run is_yes "Y"
    [ "$status" -eq 0 ]
}
@test "is_yes returns 0 for 'yes' input" {
    run is_yes "yes"
    [ "$status" -eq 0 ]
}
@test "is_yes returns 0 for 'YES' input" {
    run is_yes "YES"
    [ "$status" -eq 0 ]
}
@test "is_yes returns 1 for 'n' input" {
    run is_yes "n"
    [ "$status" -eq 1 ]
}
@test "is_yes returns 1 for 'no' input" {
    run is_yes "no"
    [ "$status" -eq 1 ]
}
@test "is_yes returns 1 for random input" {
    run is_yes "random-string"
    [ "$status" -eq 1 ]
}
