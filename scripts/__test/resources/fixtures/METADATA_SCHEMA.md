# Fixture Metadata Schema

This document defines the standardized schema for all fixture metadata files in the Vrooli test framework. Each fixture collection (images, audio, documents, workflows) must have a `metadata.yaml` file following this common format.

## 🏗️ Schema Structure

### Common Format

All `metadata.yaml` files must follow this structure:

```yaml
version: 1.0
type: images|audio|documents|workflows
description: "Brief description of this fixture collection"
schema:
  commonProperties:     # Properties all fixtures in this collection have
    - path             # Relative path from collection directory (REQUIRED)
    - tags             # Test categories (REQUIRED)  
    - testData         # Test-specific configuration (REQUIRED)
  typeProperties:      # Properties specific to this fixture type
    - [type-specific fields...]

[COLLECTION_NAME]:     # Main fixture data section
  # Structure varies by type but follows categorization patterns
  
testSuites:           # Test suite mappings (REQUIRED)
  suiteName:
    - fixture/path/1
    - fixture/path/2

# Additional type-specific sections allowed
[TYPE_SPECIFIC_SECTIONS]:
  # e.g. transcriptionExpectations, platforms, errorScenarios, etc.
```

### Required Fields

**All metadata files MUST include:**

1. **`version`** - Schema version (currently `1.0`)
2. **`type`** - Collection type (`images`, `audio`, `documents`, `workflows`)
3. **`description`** - Brief description of the fixture collection
4. **`schema`** - Schema definition with `commonProperties` and `typeProperties`
5. **`[COLLECTION_NAME]`** - Main fixture data (name matches `type`)
6. **`testSuites`** - Test suite mappings for data-driven testing

**All fixtures MUST include:**
- `path` - Relative path from collection directory
- `tags` - Array of test categories this fixture supports
- `testData` - Object with test-specific configuration/expectations

### Optional Fields

**Type-specific sections** are allowed to extend functionality:
- `transcriptionExpectations` (audio)
- `platforms` (workflows) 
- `integrationExpectations` (workflows)
- `errorScenarios` (workflows)
- `performanceBenchmarks` (workflows)

## 📁 File Naming Convention

- **Metadata files:** `metadata.yaml` (in each fixture collection directory)
- **Validation scripts:** `validate-[type].sh` (type-specific validation)

## 🔍 Validation Requirements

### Central Validation (`fixtures/validate-metadata.sh`)

Validates common requirements across all fixture collections:

1. **File existence:** `metadata.yaml` exists in each collection directory
2. **Schema compliance:** Required fields present and correctly typed
3. **Path validation:** All fixture paths exist and are accessible
4. **Test suite integrity:** All referenced fixtures exist
5. **Tag consistency:** Tags follow naming conventions
6. **Type-specific validator:** Calls collection-specific validation script

### Type-Specific Validation (`[collection]/validate-[type].sh`)

Validates type-specific requirements:

1. **Format validation:** File formats are correct for type
2. **Content validation:** Files contain expected content/structure  
3. **Integration testing:** Test fixtures with available resources
4. **Performance checks:** Verify fixtures meet performance requirements
5. **Type-specific schemas:** Validate type-specific sections

## 📊 Schema Examples

### Images Collection (`images/metadata.yaml`)

```yaml
version: 1.0
type: images
description: "Test images for computer vision and image processing tests"
schema:
  commonProperties: [path, tags, testData]
  typeProperties: [format, dimensions, fileSize]

images:
  synthetic:
    colors:
      - path: synthetic/colors/medium-blue.png
        format: png
        dimensions: [100, 100]
        fileSize: 217
        tags: [basic, color, format-conversion]
        testData:
          dominantColor: "#0000FF"

testSuites:
  basicUpload:
    - synthetic/colors/medium-blue.png
```

### Audio Collection (`audio/metadata.yaml`)

```yaml
version: 1.0
type: audio
description: "Test audio files for speech recognition and audio processing tests"
schema:
  commonProperties: [path, tags, testData]
  typeProperties: [format, duration, sampleRate, bitRate, channels, contentType]

audio:
  speech:
    - path: speech/speech_test.mp3
      format: mp3
      duration: 5
      sampleRate: 44100
      bitRate: 128
      channels: 1
      contentType: speech
      tags: [speech, transcription, short-form]
      testData:
        language: "en"
        expectedTranscription: "Hello world"

testSuites:
  transcriptionAccuracy:
    - speech/speech_test.mp3

transcriptionExpectations:
  speech_test.mp3:
    exactMatch: "Hello world"
    confidence: "high"
```

### Documents Collection (`documents/metadata.yaml`)

```yaml
version: 1.0
type: documents
description: "Test documents for document processing and text extraction tests"
schema:
  commonProperties: [path, tags, testData]
  typeProperties: [format, fileSize, pageCount, language]

documents:
  structured:
    - path: structured/customers.csv
      format: csv
      fileSize: 1024
      pageCount: 1
      language: "en"
      tags: [structured-data, csv, tabular]
      testData:
        hasHeaders: true
        rowCount: 100
        encoding: "utf-8"

testSuites:
  dataExtraction:
    - structured/customers.csv
```

### Workflows Collection (`workflows/metadata.yaml`)

```yaml
version: 1.0
type: workflows
description: "Test workflow definitions for automation platform integration tests"
schema:
  commonProperties: [path, tags, testData]
  typeProperties: [platform, integration, complexity, expectedDuration, resourceReqs]

workflows:
  n8n:
    - path: n8n/basic-workflow.json
      platform: n8n
      integration: [ollama]
      complexity: basic
      expectedDuration: 15
      resourceReqs:
        cpu: low
        memory: low
        network: true
      tags: [smoke-test, ai-integration]
      testData:
        model: "llama3.1:8b"
        expectedOutput: "AI response"

testSuites:
  smokeTests:
    - n8n/basic-workflow.json

platforms:
  n8n:
    port: 5678
    health_check: "/healthz"
```

## 🚨 Common Validation Errors

### Schema Errors
- Missing required fields (`version`, `type`, `description`, `schema`, collection name, `testSuites`)
- Invalid type value (must be `images`, `audio`, `documents`, or `workflows`)
- Schema version mismatch

### Path Errors
- Fixture paths don't exist on filesystem
- Paths are absolute instead of relative
- Referenced fixtures missing from test suites

### Content Errors
- Empty or malformed YAML syntax
- Missing required fixture properties (`path`, `tags`, `testData`)
- Invalid tag names (must be lowercase, hyphen-separated)

### Integration Errors
- Type-specific validator script missing or not executable
- Fixture files don't match declared format/properties
- Integration tests fail with available resources

## 📈 Migration Guide

When migrating existing YAML files to this schema:

1. **Rename file:** `[type].yaml` → `metadata.yaml` 
2. **Add required fields:** `type`, `description` if missing
3. **Restructure schema:** Move to `commonProperties`/`typeProperties` format
4. **Validate paths:** Ensure all paths are relative and exist
5. **Update references:** Fix any hardcoded filenames in documentation/scripts
6. **Test validation:** Run both central and type-specific validators

## 🔧 Tools and Scripts

- **Central validator:** `fixtures/validate-metadata.sh`
- **Type validators:** `[collection]/validate-[type].sh`
- **Schema checker:** Built into central validator
- **Integration tests:** Built into type-specific validators

This standardized approach ensures consistency, maintainability, and automatic discovery of fixture collections across the entire test framework.