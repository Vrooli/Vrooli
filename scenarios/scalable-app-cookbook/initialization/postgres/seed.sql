-- Seed data for Scalable App Cookbook
-- Populates the database with core architectural patterns from the master outline

SET search_path TO scalable_app_cookbook, public;

-- Part A - Architectural Foundations

-- 1. Architecture Styles & Boundaries
INSERT INTO patterns (title, chapter, section, maturity_level, tags, what_and_why, when_to_use, tradeoffs) VALUES
(
  'Modular Monolith',
  'Part A - Architectural Foundations',
  'Architecture Styles & Boundaries',
  'L1',
  ARRAY['architecture', 'modularity', 'monolith', 'boundaries'],
  'A monolithic application organized into distinct modules with clear boundaries and minimal coupling. Each module encapsulates related functionality and communicates through well-defined interfaces, providing the benefits of modularity without the complexity of distributed systems.',
  'Use when you need clear separation of concerns but want to avoid the complexity of microservices. Ideal for teams starting with a monolith but planning for future extraction of services. Perfect for applications with clear domain boundaries but shared infrastructure needs.',
  'Pros: Easier deployment, debugging, and testing than microservices. Shared database enables ACID transactions. Lower operational complexity. Cons: Still shares runtime and database, making independent scaling difficult. Technology stack coupling. Risk of module boundary erosion over time.'
),
(
  'Hexagonal Architecture (Ports & Adapters)',
  'Part A - Architectural Foundations',
  'Architecture Styles & Boundaries',
  'L2',
  ARRAY['architecture', 'clean-architecture', 'ports-adapters', 'testability'],
  'An architectural pattern that isolates core business logic from external concerns through ports (interfaces) and adapters (implementations). The business logic sits at the center, with all external dependencies (databases, web frameworks, external services) connected through adapters.',
  'Use when you need high testability, technology independence, and clear separation between business logic and infrastructure. Essential for applications that must support multiple interfaces (web, CLI, API) or may need to swap out infrastructure components.',
  'Pros: Highly testable with business logic isolated from frameworks. Easy to swap infrastructure components. Clear dependency direction. Technology-agnostic core. Cons: Initial complexity overhead. More interfaces and indirection. Can be over-engineering for simple CRUD applications.'
),
(
  'Domain-Driven Design (DDD) Bounded Contexts',
  'Part A - Architectural Foundations',
  'Architecture Styles & Boundaries',
  'L3',
  ARRAY['ddd', 'bounded-context', 'domain-modeling', 'microservices'],
  'Strategic design pattern that divides complex domains into smaller, more manageable bounded contexts. Each context has its own ubiquitous language, models, and business rules, with explicit integration patterns between contexts.',
  'Use for complex business domains with multiple subdomains. Essential when different parts of the system have different business rules, data models, or evolution rates. Critical for large teams working on different business areas.',
  'Pros: Reduces complexity by establishing clear boundaries. Enables independent team ownership. Supports different technical approaches per context. Facilitates microservices extraction. Cons: Requires domain expertise. Complex inter-context integration. Potential for over-segmentation leading to chatty interfaces.'
)
ON CONFLICT (title) DO UPDATE SET
  chapter = EXCLUDED.chapter,
  section = EXCLUDED.section,
  maturity_level = EXCLUDED.maturity_level,
  tags = EXCLUDED.tags,
  what_and_why = EXCLUDED.what_and_why,
  when_to_use = EXCLUDED.when_to_use,
  tradeoffs = EXCLUDED.tradeoffs,
  updated_at = CURRENT_TIMESTAMP;

-- 2. Non-Functional Requirements & SLOs
INSERT INTO patterns (title, chapter, section, maturity_level, tags, what_and_why, when_to_use, tradeoffs) VALUES
(
  'SLI/SLO Framework',
  'Part A - Architectural Foundations',
  'Non-Functional Requirements & SLOs',
  'L2',
  ARRAY['slo', 'monitoring', 'reliability', 'error-budgets'],
  'Service Level Indicators (SLIs) measure system behavior, Service Level Objectives (SLOs) define target reliability, and Error Budgets balance reliability with feature velocity. This framework provides objective reliability measurement and decision-making criteria.',
  'Use when you need objective reliability measurement, want to balance feature development with stability, or need to establish SRE practices. Essential for production systems with reliability requirements and multiple stakeholders with different risk tolerances.',
  'Pros: Objective reliability measurement. Data-driven decision making. Balances innovation with stability. Enables error budget policies. Cons: Requires monitoring infrastructure investment. Can be complex to implement correctly. Risk of gaming metrics if not carefully designed.'
)
ON CONFLICT (title) DO UPDATE SET
  chapter = EXCLUDED.chapter,
  section = EXCLUDED.section,
  maturity_level = EXCLUDED.maturity_level,
  tags = EXCLUDED.tags,
  what_and_why = EXCLUDED.what_and_why,
  when_to_use = EXCLUDED.when_to_use,
  tradeoffs = EXCLUDED.tradeoffs,
  updated_at = CURRENT_TIMESTAMP;

-- 3. Multi-Tenant SaaS Patterns
INSERT INTO patterns (title, chapter, section, maturity_level, tags, what_and_why, when_to_use, tradeoffs) VALUES
(
  'Database-per-Tenant Isolation',
  'Part A - Architectural Foundations',
  'Multi-Tenant SaaS Patterns',
  'L3',
  ARRAY['multi-tenancy', 'isolation', 'database', 'saas'],
  'Each tenant gets their own dedicated database instance, providing maximum isolation and customization flexibility. This pattern ensures complete data separation and allows per-tenant scaling, backup, and compliance policies.',
  'Use for enterprise SaaS with strict compliance requirements, highly regulated industries, or tenants with significantly different scaling needs. Essential when tenants require custom schemas or data residency in specific regions.',
  'Pros: Maximum isolation and security. Per-tenant scaling and optimization. Custom schema support. Independent backup/restore. Cons: Higher infrastructure costs. Complex operational overhead. Database proliferation management. Cross-tenant analytics complexity.'
),
(
  'Schema-per-Tenant Isolation',
  'Part A - Architectural Foundations',
  'Multi-Tenant SaaS Patterns',
  'L2',
  ARRAY['multi-tenancy', 'schema-isolation', 'postgres', 'cost-effective'],
  'Each tenant gets their own database schema within a shared database instance. Provides good isolation while sharing infrastructure costs and operational complexity. Uses row-level security and schema-level permissions for data protection.',
  'Use for SaaS applications needing good isolation without the operational overhead of separate databases. Ideal for B2B applications with moderate security requirements and predictable tenant sizes.',
  'Pros: Good isolation with shared infrastructure. Lower operational complexity than database-per-tenant. Cost-effective for many tenants. Cons: Shared database performance impact. Complex schema migration coordination. Potential for resource contention between tenants.'
)
ON CONFLICT (title) DO UPDATE SET
  chapter = EXCLUDED.chapter,
  section = EXCLUDED.section,
  maturity_level = EXCLUDED.maturity_level,
  tags = EXCLUDED.tags,
  what_and_why = EXCLUDED.what_and_why,
  when_to_use = EXCLUDED.when_to_use,
  tradeoffs = EXCLUDED.tradeoffs,
  updated_at = CURRENT_TIMESTAMP;

-- 4. Data Architecture & Consistency
INSERT INTO patterns (title, chapter, section, maturity_level, tags, what_and_why, when_to_use, tradeoffs) VALUES
(
  'CQRS (Command Query Responsibility Segregation)',
  'Part A - Architectural Foundations',
  'Data Architecture & Consistency',
  'L3',
  ARRAY['cqrs', 'event-sourcing', 'read-models', 'scalability'],
  'Separates read and write operations into different models, allowing optimization of each for their specific use cases. Commands modify state through an aggregate model, while queries use optimized read models that can be denormalized and cached.',
  'Use when read and write patterns are very different, you need independent scaling of reads vs writes, or want to optimize query performance without compromising command validation. Common in event-sourced systems and high-read-volume applications.',
  'Pros: Independent optimization of read/write paths. Better performance through specialized models. Supports complex business logic validation. Enables event sourcing. Cons: Increased complexity and development overhead. Eventual consistency between command and query models. Duplicate logic risk.'
),
(
  'Transactional Outbox Pattern',
  'Part A - Architectural Foundations',
  'Data Architecture & Consistency',
  'L2',
  ARRAY['outbox', 'consistency', 'events', 'reliability'],
  'Ensures reliable event publishing by storing events in the same database transaction as business data changes. A separate process reads from the outbox table and publishes events to external systems, guaranteeing at-least-once delivery.',
  'Use when you need reliable event publishing without distributed transactions. Essential for microservices communication, audit trails, and any system where events must not be lost even if external systems are unavailable.',
  'Pros: Guaranteed event delivery without 2PC. Uses local transactions only. Survives external system failures. Simple to implement and understand. Cons: Requires additional outbox processing. Potential for duplicate events (requires idempotent consumers). Eventual consistency with external systems.'
)
ON CONFLICT (title) DO UPDATE SET
  chapter = EXCLUDED.chapter,
  section = EXCLUDED.section,
  maturity_level = EXCLUDED.maturity_level,
  tags = EXCLUDED.tags,
  what_and_why = EXCLUDED.what_and_why,
  when_to_use = EXCLUDED.when_to_use,
  tradeoffs = EXCLUDED.tradeoffs,
  updated_at = CURRENT_TIMESTAMP;

-- 5. API Design & Evolution
INSERT INTO patterns (title, chapter, section, maturity_level, tags, what_and_why, when_to_use, tradeoffs) VALUES
(
  'API Versioning Strategy',
  'Part A - Architectural Foundations',
  'API Design & Evolution',
  'L2',
  ARRAY['api-versioning', 'backwards-compatibility', 'evolution', 'rest'],
  'Systematic approach to evolving APIs while maintaining backwards compatibility. Includes URL versioning, header versioning, and content negotiation strategies with clear deprecation policies and migration paths.',
  'Use for public APIs, APIs with external consumers, or any API that needs to evolve without breaking existing clients. Critical for SaaS platforms and any service with a significant integration ecosystem.',
  'Pros: Enables independent client and server evolution. Prevents breaking changes. Supports gradual migration. Clear deprecation timeline. Cons: Increased implementation complexity. Multiple version maintenance overhead. Client confusion with too many versions.'
),
(
  'GraphQL Federation',
  'Part A - Architectural Foundations',
  'API Design & Evolution',
  'L3',
  ARRAY['graphql', 'federation', 'microservices', 'gateway'],
  'Architectural pattern for composing multiple GraphQL services into a single unified graph. Each service owns part of the schema, and a gateway federates queries across services while maintaining type safety and resolving dependencies.',
  'Use when building GraphQL APIs across multiple microservices, need unified API experience, or want to maintain service autonomy while providing cohesive client experience. Essential for large organizations with multiple API teams.',
  'Pros: Unified API experience across services. Service autonomy with shared schema. Type safety across federation. Efficient query resolution. Cons: Complex federation logic. Potential for circular dependencies. Gateway becomes critical path. Schema coordination overhead.'
)
ON CONFLICT (title) DO UPDATE SET
  chapter = EXCLUDED.chapter,
  section = EXCLUDED.section,
  maturity_level = EXCLUDED.maturity_level,
  tags = EXCLUDED.tags,
  what_and_why = EXCLUDED.what_and_why,
  when_to_use = EXCLUDED.when_to_use,
  tradeoffs = EXCLUDED.tradeoffs,
  updated_at = CURRENT_TIMESTAMP;

-- Part B - Resiliency & Scale

-- 7. Resiliency Patterns  
INSERT INTO patterns (title, chapter, section, maturity_level, tags, what_and_why, when_to_use, tradeoffs) VALUES
(
  'Circuit Breaker Pattern',
  'Part B - Resiliency & Scale',
  'Resiliency Patterns',
  'L2',
  ARRAY['circuit-breaker', 'resilience', 'fault-tolerance', 'microservices'],
  'Prevents cascading failures by monitoring external service calls and "opening the circuit" when failure rates exceed thresholds. Provides fast failure and automatic recovery detection, protecting both caller and called service from overload.',
  'Use when calling external services that may be unreliable or when you need to prevent cascading failures. Essential in microservices architectures and any system with external dependencies that could impact user experience.',
  'Pros: Prevents cascading failures. Fast failure detection. Automatic recovery. Protects downstream services. Configurable thresholds. Cons: Adds complexity to service calls. Risk of false positives. Requires careful threshold tuning. May hide underlying issues.'
),
(
  'Bulkhead Pattern',
  'Part B - Resiliency & Scale',
  'Resiliency Patterns',
  'L2',
  ARRAY['bulkhead', 'isolation', 'resource-pooling', 'resilience'],
  'Isolates critical resources into separate pools to prevent resource exhaustion in one area from affecting others. Like watertight compartments in ships, failures in one bulkhead don''t sink the entire system.',
  'Use when different operations have different criticality levels, varying resource requirements, or different SLA expectations. Critical for systems where some operations must remain available even when others are struggling.',
  'Pros: Fault isolation between different operations. Prevents resource starvation. Supports different SLA requirements. Enables priority-based resource allocation. Cons: Resource utilization inefficiency. Increased operational complexity. Requires careful resource planning.'
)
ON CONFLICT (title) DO UPDATE SET
  chapter = EXCLUDED.chapter,
  section = EXCLUDED.section,
  maturity_level = EXCLUDED.maturity_level,
  tags = EXCLUDED.tags,
  what_and_why = EXCLUDED.what_and_why,
  when_to_use = EXCLUDED.when_to_use,
  tradeoffs = EXCLUDED.tradeoffs,
  updated_at = CURRENT_TIMESTAMP;

-- Now create initial recipes for some core patterns

-- Dependency Injection recipes
INSERT INTO recipes (pattern_id, title, type, prerequisites, steps, config_snippets, validation_checks, artifacts, prompts, timeout_sec) VALUES
(
  (SELECT id FROM patterns WHERE title = 'Modular Monolith'),
  'Modular Monolith with Clean Boundaries',
  'greenfield',
  ARRAY['Go knowledge', 'Database design experience'],
  '[
    {"id": "step1", "desc": "Design module boundaries based on business capabilities", "cmds": ["mkdir -p internal/{user,product,order}/domain", "mkdir -p internal/{user,product,order}/infrastructure"]},
    {"id": "step2", "desc": "Create module interfaces and dependency contracts", "code": "type UserService interface {\n  CreateUser(ctx context.Context, user User) error\n  GetUser(ctx context.Context, id string) (*User, error)\n}"},
    {"id": "step3", "desc": "Implement dependency injection container", "cmds": ["go mod init modular-monolith", "go get github.com/google/wire"]},
    {"id": "step4", "desc": "Set up module-to-module communication rules", "code": "// Module communication rules\n// 1. No direct database access across modules\n// 2. Use interfaces for all module dependencies\n// 3. Events for loose coupling"},
    {"id": "step5", "desc": "Create integration tests for module boundaries", "cmds": ["mkdir -p test/integration", "go get github.com/stretchr/testify"]}
  ]',
  '{"docker": {"compose": "version: 3.8\nservices:\n  app:\n    build: .\n    ports:\n      - 8080:8080\n    environment:\n      - DB_URL=postgres://..."}}',
  ARRAY['All modules follow interface contracts', 'No circular dependencies between modules', 'Integration tests pass', 'Module boundaries enforced by linting'],
  ARRAY['Module dependency diagram', 'API documentation per module', 'Integration test suite'],
  ARRAY['Design module boundaries for business domain X', 'Implement clean dependency injection for module Y', 'Create integration tests for module boundaries'],
  300
);

-- JWT Authentication recipe
INSERT INTO recipes (pattern_id, title, type, prerequisites, steps, config_snippets, validation_checks, artifacts, prompts, timeout_sec) VALUES
(
  (SELECT id FROM patterns WHERE title = 'API Versioning Strategy'),
  'JWT Authentication with Refresh Tokens',
  'greenfield',
  ARRAY['Understanding of JWT standards', 'Database for token storage', 'HTTPS setup'],
  '[
    {"id": "auth1", "desc": "Set up JWT signing and validation", "code": "type TokenService struct {\n  signingKey []byte\n  accessTTL time.Duration\n  refreshTTL time.Duration\n}", "cmds": ["go get github.com/golang-jwt/jwt/v5"]},
    {"id": "auth2", "desc": "Implement secure refresh token rotation", "code": "func (ts *TokenService) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {\n  // Validate refresh token\n  // Issue new access + refresh tokens\n  // Invalidate old refresh token\n  return &TokenPair{Access: newAccess, Refresh: newRefresh}, nil\n}"},
    {"id": "auth3", "desc": "Add middleware for token validation", "code": "func JWTMiddleware(tokenService *TokenService) mux.MiddlewareFunc {\n  return func(next http.Handler) http.Handler {\n    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {\n      // Extract and validate JWT\n      // Set user context\n      next.ServeHTTP(w, r)\n    })\n  }\n}"},
    {"id": "auth4", "desc": "Implement logout with token blacklisting", "cmds": ["CREATE TABLE revoked_tokens (token_id VARCHAR(255) PRIMARY KEY, revoked_at TIMESTAMP)"]},
    {"id": "auth5", "desc": "Add rate limiting for auth endpoints", "code": "// Rate limit: 5 login attempts per minute per IP\nfunc LoginRateLimiter() mux.MiddlewareFunc {\n  limiter := rate.NewLimiter(rate.Every(12*time.Second), 5)\n  return middleware.RateLimit(limiter)\n}"}
  ]',
  '{"security": {"jwt_secret_rotation": "0 2 * * 0", "token_cleanup": "*/5 * * * *"}, "nginx": {"ssl_protocols": "TLSv1.2 TLSv1.3", "add_header": "Strict-Transport-Security max-age=31536000"}}',
  ARRAY['JWT tokens validate correctly', 'Refresh token rotation works', 'Revoked tokens are rejected', 'Rate limiting prevents brute force', 'HTTPS enforced in production'],
  ARRAY['JWT middleware implementation', 'Token rotation automation', 'Security policy documentation', 'Rate limiting configuration'],
  ARRAY['Implement secure JWT authentication for service X', 'Add refresh token rotation to existing auth system', 'Create auth middleware with proper error handling'],
  180
);

-- Circuit Breaker recipe
INSERT INTO recipes (pattern_id, title, type, prerequisites, steps, config_snippets, validation_checks, artifacts, prompts, timeout_sec) VALUES
(
  (SELECT id FROM patterns WHERE title = 'Circuit Breaker Pattern'),
  'Circuit Breaker with Exponential Backoff',
  'greenfield',
  ARRAY['Understanding of failure modes', 'Monitoring system'],
  '[
    {"id": "cb1", "desc": "Implement circuit breaker state machine", "code": "type CircuitBreaker struct {\n  state State // Closed, Open, HalfOpen\n  failures int\n  lastFailure time.Time\n  threshold int\n  timeout time.Duration\n  mutex sync.RWMutex\n}"},
    {"id": "cb2", "desc": "Add failure detection and state transitions", "code": "func (cb *CircuitBreaker) Call(fn func() error) error {\n  cb.mutex.Lock()\n  defer cb.mutex.Unlock()\n  \n  if cb.state == Open {\n    if time.Since(cb.lastFailure) > cb.timeout {\n      cb.state = HalfOpen\n    } else {\n      return ErrCircuitOpen\n    }\n  }\n  \n  err := fn()\n  return cb.handleResult(err)\n}"},
    {"id": "cb3", "desc": "Implement exponential backoff for retry logic", "code": "func ExponentialBackoff(attempt int, base time.Duration, max time.Duration) time.Duration {\n  backoff := time.Duration(math.Pow(2, float64(attempt))) * base\n  if backoff > max {\n    return max\n  }\n  return backoff\n}"},
    {"id": "cb4", "desc": "Add monitoring and alerting", "cmds": ["# Prometheus metrics", "circuit_breaker_state{service=\"api\"}", "circuit_breaker_failures_total{service=\"api\"}"]},
    {"id": "cb5", "desc": "Create fallback mechanisms", "code": "func (cb *CircuitBreaker) CallWithFallback(fn func() error, fallback func() error) error {\n  err := cb.Call(fn)\n  if err == ErrCircuitOpen && fallback != nil {\n    return fallback()\n  }\n  return err\n}"}
  ]',
  '{"thresholds": {"failure_count": 5, "failure_rate": 0.5, "timeout_seconds": 60}, "monitoring": {"alert_on_open": true, "dashboard_panels": ["state", "failure_rate", "request_volume"]}}',
  ARRAY['Circuit breaker opens on threshold breach', 'Automatic recovery after timeout', 'Fallback mechanisms work', 'Metrics are exported correctly', 'No false positive openings'],
  ARRAY['Circuit breaker implementation', 'Monitoring dashboard', 'Alerting rules', 'Fallback strategy documentation'],
  ARRAY['Add circuit breaker to external service calls', 'Implement fallback strategy for service X', 'Create monitoring for circuit breaker health'],
  240
);

-- Now add implementations for the JWT recipe
INSERT INTO implementations (recipe_id, language, code, file_path, description, dependencies, test_code) VALUES
(
  (SELECT id FROM recipes WHERE title = 'JWT Authentication with Refresh Tokens'),
  'go',
  'package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenService struct {
	signingKey  []byte
	accessTTL   time.Duration
	refreshTTL  time.Duration
	tokenStore  TokenStore
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Roles  []string `json:"roles"`
	jwt.RegisteredClaims
}

func NewTokenService(signingKey []byte, accessTTL, refreshTTL time.Duration, store TokenStore) *TokenService {
	return &TokenService{
		signingKey: signingKey,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		tokenStore: store,
	}
}

func (ts *TokenService) GenerateTokens(ctx context.Context, userID, email string, roles []string) (*TokenPair, error) {
	// Generate access token
	accessClaims := Claims{
		UserID: userID,
		Email:  email,
		Roles:  roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ts.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString(ts.signingKey)
	if err != nil {
		return nil, err
	}

	// Generate refresh token
	refreshToken, err := generateSecureToken()
	if err != nil {
		return nil, err
	}

	// Store refresh token
	err = ts.tokenStore.StoreRefreshToken(ctx, refreshToken, userID, time.Now().Add(ts.refreshTTL))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(ts.accessTTL.Seconds()),
	}, nil
}

func (ts *TokenService) RefreshTokens(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Validate refresh token
	userID, err := ts.tokenStore.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// Revoke old refresh token
	err = ts.tokenStore.RevokeRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	// Get user info for new tokens
	// This would typically involve a database lookup
	return ts.GenerateTokens(ctx, userID, "", []string{})
}

func generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

type TokenStore interface {
	StoreRefreshToken(ctx context.Context, token, userID string, expiresAt time.Time) error
	ValidateRefreshToken(ctx context.Context, token string) (userID string, err error)
	RevokeRefreshToken(ctx context.Context, token string) error
}',
  'internal/auth/token_service.go',
  'Complete JWT authentication service with refresh token rotation and secure storage',
  ARRAY['github.com/golang-jwt/jwt/v5'],
  'func TestTokenService_GenerateTokens(t *testing.T) {
	service := NewTokenService([]byte("test-key"), time.Hour, 24*time.Hour, &mockTokenStore{})
	
	tokens, err := service.GenerateTokens(context.Background(), "user123", "test@example.com", []string{"user"})
	
	assert.NoError(t, err)
	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.Equal(t, int64(3600), tokens.ExpiresIn)
}'
);

-- Add Circuit Breaker Go implementation
INSERT INTO implementations (recipe_id, language, code, file_path, description, dependencies) VALUES
(
  (SELECT id FROM recipes WHERE title = 'Circuit Breaker with Exponential Backoff'),
  'go',
  'package resilience

import (
	"context"
	"errors"
	"math"
	"sync"
	"time"
)

var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
	ErrTooManyRequests = errors.New("too many requests")
)

type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

type CircuitBreaker struct {
	mutex         sync.RWMutex
	state         State
	failures      int
	requests      int
	lastFailure   time.Time
	lastSuccess   time.Time
	
	// Configuration
	maxFailures   int
	timeout       time.Duration
	maxRequests   int // for half-open state
	
	// Callbacks
	onStateChange func(from, to State)
}

type Config struct {
	MaxFailures   int
	Timeout       time.Duration
	MaxRequests   int
	OnStateChange func(from, to State)
}

func NewCircuitBreaker(config Config) *CircuitBreaker {
	return &CircuitBreaker{
		state:         StateClosed,
		maxFailures:   config.MaxFailures,
		timeout:       config.Timeout,
		maxRequests:   config.MaxRequests,
		onStateChange: config.OnStateChange,
	}
}

func (cb *CircuitBreaker) Call(ctx context.Context, fn func() error) error {
	err := cb.beforeRequest()
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			cb.onFailure()
			panic(r)
		}
	}()

	err = fn()
	if err != nil {
		cb.onFailure()
		return err
	}

	cb.onSuccess()
	return nil
}

func (cb *CircuitBreaker) beforeRequest() error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()

	switch cb.state {
	case StateOpen:
		if now.Sub(cb.lastFailure) > cb.timeout {
			cb.setState(StateHalfOpen)
			cb.requests = 0
			return nil
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		if cb.requests >= cb.maxRequests {
			return ErrTooManyRequests
		}
		cb.requests++
		return nil

	default: // StateClosed
		return nil
	}
}

func (cb *CircuitBreaker) onSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.lastSuccess = time.Now()

	if cb.state == StateHalfOpen {
		cb.setState(StateClosed)
		cb.failures = 0
	}
}

func (cb *CircuitBreaker) onFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.state == StateHalfOpen || cb.failures >= cb.maxFailures {
		cb.setState(StateOpen)
	}
}

func (cb *CircuitBreaker) setState(state State) {
	if cb.state == state {
		return
	}

	oldState := cb.state
	cb.state = state

	if cb.onStateChange != nil {
		cb.onStateChange(oldState, state)
	}
}

func (cb *CircuitBreaker) State() State {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// Exponential backoff utility
func ExponentialBackoff(attempt int, baseDelay, maxDelay time.Duration, jitter bool) time.Duration {
	delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay
	
	if delay > maxDelay {
		delay = maxDelay
	}
	
	if jitter {
		// Add up to 25% jitter
		jitterAmount := float64(delay) * 0.25 * (0.5 - rand.Float64())
		delay += time.Duration(jitterAmount)
	}
	
	return delay
}',
  'internal/resilience/circuit_breaker.go',
  'Production-ready circuit breaker with state management and exponential backoff',
  ARRAY['context support', 'thread-safe implementation']
);

-- Add TypeScript implementation for JWT
INSERT INTO implementations (recipe_id, language, code, file_path, description, dependencies, test_code) VALUES
(
  (SELECT id FROM recipes WHERE title = 'JWT Authentication with Refresh Tokens'),
  'typescript',
  'import jwt from ''jsonwebtoken'';
import crypto from ''crypto'';

interface TokenPair {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
}

interface Claims {
  userId: string;
  email: string;
  roles: string[];
  iat?: number;
  exp?: number;
}

interface TokenStore {
  storeRefreshToken(token: string, userId: string, expiresAt: Date): Promise<void>;
  validateRefreshToken(token: string): Promise<string>;
  revokeRefreshToken(token: string): Promise<void>;
}

export class TokenService {
  private signingKey: string;
  private accessTTL: number;
  private refreshTTL: number;
  private tokenStore: TokenStore;

  constructor(
    signingKey: string,
    accessTTL: number = 3600, // 1 hour
    refreshTTL: number = 604800, // 7 days
    tokenStore: TokenStore
  ) {
    this.signingKey = signingKey;
    this.accessTTL = accessTTL;
    this.refreshTTL = refreshTTL;
    this.tokenStore = tokenStore;
  }

  async generateTokens(userId: string, email: string, roles: string[] = []): Promise<TokenPair> {
    // Generate access token
    const accessPayload: Claims = {
      userId,
      email,
      roles,
    };

    const accessToken = jwt.sign(accessPayload, this.signingKey, {
      expiresIn: this.accessTTL,
      issuer: ''scalable-app'',
      subject: userId,
    });

    // Generate secure refresh token
    const refreshToken = crypto.randomBytes(32).toString(''base64url'');

    // Store refresh token
    const expiresAt = new Date(Date.now() + this.refreshTTL * 1000);
    await this.tokenStore.storeRefreshToken(refreshToken, userId, expiresAt);

    return {
      accessToken,
      refreshToken,
      expiresIn: this.accessTTL,
    };
  }

  async refreshTokens(refreshToken: string): Promise<TokenPair> {
    try {
      // Validate refresh token
      const userId = await this.tokenStore.validateRefreshToken(refreshToken);
      
      // Revoke old refresh token (rotation)
      await this.tokenStore.revokeRefreshToken(refreshToken);
      
      // Generate new token pair
      // Note: In real implementation, fetch user details from database
      return this.generateTokens(userId, '''', []);
    } catch (error) {
      throw new Error(''Invalid refresh token'');
    }
  }

  verifyAccessToken(token: string): Claims {
    try {
      const decoded = jwt.verify(token, this.signingKey) as Claims;
      return decoded;
    } catch (error) {
      throw new Error(''Invalid access token'');
    }
  }
}

// Express.js middleware
export function jwtMiddleware(tokenService: TokenService) {
  return (req: any, res: any, next: any) => {
    const authHeader = req.headers.authorization;
    
    if (!authHeader?.startsWith(''Bearer '')) {
      return res.status(401).json({ error: ''Missing or invalid authorization header'' });
    }

    const token = authHeader.substring(7);
    
    try {
      const claims = tokenService.verifyAccessToken(token);
      req.user = claims;
      next();
    } catch (error) {
      res.status(401).json({ error: ''Invalid or expired token'' });
    }
  };
}',
  'src/auth/tokenService.ts',
  'TypeScript JWT service with Express.js middleware integration',
  ARRAY['jsonwebtoken', '@types/jsonwebtoken'],
  'describe(''TokenService'', () => {
  let tokenService: TokenService;
  let mockTokenStore: jest.Mocked<TokenStore>;

  beforeEach(() => {
    mockTokenStore = {
      storeRefreshToken: jest.fn(),
      validateRefreshToken: jest.fn(),
      revokeRefreshToken: jest.fn(),
    };
    tokenService = new TokenService(''test-secret'', 3600, 604800, mockTokenStore);
  });

  it(''should generate valid token pairs'', async () => {
    const tokens = await tokenService.generateTokens(''user123'', ''test@example.com'', [''user'']);
    
    expect(tokens.accessToken).toBeTruthy();
    expect(tokens.refreshToken).toBeTruthy();
    expect(tokens.expiresIn).toBe(3600);
  });
});'
);

-- Add some sample CI gates
INSERT INTO ci_gates (pattern_id, gate_type, gate_name, gate_config, enforcement_level, description) VALUES
(
  (SELECT id FROM patterns WHERE title = 'JWT Authentication with Refresh Tokens'),
  'security',
  'JWT Security Validation',
  '{"checks": ["no_hardcoded_secrets", "proper_key_rotation", "secure_token_storage", "rate_limiting_enabled"], "tools": ["semgrep", "gosec"], "fail_on": ["high_severity"]}',
  'error',
  'Validates JWT implementation follows security best practices'
),
(
  (SELECT id FROM patterns WHERE title = 'Circuit Breaker Pattern'),
  'performance',
  'Circuit Breaker Performance',
  '{"max_latency_ms": 100, "max_memory_mb": 50, "load_test_duration": "30s", "success_rate_threshold": 0.95}',
  'warning',
  'Ensures circuit breaker implementation meets performance requirements'
);

-- Add observability configurations
INSERT INTO observability (pattern_id, observability_type, name, config) VALUES
(
  (SELECT id FROM patterns WHERE title = 'Circuit Breaker Pattern'),
  'sli',
  'Circuit Breaker Request Success Rate',
  '{"metric": "circuit_breaker_requests_total", "selector": "job=\"api\"", "threshold": ">= 0.99", "window": "5m"}'
),
(
  (SELECT id FROM patterns WHERE title = 'Circuit Breaker Pattern'),
  'dashboard',
  'Circuit Breaker Health Dashboard',
  '{"panels": [{"title": "Circuit State", "type": "stat", "query": "circuit_breaker_state"}, {"title": "Failure Rate", "type": "graph", "query": "rate(circuit_breaker_failures_total[5m])"}]}'
),
(
  (SELECT id FROM patterns WHERE title = 'JWT Authentication with Refresh Tokens'),
  'alert',
  'High Authentication Failure Rate',
  '{"condition": "rate(auth_failures_total[5m]) > 10", "severity": "warning", "description": "Authentication failure rate is above normal thresholds"}'
);
