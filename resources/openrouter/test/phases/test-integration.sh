#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
TEST_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${TEST_DIR}/.." && builtin pwd)"

manual_models_file="$(mktemp)"
trap 'rm -f "${manual_models_file}"' EXIT

printf '%s\n' '[{"id":"openai/gpt-test","name":"OpenAI Test Model","display_name":"OpenAI Test Model","provider":"openai","description":"Synthetic model for integration tests","context_length":8192,"pricing":{"prompt":0.000001,"completion":0.000002}}]' > "${manual_models_file}"

cd "${RESOURCE_DIR}/cli"
go test ./...
OPENROUTER_MANUAL_MODELS_FILE="${manual_models_file}" go run . content models --json --search gpt-test | grep -q '"id": "openai/gpt-test"'
OPENROUTER_MANUAL_MODELS_FILE="${manual_models_file}" go run . list-models --provider openai --search gpt-test | grep -q 'openai/gpt-test'
go run . configure --api-key sk-or-test-key >/dev/null
go run . show-config | grep -q 'Credentials File:'
