#!/bin/bash
# Automatically generates an .swcrc file based on the environment

# Check if an argument was provided
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <environment>"
    exit 1
fi

# Start of the .swcrc configuration
echo '{
    "exclude": ['

# Conditional exclude for non-tests
if [ "$1" != "test" ]; then
    echo '        "/.*__mocks__.*/",'
fi

# More of the .swcrc configuration
echo '        "/.*\\/types\\.ts$/",
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
    },'

# Conditional sourceMaps configuration
if [ "$1" = "development" ]; then
    echo '    "sourceMaps": true'
else
    echo '    "sourceMaps": false'
fi

echo '}'
