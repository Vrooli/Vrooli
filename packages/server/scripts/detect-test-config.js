#!/usr/bin/env node
import os from 'os';
import fs from 'fs';
import { execSync } from 'child_process';

function getSystemResources() {
    const totalMemoryGB = os.totalmem() / (1024 * 1024 * 1024);
    const freeMemoryGB = os.freemem() / (1024 * 1024 * 1024);
    const cpuCores = os.cpus().length;
    
    // Check if running in CI
    const isCI = process.env.CI === 'true' || 
                 process.env.GITHUB_ACTIONS === 'true' ||
                 process.env.GITLAB_CI === 'true' ||
                 process.env.JENKINS === 'true' ||
                 process.env.CIRCLECI === 'true';
    
    // Check Docker memory limits (cgroup v2)
    let dockerMemoryLimit = null;
    try {
        const cgroupMemory = fs.readFileSync('/sys/fs/cgroup/memory.max', 'utf8').trim();
        if (cgroupMemory !== 'max') {
            dockerMemoryLimit = parseInt(cgroupMemory) / (1024 * 1024 * 1024);
        }
    } catch (e) {
        // Try cgroup v1
        try {
            const cgroupMemoryV1 = fs.readFileSync('/sys/fs/cgroup/memory/memory.limit_in_bytes', 'utf8').trim();
            const limitBytes = parseInt(cgroupMemoryV1);
            // Check if it's not the max value (which is very large)
            if (limitBytes < 9223372036854771712) {
                dockerMemoryLimit = limitBytes / (1024 * 1024 * 1024);
            }
        } catch (e2) {
            // Not in Docker or can't read cgroup
        }
    }
    
    // Check if we're in a resource-constrained environment
    const isResourceConstrained = 
        process.env.RESOURCE_CONSTRAINED === 'true' ||
        process.env.LOW_MEMORY === 'true';
    
    return {
        totalMemoryGB,
        freeMemoryGB,
        effectiveMemoryGB: dockerMemoryLimit || totalMemoryGB,
        cpuCores,
        isCI,
        isResourceConstrained,
    };
}

function selectConfig() {
    const resources = getSystemResources();
    
    // Allow manual override
    if (process.env.VITEST_CONFIG) {
        const override = process.env.VITEST_CONFIG.toLowerCase();
        if (override === 'single' || override === 'parallel') {
            if (process.env.LOG_LEVEL === 'debug') {
                console.error(`Manual override: ${override}`);
            }
            return override === 'parallel' ? 'vitest.config.parallel.ts' : 'vitest.config.single.ts';
        }
    }
    
    // Force single-fork in certain conditions
    if (resources.isResourceConstrained) {
        if (process.env.LOG_LEVEL === 'debug') {
            console.error('Resource constrained environment detected, using single config');
        }
        return 'vitest.config.single.ts';
    }
    
    // Decision logic
    const useParallel = 
        resources.effectiveMemoryGB >= 8 &&     // At least 8GB RAM
        resources.freeMemoryGB >= 4 &&           // At least 4GB free
        resources.cpuCores >= 4 &&               // At least 4 cores
        !resources.isCI;                         // Not in CI (unless overridden)
    
    // Log decision if debug mode
    if (process.env.LOG_LEVEL === 'debug' || process.env.VITEST_DEBUG === 'true') {
        console.error('System Resources:', {
            memory: `${resources.effectiveMemoryGB.toFixed(1)}GB (${resources.freeMemoryGB.toFixed(1)}GB free)`,
            cores: resources.cpuCores,
            docker: resources.effectiveMemoryGB !== resources.totalMemoryGB ? `${resources.effectiveMemoryGB.toFixed(1)}GB limit` : 'no limit',
            ci: resources.isCI,
            constrained: resources.isResourceConstrained,
            decision: useParallel ? 'parallel' : 'single'
        });
    }
    
    return useParallel ? 'vitest.config.parallel.ts' : 'vitest.config.single.ts';
}

// Output only the config filename to stdout for script usage
console.log(selectConfig());