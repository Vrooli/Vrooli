#!/usr/bin/env ts-node
/**
 * Dashboard Capture Example
 * 
 * This example demonstrates how to capture screenshots of internal dashboards
 * using Browserless from TypeScript/Node.js applications.
 */

import * as fs from 'fs/promises';
import * as path from 'path';

// Configuration
const BROWSERLESS_URL = process.env.BROWSERLESS_URL || 'http://localhost:4110';
const BROWSERLESS_TOKEN = process.env.BROWSERLESS_TOKEN || '';

interface DashboardConfig {
    name: string;
    url: string;
    width?: number;
    height?: number;
    waitFor?: number;
    fullPage?: boolean;
}

// Define dashboards to capture
const DASHBOARDS: DashboardConfig[] = [
    {
        name: 'grafana-system-metrics',
        url: 'http://localhost:3000/d/system-metrics/system-metrics',
        width: 1920,
        height: 1080,
        waitFor: 3000, // Wait 3 seconds for metrics to load
        fullPage: true,
    },
    {
        name: 'comfyui-interface',
        url: 'http://localhost:8188',
        width: 1920,
        height: 1080,
        waitFor: 2000,
    },
    {
        name: 'node-red-flows',
        url: 'http://localhost:1880',
        width: 1600,
        height: 900,
        waitFor: 1500,
    },
    {
        name: 'n8n-workflows',
        url: 'http://localhost:5678',
        width: 1920,
        height: 1080,
        waitFor: 2000,
    },
];

/**
 * Capture a screenshot using Browserless
 */
async function captureScreenshot(config: DashboardConfig): Promise<Buffer> {
    const body = {
        url: config.url,
        options: {
            fullPage: config.fullPage ?? false,
            type: 'png',
            viewport: {
                width: config.width || 1920,
                height: config.height || 1080,
                deviceScaleFactor: 1,
            },
        },
        gotoOptions: {
            waitUntil: 'networkidle2' as const,
            timeout: 30000,
        },
        waitFor: config.waitFor || 2000,
    };

    const headers: HeadersInit = {
        'Content-Type': 'application/json',
    };

    if (BROWSERLESS_TOKEN) {
        headers['Authorization'] = `Bearer ${BROWSERLESS_TOKEN}`;
    }

    const response = await fetch(`${BROWSERLESS_URL}/chrome/screenshot`, {
        method: 'POST',
        headers,
        body: JSON.stringify(body),
    });

    if (!response.ok) {
        const error = await response.text();
        throw new Error(`Screenshot failed for ${config.name}: ${error}`);
    }

    return Buffer.from(await response.arrayBuffer());
}

/**
 * Check if a dashboard is accessible
 */
async function checkDashboardHealth(url: string): Promise<boolean> {
    try {
        const response = await fetch(url, { method: 'HEAD' });
        return response.ok;
    } catch {
        return false;
    }
}

/**
 * Main function to capture all dashboards
 */
async function captureDashboards() {
    console.log('🚀 Starting dashboard capture process...\n');

    // Create output directory
    const outputDir = path.join(process.cwd(), 'dashboard-screenshots');
    await fs.mkdir(outputDir, { recursive: true });

    const results = {
        successful: [] as string[],
        failed: [] as { name: string; error: string }[],
    };

    for (const dashboard of DASHBOARDS) {
        console.log(`📸 Capturing ${dashboard.name}...`);

        try {
            // Check if dashboard is accessible
            const isHealthy = await checkDashboardHealth(dashboard.url);
            if (!isHealthy) {
                console.log(`⚠️  ${dashboard.name} is not accessible at ${dashboard.url}`);
                results.failed.push({
                    name: dashboard.name,
                    error: 'Dashboard not accessible',
                });
                continue;
            }

            // Capture screenshot
            const screenshot = await captureScreenshot(dashboard);

            // Save to file
            const timestamp = new Date().toISOString().replace(/:/g, '-').split('.')[0];
            const filename = `${dashboard.name}-${timestamp}.png`;
            const filepath = path.join(outputDir, filename);

            await fs.writeFile(filepath, screenshot);

            console.log(`✅ Saved ${filename} (${(screenshot.length / 1024).toFixed(1)} KB)`);
            results.successful.push(filename);

        } catch (error) {
            console.error(`❌ Failed to capture ${dashboard.name}:`, error.message);
            results.failed.push({
                name: dashboard.name,
                error: error.message,
            });
        }
    }

    // Summary
    console.log('\n📊 Capture Summary:');
    console.log(`✅ Successful: ${results.successful.length}`);
    console.log(`❌ Failed: ${results.failed.length}`);

    if (results.failed.length > 0) {
        console.log('\nFailed dashboards:');
        results.failed.forEach(f => {
            console.log(`  - ${f.name}: ${f.error}`);
        });
    }

    // Save manifest
    const manifest = {
        capturedAt: new Date().toISOString(),
        outputDirectory: outputDir,
        dashboards: DASHBOARDS,
        results,
    };

    await fs.writeFile(
        path.join(outputDir, 'manifest.json'),
        JSON.stringify(manifest, null, 2),
    );

    console.log(`\n💾 Screenshots saved to: ${outputDir}`);
    console.log('📄 Manifest saved to: manifest.json');
}

// Run if executed directly
if (require.main === module) {
    captureDashboards().catch(console.error);
}

export { captureDashboards, captureScreenshot, checkDashboardHealth };