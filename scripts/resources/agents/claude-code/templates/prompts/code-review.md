# Code Review Template

Comprehensive code review focusing on quality, security, and best practices.

## Variables
This template supports the following variables:
- `{files}` - Files to review (e.g., "src/**/*.ts", "lib/auth.js")
- `{focus}` - Specific review focus (e.g., "security", "performance", "maintainability")
- `{context}` - Additional context about the changes or codebase

## Template Content

You are an expert software engineer performing a comprehensive code review.

**Files to review:** {files}
**Review focus:** {focus}
**Context:** {context}

Please perform a thorough code review of the specified files, focusing on:

## 🔍 Code Quality Analysis
1. **Code structure and organization**
   - Are functions and classes well-organized?
   - Is the code modular and reusable?
   - Are there any code smells or anti-patterns?

2. **Readability and maintainability**
   - Is the code self-documenting?
   - Are variable and function names descriptive?
   - Is the logic flow clear and easy to follow?

3. **Error handling**
   - Are errors properly caught and handled?
   - Are edge cases considered?
   - Is error messaging helpful for debugging?

## 🛡️ Security Review
1. **Input validation and sanitization**
   - Are all inputs properly validated?
   - Is there protection against injection attacks?
   - Are file uploads handled safely?

2. **Authentication and authorization**
   - Are authentication mechanisms secure?
   - Is access control properly implemented?
   - Are credentials handled securely?

3. **Data protection**
   - Is sensitive data properly encrypted?
   - Are secrets kept out of the codebase?
   - Is personal data handled according to privacy standards?

## ⚡ Performance Considerations
1. **Algorithm efficiency**
   - Are there any performance bottlenecks?
   - Can any operations be optimized?
   - Are database queries efficient?

2. **Resource usage**
   - Is memory usage reasonable?
   - Are there potential memory leaks?
   - Is I/O usage optimized?

## 🧪 Testing and Documentation
1. **Test coverage**
   - Are there adequate unit tests?
   - Are edge cases tested?
   - Are tests meaningful and not just for coverage?

2. **Documentation**
   - Are complex functions documented?
   - Is the API properly documented?
   - Are README files up to date?

## 📝 Specific Recommendations

For each issue found, provide:
- **Location:** File and line number
- **Issue:** Clear description of the problem
- **Impact:** Why this matters (security, performance, maintainability)
- **Recommendation:** Specific fix or improvement
- **Priority:** High/Medium/Low

## ✨ Positive Findings

Also highlight:
- Well-written code sections
- Good architectural decisions
- Effective use of design patterns
- Clear and helpful comments

Please provide actionable feedback that will help improve the code quality while acknowledging good practices already in place.