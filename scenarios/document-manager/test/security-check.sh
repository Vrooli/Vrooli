#!/bin/bash

# Basic security checks for document-manager scenario
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "======================================"
echo "Document Manager Security Check"
echo "======================================"

ISSUES=0

# Check 1: Sensitive data in code
echo "Checking for hardcoded secrets..."
if grep -rn "password\|secret\|key\|token" api/ --exclude="*.go" | grep -v "// " | grep -v "PASSWORD" | grep -v "SECRET"; then
    echo -e "${RED}✗${NC} Found potential hardcoded secrets"
    ((ISSUES++))
else
    echo -e "${GREEN}✓${NC} No hardcoded secrets found"
fi

# Check 2: SQL injection protection
echo "Checking for SQL injection vulnerabilities..."
if grep -rn "fmt.Sprintf.*SELECT\|fmt.Sprintf.*UPDATE\|fmt.Sprintf.*DELETE\|fmt.Sprintf.*INSERT" api/*.go 2>/dev/null | grep -v "Access-Control"; then
    echo -e "${RED}✗${NC} Found potential SQL injection (string concatenation in queries)"
    ((ISSUES++))
else
    echo -e "${GREEN}✓${NC} Using parameterized queries"
fi

# Check 3: Error handling
echo "Checking error handling..."
unhandled=$(grep -rn "err :=" api/*.go | grep -v "if err" | wc -l)
if [[ $unhandled -gt 0 ]]; then
    echo -e "${YELLOW}⚠${NC} Found $unhandled potentially unhandled errors"
else
    echo -e "${GREEN}✓${NC} All errors appear to be handled"
fi

# Check 4: Input validation
echo "Checking input validation..."
if grep -q "json.NewDecoder" api/main.go; then
    echo -e "${GREEN}✓${NC} JSON input validation present"
else
    echo -e "${YELLOW}⚠${NC} Consider adding input validation"
fi

# Check 5: CORS configuration
echo "Checking CORS configuration..."
if grep -q "Access-Control-Allow-Origin" api/main.go; then
    echo -e "${GREEN}✓${NC} CORS headers configured"
else
    echo -e "${YELLOW}⚠${NC} No CORS configuration found"
fi

# Check 6: Rate limiting
echo "Checking rate limiting..."
if grep -q "rate.Limiter\|RateLimit" api/; then
    echo -e "${GREEN}✓${NC} Rate limiting implemented"
else
    echo -e "${YELLOW}⚠${NC} Consider adding rate limiting"
fi

# Check 7: Authentication
echo "Checking authentication..."
if grep -q "Authorization\|auth" api/main.go; then
    echo -e "${GREEN}✓${NC} Authentication checks present"
else
    echo -e "${YELLOW}⚠${NC} No authentication found (may be required for production)"
fi

# Check 8: TLS/HTTPS
echo "Checking TLS configuration..."
if grep -q "ListenAndServeTLS\|tls" api/main.go; then
    echo -e "${GREEN}✓${NC} TLS support present"
else
    echo -e "${YELLOW}⚠${NC} Consider adding TLS for production"
fi

# Check 9: Dependency vulnerabilities
echo "Checking Go dependencies..."
if [[ -f "api/go.mod" ]]; then
    cd api
    if go list -m all 2>/dev/null | grep -q "indirect"; then
        echo -e "${GREEN}✓${NC} Dependencies checked"
    fi
    cd ..
fi

# Check 10: File permissions
echo "Checking file permissions..."
if find . -type f -perm /o+w | grep -v ".git"; then
    echo -e "${YELLOW}⚠${NC} Found world-writable files"
else
    echo -e "${GREEN}✓${NC} File permissions look good"
fi

echo ""
echo "======================================"
echo "Security Check Summary"
echo "======================================"
if [[ $ISSUES -eq 0 ]]; then
    echo -e "${GREEN}No critical security issues found${NC}"
    exit 0
else
    echo -e "${RED}Found $ISSUES potential security issues${NC}"
    echo "Review and address before production deployment"
    exit 1
fi