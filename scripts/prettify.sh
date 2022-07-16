#!/bin/bash
# These functions help to prettify echos

# Print header message
header() {
    echo "$(tput setaf 7)$1"
}

# Print info message
info() {
    echo "$(tput setaf 3)$1$(tput setaf 7)"
}

# Print success message
success() {
    echo "$(tput setaf 2)$1$(tput setaf 7)"
}

# Print error message
error() {
    echo "$(tput setaf 1)$1$(tput setaf 7)"
}