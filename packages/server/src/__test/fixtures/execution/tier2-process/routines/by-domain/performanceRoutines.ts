/**
 * Performance and optimization routine fixtures
 * 
 * Routines for performance analysis, cost optimization, and bottleneck detection
 */

import type { 
    RoutineVersionConfigObject,
    CallDataActionConfigObject,
    CallDataCodeConfigObject,
    CallDataGenerateConfigObject,
    FormInputConfigObject,
    FormOutputConfigObject,
} from "@vrooli/shared";
import { McpToolName, ResourceSubType, InputType, BotStyle } from "@vrooli/shared";
import type { RoutineFixture, RoutineFixtureCollection } from "./types.js";

/**
 * Performance Bottleneck Detection Routine
 * Used by: Routine Performance Agent
 * Purpose: Identifies performance bottlenecks in routine execution
 */
export const PERFORMANCE_BOTTLENECK_DETECTION: RoutineFixture = {
    id: "performance_bottleneck_detection",
    name: "Performance Bottleneck Detection",
    description: "Analyzes routine execution metrics to identify performance bottlenecks",
    version: "1.0.0",
    resourceSubType: ResourceSubType.RoutineCode,
    config: {
        __version: "1.0",
        callDataCode: {
            __version: "1.0",
            schema: {
                inputTemplate: {
                    executionMetrics: "{{input.metrics}}",
                    routineConfig: "{{input.config}}",
                    threshold: "{{input.threshold}}",
                },
                outputMappings: [
                    {
                        schemaIndex: 0,
                        mapping: {
                            bottlenecks: "bottlenecks",
                            optimizationOpportunities: "opportunities",
                            performanceScore: "score",
                            recommendations: "recommendations",
                        },
                    },
                ],
            },
        } as CallDataCodeConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "metrics",
                        id: "metrics",
                        label: "Execution Metrics",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "config",
                        id: "config",
                        label: "Routine Configuration",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "threshold",
                        id: "threshold",
                        label: "Performance Threshold (ms)",
                        type: InputType.IntegerInput,
                        isRequired: true,
                        defaultValue: 1000,
                    },
                ],
            },
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "bottlenecks",
                        id: "bottlenecks",
                        label: "Detected Bottlenecks",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                    {
                        fieldName: "optimizationOpportunities",
                        id: "optimization_opportunities",
                        label: "Optimization Opportunities",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                    {
                        fieldName: "performanceScore",
                        id: "performance_score",
                        label: "Performance Score",
                        type: InputType.IntegerInput,
                        props: { disabled: true, min: 0, max: 100 },
                    },
                    {
                        fieldName: "recommendations",
                        id: "recommendations",
                        label: "Recommendations",
                        type: InputType.Text,
                        props: { disabled: true },
                    },
                ],
            },
        } as FormOutputConfigObject,
        executionStrategy: "deterministic" as const,
    } as RoutineVersionConfigObject,
};

/**
 * Cost Analysis Routine
 * Used by: Cost Optimization Agent
 * Purpose: Analyzes operational costs and identifies savings opportunities
 */
export const COST_ANALYSIS: RoutineFixture = {
    id: "cost_analysis",
    name: "Operational Cost Analysis",
    description: "Analyzes operational costs to identify reduction opportunities",
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
                    resource_type: "Cost",
                    filters: {
                        billingData: "{{input.billingData}}",
                        resourceUsage: "{{input.resourceUsage}}",
                        timeframe: "{{input.timeframe}}",
                    },
                    limit: 1000,
                }),
                outputMapping: {
                    currentCost: "totalCost",
                    breakdown: "costBreakdown",
                    savingsOpportunities: "opportunities",
                    recommendations: "recommendations",
                },
            },
        } as CallDataActionConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "billingData",
                        id: "billing_data",
                        label: "Billing Data",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "resourceUsage",
                        id: "resource_usage",
                        label: "Resource Usage Data",
                        type: InputType.JSON,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "timeframe",
                        id: "timeframe",
                        label: "Analysis Timeframe",
                        type: InputType.Dropdown,
                        isRequired: true,
                        defaultValue: "30d",
                        props: {
                            options: [
                                { label: "Last 7 days", value: "7d" },
                                { label: "Last 30 days", value: "30d" },
                                { label: "Last 90 days", value: "90d" },
                            ],
                        },
                    },
                ],
            },
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "currentCost",
                        id: "current_cost",
                        label: "Total Cost",
                        type: InputType.IntegerInput,
                        props: { disabled: true },
                    },
                    {
                        fieldName: "breakdown",
                        id: "breakdown",
                        label: "Cost Breakdown",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                    {
                        fieldName: "savingsOpportunities",
                        id: "savings_opportunities",
                        label: "Savings Opportunities",
                        type: InputType.JSON,
                        props: { disabled: true },
                    },
                    {
                        fieldName: "recommendations",
                        id: "recommendations",
                        label: "Cost Reduction Recommendations",
                        type: InputType.Text,
                        props: { disabled: true },
                    },
                ],
            },
        } as FormOutputConfigObject,
        executionStrategy: "reasoning" as const,
    } as RoutineVersionConfigObject,
};

/**
 * Output Quality Assessment Routine
 * Used by: Output Quality Monitor
 * Purpose: Assesses the quality of generated outputs
 */
export const OUTPUT_QUALITY_ASSESSMENT: RoutineFixture = {
    id: "output_quality_assessment",
    name: "Output Quality Assessment",
    description: "Evaluates output quality including accuracy, completeness, and bias detection",
    version: "1.0.0",
    resourceSubType: ResourceSubType.RoutineGenerate,
    config: {
        __version: "1.0",
        callDataGenerate: {
            __version: "1.0",
            schema: {
                botStyle: BotStyle.Analytical,
                maxTokens: 800,
                model: null,
                prompt: "Assess the quality of the following output:\n\nOutput: {{input.output}}\nExpected Criteria: {{input.criteria}}\n\nEvaluate:\n1. Accuracy score (0-100)\n2. Completeness score (0-100)\n3. Consistency score (0-100)\n4. Bias detection (demographic, cultural, linguistic)\n5. Overall quality score\n\nProvide detailed assessment with specific examples.",
                respondingBot: null,
            },
        } as CallDataGenerateConfigObject,
        formInput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "output",
                        id: "output",
                        label: "Output to Assess",
                        type: InputType.Text,
                        isRequired: true,
                        props: {},
                    },
                    {
                        fieldName: "criteria",
                        id: "criteria",
                        label: "Quality Criteria",
                        type: InputType.JSON,
                        isRequired: true,
                        defaultValue: {
                            accuracyTarget: 0.85,
                            completenessTarget: 0.90,
                            biasThreshold: 0.15,
                        },
                    },
                ],
            },
        } as FormInputConfigObject,
        formOutput: {
            __version: "1.0",
            schema: {
                containers: [],
                elements: [
                    {
                        fieldName: "response",
                        id: "response",
                        label: "Quality Assessment Report",
                        type: InputType.Text,
                        props: { disabled: true },
                    },
                ],
            },
        } as FormOutputConfigObject,
        executionStrategy: "reasoning" as const,
    } as RoutineVersionConfigObject,
};

export const PERFORMANCE_ROUTINES: RoutineFixtureCollection<"PERFORMANCE_BOTTLENECK_DETECTION" | "COST_ANALYSIS" | "OUTPUT_QUALITY_ASSESSMENT"> = {
    PERFORMANCE_BOTTLENECK_DETECTION,
    COST_ANALYSIS,
    OUTPUT_QUALITY_ASSESSMENT,
};
