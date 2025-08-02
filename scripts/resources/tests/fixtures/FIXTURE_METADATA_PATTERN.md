# Fixture Metadata Pattern

> **⚠️ DEPRECATED**: This file has been superseded by the new standardized metadata system.
> 
> **👉 See [METADATA_SCHEMA.md](./METADATA_SCHEMA.md) for the current metadata specification.**

## Migration Notice

As of the metadata standardization initiative, all fixture collections now use:

- **Standardized filename**: `metadata.yaml` (in each fixture collection directory)
- **Common schema format**: Documented in [METADATA_SCHEMA.md](./METADATA_SCHEMA.md)
- **Central validation**: `fixtures/validate-metadata.sh` for schema compliance
- **Type-specific validation**: `[collection]/validate-[type].sh` for detailed validation

## Previous Pattern (Deprecated)

The previous pattern used collection-specific YAML filenames:
- `images/images.yaml` → **Now**: `images/metadata.yaml`
- `audio/audio.yaml` → **Now**: `audio/metadata.yaml` 
- `workflows/workflows.yaml` → **Now**: `workflows/metadata.yaml`
- `documents/[none]` → **Now**: `documents/metadata.yaml`

## Migration Benefits

The new standardized approach provides:

1. **Consistency**: All collections use `metadata.yaml`
2. **Automation**: Scripts can discover collections by globbing `*/metadata.yaml`
3. **Validation**: Common schema validation across all fixture types
4. **Maintainability**: Shared validation logic reduces duplication
5. **Scalability**: Clear pattern for adding new fixture types

## Current Schema Overview

```yaml
version: 1.0
type: images|audio|documents|workflows
description: "Brief description of fixture collection"
schema:
  commonProperties: [path, tags, testData]  # Required for all fixtures
  typeProperties: [...]                     # Specific to fixture type
[COLLECTION_NAME]:
  # Fixture data organized by category
testSuites:
  # Test suite mappings for data-driven testing
# Type-specific sections as needed
```

## Documentation

For complete documentation of the current metadata system:

- **Schema specification**: [METADATA_SCHEMA.md](./METADATA_SCHEMA.md)
- **Central validator**: [validate-metadata.sh](./validate-metadata.sh)
- **Hub documentation**: [README.md](./README.md)

## Migration Status

- ✅ `images/metadata.yaml` - Migrated
- ✅ `audio/metadata.yaml` - Migrated  
- ✅ `workflows/metadata.yaml` - Migrated
- ✅ `documents/metadata.yaml` - Created
- ✅ Central validation system - Implemented
- ✅ Type-specific validators - Created
- ✅ Documentation updates - Completed

---

**For current metadata usage, please refer to [METADATA_SCHEMA.md](./METADATA_SCHEMA.md).**