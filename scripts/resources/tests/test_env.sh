export TEST_ID="test_$(date +%s)"
export TEST_TIMEOUT="30"
export TEST_VERBOSE="true"
export TEST_CLEANUP="true"
export SCRIPT_DIR="/home/matthalloran8/Vrooli/scripts/resources/tests"
export RESOURCES_DIR="/home/matthalloran8/Vrooli/scripts/resources"
# Export healthy resources as a space-separated string
export HEALTHY_RESOURCES_STR="ollama"
# Recreate the array from the string
HEALTHY_RESOURCES=($HEALTHY_RESOURCES_STR)
