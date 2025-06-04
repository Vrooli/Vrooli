#!/usr/bin/env bash
# This script sets up the Stripe CLI for local testing on Debian/Ubuntu distributions.
# It expects logging functions (header, info, success, error) to be available from the calling script.

SETUP_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/log.sh"
# shellcheck disable=SC1091
source "${SETUP_DIR}/../utils/exit_codes.sh"

stripe_cli::setup() {
    log::header "Setting up the Stripe CLI..."

    # Check if Stripe CLI is already installed
    if command -v stripe &> /dev/null; then
        log::info "Stripe CLI is already installed. Verifying version:"
        stripe version
        log::success "Stripe CLI setup check complete."
        return 0 # Use return for function success
    fi

    log::info "Stripe CLI not found. Proceeding with installation."

    # Ensure the script doesn't fail if any command fails, but log errors
    # Using a subshell for set -e and trap to isolate their effects
    (
        set -e # Exit immediately if a command exits with a non-zero status.
        trap 'log::error "An error occurred during Stripe CLI setup. Please check the output above."; exit "${ERROR_OPERATION_FAILED:?ERROR_OPERATION_FAILED is not set}";' ERR

        # Check for sudo privileges early, as most commands require it.
        if [ "$(id -u)" -ne 0 ]; then
            log::info "Attempting to use sudo for required commands."
            # Test sudo upfront
            if ! sudo -n true 2>/dev/null; then
                log::error "Sudo privileges are required and passwordless sudo is not available, or sudo is not installed."
                log::error "Please run the main setup script with sudo, or ensure sudo access without a password for this script."
                exit "${ERROR_SUDO_REQUIRED:?ERROR_SUDO_REQUIRED is not set}"
            fi
        fi

        log::info "Adding Stripe CLI's GPG key to the apt sources keyring..."
        curl -s https://packages.stripe.dev/api/security/keypair/stripe-cli-gpg/public | gpg --dearmor | sudo tee /usr/share/keyrings/stripe.gpg > /dev/null

        log::info "Adding CLI's apt repository to the apt sources list..."
        echo "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main" | sudo tee /etc/apt/sources.list.d/stripe.list > /dev/null

        log::info "Updating the package list..."
        sudo apt-get update -qq

        log::info "Installing the Stripe CLI..."
        sudo apt-get install -y stripe

        log::info "Verifying Stripe CLI installation..."
        if command -v stripe &> /dev/null; then
            stripe version
            log::info "Running a test command (this might initiate login if not already authenticated)..."
            stripe customers list --limit=1 || log::info "Test command 'stripe customers list' run (may require login if CLI is not authenticated)."
            log::success "Stripe CLI setup complete!"
        else
            log::error "Stripe CLI installation appears to have failed."
            exit "${ERROR_INSTALLATION_FAILED:?ERROR_INSTALLATION_FAILED is not set}"
        fi
    )

    # Check the subshell exit status
    local subshell_exit_status=$?
    if [ "${subshell_exit_status}" -ne 0 ]; then
        log::error "Stripe CLI setup function failed."
        return 1 # Propagate failure
    fi

    return 0 # Explicitly return success from function
}

# End of script. Functions are defined above and intended to be called by a sourcing script. 