#!/bin/bash
# test-golden-files.sh
# Purpose: Performs golden file testing for the Vrooli Helm chart.
# 
# This script renders the Helm chart with predefined value scenarios and compares
# the output to "golden" snapshot files stored in the 'goldenfiles/' directory.
# If the output differs from a golden file, it indicates that the chart's
# rendered Kubernetes manifests have changed.
#
# This helps detect unintended modifications to the chart's output and ensures
# that changes are reviewed and approved by updating the golden files.
#
# Usage:
#   1. To run tests: ./test-golden-files.sh
#   2. To update golden files (after verifying changes are intentional):
#      UPDATE_GOLDEN_FILES=true ./test-golden-files.sh
#
# Prerequisites:
#   - Helm CLI installed and in PATH.
#   - Script must be run from the chart's root directory (e.g., k8s/chart/)
#     or adjust paths accordingly.

set -e # Exit immediately if a command exits with a non-zero status.

CHART_PATH="."
GOLDEN_FILES_DIR="./tests/goldenfiles"
VALUES_FILE_DEFAULT="./values.yaml"
VALUES_FILE_DEV="./values-dev.yaml"
VALUES_FILE_PROD="./values-prod.yaml"

# Ensure the golden files directory exists
mkdir -p "${GOLDEN_FILES_DIR}"

# --- Helper Function to Test a Scenario ---
# Arguments:
#   $1: Scenario name (e.g., "default", "prod") - used for release name and golden file name.
#   $2: Helm template arguments (e.g., "-f values.yaml -f values-prod.yaml")
test_scenario() {
    local scenario_name="$1"
    # Remaining arguments are helm template flags for values files
    shift
    local helm_values_flags=("$@")

    local release_name="vrooli-${scenario_name}"
    local golden_file="${GOLDEN_FILES_DIR}/${scenario_name}.golden.yaml"
    local temp_output_file="${GOLDEN_FILES_DIR}/${scenario_name}.rendered.tmp.yaml"

    echo "-------------------------------------------------------------"
    echo "Testing scenario: ${scenario_name}"
    echo "Release name: ${release_name}"
    echo "Golden file: ${golden_file}"
    echo "Helm flags: ${helm_values_flags[*]}"
    echo "-------------------------------------------------------------"

    # Render the template
    # Note: The "helm template" command now requires the chart path argument
    # to be specified after the release name argument when also providing value files.
    helm template "${release_name}" "${CHART_PATH}" "${helm_values_flags[@]}" > "${temp_output_file}"

    if [ "${UPDATE_GOLDEN_FILES}" = "true" ]; then
        echo "UPDATE_GOLDEN_FILES is true. Updating golden file for ${scenario_name}..."
        mv "${temp_output_file}" "${golden_file}"
        echo "Golden file ${golden_file} updated."
    else
        if [ ! -f "${golden_file}" ]; then
            echo "ERROR: Golden file ${golden_file} does not exist!"
            echo "To create it, run: UPDATE_GOLDEN_FILES=true ./tests/test-golden-files.sh"
            # Clean up temp file before exiting
            rm "${temp_output_file}"
            return 1 # Indicate failure
        fi

        echo "Comparing output with ${golden_file}..."
        if diff --unified "${golden_file}" "${temp_output_file}"; then
            echo "SUCCESS: Output for ${scenario_name} matches golden file."
            rm "${temp_output_file}" # Clean up temp file
            return 0 # Indicate success
        else
            echo "ERROR: Output for ${scenario_name} does NOT match golden file ${golden_file}!"
            echo "Review the differences above. If they are intentional, update the golden file by running:"
            echo "  UPDATE_GOLDEN_FILES=true ./tests/test-golden-files.sh"
            # Do not delete temp_output_file so user can inspect it
            echo "Rendered output saved to: ${temp_output_file}"
            return 1 # Indicate failure
        fi
    fi
    echo ""
}

# --- Define Scenarios to Test ---
# Add more scenarios as needed

# Scenario 1: Default values
# test_scenario "default" -f "${VALUES_FILE_DEFAULT}"
# For this one, let's assume we only care about prod and dev for now,
# as default is often less used directly without overrides.

# Scenario 2: Development values
test_scenario "dev" -f "${VALUES_FILE_DEFAULT}" -f "${VALUES_FILE_DEV}"
SCENARIO_DEV_RESULT=$?

# Scenario 3: Production values
test_scenario "prod" -f "${VALUES_FILE_DEFAULT}" -f "${VALUES_FILE_PROD}"
SCENARIO_PROD_RESULT=$?

# Exit with an overall status code
if [ ${SCENARIO_DEV_RESULT} -ne 0 ] || [ ${SCENARIO_PROD_RESULT} -ne 0 ]; then
    echo "One or more golden file tests failed."
    exit 1
else
    echo "All golden file tests passed."
    exit 0
fi 