import { Logger } from '../types.js';
import { parseArgs } from './args.js';

export interface ServerConfig {
    mode: 'stdio' | 'sse';
    port: number;
    messagePath: string;
    heartbeatInterval: number;
    serverInfo: {
        name: string;
        version: string;
    };
}

const DEFAULT_CONFIG: ServerConfig = {
    mode: 'sse',
    port: 3100,
    messagePath: '/mcp/messages',
    heartbeatInterval: 30000,
    serverInfo: {
        name: 'vrooli-mcp-server',
        version: '0.1.0'
    }
};

/**
 * Parses and validates the configuration from command line arguments
 * @param args Command line arguments
 * @param logger Logger instance for logging configuration details
 * @returns Validated ServerConfig
 */
export function parseConfig(args: string[], logger?: Logger): ServerConfig {
    const parsedArgs = parseArgs(args);
    const config: ServerConfig = {
        ...DEFAULT_CONFIG,
        mode: parsedArgs.mode || DEFAULT_CONFIG.mode
    };

    logger?.info(`Server configuration loaded: Mode=${config.mode}, Port=${config.port}`);
    return config;
} 