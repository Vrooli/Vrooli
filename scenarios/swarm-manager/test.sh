#!/usr/bin/env bash

# Swarm Manager Test Script
# Verifies basic functionality

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI="${SCRIPT_DIR}/cli.sh"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Testing Swarm Manager${NC}"
echo "====================="

# Test 1: Check CLI help
echo -e "\n${BLUE}Test 1: CLI Help${NC}"
if $CLI help | grep -q "Swarm Manager"; then
    echo -e "${GREEN}✓ CLI help works${NC}"
else
    echo -e "${RED}✗ CLI help failed${NC}"
    exit 1
fi

# Test 2: Check directory structure
echo -e "\n${BLUE}Test 2: Directory Structure${NC}"
required_dirs=(
    "tasks/active"
    "tasks/backlog/manual"
    "tasks/backlog/generated"
    "tasks/staged"
    "tasks/completed"
    "tasks/failed"
    "prompts"
    "config"
    "logs"
)

all_exist=true
for dir in "${required_dirs[@]}"; do
    if [ -d "${SCRIPT_DIR}/${dir}" ]; then
        echo -e "${GREEN}✓ ${dir} exists${NC}"
    else
        echo -e "${RED}✗ ${dir} missing${NC}"
        all_exist=false
    fi
done

if ! $all_exist; then
    echo -e "${RED}Directory structure incomplete${NC}"
    exit 1
fi

# Test 3: Check configuration files
echo -e "\n${BLUE}Test 3: Configuration Files${NC}"
config_files=(
    "config/settings.yaml"
    "config/priority-weights.yaml"
    "config/scenario-registry.yaml"
)

for file in "${config_files[@]}"; do
    if [ -f "${SCRIPT_DIR}/${file}" ]; then
        echo -e "${GREEN}✓ ${file} exists${NC}"
    else
        echo -e "${RED}✗ ${file} missing${NC}"
        exit 1
    fi
done

# Test 4: Check prompts
echo -e "\n${BLUE}Test 4: Prompt Files${NC}"
prompt_files=(
    "prompts/task-analyzer.md"
    "prompts/task-executor.md"
    "prompts/backlog-generator.md"
)

for file in "${prompt_files[@]}"; do
    if [ -f "${SCRIPT_DIR}/${file}" ]; then
        echo -e "${GREEN}✓ ${file} exists${NC}"
    else
        echo -e "${RED}✗ ${file} missing${NC}"
        exit 1
    fi
done

# Test 5: Add a test task
echo -e "\n${BLUE}Test 5: Task Management${NC}"
test_task_title="Test task $(date +%s)"
$CLI add-task "$test_task_title"

# Check if task was created
task_count=$(find "${SCRIPT_DIR}/tasks/backlog/manual" -name "*.yaml" | wc -l)
if [ "$task_count" -gt 0 ]; then
    echo -e "${GREEN}✓ Task creation works${NC}"
else
    echo -e "${RED}✗ Task creation failed${NC}"
    exit 1
fi

# Test 6: List tasks
echo -e "\n${BLUE}Test 6: List Tasks${NC}"
if $CLI list-tasks >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Task listing works${NC}"
else
    echo -e "${RED}✗ Task listing failed${NC}"
    exit 1
fi

# Test 7: Check example tasks
echo -e "\n${BLUE}Test 7: Example Tasks${NC}"
example_files=(
    "examples/task-improve-n8n-reliability.yaml"
    "examples/task-create-invoice-generator.yaml"
)

for file in "${example_files[@]}"; do
    if [ -f "${SCRIPT_DIR}/${file}" ]; then
        echo -e "${GREEN}✓ ${file} exists${NC}"
    else
        echo -e "${RED}✗ ${file} missing${NC}"
    fi
done

# Test 8: Check API source
echo -e "\n${BLUE}Test 8: API Source${NC}"
if [ -f "${SCRIPT_DIR}/api/main.go" ]; then
    echo -e "${GREEN}✓ API source exists${NC}"
elif [ -f "${SCRIPT_DIR}/api/swarm-manager-api" ]; then
    echo -e "${GREEN}✓ API binary exists${NC}"
else
    echo -e "${RED}✗ API source/binary missing${NC}"
    exit 1
fi

# Test 9: Check UI files
echo -e "\n${BLUE}Test 9: UI Files${NC}"
ui_files=(
    "ui/index.html"
    "ui/src/app.js"
    "ui/src/styles.css"
)

for file in "${ui_files[@]}"; do
    if [ -f "${SCRIPT_DIR}/${file}" ]; then
        echo -e "${GREEN}✓ ${file} exists${NC}"
    else
        echo -e "${RED}✗ ${file} missing${NC}"
        exit 1
    fi
done

# Test 10: Check schedulers implementation
echo -e "\n${BLUE}Test 10: Scheduler Implementation${NC}"
if grep -q "startSchedulers" "${SCRIPT_DIR}/api/main.go" 2>/dev/null; then
    echo -e "${GREEN}✓ Internal schedulers implemented${NC}"
else
    echo -e "${RED}✗ Schedulers not found in API${NC}"
    exit 1
fi

# Summary
echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}All tests passed! Swarm Manager is ready.${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "To start using Swarm Manager:"
echo "  1. Run: swarm-manager start"
echo "  2. Open UI: http://localhost:31011"
echo "  3. Add tasks: swarm-manager add-task 'Your task here'"
echo ""
echo "For full autonomy (YOLO mode):"
echo "  swarm-manager config set yolo_mode true"