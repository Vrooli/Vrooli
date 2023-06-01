#!/bin/bash

shopt -s nullglob

for file in models/base/*.ts; do
    filename=$(basename "$file" ".ts")
    first_letter=$(echo "${filename:0:1}" | tr '[:lower:]' '[:upper:]')
    remainder=$(echo "${filename:1}")
    cap_filename="$first_letter$remainder"

    formatname=${cap_filename}"Format"

    # Extract the __typename const
    typename=$(grep "^const __typename" "$file")

    # Extract the format section
    format=$(awk '/format: {/{flag=1; count=1; next} flag && /{/{count+=gsub(/{/,"")} flag && /}/{count-=gsub(/}/,"")} flag && count>=0{print}' "$file")

    # Write to the format file
    echo "import { Formatter } from \"../types\";" >>"models/format/$filename.ts"
    echo "" >>"models/format/$filename.ts"
    echo "$typename" >>"models/format/$filename.ts"
    echo "export const $formatname: Formatter<Model${cap_filename}Logic> = {" >>"models/format/$filename.ts"
    echo "$format" >>"models/format/$filename.ts"
    echo "};" >>"models/format/$filename.ts"
done
