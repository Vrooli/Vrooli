# RoutineData (Data Processing) Prompt

## Overview

You are an expert AI system designer tasked with creating **RoutineData** routines for Vrooli. These are single-step routines that perform specialized data processing, transformation, and manipulation operations. They serve as atomic building blocks for data workflows, ETL processes, and information management within larger systems.

## Purpose

RoutineData routines are used for:
- **Data transformation**: Convert data between formats and structures
- **Data validation**: Check data integrity and quality
- **Data enrichment**: Add metadata and computed fields
- **Data aggregation**: Combine and summarize datasets
- **Data filtering**: Extract subsets based on criteria

## Execution Context

### Tier 3 Execution
RoutineData operates at **Tier 3** of Vrooli's architecture:
- **Optimized processing**: Efficient data manipulation engines
- **Format support**: Multiple data formats (JSON, CSV, XML, etc.)
- **Template-based operations**: Dynamic data processing rules
- **Memory management**: Efficient handling of large datasets

### Integration with Multi-Step Workflows
RoutineData routines are commonly used as subroutines within RoutineMultiStep workflows for:
- **ETL steps**: Extract, transform, and load data between systems
- **Preprocessing steps**: Clean and prepare data for analysis
- **Postprocessing steps**: Format results for presentation
- **Data quality steps**: Validate and ensure data integrity

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
        "callDataData": {
          "__version": "1.0",
          "schema": {
            "operation": "transform",
            "inputFormat": "json",
            "outputFormat": "json",
            "transformationRules": {
              "mappings": [
                {
                  "source": "{{input.sourceField}}",
                  "target": "{{input.targetField}}",
                  "transform": "{{input.transformation}}"
                }
              ],
              "filters": "{{input.filterCriteria}}",
              "aggregations": "{{input.aggregationRules}}"
            },
            "validationRules": {
              "required": "{{input.requiredFields}}",
              "types": "{{input.fieldTypes}}",
              "constraints": "{{input.constraints}}"
            },
            "options": {
              "preserveMetadata": true,
              "errorHandling": "{{input.errorHandling}}",
              "batchSize": "{{input.batchSize}}"
            }
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
                "placeholder": "Enter data to process"
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
                "fieldName": "processingMetadata",
                "id": "processing_metadata_output",
                "label": "Processing Metadata",
                "type": "JSON"
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
          "name": "Descriptive Data Routine Name",
          "description": "What this data routine processes and transforms.",
          "instructions": "How to use this routine and what data to provide."
        }
      ]
    }
  ]
}
```

## Configuration Details

### callDataData Schema

#### Required Fields
- **`operation`**: Type of data operation to perform
- **`inputFormat`**: Format of input data
- **`outputFormat`**: Format of output data

#### Optional Fields
- **`transformationRules`**: Rules for data transformation
- **`validationRules`**: Rules for data validation
- **`options`**: Processing options and configuration

### Data Operations

#### Transform Operations
```json
{
  "operation": "transform",
  "transformationRules": {
    "mappings": [
      {
        "source": "{{input.sourceField}}",
        "target": "{{input.targetField}}",
        "transform": "{{input.transformationType}}"
      }
    ]
  }
}
```

#### Validation Operations
```json
{
  "operation": "validate",
  "validationRules": {
    "required": ["{{input.requiredField1}}", "{{input.requiredField2}}"],
    "types": {
      "{{input.field1}}": "string",
      "{{input.field2}}": "number"
    },
    "constraints": "{{input.validationConstraints}}"
  }
}
```

#### Aggregation Operations
```json
{
  "operation": "aggregate",
  "transformationRules": {
    "aggregations": [
      {
        "field": "{{input.aggregateField}}",
        "function": "{{input.aggregateFunction}}",
        "groupBy": "{{input.groupByField}}"
      }
    ]
  }
}
```

#### Filter Operations
```json
{
  "operation": "filter",
  "transformationRules": {
    "filters": [
      {
        "field": "{{input.filterField}}",
        "operator": "{{input.filterOperator}}",
        "value": "{{input.filterValue}}"
      }
    ]
  }
}
```

### Data Formats

#### JSON Processing
```json
{
  "inputFormat": "json",
  "outputFormat": "json"
}
```

#### CSV Processing
```json
{
  "inputFormat": "csv",
  "outputFormat": "json",
  "options": {
    "delimiter": "{{input.csvDelimiter}}",
    "headers": "{{input.hasHeaders}}",
    "encoding": "{{input.encoding}}"
  }
}
```

#### XML Processing
```json
{
  "inputFormat": "xml",
  "outputFormat": "json",
  "options": {
    "rootElement": "{{input.xmlRoot}}",
    "attributeHandling": "{{input.attributeMode}}"
  }
}
```

### Transformation Rules

#### Field Mapping
```json
{
  "mappings": [
    {
      "source": "old_field_name",
      "target": "new_field_name",
      "transform": "rename"
    },
    {
      "source": "numeric_string",
      "target": "numeric_value",
      "transform": "toNumber"
    },
    {
      "source": "timestamp_string",
      "target": "datetime",
      "transform": "parseDate"
    }
  ]
}
```

#### Data Enrichment
```json
{
  "mappings": [
    {
      "source": "computed",
      "target": "full_name",
      "transform": "concat",
      "params": ["first_name", " ", "last_name"]
    },
    {
      "source": "computed",
      "target": "category_score",
      "transform": "calculate",
      "params": {"formula": "score * weight"}
    }
  ]
}
```

### Validation Rules

#### Field Requirements
```json
{
  "validationRules": {
    "required": ["id", "name", "email"],
    "types": {
      "id": "string",
      "age": "number",
      "active": "boolean",
      "tags": "array"
    },
    "constraints": {
      "email": {"pattern": "^[^@]+@[^@]+\\.[^@]+$"},
      "age": {"min": 0, "max": 150},
      "name": {"minLength": 1, "maxLength": 100}
    }
  }
}
```

## Common Use Cases & Templates

### 1. CSV to JSON Conversion
```json
{
  "callDataData": {
    "schema": {
      "operation": "transform",
      "inputFormat": "csv",
      "outputFormat": "json",
      "transformationRules": {
        "mappings": [
          {
            "source": "{{input.csvHeaders}}",
            "target": "{{input.jsonFields}}",
            "transform": "fieldMapping"
          }
        ]
      },
      "options": {
        "delimiter": "{{input.delimiter}}",
        "hasHeaders": "{{input.hasHeaders}}",
        "preserveMetadata": true
      }
    }
  }
}
```

### 2. Data Validation and Cleaning
```json
{
  "callDataData": {
    "schema": {
      "operation": "validate",
      "inputFormat": "json",
      "outputFormat": "json",
      "validationRules": {
        "required": "{{input.requiredFields}}",
        "types": "{{input.fieldTypes}}",
        "constraints": {
          "email": {"pattern": "^[^@]+@[^@]+\\.[^@]+$"},
          "phone": {"pattern": "^\\+?[1-9]\\d{1,14}$"}
        }
      },
      "transformationRules": {
        "mappings": [
          {
            "source": "email",
            "target": "email",
            "transform": "toLowerCase"
          },
          {
            "source": "name",
            "target": "name",
            "transform": "trim"
          }
        ]
      },
      "options": {
        "errorHandling": "{{input.errorMode}}",
        "cleanInvalidRecords": "{{input.autoClean}}"
      }
    }
  }
}
```

### 3. Data Aggregation and Summarization
```json
{
  "callDataData": {
    "schema": {
      "operation": "aggregate",
      "inputFormat": "json",
      "outputFormat": "json",
      "transformationRules": {
        "aggregations": [
          {
            "field": "{{input.valueField}}",
            "function": "{{input.aggregateFunction}}",
            "groupBy": "{{input.groupByField}}"
          },
          {
            "field": "record_count",
            "function": "count",
            "groupBy": "{{input.groupByField}}"
          }
        ]
      },
      "options": {
        "includeMetadata": true,
        "sortResults": "{{input.sortOrder}}"
      }
    }
  }
}
```

### 4. Data Filtering and Subset Extraction
```json
{
  "callDataData": {
    "schema": {
      "operation": "filter",
      "inputFormat": "json",
      "outputFormat": "json",
      "transformationRules": {
        "filters": [
          {
            "field": "{{input.filterField}}",
            "operator": "{{input.filterOperator}}",
            "value": "{{input.filterValue}}"
          },
          {
            "field": "{{input.dateField}}",
            "operator": "between",
            "value": ["{{input.startDate}}", "{{input.endDate}}"]
          }
        ]
      },
      "options": {
        "limit": "{{input.maxRecords}}",
        "offset": "{{input.skipRecords}}"
      }
    }
  }
}
```

### 5. Data Format Conversion
```json
{
  "callDataData": {
    "schema": {
      "operation": "transform",
      "inputFormat": "{{input.sourceFormat}}",
      "outputFormat": "{{input.targetFormat}}",
      "transformationRules": {
        "mappings": "{{input.fieldMappings}}",
        "formatOptions": {
          "dateFormat": "{{input.dateFormat}}",
          "numberFormat": "{{input.numberFormat}}",
          "encoding": "{{input.outputEncoding}}"
        }
      },
      "options": {
        "preserveStructure": "{{input.preserveStructure}}",
        "flattenNested": "{{input.flattenNested}}"
      }
    }
  }
}
```

### 6. Data Enrichment
```json
{
  "callDataData": {
    "schema": {
      "operation": "transform",
      "inputFormat": "json",
      "outputFormat": "json",
      "transformationRules": {
        "mappings": [
          {
            "source": "computed",
            "target": "full_address",
            "transform": "concat",
            "params": ["street", ", ", "city", ", ", "state"]
          },
          {
            "source": "created_at",
            "target": "age_days",
            "transform": "dateDiff",
            "params": {"unit": "days", "from": "now"}
          },
          {
            "source": "metadata",
            "target": "enriched_at",
            "transform": "addTimestamp"
          }
        ]
      },
      "options": {
        "addMetadata": true,
        "computedFieldPrefix": "computed_"
      }
    }
  }
}
```

## Form Configuration

### Input Form Elements

#### Source Data
```json
{
  "fieldName": "sourceData",
  "type": "JSON",
  "isRequired": true,
  "label": "Source Data",
  "placeholder": "Enter data to process as JSON"
},
{
  "fieldName": "dataFormat",
  "type": "Selector",
  "label": "Input Format",
  "options": ["json", "csv", "xml"],
  "defaultValue": "json"
}
```

#### Processing Configuration
```json
{
  "fieldName": "operation",
  "type": "Selector",
  "isRequired": true,
  "label": "Operation Type",
  "options": ["transform", "validate", "aggregate", "filter"]
},
{
  "fieldName": "transformationRules",
  "type": "JSON",
  "label": "Transformation Rules",
  "placeholder": "Enter transformation configuration"
}
```

#### Processing Options
```json
{
  "fieldName": "errorHandling",
  "type": "Selector",
  "label": "Error Handling",
  "options": ["strict", "lenient", "skip"],
  "defaultValue": "lenient"
},
{
  "fieldName": "batchSize",
  "type": "IntegerInput",
  "label": "Batch Size",
  "defaultValue": 1000,
  "min": 1,
  "max": 100000
}
```

### Output Form Elements

#### Processed Results
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
  "fieldName": "processingTime",
  "type": "IntegerInput",
  "label": "Processing Time (ms)"
}
```

#### Quality Metrics
```json
{
  "fieldName": "validRecords",
  "type": "IntegerInput",
  "label": "Valid Records"
},
{
  "fieldName": "invalidRecords",
  "type": "IntegerInput",
  "label": "Invalid Records"
},
{
  "fieldName": "qualityScore",
  "type": "IntegerInput",
  "label": "Data Quality Score (0-100)"
}
```

## Execution Strategies

### Deterministic Strategy (Recommended)
**Best for**: Consistent data transformations, ETL processes, validation
```json
{
  "executionStrategy": "deterministic"
}
```

### Reasoning Strategy  
**Best for**: Complex data processing requiring decision-making
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
4. **Call Data**: Must have `callDataData` with proper schema
5. **Operation**: Must specify valid data operation
6. **Input/Output Formats**: Must specify supported data formats
7. **Forms**: Must have both `formInput` and `formOutput`
8. **IDs**: 19-digit snowflake IDs for all `id` fields
9. **Public IDs**: 10-12 character alphanumeric for `publicId` fields
10. **Tags**: Must be empty array `[]`

### Data-Specific Requirements
1. **Format Support**: Input/output formats must be supported
2. **Rule Structure**: Transformation and validation rules must be valid
3. **Template Variables**: Proper `{{variable}}` syntax
4. **Performance**: Consider memory and processing limits
5. **Error Handling**: Appropriate error handling configuration

### Quality Requirements  
1. **Data Integrity**: Ensure data consistency and accuracy
2. **Performance**: Optimize for large dataset processing
3. **Validation**: Comprehensive data quality checks
4. **Error Recovery**: Handle malformed data gracefully

## Generation Guidelines

### 1. Understand the Data Processing Need
- **Data Types**: What kind of data will be processed?
- **Transformations**: What changes need to be made?
- **Quality Requirements**: What validation is needed?
- **Performance**: What are the volume and speed requirements?

### 2. Design the Processing Logic
- **Input Handling**: How to parse and validate input data
- **Transformation Rules**: What operations to perform
- **Output Formatting**: How to structure the results
- **Error Handling**: How to handle malformed or invalid data

### 3. Configure Processing Options
- **Batch Size**: Balance memory usage and performance
- **Error Handling**: Choose appropriate error handling strategy
- **Validation**: Set appropriate data quality checks
- **Metadata**: Decide what processing metadata to include

### 4. Plan Input/Output Structure
- **Data Format**: Choose optimal input/output formats
- **Field Mapping**: Define clear field transformations
- **Result Structure**: Organize outputs for downstream use
- **Quality Metrics**: Include data quality information

### 5. Consider Performance
- **Memory Usage**: Estimate memory requirements for datasets
- **Processing Speed**: Optimize transformation algorithms
- **Scalability**: Design for growing data volumes
- **Resource Limits**: Set appropriate processing limits

## Quality Checklist

Before generating a RoutineData routine, verify:

### Structure:
- [ ] `resourceSubType` is `"RoutineData"`
- [ ] `callDataData` schema is complete and valid
- [ ] Both `formInput` and `formOutput` are defined
- [ ] All IDs follow correct format
- [ ] `executionStrategy` is appropriate for the data operation

### Data Configuration:
- [ ] Operation type is valid and supported
- [ ] Input/output formats are specified and supported
- [ ] Transformation rules are properly structured
- [ ] Validation rules are comprehensive
- [ ] Template variables use correct syntax

### Processing Logic:
- [ ] Data transformations are efficient and correct
- [ ] Validation rules ensure data quality
- [ ] Error handling covers edge cases
- [ ] Performance considerations are addressed

### Forms:
- [ ] Input form captures all necessary processing parameters
- [ ] Output form includes results and quality metrics
- [ ] Field names match template variables
- [ ] Required fields are properly marked

### Quality Assurance:
- [ ] Data integrity is maintained throughout processing
- [ ] Validation rules prevent bad data propagation
- [ ] Error handling provides useful feedback
- [ ] Performance is optimized for expected data volumes

### Metadata:
- [ ] Name clearly describes the data operation
- [ ] Description explains what data processing is performed
- [ ] Instructions guide users on proper data format
- [ ] Performance characteristics are documented

Generate RoutineData routines that provide reliable, efficient data processing capabilities while maintaining data quality and being easily composable into larger data workflows.