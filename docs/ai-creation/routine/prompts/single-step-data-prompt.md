# RoutineData (Data Processing) Prompt

## Overview

You are an expert AI system designer tasked with creating **RoutineData** routines for Vrooli. These are single-step routines that perform data processing, transformation, validation, and ETL (Extract, Transform, Load) operations. They serve as atomic building blocks for data manipulation workflows.

## Purpose

RoutineData routines are used for:
- **Data transformation**: Convert data between formats (JSON, CSV, XML, etc.)
- **Data validation**: Verify data structure, format, and content integrity
- **Data processing**: Clean, filter, aggregate, and manipulate data
- **ETL operations**: Extract, transform, and load data workflows
- **Format conversion**: Transform data structures and schemas

## Execution Context

### Tier 3 Execution
RoutineData operates at **Tier 3** of Vrooli's architecture:
- **Direct data processing**: Immediate data manipulation and transformation
- **Sandboxed execution**: Secure code execution for data operations
- **Template-based processing**: Dynamic data transformation based on inputs
- **Multiple output formats**: Support for various data output structures

### Integration with Multi-Step Workflows
RoutineData routines are commonly used as subroutines within RoutineMultiStep workflows for:
- **Data preparation steps**: Clean and format data for further processing
- **Data validation steps**: Ensure data quality and structure compliance
- **Format conversion steps**: Transform data between different formats
- **Data enrichment steps**: Add computed fields and derived values

## Important Note

**RoutineData does not have a dedicated call data configuration type.** Instead, it uses **RoutineCode** configuration (`callDataCode`) to execute sandboxed data processing operations. This provides flexibility to handle various data processing tasks using code execution.

## JSON Structure Requirements

### Complete RoutineData Structure

```json
{
  "id": "[generate-19-digit-snowflake-id]",
  "publicId": "[generate-10-12-char-alphanumeric]",
  "resourceType": "Routine",
  "isPrivate": false,
  "permissions": "{}",
  "isInternal": false,
  "tags": [],
  "versions": [
    {
      "id": "[generate-19-digit-snowflake-id]",
      "publicId": "[generate-10-12-char-alphanumeric]", 
      "versionLabel": "1.0.0",
      "versionNotes": "Initial version",
      "isComplete": true,
      "isPrivate": false,
      "versionIndex": 0,
      "isAutomatable": true,
      "resourceSubType": "RoutineData",
      "config": {
        "__version": "1.0",
        "callDataCode": {
          "__version": "1.0",
          "schema": {
            "inputTemplate": {
              "data": "{{input.sourceData}}",
              "format": "{{input.inputFormat}}",
              "options": "{{input.processingOptions}}"
            },
            "outputMappings": [
              {
                "schemaIndex": 0,
                "mapping": {
                  "processedData": "result.data",
                  "format": "result.format",
                  "recordCount": "result.count"
                }
              }
            ]
          }
        },
        "formInput": {
          "__version": "1.0",
          "schema": {
            "elements": [
              {
                "fieldName": "sourceData",
                "id": "source_data_input",
                "label": "Source Data",
                "type": "JSON",
                "isRequired": true,
                "placeholder": "Enter source data to process..."
              }
            ]
          }
        },
        "formOutput": {
          "__version": "1.0", 
          "schema": {
            "elements": [
              {
                "fieldName": "processedData",
                "id": "processed_data_output",
                "label": "Processed Data",
                "type": "JSON"
              },
              {
                "fieldName": "recordCount",
                "id": "record_count_output",
                "label": "Record Count",
                "type": "IntegerInput"
              }
            ]
          }
        },
        "executionStrategy": "deterministic"
      },
      "translations": [
        {
          "id": "[generate-19-digit-snowflake-id]",
          "language": "en",
          "name": "Descriptive Data Processing Routine Name",
          "description": "What this routine processes and how it transforms the data.",
          "instructions": "How to use this routine and what data formats to provide."
        }
      ]
    }
  ]
}
```

## Configuration Details

### Using callDataCode for Data Processing

Since RoutineData uses code execution for data processing, configure the `callDataCode` schema:

#### Input Template Structure
```json
{
  "inputTemplate": {
    "data": "{{input.sourceData}}",
    "operation": "{{input.operation}}",
    "options": "{{input.options}}"
  }
}
```

#### Output Mappings
```json
{
  "outputMappings": [
    {
      "schemaIndex": 0,
      "mapping": {
        "processedData": "result.data",
        "metadata": "result.metadata",
        "errors": "result.errors"
      }
    }
  ]
}
```

### Data Processing Operations

#### Data Validation
```json
{
  "inputTemplate": {
    "data": "{{input.dataset}}",
    "schema": "{{input.validationSchema}}",
    "strictMode": "{{input.strictValidation}}"
  }
}
```

#### Format Conversion
```json
{
  "inputTemplate": {
    "sourceData": "{{input.sourceData}}",
    "fromFormat": "{{input.sourceFormat}}",
    "toFormat": "{{input.targetFormat}}"
  }
}
```

#### Data Aggregation
```json
{
  "inputTemplate": {
    "dataset": "{{input.dataset}}",
    "groupBy": "{{input.groupByFields}}",
    "aggregations": "{{input.aggregationFunctions}}"
  }
}
```

### Form Configuration

#### Input Form Elements
Common input types for data processing:
```json
{
  "fieldName": "sourceData",
  "type": "JSON",
  "isRequired": true,
  "label": "Source Data",
  "placeholder": "Enter data to process..."
},
{
  "fieldName": "operation",
  "type": "Selector",
  "label": "Processing Operation",
  "options": ["validate", "transform", "aggregate", "filter"]
},
{
  "fieldName": "outputFormat",
  "type": "Selector",
  "label": "Output Format", 
  "options": ["JSON", "CSV", "XML", "TSV"]
}
```

#### Output Form Elements
Standard output for data processing:
```json
{
  "fieldName": "processedData",
  "type": "JSON",
  "label": "Processed Data"
},
{
  "fieldName": "recordCount",
  "type": "IntegerInput",
  "label": "Records Processed"
},
{
  "fieldName": "validationErrors",
  "type": "JSON",
  "label": "Validation Errors (if any)"
}
```

## Common Use Cases & Templates

### 1. CSV to JSON Conversion
```json
{
  "callDataCode": {
    "schema": {
      "inputTemplate": {
        "csvData": "{{input.csvContent}}",
        "delimiter": "{{input.delimiter}}",
        "headers": "{{input.hasHeaders}}"
      },
      "outputMappings": [
        {
          "schemaIndex": 0,
          "mapping": {
            "jsonData": "result.json",
            "recordCount": "result.count"
          }
        }
      ]
    }
  }
}
```

### 2. Data Validation
```json
{
  "callDataCode": {
    "schema": {
      "inputTemplate": {
        "dataset": "{{input.data}}",
        "rules": "{{input.validationRules}}",
        "mode": "{{input.validationMode}}"
      },
      "outputMappings": [
        {
          "schemaIndex": 0,
          "mapping": {
            "isValid": "result.valid",
            "errors": "result.errors",
            "validRecords": "result.validCount"
          }
        }
      ]
    }
  }
}
```

### 3. Data Aggregation
```json
{
  "callDataCode": {
    "schema": {
      "inputTemplate": {
        "dataset": "{{input.dataset}}",
        "groupByFields": "{{input.groupBy}}",
        "aggregationFunctions": "{{input.aggregations}}"
      },
      "outputMappings": [
        {
          "schemaIndex": 0,
          "mapping": {
            "aggregatedData": "result.aggregations",
            "groupCount": "result.groups"
          }
        }
      ]
    }
  }
}
```

### 4. Data Filtering
```json
{
  "callDataCode": {
    "schema": {
      "inputTemplate": {
        "dataset": "{{input.sourceData}}",
        "filterCriteria": "{{input.filters}}",
        "operator": "{{input.logicalOperator}}"
      },
      "outputMappings": [
        {
          "schemaIndex": 0,
          "mapping": {
            "filteredData": "result.data",
            "matchCount": "result.matches"
          }
        }
      ]
    }
  }
}
```

### 5. Data Transformation
```json
{
  "callDataCode": {
    "schema": {
      "inputTemplate": {
        "sourceData": "{{input.data}}",
        "transformations": "{{input.transformRules}}",
        "outputSchema": "{{input.targetSchema}}"
      },
      "outputMappings": [
        {
          "schemaIndex": 0,
          "mapping": {
            "transformedData": "result.data",
            "transformationLog": "result.log"
          }
        }
      ]
    }
  }
}
```

## Execution Strategies

### Deterministic Strategy (Recommended)
**Best for**: Data processing operations requiring consistent results
```json
{
  "executionStrategy": "deterministic"
}
```

### Reasoning Strategy
**Best for**: Complex data analysis requiring decision-making
```json
{
  "executionStrategy": "reasoning"
}
```

## Validation Rules

### Critical Requirements
1. **Resource Type**: Must be `"Routine"` 
2. **Resource Sub Type**: Must be `"RoutineData"`
3. **Config Version**: Must have `"__version": "1.0"` at root level
4. **Call Data**: Must have `callDataCode` (NOT callDataData) with proper schema
5. **Forms**: Must have both `formInput` and `formOutput` 
6. **IDs**: 19-digit snowflake IDs for all `id` fields
7. **Public IDs**: 10-12 character alphanumeric for `publicId` fields
8. **Tags**: Must be empty array `[]`

### Code Configuration Requirements
1. **Input Template**: Must be valid object structure
2. **Output Mappings**: Must include at least one mapping with schemaIndex
3. **Field Names**: Must match between form fields and template variables
4. **Data Types**: Input/output types should match expected data structures

## Generation Guidelines

### 1. Understand the Data Operation
- **Input Format**: What data format is being processed?
- **Transformation**: What processing or transformation is needed?
- **Output Format**: What format should the result be in?
- **Validation**: What validation rules should be applied?

### 2. Design the Processing Logic
- **Input Structure**: Design clear input template for code execution
- **Processing Steps**: Define the data processing operations
- **Output Mapping**: Map processing results to routine outputs
- **Error Handling**: Plan for data processing errors

### 3. Configure Data Flow
- **Input Validation**: Ensure input data meets requirements
- **Processing Options**: Allow configuration of processing parameters
- **Output Structure**: Provide structured, useful output format

### 4. Plan for Reusability
- **Generic Operations**: Design for common data processing patterns
- **Configurable Parameters**: Allow customization of processing behavior
- **Modular Design**: Focus on single data processing responsibility

## Quality Checklist

Before generating a RoutineData routine, verify:

### Structure:
- [ ] `resourceSubType` is `"RoutineData"`
- [ ] Uses `callDataCode` configuration (NOT callDataData)
- [ ] Both `formInput` and `formOutput` are defined
- [ ] All IDs follow correct format
- [ ] `executionStrategy` is appropriate for data processing

### Code Configuration:
- [ ] `inputTemplate` properly maps form inputs
- [ ] `outputMappings` includes proper schemaIndex and mapping
- [ ] Template variables use correct syntax
- [ ] All input fields are referenced in template

### Forms:
- [ ] Input form captures all necessary data and options
- [ ] Output form includes processed data and metadata
- [ ] Field names match template variables
- [ ] Data types are appropriate for processing operations

### Metadata:
- [ ] Name clearly describes the data processing operation
- [ ] Description explains what data is processed and how
- [ ] Instructions guide users on input formats and options

Generate RoutineData routines that provide reliable, efficient data processing capabilities while being easily composable into larger data workflows.