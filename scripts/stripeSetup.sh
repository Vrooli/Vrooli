#!/bin/bash
# This scripts sets up the Stripe CLI for local testing on Debian/Ubuntu distributions.
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
. "${HERE}/utils.sh"

header "Setting up the Stripe CLI..."

info "Adding Stripe CLI’s GPG key to the apt sources keyring..."
curl -s https://packages.stripe.dev/api/security/keypair/stripe-cli-gpg/public | gpg --dearmor | sudo tee /usr/share/keyrings/stripe.gpg

info "Adding CLI’s apt repository to the apt sources list..."
echo "deb [signed-by=/usr/share/keyrings/stripe.gpg] https://packages.stripe.dev/stripe-cli-debian-local stable main" | sudo tee -a /etc/apt/sources.list.d/stripe.list

info "Updating the package list..."
sudo apt update

info "Installing the CLI..."
sudo apt install stripe

info "Verifying Stripe CLI installation..."
stripe version

info "Running test command. This will start the login process if the CLI is not already logged in..."
stripe customers list --limit=1

success "Stripe CLI setup complete!"
