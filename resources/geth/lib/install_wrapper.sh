#!/usr/bin/env bash
# Geth Install Wrapper Functions for v2.0 CLI

# Execute installation
geth::install::execute() {
    geth::install "$@"
}

# Execute uninstallation
geth::install::uninstall() {
    geth::uninstall
}