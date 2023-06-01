#!/bin/bash

shopt -s nullglob

# Create or empty models/base/types.ts
>models/base/types.ts

for file in models/base/*.ts; do
    filename=$(basename "$file" .ts)

    # Make the first letter of the filename uppercase
    typename=$(echo "${filename^}ModelLogic")

    # Extract the generic parameter of ModelLogic and turn it into its own type
    awk -v typename="$typename" '
        /ModelLogic</ { 
            print "export type " typename " = {";
            flag=1; 
            sub(/.*ModelLogic</, ""); 
            gsub(/{/, ""); 
        }
        flag { 
            print; 
            if (/}/) flag=0; 
        }
    ' "$file" >>"models/base/types.ts"

    # Add an import statement for the type right after the existing imports
    awk -v typename="$typename" -v filename="$filename" '
        BEGIN { printflag=0; }
        /^import/ { 
            print;
            if (!printflag) {
                print "import { " typename " } from \"./types\";";
                printflag=1;
            }
        }
        !/^import/ {
            print;
        }
    ' "$file" >"models/base/temp_$filename.ts"

    # Move the temp file to the original file
    mv "models/base/temp_$filename.ts" "$file"
done
