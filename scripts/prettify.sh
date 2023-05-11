#!/bin/bash
# These functions help to prettify echos

export TERM=${TERM:-xterm}

# Determine if tput is available
if [ -n "$(command -v tput)" ]; then
    # Set colors
    RED=$(tput setaf 1)
    GREEN=$(tput setaf 2)
    YELLOW=$(tput setaf 3)
    BLUE=$(tput setaf 4)
    MAGENTA=$(tput setaf 5)
    CYAN=$(tput setaf 6)
    WHITE=$(tput setaf 7)
    ORANGE=$(tput setaf 208)
    RESET=$(tput sgr0)
else
    RED=""
    GREEN=""
    YELLOW=""
    BLUE=""
    MAGENTA=""
    CYAN=""
    WHITE=""
    ORANGE=""
    RESET=""
fi

# Print header message
header() {
    echo "${MAGENTA}${1}${RESET}"
}

# Print info message
info() {
    echo "${CYAN}${1}${RESET}"
}

# Print success message
success() {
    echo "${GREEN}${1}${RESET}"
}

# Print error message
error() {
    echo "${RED}${1}${RESET}"
}

# Print warning message
warning() {
    echo "${YELLOW}${1}${RESET}"
}

# Print input prompt message
prompt() {
    echo "${ORANGE}${1}${RESET}"
}
