#!/usr/bin/env python3
"""
Test patterns for Claude Code sandbox testing
Includes various code quality issues for testing analysis capabilities
"""

import os
import json
from typing import List, Dict, Any

class DataProcessor:
    def __init__(self):
        self.data = []
    
    def load_data(self, filename: str) -> List[Dict[str, Any]]:
        """Load JSON data from file - missing error handling"""
        with open(filename, 'r') as f:
            self.data = json.load(f)
        return self.data
    
    def process_records(self, records: List[Dict]) -> Dict[str, int]:
        """Process records with potential performance issues"""
        results = {}
        
        # Inefficient nested loops - O(n²) complexity
        for record in records:
            for key in record:
                if key in results:
                    results[key] += 1
                else:
                    results[key] = 1
        
        return results
    
    def save_results(self, results: Dict, output_path: str):
        """Save results - potential security issue with path"""
        # No validation of output_path - could write anywhere
        full_path = os.path.join("/tmp", output_path)
        with open(full_path, 'w') as f:
            json.dump(results, f)

def unsafe_execute(command: str) -> str:
    """UNSAFE: Executes system commands - for testing only"""
    # Security vulnerability: command injection
    import subprocess
    result = subprocess.run(command, shell=True, capture_output=True, text=True)
    return result.stdout

def calculate_metrics(data: List[float]) -> Dict[str, float]:
    """Calculate statistics with potential division by zero"""
    total = sum(data)
    count = len(data)
    
    # Bug: No check for empty list
    average = total / count
    
    # More calculations...
    return {
        "total": total,
        "count": count,
        "average": average
    }

# Global variable - bad practice
GLOBAL_CONFIG = {
    "debug": True,
    "api_key": "hardcoded-key-bad-practice"
}

if __name__ == "__main__":
    # Test code
    processor = DataProcessor()
    print("DataProcessor initialized")