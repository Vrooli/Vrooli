import { type ExecutionResult } from "@local/shared";

/**
 * Temporary types until proper imports are available
 */
interface SubroutineIODisplayInfo {
    defaultValue: unknown | undefined;
    description: string | undefined;
    name: string | undefined;
    props: SubroutineOutputDisplayInfoProps | undefined;
    value: unknown | undefined;
}

interface SubroutineInputDisplayInfo extends SubroutineIODisplayInfo {
    isRequired: boolean;
}

type SubroutineOutputDisplayInfo = SubroutineIODisplayInfo;

interface SubroutineOutputDisplayInfoProps {
    type: string;
    schema: string | undefined;
    [key: string]: unknown;
}

interface SubroutineIOMapping {
    inputs: Record<string, SubroutineInputDisplayInfo>;
    outputs: Record<string, SubroutineOutputDisplayInfo>;
}

// Constants for default values
const DEFAULT_TEXT_MAX_LENGTH = 50;
const DEFAULT_NUMERIC_VALUE = 42;

/**
 * Test execution component extracted from legacy SubroutineExecutor.
 * Handles zero-credit execution simulation with type-aware dummy value generation.
 */
export class TestExecutor {
    /**
     * Migrated from SubroutineExecutor.dummyRun()
     * Zero-credit execution simulation
     */
    async executeTestMode(ioMapping: SubroutineIOMapping): Promise<ExecutionResult> {
        const result: ExecutionResult = {
            success: true,
            ioMapping: { ...ioMapping },
            creditsUsed: BigInt(0),
            timeElapsed: 0,
            toolCallsCount: 0,
            metadata: {
                strategy: "test_mode",
                warnings: ["This is a test run. No actual execution was performed."],
            } as any,
        };

        // Generate dummy inputs for required missing inputs
        for (const [key, input] of Object.entries(ioMapping.inputs)) {
            if (input.isRequired && input.value === undefined) {
                result.ioMapping.inputs[key] = {
                    ...input,
                    value: this.generateDummyValue(input),
                };
            }
        }

        // Generate dummy outputs
        for (const [key, output] of Object.entries(ioMapping.outputs)) {
            if (output.value === undefined) {
                result.ioMapping.outputs[key] = {
                    ...output,
                    value: this.generateDummyValue(output),
                };
            }
        }

        return result;
    }

    /**
     * Migrated from SubroutineExecutor.generateDummyValue()
     * Type-aware dummy value generation based on props
     */
    private generateDummyValue(ioInfo: SubroutineInputDisplayInfo | SubroutineOutputDisplayInfo): unknown {
        // Use default value if available
        if (ioInfo.defaultValue !== undefined) {
            return ioInfo.defaultValue;
        }

        // If no props available, return simple dummy value
        if (!ioInfo.props) {
            return "dummy_value";
        }

        const { type } = ioInfo.props;

        // Handle different types
        switch (type) {
            case "Boolean":
                return true;

            case "Integer":
                return this.generateDummyInteger(ioInfo.props);

            case "Number":
                return this.generateDummyNumber(ioInfo.props);

            case "Text":
                return this.generateDummyText(ioInfo.props);

            case "URL":
                return this.generateDummyUrl(ioInfo.props);

            case "List of language codes":
                return ["en", "es"];

            case "List of alphanumeric tags (a.k.a. hashtags, keywords)":
                return ["dummy", "test", "example"];

            case "File":
                return this.generateDummyFile(ioInfo.props);

            case "Files":
                return this.generateDummyFiles(ioInfo.props);

            default:
                // Handle complex types
                if (type.startsWith("List of")) {
                    return this.generateDummyList(type, ioInfo.props);
                }

                if (type.startsWith("One of")) {
                    return this.generateDummySelection(type, ioInfo.props);
                }

                if (type.includes("JSON") || type.includes("schema")) {
                    return this.generateDummyJson(ioInfo.props);
                }

                if (type.includes("Code")) {
                    return this.generateDummyCode(type);
                }

                if (type.includes("Item ID")) {
                    return "RoutineVersion:dummy-123-456-789";
                }

                // Fallback for unknown types
                return "dummy_value";
        }
    }

    /**
     * Generate dummy integer value with constraints
     */
    private generateDummyInteger(props: SubroutineOutputDisplayInfoProps): number {
        const min = typeof props.min === "number" ? props.min : 0;
        const max = typeof props.max === "number" ? props.max : 100;
        return Math.floor(Math.random() * (max - min + 1)) + min;
    }

    /**
     * Generate dummy number value with constraints
     */
    private generateDummyNumber(props: SubroutineOutputDisplayInfoProps): number {
        const min = typeof props.min === "number" ? props.min : 0;
        const max = typeof props.max === "number" ? props.max : 100;
        return Math.random() * (max - min) + min;
    }

    /**
     * Generate dummy text value with length constraints
     */
    private generateDummyText(props: SubroutineOutputDisplayInfoProps): string {
        const maxLength = typeof props.maxLength === "number" ? props.maxLength : DEFAULT_TEXT_MAX_LENGTH;
        const text = "This is dummy text for testing purposes.";
        return text.length > maxLength ? text.substring(0, maxLength) : text;
    }

    /**
     * Generate dummy URL with host constraints
     */
    private generateDummyUrl(props: SubroutineOutputDisplayInfoProps): string {
        const acceptedHosts = props.acceptedHosts as string[] | undefined;
        if (acceptedHosts && acceptedHosts.length > 0) {
            const host = acceptedHosts[0];
            return `https://${host}/dummy-path`;
        }
        return "https://example.com/dummy-path";
    }

    /**
     * Generate dummy file object
     */
    private generateDummyFile(props: SubroutineOutputDisplayInfoProps): object {
        const acceptedTypes = props.acceptedFileTypes as string[] | undefined;
        const fileType = acceptedTypes && acceptedTypes.length > 0 ? acceptedTypes[0] : "text/plain";

        return {
            name: "dummy-file.txt",
            type: fileType,
            size: 1024,
            lastModified: Date.now(),
            // Note: In real implementation, this would be a File object
            content: "This is dummy file content for testing.",
        };
    }

    /**
     * Generate dummy files array
     */
    private generateDummyFiles(props: SubroutineOutputDisplayInfoProps): object[] {
        const maxFiles = typeof props.maxFiles === "number" ? props.maxFiles : 2;
        const files: object[] = [];

        for (let i = 0; i < Math.min(maxFiles, 3); i++) {
            files.push({
                ...this.generateDummyFile(props),
                name: `dummy-file-${i + 1}.txt`,
            });
        }

        return files;
    }

    /**
     * Generate dummy list values
     */
    private generateDummyList(type: string, props: SubroutineOutputDisplayInfoProps): unknown[] {
        const options = props.options as { label: string; value: unknown }[] | undefined;

        if (options && options.length > 0) {
            // Return first two options as dummy selection
            return options.slice(0, Math.min(2, options.length)).map(opt => opt.value);
        }

        // Generate generic list based on type
        if (type.includes("custom")) {
            return ["dummy_option_1", "dummy_option_2", "custom_value"];
        }

        return ["option_1", "option_2"];
    }

    /**
     * Generate dummy selection from options
     */
    private generateDummySelection(type: string, props: SubroutineOutputDisplayInfoProps): unknown {
        const options = props.options as { label: string; value: unknown }[] | undefined;

        if (options && options.length > 0) {
            // Return first option as dummy selection
            return options[0].value;
        }

        if (type.includes("null")) {
            return null;
        }

        return "dummy_selection";
    }

    /**
     * Generate dummy JSON object
     */
    private generateDummyJson(props: SubroutineOutputDisplayInfoProps): object {
        if (props.schema) {
            try {
                // Try to parse schema and generate appropriate structure
                const schema = JSON.parse(props.schema);
                return this.generateFromSchema(schema);
            } catch {
                // If schema parsing fails, return simple object
            }
        }

        return {
            message: "This is a dummy JSON object",
            timestamp: Date.now(),
            success: true,
            data: {
                example: "value",
                nested: {
                    property: "dummy",
                },
            },
        };
    }

    /**
     * Generate dummy code snippet
     */
    private generateDummyCode(type: string): string {
        const languages = type.split(":").slice(1).join(":").trim();

        if (languages.includes("python")) {
            return "# Dummy Python code\nprint('Hello, World!')";
        }

        if (languages.includes("javascript")) {
            return "// Dummy JavaScript code\nconsole.log('Hello, World!');";
        }

        if (languages.includes("sql")) {
            return "-- Dummy SQL code\nSELECT 'Hello, World!' AS message;";
        }

        return "# Dummy code\nprint('Hello, World!')";
    }

    /**
     * Generate object from JSON schema (simplified)
     */
    private generateFromSchema(schema: any): object {
        if (typeof schema !== "object" || !schema) {
            return { dummy: "value" };
        }

        const result: any = {};

        if (schema.properties) {
            for (const [key, propSchema] of Object.entries(schema.properties)) {
                result[key] = this.generateFromPropertySchema(propSchema as any);
            }
        }

        return result;
    }

    /**
     * Generate value from property schema
     */
    private generateFromPropertySchema(propSchema: any): unknown {
        if (!propSchema || typeof propSchema !== "object") {
            return "dummy";
        }

        switch (propSchema.type) {
            case "string":
                return propSchema.default || "dummy_string";
            case "number":
                return propSchema.default || DEFAULT_NUMERIC_VALUE;
            case "integer":
                return propSchema.default || DEFAULT_NUMERIC_VALUE;
            case "boolean":
                return propSchema.default !== undefined ? propSchema.default : true;
            case "array":
                return propSchema.default || ["dummy_item"];
            case "object":
                return propSchema.default || { nested: "dummy" };
            default:
                return propSchema.default || "dummy_value";
        }
    }
} 
