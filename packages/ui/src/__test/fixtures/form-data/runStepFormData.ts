/**
 * Form data fixtures for run step-related forms
 * These represent data as it appears in form state before submission
 */

import { RunStepStatus, DUMMY_ID, type RunStepShape } from "@vrooli/shared";

/**
 * Minimal run step form input - required fields only
 */
export const minimalRunStepFormInput: Partial<RunStepShape> = {
    name: "Basic Processing Step",
    complexity: 1,
    nodeId: "node_001",
    order: 0,
    resourceInId: "123456789012345678",
    run: { id: "987654321098765432" },
};

/**
 * Complete run step form input with all fields
 */
export const completeRunStepFormInput: Partial<RunStepShape> = {
    name: "Advanced Data Processing Step",
    complexity: 150,
    contextSwitches: 3,
    nodeId: "advanced_processor_node_001",
    order: 2,
    status: RunStepStatus.InProgress,
    resourceInId: "123456789012345678",
    timeElapsed: 2500,
    run: { id: "987654321098765432" },
    resourceVersion: { id: "456789012345678901" },
    startedAt: "2024-01-15T10:00:00Z",
    completedAt: null,
};

/**
 * Run step form input for different status variants
 */
export const runStepStatusVariants = {
    inProgress: {
        name: "Active Processing Step",
        complexity: 50,
        nodeId: "active_node",
        order: 1,
        status: RunStepStatus.InProgress,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
        startedAt: "2024-01-15T10:00:00Z",
        timeElapsed: 1200,
    },
    completed: {
        name: "Completed Analysis Step",
        complexity: 100,
        nodeId: "completed_node",
        order: 2,
        status: RunStepStatus.Completed,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
        startedAt: "2024-01-15T10:00:00Z",
        completedAt: "2024-01-15T10:05:00Z",
        timeElapsed: 300000,
    },
    skipped: {
        name: "Skipped Validation Step",
        complexity: 25,
        nodeId: "skipped_node",
        order: 3,
        status: RunStepStatus.Skipped,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
        timeElapsed: 0,
    },
};

/**
 * Run step form input for different complexity levels
 */
export const runStepComplexityVariants = {
    minimal: {
        name: "Simple Step",
        complexity: 0,
        nodeId: "simple_node",
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    low: {
        name: "Low Complexity Step",
        complexity: 10,
        nodeId: "low_complexity_node",
        order: 1,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    medium: {
        name: "Medium Complexity Step",
        complexity: 50,
        nodeId: "medium_complexity_node",
        order: 2,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    high: {
        name: "High Complexity Step",
        complexity: 200,
        nodeId: "high_complexity_node",
        order: 3,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    extreme: {
        name: "Extreme Complexity Step",
        complexity: 999999,
        nodeId: "extreme_complexity_node",
        order: 4,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
};

/**
 * Run step form input for context switching scenarios
 */
export const runStepContextSwitchVariants = {
    noSwitches: {
        name: "Focused Work Step",
        complexity: 50,
        contextSwitches: undefined, // Not set (for new steps)
        nodeId: "focused_node",
        order: 1,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    minimal: {
        name: "Single Context Switch Step",
        complexity: 50,
        contextSwitches: 1,
        nodeId: "minimal_switch_node",
        order: 2,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    moderate: {
        name: "Moderate Context Switch Step",
        complexity: 75,
        contextSwitches: 5,
        nodeId: "moderate_switch_node",
        order: 3,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    frequent: {
        name: "Frequent Context Switch Step",
        complexity: 100,
        contextSwitches: 25,
        nodeId: "frequent_switch_node",
        order: 4,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
};

/**
 * Run step form input for timing scenarios
 */
export const runStepTimingVariants = {
    quick: {
        name: "Quick Processing Step",
        complexity: 5,
        nodeId: "quick_node",
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
        timeElapsed: 100, // 100ms
    },
    normal: {
        name: "Normal Processing Step",
        complexity: 50,
        nodeId: "normal_node",
        order: 1,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
        timeElapsed: 30000, // 30 seconds
    },
    longRunning: {
        name: "Long Running Step",
        complexity: 200,
        nodeId: "long_running_node",
        order: 2,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
        timeElapsed: 300000, // 5 minutes
    },
    marathon: {
        name: "Marathon Processing Step",
        complexity: 500,
        nodeId: "marathon_node",
        order: 3,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
        timeElapsed: 3600000, // 1 hour
    },
};

/**
 * Run step form input for various node types
 */
export const runStepNodeTypeVariants = {
    dataInput: {
        name: "Data Input Step",
        complexity: 10,
        nodeId: "data_input_node",
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    validation: {
        name: "Data Validation Step",
        complexity: 25,
        nodeId: "validation_node",
        order: 1,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    processing: {
        name: "Data Processing Step",
        complexity: 100,
        nodeId: "processing_node",
        order: 2,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    analysis: {
        name: "Analysis Step",
        complexity: 75,
        nodeId: "analysis_node",
        order: 3,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    output: {
        name: "Output Generation Step",
        complexity: 20,
        nodeId: "output_node",
        order: 4,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
};

/**
 * Run step update form input - for editing existing steps
 */
export const runStepUpdateFormInput = {
    minimal: {
        id: "123456789012345678",
        status: RunStepStatus.Completed,
    },
    withContextSwitches: {
        id: "123456789012345678",
        contextSwitches: 8,
        status: RunStepStatus.InProgress,
    },
    withTimeElapsed: {
        id: "123456789012345678",
        timeElapsed: 4500,
        status: RunStepStatus.InProgress,
    },
    complete: {
        id: "123456789012345678",
        contextSwitches: 12,
        status: RunStepStatus.Completed,
        timeElapsed: 7200,
    },
};

/**
 * Invalid form inputs for testing validation
 */
export const invalidRunStepFormInputs = {
    missingName: {
        // @ts-expect-error - Testing missing required name
        name: undefined,
        complexity: 1,
        nodeId: "node_001",
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    emptyName: {
        name: "",
        complexity: 1,
        nodeId: "node_001",
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    whitespaceOnlyName: {
        name: "   \n\t   ",
        complexity: 1,
        nodeId: "node_001",
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    missingComplexity: {
        name: "Test Step",
        // @ts-expect-error - Testing missing required complexity
        complexity: undefined,
        nodeId: "node_001",
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    negativeComplexity: {
        name: "Test Step",
        complexity: -1,
        nodeId: "node_001",
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    missingNodeId: {
        name: "Test Step",
        complexity: 1,
        // @ts-expect-error - Testing missing required nodeId
        nodeId: undefined,
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    emptyNodeId: {
        name: "Test Step",
        complexity: 1,
        nodeId: "",
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    nodeIdTooLong: {
        name: "Test Step",
        complexity: 1,
        nodeId: "x".repeat(129), // Over 128 character limit
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    missingOrder: {
        name: "Test Step",
        complexity: 1,
        nodeId: "node_001",
        // @ts-expect-error - Testing missing required order
        order: undefined,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    negativeOrder: {
        name: "Test Step",
        complexity: 1,
        nodeId: "node_001",
        order: -1,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    missingResourceInId: {
        name: "Test Step",
        complexity: 1,
        nodeId: "node_001",
        order: 0,
        // @ts-expect-error - Testing missing required resourceInId
        resourceInId: undefined,
        run: { id: "987654321098765432" },
    },
    invalidResourceInId: {
        name: "Test Step",
        complexity: 1,
        nodeId: "node_001",
        order: 0,
        resourceInId: "invalid-id",
        run: { id: "987654321098765432" },
    },
    missingRun: {
        name: "Test Step",
        complexity: 1,
        nodeId: "node_001",
        order: 0,
        resourceInId: "123456789012345678",
        // @ts-expect-error - Testing missing required run
        run: undefined,
    },
    invalidRunId: {
        name: "Test Step",
        complexity: 1,
        nodeId: "node_001",
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "invalid-id" },
    },
    invalidStatus: {
        name: "Test Step",
        complexity: 1,
        nodeId: "node_001",
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
        // @ts-expect-error - Testing invalid status
        status: "InvalidStatus",
    },
    zeroContextSwitches: {
        name: "Test Step",
        complexity: 1,
        contextSwitches: 0, // Should be positive when provided
        nodeId: "node_001",
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    negativeContextSwitches: {
        name: "Test Step",
        complexity: 1,
        contextSwitches: -1,
        nodeId: "node_001",
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
    },
    negativeTimeElapsed: {
        name: "Test Step",
        complexity: 1,
        nodeId: "node_001",
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
        timeElapsed: -100,
    },
    invalidResourceVersionId: {
        name: "Test Step",
        complexity: 1,
        nodeId: "node_001",
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
        resourceVersion: { id: "invalid-id" },
    },
};

/**
 * Form validation states for run steps
 */
export const runStepFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            name: "",
            complexity: -1,
            nodeId: "",
            order: -1,
            resourceInId: "",
            run: { id: "" },
        },
        errors: {
            name: "Step name is required",
            complexity: "Complexity must be 0 or greater",
            nodeId: "Node ID is required",
            order: "Order must be 0 or greater",
            resourceInId: "Resource ID is required",
            run: "Run connection is required",
        },
        touched: {
            name: true,
            complexity: true,
            nodeId: true,
            order: true,
            resourceInId: true,
            run: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalRunStepFormInput,
        errors: {},
        touched: {
            name: true,
            complexity: true,
            nodeId: true,
            order: true,
            resourceInId: true,
            run: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: completeRunStepFormInput,
        errors: {},
        touched: {},
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create run step form initial values
 */
export const createRunStepFormInitialValues = (stepData?: Partial<RunStepShape>) => ({
    id: stepData?.id || DUMMY_ID,
    name: stepData?.name || "",
    complexity: stepData?.complexity ?? 1,
    contextSwitches: stepData?.contextSwitches || undefined,
    nodeId: stepData?.nodeId || "",
    order: stepData?.order ?? 0,
    status: stepData?.status || undefined,
    resourceInId: stepData?.resourceInId || "",
    timeElapsed: stepData?.timeElapsed || undefined,
    run: stepData?.run || { id: "" },
    resourceVersion: stepData?.resourceVersion || null,
    startedAt: stepData?.startedAt || undefined,
    completedAt: stepData?.completedAt || undefined,
    ...stepData,
});

/**
 * Helper function to validate run step name
 */
export const validateRunStepName = (name: string): string | null => {
    if (!name) return "Step name is required";
    if (!name.trim()) return "Step name cannot be empty";
    if (name.length < 1) return "Step name must be at least 1 character";
    if (name.length > 250) return "Step name must be less than 250 characters";
    return null;
};

/**
 * Helper function to validate complexity
 */
export const validateComplexity = (complexity: number): string | null => {
    if (complexity === undefined || complexity === null) return "Complexity is required";
    if (typeof complexity !== "number") return "Complexity must be a number";
    if (complexity < 0) return "Complexity must be 0 or greater";
    return null;
};

/**
 * Helper function to validate node ID
 */
export const validateNodeId = (nodeId: string): string | null => {
    if (!nodeId) return "Node ID is required";
    if (!nodeId.trim()) return "Node ID cannot be empty";
    if (nodeId.length > 128) return "Node ID must be less than 128 characters";
    return null;
};

/**
 * Helper function to validate order
 */
export const validateOrder = (order: number): string | null => {
    if (order === undefined || order === null) return "Order is required";
    if (typeof order !== "number") return "Order must be a number";
    if (order < 0) return "Order must be 0 or greater";
    return null;
};

/**
 * Helper function to validate context switches
 */
export const validateContextSwitches = (contextSwitches?: number): string | null => {
    if (contextSwitches === undefined || contextSwitches === null) return null; // Optional field
    if (typeof contextSwitches !== "number") return "Context switches must be a number";
    if (contextSwitches < 1) return "Context switches must be 1 or greater when provided";
    return null;
};

/**
 * Helper function to validate time elapsed
 */
export const validateTimeElapsed = (timeElapsed?: number): string | null => {
    if (timeElapsed === undefined || timeElapsed === null) return null; // Optional field
    if (typeof timeElapsed !== "number") return "Time elapsed must be a number";
    if (timeElapsed < 0) return "Time elapsed must be 0 or greater";
    return null;
};

/**
 * Helper function to transform form data to API format
 */
export const transformRunStepFormToApiInput = (formData: Partial<RunStepShape>) => ({
    ...formData,
    // Convert run connection to connect format
    runConnect: formData.run?.id,
    // Convert resourceVersion connection to connect format if provided
    ...(formData.resourceVersion?.id && { resourceVersionConnect: formData.resourceVersion.id }),
    // Remove form-specific fields
    run: undefined,
    resourceVersion: undefined,
});

/**
 * Mock status options for form selects
 */
export const mockRunStepStatusOptions = [
    { value: RunStepStatus.InProgress, label: "In Progress", description: "Step is currently executing" },
    { value: RunStepStatus.Completed, label: "Completed", description: "Step finished successfully" },
    { value: RunStepStatus.Skipped, label: "Skipped", description: "Step was skipped during execution" },
];

/**
 * Mock run options for form selects
 */
export const mockRunOptions = [
    { value: "987654321098765432", label: "Data Processing Run", description: "Main data processing workflow" },
    { value: "876543210987654321", label: "Analysis Pipeline", description: "Analytical processing pipeline" },
    { value: "765432109876543210", label: "Machine Learning Run", description: "ML model training run" },
    { value: "654321098765432109", label: "Report Generation", description: "Automated report generation" },
];

/**
 * Mock resource version options for form selects
 */
export const mockResourceVersionOptions = [
    { value: "123456789012345678", label: "Data Pipeline v1.0", description: "Main data processing pipeline" },
    { value: "234567890123456789", label: "Analysis Framework v2.1", description: "Advanced analysis framework" },
    { value: "345678901234567890", label: "ML Workflow v1.5", description: "Machine learning workflow" },
    { value: "456789012345678901", label: "Validation Suite v3.0", description: "Data validation and cleaning" },
];

/**
 * Run step form scenarios for different contexts
 */
export const runStepFormScenarios = {
    dataIngestion: {
        name: "Data Ingestion Step",
        complexity: 30,
        nodeId: "data_ingestion_node",
        order: 0,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
        status: RunStepStatus.InProgress,
    },
    preprocessing: {
        name: "Data Preprocessing Step",
        complexity: 75,
        nodeId: "preprocessing_node",
        order: 1,
        contextSwitches: 2,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
        status: RunStepStatus.InProgress,
        timeElapsed: 15000,
    },
    modelTraining: {
        name: "Model Training Step",
        complexity: 500,
        nodeId: "training_node",
        order: 2,
        contextSwitches: 1,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
        status: RunStepStatus.InProgress,
        timeElapsed: 180000,
    },
    validation: {
        name: "Model Validation Step",
        complexity: 100,
        nodeId: "validation_node",
        order: 3,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
        status: RunStepStatus.Completed,
        timeElapsed: 45000,
        startedAt: "2024-01-15T10:00:00Z",
        completedAt: "2024-01-15T10:45:00Z",
    },
    deployment: {
        name: "Model Deployment Step",
        complexity: 50,
        nodeId: "deployment_node",
        order: 4,
        resourceInId: "123456789012345678",
        run: { id: "987654321098765432" },
        status: RunStepStatus.Skipped,
        timeElapsed: 0,
    },
};