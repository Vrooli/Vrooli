# Local Resources System

This system allows Vrooli to discover and interact with locally running services like AI models, automation platforms, browser agents, and storage services.

## How It Works

### 1. Resource Registration
Resources are registered at module load time using the `@RegisterResource` decorator:

```typescript
@RegisterResource
export class OllamaResource extends LocalResourceProvider<OllamaConfig> {
    readonly id = 'ollama';
    readonly category = ResourceCategory.AI;
    // ... implementation
}
```

### 2. Configuration
Resources are configured via `.vrooli/resources.local.json`:

```json
{
  "enabled": true,
  "services": {
    "ai": {
      "ollama": {
        "enabled": true,
        "baseUrl": "http://localhost:11434"
      }
    }
  }
}
```

### 3. Discovery Process
1. Registry loads configuration
2. Instantiates registered resource classes
3. Runs discovery to find running services
4. Starts health monitoring for discovered services

### 4. Health Monitoring
- Each resource performs periodic health checks
- Resources emit events on status changes
- Registry maintains overall system state

## Health Check Integration

Add to your health check endpoint:

```typescript
import { LocalResourceRegistry } from '@services/resources';

// In your health check endpoint
app.get('/health/resources', async (req, res) => {
    const registry = LocalResourceRegistry.getInstance();
    const healthCheck = registry.getHealthCheck();
    
    // Return appropriate HTTP status based on health
    const statusCode = healthCheck.status === 'Down' ? 503 : 
                      healthCheck.status === 'Degraded' ? 207 : 200;
    
    res.status(statusCode).json(healthCheck);
});
```

## Health Check Response

The health check provides comprehensive information:

```json
{
  "status": "Degraded",
  "message": "2 of 3 enabled services are available",
  "timestamp": "2024-01-20T10:00:00Z",
  "stats": {
    "totalSupported": 27,      // All resources we aim to support
    "totalRegistered": 5,      // Resources we've implemented
    "totalEnabled": 3,         // Resources enabled in config
    "totalActive": 2           // Resources verified available
  },
  "missingImplementations": [  // Supported but not implemented
    {
      "id": "localai",
      "name": "LocalAI",
      "category": "ai"
    }
  ],
  "unavailableServices": [     // Enabled but not available
    {
      "id": "n8n",
      "name": "n8n",
      "category": "automation",
      "reason": "Service not found"
    }
  ],
  "categories": {              // Breakdown by category
    "ai": {
      "supported": 6,
      "registered": 2,
      "enabled": 1,
      "active": 1
    }
  },
  "resources": [...]           // Detailed status of all resources
}
```

## Health Status Logic

- **Operational**: All enabled services are available (or no services enabled)
- **Degraded**: Some enabled services are unavailable
- **Down**: All enabled services are unavailable

## Creating a New Resource

1. Create a class extending `LocalResourceProvider`:

```typescript
import { LocalResourceProvider, RegisterResource, ResourceCategory } from '@services/resources';

@RegisterResource
export class MyResource extends LocalResourceProvider<MyConfig> {
    readonly id = 'myresource';
    readonly category = ResourceCategory.AI;
    readonly displayName = 'My Resource';
    readonly description = 'Description of my resource';
    readonly isSupported = true;
    
    protected async performDiscovery(): Promise<boolean> {
        // Check if service is running
        try {
            const result = await this.httpClient!.makeRequest({
                url: `${this.config.baseUrl}/health`,
                method: "GET",
            });
            return result.success;
        } catch {
            return false;
        }
    }
    
    protected async performHealthCheck(): Promise<HealthCheckResult> {
        // Verify service is healthy
        const result = await this.httpClient!.makeRequest({
            url: `${this.config.baseUrl}/health`,
            method: "GET",
        });
        return {
            healthy: result.success,
            message: result.success ? 'Service is healthy' : 'Service is unhealthy',
            timestamp: new Date(),
        };
    }
}
```

2. Add to `SUPPORTED_RESOURCES` in `constants.ts`
3. Configure in `.vrooli/resources.local.json`
4. The resource will be automatically discovered and monitored

## Best Practices

1. **Discovery**: Should be fast and non-blocking
2. **Health Checks**: Include authentication validation
3. **Error Handling**: Log errors but don't throw from health checks
4. **Events**: Emit appropriate events on status changes
5. **Timeouts**: Use reasonable timeouts for network requests