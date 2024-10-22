#!/usr/bin/env bats
bats_require_minimum_version 1.5.0
load '__testHelper.bash'

SCRIPT_PATH="$BATS_TEST_DIRNAME/../utils.sh"
. "$SCRIPT_PATH"

# Test color code retrieval
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

# Tests for prompt_confirm function
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
    [ "${lines[0]}" = "Test error message" ]
}

@test "exit_with_error exits with provided message and custom code" {
    run exit_with_error "Custom error message" 2
    [ "$status" -eq 2 ]
    [ "${lines[0]}" = "Custom error message" ]
}
