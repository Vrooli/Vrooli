# Playwright Driver: Current State & Action Plan

## 🚨 Critical Finding

**CONTRACT VIOLATION DETECTED**: The Playwright engine reports `SupportsIframes: true` in its capabilities, but **frame-switch is not implemented**. This could cause silent workflow failures when users attempt iframe navigation.

## Current State

### ✅ Implemented (13/28 node types)
- navigate, click, hover, type, scroll, wait
- evaluate, extract, assert
- uploadfile, download, screenshot

### ❌ Missing (15 node types)
- **frame-switch** ⚠️ CRITICAL - marked as supported!
- blur, focus
- cookie-storage
- drag-drop, gesture
- keyboard, shortcut
- network-mock
- rotate
- select
- tab-switch

### 🏗️ Architecture Issues
- Single 503-line JavaScript file (unmaintainable)
- No type safety
- No structured logging or metrics
- Minimal error handling
- No testing infrastructure
- No session timeout/cleanup
- No resource limits

## Recommended Action Plan

### Phase 1: Foundation (Week 1)
**Goal**: Migrate to TypeScript with proper architecture

**Structure**:
```
src/
├── types/           # Go contract types
├── session/         # Session lifecycle
├── handlers/        # One file per instruction category
├── telemetry/       # Event collection
├── routes/          # HTTP endpoints
├── middleware/      # Error handling, logging
└── utils/           # Logger, metrics, validation
```

**Key Tasks**:
1. TypeScript setup with strict type checking
2. Define type-safe contracts matching Go
3. Set up Winston logging + Prometheus metrics
4. Implement proper session management
5. Migrate existing 13 handlers
6. Testing infrastructure with Jest
7. Achieve >80% test coverage

### Phase 2: Critical Features (Week 2)
**Goal**: Fix contract violation and implement high-value features

**Priority Order**:
1. **Frame-switch** (CRITICAL - fixes contract violation)
2. Focus/blur (simple, high value)
3. Cookie/storage operations (essential for auth flows)
4. Select dropdown (common UI pattern)
5. Keyboard/shortcuts (essential for power users)

**Outcome**: Contract compliance + 18/28 node types

### Phase 3: Advanced Features (Week 3)
**Goal**: Complete remaining node types

1. Drag/drop and gestures
2. Multi-tab support (update `AllowsParallelTabs: true`)
3. Network mocking (essential for testing)
4. Device rotation

**Outcome**: All 28 node types implemented

### Phase 4: Production (Week 4)
**Goal**: Production-grade reliability and observability

1. Performance optimization
   - Screenshot caching/deduplication
   - Session pooling
   - Resource cleanup
2. Enhanced error handling
   - Structured error hierarchy
   - Map to Go FailureKind taxonomy
3. Observability
   - Comprehensive metrics (p50, p95, p99)
   - Optional OpenTelemetry tracing
   - Deep health checks
4. Documentation
   - Architecture diagrams
   - API specification
   - Deployment guide
   - Monitoring runbook
5. Deployment
   - Production Dockerfile
   - Docker Compose
   - Deployment automation

## Expected Outcomes

### Code Quality
- ✅ 100% TypeScript with strict types
- ✅ >95% test coverage
- ✅ Zero ESLint errors
- ✅ Comprehensive documentation

### Functionality
- ✅ All 28 node types implemented
- ✅ 100% contract compliance
- ✅ Frame-switch working
- ✅ Multi-tab support enabled

### Performance
- ✅ <100ms overhead per instruction
- ✅ <500ms screenshot capture
- ✅ <2s session startup
- ✅ 10+ concurrent sessions

### Reliability
- ✅ Zero memory leaks
- ✅ Graceful error handling
- ✅ Proper resource cleanup
- ✅ Production-ready monitoring

## Timeline

| Week | Phase | Key Deliverable |
|------|-------|----------------|
| 1 | Foundation | TypeScript migration complete |
| 2 | Critical Features | Contract compliance + 18/28 types |
| 3 | Advanced Features | All 28/28 types implemented |
| 4 | Production | Deployment ready |

**Total: 4 weeks**

## Immediate Next Steps

1. ✅ Review and approve plan
2. Create Phase 1 project structure
3. Set up TypeScript + tooling
4. Define type contracts
5. **Implement frame-switch handler** (highest priority!)
6. Migrate existing handlers
7. Write tests

## Files to Reference

- **Detailed Plan**: `docs/plans/playwright-driver-completion.md`
- **Go Contracts**: `api/automation/contracts/*.go`
- **Current Driver**: `api/automation/playwright-driver/server.js`
- **Handler Docs**: `docs/nodes/*.md`
- **Automation README**: `api/automation/README.md`

## Success Criteria

Before marking this complete, verify:

- [ ] All 28 node types implemented and tested
- [ ] Frame-switch working (contract violation fixed)
- [ ] Test coverage >95%
- [ ] TypeScript with strict mode
- [ ] Production deployment tested
- [ ] Documentation complete
- [ ] Performance benchmarks met
- [ ] Zero memory leaks
- [ ] Integration tests passing with Go API

---

**Ready to begin? Start with Phase 1 foundation work.**
