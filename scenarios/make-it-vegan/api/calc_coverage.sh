#!/bin/bash
# Calculate coverage for production files only
go tool cover -func=coverage.out 2>/dev/null | \
grep -E "^make-it-vegan/(main|cache|vegan_database)\.go:" | \
awk '{
    split($2, a, "%"); 
    cov = a[1]; 
    total += cov; 
    count++
} 
END {
    if (count > 0) 
        printf "Production code coverage: %.1f%%\n", total/count
}'
