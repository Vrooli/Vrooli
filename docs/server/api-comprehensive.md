# Vrooli API Comprehensive Guide

This document provides complete coverage of Vrooli's API architecture, development patterns, and implementation guidelines for all three communication protocols.

## Table of Contents

- [Overview](#overview)
- [API Architecture](#api-architecture)
- [REST API](#rest-api)
- [WebSocket API](#websocket-api) 
- [MCP (Model Context Protocol) Server](#mcp-model-context-protocol-server)
- [Authentication & Authorization](#authentication--authorization)
- [Rate Limiting & Security](#rate-limiting--security)
- [Development Patterns](#development-patterns)
- [Performance & Scalability](#performance--scalability)
- [Monitoring & Health Checks](#monitoring--health-checks)
- [Error Handling](#error-handling)
- [Testing APIs](#testing-apis)
- [Configuration](#configuration)
- [Best Practices](#best-practices)

## Overview

Vrooli's API layer provides three distinct communication protocols designed for different use cases and integration patterns:

```mermaid
graph TB
    subgraph "Client Applications"
        PWA[Progressive Web App]
        Mobile[Mobile Apps]
        Extension[Browser Extension]
        Desktop[Electron Desktop]
        External[External Integrations]
        AI[AI Agents & MCP Clients]
    end
    
    subgraph "API Gateway Layer"
        REST[REST API<br/>HTTP/HTTPS<br/>CRUD Operations]
        WS[WebSocket API<br/>Real-time Communication<br/>Live Updates]
        MCP[MCP Server<br/>Model Context Protocol<br/>AI Tool Integration]
    end
    
    subgraph "Core Services"
        Auth[Authentication Service]
        RateLimit[Rate Limiting]
        Validation[Request Validation]
        CORS[CORS Security]
        Health[Health Monitoring]
    end
    
    PWA --> REST
    PWA --> WS
    Mobile --> REST
    Mobile --> WS
    Extension --> WS
    Desktop --> REST
    Desktop --> WS
    External --> REST
    AI --> MCP
    
    REST --> Auth
    WS --> Auth
    MCP --> Auth
    
    REST --> RateLimit
    WS --> RateLimit
    MCP --> RateLimit
    
    REST --> Validation
    WS --> Validation
    MCP --> Validation
    
    REST --> CORS
    WS --> CORS
    MCP --> CORS
```

### Key Principles

- **Type Safety**: Full TypeScript integration across all protocols
- **Horizontal Scalability**: Redis-based clustering and stateless design
- **Security by Design**: Multiple authentication methods and comprehensive validation
- **Event-Driven Architecture**: Real-time updates and background processing
- **Developer Experience**: Consistent patterns and comprehensive documentation

## API Architecture

### Middleware Processing Pipeline

```mermaid
graph LR
    Request[HTTP Request] --> Parse[Request Parser]
    Parse --> Cookie[Cookie Parser]
    Cookie --> Auth[Authentication]
    Auth --> Origin[Origin Validation]
    Origin --> Rate[Rate Limiting]
    Rate --> CORS[CORS Validation]
    CORS --> JSON[JSON Parser]
    JSON --> Upload[File Upload Handler]
    Upload --> Routes[Route Handler]
    Routes --> Response[HTTP Response]
    
    subgraph "Error Handling"
        Routes --> Error[Error Handler]
        Error --> ErrorResp[Error Response]
    end
```

### Core Components Structure

```
packages/server/src/
├── endpoints/              # REST API endpoints
│   ├── rest.ts            # Main REST API setup & routing
│   ├── logic/             # Business logic handlers (60+ resources)
│   ├── generated/         # Auto-generated GraphQL-style selectors
│   └── helpers/           # Endpoint utility functions
├── sockets/               # WebSocket implementation
│   ├── io.ts             # Socket.IO server with Redis clustering
│   └── rooms/            # Room-based event handlers
├── services/mcp/          # Model Context Protocol implementation
│   ├── server.ts         # MCP server application class
│   ├── transport.ts      # SSE transport management
│   ├── registry.ts       # Tool registry and execution
│   └── context.ts        # Request context management
├── auth/                  # Authentication & session management
├── middleware/            # Express middleware stack
└── utils/                 # Shared utilities and helpers
```

## REST API

### Endpoint Definition Pattern

Vrooli uses a tuple-based pattern for consistent endpoint definitions:

```typescript
type EndpointTuple = readonly [
    EndpointDef,                    // Route definition (path + method)
    ApiEndpoint<never, unknown>,    // Business logic handler
    PartialApiInfo,                 // Data selection schema
    UploadConfig?                   // File upload configuration
];

type EndpointDef = {
    endpoint: string;              // e.g., "/user/:id"
    method: "GET" | "POST" | "PUT" | "DELETE";
};
```

### Creating New Endpoints

#### 1. Define Route Configuration

```typescript
// In packages/server/src/endpoints/logic/user.ts
const endpointsUser = {
    findOne: { endpoint: "/user/:id", method: "GET" } as const,
    createOne: { endpoint: "/user", method: "POST" } as const,
    updateOne: { endpoint: "/user/:id", method: "PUT" } as const,
    deleteOne: { endpoint: "/user/:id", method: "DELETE" } as const,
} as const;
```

#### 2. Implement Business Logic Handler

```typescript
export const user: EndpointsUser = {
    findOne: async ({ input }, { req }, info) => {
        // Rate limiting
        await RequestService.get().rateLimit({ maxUser: 1000, req });
        
        // Authorization check
        RequestService.assertRequestFrom(req, { hasReadPublicPermissions: true });
        
        // Input validation
        const { id } = input;
        if (!id || typeof id !== 'string') {
            throw new CustomError("0001", "InvalidId");
        }
        
        // Business logic with helper
        return readOneHelper({ info, input, objectType: "User", req });
    },
    
    createOne: async ({ input }, { req }, info) => {
        // Authentication required for creation
        RequestService.assertRequestFrom(req, { isLoggedIn: true });
        
        // Input validation with schema
        const validatedInput = CreateUserSchema.parse(input);
        
        // Business logic
        return await UserService.create(validatedInput, info);
    },
};
```

#### 3. Define Data Selection Schema

```typescript
// In packages/server/src/endpoints/generated/user.ts
export const user_findOne = {
    id: true,
    email: true,
    name: true,
    profile: {
        bio: true,
        avatar: true,
    },
    createdAt: true,
} as const;
```

#### 4. Register Endpoint

```typescript
// In packages/server/src/endpoints/rest.ts
const routes = setupRoutes([
    [endpointsUser.findOne, Logic.user.findOne, Select.user_findOne],
    [endpointsUser.createOne, Logic.user.createOne, Select.user_createOne],
    // ... other endpoints
]);
```

### File Upload Configuration

```typescript
const userImagesConfig: UploadConfig = {
    acceptsFiles: true,
    fields: [{
        fieldName: "profileImage",
        allowedExtensions: ["png", "jpg", "jpeg", "webp"],
        maxFileSize: 5 * 1024 * 1024, // 5MB
        maxFiles: 1,
        imageSizes: [
            { width: 256, height: 256 },
            { width: 128, height: 128 },
        ],
        fileNameBase: (_, currentUser) => `${currentUser.id}-profile`,
    }],
};

// Register with upload support
[endpointsUser.updateProfile, Logic.user.updateProfile, Select.user_updateProfile, userImagesConfig]
```

### Request/Response Flow

```typescript
// Example request processing in setupRoutes()
async function handleEndpointHelper(req: Request, res: Response) {
    // Parse input based on method
    let input: Record<string, unknown> = {};
    if (methodInLowerCase === "get") {
        input = Object.assign({}, req.params, parseInput(req.query));
    } else {
        input = Object.assign({}, req.params, req.body);
    }
    
    // Process file uploads if configured
    if (uploadConfig?.acceptsFiles && req.files?.length > 0) {
        const userData = SessionService.getUser(req);
        if (!userData) {
            return ResponseService.sendError(res, 
                { trace: "0509", code: "NotLoggedIn" }, 
                HttpStatus.Unauthorized
            );
        }
        const fileNames = await processAndStoreFiles(req.files, input, userData, uploadConfig);
        // Add processed files to input
        Object.assign(input, fileNames);
    }
    
    // Execute endpoint handler
    await handleEndpoint(endpointHandler, selection, input, req, res);
}
```

## WebSocket API

### Architecture Features

- **Socket.IO v4+**: Production-ready WebSocket implementation with fallbacks
- **Redis Adapter**: Horizontal scaling with `@socket.io/redis-adapter`  
- **Room-based Organization**: Segregated handlers by feature (chat, runs, users)
- **Connection State Recovery**: Automatic reconnection and state synchronization

### Scaling Architecture

```mermaid
graph TB
    subgraph "Load Balancer"
        LB[Nginx/HAProxy]
    end
    
    subgraph "Server Instance 1"
        S1[SocketService 1]
        Local1[Local Socket Maps]
    end
    
    subgraph "Server Instance 2" 
        S2[SocketService 2]
        Local2[Local Socket Maps]
    end
    
    subgraph "Redis Cluster"
        Adapter[Redis Adapter]
        PubSub[Pub/Sub Channels]
        Rooms[Room State]
    end
    
    LB --> S1
    LB --> S2
    
    S1 <--> Adapter
    S2 <--> Adapter
    Adapter <--> PubSub
    Adapter <--> Rooms
    
    Client1[Client 1] --> LB
    Client2[Client 2] --> LB
    
    S1 -.->|Cross-instance Events| S2
    S2 -.->|Cross-instance Events| S1
```

### Connection Management

```typescript
export class SocketService {
    // Instance-local maps for direct connections
    public userSockets: Map<string, Set<string>>;
    public sessionSockets: Map<string, Set<string>>;
    public io: Server; // Socket.IO server instance
    
    /**
     * Adds a connected socket to local tracking maps
     */
    public addSocket(socket: Socket): void {
        const user = SessionService.getUser(socket);
        const userId = user?.id;
        const sessionId = user?.session.id;

        if (userId && sessionId) {
            if (!this.userSockets.has(userId)) {
                this.userSockets.set(userId, new Set<string>());
            }
            this.userSockets.get(userId)?.add(socket.id);

            if (!this.sessionSockets.has(sessionId)) {
                this.sessionSockets.set(sessionId, new Set<string>());
            }
            this.sessionSockets.get(sessionId)?.add(socket.id);
        }
    }
    
    /**
     * Emits event to all clients in a room across all instances
     */
    public emitSocketEvent<T extends EmitSocketEvent>(
        event: T, 
        roomId: string, 
        payload: SocketEventPayloads[T]
    ): void {
        this.io.in(roomId).fetchSockets().then((sockets) => {
            for (const socket of sockets) {
                const session = (socket as { session?: SessionData }).session;
                if (!session) {
                    socket.emit(event, payload);
                    continue;
                }
                const isExpired = AuthTokensService.isAccessTokenExpired(session);
                if (isExpired) {
                    socket.disconnect();
                } else {
                    socket.emit(event, payload);
                }
            }
        });
    }
    
    /**
     * Closes all sockets for a user across all instances
     */
    public closeUserSockets(userId: string): void {
        this.io.serverSideEmit("internal:disconnect_user_sockets", userId);
    }
}
```

### Room Handlers

#### Chat Rooms (`sockets/rooms/chat.ts`)
- Join/leave chat conversations
- Real-time message delivery  
- Typing indicators
- Response cancellation for AI interactions

#### Run Rooms (`sockets/rooms/run.ts`)
- Task execution monitoring
- Progress updates
- Error notifications

#### User Rooms (`sockets/rooms/user.ts`)
- Profile updates
- Notification delivery
- Session management

### Authentication & Security

```typescript
// WebSocket authentication during handshake
io.use(async (socket, next) => {
    try {
        const token = socket.handshake.auth.token || socket.handshake.headers.authorization;
        const session = await AuthTokensService.validateToken(token);
        
        if (!session) {
            return next(new Error('Authentication failed'));
        }
        
        // Attach session to socket
        (socket as any).session = session;
        next();
    } catch (error) {
        next(new Error('Authentication failed'));
    }
});
```

## MCP (Model Context Protocol) Server

### Architecture Overview

**Location**: `packages/server/src/services/mcp/`  
**Modes**: STDIO (local development) and SSE (Server-Sent Events for production)

The MCP server provides standardized AI model integration implementing the [Model Context Protocol specification](https://spec.modelcontextprotocol.io/).

### Server Modes

```mermaid
graph TB
    subgraph "STDIO Mode (Local Development)"
        STDIO[Standard Input/Output]
        LocalMCP[Local MCP Client]
        STDIO <--> LocalMCP
    end
    
    subgraph "SSE Mode (Production)"
        SSE[Server-Sent Events]
        HTTP[HTTP Message Endpoint]
        RemoteMCP[Remote MCP Client]
        
        RemoteMCP --> SSE
        RemoteMCP --> HTTP
    end
    
    subgraph "MCP Server Core"
        ToolRegistry[Tool Registry]
        Transport[Transport Manager]
        Context[Execution Context]
        Dynamic[Dynamic Tool Servers]
    end
    
    STDIO --> ToolRegistry
    SSE --> ToolRegistry
    HTTP --> ToolRegistry
    ToolRegistry --> Dynamic
```

### Implementation Architecture

```typescript
export class McpServerApp {
    private app: Express;
    private config: ServerConfig;
    private mcpServer: McpServer;
    private toolRegistry: ToolRegistry;
    private transportManager?: TransportManager;
    private dynamicServers: Map<string, McpServer> = new Map();
    private dynamicTransportManagers: Map<string, TransportManager> = new Map();

    constructor(config: ServerConfig, logger: Logger, app: Express) {
        this.config = config;
        this.toolRegistry = new ToolRegistry(logger);
        
        // Initialize MCP server with core capabilities
        this.mcpServer = new McpServer(
            this.config.serverInfo,
            { capabilities: { tools: {} } }
        );
        
        this.setupRequestHandlers();
    }
    
    private setupRequestHandlers(): void {
        // List tools handler
        this.mcpServer.setRequestHandler(ListToolsRequestSchema, async () => {
            const tools = this.toolRegistry.getBuiltInDefinitions();
            return { tools } as ServerResult;
        });

        // Call tool handler  
        this.mcpServer.setRequestHandler(CallToolRequestSchema, async (request) => {
            const { name, arguments: args } = request.params;
            return this.toolRegistry.execute(name as McpToolName, args) as Promise<ServerResult>;
        });
    }
}
```

### Endpoint Structure

```typescript
// Standard MCP endpoints
GET  /mcp/sse                        // SSE connection for built-in tools
POST /mcp/messages                   // Message handling for built-in tools  
GET  /mcp/health                     // Health check

// Dynamic tool endpoints (tool-specific isolation)
GET  /mcp/tool/:tool_id/sse         // SSE connection for specific tool
POST /mcp/tool/:tool_id/message     // Message handling for specific tool
```

### Tool Registry & Execution

```typescript
export class ToolRegistry {
    async execute<T extends McpToolName>(name: T, args: ToolArgsMap[T]): Promise<ToolResponse> {
        try {
            switch (name) {
                case McpRoutineToolName.FindRoutines:
                    return await BuiltInTools.findRoutines(args as FindRoutinesArgs);
                case McpRoutineToolName.RunRoutine:
                    return await BuiltInTools.runRoutine(args as RunRoutineArgs);
                default: {
                    const exhaustiveCheck: never = name;
                    return {
                        isError: true,
                        content: [{
                            type: "text",
                            text: `Error: Tool handler for '${exhaustiveCheck}' not found.`,
                        }],
                    };
                }
            }
        } catch (error) {
            return {
                isError: true,
                content: [{
                    type: "text", 
                    text: `Error executing tool '${name}': ${(error as Error).message}`,
                }],
            };
        }
    }
}
```

### Context Management

```typescript
// MCP HTTP context using AsyncLocalStorage
const asyncLocalStorage = new AsyncLocalStorage<McpContext>();

export function runWithMcpContext<T>(req: Request, res: Response, fn: () => Promise<T>): Promise<T> {
    return asyncLocalStorage.run({ req, res }, fn);
}

export function getCurrentMcpContext(): McpContext {
    const context = asyncLocalStorage.getStore();
    if (!context) {
        throw new Error("No MCP HTTP context is set. Ensure runWithMcpContext is used.");
    }
    return context;
}
```

## Authentication & Authorization

### Multi-Method Authentication Strategy

```mermaid
graph TB
    Request[Incoming Request] --> Check{Request Type?}
    Check -->|Webhook| Webhook[Webhook Auth<br/>Stripe, etc.]
    Check -->|Standard| Standard[Standard Auth Flow]
    
    Standard --> Origin{Safe Origin?}
    Origin -->|No + No API Key| Reject[403 Forbidden]  
    Origin -->|Yes or API Key| Auth{Auth Method?}
    
    Auth -->|JWT Cookie| JWT[JWT Token Validation]
    Auth -->|Bearer Token| Bearer[Bearer Token Validation]
    Auth -->|API Key| APIKey[API Key Validation]
    Auth -->|Environment| EnvKey[Environment Key Check]
    
    JWT --> Session[Session Creation]
    Bearer --> Session
    APIKey --> Permissions[Permission Check]
    EnvKey --> SystemAccess[System Access]
    
    Session --> Context[Request Context]
    Permissions --> Context
    SystemAccess --> Context
    Context --> Continue[Continue to Handler]
```

### Authentication Methods

#### 1. JWT Session Tokens
```typescript
interface SessionData {
    userId: string;
    sessionId: string;
    issuedAt: number;
    expiresAt: number;
    permissions: UserPermissions;
}

// JWT validation with automatic refresh
const session = await AuthTokensService.validateToken(token);
if (AuthTokensService.isAccessTokenExpired(session)) {
    // Attempt refresh or require re-authentication
    await AuthTokensService.refreshAccessToken(session);
}
```

#### 2. API Keys
```typescript
interface ApiKeyPermissions {
    [ApiKeyPermission.ReadPublic]: boolean;
    [ApiKeyPermission.ReadPrivate]: boolean;
    [ApiKeyPermission.WritePublic]: boolean;
    [ApiKeyPermission.WritePrivate]: boolean;
    [ApiKeyPermission.Admin]: boolean;
}

// API key validation
const { hasApiToken, permissions } = RequestService.getRequestPermissions(req);
if (!permissions[ApiKeyPermission.ReadPublic]) {
    throw new CustomError("0102", "InsufficientPermissions");
}
```

#### 3. Environment Keys
```typescript
// For development and system-to-system communication
const mcpApiKey = process.env.MCP_API_KEY;
if (req.headers['x-api-key'] === mcpApiKey) {
    // System access granted
    return { systemAccess: true };
}
```

### Authorization Patterns

```typescript
// Endpoint-level authorization
export const sensitiveOperation = async ({ input }, { req }, info) => {
    // Authentication check
    const userData = SessionService.getUser(req);
    if (!userData) {
        throw new CustomError("0100", "NotLoggedIn");
    }
    
    // Permission check
    if (!hasPermission(userData.id, "SENSITIVE_OPERATION")) {
        throw new CustomError("0101", "InsufficientPermissions");
    }
    
    // Resource-level authorization
    const resource = await getResource(input.resourceId);
    if (!canAccessResource(userData.id, resource)) {
        throw new CustomError("0102", "ResourceAccessDenied");
    }
    
    // Proceed with operation
    return await performOperation(input);
};
```

## Rate Limiting & Security

### Token Bucket Implementation

```typescript
interface RateLimitProps {
    maxApi?: number;    // API key limit (default: maxUser * 1000)
    maxIp?: number;     // IP limit (default: maxUser)  
    maxUser?: number;   // User limit (default: 250/day)
    window?: number;    // Time window in seconds (default: 86400)
}

// Rate limiting tiers
const rateLimitTiers = {
    apiToken: 250000,    // High limits for API access
    ipAddress: 250,      // Moderate limits for anonymous access
    userAccount: 250,    // Per-user limits
};
```

### Security Pipeline

```typescript
// Request validation pipeline
async function validateRequest(req: Request): Promise<RequestContext> {
    // 1. Origin validation
    if (!RequestService.isSafeOrigin(req) && !hasApiKey(req)) {
        throw new CustomError("0200", "UnsafeOrigin");
    }
    
    // 2. Authentication
    const session = await AuthTokensService.validateRequest(req);
    
    // 3. Rate limiting  
    await RequestService.get().rateLimit({ req, maxUser: 250 });
    
    // 4. Input validation
    const validatedInput = await validateInputSchema(req.body);
    
    return { session, validatedInput };
}
```

### CORS Configuration

```typescript
const corsOptions = {
    origin: (origin: string | undefined, callback: Function) => {
        const allowedOrigins = [
            'https://vrooli.com',
            'https://app.vrooli.com',
            ...(process.env.NODE_ENV === 'development' ? ['http://localhost:3000'] : []),
        ];
        
        if (!origin || allowedOrigins.includes(origin)) {
            callback(null, true);
        } else {
            callback(new Error('Not allowed by CORS'));
        }
    },
    credentials: true,
    methods: ['GET', 'POST', 'PUT', 'DELETE', 'OPTIONS'],
    allowedHeaders: ['Content-Type', 'Authorization', 'x-api-key'],
};
```

## Development Patterns

### Input Validation with Zod

```typescript
import { z } from 'zod';

const CreateUserSchema = z.object({
    email: z.string().email().max(255),
    name: z.string().min(1).max(100),
    age: z.number().int().min(13).max(120).optional(),
    preferences: z.object({
        theme: z.enum(['light', 'dark']),
        notifications: z.boolean(),
    }).optional(),
});

// Use in endpoint
export const createUser = async ({ input }, { req }, info) => {
    const validatedInput = CreateUserSchema.parse(input);
    return await UserService.create(validatedInput, info);
};
```

### Error Handling System

```typescript
// Custom error with trace IDs
export class CustomError extends Error {
    constructor(
        public trace: string,
        public code: string,
        public statusCode: number = 400,
        public details?: object
    ) {
        super(`${code}: ${trace}`);
        this.name = 'CustomError';
    }
    
    toServerError(): ServerError {
        return {
            trace: this.trace,
            code: this.code,
            details: this.details,
        };
    }
}

// Standardized error response
interface ServerError {
    trace: string;        // Internal tracking ID (e.g., "0200")
    code: string;         // Client error code (e.g., "ValidationFailed")
    details?: object;     // Additional context
}
```

### Context Pattern

```typescript
interface RequestContext {
    req: Request;           // Express request
    res: Response;          // Express response  
    userId?: string;        // Current user ID
    sessionId?: string;     // Current session ID
    permissions?: object;   // API key permissions
    language: string;       // User's language
    timeZone?: string;      // User's timezone
}

// Context creation
const context = ({ req, res }: { req: Request; res: Response }): RequestContext => {
    const userData = SessionService.getUser(req);
    const { permissions } = RequestService.getRequestPermissions(req);
    
    return {
        req,
        res,
        userId: userData?.id,
        sessionId: userData?.session.id,
        permissions,
        language: req.headers['accept-language'] || 'en',
        timeZone: req.headers['x-timezone'],
    };
};
```

### Query Parameter Parsing

```typescript
function parseInput(input: Record<string, unknown>): Record<string, unknown> {
    const parsed: Record<string, unknown> = {};
    Object.entries(input).forEach(([key, value]) => {
        try {
            // Handle JSON-encoded values like "true" -> true, "1" -> 1
            parsed[key] = decodeValue(JSON.parse(value as string));
        } catch (error) {
            parsed[key] = value;
        }
    });
    return parsed;
}

// Usage in GET endpoints
export const findUsers = async ({ input }, { req }, info) => {
    const page = Math.max(1, parseInt(input.page || '1', 10));
    const limit = Math.min(100, Math.max(1, parseInt(input.limit || '20', 10)));
    const active = input.active === 'true';
    
    return await UserService.findMany({ page, limit, active }, info);
};
```

## Performance & Scalability

### Caching Strategy

```mermaid
graph TB
    Request[API Request] --> L1[L1 Cache<br/>Local LRU<br/>~1ms]
    L1 -->|Miss| L2[L2 Cache<br/>Redis<br/>~5ms]
    L2 -->|Miss| L3[L3 Cache<br/>PostgreSQL<br/>~50ms]
    L3 -->|Miss| Compute[Compute Result]
    
    Compute --> Write[Write to All Layers]
    L2 --> Hit1[Cache Hit]
    L1 --> Hit2[Cache Hit]
    
    subgraph "Cache Invalidation"
        Event[Data Change Event] --> Invalidate[Invalidate Caches]
        Invalidate --> L1
        Invalidate --> L2
    end
```

### Horizontal Scaling

```typescript
// Redis-based scaling configuration
const redisAdapter = createAdapter(pubClient, subClient);
const io = new Server(server, {
    adapter: redisAdapter,
    cors: { origin: safeOrigins },
});

// Health check for scalability
interface HealthCheckResponse {
    status: "operational" | "degraded" | "down";
    services: {
        database: ServiceHealth;
        redis: ServiceHealth;  
        websocket: ServiceHealth;
        mcp: ServiceHealth;
    };
    metrics: {
        connectedClients: number;
        activeRooms: number;
        requestCount: number;
    };
}
```

### Connection Pooling

```typescript
// Database connection pooling with Prisma
const prisma = new PrismaClient({
    datasources: {
        db: {
            url: process.env.DATABASE_URL,
        },
    },
});

// Redis connection pooling
const redis = new IORedis(redisUrl, {
    retryDelayOnFailover: 100,
    enableReadyCheck: false,
    maxRetriesPerRequest: null,
});
```

## Monitoring & Health Checks

### Health Check Implementation

```typescript
// Main health check endpoint
app.get('/healthcheck', async (req, res) => {
    const healthService = HealthService.get();
    const healthStatus = await healthService.getOverallHealth();
    
    res.status(healthStatus.status === 'operational' ? 200 : 503).json(healthStatus);
});

// Individual service health checks
export class HealthService {
    async getDatabaseHealth(): Promise<ServiceHealth> {
        try {
            await DbProvider.get().$queryRaw`SELECT 1`;
            return { status: 'operational', responseTime: Date.now() - startTime };
        } catch (error) {
            return { status: 'down', error: error.message };
        }
    }
    
    async getRedisHealth(): Promise<ServiceHealth> {
        try {
            const cache = CacheService.get();
            await cache.ping();
            return { status: 'operational' };
        } catch (error) {
            return { status: 'down', error: error.message };
        }
    }
    
    async getSocketHealth(): Promise<ServiceHealth & SocketHealthDetails> {
        const socketService = SocketService.get();
        return {
            status: 'operational',
            connectedClients: socketService.io.engine.clientsCount,
            activeRooms: socketService.io.sockets.adapter.rooms?.size || 0,
            namespaces: socketService.io._nsps?.size || 1,
        };
    }
}
```

### Performance Monitoring

```typescript
// Response time middleware
app.use((req, res, next) => {
    const startTime = Date.now();
    
    res.on('finish', () => {
        const duration = Date.now() - startTime;
        logger.info('Request completed', {
            method: req.method,
            url: req.url,
            statusCode: res.statusCode,
            duration,
            userAgent: req.headers['user-agent'],
        });
    });
    
    next();
});
```

## Error Handling

### Global Error Handler

```typescript
app.use((err: unknown, req: Request, res: Response, next: NextFunction) => {
    let serverError: ServerError;
    
    if (err instanceof CustomError) {
        serverError = err.toServerError();
    } else if ((err as Error).name === "ValidationError") {
        serverError = { trace: "0455", code: "ValidationFailed" };
    } else if ((err as Error).name === "CastError") {
        serverError = { trace: "0456", code: "InvalidObjectId" };
    } else {
        logger.error('Unhandled error:', err);
        serverError = { trace: "0460", code: "InternalError" };
    }
    
    ResponseService.sendError(res, serverError, HttpStatus.InternalServerError);
});
```

### Error Categories & Status Codes

| HTTP Status | Error Category | Example Codes |
|-------------|----------------|---------------|
| 400 | Validation Errors | `ValidationFailed`, `InvalidInput` |
| 401 | Authentication Errors | `NotLoggedIn`, `InvalidToken` |
| 403 | Authorization Errors | `InsufficientPermissions`, `UnsafeOrigin` |
| 404 | Not Found Errors | `UserNotFound`, `ResourceNotFound` |
| 429 | Rate Limit Errors | `RateLimitExceeded` |
| 500 | Server Errors | `InternalError`, `DatabaseError` |

## Testing APIs

### Unit Testing Patterns

```typescript
import { describe, it, expect, beforeEach } from '@jest/testing-library';
import { user } from '../endpoints/logic/user';
import { mockContext, mockUserService } from '../__test/mocks';

describe('User Endpoints', () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });
    
    describe('findOne', () => {
        it('should return user when found', async () => {
            // Arrange
            const input = { id: 'user-123' };
            const context = mockContext({ userId: 'user-123' });
            const selection = { id: true, name: true };
            const expectedUser = { id: 'user-123', name: 'John Doe' };
            
            mockUserService.findById.mockResolvedValue(expectedUser);
            
            // Act
            const result = await user.findOne({ input }, context, selection);
            
            // Assert
            expect(result).toEqual(expectedUser);
            expect(mockUserService.findById).toHaveBeenCalledWith('user-123', selection);
        });
        
        it('should throw error when user not found', async () => {
            const input = { id: 'nonexistent' };
            const context = mockContext({ userId: 'user-123' });
            const selection = { id: true };
            
            mockUserService.findById.mockResolvedValue(null);
            
            await expect(
                user.findOne({ input }, context, selection)
            ).rejects.toThrow('UserNotFound');
        });
    });
});
```

### Integration Testing

```typescript
import request from 'supertest';
import { app } from '../app';
import { setupTestDatabase, teardownTestDatabase } from '../__test/database';

describe('User API Integration', () => {
    beforeAll(async () => {
        await setupTestDatabase();
    });
    
    afterAll(async () => {
        await teardownTestDatabase();
    });
    
    describe('GET /api/user/:id', () => {
        it('should return user data', async () => {
            const user = await createTestUser();
            const authToken = await generateAuthToken(user);
            
            const response = await request(app)
                .get(`/api/user/${user.id}`)
                .set('Cookie', `jwt=${authToken}`)
                .expect(200);
            
            expect(response.body).toMatchObject({
                id: user.id,
                name: user.name,
                email: user.email,
            });
        });
        
        it('should return 401 for unauthenticated requests', async () => {
            const response = await request(app)
                .get('/api/user/123')
                .expect(401);
            
            expect(response.body.code).toBe('NotLoggedIn');
        });
    });
});
```

### WebSocket Testing

```typescript
import { io as Client } from 'socket.io-client';
import { SocketService } from '../sockets/io';

describe('WebSocket API', () => {
    let clientSocket: any;
    let serverSocket: any;
    
    beforeAll((done) => {
        clientSocket = Client('http://localhost:3001', {
            auth: { token: validJwtToken },
        });
        
        SocketService.get().io.on('connection', (socket) => {
            serverSocket = socket;
        });
        
        clientSocket.on('connect', done);
    });
    
    afterAll(() => {
        clientSocket.close();
        SocketService.get().io.close();
    });
    
    it('should emit message to room', (done) => {
        const roomId = 'test-room';
        const message = { text: 'Hello World' };
        
        clientSocket.emit('join_room', roomId);
        clientSocket.on('message_received', (data) => {
            expect(data).toEqual(message);
            done();
        });
        
        SocketService.get().emitSocketEvent('message_received', roomId, message);
    });
});
```

## Configuration

### Environment Variables

```bash
# Server Configuration
SERVER_PORT=3001
SERVER_URL=http://localhost:3001
WORKER_ID=0  # For distributed ID generation

# Security (Required)
JWT_PRIV=<RS256_private_key>
JWT_PUB=<RS256_public_key>
VAPID_PUBLIC_KEY=<push_notification_public>
VAPID_PRIVATE_KEY=<push_notification_private>

# Database & Caching
DATABASE_URL=postgresql://user:pass@host:5432/db
REDIS_URL=redis://localhost:6379

# External Services
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
S3_BUCKET_NAME=vrooli-files

# MCP Configuration  
MCP_API_KEY=<development_mcp_key>
MCP_SERVER_MODE=sse  # or stdio
```

### Development vs Production

| Configuration | Development | Production |
|---------------|-------------|------------|
| **CORS** | Allow all origins | Restricted safe origins |
| **Rate Limiting** | Relaxed limits | Strict enforcement |
| **SSL/TLS** | Optional HTTP | Required HTTPS |
| **WebSocket Adapter** | In-memory | Redis adapter |
| **MCP Mode** | STDIO | SSE |
| **Logging Level** | Debug | Info/Warn/Error |
| **File Upload Limits** | 20MB | 10MB |
| **Session Duration** | Extended | Standard |

## Best Practices

### API Design

1. **RESTful Principles**: Use appropriate HTTP methods and status codes
2. **Consistent Naming**: camelCase for JSON, kebab-case for URLs
3. **Versioning Strategy**: Include version in URL path (`/api/v1/`)
4. **Pagination**: Always paginate list endpoints
5. **Field Selection**: Allow clients to specify required fields

### Security

1. **Input Validation**: Validate all inputs with schemas
2. **Authentication**: Require authentication for sensitive operations
3. **Authorization**: Implement granular permission checks
4. **Rate Limiting**: Apply appropriate limits per endpoint
5. **Error Messages**: Don't leak sensitive information

### Performance

1. **Caching**: Implement multi-level caching strategy
2. **Database Queries**: Optimize with proper indexes and selection
3. **Connection Pooling**: Use connection pools for all external services
4. **Monitoring**: Track performance metrics and errors
5. **Horizontal Scaling**: Design for stateless, scalable architecture

### Code Quality

1. **Type Safety**: Use TypeScript for all API code
2. **Error Handling**: Implement comprehensive error handling
3. **Testing**: Write unit and integration tests
4. **Documentation**: Document all endpoints and patterns
5. **Consistent Patterns**: Follow established patterns for new endpoints

### Monitoring

1. **Health Checks**: Implement health checks for all services
2. **Logging**: Use structured logging with correlation IDs
3. **Metrics**: Track key performance indicators
4. **Alerting**: Set up alerts for critical failures
5. **Observability**: Provide visibility into system behavior

---

This comprehensive guide covers all aspects of Vrooli's API architecture and development patterns. For specific implementation details, refer to the source code and individual component documentation. 