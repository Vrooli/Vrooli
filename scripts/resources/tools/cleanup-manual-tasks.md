# Manual Cleanup Tasks Required

## Tasks Requiring sudo Access

### PostgreSQL Instances (6 remaining)
```bash
sudo rm -rf /home/matthalloran8/Vrooli/scripts/resources/storage/postgres/instances/test-client
sudo rm -rf /home/matthalloran8/Vrooli/scripts/resources/storage/postgres/instances/test-comprehensive
sudo rm -rf /home/matthalloran8/Vrooli/scripts/resources/storage/postgres/instances/test-comprehensive-2
sudo rm -rf /home/matthalloran8/Vrooli/scripts/resources/storage/postgres/instances/network-test-new
sudo rm -rf /home/matthalloran8/Vrooli/scripts/resources/storage/postgres/instances/ecommerce-test
sudo rm -rf /home/matthalloran8/Vrooli/scripts/resources/storage/postgres/instances/better-port-check
```

### Agent-S2 Testing Directory
```bash
sudo rm -rf /home/matthalloran8/Vrooli/scripts/resources/agents/agent-s2/testing
```

## Tasks for Manual Review

### Whisper Audio Test Files
- Location: `/home/matthalloran8/Vrooli/scripts/resources/ai/whisper/tests/`
- Contains audio files that may be unique test fixtures
- Action: Review and either move to centralized fixtures or remove

## Completed Cleanups

### Phase 1: PostgreSQL
- ✅ Removed 12 test instances
- ⏳ 6 instances need sudo removal

### Phase 2: Test Directories
- ✅ agent-s2: Removed backups/, preserved Python unit tests
- ✅ unstructured-io: Removed test directories, moved PDFs to centralized fixtures
- ✅ windmill: Removed fixtures/ and test_fixtures/
- ✅ node-red: Removed test_fixtures/
- ✅ huginn: Removed test-fixtures/ and test_fixtures/
- ✅ comfyui: Removed test/
- ℹ️ whisper: Preserved tests/ for manual review (contains audio files)
- ℹ️ claude-code: Kept sandbox/test-files/ (contains examples)

### Next: Phase 3
Examples directories rationalization in progress...