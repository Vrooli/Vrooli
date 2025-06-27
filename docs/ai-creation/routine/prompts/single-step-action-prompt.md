# RoutineInternalAction (MCP Tool) Prompt

## Overview

You are an expert AI system designer tasked with creating **RoutineInternalAction** routines for Vrooli. These are single-step routines that execute Vrooli platform actions using MCP (Model Context Protocol) tools. They serve as atomic building blocks for platform integrations, data management, and system interactions within workflows.

## Purpose

RoutineInternalAction routines are used for:
- **Platform operations**: Create, read, update, delete Vrooli resources
- **Data management**: Manage users, projects, routines, and other platform data
- **System interactions**: Send notifications, manage permissions, trigger events
- **Resource orchestration**: Coordinate platform resources and workflows
- **Administrative tasks**: User management, system configuration, monitoring

## Execution Context

### Tier 3 Execution
RoutineInternalAction operates at **Tier 3** of Vrooli's architecture:
- **MCP tool integration**: Direct calls to Vrooli's internal MCP tools
- **Platform security**: Secure access to platform resources with proper permissions
- **Template-based calls**: Dynamic tool parameter construction
- **Resource management**: Efficient handling of platform resources

### Integration with Multi-Step Workflows
RoutineInternalAction routines are commonly used as subroutines within RoutineMultiStep workflows for:
- **Resource creation steps**: Create new platform resources during workflows
- **Data retrieval steps**: Fetch platform data for processing
- **Notification steps**: Send alerts and updates to users
- **Cleanup steps**: Manage resources after workflow completion

## JSON Structure Requirements

### Complete RoutineInternalAction Structure

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
      "resourceSubType": "RoutineInternalAction",
      "config": {
        "__version": "1.0",
        "callDataAction": {
          "__version": "1.0",
          "schema": {
            "toolName": "ResourceManage",
            "inputTemplate": {
              "op": "create",
              "resource_type": "{{input.resourceType}}",
              "data": "{{input.resourceData}}",
              "options": {
                "validate": true,
                "notify": "{{input.shouldNotify}}"
              }
            },
            "outputMapping": {
              "createdResource": "result",
              "resourceId": "result.id",
              "success": "success",
              "errorMessage": "error"
            }
          }
        },
        "formInput": {
          "__version": "1.0",
          "schema": {
            "elements": [
              {
                "fieldName": "resourceType",
                "id": "resource_type_input",
                "label": "Resource Type",
                "type": "Selector",
                "isRequired": true,
                "options": ["Project", "Routine", "User", "Team"]
              }
            ]
          }
        },
        "formOutput": {
          "__version": "1.0", 
          "schema": {
            "elements": [
              {
                "fieldName": "createdResource",
                "id": "created_resource_output",
                "label": "Created Resource",
                "type": "JSON"
              },
              {
                "fieldName": "resourceId",
                "id": "resource_id_output",
                "label": "Resource ID",
                "type": "TextInput"
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
          "name": "Descriptive Action Routine Name",
          "description": "What this platform action does and what resources it manages.",
          "instructions": "How to use this routine and what parameters to provide."
        }
      ]
    }
  ]
}
```

## Configuration Details

### callDataAction Schema

#### Required Fields
- **`toolName`**: Name of the MCP tool to execute
- **`inputTemplate`**: Template for tool input parameters
- **`outputMapping`**: How to map tool outputs to routine outputs

#### Available MCP Tools

##### ResourceManage
**Purpose**: Create, read, update, delete platform resources
```json
{
  "toolName": "ResourceManage",
  "inputTemplate": {
    "op": "{{input.operation}}",
    "resource_type": "{{input.resourceType}}",
    "resource_id": "{{input.resourceId}}",
    "data": "{{input.resourceData}}",
    "filters": "{{input.searchFilters}}"
  }
}
```

Operations: "create", "read", "update", "delete", "search", "list"

##### NotificationSend
**Purpose**: Send notifications to users
```json
{
  "toolName": "NotificationSend",
  "inputTemplate": {
    "type": "{{input.notificationType}}",
    "recipients": "{{input.userIds}}",
    "subject": "{{input.subject}}",
    "message": "{{input.message}}",
    "data": "{{input.notificationData}}"
  }
}
```

##### UserManage
**Purpose**: Manage user accounts and permissions
```json
{
  "toolName": "UserManage",
  "inputTemplate": {
    "action": "{{input.userAction}}",
    "user_id": "{{input.userId}}",
    "updates": "{{input.userUpdates}}",
    "permissions": "{{input.newPermissions}}"
  }
}
```

##### TeamManage  
**Purpose**: Manage teams and memberships
```json
{
  "toolName": "TeamManage",
  "inputTemplate": {
    "operation": "{{input.teamOperation}}",
    "team_id": "{{input.teamId}}",
    "member_id": "{{input.memberId}}",
    "role": "{{input.memberRole}}",
    "team_data": "{{input.teamData}}"
  }
}
```

##### ProjectManage
**Purpose**: Manage projects and project resources
```json
{
  "toolName": "ProjectManage",
  "inputTemplate": {
    "action": "{{input.projectAction}}",
    "project_id": "{{input.projectId}}",
    "project_data": "{{input.projectData}}",
    "member_permissions": "{{input.permissions}}"
  }
}
```

##### RoutineManage
**Purpose**: Manage routines and their execution
```json
{
  "toolName": "RoutineManage",
  "inputTemplate": {
    "operation": "{{input.routineOperation}}",
    "routine_id": "{{input.routineId}}",
    "run_data": "{{input.runData}}",
    "execution_config": "{{input.executionConfig}}"
  }
}
```

##### SystemMonitor
**Purpose**: Monitor system health and metrics
```json
{
  "toolName": "SystemMonitor",
  "inputTemplate": {
    "metric": "{{input.metricType}}",
    "time_range": "{{input.timeRange}}",
    "filters": "{{input.metricFilters}}",
    "aggregation": "{{input.aggregationType}}"
  }
}
```

### Input Template Configuration

#### Basic Operations
```json
{
  "inputTemplate": {
    "op": "{{input.operation}}",
    "resource_type": "{{input.type}}",
    "data": "{{input.data}}"
  }
}
```

#### Complex Operations
```json
{
  "inputTemplate": {
    "action": "{{input.action}}",
    "target": {
      "type": "{{input.targetType}}",
      "id": "{{input.targetId}}",
      "filters": "{{input.filters}}"
    },
    "payload": {
      "data": "{{input.payloadData}}",
      "options": "{{input.options}}",
      "metadata": {
        "source": "routine",
        "timestamp": "{{now()}}",
        "user_id": "{{userId}}"
      }
    }
  }
}
```

### Output Mapping Configuration

#### Standard Output Mapping
```json
{
  "outputMapping": {
    "result": "toolResult",
    "success": "operationSuccess",
    "error": "errorMessage",
    "metadata": "operationMetadata"
  }
}
```

#### Specific Resource Mapping
```json
{
  "outputMapping": {
    "createdResource": "result.resource",
    "resourceId": "result.resource.id",
    "resourceUrl": "result.resource.url",
    "operationTime": "result.metadata.duration"
  }
}
```

## Common Use Cases & Templates

### 1. Resource Creation
```json
{
  "callDataAction": {
    "schema": {
      "toolName": "ResourceManage",
      "inputTemplate": {
        "op": "create",
        "resource_type": "{{input.resourceType}}",
        "data": {
          "name": "{{input.resourceName}}",
          "description": "{{input.description}}",
          "isPrivate": "{{input.isPrivate}}",
          "tags": "{{input.tags}}",
          "metadata": {
            "createdBy": "{{userId}}",
            "createdAt": "{{now()}}"
          }
        },
        "options": {
          "validate": true,
          "generateId": true
        }
      },
      "outputMapping": {
        "createdResource": "result",
        "resourceId": "result.id",
        "resourceUrl": "result.publicUrl"
      }
    }
  }
}
```

### 2. User Notification
```json
{
  "callDataAction": {
    "schema": {
      "toolName": "NotificationSend",
      "inputTemplate": {
        "type": "{{input.notificationType}}",
        "recipients": ["{{input.userId}}"],
        "subject": "{{input.subject}}",
        "message": "{{input.message}}",
        "data": {
          "actionUrl": "{{input.actionUrl}}",
          "priority": "{{input.priority}}",
          "category": "{{input.category}}"
        },
        "options": {
          "sendEmail": "{{input.sendEmail}}",
          "sendPush": "{{input.sendPush}}"
        }
      },
      "outputMapping": {
        "notificationId": "result.id",
        "deliveryStatus": "result.status",
        "recipientCount": "result.recipientCount"
      }
    }
  }
}
```

### 3. Team Management
```json
{
  "callDataAction": {
    "schema": {
      "toolName": "TeamManage",
      "inputTemplate": {
        "operation": "{{input.operation}}",
        "team_id": "{{input.teamId}}",
        "member_id": "{{input.memberId}}",
        "role": "{{input.memberRole}}",
        "team_data": {
          "name": "{{input.teamName}}",
          "description": "{{input.teamDescription}}",
          "settings": "{{input.teamSettings}}"
        }
      },
      "outputMapping": {
        "teamInfo": "result.team",
        "membershipStatus": "result.membership",
        "operationResult": "result.success"
      }
    }
  }
}
```

### 4. Data Search and Retrieval
```json
{
  "callDataAction": {
    "schema": {
      "toolName": "ResourceManage",
      "inputTemplate": {
        "op": "search",
        "resource_type": "{{input.resourceType}}",
        "filters": {
          "query": "{{input.searchQuery}}",
          "tags": "{{input.filterTags}}",
          "dateRange": {
            "start": "{{input.startDate}}",
            "end": "{{input.endDate}}"
          },
          "isPrivate": "{{input.includePrivate}}"
        },
        "options": {
          "limit": "{{input.maxResults}}",
          "sort": "{{input.sortOrder}}",
          "includeMetadata": true
        }
      },
      "outputMapping": {
        "searchResults": "result.items",
        "totalCount": "result.total",
        "searchMetadata": "result.metadata"
      }
    }
  }
}
```

### 5. System Monitoring
```json
{
  "callDataAction": {
    "schema": {
      "toolName": "SystemMonitor", 
      "inputTemplate": {
        "metric": "{{input.metricType}}",
        "time_range": {
          "start": "{{input.startTime}}",
          "end": "{{input.endTime}}"
        },
        "filters": {
          "component": "{{input.component}}",
          "severity": "{{input.severity}}",
          "user_id": "{{input.userId}}"
        },
        "aggregation": "{{input.aggregation}}"
      },
      "outputMapping": {
        "metrics": "result.data",
        "summary": "result.summary",
        "alerts": "result.alerts"
      }
    }
  }
}
```

### 6. Routine Execution Management
```json
{
  "callDataAction": {
    "schema": {
      "toolName": "RoutineManage",
      "inputTemplate": {
        "operation": "{{input.operation}}",
        "routine_id": "{{input.routineId}}",
        "run_data": {
          "inputs": "{{input.routineInputs}}",
          "config": "{{input.executionConfig}}",
          "context": {
            "triggeredBy": "{{userId}}",
            "triggerTime": "{{now()}}",
            "parentRun": "{{input.parentRunId}}"
          }
        }
      },
      "outputMapping": {
        "runId": "result.run.id",
        "runStatus": "result.run.status",
        "executionMetadata": "result.metadata"
      }
    }
  }
}
```

## Form Configuration

### Input Form Elements

#### Resource Management
```json
{
  "fieldName": "resourceType",
  "type": "Selector",
  "isRequired": true,
  "label": "Resource Type",
  "options": ["Project", "Routine", "User", "Team", "Note"]
},
{
  "fieldName": "operation",
  "type": "Selector", 
  "isRequired": true,
  "label": "Operation",
  "options": ["create", "read", "update", "delete", "search"]
}
```

#### Data Inputs
```json
{
  "fieldName": "resourceData",
  "type": "JSON",
  "label": "Resource Data",
  "placeholder": "Enter resource data as JSON"
},
{
  "fieldName": "searchFilters",
  "type": "JSON",
  "label": "Search Filters",
  "placeholder": "Enter search criteria"
}
```

#### Configuration Options
```json
{
  "fieldName": "options",
  "type": "JSON",
  "label": "Operation Options",
  "defaultValue": "{\"validate\": true}"
},
{
  "fieldName": "includeMetadata",
  "type": "Checkbox",
  "label": "Include Metadata",
  "defaultValue": true
}
```

### Output Form Elements

#### Operation Results
```json
{
  "fieldName": "operationResult",
  "type": "JSON",
  "label": "Operation Result"
},
{
  "fieldName": "success",
  "type": "Checkbox",
  "label": "Operation Successful"
},
{
  "fieldName": "errorMessage",
  "type": "TextInput",
  "label": "Error Message (if any)"
}
```

#### Resource Information
```json
{
  "fieldName": "resourceId",
  "type": "TextInput",
  "label": "Resource ID"
},
{
  "fieldName": "resourceUrl",
  "type": "TextInput",
  "label": "Resource URL"
},
{
  "fieldName": "operationMetadata",
  "type": "JSON",
  "label": "Operation Metadata"
}
```

## Template Variables

### Platform Variables
- `{{userId}}`: Current user ID
- `{{userEmail}}`: Current user email
- `{{userRole}}`: Current user role
- `{{teamId}}`: Current team ID (if applicable)

### System Variables
- `{{now()}}`: Current timestamp
- `{{generatePrimaryKey()}}`: Generate unique ID
- `{{environment}}`: Current environment (dev/staging/prod)

### Input Variables
```json
{
  "inputTemplate": {
    "user_context": {
      "user_id": "{{userId}}",
      "operation_time": "{{now()}}",
      "requested_action": "{{input.action}}"
    }
  }
}
```

## Execution Strategies

### Deterministic Strategy (Recommended)
**Best for**: Most platform operations requiring consistent execution
```json
{
  "executionStrategy": "deterministic"
}
```

### Reasoning Strategy  
**Best for**: Operations requiring decision-making about parameters
```json
{
  "executionStrategy": "reasoning"
}
```

## Validation Rules

### Critical Requirements
1. **Resource Type**: Must be `"Routine"` 
2. **Resource Sub Type**: Must be `"RoutineInternalAction"`
3. **Config Version**: Must have `"__version": "1.0"` at root level
4. **Call Data**: Must have `callDataAction` with proper schema
5. **Tool Name**: Must be valid MCP tool name
6. **Input Template**: Must define proper tool parameters
7. **Output Mapping**: Must map tool outputs to routine outputs
8. **Forms**: Must have both `formInput` and `formOutput`
9. **IDs**: 19-digit snowflake IDs for all `id` fields
10. **Public IDs**: 10-12 character alphanumeric for `publicId` fields
11. **Tags**: Must be empty array `[]`

### MCP-Specific Requirements
1. **Tool Availability**: Tool must exist and be accessible
2. **Parameter Format**: Input template must match tool schema
3. **Output Structure**: Output mapping must match tool response
4. **Permissions**: User must have required permissions for tool
5. **Resource Access**: Must have access to target resources

### Security Requirements  
1. **Permission Checks**: Validate user permissions for operations
2. **Resource Validation**: Ensure target resources exist and are accessible
3. **Input Sanitization**: Validate and sanitize all input parameters
4. **Error Handling**: Don't expose sensitive information in errors

## Generation Guidelines

### 1. Understand the Platform Operation
- **Tool Purpose**: What does the MCP tool accomplish?
- **Required Permissions**: What permissions are needed?
- **Resource Types**: What platform resources are involved?
- **Operation Scope**: Is this a read or write operation?

### 2. Design the Tool Call
- **Parameter Mapping**: Map form inputs to tool parameters
- **Operation Configuration**: Set appropriate tool options
- **Error Handling**: Plan for common failure scenarios
- **Output Processing**: Extract useful data from tool responses

### 3. Configure Security
- **Permission Requirements**: Document required permissions
- **Input Validation**: Validate all user inputs
- **Resource Access**: Ensure proper resource access controls
- **Audit Trail**: Include metadata for operation tracking

### 4. Plan Input/Output
- **Required Parameters**: Identify essential tool parameters
- **Optional Parameters**: Include commonly used optional parameters
- **Result Processing**: Extract meaningful data from tool responses
- **Status Information**: Capture operation success and error details

### 5. Consider Integration
- **Workflow Context**: How will this be used in larger workflows?
- **Data Flow**: What data needs to flow between operations?
- **Error Recovery**: How should failures be handled?
- **Performance**: Consider operation speed and resource usage

## Quality Checklist

Before generating a RoutineInternalAction routine, verify:

### Structure:
- [ ] `resourceSubType` is `"RoutineInternalAction"`
- [ ] `callDataAction` schema is complete and valid
- [ ] Both `formInput` and `formOutput` are defined
- [ ] All IDs follow correct format
- [ ] `executionStrategy` is appropriate for the operation

### MCP Configuration:
- [ ] Tool name is valid and available
- [ ] Input template matches tool parameter schema
- [ ] Output mapping extracts all necessary data
- [ ] Template variables use correct syntax
- [ ] Required permissions are documented

### Security:
- [ ] Operation requires appropriate permissions
- [ ] Input validation prevents security issues
- [ ] Resource access is properly controlled
- [ ] Error responses don't expose sensitive data

### Platform Integration:
- [ ] Operation aligns with platform capabilities
- [ ] Resource types are valid
- [ ] Tool usage follows platform patterns
- [ ] Workflow integration is considered

### Forms:
- [ ] Input form captures all necessary tool parameters
- [ ] Output form includes results and status information
- [ ] Field names match template variables
- [ ] Required fields are properly marked

### Metadata:
- [ ] Name clearly describes the platform operation
- [ ] Description explains what the action accomplishes
- [ ] Instructions guide users on proper parameter usage
- [ ] Required permissions are documented

Generate RoutineInternalAction routines that provide reliable, secure platform operations while being easily composable into larger workflows.