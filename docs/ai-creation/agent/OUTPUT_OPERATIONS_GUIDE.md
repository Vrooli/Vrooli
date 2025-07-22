# OutputOperations Guide for Agent Development

## Overview

OutputOperations enable agents to **accumulate and aggregate data** in the shared blackboard when routine executions complete. This powerful feature allows swarms to build sophisticated data-driven behaviors without complex coordination logic.

## Core Concept

Instead of just returning results, routines can now **update the blackboard** with accumulation operations:
- **Append** arrays to collect results across multiple executions
- **Increment** counters to track totals and metrics  
- **Merge** objects to combine configuration and metadata
- **DeepMerge** nested objects for complex data structures
- **Set** values for simple assignments and status updates

## Supported Operations

### 1. **Append Operations** 
Add arrays to existing blackboard arrays. Perfect for collecting results across multiple routine runs.

```json
"outputOperations": {
  "append": [
    {
      "routineOutput": "result.findings", 
      "blackboardId": "allFindings"
    },
    {
      "routineOutput": "issues",
      "blackboardId": "aggregatedIssues" 
    }
  ]
}
```

**Use Cases:**
- Collecting test results from multiple test runs
- Aggregating log entries or error reports
- Building lists of processed items or completed tasks
- Gathering metrics from multiple sources

### 2. **Increment Operations**
Add numbers to existing blackboard counters. Essential for tracking totals and metrics.

```json
"outputOperations": {
  "increment": [
    {
      "routineOutput": "result.itemsProcessed",
      "blackboardId": "totalItemsProcessed"
    },
    {
      "routineOutput": "metadata.duration", 
      "blackboardId": "totalExecutionTime"
    }
  ]
}
```

**Use Cases:**
- Counting total tests run, passed, failed
- Tracking files processed, lines of code analyzed
- Accumulating execution times and performance metrics
- Building running totals for business metrics

### 3. **Merge Operations**
Shallow merge objects into blackboard objects. Combines top-level properties.

```json
"outputOperations": {
  "merge": [
    {
      "routineOutput": "result.metadata",
      "blackboardId": "projectMetadata"
    },
    {
      "routineOutput": "config.settings",
      "blackboardId": "systemConfiguration"
    }
  ]
}
```

**Use Cases:**
- Updating configuration with new settings
- Combining metadata from multiple sources
- Building comprehensive status objects
- Aggregating simple statistics and summaries

### 4. **DeepMerge Operations**
Recursively merge nested objects. Preserves complex data structures.

```json
"outputOperations": {
  "deepMerge": [
    {
      "routineOutput": "result.nestedConfig",
      "blackboardId": "complexConfiguration"
    },
    {
      "routineOutput": "analysis.detailedMetrics",
      "blackboardId": "comprehensiveMetrics"
    }
  ]
}
```

**Use Cases:**
- Merging complex configuration objects
- Combining nested performance metrics
- Building detailed analysis reports with multiple levels
- Preserving hierarchical data structures

### 5. **Set Operations**
Simple assignment to blackboard keys. Overwrites existing values.

```json
"outputOperations": {
  "set": [
    {
      "routineOutput": "result.status", 
      "blackboardId": "currentStatus"
    },
    {
      "routineOutput": "summary.lastUpdated",
      "blackboardId": "lastProcessingTime"
    }
  ]
}
```

**Use Cases:**
- Setting current status and state information
- Storing latest results and timestamps
- Updating simple configuration values
- Recording final outcomes and decisions

## Dot Notation Support

Extract values from nested routine results using dot notation:

```json
"outputOperations": {
  "append": [{"routineOutput": "analysis.results.items", "blackboardId": "allItems"}],
  "increment": [{"routineOutput": "metrics.performance.duration", "blackboardId": "totalTime"}],
  "set": [{"routineOutput": "status.phase.current", "blackboardId": "currentPhase"}]
}
```

**Examples:**
- `result.data.items` → Extract items from nested result structure
- `metadata.timing.execution` → Get execution time from metadata
- `analysis.summary.confidence` → Extract confidence score from analysis
- `config.database.connection.status` → Get deep nested status value

## Practical Examples

### Software Development Test Aggregation

```json
{
  "trigger": {"topic": "run/completed", "when": "event.data.routine_name == 'run-tests'"},
  "action": {
    "type": "routine",
    "label": "Test Results Processor", 
    "outputOperations": {
      "append": [
        {"routineOutput": "results.testCases", "blackboardId": "all_test_results"},
        {"routineOutput": "results.failures", "blackboardId": "test_failures"}
      ],
      "increment": [
        {"routineOutput": "summary.totalTests", "blackboardId": "total_tests_run"},
        {"routineOutput": "summary.passedTests", "blackboardId": "total_tests_passed"}
      ],
      "merge": [
        {"routineOutput": "metrics.performance", "blackboardId": "test_performance_metrics"}
      ],
      "set": [
        {"routineOutput": "summary.lastRun", "blackboardId": "last_test_execution"}
      ]
    }
  }
}
```

### Marketing Campaign Data Collection

```json
{
  "trigger": {"topic": "step/completed", "when": "event.data.stepType == 'campaign_analysis'"},
  "action": {
    "type": "routine",
    "label": "Campaign Metrics Analyzer",
    "outputOperations": {
      "append": [
        {"routineOutput": "campaigns.results", "blackboardId": "all_campaign_results"}
      ],
      "increment": [
        {"routineOutput": "metrics.impressions", "blackboardId": "total_impressions"},
        {"routineOutput": "metrics.clicks", "blackboardId": "total_clicks"}
      ],
      "deepMerge": [
        {"routineOutput": "analytics.detailed", "blackboardId": "campaign_analytics"}
      ]
    }
  }
}
```

### Code Quality Monitoring

```json
{
  "trigger": {"topic": "run/completed", "when": "event.data.routine_name.includes('code-analysis')"},
  "action": {
    "type": "routine", 
    "label": "Code Quality Processor",
    "outputOperations": {
      "append": [
        {"routineOutput": "issues.critical", "blackboardId": "critical_issues"},
        {"routineOutput": "issues.warnings", "blackboardId": "code_warnings"}
      ],
      "increment": [
        {"routineOutput": "metrics.filesAnalyzed", "blackboardId": "total_files_analyzed"},
        {"routineOutput": "metrics.linesOfCode", "blackboardId": "total_lines_analyzed"}
      ],
      "merge": [
        {"routineOutput": "summary.qualityScores", "blackboardId": "quality_metrics"}
      ]
    }
  }
}
```

## Best Practices

### 1. **Design for Accumulation**
Think about what data should accumulate over time vs. what should be overwritten:
- **Accumulate**: Test results, metrics, issues, processed items
- **Overwrite**: Current status, latest timestamp, active configuration

### 2. **Use Appropriate Operations**
- **Append** for lists and collections that grow over time
- **Increment** for numeric totals and counters
- **Merge** for simple object updates and status changes
- **DeepMerge** for complex nested structures that need preservation
- **Set** for simple values and current state information

### 3. **Plan Blackboard Structure**
Design your blackboard keys to be descriptive and hierarchical:
```
Good: "test_results", "total_tests_run", "code_quality_metrics"
Avoid: "data", "results", "temp"
```

### 4. **Handle Data Types Correctly**
Ensure routine outputs match expected operation types:
- **Append**: Routine must output arrays
- **Increment**: Routine must output numbers
- **Merge/DeepMerge**: Routine must output objects
- **Set**: Routine can output any type

### 5. **Monitor Blackboard Growth**
Consider blackboard size for long-running swarms:
- Implement periodic cleanup routines
- Use summary operations to condense historical data
- Set up monitoring for blackboard memory usage

## Error Handling

OutputOperations include robust error handling:

- **Type Mismatches**: Operations that don't match data types are logged but don't crash execution
- **Missing Values**: Missing routine outputs are handled gracefully 
- **Invalid Paths**: Dot notation paths that don't exist return undefined safely
- **Execution Failures**: OutputOperations failures don't prevent routine completion

All errors are logged for debugging while maintaining system stability.

## Advanced Patterns

### Conditional Accumulation
Use trigger conditions to selectively accumulate data:

```json
{
  "trigger": {
    "topic": "run/completed",
    "when": "event.data.success && event.data.duration > 1000"
  },
  "action": {
    "outputOperations": {
      "append": [{"routineOutput": "result", "blackboardId": "slow_successful_runs"}]
    }
  }
}
```

### Multi-Stage Accumulation
Build complex data structures through multiple behaviors:

```json
// Stage 1: Collect raw data
"outputOperations": {
  "append": [{"routineOutput": "rawData", "blackboardId": "raw_results"}]
}

// Stage 2: Process and accumulate summaries  
"outputOperations": {
  "merge": [{"routineOutput": "summary", "blackboardId": "processed_summaries"}]
}

// Stage 3: Generate final reports using accumulated data
"inputMap": {
  "rawData": "blackboard.raw_results",
  "summaries": "blackboard.processed_summaries"
}
```

### Cross-Agent Coordination
Use blackboard updates to trigger other agents:

```json
// Agent A accumulates data
"outputOperations": {
  "increment": [{"routineOutput": "count", "blackboardId": "items_processed"}]
}

// Agent B triggers when threshold reached
{
  "trigger": {
    "topic": "swarm/blackboard/updated", 
    "when": "event.data.key == 'items_processed' && event.data.value >= 100"
  },
  "action": {"type": "routine", "label": "Process Batch"}
}
```

## Migration from OutputMapping

If you have existing agents using the deprecated `outputMapping`, update them to use `outputOperations`:

### Old (Deprecated):
```json
"outputMapping": {
  "blackboard.results": "routine.output.data",
  "blackboard.count": "routine.output.total"
}
```

### New (Recommended):
```json
"outputOperations": {
  "set": [
    {"routineOutput": "data", "blackboardId": "results"}
  ],
  "increment": [
    {"routineOutput": "total", "blackboardId": "count"}
  ]
}
```

The new format provides:
- **Type safety** with specific operations
- **Accumulation capabilities** instead of just overwriting
- **Dot notation support** for nested data extraction
- **Better error handling** and validation
- **Clearer semantics** about what operations perform

## Next Steps

1. **Update Existing Agents**: Migrate from `outputMapping` to `outputOperations`
2. **Design for Accumulation**: Think about what data should accumulate vs. overwrite
3. **Test Integration**: Verify outputOperations work with your routines
4. **Monitor Performance**: Watch blackboard growth and optimization opportunities
5. **Iterate**: Refine your accumulation patterns based on swarm behavior

OutputOperations unlock powerful emergent behaviors through data accumulation - use them to build sophisticated, coordinated swarm intelligence!