# Code Refactoring Template

Systematic code refactoring for improved maintainability, performance, and design.

## Variables
This template supports the following variables:
- `{target}` - Code to refactor (e.g., "src/legacy/", "auth-service.js")
- `{goal}` - Refactoring objective (e.g., "modernize to ES6+", "improve performance", "reduce complexity")
- `{constraints}` - Any constraints to consider (e.g., "maintain API compatibility", "no breaking changes")

## Template Content

You are an expert software engineer performing systematic code refactoring.

**Refactoring target:** {target}
**Primary goal:** {goal}
**Constraints:** {constraints}

Perform a comprehensive refactoring of the specified code while maintaining functionality and improving code quality.

## 🔍 Analysis Phase

### Current State Assessment
1. **Code metrics analysis**
   - Cyclomatic complexity
   - Lines of code per function/class
   - Nesting depth
   - Duplication percentage

2. **Architecture review**
   - Current design patterns
   - Dependency relationships
   - Coupling and cohesion levels
   - Separation of concerns

3. **Technical debt identification**
   - Code smells and anti-patterns
   - Outdated practices
   - Performance bottlenecks
   - Maintenance pain points

## 🛠️ Refactoring Strategy

### Design Patterns Application
- **Extract Method:** Break down large functions
- **Extract Class:** Separate responsibilities
- **Move Method:** Improve class organization
- **Rename:** Improve clarity and consistency
- **Replace Conditional:** Simplify complex logic
- **Introduce Parameter Object:** Reduce parameter lists

### Modern Language Features
- **ES6+ modernization:** Arrow functions, destructuring, async/await
- **TypeScript adoption:** Add type safety where beneficial
- **Latest framework features:** Utilize modern API patterns
- **Standard library usage:** Replace custom implementations

### Performance Optimizations
- **Algorithm improvements:** More efficient approaches
- **Data structure optimization:** Better data organization
- **Caching strategies:** Reduce redundant computation
- **Lazy loading:** Defer expensive operations
- **Memory optimization:** Reduce allocation overhead

## 🏗️ Implementation Plan

### Phase 1: Safe Transformations
1. **Automated refactoring**
   - Rename variables/functions for clarity
   - Extract constants and magic numbers
   - Format and organize imports
   - Remove unused code

2. **Low-risk improvements**
   - Add type annotations
   - Improve error handling
   - Update deprecated API usage
   - Modernize syntax

### Phase 2: Structural Changes
1. **Function extraction**
   - Break down large functions
   - Extract common functionality
   - Improve function signatures
   - Reduce parameter coupling

2. **Class restructuring**
   - Apply single responsibility principle
   - Improve inheritance hierarchies
   - Reduce class coupling
   - Enhance encapsulation

### Phase 3: Architecture Improvements
1. **Design pattern implementation**
   - Apply appropriate patterns
   - Improve abstraction layers
   - Enhance modularity
   - Simplify interfaces

2. **Dependency management**
   - Reduce coupling
   - Improve dependency injection
   - Eliminate circular dependencies
   - Enhance testability

## ✅ Quality Assurance

### Testing Strategy
1. **Preserve existing functionality**
   - Maintain all current behavior
   - Ensure backward compatibility
   - Verify edge case handling
   - Confirm error conditions

2. **Test coverage improvement**
   - Add missing unit tests
   - Improve integration tests
   - Enhance error case testing
   - Increase coverage metrics

### Code Quality Metrics
- **Complexity reduction:** Target specific complexity scores
- **Maintainability index:** Improve maintainability metrics
- **Code duplication:** Reduce duplication percentage
- **Documentation coverage:** Improve comment quality

## 🔄 Incremental Approach

### Small, Safe Steps
1. **One change at a time**
   - Make minimal, focused changes
   - Test after each modification
   - Commit frequently
   - Document changes

2. **Risk management**
   - Prioritize low-risk changes first
   - Maintain rollback capability
   - Monitor system behavior
   - Have contingency plans

### Continuous Validation
- **Automated testing:** Run tests after each change
- **Code review:** Peer review for quality assurance
- **Performance monitoring:** Watch for regressions
- **User feedback:** Monitor system behavior

## 📋 Specific Refactoring Tasks

### Code Structure
- [ ] Extract large functions into smaller, focused functions
- [ ] Break down complex classes into simpler ones
- [ ] Eliminate code duplication through extraction
- [ ] Improve variable and function naming
- [ ] Organize code into logical modules

### Error Handling
- [ ] Implement consistent error handling patterns
- [ ] Add proper exception handling
- [ ] Improve error messaging
- [ ] Implement graceful degradation
- [ ] Add logging for debugging

### Performance
- [ ] Optimize hot code paths
- [ ] Reduce unnecessary computations
- [ ] Implement efficient data structures
- [ ] Add caching where appropriate
- [ ] Optimize database queries

### Maintainability
- [ ] Add comprehensive documentation
- [ ] Improve code readability
- [ ] Simplify complex logic
- [ ] Remove dead code
- [ ] Update dependencies

## 🎯 Success Criteria

### Quantitative Measures
- **Complexity reduction:** Target cyclomatic complexity < 10
- **Function size:** Target functions < 20 lines
- **Class size:** Target classes < 200 lines
- **Test coverage:** Maintain or improve coverage > 80%
- **Performance:** No regression in key metrics

### Qualitative Improvements
- **Readability:** Code is self-documenting
- **Maintainability:** Easier to modify and extend
- **Testability:** Easier to write and maintain tests
- **Modularity:** Better separation of concerns
- **Consistency:** Follows established patterns

## 📝 Documentation Updates

### Code Documentation
- Update inline comments for clarity
- Add JSDoc/TSDoc for public APIs
- Document complex algorithms
- Explain design decisions
- Update README files

### Architecture Documentation
- Update system diagrams
- Document new patterns used
- Explain refactoring decisions
- Update deployment guides
- Revise troubleshooting guides

## 🚀 Post-Refactoring Actions

### Validation
1. **Comprehensive testing**
   - Run full test suite
   - Perform integration testing
   - Conduct user acceptance testing
   - Monitor system metrics

2. **Code review**
   - Peer review all changes
   - Architecture review
   - Security review
   - Performance review

### Knowledge Transfer
- Share refactoring lessons learned
- Update team coding standards
- Conduct code walkthrough sessions
- Document new patterns for future use

Focus on making systematic, well-tested improvements that enhance code quality while preserving all existing functionality.