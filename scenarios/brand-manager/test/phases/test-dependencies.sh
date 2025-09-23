#!/bin/bash

set -e

echo "=== Testing Dependencies ==="

# Check dependencies
go mod tidy &amp;&amp; echo "Dependencies tidy passed."

echo "Dependencies tests passed."