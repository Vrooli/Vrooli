#!/bin/sh
# Automatically generates an .swcrc file based on the environment. 
# Customizes the "exclude" and "sourceMaps" properties based on the environment.

# Check if an argument was provided
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <environment>"
    exit 1
fi

# More of the .swcrc configuration
cat << EOF
{
    "exclude": [
$( [ "$1" != "test" ] && echo '        "/.*__mocks__.*/",' )
        "/.*\\\/types\\\.ts$/",
        "/.*\\\/types\\\.d\\\.ts$/"
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
    "sourceMaps": $( [ "$1" = "development" ] && echo "true" || echo false )
}
EOF
