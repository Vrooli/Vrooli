"""
Python Data Processing Example for Windmill

This script demonstrates data processing capabilities including:
- CSV data manipulation
- Statistical analysis
- Data validation
- Error handling
"""

import json
import csv
from io import StringIO
from typing import Dict, List, Any, Optional
from datetime import datetime
import statistics

def main(
    csv_data: str,
    analysis_type: str = "summary",
    filter_column: Optional[str] = None,
    filter_value: Optional[str] = None
) -> Dict[str, Any]:
    """
    Process CSV data and return analysis results.
    
    Args:
        csv_data: CSV data as string
        analysis_type: Type of analysis ('summary', 'detailed', 'statistics')
        filter_column: Column to filter by (optional)
        filter_value: Value to filter for (optional)
    
    Returns:
        Dictionary containing analysis results
    """
    
    # Validate inputs
    if not csv_data or csv_data.strip() == '':
        raise ValueError("CSV data is required and cannot be empty")
    
    if analysis_type not in ['summary', 'detailed', 'statistics']:
        raise ValueError("analysis_type must be one of: summary, detailed, statistics")
    
    try:
        # Parse CSV data
        csv_reader = csv.DictReader(StringIO(csv_data))
        rows = list(csv_reader)
        
        if not rows:
            raise ValueError("CSV data contains no rows")
        
        # Apply filtering if specified
        if filter_column and filter_value:
            if filter_column not in rows[0]:
                raise ValueError(f"Filter column '{filter_column}' not found in data")
            
            original_count = len(rows)
            rows = [row for row in rows if row.get(filter_column) == filter_value]
            filtered_count = len(rows)
        else:
            original_count = filtered_count = len(rows)
        
        # Get column information
        columns = list(rows[0].keys()) if rows else []
        
        # Perform analysis based on type
        result = {
            "analysis_type": analysis_type,
            "timestamp": datetime.now().isoformat(),
            "data_info": {
                "total_rows": original_count,
                "filtered_rows": filtered_count,
                "columns": columns,
                "column_count": len(columns)
            }
        }
        
        if filter_column and filter_value:
            result["filter_applied"] = {
                "column": filter_column,
                "value": filter_value,
                "rows_removed": original_count - filtered_count
            }
        
        if analysis_type == "summary":
            result["summary"] = generate_summary(rows)
        elif analysis_type == "detailed":
            result["detailed"] = generate_detailed_analysis(rows)
        elif analysis_type == "statistics":
            result["statistics"] = generate_statistics(rows)
        
        return result
        
    except csv.Error as e:
        raise ValueError(f"Invalid CSV format: {str(e)}")
    except Exception as e:
        raise RuntimeError(f"Data processing error: {str(e)}")

def generate_summary(rows: List[Dict[str, str]]) -> Dict[str, Any]:
    """Generate basic summary of the data."""
    if not rows:
        return {"message": "No data to summarize"}
    
    # Sample first few rows
    sample_rows = rows[:5]
    
    # Count non-empty values per column
    column_stats = {}
    for column in rows[0].keys():
        non_empty = sum(1 for row in rows if row.get(column, '').strip())
        column_stats[column] = {
            "non_empty_values": non_empty,
            "empty_values": len(rows) - non_empty,
            "fill_rate": f"{(non_empty / len(rows) * 100):.1f}%"
        }
    
    return {
        "row_count": len(rows),
        "column_stats": column_stats,
        "sample_data": sample_rows
    }

def generate_detailed_analysis(rows: List[Dict[str, str]]) -> Dict[str, Any]:
    """Generate detailed analysis including unique values and patterns."""
    if not rows:
        return {"message": "No data to analyze"}
    
    analysis = {}
    
    for column in rows[0].keys():
        values = [row.get(column, '').strip() for row in rows]
        non_empty_values = [v for v in values if v]
        
        unique_values = list(set(non_empty_values))
        
        # Try to detect numeric columns
        numeric_values = []
        for value in non_empty_values:
            try:
                numeric_values.append(float(value))
            except ValueError:
                continue
        
        column_analysis = {
            "total_values": len(values),
            "non_empty_values": len(non_empty_values),
            "unique_values": len(unique_values),
            "sample_unique": unique_values[:10],  # First 10 unique values
            "is_numeric": len(numeric_values) == len(non_empty_values) and len(non_empty_values) > 0
        }
        
        if column_analysis["is_numeric"] and numeric_values:
            column_analysis["numeric_stats"] = {
                "min": min(numeric_values),
                "max": max(numeric_values),
                "mean": statistics.mean(numeric_values),
                "median": statistics.median(numeric_values)
            }
        
        analysis[column] = column_analysis
    
    return analysis

def generate_statistics(rows: List[Dict[str, str]]) -> Dict[str, Any]:
    """Generate comprehensive statistics for numeric columns."""
    if not rows:
        return {"message": "No data for statistics"}
    
    statistics_result = {
        "numeric_columns": {},
        "text_columns": {}
    }
    
    for column in rows[0].keys():
        values = [row.get(column, '').strip() for row in rows if row.get(column, '').strip()]
        
        # Try to convert to numeric
        numeric_values = []
        for value in values:
            try:
                numeric_values.append(float(value))
            except ValueError:
                continue
        
        if len(numeric_values) >= len(values) * 0.8:  # 80% numeric threshold
            # Treat as numeric column
            if numeric_values:
                stats = {
                    "count": len(numeric_values),
                    "min": min(numeric_values),
                    "max": max(numeric_values),
                    "mean": statistics.mean(numeric_values),
                    "median": statistics.median(numeric_values),
                    "range": max(numeric_values) - min(numeric_values)
                }
                
                if len(numeric_values) > 1:
                    stats["std_dev"] = statistics.stdev(numeric_values)
                    stats["variance"] = statistics.variance(numeric_values)
                
                statistics_result["numeric_columns"][column] = stats
        else:
            # Treat as text column
            if values:
                lengths = [len(v) for v in values]
                statistics_result["text_columns"][column] = {
                    "count": len(values),
                    "unique_count": len(set(values)),
                    "avg_length": statistics.mean(lengths),
                    "min_length": min(lengths),
                    "max_length": max(lengths),
                    "most_common": max(set(values), key=values.count) if values else None
                }
    
    return statistics_result

# Example usage:
# Input CSV:
# name,age,city,salary
# John,30,New York,50000
# Jane,25,San Francisco,60000
# Bob,35,Chicago,55000
#
# Input parameters:
# {
#   "csv_data": "name,age,city,salary\nJohn,30,New York,50000\nJane,25,San Francisco,60000",
#   "analysis_type": "statistics",
#   "filter_column": "city",
#   "filter_value": "New York"
# }