#!/bin/bash
# Moves @codemirror languages to the public directory,
# so that they can be lazy loaded by the CodeMirror editor.
HERE=$(dirname $0)
. "${HERE}/prettify.sh"

SOURCE_DIR="${HERE}/../node_modules/@codemirror/"
DEST_DIR="${HERE}/../packages/ui/public/codemirror"

# Check if the source directory exists
if [ ! -d "$SOURCE_DIR" ]; then
    error "@codemirror source directory not found: $SOURCE_DIR"
    exit 1
fi

# Create the destination directory
mkdir -p "$DEST_DIR"

# Copy dist/index.js from language folders in @codemirror directory
info "Copying dist/index.js from language folders in $SOURCE_DIR to $DEST_DIR"
for lang_folder in "$SOURCE_DIR"lang-*; do
    if [ -d "$lang_folder" ]; then
        lang_name=$(basename "$lang_folder")
        # Remove the "lang-" prefix from the destination folder name
        dest_folder="$DEST_DIR/${lang_name#lang-}"
        # Create the destination folder if it doesn't exist
        mkdir -p "$dest_folder"
        # Copy dist/index.js to the destination folder and rename it
        cp "$lang_folder/dist/index.js" "$dest_folder/${lang_name#lang-}.js"
    fi
done

# Copy .cjs files from @codemirror/legacy-modes/mode
legacy_modes_source="${SOURCE_DIR}legacy-modes/mode"
info "Copying .cjs files from $legacy_modes_source to $DEST_DIR"
for mode_file in "$legacy_modes_source"/*.cjs; do
    if [ -f "$mode_file" ]; then
        mode_name=$(basename "$mode_file" .cjs)
        dest_folder="$DEST_DIR/$mode_name"
        mkdir -p "$dest_folder"
        cp "$mode_file" "$dest_folder/$mode_name.cjs"
    fi
done

# Create a README file
readme_content="This code is taken from the @codemirror package in node_modules for the purpose of lazy-loading language modes and legacy modes in the CodeMirror editor.

The contents of this directory are used to enable on-demand loading of CodeMirror language modes and legacy modes when editing code. These files are provided by the CodeMirror project.

For more information, please refer to the CodeMirror documentation: https://codemirror.net/"
echo "$readme_content" >"$DEST_DIR/README.md"

success "Copy complete."
