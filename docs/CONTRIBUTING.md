# Contributing to Vrooli

Thank you for your interest in contributing to Vrooli! This document provides guidelines and standards for contributing to make the process smooth and consistent for everyone.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Process](#development-process)
- [Coding Standards](#coding-standards)
- [Testing Requirements](#testing-requirements)
- [Pull Request Process](#pull-request-process)
- [Commit Message Guidelines](#commit-message-guidelines)
- [Review Process](#review-process)
- [Documentation Standards](#documentation-standards)
- [Security Guidelines](#security-guidelines)
- [Community](#community)

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct:

- **Be Respectful**: Treat all contributors with respect and professionalism
- **Be Inclusive**: Welcome contributors of all backgrounds and experience levels
- **Be Constructive**: Provide helpful feedback and accept criticism gracefully
- **Be Collaborative**: Work together to solve problems and improve the project

## Getting Started

### Prerequisites

Before contributing, ensure you have:

1. Read the [README.md](/README.md) and [CLAUDE.md](/CLAUDE.md)
2. Set up your development environment following [/docs/GETTING_STARTED.md](/docs/GETTING_STARTED.md)
3. Familiarized yourself with the [architecture documentation](/docs/ARCHITECTURE_OVERVIEW.md)
4. Reviewed recent commits and issues to understand current development

### First-Time Contributors

1. **Find an Issue**: Look for issues labeled `good-first-issue` or `help-wanted`
2. **Comment on the Issue**: Let us know you're working on it
3. **Ask Questions**: Don't hesitate to ask for clarification or guidance
4. **Start Small**: Consider documentation improvements or bug fixes for your first contribution

## Development Process

### 1. Fork and Clone

```bash
# Fork the repository on GitHub, then:
git clone https://github.com/YOUR_USERNAME/Vrooli.git
cd Vrooli
git remote add upstream https://github.com/Vrooli/Vrooli.git
```

### 2. Create a Feature Branch

```bash
# Update your fork
git checkout dev
git pull upstream dev

# Create your feature branch
git checkout -b feature/your-feature-name
```

### 3. Set Up Development Environment

```bash
# Install dependencies
pnpm install

# Start development environment  
vrooli develop
```

### 4. Make Your Changes

Follow our coding standards and ensure all tests pass before committing.

## Coding Standards

### TypeScript Guidelines

1. **Type Safety**
   - Use explicit types, avoid `any`
   - Leverage TypeScript's strict mode
   - Define interfaces for complex objects

2. **Import Requirements**
   ```typescript
   // ALWAYS use .js extension in imports
   import { foo } from "./bar.js";
   
   // For monorepo packages, use package name only
   import { types } from "@vrooli/shared";
   ```

3. **Naming Conventions**
   - Components: PascalCase (e.g., `UserProfile`)
   - Functions/Variables: camelCase (e.g., `getUserData`)
   - Constants: UPPER_SNAKE_CASE (e.g., `MAX_RETRIES`)
   - Files: Match the main export (e.g., `UserProfile.tsx`)

4. **Code Organization**
   - Keep files focused and under 300 lines when possible
   - Group related functionality
   - Use barrel exports for clean imports

### React Guidelines

1. **Component Structure**
   - Use functional components with hooks
   - Keep components focused on a single responsibility
   - Extract complex logic into custom hooks

2. **State Management**
   - Use appropriate state solutions (local state, context, stores)
   - Avoid unnecessary re-renders
   - Document complex state logic

### Architecture Principles

1. **Three-Tier Architecture**: Respect the coordination/process/execution intelligence model
2. **Event-Driven Design**: Use the event bus for component communication
3. **Emergent Capabilities**: Don't hard-code what can emerge from agent intelligence
4. **Type Safety**: Maintain type safety across package boundaries

## Testing Requirements

### Test Coverage

- **Minimum**: 80% code coverage for new code
- **Critical Paths**: 100% coverage for authentication, payment, and security code
- **Integration Tests**: Use testcontainers for Redis/PostgreSQL (never mock core infrastructure)

### Writing Tests

```typescript
// Test file naming: ComponentName.test.ts or functionName.test.ts
// Place in __test directories (NOT __tests)

describe('Component/Function Name', () => {
  it('should [expected behavior] when [condition]', () => {
    // Arrange
    // Act
    // Assert
  });
});
```

### Running Tests

```bash
# Run all tests
pnpm test

# Run scenario tests
vrooli scenario test <scenario-name>

# Run resource tests
vrooli test resources

# Watch mode for development
vrooli test --watch
```

## Pull Request Process

### Before Submitting

1. **Update from upstream**
   ```bash
   git pull upstream dev
   git rebase upstream/dev
   ```

2. **Run all checks**
   ```bash
   # Type checking
   pnpm run type-check
   
   # Linting
   pnpm run lint
   
   # Tests
   pnpm test
   ```

3. **Update documentation** if you've:
   - Added new features
   - Changed APIs
   - Modified configuration options

### PR Guidelines

1. **Title Format**: `type(scope): description`
   - Types: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
   - Example: `feat(server): add rate limiting to API endpoints`

2. **Description Template**:
   ```markdown
   ## Description
   Brief description of changes
   
   ## Type of Change
   - [ ] Bug fix
   - [ ] New feature
   - [ ] Breaking change
   - [ ] Documentation update
   
   ## Testing
   - [ ] Unit tests pass
   - [ ] Integration tests pass
   - [ ] Manual testing completed
   
   ## Checklist
   - [ ] Code follows style guidelines
   - [ ] Self-review completed
   - [ ] Documentation updated
   - [ ] No new warnings
   ```

3. **PR Size**: Keep PRs focused and under 400 lines when possible

## Commit Message Guidelines

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
type(scope): subject

body

footer
```

### Examples

```bash
# Feature
feat(ui): add dark mode toggle to settings

# Bug fix
fix(server): resolve memory leak in WebSocket handler

# Documentation
docs(api): update authentication flow diagram

# Performance
perf(jobs): optimize embedding generation batch size

# Breaking change
feat(api)!: change user endpoint response format

BREAKING CHANGE: The user endpoint now returns a nested structure
```

## Review Process

### For Contributors

1. **Respond to Feedback**: Address reviewer comments promptly
2. **Update Your PR**: Push fixes as new commits (we'll squash on merge)
3. **Re-request Review**: Use GitHub's re-request feature when ready

### For Reviewers

1. **Be Constructive**: Suggest improvements, don't just criticize
2. **Check for**:
   - Code quality and standards compliance
   - Test coverage and quality
   - Documentation updates
   - Security implications
   - Performance impact

3. **Use Review Comments**:
   - ✅ Approve: Ready to merge
   - 💬 Comment: Suggestions but not blocking
   - 🚫 Request Changes: Must be addressed before merge

## Documentation Standards

### When to Update Documentation

Update documentation when you:
- Add new features or APIs
- Change existing behavior
- Fix documentation errors
- Improve clarity or examples

### Documentation Guidelines

1. **Location**: Place docs in the appropriate `/docs/` subdirectory
2. **Format**: Use Markdown with clear headings and examples
3. **Diagrams**: Use Mermaid for technical diagrams
4. **Examples**: Include practical, runnable examples
5. **Cross-references**: Link to related documentation

## Security Guidelines

### Security Considerations

1. **Never commit**:
   - Passwords, API keys, or secrets
   - Personal data or PII
   - Security vulnerabilities

2. **Always**:
   - Validate and sanitize user input
   - Use parameterized queries
   - Follow OWASP guidelines
   - Report security issues privately

### Reporting Security Issues

For security vulnerabilities, please email security@vrooli.com instead of creating a public issue.

## Community

### Getting Help

- **Discord**: Join our [Discord server](https://discord.gg/vrooli) for real-time help
- **Discussions**: Use GitHub Discussions for questions and ideas
- **Issues**: Report bugs and request features through GitHub Issues

### Recognition

We value all contributions! Contributors are:
- Listed in our [Contributors](https://github.com/Vrooli/Vrooli/graphs/contributors) page
- Mentioned in release notes for significant contributions
- Invited to our contributor recognition program

## Questions?

If you have questions about contributing, please:
1. Check existing documentation
2. Search closed issues and discussions
3. Ask in our Discord #contributing channel
4. Create a new discussion if needed

Thank you for contributing to Vrooli! Together, we're building the future of AI-powered productivity. 🚀