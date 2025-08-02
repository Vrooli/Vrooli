#!/usr/bin/env bash
# Script to create test PDFs for Unstructured.io testing

set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
PDF_DIR="${SCRIPT_DIR}/pdfs"

echo "Creating test PDFs in $PDF_DIR..."

# Test PDF 1: Simple text document
cat > /tmp/simple_text.ps << 'EOF'
%!PS-Adobe-3.0
%%Title: Simple Text Document
%%Pages: 1
%%BoundingBox: 0 0 612 792

/Times-Roman findfont 12 scalefont setfont
50 700 moveto (Simple Text Document) show
50 650 moveto (This is a simple text document for testing basic text extraction.) show
50 600 moveto (It contains multiple paragraphs and different text styles.) show
50 550 moveto (Lorem ipsum dolor sit amet, consectetur adipiscing elit.) show
50 500 moveto (Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.) show
showpage
%%EOF
EOF
ps2pdf /tmp/simple_text.ps "$PDF_DIR/simple_text.pdf"

# Test PDF 2: Document with table
cat > /tmp/table_document.ps << 'EOF'
%!PS-Adobe-3.0
%%Title: Document with Tables
%%Pages: 1
%%BoundingBox: 0 0 612 792

/Times-Bold findfont 14 scalefont setfont
50 700 moveto (Financial Report Q1 2024) show

/Times-Roman findfont 12 scalefont setfont
50 650 moveto (Quarterly financial summary:) show

% Table headers
/Times-Bold findfont 10 scalefont setfont
50 600 moveto (Department) show
200 600 moveto (Revenue) show
300 600 moveto (Expenses) show
400 600 moveto (Profit) show

% Table data
/Times-Roman findfont 10 scalefont setfont
50 580 moveto (Sales) show
200 580 moveto ($500,000) show
300 580 moveto ($300,000) show
400 580 moveto ($200,000) show

50 560 moveto (Marketing) show
200 560 moveto ($200,000) show
300 560 moveto ($150,000) show
400 560 moveto ($50,000) show

50 540 moveto (Engineering) show
200 540 moveto ($800,000) show
300 540 moveto ($600,000) show
400 540 moveto ($200,000) show

% Draw table lines
newpath
45 615 moveto
450 615 lineto
45 530 moveto
450 530 lineto
stroke

showpage
%%EOF
EOF
ps2pdf /tmp/table_document.ps "$PDF_DIR/table_document.pdf"

# Test PDF 3: Multi-page document
cat > /tmp/multipage.ps << 'EOF'
%!PS-Adobe-3.0
%%Title: Multi-page Document
%%Pages: 3
%%BoundingBox: 0 0 612 792

%%Page: 1 1
/Times-Bold findfont 16 scalefont setfont
50 700 moveto (Multi-page Test Document) show
/Times-Roman findfont 12 scalefont setfont
50 650 moveto (Page 1 of 3) show
50 600 moveto (This document tests multi-page PDF processing.) show
50 550 moveto (Each page contains different content types.) show
showpage

%%Page: 2 2
/Times-Bold findfont 14 scalefont setfont
50 700 moveto (Chapter 1: Introduction) show
/Times-Roman findfont 12 scalefont setfont
50 650 moveto (This is the second page of the document.) show
50 600 moveto (It contains chapter information and body text.) show
50 550 moveto (The processing should maintain page boundaries.) show
showpage

%%Page: 3 3
/Times-Bold findfont 14 scalefont setfont
50 700 moveto (Conclusion) show
/Times-Roman findfont 12 scalefont setfont
50 650 moveto (This is the final page of the document.) show
50 600 moveto (All pages should be processed correctly.) show
50 550 moveto (Page numbers and metadata should be preserved.) show
showpage
%%EOF
EOF
ps2pdf /tmp/multipage.ps "$PDF_DIR/multipage.pdf"

# Test PDF 4: Document with special characters and formatting
cat > /tmp/special_chars.ps << 'EOF'
%!PS-Adobe-3.0
%%Title: Special Characters Test
%%Pages: 1
%%BoundingBox: 0 0 612 792

/Times-Roman findfont 12 scalefont setfont
50 700 moveto (Special Characters & Formatting Test) show
50 650 moveto (Unicode support: Français, Español, Deutsch) show
50 600 moveto (Mathematical: 2 + 2 = 4, x² + y² = r²) show
50 550 moveto (Symbols: © ® ™ € $ £ ¥) show
50 500 moveto (Email: test@example.com) show
50 450 moveto (URL: https://www.example.com/test?param=value&other=123) show
50 400 moveto (Phone: +1 (555) 123-4567) show
showpage
%%EOF
EOF
ps2pdf /tmp/special_chars.ps "$PDF_DIR/special_chars.pdf"

# Test PDF 5: Large document (100KB+)
cat > /tmp/large_document.ps << 'EOF'
%!PS-Adobe-3.0
%%Title: Large Document Test
%%Pages: 10
%%BoundingBox: 0 0 612 792
EOF

# Generate 10 pages of content
for i in {1..10}; do
    cat >> /tmp/large_document.ps << EOF
%%Page: $i $i
/Times-Bold findfont 14 scalefont setfont
50 700 moveto (Large Document - Page $i) show
/Times-Roman findfont 10 scalefont setfont
EOF
    
    # Add 20 lines of Lorem Ipsum per page
    y=650
    for j in {1..20}; do
        echo "50 $y moveto (Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.) show" >> /tmp/large_document.ps
        y=$((y - 25))
    done
    
    echo "showpage" >> /tmp/large_document.ps
done

echo "%%EOF" >> /tmp/large_document.ps
ps2pdf /tmp/large_document.ps "$PDF_DIR/large_document.pdf"

# Clean up temporary files
rm -f /tmp/*.ps

# List created PDFs
echo "Created test PDFs:"
ls -la "$PDF_DIR"/*.pdf

echo "✅ Test PDFs created successfully!"