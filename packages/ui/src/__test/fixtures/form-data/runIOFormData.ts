import type { RunIOCreateInput, RunIOUpdateInput } from "@vrooli/shared";

/**
 * Form data fixtures for runIO-related forms
 * These represent data as it appears in form state before submission
 * 
 * RunIO represents input/output data for specific nodes during run execution.
 * Each runIO entry connects to a specific node input and contains the data
 * that flows through that input during processing.
 */

/**
 * Minimal runIO form input - basic text data
 */
export const minimalRunIOFormInput: Partial<RunIOCreateInput> = {
    data: "input data",
    nodeInputName: "input1",
    nodeName: "ProcessNode",
};

/**
 * Complete runIO form input with complex JSON data
 */
export const completeRunIOFormInput = {
    data: JSON.stringify({
        type: "object",
        properties: {
            name: { type: "string" },
            value: { type: "number" },
            enabled: { type: "boolean" },
        },
        values: {
            name: "test input",
            value: 42,
            enabled: true,
        },
        metadata: {
            timestamp: "2024-01-01T00:00:00Z",
            source: "user_input",
            version: "1.0.0",
        },
    }),
    nodeInputName: "complexInput",
    nodeName: "DataProcessorNode",
};

/**
 * RunIO form variants for different data types
 */
export const runIOFormVariants = {
    stringData: {
        data: "Simple string input for text processing",
        nodeInputName: "textInput",
        nodeName: "TextProcessorNode",
    },
    numberData: {
        data: "123.456",
        nodeInputName: "numberInput",
        nodeName: "NumberProcessorNode",
    },
    booleanData: {
        data: "true",
        nodeInputName: "booleanInput",
        nodeName: "BooleanProcessorNode",
    },
    jsonData: {
        data: JSON.stringify({
            message: "hello world",
            count: 5,
            tags: ["test", "data", "json"],
            nested: {
                enabled: true,
                priority: "high",
            },
        }),
        nodeInputName: "jsonInput",
        nodeName: "JsonProcessorNode",
    },
    arrayData: {
        data: JSON.stringify([
            { id: 1, name: "First item" },
            { id: 2, name: "Second item" },
            { id: 3, name: "Third item" },
        ]),
        nodeInputName: "arrayInput",
        nodeName: "ArrayProcessorNode",
    },
    xmlData: {
        data: `<?xml version="1.0" encoding="UTF-8"?>
<root>
    <item id="1">
        <name>Test Item</name>
        <value>42</value>
    </item>
    <item id="2">
        <name>Another Item</name>
        <value>84</value>
    </item>
</root>`,
        nodeInputName: "xmlInput",
        nodeName: "XmlProcessorNode",
    },
    csvData: {
        data: `name,age,email
John Doe,30,john@example.com
Jane Smith,25,jane@example.com
Bob Johnson,35,bob@example.com`,
        nodeInputName: "csvInput",
        nodeName: "CsvProcessorNode",
    },
    binaryData: {
        data: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg==",
        nodeInputName: "imageInput",
        nodeName: "ImageProcessorNode",
    },
    multilineText: {
        data: `This is a multi-line text input
with several lines of content.

It includes empty lines and
    indented sections
        with varying levels of indentation.
        
Special characters: !@#$%^&*()[]{}`,
        nodeInputName: "textInput",
        nodeName: "TextProcessorNode",
    },
    unicodeData: {
        data: "Unicode test: 🚀 🌟 ⭐ 💫 ✨ 中文 العربية हिन्दी Ελληνικά Русский",
        nodeInputName: "unicodeInput",
        nodeName: "UnicodeProcessorNode",
    },
};

/**
 * RunIO update form variants
 */
export const runIOUpdateFormVariants = {
    simpleUpdate: {
        data: "updated input data",
    },
    resultUpdate: {
        data: JSON.stringify({
            status: "processed",
            result: {
                output: "processing complete",
                recordsProcessed: 1000,
                errors: 0,
                warnings: 2,
            },
            metadata: {
                processingTime: 1500,
                completedAt: "2024-01-01T10:30:00Z",
            },
        }),
    },
    errorUpdate: {
        data: JSON.stringify({
            status: "error",
            error: {
                code: "PROCESSING_FAILED",
                message: "Unable to process input data",
                details: "Invalid data format in row 15",
            },
            metadata: {
                failedAt: "2024-01-01T10:15:00Z",
                retryAttempts: 3,
            },
        }),
    },
    progressUpdate: {
        data: JSON.stringify({
            status: "in_progress",
            progress: {
                completed: 750,
                total: 1000,
                percentage: 75,
                currentStep: "validation",
            },
            metadata: {
                startedAt: "2024-01-01T10:00:00Z",
                estimatedCompletion: "2024-01-01T10:45:00Z",
            },
        }),
    },
};

/**
 * RunIO form data for different node types
 */
export const runIONodeTypeVariants = {
    inputNode: {
        data: JSON.stringify({
            source: "file_upload",
            filename: "data.csv",
            size: 1024000,
            checksum: "abc123def456",
        }),
        nodeInputName: "fileInput",
        nodeName: "FileInputNode",
    },
    transformNode: {
        data: JSON.stringify({
            transformation: "normalize",
            parameters: {
                method: "z-score",
                columns: ["age", "income", "score"],
            },
        }),
        nodeInputName: "transformConfig",
        nodeName: "DataTransformNode",
    },
    filterNode: {
        data: JSON.stringify({
            condition: "age > 18 AND status = 'active'",
            matchedRecords: 850,
            filteredRecords: 150,
        }),
        nodeInputName: "filterCriteria",
        nodeName: "FilterNode",
    },
    aggregateNode: {
        data: JSON.stringify({
            groupBy: ["category", "region"],
            aggregations: {
                count: "COUNT(*)",
                total: "SUM(amount)",
                average: "AVG(value)",
            },
            results: [
                { category: "A", region: "North", count: 100, total: 5000, average: 50 },
                { category: "B", region: "South", count: 150, total: 7500, average: 50 },
            ],
        }),
        nodeInputName: "aggregateConfig",
        nodeName: "AggregateNode",
    },
    outputNode: {
        data: JSON.stringify({
            format: "json",
            destination: "results.json",
            recordCount: 1000,
            fileSize: 2048000,
            generatedAt: "2024-01-01T11:00:00Z",
        }),
        nodeInputName: "outputConfig",
        nodeName: "FileOutputNode",
    },
};

/**
 * RunIO form data for testing edge cases
 */
export const runIOEdgeCases = {
    maxLengthData: {
        data: "a".repeat(8192), // Maximum allowed length
        nodeInputName: "maxInput",
        nodeName: "MaxDataNode",
    },
    maxLengthNames: {
        data: "test data",
        nodeInputName: "a".repeat(128), // Maximum length for node input name
        nodeName: "b".repeat(128), // Maximum length for node name
    },
    specialCharacters: {
        data: "Special chars test: !@#$%^&*()[]{}|;':\",./<>?`~+=\\",
        nodeInputName: "special_input-123",
        nodeName: "Special-Node_123",
    },
    emptyData: {
        data: "",
        nodeInputName: "emptyInput",
        nodeName: "EmptyDataNode",
    },
    whitespaceData: {
        data: "   \n\t   \r\n   ",
        nodeInputName: "whitespaceInput",
        nodeName: "WhitespaceNode",
    },
};

/**
 * Invalid runIO form inputs for validation testing
 */
export const invalidRunIOFormInputs = {
    missingData: {
        // data is missing (required field)
        nodeInputName: "input1",
        nodeName: "ProcessNode",
    },
    missingNodeInputName: {
        data: "test data",
        // nodeInputName is missing (required field)
        nodeName: "ProcessNode",
    },
    missingNodeName: {
        data: "test data",
        nodeInputName: "input1",
        // nodeName is missing (required field)
    },
    emptyNodeInputName: {
        data: "test data",
        nodeInputName: "", // Empty string (invalid)
        nodeName: "ProcessNode",
    },
    emptyNodeName: {
        data: "test data",
        nodeInputName: "input1",
        nodeName: "", // Empty string (invalid)
    },
    tooLongData: {
        data: "x".repeat(8193), // Exceeds maximum length
        nodeInputName: "input1",
        nodeName: "ProcessNode",
    },
    tooLongNodeInputName: {
        data: "test data",
        nodeInputName: "x".repeat(129), // Exceeds maximum length
        nodeName: "ProcessNode",
    },
    tooLongNodeName: {
        data: "test data",
        nodeInputName: "input1",
        nodeName: "x".repeat(129), // Exceeds maximum length
    },
};

/**
 * Form validation states for testing
 */
export const runIOFormValidationStates = {
    pristine: {
        values: {},
        errors: {},
        touched: {},
        isValid: false,
        isSubmitting: false,
    },
    withErrors: {
        values: {
            data: "", // Required but empty
            nodeInputName: "", // Required but empty
            nodeName: "", // Required but empty
        },
        errors: {
            data: "Data is required",
            nodeInputName: "Node input name is required",
            nodeName: "Node name is required",
        },
        touched: {
            data: true,
            nodeInputName: true,
            nodeName: true,
        },
        isValid: false,
        isSubmitting: false,
    },
    valid: {
        values: minimalRunIOFormInput,
        errors: {},
        touched: {
            data: true,
            nodeInputName: true,
            nodeName: true,
        },
        isValid: true,
        isSubmitting: false,
    },
    submitting: {
        values: completeRunIOFormInput,
        errors: {},
        touched: {},
        isValid: true,
        isSubmitting: true,
    },
};

/**
 * Helper function to create runIO form initial values
 */
export const createRunIOFormInitialValues = (runIOData?: Partial<any>) => ({
    data: runIOData?.data || "",
    nodeInputName: runIOData?.nodeInputName || "",
    nodeName: runIOData?.nodeName || "",
    ...runIOData,
});

/**
 * Helper function to validate runIO data
 */
export const validateRunIOData = (data: string): string | null => {
    if (!data) return "Data is required";
    if (data.length > 8192) return "Data must be less than 8192 characters";
    return null;
};

/**
 * Helper function to validate node input name
 */
export const validateNodeInputName = (name: string): string | null => {
    if (!name) return "Node input name is required";
    if (name.length > 128) return "Node input name must be less than 128 characters";
    return null;
};

/**
 * Helper function to validate node name
 */
export const validateNodeName = (name: string): string | null => {
    if (!name) return "Node name is required";
    if (name.length > 128) return "Node name must be less than 128 characters";
    return null;
};

/**
 * Helper function to transform form data to API format
 */
export const transformRunIOFormToApiInput = (formData: any, runId?: string) => ({
    data: formData.data,
    nodeInputName: formData.nodeInputName,
    nodeName: formData.nodeName,
    ...(runId && { runConnect: runId }),
});

/**
 * Helper function to format data for display
 */
export const formatRunIODataForDisplay = (data: string, maxLength: number = 100): string => {
    if (!data) return "";
    
    // Try to parse as JSON for pretty formatting
    try {
        const parsed = JSON.parse(data);
        const formatted = JSON.stringify(parsed, null, 2);
        return formatted.length > maxLength 
            ? formatted.substring(0, maxLength) + "..."
            : formatted;
    } catch {
        // Not JSON, return as string with length limit
        return data.length > maxLength 
            ? data.substring(0, maxLength) + "..."
            : data;
    }
};

/**
 * Helper function to detect data type
 */
export const detectRunIODataType = (data: string): string => {
    if (!data) return "empty";
    
    // Check for JSON
    try {
        JSON.parse(data);
        return "json";
    } catch {
        // Not JSON
    }
    
    // Check for XML
    if (data.trim().startsWith("<?xml") || data.trim().startsWith("<")) {
        return "xml";
    }
    
    // Check for CSV (simple heuristic)
    if (data.includes(",") && data.split("\n").length > 1) {
        const lines = data.split("\n");
        const firstLine = lines[0];
        const secondLine = lines[1];
        if (firstLine && secondLine && firstLine.split(",").length === secondLine.split(",").length) {
            return "csv";
        }
    }
    
    // Check for base64 data URI
    if (data.startsWith("data:")) {
        return "binary";
    }
    
    // Check for number
    if (!isNaN(Number(data)) && data.trim() === Number(data).toString()) {
        return "number";
    }
    
    // Check for boolean
    if (data === "true" || data === "false") {
        return "boolean";
    }
    
    // Default to text
    return "text";
};

/**
 * Mock node options for form selects
 */
export const mockNodeOptions = [
    {
        value: "InputNode",
        label: "Input Node",
        type: "input",
        description: "Receives external data input",
    },
    {
        value: "ProcessorNode",
        label: "Processor Node",
        type: "transform",
        description: "Processes and transforms data",
    },
    {
        value: "FilterNode",
        label: "Filter Node",
        type: "filter",
        description: "Filters data based on conditions",
    },
    {
        value: "AggregateNode",
        label: "Aggregate Node",
        type: "aggregate",
        description: "Aggregates data using functions",
    },
    {
        value: "OutputNode",
        label: "Output Node",
        type: "output",
        description: "Outputs processed data",
    },
];

/**
 * Mock input options for form selects
 */
export const mockInputOptions = [
    {
        value: "dataInput",
        label: "Data Input",
        type: "data",
        description: "Main data input port",
    },
    {
        value: "configInput",
        label: "Configuration Input",
        type: "config",
        description: "Node configuration parameters",
    },
    {
        value: "triggerInput",
        label: "Trigger Input",
        type: "trigger",
        description: "Execution trigger signal",
    },
    {
        value: "metadataInput",
        label: "Metadata Input",
        type: "metadata",
        description: "Additional metadata information",
    },
];