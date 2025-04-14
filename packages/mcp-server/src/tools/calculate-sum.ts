import { Logger, ToolResponse } from '../types.js';

/**
 * Implementation of the calculate_sum tool.
 * @param args - The arguments for the tool
 * @param logger - The logger instance for logging operations
 * @returns The result of the calculation
 */
export function calculateSum(args: { a: number, b: number }, logger: Logger): ToolResponse {
    const { a, b } = args;
    const sum = a + b;

    logger.info(`Calculated sum: ${sum}`);
    return {
        content: [
            {
                type: 'text',
                text: String(sum)
            }
        ]
    };
} 