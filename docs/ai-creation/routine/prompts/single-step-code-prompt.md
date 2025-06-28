# RoutineCode (Sandboxed Code Execution) Prompt

## Overview

You are an expert AI system designer tasked with creating **RoutineCode** routines for Vrooli. These are single-step routines that execute sandboxed code for data processing, calculations, transformations, and algorithmic operations. They serve as atomic building blocks for computational tasks within larger workflows.

## Purpose

RoutineCode routines are used for:
- **Data transformation**: Convert, clean, and restructure data
- **Mathematical calculations**: Complex computations and statistical analysis
- **Algorithmic operations**: Sorting, filtering, searching, optimization
- **Format conversions**: Transform data between different formats
- **Validation logic**: Custom validation rules and data integrity checks

## Execution Context

### Tier 3 Execution
RoutineCode operates at **Tier 3** of Vrooli's architecture:
- **Sandboxed execution**: Secure, isolated code execution environment
- **Multi-language support**: JavaScript, Python, and other supported languages
- **Template-based inputs**: Dynamic data injection via template variables
- **Resource limits**: CPU, memory, and time constraints for security

### Integration with Multi-Step Workflows
RoutineCode routines are commonly used as subroutines within RoutineMultiStep workflows for:
- **Processing steps**: Transform data between workflow stages
- **Calculation steps**: Perform complex computations on workflow data
- **Validation steps**: Check data integrity and business rules
- **Aggregation steps**: Combine and summarize data from multiple sources

## JSON Structure Requirements

### Complete RoutineCode Structure

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
      "resourceSubType": "RoutineCode",
      "config": {
        "__version": "1.0",
        "callDataCode": {
          "__version": "1.0",
          "schema": {
            "inputTemplate": {
              "data": "{{input.sourceData}}",
              "config": "{{input.processingConfig}}",
              "options": "{{input.options}}"
            },
            "outputMapping": {
              "result": "processedData",
              "metadata": "processingMetadata",
              "status": "success"
            },
            "language": "javascript",
            "timeoutMs": 30000,
            "memoryLimitMb": 128
          }
        },
        "formInput": {
          "__version": "1.0",
          "schema": {
            "containers": [],
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
            "containers": [],
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
          "name": "Descriptive Code Routine Name",
          "description": "What this code routine processes and transforms.",
          "instructions": "How to use this routine and what data to provide."
        }
      ]
    }
  ]
}
```

## Configuration Details

### callDataCode Schema

#### Required Fields
- **`inputTemplate`**: Object mapping template variables to code inputs
- **`outputMapping`**: Object defining how code outputs map to routine outputs
- **`language`**: Programming language for code execution

#### Optional Fields
- **`timeoutMs`**: Execution timeout in milliseconds (default: 30000)
- **`memoryLimitMb`**: Memory limit in megabytes (default: 128)
- **`code`**: Inline code (alternative to external code files)

### Input Template Configuration

#### Basic Input Mapping
```json
{
  "inputTemplate": {
    "rawData": "{{input.sourceData}}",
    "processingMode": "{{input.mode}}",
    "threshold": "{{input.threshold}}"
  }
}
```

#### Complex Input Structures
```json
{
  "inputTemplate": {
    "dataset": {
      "values": "{{input.dataValues}}",
      "metadata": {
        "source": "{{input.dataSource}}",
        "timestamp": "{{now()}}",
        "version": "{{input.dataVersion}}"
      }
    },
    "config": {
      "algorithm": "{{input.algorithm}}",
      "parameters": "{{input.algorithmParams}}",
      "options": {
        "validateOutput": true,
        "includeMetrics": "{{input.includeMetrics}}"
      }
    }
  }
}
```

### Output Mapping Configuration

#### Basic Output Mapping
```json
{
  "outputMapping": {
    "result": "processedResult",
    "count": "itemCount",
    "status": "processingStatus"
  }
}
```

#### Complex Output Mapping
```json
{
  "outputMapping": {
    "primaryResult": "mainOutput",
    "metrics": "performanceMetrics",
    "alternativeResult": "alternativeOutput",
    "warnings": "processingWarnings"
  }
}
```

### Programming Language Support

#### JavaScript (Recommended)
```json
{
  "language": "javascript",
  "inputTemplate": {
    "data": "{{input.jsonData}}",
    "options": "{{input.processingOptions}}"
  }
}
```

#### Python
```json
{
  "language": "python",
  "inputTemplate": {
    "data": "{{input.dataFrame}}",
    "config": "{{input.analysisConfig}}"
  }
}
```

### Resource Limits

#### Performance Configuration
```json
{
  "timeoutMs": 60000,
  "memoryLimitMb": 256,
  "cpuLimitPercent": 80
}
```

#### Security Configuration
```json
{
  "timeoutMs": 15000,
  "memoryLimitMb": 64,
  "networkAccess": false,
  "fileSystemAccess": false
}
```

## Common Use Cases & Templates

### 1. Data Transformation
```json
{
  "callDataCode": {
    "schema": {
      "inputTemplate": {
        "sourceData": "{{input.rawData}}",
        "transformationRules": "{{input.rules}}",
        "outputFormat": "{{input.targetFormat}}"
      },
      "outputMapping": {
        "transformedData": "result",
        "transformationLog": "metadata.log",
        "errorCount": "metadata.errors"
      },
      "language": "javascript",
      "timeoutMs": 45000
    }
  }
}
```

### 2. Statistical Analysis
```json
{
  "callDataCode": {
    "schema": {
      "inputTemplate": {
        "dataset": "{{input.numericalData}}",
        "analysisType": "{{input.statisticalMethod}}",
        "confidenceLevel": "{{input.confidence}}"
      },
      "outputMapping": {
        "statistics": "analysisResults",
        "summary": "summaryStats",
        "charts": "visualizations"
      },
      "language": "python",
      "timeoutMs": 90000,
      "memoryLimitMb": 256
    }
  }
}
```

### 3. Data Validation
```json
{
  "callDataCode": {
    "schema": {
      "inputTemplate": {
        "records": "{{input.dataRecords}}",
        "validationRules": "{{input.rules}}",
        "strictMode": "{{input.strict}}"
      },
      "outputMapping": {
        "validRecords": "valid",
        "invalidRecords": "invalid",
        "validationReport": "report"
      },
      "language": "javascript",
      "timeoutMs": 30000
    }
  }
}
```

### 4. Algorithm Implementation
```json
{
  "callDataCode": {
    "schema": {
      "inputTemplate": {
        "inputArray": "{{input.dataArray}}",
        "algorithmConfig": {
          "method": "{{input.sortMethod}}",
          "ascending": "{{input.ascending}}",
          "customComparator": "{{input.comparator}}"
        }
      },
      "outputMapping": {
        "sortedArray": "result",
        "comparisonCount": "metrics.comparisons",
        "executionTime": "metrics.timeMs"
      },
      "language": "javascript",
      "timeoutMs": 20000
    }
  }
}
```

### 5. Format Conversion
```json
{
  "callDataCode": {
    "schema": {
      "inputTemplate": {
        "sourceData": "{{input.inputData}}",
        "sourceFormat": "{{input.fromFormat}}",
        "targetFormat": "{{input.toFormat}}",
        "conversionOptions": "{{input.options}}"
      },
      "outputMapping": {
        "convertedData": "output",
        "conversionMetadata": "metadata",
        "warningsLog": "warnings"
      },
      "language": "javascript",
      "timeoutMs": 25000
    }
  }
}
```

## Code Examples

### JavaScript Data Processing
```javascript
// Input: { data: [1,2,3,4,5], operation: "square" }
const processData = (input) => {
  const { data, operation } = input;
  let result;
  
  switch(operation) {
    case "square":
      result = data.map(x => x * x);
      break;
    case "sum":
      result = data.reduce((a, b) => a + b, 0);
      break;
    default:
      result = data;
  }
  
  return {
    result: result,
    count: data.length,
    operation: operation,
    timestamp: new Date().toISOString()
  };
};

return processData(input);
```

### Python Statistical Analysis
```python
import json
import statistics

def analyze_data(input_data):
    data = input_data['dataset']
    analysis_type = input_data.get('analysisType', 'basic')
    
    results = {
        'mean': statistics.mean(data),
        'median': statistics.median(data),
        'mode': statistics.mode(data) if len(set(data)) < len(data) else None,
        'stdev': statistics.stdev(data) if len(data) > 1 else 0
    }
    
    if analysis_type == 'advanced':
        results.update({
            'variance': statistics.variance(data) if len(data) > 1 else 0,
            'min': min(data),
            'max': max(data),
            'range': max(data) - min(data)
        })
    
    return {
        'statistics': results,
        'sample_size': len(data),
        'analysis_type': analysis_type
    }

return analyze_data(input)
```

## Form Configuration

### Input Form Elements

#### Data Inputs
```json
{
  "fieldName": "sourceData",
  "type": "JSON",
  "isRequired": true,
  "label": "Source Data",
  "placeholder": "Enter JSON data to process"
},
{
  "fieldName": "processingConfig",
  "type": "JSON",
  "label": "Processing Configuration",
  "placeholder": "Enter processing parameters"
}
```

#### Processing Options
```json
{
  "fieldName": "algorithm",
  "type": "Selector",
  "label": "Algorithm",
  "options": ["quicksort", "mergesort", "heapsort"],
  "defaultValue": "quicksort"
},
{
  "fieldName": "batchSize",
  "type": "IntegerInput",
  "label": "Batch Size",
  "defaultValue": 1000,
  "min": 1,
  "max": 10000
}
```

### Output Form Elements

#### Results
```json
{
  "fieldName": "processedData",
  "type": "JSON",
  "label": "Processed Data"
},
{
  "fieldName": "processingMetrics",
  "type": "JSON",
  "label": "Processing Metrics"
},
{
  "fieldName": "executionTime",
  "type": "IntegerInput",
  "label": "Execution Time (ms)"
}
```

#### Status Information
```json
{
  "fieldName": "success",
  "type": "Checkbox",
  "label": "Processing Successful"
},
{
  "fieldName": "errorLog",
  "type": "Text",
  "label": "Error Messages (if any)"
},
{
  "fieldName": "warningCount",
  "type": "IntegerInput",
  "label": "Warning Count"
}
```

## Execution Strategies

### Deterministic Strategy (Recommended)
**Best for**: Mathematical calculations, data transformations, algorithmic operations
```json
{
  "executionStrategy": "deterministic"
}
```

### Reasoning Strategy  
**Best for**: Code that requires decision-making about processing approach
```json
{
  "executionStrategy": "reasoning"
}
```

## Validation Rules

### Critical Requirements
1. **Resource Type**: Must be `"Routine"` 
2. **Resource Sub Type**: Must be `"RoutineCode"`
3. **Config Version**: Must have `"__version": "1.0"` at root level
4. **Call Data**: Must have `callDataCode` with proper schema
5. **Input Template**: Must define mapping from form inputs to code variables
6. **Output Mappings**: Must define how code outputs map to routine outputs
7. **Language**: Must specify supported programming language
8. **Forms**: Must have both `formInput` and `formOutput`
9. **IDs**: 19-digit snowflake IDs for all `id` fields
10. **Public IDs**: 10-12 character alphanumeric for `publicId` fields
11. **Tags**: Must be empty array `[]`

### Code-Specific Requirements
1. **Input Template**: Valid JSON structure with template variables
2. **Output Mappings**: Proper mapping configuration
3. **Resource Limits**: Reasonable timeout and memory limits
4. **Language Support**: Use supported programming languages
5. **Security**: Avoid unsafe operations and external dependencies

### Performance Requirements  
1. **Timeout**: Set appropriate execution timeout (5000-300000 ms)
2. **Memory**: Set reasonable memory limits (32-512 MB)
3. **Complexity**: Avoid computationally expensive algorithms
4. **Efficiency**: Optimize for execution speed and resource usage

## Generation Guidelines

### 1. Understand the Processing Task
- **Data Types**: What kind of data will be processed?
- **Algorithms**: What computational operations are needed?
- **Performance**: What are the speed and memory requirements?
- **Output Format**: What should the processed data look like?

### 2. Design the Code Logic
- **Input Handling**: Validate and prepare input data
- **Core Processing**: Implement the main algorithm or transformation
- **Error Handling**: Handle edge cases and invalid inputs
- **Output Formatting**: Structure results for downstream use

### 3. Configure Resource Limits
- **Execution Time**: Set timeout based on expected processing time
- **Memory Usage**: Estimate memory requirements
- **Security Constraints**: Apply appropriate sandboxing limits

### 4. Plan Input/Output Mapping
- **Input Parameters**: Map form inputs to code variables
- **Output Extraction**: Extract meaningful results from code execution
- **Metadata Capture**: Include processing metrics and status information

### 5. Choose Programming Language
- **JavaScript**: For general data processing and web-compatible operations
- **Python**: For statistical analysis, data science, and complex algorithms
- **Language Features**: Consider what language features are needed

## Quality Checklist

Before generating a RoutineCode routine, verify:

### Structure:
- [ ] `resourceSubType` is `"RoutineCode"`
- [ ] `callDataCode` schema is complete and valid
- [ ] Both `formInput` and `formOutput` are defined
- [ ] All IDs follow correct format
- [ ] `executionStrategy` is appropriate for the code task

### Code Configuration:
- [ ] Input template properly maps form inputs to code variables
- [ ] Output mappings extract all necessary results
- [ ] Programming language is specified and supported
- [ ] Resource limits are reasonable and secure
- [ ] Template variables use correct syntax

### Processing Logic:
- [ ] Code logic is efficient and optimized
- [ ] Error handling covers edge cases
- [ ] Input validation prevents security issues
- [ ] Output format is consistent and useful

### Forms:
- [ ] Input form captures all necessary processing parameters
- [ ] Output form includes results and metadata
- [ ] Field names match template variables and mappings
- [ ] Required fields are properly marked

### Security:
- [ ] Resource limits prevent abuse
- [ ] No unsafe operations or external dependencies
- [ ] Input validation prevents injection attacks
- [ ] Sandboxing constraints are appropriate

### Metadata:
- [ ] Name clearly describes the processing operation
- [ ] Description explains what the code does
- [ ] Instructions guide users on proper data format
- [ ] Version information is complete

Generate RoutineCode routines that provide reliable, efficient data processing capabilities while being easily composable into larger workflows.