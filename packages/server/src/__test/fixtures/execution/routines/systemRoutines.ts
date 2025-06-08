/**
 * System monitoring and resilience routine fixtures
 * 
 * Routines for system health monitoring, failure analysis, and resilience optimization
 */

import type { 
    RoutineVersionConfigObject,
    CallDataActionConfigObject,
    CallDataWebConfigObject,
    FormInputConfigObject,
    FormOutputConfigObject,
} from "@vrooli/shared";
import { McpToolName, ResourceSubType, InputType } from "@vrooli/shared";
import type { RoutineFixture, RoutineFixtureCollection } from "./types.js";

/**
 * System Log Search Routine
 * Used by: System monitoring agents
 * Purpose: Searches system logs for specific patterns
 */
export const SYSTEM_LOG_SEARCH: RoutineFixture = {
    id: "system_log_search",
    name: "System Log Search",
    description: "Searches system logs using resource management",
    version: "1.0.0",
    resourceSubType: ResourceSubType.RoutineInternalAction,
    config: {
        __version: "1.0",
        callDataAction: {
            __version: "1.0",
            schema: {
                toolName: McpToolName.ResourceManage,
                inputTemplate: JSON.stringify({
                    op: "search",
                    resource_type: "Log",
                    filters: {
                        content: "{{input.searchPattern}}",
                        timeRange: "{{input.timeRange}}",
                        severity: "{{input.severity}}"
                    },
                    limit: 500
                }),
                outputMapping: {
                    logs: "items",
                    count: "count"
                }
            }
        } as CallDataActionConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "searchPattern",
                        id: "search_pattern",
                        label: "Search Pattern",
                        type: InputType.Text,
                        isRequired: true
                    },
                    {
                        fieldName: "timeRange",
                        id: "time_range",
                        label: "Time Range",
                        type: InputType.Text,
                        isRequired: true,
                        defaultValue: "24h"
                    },
                    {
                        fieldName: "severity",
                        id: "severity",
                        label: "Log Severity",
                        type: InputType.Dropdown,
                        isRequired: false,
                        props: {
                            options: [
                                { label: "All", value: "all" },
                                { label: "Error", value: "error" },
                                { label: "Warning", value: "warning" },
                                { label: "Info", value: "info" }
                            ]
                        }
                    }
                ]
            }
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "logs",
                        id: "logs",
                        label: "Found Logs",
                        type: InputType.JSON,
                        props: { disabled: true }
                    },
                    {
                        fieldName: "count",
                        id: "count",
                        label: "Total Count",
                        type: InputType.IntegerInput,
                        props: { disabled: true }
                    }
                ]
            }
        } as FormOutputConfigObject,
        executionStrategy: "deterministic" as const,
    } as RoutineVersionConfigObject,
};

/**
 * System Health Check Routine
 * Used by: System Health Monitor
 * Purpose: Monitors overall system health and predicts issues
 */
export const SYSTEM_HEALTH_CHECK: RoutineFixture = {
    id: "system_health_check",
    name: "System Health Check",
    description: "Comprehensive system health monitoring and anomaly detection",
    version: "1.0.0",
    resourceSubType: ResourceSubType.RoutineWeb,
    config: {
        __version: "1.0",
        callDataWeb: {
            __version: "1.0",
            schema: {
                queryTemplate: "system monitoring {{input.systemName}} health metrics best practices",
                searchEngine: "google",
                maxResults: 5,
                outputMapping: {
                    bestPractices: "results[*].snippet",
                    resources: "results[*].link"
                }
            }
        } as CallDataWebConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "systemName",
                        id: "system_name",
                        label: "System Name",
                        type: InputType.Text,
                        isRequired: true,
                        defaultValue: "kubernetes cluster"
                    }
                ]
            }
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "bestPractices",
                        id: "best_practices",
                        label: "Monitoring Best Practices",
                        type: InputType.JSON,
                        props: { disabled: true }
                    },
                    {
                        fieldName: "resources",
                        id: "resources",
                        label: "Reference Resources",
                        type: InputType.JSON,
                        props: { disabled: true }
                    }
                ]
            }
        } as FormOutputConfigObject,
        executionStrategy: "deterministic" as const,
    } as RoutineVersionConfigObject,
};

export const SYSTEM_ROUTINES: RoutineFixtureCollection<"SYSTEM_LOG_SEARCH" | "SYSTEM_HEALTH_CHECK"> = {
    SYSTEM_LOG_SEARCH,
    SYSTEM_HEALTH_CHECK,
} as const;