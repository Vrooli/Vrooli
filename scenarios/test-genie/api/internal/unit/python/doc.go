// Package python provides a unit test runner for Python projects.
//
// The runner detects Python projects by checking for common indicators:
//   - requirements.txt
//   - pyproject.toml
//   - setup.py
//   - tests/ directory
//
// Test execution uses pytest if available, falling back to unittest discover.
//
// Detection:
//   - Checks for python/ subdirectory or indicators in scenario root
//   - Verifies python3 or python command is available
//   - Scans for test_*.py or *_test.py files
//
// Execution:
//   - Prefers pytest for richer output and better test discovery
//   - Falls back to `python -m unittest discover` if pytest unavailable
package python
