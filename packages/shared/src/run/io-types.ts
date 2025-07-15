/**
 * IO types extracted from run/types.ts to break circular dependencies
 */

export type SubroutineOutputDisplayInfoProps = ({
    /**
     * The type of the output, however you want to represent it based on the standard version 
     * linked to the input/output. 
     * 
     * Examples:
     * - "JSON"
     * - "CSV"
     * - "Markdown"
     * - "HTML"
     * - "Text"
     * - "True/False"
     * - "List of URLs"
     * - "JavaScript code"
     * - "Python code"
     * - "SQL"
     * - "GraphQL"
     * - "JSON Schema"
     * - "OpenAPI (Swagger)"
     */
    type: string;
    /**
     * The schema that the input/output should conform to, if any.
     * 
     * This is typically a JSON Schema, but could be something else depending on how the 
     * standard version is configured.
     */
    schema: string | undefined;
    // Any other props that are relevant to the input/output can be added here, such as max length, min value, etc.
} & { [key: string]: unknown })

/**
 * Display information for a subroutine output.
 */
export type SubroutineOutputDisplayInfo = {
    /** 
     * What value we'll set if the user or LLM don't provide one.
     * 
     * This should not be shown to the LLM, so as not to influence the AI's output.
     */
    defaultValue: unknown | undefined;
    /** The description of the output */
    description: string | undefined;
    /** The name of the output */
    name: string | undefined;
    /** 
     * The configuration of the input/output, if any.
     * 
     * With this information, we can tell the LLM something like:
     * "Generate the following inputs:
     * {
     *     "inputA": {
     *         "name": "Input A",
     *         "description": "This is the first input",
     *         "isRequired": true,
     *         "type": "JSON object",
     *         "schema": {
     *             "propA": {
     *                 "type": "string",
     *                 "isRequired": true,
     *                 "description": "This is the first property of the input"
     *             }
     *         }
     *     }
     *     //...
     * }
     */
    props: SubroutineOutputDisplayInfoProps | undefined;
    /** The existing value of the output, if any */
    value: unknown | undefined;
}

/**
 * Display information for a subroutine input.
 */
export type SubroutineInputDisplayInfo = SubroutineOutputDisplayInfo & {
    /** Whether the input is required */
    isRequired: boolean;
}

/**
 * A mapping of input and output names to their current value and display information.
 * 
 * It's important that we have enough information for an AI to understand what each input and output is for.
 */
export type SubroutineIOMapping = {
    inputs: Record<string, SubroutineInputDisplayInfo>;
    outputs: Record<string, SubroutineOutputDisplayInfo>;
}
