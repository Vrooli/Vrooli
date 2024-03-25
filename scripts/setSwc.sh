#!/bin/sh
# Automatically generates an .swcrc file based on the environment

# Check if an argument was provided
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <environment>"
    exit 1
fi

# Start of the .swcrc configuration
cat << 'EOF'
{
    "exclude": [
EOF

# Conditional exclude for non-tests
if [ "$1" != "test" ]; then
cat << 'EOF'
        "/.*__mocks__.*/",
EOF
fi

# More of the .swcrc configuration
cat << 'EOF'
        "/.*\\/types\\.ts$/",
        "/.*\\/types\\.d\\.ts$/"
    ],
    "jsc": {
        "minify": {
            "compress": true,
            "mangle": true
        },
        "parser": {
            "syntax": "typescript",
            "tsx": true,
            "decorators": true,
            "dynamicImport": true
        },
        "target": "es2017"
    },
    "module": {
        "type": "es6"
    },
EOF

# Conditional sourceMaps configuration
if [ "$1" = "development" ]; then
cat << 'EOF'
        "sourceMaps": true
EOF
else
cat << 'EOF'
        "sourceMaps": false
EOF
fi

cat << 'EOF'
}
EOF
