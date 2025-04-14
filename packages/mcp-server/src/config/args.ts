/**
 * Interface representing parsed command line arguments
 */
export interface ParsedArgs {
    mode?: 'stdio' | 'sse';
}

/**
 * Parses command line arguments
 * @param args Command line arguments array (typically process.argv.slice(2))
 * @returns Object containing parsed arguments
 */
export function parseArgs(args: string[]): ParsedArgs {
    const parsedArgs: ParsedArgs = {};

    args.forEach(arg => {
        if (arg.startsWith('--mode=')) {
            const value = arg.split('=')[1];
            if (value === 'sse' || value === 'stdio') {
                parsedArgs.mode = value;
            }
        }
    });

    return parsedArgs;
} 