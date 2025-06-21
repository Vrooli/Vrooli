# API Response Fixtures

This directory contains fixtures for API responses and MSW (Mock Service Worker) handlers. These fixtures provide realistic API responses for testing components without requiring a live backend.

## Purpose

API response fixtures provide:
- **Realistic API responses** that match actual backend behavior
- **MSW handler generation** for comprehensive network mocking
- **Error scenario simulation** for robust error handling tests
- **Response variation testing** for different data states
- **Network condition simulation** including delays and failures

## Architecture

API response fixtures follow this pattern:

```typescript
export interface APIResponseFactory<TResponse> {
    createSuccessResponse(data: TResponse, metadata?: ResponseMetadata): APIResponse<TResponse>;
    createErrorResponse(error: APIError): APIErrorResponse;
    createLoadingResponse(delay?: number): Promise<APIResponse<TResponse>>;
    
    // MSW handler generation
    createMSWHandlers(): MSWHandlers;
    createCustomHandler(config: HandlerConfig): RestHandler;
}
```

## Response Types

### 1. Success Responses
Standard successful API responses:
- **Create responses**: 201 status with created resource
- **Read responses**: 200 status with requested data
- **Update responses**: 200 status with updated resource
- **Delete responses**: 204 status with no content

### 2. Error Responses
Comprehensive error scenarios:
- **Validation errors**: 400 status with field-level errors
- **Authentication errors**: 401 status with auth requirements
- **Permission errors**: 403 status with access restrictions
- **Not found errors**: 404 status with helpful messages
- **Server errors**: 500 status with error details

### 3. Pagination Responses
Paginated data responses:
- **List responses**: With pagination metadata
- **Search responses**: With result counts and filters
- **Infinite scroll**: With cursor-based pagination

### 4. Real-time Responses
WebSocket and SSE responses:
- **Connection events**: Connection established/lost
- **Data updates**: Real-time data changes
- **Progress events**: Long-running operation updates

## Bookmark Response Examples

### Success Response Structure
```typescript
interface BookmarkAPIResponse {
    data: Bookmark;
    meta: {
        timestamp: string;
        requestId: string;
        version: string;
    };
    links?: {
        self: string;
        list: string;
        target: string;
    };
}
```

### Error Response Structure
```typescript
interface BookmarkAPIError {
    error: {
        code: string;
        message: string;
        details?: {
            field?: string;
            value?: any;
            constraint?: string;
        };
    };
    meta: {
        timestamp: string;
        requestId: string;
        path: string;
    };
}
```

## MSW Integration

### Handler Generation
```typescript
export class BookmarkMSWFactory {
    createHandlers(): RestHandler[] {
        return [
            // Create bookmark
            rest.post('/api/bookmark', async (req, res, ctx) => {
                const body = await req.json();
                
                // Simulate validation
                const validation = this.validateCreateInput(body);
                if (!validation.valid) {
                    return res(
                        ctx.status(400),
                        ctx.json(this.createValidationErrorResponse(validation.errors))
                    );
                }
                
                // Simulate creation
                const bookmark = this.createBookmarkFromInput(body);
                
                return res(
                    ctx.status(201),
                    ctx.json(this.createSuccessResponse(bookmark))
                );
            }),
            
            // Get bookmark
            rest.get('/api/bookmark/:id', (req, res, ctx) => {
                const { id } = req.params;
                
                // Simulate not found
                if (id === 'not-found') {
                    return res(
                        ctx.status(404),
                        ctx.json(this.createNotFoundErrorResponse(id))
                    );
                }
                
                const bookmark = this.createBookmarkWithId(id as string);
                
                return res(
                    ctx.status(200),
                    ctx.json(this.createSuccessResponse(bookmark))
                );
            }),
            
            // Update bookmark
            rest.put('/api/bookmark/:id', async (req, res, ctx) => {
                const { id } = req.params;
                const body = await req.json();
                
                const bookmark = this.createUpdatedBookmark(id as string, body);
                
                return res(
                    ctx.status(200),
                    ctx.json(this.createSuccessResponse(bookmark))
                );
            }),
            
            // Delete bookmark
            rest.delete('/api/bookmark/:id', (req, res, ctx) => {
                return res(ctx.status(204));
            })
        ];
    }
}
```

### Dynamic Response Configuration
```typescript
export class DynamicResponseFactory {
    private responseConfig: Map<string, ResponseConfig> = new Map();
    
    configureEndpoint(endpoint: string, config: ResponseConfig): void {
        this.responseConfig.set(endpoint, config);
    }
    
    createHandler(endpoint: string): RestHandler {
        const config = this.responseConfig.get(endpoint);
        if (!config) {
            throw new Error(`No configuration found for endpoint: ${endpoint}`);
        }
        
        return rest[config.method](endpoint, async (req, res, ctx) => {
            // Apply delay if configured
            if (config.delay) {
                await new Promise(resolve => setTimeout(resolve, config.delay));
            }
            
            // Return error if configured
            if (config.shouldError) {
                return res(
                    ctx.status(config.errorStatus || 500),
                    ctx.json(config.errorResponse)
                );
            }
            
            // Return success response
            return res(
                ctx.status(config.successStatus || 200),
                ctx.json(config.successResponse)
            );
        });
    }
}
```

## Error Simulation

### Network Errors
```typescript
export class NetworkErrorSimulator {
    createNetworkErrorHandlers(): RestHandler[] {
        return [
            // Connection timeout
            rest.post('/api/bookmark', (req, res, ctx) => {
                return res(ctx.delay('infinite'));
            }),
            
            // Network failure
            rest.post('/api/bookmark', (req, res, ctx) => {
                return res.networkError('Network connection failed');
            }),
            
            // Slow network
            rest.get('/api/bookmark/:id', (req, res, ctx) => {
                return res(
                    ctx.delay(5000), // 5 second delay
                    ctx.status(200),
                    ctx.json(this.createMockBookmark())
                );
            })
        ];
    }
}
```

### Server Errors
```typescript
export class ServerErrorSimulator {
    createServerErrorHandlers(): RestHandler[] {
        return [
            // Internal server error
            rest.post('/api/bookmark', (req, res, ctx) => {
                return res(
                    ctx.status(500),
                    ctx.json({
                        error: {
                            code: 'INTERNAL_SERVER_ERROR',
                            message: 'An unexpected error occurred',
                            details: {
                                errorId: 'ERR_' + Date.now(),
                                timestamp: new Date().toISOString()
                            }
                        }
                    })
                );
            }),
            
            // Database error
            rest.post('/api/bookmark', (req, res, ctx) => {
                return res(
                    ctx.status(503),
                    ctx.json({
                        error: {
                            code: 'DATABASE_UNAVAILABLE',
                            message: 'Database is temporarily unavailable',
                            retryAfter: 30
                        }
                    })
                );
            })
        ];
    }
}
```

## Response Validation

### Schema Validation
```typescript
export class ResponseValidator {
    validateBookmarkResponse(response: any): ValidationResult {
        const errors: string[] = [];
        
        // Validate required fields
        if (!response.data) {
            errors.push('Response must contain data field');
        }
        
        if (!response.data?.id) {
            errors.push('Bookmark data must contain id field');
        }
        
        if (!response.data?.__typename) {
            errors.push('Bookmark data must contain __typename field');
        }
        
        // Validate data types
        if (response.data?.createdAt && !this.isValidDate(response.data.createdAt)) {
            errors.push('createdAt must be a valid ISO date string');
        }
        
        return {
            valid: errors.length === 0,
            errors
        };
    }
    
    private isValidDate(dateString: string): boolean {
        const date = new Date(dateString);
        return !isNaN(date.getTime());
    }
}
```

### Type Safety
```typescript
export class TypeSafeResponseFactory {
    createTypedResponse<T>(
        data: T,
        schema: JSONSchema7
    ): APIResponse<T> {
        // Validate against schema in development
        if (process.env.NODE_ENV === 'development') {
            const validation = this.validateAgainstSchema(data, schema);
            if (!validation.valid) {
                console.warn('Response data does not match schema:', validation.errors);
            }
        }
        
        return {
            data,
            meta: {
                timestamp: new Date().toISOString(),
                requestId: this.generateRequestId(),
                version: '1.0'
            }
        };
    }
}
```

## Usage Examples

### Component Testing
```typescript
describe('BookmarkButton Component', () => {
    beforeEach(() => {
        // Setup MSW with bookmark handlers
        server.use(...bookmarkResponseFixtures.createSuccessHandlers());
    });
    
    it('should create bookmark successfully', async () => {
        render(<BookmarkButton objectId="test-123" objectType="Resource" />);
        
        await user.click(screen.getByRole('button', { name: 'Bookmark' }));
        
        // Wait for success state
        await waitFor(() => {
            expect(screen.getByText('Bookmarked!')).toBeInTheDocument();
        });
    });
    
    it('should handle validation errors', async () => {
        // Configure error response
        server.use(
            rest.post('/api/bookmark', (req, res, ctx) => {
                return res(
                    ctx.status(400),
                    ctx.json(bookmarkResponseFixtures.createValidationErrorResponse({
                        forConnect: 'Object ID is required'
                    }))
                );
            })
        );
        
        render(<BookmarkButton objectId="" objectType="Resource" />);
        
        await user.click(screen.getByRole('button', { name: 'Bookmark' }));
        
        await waitFor(() => {
            expect(screen.getByText('Object ID is required')).toBeInTheDocument();
        });
    });
});
```

### Integration Testing
```typescript
describe('Bookmark API Integration', () => {
    it('should handle complete bookmark flow', async () => {
        // Test actual API integration
        const createResponse = await apiClient.post('/api/bookmark', {
            bookmarkFor: 'Resource',
            forConnect: 'test-resource-123'
        });
        
        expect(createResponse.status).toBe(201);
        expect(createResponse.data.id).toBeDefined();
        
        // Verify response format
        const validation = responseValidator.validateBookmarkResponse(createResponse);
        expect(validation.valid).toBe(true);
    });
});
```

### Error Handling Testing
```typescript
describe('Error Handling', () => {
    it('should handle network errors gracefully', async () => {
        // Setup network error
        server.use(...networkErrorSimulator.createNetworkErrorHandlers());
        
        render(<BookmarkForm />);
        
        await user.click(screen.getByRole('button', { name: 'Save' }));
        
        await waitFor(() => {
            expect(screen.getByText('Network error. Please try again.')).toBeInTheDocument();
        });
    });
});
```

## Performance Optimization

### Response Caching
```typescript
export class ResponseCache {
    private cache: Map<string, CachedResponse> = new Map();
    
    getCachedResponse(key: string): CachedResponse | null {
        const cached = this.cache.get(key);
        if (!cached) return null;
        
        // Check expiration
        if (Date.now() > cached.expiresAt) {
            this.cache.delete(key);
            return null;
        }
        
        return cached;
    }
    
    setCachedResponse(key: string, response: any, ttl: number = 5000): void {
        this.cache.set(key, {
            data: response,
            expiresAt: Date.now() + ttl
        });
    }
}
```

### Lazy Response Generation
```typescript
export class LazyResponseFactory {
    private responseGenerators: Map<string, () => any> = new Map();
    
    registerGenerator(key: string, generator: () => any): void {
        this.responseGenerators.set(key, generator);
    }
    
    generateResponse(key: string): any {
        const generator = this.responseGenerators.get(key);
        if (!generator) {
            throw new Error(`No generator found for key: ${key}`);
        }
        
        return generator();
    }
}
```

## Best Practices

### DO's ✅
- Create realistic response data that matches production
- Include proper HTTP status codes and headers
- Test both success and error scenarios
- Validate response formats against schemas
- Use MSW for comprehensive network mocking
- Include response timing simulation
- Test edge cases and error conditions

### DON'Ts ❌
- Use fake data that doesn't match real API responses
- Skip error scenario testing
- Hardcode response data without variation
- Ignore HTTP semantics and status codes
- Mock at too low a level (prefer MSW over fetch mocks)
- Forget to test network conditions
- Create responses without proper typing

## Troubleshooting

### Common Issues
1. **Response Format Mismatches**: Ensure test responses match actual API format
2. **MSW Handler Conflicts**: Check handler order and specificity
3. **Timing Issues**: Use proper async/await and waitFor patterns
4. **Type Safety**: Ensure response types match expected interfaces
5. **Error Simulation**: Verify error responses trigger correct UI states

### Debug Strategies
- Log MSW requests and responses in development
- Validate response schemas in tests
- Use browser dev tools to inspect network traffic
- Test against real API endpoints when possible
- Monitor response timing and performance

This comprehensive API response testing approach ensures components handle all network scenarios correctly and provides confidence in the application's network resilience.