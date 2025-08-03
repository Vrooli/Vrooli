# Document Test Fixtures

Comprehensive document test suite for validating Vrooli's resource ecosystem integration. This collection enables thorough testing of document processing, AI analysis, search indexing, storage workflows, and automation pipelines.

## 🎯 Purpose

These fixtures support testing of:
- **Unstructured-io**: Document parsing across formats
- **Ollama vision models**: Document image analysis and OCR
- **Search integration**: Content indexing and retrieval
- **Storage workflows**: File handling and metadata extraction
- **Automation pipelines**: Document processing workflows
- **Error handling**: Edge cases and problematic files

## 📁 Directory Structure

### `/office/` - Business Documents
Standard business document formats for enterprise workflows.

#### `/office/pdf/`
- `simple_text.pdf` - Basic PDF with plain text
- `formatted_report.pdf` - Complex formatting, tables, images
- `scanned_document.pdf` - Image-based PDF for OCR testing
- `password_protected.pdf` - Password: "test123"
- `large_manual.pdf` - 10+ MB technical manual
- `corrupted.pdf` - Intentionally damaged file for error testing

#### `/office/word/`
- `simple_letter.docx` - Basic Word document
- `complex_report.docx` - Tables, images, formatting
- `legacy_format.doc` - Old Word format compatibility

#### `/office/excel/`
- `sales_data.xlsx` - Spreadsheet with sample data
- `financial_report.xlsx` - Complex formulas and charts
- `legacy_format.xls` - Old Excel format compatibility

#### `/office/powerpoint/`
- `presentation.pptx` - Standard business presentation
- `image_heavy.pptx` - Image-rich slides for vision model testing

### `/structured/` - Data Formats
Structured data files for parsing and transformation testing.

- `customers.csv` - Customer data in CSV format
- `config.xml` - XML configuration file
- `settings.yaml` - YAML configuration
- `database_export.json` - Large JSON dataset
- `inventory.tsv` - Tab-separated values

### `/code/` - Programming Files
Source code and configuration files for development workflows.

#### `/code/python/`
- `web_scraper.py` - Python web scraping script
- `data_analysis.py` - Data science code example

#### `/code/javascript/`
- `frontend.js` - Frontend JavaScript code
- `api_client.ts` - TypeScript API client

#### `/code/configs/`
- `docker-compose.yml` - Infrastructure configuration
- `package.json` - NPM package configuration
- `requirements.txt` - Python dependencies

#### `/code/documentation/`
- `API.md` - Technical API documentation
- `README.rst` - ReStructuredText format example

### `/web/` - Web Documents
HTML and web-related files for browser automation testing.

- `article.html` - News article with CSS/JS
- `form.html` - HTML form for automation testing
- `dashboard.html` - Complex web interface
- `broken.html` - Invalid HTML for error handling

### `/rich_text/` - Formatted Documents
Rich text and academic document formats.

- `academic_paper.rtf` - Rich text format document
- `thesis.tex` - LaTeX academic paper
- `formatted_letter.rtf` - Business correspondence

### `/international/` - Multi-language Content
Documents in various languages and character sets.

- `chinese_document.pdf` - Chinese text content
- `arabic_text.rtf` - Arabic (RTL) text
- `mixed_languages.docx` - English + Spanish + French
- `unicode_test.txt` - Various Unicode characters

### `/edge_cases/` - Error and Boundary Testing
Files designed to test error handling and edge conditions.

- `empty.pdf` - Zero-byte file
- `minimal.docx` - Smallest valid document
- `huge_text.txt` - 50+ MB text file
- `binary_disguised.pdf` - Non-PDF with PDF extension
- `unsupported.pages` - Mac Pages format
- `network_path.url` - Internet shortcut file

### `/samples/` - Real-world Examples
Realistic business and academic documents.

- `invoice_template.pdf` - Business invoice form
- `resume_sample.docx` - HR document example
- `research_paper.pdf` - Academic research paper
- `user_manual.pdf` - Technical documentation
- `meeting_transcript.txt` - Business communications

## 🧪 Testing Scenarios

### Document Processing Pipeline
1. **Parse PDF with tables and images** → Unstructured-io
2. **Extract data from Excel** → Structured data processing
3. **OCR scanned documents** → Vision model integration
4. **Convert between formats** → Automation workflows

### AI Analysis Workflows
1. **Analyze presentation slides** → Vision model processing
2. **Summarize long documents** → LLM integration
3. **Extract entities from text** → NLP processing
4. **Classify document types** → ML classification

### Search and Indexing
1. **Index multilingual content** → Search engine testing
2. **Full-text search across formats** → Content retrieval
3. **Metadata extraction** → Document properties
4. **Similar document finding** → Vector search

### Error Handling
1. **Process corrupted files** → Graceful degradation
2. **Handle unsupported formats** → Error recovery
3. **Manage large files** → Performance testing
4. **Security validation** → Malformed input handling

## 📊 File Statistics

- **Total Files**: ~60 documents
- **Size Range**: 1KB - 50MB
- **Formats**: 15+ different file types
- **Languages**: English, Chinese, Arabic, Mixed
- **Complexity**: Simple text to complex multimedia

## 🔧 Usage

### With Unstructured-io
```bash
# Process PDF with complex formatting
curl -X POST http://localhost:8080/general/v0/general \
  -F "files=@office/pdf/formatted_report.pdf" \
  -F "strategy=fast"
```

### With Ollama Vision Models
```bash
# Analyze scanned document
curl -X POST http://localhost:11434/api/generate \
  -d '{"model": "llama3.2-vision:11b", "prompt": "Analyze this document", "images": ["office/pdf/scanned_document.pdf"]}'
```

### With Automation (n8n/Node-RED)
- Import workflows from `/workflows/` directory
- Configure file input nodes to process document batches
- Set up error handling for problematic files

## 🚀 Generation Scripts

### `generate_fixtures.sh`
Automatically creates all programmatically generated fixtures.

### `validate_fixtures.sh`
Verifies all fixtures are valid and accessible by resources.

### `clean_fixtures.sh`
Removes large/temporary files for CI environments.

## ⚖️ Legal Compliance

- **Public Domain**: All content is public domain or generated
- **No Copyright**: No copyrighted material included
- **Attribution**: Sources credited where applicable
- **Privacy**: No real personal or sensitive information

## 📈 Maintenance

### Adding New Fixtures
1. Place files in appropriate category directory
2. Update this README with file descriptions
3. Add validation tests in `validate_fixtures.sh`
4. Test with relevant resources

### Size Management
- Files > 5MB use Git LFS
- Large files can be downloaded separately
- CI environments can skip large file tests

---

**This fixture collection enables comprehensive validation of Vrooli's document processing capabilities across the entire resource ecosystem.**