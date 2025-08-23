#!/usr/bin/env ts-node
/**
 * Service Monitoring Example
 * 
 * This example shows how to monitor internal services by:
 * 1. Checking service health pages
 * 2. Extracting status information
 * 3. Taking screenshots for visual verification
 * 4. Sending alerts if issues are detected
 */

import * as fs from 'fs/promises';
import * as path from 'path';

// Configuration
const BROWSERLESS_URL = process.env.BROWSERLESS_URL || 'http://localhost:4110';
const BROWSERLESS_TOKEN = process.env.BROWSERLESS_TOKEN || '';

interface ServiceConfig {
    name: string;
    url: string;
    healthUrl?: string;
    statusSelector?: string;
    versionSelector?: string;
    expectedStatus?: string;
    screenshotOnError?: boolean;
}

// Services to monitor
const SERVICES: ServiceConfig[] = [
    {
        name: 'Grafana',
        url: 'http://localhost:3000',
        healthUrl: 'http://localhost:3000/api/health',
        statusSelector: '.health-status',
        expectedStatus: 'ok',
        screenshotOnError: true,
    },
    {
        name: 'ComfyUI',
        url: 'http://localhost:8188',
        statusSelector: '#status',
        screenshotOnError: true,
    },
    {
        name: 'n8n',
        url: 'http://localhost:5678',
        healthUrl: 'http://localhost:5678/healthz',
        expectedStatus: 'ok',
        screenshotOnError: true,
    },
    {
        name: 'Node-RED',
        url: 'http://localhost:1880',
        statusSelector: '#red-ui-header-button-sidemenu',
        screenshotOnError: false,
    },
    {
        name: 'Ollama',
        url: 'http://localhost:11434',
        healthUrl: 'http://localhost:11434/api/tags',
        screenshotOnError: false,
    },
];

interface ServiceStatus {
    name: string;
    url: string;
    healthy: boolean;
    status?: string;
    version?: string;
    error?: string;
    screenshot?: string;
    checkedAt: string;
    responseTime?: number;
}

/**
 * Check service health via API endpoint
 */
async function checkHealthEndpoint(url: string): Promise<{
    healthy: boolean;
    status?: any;
    responseTime: number;
}> {
    const startTime = Date.now();
    
    try {
        const response = await fetch(url, {
            method: 'GET',
            timeout: 5000,
        });
        
        const responseTime = Date.now() - startTime;
        
        if (!response.ok) {
            return { healthy: false, responseTime };
        }
        
        try {
            const status = await response.json();
            return { healthy: true, status, responseTime };
        } catch {
            // If not JSON, just check if response is OK
            return { healthy: true, responseTime };
        }
    } catch (error) {
        return { 
            healthy: false, 
            responseTime: Date.now() - startTime 
        };
    }
}

/**
 * Scrape status information from service UI
 */
async function scrapeServiceStatus(
    service: ServiceConfig,
): Promise<{
    status?: string;
    version?: string;
    error?: string;
}> {
    if (!service.statusSelector && !service.versionSelector) {
        return {};
    }

    const elements = [];
    if (service.statusSelector) {
        elements.push({
            selector: service.statusSelector,
            property: 'innerText',
        });
    }
    if (service.versionSelector) {
        elements.push({
            selector: service.versionSelector,
            property: 'innerText',
        });
    }

    const body = {
        url: service.url,
        elements,
        gotoOptions: {
            waitUntil: 'networkidle2' as const,
            timeout: 15000,
        },
        waitFor: 1000,
    };

    const headers: HeadersInit = {
        'Content-Type': 'application/json',
    };

    if (BROWSERLESS_TOKEN) {
        headers['Authorization'] = `Bearer ${BROWSERLESS_TOKEN}`;
    }

    try {
        const response = await fetch(`${BROWSERLESS_URL}/chrome/scrape`, {
            method: 'POST',
            headers,
            body: JSON.stringify(body),
        });

        if (!response.ok) {
            const error = await response.text();
            return { error: `Scraping failed: ${error}` };
        }

        const results = await response.json();
        
        return {
            status: service.statusSelector ? results[0]?.innerText : undefined,
            version: service.versionSelector ? results[results.length - 1]?.innerText : undefined,
        };
    } catch (error) {
        return { error: error.message };
    }
}

/**
 * Take a screenshot of the service
 */
async function takeServiceScreenshot(
    service: ServiceConfig,
    outputDir: string,
): Promise<string | undefined> {
    const body = {
        url: service.url,
        options: {
            fullPage: false,
            type: 'png' as const,
            viewport: {
                width: 1280,
                height: 720,
                deviceScaleFactor: 1,
            },
        },
        gotoOptions: {
            waitUntil: 'networkidle2' as const,
            timeout: 15000,
        },
        waitFor: 1000,
    };

    const headers: HeadersInit = {
        'Content-Type': 'application/json',
    };

    if (BROWSERLESS_TOKEN) {
        headers['Authorization'] = `Bearer ${BROWSERLESS_TOKEN}`;
    }

    try {
        const response = await fetch(`${BROWSERLESS_URL}/chrome/screenshot`, {
            method: 'POST',
            headers,
            body: JSON.stringify(body),
        });

        if (!response.ok) {
            console.error(`Screenshot failed for ${service.name}`);
            return undefined;
        }

        const screenshot = Buffer.from(await response.arrayBuffer());
        const filename = `${service.name.toLowerCase()}-error-${Date.now()}.png`;
        const filepath = path.join(outputDir, filename);
        
        await fs.writeFile(filepath, screenshot);
        return filename;
    } catch (error) {
        console.error(`Screenshot error for ${service.name}:`, error.message);
        return undefined;
    }
}

/**
 * Monitor a single service
 */
async function monitorService(
    service: ServiceConfig,
    outputDir: string,
): Promise<ServiceStatus> {
    const status: ServiceStatus = {
        name: service.name,
        url: service.url,
        healthy: false,
        checkedAt: new Date().toISOString(),
    };

    // Check health endpoint if available
    if (service.healthUrl) {
        const health = await checkHealthEndpoint(service.healthUrl);
        status.healthy = health.healthy;
        status.responseTime = health.responseTime;
        
        if (!health.healthy) {
            status.error = 'Health check failed';
        }
    }

    // Scrape status from UI if selectors provided
    if (service.statusSelector || service.versionSelector) {
        const scraped = await scrapeServiceStatus(service);
        
        if (scraped.error) {
            status.error = scraped.error;
            status.healthy = false;
        } else {
            status.status = scraped.status;
            status.version = scraped.version;
            
            // Check expected status if provided
            if (service.expectedStatus && status.status) {
                if (!status.status.toLowerCase().includes(service.expectedStatus.toLowerCase())) {
                    status.healthy = false;
                    status.error = `Unexpected status: ${status.status}`;
                } else {
                    status.healthy = true;
                }
            }
        }
    }

    // If no health check or scraping, just check if URL is accessible
    if (!service.healthUrl && !service.statusSelector) {
        const health = await checkHealthEndpoint(service.url);
        status.healthy = health.healthy;
        status.responseTime = health.responseTime;
    }

    // Take screenshot if service is unhealthy and configured to do so
    if (!status.healthy && service.screenshotOnError) {
        status.screenshot = await takeServiceScreenshot(service, outputDir);
    }

    return status;
}

/**
 * Main monitoring function
 */
async function monitorServices() {
    console.log('🔍 Starting service monitoring...\n');

    // Create output directory for screenshots
    const outputDir = path.join(process.cwd(), 'service-monitoring');
    await fs.mkdir(outputDir, { recursive: true });

    const results: ServiceStatus[] = [];
    let healthyCount = 0;
    let unhealthyCount = 0;

    for (const service of SERVICES) {
        console.log(`🔍 Checking ${service.name}...`);
        
        const status = await monitorService(service, outputDir);
        results.push(status);

        if (status.healthy) {
            console.log(`✅ ${service.name} is healthy`);
            if (status.status) console.log(`   Status: ${status.status}`);
            if (status.version) console.log(`   Version: ${status.version}`);
            if (status.responseTime) console.log(`   Response time: ${status.responseTime}ms`);
            healthyCount++;
        } else {
            console.log(`❌ ${service.name} is unhealthy`);
            if (status.error) console.log(`   Error: ${status.error}`);
            if (status.screenshot) console.log(`   Screenshot: ${status.screenshot}`);
            unhealthyCount++;
        }
        console.log();
    }

    // Summary
    console.log('📊 Monitoring Summary:');
    console.log(`✅ Healthy services: ${healthyCount}`);
    console.log(`❌ Unhealthy services: ${unhealthyCount}`);

    // Save report
    const report = {
        monitoredAt: new Date().toISOString(),
        totalServices: SERVICES.length,
        healthyCount,
        unhealthyCount,
        services: results,
    };

    await fs.writeFile(
        path.join(outputDir, 'monitoring-report.json'),
        JSON.stringify(report, null, 2),
    );

    console.log(`\n📄 Report saved to: ${path.join(outputDir, 'monitoring-report.json')}`);

    // Return summary for further processing (e.g., sending alerts)
    return {
        healthy: unhealthyCount === 0,
        report,
    };
}

// Run if executed directly
if (require.main === module) {
    monitorServices()
        .then(result => {
            if (!result.healthy) {
                console.error('\n⚠️  Some services are unhealthy!');
                process.exit(1);
            }
        })
        .catch(error => {
            console.error('Monitoring failed:', error);
            process.exit(1);
        });
}

export { monitorServices, monitorService, ServiceConfig, ServiceStatus };