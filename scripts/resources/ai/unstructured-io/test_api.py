#!/usr/bin/env python3
"""
Unstructured.io API Test Script

This script demonstrates how to interact with the Unstructured.io API
for document processing and provides examples of various use cases.
"""

import json
import os
import sys
import time
from typing import Dict, List, Any, Optional
import requests
from pathlib import Path

# Configuration
BASE_URL = os.getenv("UNSTRUCTURED_IO_BASE_URL", "http://localhost:11450")
TIMEOUT = 30


class UnstructuredIOClient:
    """Client for interacting with Unstructured.io API"""
    
    def __init__(self, base_url: str = BASE_URL):
        self.base_url = base_url.rstrip("/")
        self.session = requests.Session()
    
    def check_health(self) -> bool:
        """Check if the service is healthy"""
        try:
            response = self.session.get(
                f"{self.base_url}/healthcheck",
                timeout=5
            )
            return response.status_code == 200
        except Exception as e:
            print(f"Health check failed: {e}")
            return False
    
    def process_document(
        self,
        file_path: str,
        strategy: str = "hi_res",
        languages: List[str] = None,
        include_page_breaks: bool = True,
        **kwargs
    ) -> Dict[str, Any]:
        """Process a document and return structured data"""
        if not os.path.exists(file_path):
            raise FileNotFoundError(f"File not found: {file_path}")
        
        # Prepare form data
        files = {"files": open(file_path, "rb")}
        data = {
            "strategy": strategy,
            "include_page_breaks": str(include_page_breaks).lower(),
            "skip_infer_table_types": "[]",
            "encoding": "utf-8"
        }
        
        if languages:
            data["languages"] = ",".join(languages)
        
        # Add any additional parameters
        data.update(kwargs)
        
        try:
            response = self.session.post(
                f"{self.base_url}/general/v0/general",
                files=files,
                data=data,
                timeout=TIMEOUT
            )
            response.raise_for_status()
            return response.json()
        except requests.exceptions.RequestException as e:
            print(f"Error processing document: {e}")
            if hasattr(e, 'response') and e.response is not None:
                print(f"Response: {e.response.text}")
            raise
        finally:
            files["files"].close()
    
    def process_text(self, text: str, **kwargs) -> Dict[str, Any]:
        """Process raw text by saving to temp file"""
        import tempfile
        with tempfile.NamedTemporaryFile(mode='w', suffix='.txt', delete=False) as f:
            f.write(text)
            temp_path = f.name
        
        try:
            return self.process_document(temp_path, **kwargs)
        finally:
            os.unlink(temp_path)


def print_elements(elements: List[Dict[str, Any]], max_text_length: int = 100):
    """Pretty print document elements"""
    print(f"\nFound {len(elements)} elements:")
    print("-" * 80)
    
    for i, element in enumerate(elements):
        elem_type = element.get("type", "Unknown")
        text = element.get("text", "")
        
        # Truncate long text
        if len(text) > max_text_length:
            text = text[:max_text_length] + "..."
        
        print(f"{i+1}. [{elem_type}] {text}")
        
        # Show metadata if present
        metadata = element.get("metadata", {})
        if metadata:
            print(f"   Metadata: {json.dumps(metadata, indent=2)}")
    
    print("-" * 80)


def test_basic_processing():
    """Test basic document processing"""
    print("\n=== Test 1: Basic Text Processing ===")
    
    client = UnstructuredIOClient()
    
    # Create test document
    test_text = """
# Test Document

This is a test document for Unstructured.io processing.

## Features

- Document parsing
- Table extraction
- OCR capabilities

## Sample Table

| Feature | Status | Description |
|---------|--------|-------------|
| PDF     | ✓      | Full support |
| DOCX    | ✓      | Full support |
| Images  | ✓      | With OCR |

## Conclusion

This demonstrates the API capabilities.
"""
    
    try:
        elements = client.process_text(test_text, strategy="fast")
        print_elements(elements)
        
        # Analyze element types
        element_types = {}
        for elem in elements:
            elem_type = elem.get("type", "Unknown")
            element_types[elem_type] = element_types.get(elem_type, 0) + 1
        
        print("\nElement type summary:")
        for elem_type, count in element_types.items():
            print(f"  {elem_type}: {count}")
        
        return True
    except Exception as e:
        print(f"Test failed: {e}")
        return False


def test_strategies():
    """Test different processing strategies"""
    print("\n=== Test 2: Processing Strategies ===")
    
    client = UnstructuredIOClient()
    strategies = ["fast", "hi_res", "auto"]
    
    test_text = "This is a simple test document with some content."
    
    for strategy in strategies:
        print(f"\nTesting strategy: {strategy}")
        start_time = time.time()
        
        try:
            elements = client.process_text(test_text, strategy=strategy)
            duration = time.time() - start_time
            
            print(f"  ✓ Processed in {duration:.2f} seconds")
            print(f"  Elements: {len(elements)}")
        except Exception as e:
            print(f"  ✗ Failed: {e}")


def test_output_formats():
    """Test converting output to different formats"""
    print("\n=== Test 3: Output Format Conversion ===")
    
    client = UnstructuredIOClient()
    
    test_text = """
# Markdown Conversion Test

This document tests conversion capabilities.

## List Example
- Item 1
- Item 2
- Item 3

## Text Section
This is a paragraph of narrative text that should be preserved.
"""
    
    try:
        elements = client.process_text(test_text)
        
        # Convert to markdown
        print("\nMarkdown output:")
        print("-" * 40)
        for elem in elements:
            elem_type = elem.get("type", "")
            text = elem.get("text", "")
            
            if elem_type == "Title":
                print(f"# {text}")
            elif elem_type == "Header":
                print(f"## {text}")
            elif elem_type == "ListItem":
                print(f"- {text}")
            elif elem_type == "NarrativeText":
                print(f"\n{text}\n")
            elif elem_type == "PageBreak":
                print("\n---\n")
            else:
                print(text)
        
        # Convert to plain text
        print("\nPlain text output:")
        print("-" * 40)
        for elem in elements:
            print(elem.get("text", ""))
        
        return True
    except Exception as e:
        print(f"Test failed: {e}")
        return False


def test_table_extraction():
    """Test table extraction capabilities"""
    print("\n=== Test 4: Table Extraction ===")
    
    client = UnstructuredIOClient()
    
    # HTML with table
    html_content = """
<!DOCTYPE html>
<html>
<body>
    <h1>Sales Report</h1>
    <table border="1">
        <tr>
            <th>Product</th>
            <th>Q1 Sales</th>
            <th>Q2 Sales</th>
            <th>Total</th>
        </tr>
        <tr>
            <td>Widget A</td>
            <td>$10,000</td>
            <td>$15,000</td>
            <td>$25,000</td>
        </tr>
        <tr>
            <td>Widget B</td>
            <td>$8,000</td>
            <td>$12,000</td>
            <td>$20,000</td>
        </tr>
    </table>
</body>
</html>
"""
    
    import tempfile
    with tempfile.NamedTemporaryFile(mode='w', suffix='.html', delete=False) as f:
        f.write(html_content)
        temp_path = f.name
    
    try:
        elements = client.process_document(temp_path, strategy="hi_res")
        
        # Find tables
        tables = [elem for elem in elements if elem.get("type") == "Table"]
        
        print(f"\nFound {len(tables)} table(s)")
        for i, table in enumerate(tables):
            print(f"\nTable {i+1}:")
            print(table.get("text", ""))
        
        os.unlink(temp_path)
        return len(tables) > 0
    except Exception as e:
        print(f"Test failed: {e}")
        os.unlink(temp_path)
        return False


def test_error_handling():
    """Test error handling"""
    print("\n=== Test 5: Error Handling ===")
    
    client = UnstructuredIOClient()
    
    # Test with non-existent file
    print("\nTesting with non-existent file:")
    try:
        client.process_document("/tmp/non_existent_file.txt")
        print("  ✗ Should have raised FileNotFoundError")
    except FileNotFoundError:
        print("  ✓ Correctly raised FileNotFoundError")
    except Exception as e:
        print(f"  ✗ Unexpected error: {e}")
    
    # Test with invalid strategy
    print("\nTesting with invalid strategy:")
    try:
        elements = client.process_text("Test", strategy="invalid_strategy")
        # API might accept it, so just log the result
        print(f"  API accepted invalid strategy (returned {len(elements)} elements)")
    except Exception as e:
        print(f"  API rejected invalid strategy: {e}")


def display_api_info():
    """Display API information"""
    print("\n" + "=" * 80)
    print("Unstructured.io API Information")
    print("=" * 80)
    print(f"Base URL: {BASE_URL}")
    print("\nEndpoints:")
    print(f"  - Health Check: GET {BASE_URL}/healthcheck")
    print(f"  - Process Document: POST {BASE_URL}/general/v0/general")
    print("\nSupported Formats:")
    print("  - Documents: PDF, DOCX, DOC, TXT, RTF, ODT, MD, RST, HTML, XML, EPUB")
    print("  - Spreadsheets: XLSX, XLS")
    print("  - Presentations: PPTX, PPT")
    print("  - Images (OCR): PNG, JPG, JPEG, TIFF, BMP, HEIC")
    print("  - Email: EML, MSG")
    print("\nProcessing Strategies:")
    print("  - fast: Quick processing for simple documents")
    print("  - hi_res: High-resolution with advanced features (default)")
    print("  - auto: Automatically select best strategy")
    print("=" * 80)


def main():
    """Main test execution"""
    print("🧪 Unstructured.io API Test Suite (Python)")
    print("=" * 80)
    
    # Check service availability
    client = UnstructuredIOClient()
    print(f"\nChecking service at {BASE_URL}...")
    
    if not client.check_health():
        print("❌ Service is not available!")
        print("Make sure Unstructured.io is running:")
        print("  ./scripts/resources/ai/unstructured-io/manage.sh --action status")
        sys.exit(1)
    
    print("✅ Service is healthy")
    
    # Run tests
    tests_passed = 0
    total_tests = 5
    
    if test_basic_processing():
        tests_passed += 1
    
    test_strategies()  # Information only
    tests_passed += 1
    
    if test_output_formats():
        tests_passed += 1
    
    if test_table_extraction():
        tests_passed += 1
    
    test_error_handling()  # Always passes
    tests_passed += 1
    
    # Display summary
    print("\n" + "=" * 80)
    print(f"Test Summary: {tests_passed}/{total_tests} tests passed")
    print("=" * 80)
    
    # Display API info
    display_api_info()
    
    return 0 if tests_passed == total_tests else 1


if __name__ == "__main__":
    sys.exit(main())