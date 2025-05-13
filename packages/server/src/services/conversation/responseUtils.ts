import { logger } from "../../events/logger.js";

/** Error details structure */
interface ToolResultError {
    code: string;
    message: string;
}

/**
 * Represents the structured output payload for a function call result.
 * Includes success status and optional error information or successful data.
 */
export interface FunctionCallOutputPayload {
    /** Indicates if the tool execution was successful. */
    ok: boolean;
    /** Error details if ok is false. */
    error?: ToolResultError;
    /** 
     * The actual output data from the tool if ok is true.
     * Can sometimes include original output even if ok is false 
     * (e.g., for limit exceeded errors after the tool ran).
     */
    [key: string]: any; // Allow including original tool output fields
}

/**
 * Represents the complete function_call_output object structure expected by the LLM.
 */
export interface FunctionCallOutput {
    /** The type identifier for this object. */
    type: "function_call_output";
    /** The unique ID of the function call this output corresponds to. */
    call_id: string;
    /** The payload containing the result or error details. */
    output: FunctionCallOutputPayload;
}

/**
 * Provides static methods for generating standardized response objects for the LLM.
 */
export class OutputGenerator {
    /**
     * Generates a standardized `function_call_output` object for responding to the LLM.
     *
     * @param call_id - The ID of the original function call event.
     * @param options - Options object containing success status, error details, or success output.
     * @param options.ok - Boolean indicating if the operation succeeded.
     * @param options.errorCode - Optional error code string (e.g., "TOOL_ERROR", "TURN_LIMIT_EXCEEDED").
     * @param options.errorMessage - Optional descriptive error message.
     * @param options.outputData - Optional data payload for successful results, or potentially
     *                             original output even in case of certain errors (like limit exceeded).
     * @returns A structured FunctionCallOutput object.
     */
    static functionalCallOutput(
        call_id: string,
        options: {
            ok: boolean;
            errorCode?: string;
            errorMessage?: string;
            outputData?: Record<string, any>;
        },
    ): FunctionCallOutput {
        const outputPayload: FunctionCallOutputPayload = {
            ok: options.ok,
            ...(options.outputData || {}), // Spread success/original data first
        };

        if (!options.ok) {
            if (!options.errorCode || !options.errorMessage) {
                console.warn(`Generating failed function_call_output for call_id ${call_id} without error code/message.`);
            }
            // Ensure error object exists and overwrite any conflicting keys from outputData if necessary
            outputPayload.error = {
                code: options.errorCode ?? "UNKNOWN_ERROR",
                message: options.errorMessage ?? "An unknown error occurred.",
            };
        } else if (options.errorCode || options.errorMessage) {
            console.warn(`Generating successful function_call_output for call_id ${call_id} but error details were provided. Ignoring error details.`);
        }

        return {
            type: "function_call_output",
            call_id,
            output: outputPayload,
        };
    }

    /**
     * Generates a failed function_call_output object with the given error details.
     * 
     * @param call_id - The ID of the original function call event.
     * @param errorCode - The error code to be included in the output.
     * @param errorMessage - The error message to be included in the output.
     * @returns A FunctionCallOutput object with the given error details.
     */
    static functionCallOutputError(
        call_id: string,
        errorCode: string,
        errorMessage: string,
    ): FunctionCallOutput {
        logger.warning(`Generating failed function_call_output for call_id ${call_id} with error code ${errorCode} and message ${errorMessage}.`);
        return this.functionalCallOutput(call_id, {
            ok: false,
            errorCode,
            errorMessage,
        });
    }

    /**
     * Generates a successful function_call_output object with the given output data.
     * 
     * @param call_id - The ID of the original function call event.
     * @param outputData - The data payload to be included in the output.
     * @returns A FunctionCallOutput object with the given output data.
     */
    static functionCallOutputSuccess(
        call_id: string,
        outputData: Record<string, any>,
    ): FunctionCallOutput {
        return this.functionalCallOutput(call_id, { ok: true, outputData });
    }
} 
