#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

SCRIPT_PATH="$BATS_TEST_DIRNAME/../utils/log.sh"
. "$SCRIPT_PATH"

@test "log::get_color_code returns correct codes" {
    [ "$(log::get_color_code RED)" = "1" ]
    [ "$(log::get_color_code GREEN)" = "2" ]
    [ "$(log::get_color_code YELLOW)" = "3" ]
    [ "$(log::get_color_code BLUE)" = "4" ]
    [ "$(log::get_color_code MAGENTA)" = "5" ]
    [ "$(log::get_color_code CYAN)" = "6" ]
    [ "$(log::get_color_code WHITE)" = "7" ]
    [ "$(log::get_color_code INVALID)" = "0" ]
}
