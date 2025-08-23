# Office Document Test Fixtures

This directory contains 12 real-world office documents organized by type and source category for comprehensive testing of Vrooli's resource ecosystem.

## Collection Summary

### Document Types
- **PDF**: 8 documents (67%)
- **Excel**: 3 documents (25%) 
- **Word**: 3 documents (25%)
- **PowerPoint**: 0 documents (0%)

### Source Categories
- **Government**: 4 documents (33%)
- **Educational**: 6 documents (50%)
- **Corporate**: 1 document (8%)
- **International**: 1 document (8%)

## Directory Structure

```
office/
├── pdf/
│   ├── government/
│   │   ├── census_income_report.pdf (26K)
│   │   ├── constitution_annotated.pdf (19M)
│   │   ├── energy_annual_report.pdf (53K)
│   │   └── gao_report_sample.pdf (19K)
│   ├── corporate/
│   │   └── berkshire_hathaway_annual_letter.pdf (120K)
│   └── international/
│       └── un_sustainable_development_goals.pdf (369K)
├── excel/
│   └── educational/
│       ├── cmu_test_data.xls (16K)
│       ├── ou_sample_data.xlsx (29K)
│       └── usc_sample_data.xls (345K)
└── word/
    └── educational/
        ├── ceu_sample.doc (199 bytes - edge case)
        ├── iup_test_document.docx (51K)
        └── mtsac_word_template.docx (107K)
```

## Document Details

### Government PDFs
- **census_income_report.pdf**: US Census Bureau income analysis report
- **constitution_annotated.pdf**: Annotated US Constitution (large 19MB document)
- **energy_annual_report.pdf**: Department of Energy annual report
- **gao_report_sample.pdf**: Government Accountability Office audit report sample

### Educational Documents
- **cmu_test_data.xls**: Carnegie Mellon University sample dataset
- **ou_sample_data.xlsx**: University of Oklahoma sample data file
- **usc_sample_data.xls**: USC sample data (345K - substantial dataset)
- **ceu_sample.doc**: Central European University document (edge case - 199 bytes)
- **iup_test_document.docx**: Indiana University of Pennsylvania test document
- **mtsac_word_template.docx**: Mt. San Antonio College Word template

### Corporate Documents
- **berkshire_hathaway_annual_letter.pdf**: Warren Buffett's annual shareholder letter

### International Documents
- **un_sustainable_development_goals.pdf**: UN Sustainable Development Goals document

## Testing Use Cases

### Document Processing Tests
- **Text extraction**: All PDF and Word documents
- **Data parsing**: Excel files with various formats and sizes
- **Large document handling**: constitution_annotated.pdf (19MB)
- **Edge case handling**: ceu_sample.doc (corrupted/minimal content)

### AI Analysis Tests
- **Government document analysis**: Policy extraction, regulatory content
- **Financial document analysis**: Berkshire Hathaway letter
- **Academic content processing**: University documents and datasets
- **International content**: UN multilingual document processing

### Resource Integration Tests
- **Unstructured-io**: Document parsing and extraction
- **Ollama**: Content summarization and analysis
- **Search indexing**: Full-text search across document types
- **Automation workflows**: Document processing pipelines

## Validation

Run the validation script to test all documents:
```bash
./validate_fixtures.sh office
```

The script verifies:
- File integrity and accessibility
- Format validation for each document type
- Integration with available Vrooli resources
- Performance testing with various document sizes

## Notes

- All documents are from public domain or open access sources
- Document sizes range from 199 bytes to 19MB for comprehensive testing
- Collection includes both structured data (Excel) and unstructured content (PDF, Word)
- Real-world complexity ensures production-ready testing scenarios