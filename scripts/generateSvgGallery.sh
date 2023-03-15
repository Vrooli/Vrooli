#!/bin/bash
# Generates an HTML file to display all svg files in a folder
# Arguments:
# -d: Directory containing the SVG files
# -o: Output file name (default: svgGallery.html)
# -h: Show this help message
HERE=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${HERE}/prettify.sh"

# Default SVG directory
DIRECTORY="packages/shared/icons/svgs"

# Read arguments
while getopts "d:ho:" opt; do
    case $opt in
    d)
        DIRECTORY=$OPTARG
        ;;
    o)
        OUT_FILE=$OPTARG
        ;;
    h)
        echo "Usage: $0 [-d DIRECTORY] [-o OUT_FILE] [-h]"
        echo "  -d --directory: Directory containing the SVG files"
        echo "  -o --out-file: Output file name (default: svgGallery.html)"
        echo "  -h --help: Show this help message"
        exit 0
        ;;
    \?)
        error "Invalid option: -$OPTARG"
        exit 1
        ;;
    :)
        error "Option -$OPTARG requires an argument."
        exit 1
        ;;
    esac
done

# If no output file is provided, use the default
if [ -z "$OUT_FILE" ]; then
    OUT_FILE="svgGallery.html"
fi

# Check if the folder exists
if [ ! -d "$DIRECTORY" ]; then
    error "Error: Directory $DIRECTORY does not exist."
    exit 1
fi

header "Generating SVG gallery in $DIRECTORY/$OUT_FILE..."

# Create the output file with initial HTML structure.
# Should be saved one level above the SVG files folder
OUT_FILE="$DIRECTORY/../$OUT_FILE"
cat >$OUT_FILE << EOL
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>SVG Gallery</title>
<style>
  body { font-family: Arial, sans-serif; background-color: #f0f0f0; }
  h1 { text-align: center; }
  .container { display: flex; flex-wrap: wrap; justify-content: center; }
  .item { padding: 15px; text-align: center; }
  .item img { max-width: 100%; height: auto; }
</style>
<script>
  function resizeIcons() {
    var size = document.getElementById("sizeSlider").value;
    var icons = document.getElementsByTagName("img");
    for (var i = 0; i < icons.length; i++) {
      icons[i].style.width = size + "px";
    }
  }
</script>
</head>
<body>
<h1>SVG Gallery</h1>
<div style="text-align: center;">
  <label for="sizeSlider">Icon size:</label>
  <input type="range" id="sizeSlider" name="sizeSlider" min="50" max="300" value="100" onchange="resizeIcons()" style="width: 50%;">
</div>
<div class="container">
EOL

# Loop through all SVG files in the folder and add them to the HTML
for svg in "$DIRECTORY"/*.svg; do
    if [ -f "$svg" ]; then
        FILENAME=$(basename "$svg")
        RELATIVE_PATH=$(basename "$DIRECTORY")/$FILENAME
        cat >>$OUT_FILE <<EOL
  <div class="item">
    <img src="$RELATIVE_PATH" alt="$FILENAME">
    <p>$FILENAME</p>
  </div>
EOL
    fi
done

# Close the HTML structure
cat >>$OUT_FILE <<EOL
</div>
</body>
</html>
EOL

success "SVG gallery has been generated in $OUT_FILE"
