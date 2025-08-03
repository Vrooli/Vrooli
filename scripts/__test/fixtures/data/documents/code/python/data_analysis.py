#!/usr/bin/env python3
"""
Data Analysis Test Module
Test fixture for AI-powered code analysis and data science workflows
"""

import json
import csv
import statistics
from typing import Dict, List, Any, Tuple, Optional
from datetime import datetime, timedelta
from dataclasses import dataclass
from pathlib import Path

@dataclass
class DataSummary:
    """Summary statistics for a dataset."""
    total_records: int
    numeric_columns: List[str]
    categorical_columns: List[str]
    missing_values: Dict[str, int]
    data_types: Dict[str, str]
    timestamp: datetime

class DataAnalyzer:
    """
    Data analysis toolkit for processing business data.
    
    This is a test fixture for demonstrating:
    - Code analysis capabilities
    - AI-powered data science workflows
    - Documentation generation
    - Automation testing
    """
    
    def __init__(self):
        """Initialize the data analyzer."""
        self.data_cache = {}
        self.analysis_results = {}
    
    def load_csv_data(self, filepath: str) -> List[Dict[str, Any]]:
        """
        Load data from CSV file.
        
        Args:
            filepath: Path to the CSV file
            
        Returns:
            List of dictionaries representing the data
            
        Raises:
            FileNotFoundError: If the file doesn't exist
            ValueError: If the CSV is malformed
        """
        try:
            data = []
            with open(filepath, 'r', encoding='utf-8') as file:
                reader = csv.DictReader(file)
                for row in reader:
                    # Convert numeric strings to appropriate types
                    processed_row = self._process_row_types(row)
                    data.append(processed_row)
                    
            self.data_cache[filepath] = data
            return data
            
        except FileNotFoundError:
            raise FileNotFoundError(f"CSV file not found: {filepath}")
        except Exception as e:
            raise ValueError(f"Error reading CSV file: {e}")
    
    def _process_row_types(self, row: Dict[str, str]) -> Dict[str, Any]:
        """Convert string values to appropriate Python types."""
        processed = {}
        
        for key, value in row.items():
            if value == '' or value is None:
                processed[key] = None
            elif self._is_numeric(value):
                try:
                    # Try integer first, then float
                    if '.' in value:
                        processed[key] = float(value)
                    else:
                        processed[key] = int(value)
                except ValueError:
                    processed[key] = value
            elif self._is_boolean(value):
                processed[key] = value.lower() in ('true', 'yes', '1', 'on')
            else:
                processed[key] = value
                
        return processed
    
    def _is_numeric(self, value: str) -> bool:
        """Check if a string represents a numeric value."""
        try:
            float(value.replace(',', '').replace('$', '').replace('%', ''))
            return True
        except ValueError:
            return False
    
    def _is_boolean(self, value: str) -> bool:
        """Check if a string represents a boolean value."""
        return value.lower() in ('true', 'false', 'yes', 'no', '1', '0', 'on', 'off')
    
    def analyze_dataset(self, data: List[Dict[str, Any]]) -> DataSummary:
        """
        Perform comprehensive analysis of a dataset.
        
        Args:
            data: List of data records
            
        Returns:
            DataSummary object with analysis results
        """
        if not data:
            raise ValueError("Cannot analyze empty dataset")
        
        # Analyze column types and missing values
        numeric_columns = []
        categorical_columns = []
        missing_values = {}
        data_types = {}
        
        sample_row = data[0]
        
        for column in sample_row.keys():
            column_values = [row.get(column) for row in data]
            non_null_values = [v for v in column_values if v is not None]
            
            # Count missing values
            missing_count = len(column_values) - len(non_null_values)
            if missing_count > 0:
                missing_values[column] = missing_count
            
            # Determine data type
            if non_null_values:
                sample_value = non_null_values[0]
                if isinstance(sample_value, (int, float)):
                    numeric_columns.append(column)
                    data_types[column] = 'numeric'
                else:
                    categorical_columns.append(column)
                    data_types[column] = 'categorical'
            else:
                data_types[column] = 'unknown'
        
        return DataSummary(
            total_records=len(data),
            numeric_columns=numeric_columns,
            categorical_columns=categorical_columns,
            missing_values=missing_values,
            data_types=data_types,
            timestamp=datetime.now()
        )
    
    def calculate_statistics(self, data: List[Dict[str, Any]], 
                           column: str) -> Dict[str, float]:
        """
        Calculate descriptive statistics for a numeric column.
        
        Args:
            data: Dataset to analyze
            column: Column name to analyze
            
        Returns:
            Dictionary of statistical measures
        """
        values = []
        
        for row in data:
            value = row.get(column)
            if value is not None and isinstance(value, (int, float)):
                values.append(float(value))
        
        if not values:
            return {}
        
        try:
            stats = {
                'count': len(values),
                'mean': statistics.mean(values),
                'median': statistics.median(values),
                'mode': statistics.mode(values) if len(set(values)) < len(values) else None,
                'std_dev': statistics.stdev(values) if len(values) > 1 else 0,
                'variance': statistics.variance(values) if len(values) > 1 else 0,
                'min': min(values),
                'max': max(values),
                'range': max(values) - min(values),
                'q1': statistics.quantiles(values, n=4)[0] if len(values) >= 4 else None,
                'q3': statistics.quantiles(values, n=4)[2] if len(values) >= 4 else None
            }
            
            # Calculate IQR if quartiles are available
            if stats['q1'] is not None and stats['q3'] is not None:
                stats['iqr'] = stats['q3'] - stats['q1']
            
            return stats
            
        except statistics.StatisticsError as e:
            print(f"Statistics calculation error for column {column}: {e}")
            return {'count': len(values), 'error': str(e)}
    
    def generate_report(self, data: List[Dict[str, Any]], 
                       output_file: str = None) -> str:
        """
        Generate a comprehensive data analysis report.
        
        Args:
            data: Dataset to analyze
            output_file: Optional file path to save the report
            
        Returns:
            Report as a string
        """
        summary = self.analyze_dataset(data)
        
        report_lines = [
            "=" * 60,
            "DATA ANALYSIS REPORT",
            "=" * 60,
            f"Generated: {summary.timestamp.strftime('%Y-%m-%d %H:%M:%S')}",
            f"Total Records: {summary.total_records:,}",
            "",
            "COLUMN ANALYSIS",
            "-" * 20,
            f"Numeric Columns ({len(summary.numeric_columns)}): {', '.join(summary.numeric_columns)}",
            f"Categorical Columns ({len(summary.categorical_columns)}): {', '.join(summary.categorical_columns)}",
            "",
        ]
        
        # Missing values section
        if summary.missing_values:
            report_lines.extend([
                "MISSING VALUES",
                "-" * 15,
            ])
            for column, count in summary.missing_values.items():
                percentage = (count / summary.total_records) * 100
                report_lines.append(f"{column}: {count} ({percentage:.1f}%)")
            report_lines.append("")
        
        # Statistical analysis for numeric columns
        if summary.numeric_columns:
            report_lines.extend([
                "STATISTICAL ANALYSIS",
                "-" * 20,
            ])
            
            for column in summary.numeric_columns:
                stats = self.calculate_statistics(data, column)
                if stats and 'error' not in stats:
                    report_lines.extend([
                        f"\n{column.upper()}:",
                        f"  Count: {stats['count']:,}",
                        f"  Mean: {stats['mean']:.2f}",
                        f"  Median: {stats['median']:.2f}",
                        f"  Std Dev: {stats['std_dev']:.2f}",
                        f"  Min: {stats['min']:.2f}",
                        f"  Max: {stats['max']:.2f}",
                        f"  Range: {stats['range']:.2f}",
                    ])
                    
                    if stats.get('q1') and stats.get('q3'):
                        report_lines.extend([
                            f"  Q1: {stats['q1']:.2f}",
                            f"  Q3: {stats['q3']:.2f}",
                            f"  IQR: {stats['iqr']:.2f}",
                        ])
        
        report_text = "\n".join(report_lines)
        
        # Save to file if requested
        if output_file:
            try:
                with open(output_file, 'w', encoding='utf-8') as f:
                    f.write(report_text)
                print(f"Report saved to: {output_file}")
            except Exception as e:
                print(f"Error saving report: {e}")
        
        return report_text

def main():
    """Main function for testing the data analyzer."""
    analyzer = DataAnalyzer()
    
    # Test with sample data
    sample_data = [
        {'name': 'Alice', 'age': 30, 'salary': 75000, 'department': 'Engineering'},
        {'name': 'Bob', 'age': 25, 'salary': 65000, 'department': 'Marketing'},
        {'name': 'Charlie', 'age': 35, 'salary': 85000, 'department': 'Engineering'},
        {'name': 'Diana', 'age': 28, 'salary': 70000, 'department': 'Sales'},
        {'name': 'Eve', 'age': None, 'salary': 80000, 'department': 'Engineering'},
    ]
    
    print("Analyzing sample dataset...")
    
    # Generate and display report
    report = analyzer.generate_report(sample_data, 'analysis_report.txt')
    print(report)
    
    # Test with CSV file if available
    csv_path = '../structured/customers.csv'
    if Path(csv_path).exists():
        try:
            print(f"\nAnalyzing CSV file: {csv_path}")
            csv_data = analyzer.load_csv_data(csv_path)
            csv_report = analyzer.generate_report(csv_data, 'csv_analysis_report.txt')
            print("CSV analysis completed. Report saved.")
        except Exception as e:
            print(f"Error analyzing CSV: {e}")

if __name__ == "__main__":
    main()