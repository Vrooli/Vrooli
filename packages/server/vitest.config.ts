import { execSync } from 'child_process';
import path from 'path';

// Detect which config to use
let configFile: string;
try {
    // Run the detection script and get the config filename
    configFile = execSync('node scripts/detect-test-config.js', { 
        encoding: 'utf8',
        cwd: __dirname,
        stdio: ['pipe', 'pipe', 'pipe'] // Capture stdout but not stderr
    }).trim();
    
    // Log the decision to stderr so it doesn't interfere with vitest
    if (process.env.LOG_LEVEL === 'debug' || process.env.VITEST_DEBUG === 'true') {
        console.error(`Using test configuration: ${configFile}`);
    }
} catch (error) {
    // Fallback to single config if detection fails
    console.error('Test config detection failed, using single-fork config');
    configFile = 'vitest.config.single.ts';
}

// Dynamically import and export the selected configuration
let selectedConfig;
if (configFile.includes('parallel')) {
    selectedConfig = await import('./vitest.config.parallel.js');
} else {
    selectedConfig = await import('./vitest.config.single.js');
}

export default selectedConfig.default;