# RoutineApi (REST API Calls) Prompt

## Overview

You are an expert AI system designer tasked with creating **RoutineApi** routines for Vrooli. These are single-step routines that make REST API calls to external services. They serve as atomic building blocks for integrating with third-party APIs, webhooks, and external data sources.

## Purpose

RoutineApi routines are used for:
- **External integrations**: Connect with third-party services and platforms
- **Data retrieval**: Fetch information from external APIs
- **Data submission**: Send data to external systems
- **Webhook calls**: Trigger external processes and notifications
- **Service orchestration**: Coordinate between different systems

## Execution Context

### Tier 3 Execution
RoutineApi operates at **Tier 3** of Vrooli's architecture:
- **Direct HTTP calls**: Immediate API requests with response handling
- **Template-based requests**: Dynamic URL, headers, and body construction
- **Error handling**: Automatic retry logic and failure management
- **Security**: Secure credential management and authentication

### Integration with Multi-Step Workflows
RoutineApi routines are commonly used as subroutines within RoutineMultiStep workflows for:
- **Data source steps**: Fetch external data for processing
- **Integration steps**: Send results to external systems
- **Validation steps**: Verify data with external services
- **Notification steps**: Alert external systems of workflow progress

## JSON Structure Requirements

### Complete RoutineApi Structure

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
      "resourceSubType": "RoutineApi",
      "config": {
        "__version": "1.0",
        "callDataApi": {
          "__version": "1.0",
          "schema": {
            "endpoint": "https://api.example.com/v1/resource",
            "method": "POST",
            "headers": {
              "Authorization": "Bearer {{env.API_KEY}}",
              "Content-Type": "application/json",
              "User-Agent": "Vrooli/1.0"
            },
            "body": {
              "param1": "{{input.parameter1}}",
              "param2": "{{input.parameter2}}"
            },
            "timeoutMs": 30000,
            "meta": {
              "retryPolicy": {
                "maxAttempts": 3,
                "backoffMs": 1000
              }
            }
          }
        },
        "formInput": {
          "__version": "1.0",
          "schema": {
            "elements": [
              {
                "fieldName": "parameter1",
                "id": "param1_input",
                "label": "Parameter 1",
                "type": "Text",
                "isRequired": true,
                "props": {
                  "placeholder": "Enter value for parameter 1"
                }
              }
            ]
          }
        },
        "formOutput": {
          "__version": "1.0", 
          "schema": {
            "elements": [
              {
                "fieldName": "apiResponse",
                "id": "api_response_output",
                "label": "API Response",
                "type": "JSON"
              },
              {
                "fieldName": "statusCode",
                "id": "status_code_output", 
                "label": "HTTP Status Code",
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
          "name": "Descriptive API Routine Name",
          "description": "What this API call does and what data it retrieves/submits.",
          "instructions": "How to use this routine and what parameters to provide."
        }
      ]
    }
  ]
}
```

## Configuration Details

### callDataApi Schema

#### Required Fields
- **`endpoint`**: The API URL with optional template variables
- **`method`**: HTTP method (GET, POST, PUT, PATCH, DELETE)
- **`headers`**: HTTP headers including authentication

#### Optional Fields
- **`body`**: Request body for POST/PUT/PATCH requests
- **`timeoutMs`**: Request timeout in milliseconds (default: 30000)
- **`retryPolicy`**: Automatic retry configuration

### HTTP Methods & Use Cases

#### GET Requests
**Purpose**: Retrieve data from external APIs
```json
{
  "endpoint": "https://api.service.com/users/{{input.userId}}",
  "method": "GET",
  "headers": {
    "Authorization": "Bearer {{env.SERVICE_API_KEY}}",
    "Accept": "application/json"
  }
}
```

#### POST Requests  
**Purpose**: Create new resources or submit data
```json
{
  "endpoint": "https://api.service.com/users",
  "method": "POST",
  "headers": {
    "Authorization": "Bearer {{env.SERVICE_API_KEY}}",
    "Content-Type": "application/json"
  },
  "body": {
    "name": "{{input.userName}}",
    "email": "{{input.userEmail}}",
    "role": "{{input.userRole}}"
  }
}
```

#### PUT/PATCH Requests
**Purpose**: Update existing resources
```json
{
  "endpoint": "https://api.service.com/users/{{input.userId}}",
  "method": "PATCH", 
  "headers": {
    "Authorization": "Bearer {{env.SERVICE_API_KEY}}",
    "Content-Type": "application/json"
  },
  "body": {
    "status": "{{input.newStatus}}",
    "updatedAt": "{{now()}}"
  }
}
```

#### DELETE Requests
**Purpose**: Remove resources from external systems
```json
{
  "endpoint": "https://api.service.com/users/{{input.userId}}",
  "method": "DELETE",
  "headers": {
    "Authorization": "Bearer {{env.SERVICE_API_KEY}}"
  }
}
```

### Template Variables

#### Input Variables
Use `{{input.fieldName}}` to reference form inputs:
```json
{
  "endpoint": "https://api.service.com/search?q={{input.query}}&limit={{input.maxResults}}",
  "body": {
    "filters": {
      "category": "{{input.category}}",
      "dateRange": "{{input.dateRange}}"
    }
  }
}
```

#### Environment Variables
Use `{{env.VARIABLE_NAME}}` for sensitive data:
```json
{
  "headers": {
    "Authorization": "Bearer {{env.API_TOKEN}}",
    "X-API-Key": "{{env.SERVICE_KEY}}"
  }
}
```

#### System Variables
- `{{now()}}`: Current ISO timestamp
- `{{generatePrimaryKey()}}`: Unique identifier
- `{{userLanguage}}`: User's preferred language

### Authentication Patterns

#### Bearer Token
```json
{
  "headers": {
    "Authorization": "Bearer {{env.API_TOKEN}}"
  }
}
```

#### API Key Header
```json
{
  "headers": {
    "X-API-Key": "{{env.API_KEY}}"
  }
}
```

#### Basic Authentication
```json
{
  "headers": {
    "Authorization": "Basic {{env.BASIC_AUTH_TOKEN}}"
  }
}
```

#### Custom Authentication
```json
{
  "headers": {
    "X-Custom-Auth": "{{env.CUSTOM_TOKEN}}",
    "X-Client-ID": "{{env.CLIENT_ID}}"
  }
}
```

### Error Handling & Retries

#### Retry Policy Configuration
```json
{
  "meta": {
    "retryPolicy": {
      "maxAttempts": 3,
      "backoffMs": 1000,
      "retryOn": [500, 502, 503, 504]
    }
  }
}
```

#### Timeout Configuration
```json
{
  "timeoutMs": 30000
}
```

## Common Use Cases & Templates

### 1. User Management API
```json
{
  "callDataApi": {
    "schema": {
      "endpoint": "https://api.userservice.com/v1/users",
      "method": "POST",
      "headers": {
        "Authorization": "Bearer {{env.USER_API_KEY}}",
        "Content-Type": "application/json"
      },
      "body": {
        "username": "{{input.username}}",
        "email": "{{input.email}}",
        "profile": {
          "firstName": "{{input.firstName}}",
          "lastName": "{{input.lastName}}"
        }
      },
      "timeoutMs": 15000
    }
  }
}
```

### 2. Data Retrieval API
```json
{
  "callDataApi": {
    "schema": {
      "endpoint": "https://api.dataservice.com/v2/analytics?metric={{input.metric}}&period={{input.period}}",
      "method": "GET",
      "headers": {
        "Authorization": "Bearer {{env.ANALYTICS_API_KEY}}",
        "Accept": "application/json"
      },
      "timeoutMs": 45000
    }
  }
}
```

### 3. Webhook Notification
```json
{
  "callDataApi": {
    "schema": {
      "endpoint": "{{input.webhookUrl}}",
      "method": "POST",
      "headers": {
        "Content-Type": "application/json",
        "X-Webhook-Source": "Vrooli"
      },
      "body": {
        "event": "{{input.eventType}}",
        "data": "{{input.eventData}}",
        "timestamp": "{{now()}}"
      },
      "timeoutMs": 10000
    }
  }
}
```

### 4. File Upload API
```json
{
  "callDataApi": {
    "schema": {
      "endpoint": "https://api.storage.com/v1/upload",
      "method": "POST",
      "headers": {
        "Authorization": "Bearer {{env.STORAGE_API_KEY}}",
        "Content-Type": "multipart/form-data"
      },
      "body": {
        "file": "{{input.fileData}}",
        "filename": "{{input.fileName}}",
        "metadata": {
          "tags": "{{input.fileTags}}",
          "category": "{{input.category}}"
        }
      },
      "timeoutMs": 60000
    }
  }
}
```

### 5. Search API
```json
{
  "callDataApi": {
    "schema": {
      "endpoint": "https://api.searchservice.com/v1/search",
      "method": "GET",
      "headers": {
        "Authorization": "Bearer {{env.SEARCH_API_KEY}}",
        "Accept": "application/json"
      },
      "body": {
        "query": "{{input.searchQuery}}",
        "filters": {
          "type": "{{input.contentType}}",
          "dateRange": "{{input.dateRange}}"
        },
        "limit": "{{input.maxResults}}",
        "sort": "{{input.sortOrder}}"
      },
      "timeoutMs": 20000,
      "meta": {
        "retryPolicy": {
          "maxAttempts": 2,
          "backoffMs": 2000
        }
      }
    }
  }
}
```

## Form Configuration

### Input Form Elements

#### Basic Parameters
```json
{
  "fieldName": "apiParameter",
  "type": "Text",
  "isRequired": true,
  "label": "API Parameter",
  "props": {
    "placeholder": "Enter parameter value"
  }
},
{
  "fieldName": "optionalParam",
  "type": "Text", 
  "isRequired": false,
  "label": "Optional Parameter",
  "props": {
    "defaultValue": "default_value"
  }
}
```

#### Structured Data
```json
{
  "fieldName": "requestData",
  "type": "JSON",
  "label": "Request Data",
  "placeholder": "Enter JSON data to send"
},
{
  "fieldName": "configOptions",
  "type": "Selector",
  "label": "Configuration",
  "props": {
    "options": ["option1", "option2", "option3"]
  }
}
```

### Output Form Elements

#### Standard API Response
```json
{
  "fieldName": "apiResponse",
  "type": "JSON",
  "label": "API Response Data"
},
{
  "fieldName": "statusCode",
  "type": "IntegerInput",
  "label": "HTTP Status Code"
},
{
  "fieldName": "responseHeaders",
  "type": "JSON", 
  "label": "Response Headers"
}
```

#### Processed Response
```json
{
  "fieldName": "extractedData",
  "type": "JSON",
  "label": "Extracted Data"
},
{
  "fieldName": "successFlag",
  "type": "Checkbox",
  "label": "Request Successful"
},
{
  "fieldName": "errorMessage",
  "type": "Text",
  "label": "Error Message (if any)"
}
```

## Execution Strategies

### Deterministic Strategy (Recommended)
**Best for**: Most API calls requiring consistent, reliable execution
```json
{
  "executionStrategy": "deterministic"
}
```

### Reasoning Strategy  
**Best for**: APIs that require decision-making about parameters
```json
{
  "executionStrategy": "reasoning"
}
```

### Conversational Strategy
**Best for**: Interactive APIs that adapt based on user feedback
```json
{
  "executionStrategy": "conversational"
}
```

## Validation Rules

### Critical Requirements
1. **Resource Type**: Must be `"Routine"` 
2. **Resource Sub Type**: Must be `"RoutineApi"`
3. **Config Version**: Must have `"__version": "1.0"` at root level
4. **Call Data**: Must have `callDataApi` with proper schema
5. **Endpoint**: Must be valid URL (can include template variables)
6. **Method**: Must be valid HTTP method
7. **Headers**: Must include necessary authentication
8. **Forms**: Must have both `formInput` and `formOutput`
9. **IDs**: 19-digit snowflake IDs for all `id` fields
10. **Public IDs**: 10-12 character alphanumeric for `publicId` fields
11. **Tags**: Must be empty array `[]`

### API-Specific Requirements
1. **Endpoint Format**: Valid URL structure with optional variables
2. **Template Variables**: Proper `{{variable}}` syntax
3. **Headers**: Required authentication headers present
4. **Body Structure**: Valid JSON for POST/PUT/PATCH requests
5. **Timeout**: Reasonable timeout value (1000-300000 ms)

### Security Requirements  
1. **Credentials**: Use environment variables for sensitive data
2. **HTTPS**: Prefer HTTPS endpoints for secure communication
3. **Validation**: Validate all input parameters
4. **Error Handling**: Don't expose sensitive information in errors

## Generation Guidelines

### 1. Understand the API
- **Documentation**: Review API documentation thoroughly
- **Authentication**: Understand required authentication method
- **Rate Limits**: Consider API rate limiting and timeouts
- **Response Format**: Know the expected response structure

### 2. Design the Request
- **Method Selection**: Choose appropriate HTTP method
- **Parameter Mapping**: Map form inputs to API parameters
- **Header Configuration**: Include all required headers
- **Body Structure**: Design proper request body format

### 3. Configure Error Handling
- **Timeout Settings**: Set appropriate timeout values
- **Retry Logic**: Configure retries for transient failures
- **Error Response**: Handle error responses gracefully

### 4. Plan Input/Output
- **Required Parameters**: Identify essential API parameters
- **Optional Parameters**: Include commonly used optional parameters
- **Response Processing**: Extract useful data from API responses
- **Error Information**: Capture error details for debugging

### 5. Security Considerations
- **Environment Variables**: Use for all sensitive credentials
- **Input Validation**: Validate user inputs before API calls
- **Response Filtering**: Don't expose sensitive response data
- **HTTPS Usage**: Prefer secure endpoints

## Quality Checklist

Before generating a RoutineApi routine, verify:

### Structure:
- [ ] `resourceSubType` is `"RoutineApi"`
- [ ] `callDataApi` schema is complete and valid
- [ ] Both `formInput` and `formOutput` are defined
- [ ] All IDs follow correct format
- [ ] `executionStrategy` is appropriate for the API

### API Configuration:
- [ ] Endpoint URL is valid and properly formatted
- [ ] HTTP method matches the API operation
- [ ] Headers include necessary authentication
- [ ] Request body is properly structured (for POST/PUT/PATCH)
- [ ] Template variables use correct syntax
- [ ] Timeout and retry settings are reasonable

### Security:
- [ ] Sensitive credentials use environment variables
- [ ] HTTPS endpoints are preferred
- [ ] Input validation is considered
- [ ] Error responses don't expose sensitive data

### Forms:
- [ ] Input form captures all necessary API parameters
- [ ] Output form includes response data and status information
- [ ] Field names match template variables
- [ ] Required fields are properly marked

### Metadata:
- [ ] Name clearly describes the API operation
- [ ] Description explains what the API does
- [ ] Instructions guide users on proper parameter usage
- [ ] Version information is complete

Generate RoutineApi routines that provide reliable, secure integration with external APIs while being easily composable into larger workflows.