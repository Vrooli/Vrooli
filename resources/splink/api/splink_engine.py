#!/usr/bin/env python3
"""
Splink Engine - Core probabilistic record linkage functionality
"""

import os
import json
import logging
import pandas as pd
from typing import Dict, List, Optional, Any, Tuple
from datetime import datetime
import tempfile
import hashlib
import duckdb

logger = logging.getLogger(__name__)

class SpinkEngine:
    """
    Core engine for Splink record linkage operations
    Using DuckDB for local processing
    """
    
    def __init__(self, backend: str = "duckdb", data_dir: str = "/data"):
        """Initialize the Splink engine with specified backend"""
        self.backend = backend.lower()
        self.data_dir = data_dir
        
        # Ensure data directory exists
        os.makedirs(data_dir, exist_ok=True)
        os.makedirs(os.path.join(data_dir, "datasets"), exist_ok=True)
        os.makedirs(os.path.join(data_dir, "results"), exist_ok=True)
        
        # Initialize DuckDB connection
        self.conn = duckdb.connect(":memory:")
        
        logger.info(f"Initialized Splink engine with backend: {backend}")
    
    def _load_dataset(self, dataset_id: str) -> pd.DataFrame:
        """Load dataset from storage"""
        # For now, create sample data. In production, would load from MinIO/PostgreSQL
        logger.info(f"Loading dataset: {dataset_id}")
        
        # Generate sample data based on dataset_id hash for consistency
        seed = int(hashlib.md5(dataset_id.encode()).hexdigest()[:8], 16) % 10000
        
        sample_data = pd.DataFrame({
            "unique_id": range(1000),
            "first_name": [f"FirstName_{i % 50}" for i in range(1000)],
            "last_name": [f"LastName_{i % 100}" for i in range(1000)],
            "date_of_birth": pd.date_range("1950-01-01", periods=1000, freq="D").strftime("%Y-%m-%d"),
            "city": [f"City_{i % 20}" for i in range(1000)],
            "email": [f"user{i}@example.com" if i % 10 != 0 else f"user{i-1}@example.com" for i in range(1000)]
        })
        
        # Add some duplicates for testing
        duplicates = sample_data.sample(n=50, random_state=seed).copy()
        duplicates["unique_id"] = duplicates["unique_id"] + 1000
        
        return pd.concat([sample_data, duplicates], ignore_index=True)
    
    def _calculate_similarity(self, df1: pd.DataFrame, df2: pd.DataFrame, columns: List[str]) -> pd.DataFrame:
        """Calculate similarity scores between records"""
        # Simple similarity calculation using exact match and fuzzy matching
        results = []
        
        # For demonstration, compare first 100 records
        for i, row1 in df1.head(100).iterrows():
            for j, row2 in df2.head(100).iterrows():
                score = 0
                matches = 0
                
                for col in columns:
                    if col in row1 and col in row2:
                        if row1[col] == row2[col]:
                            score += 1
                        matches += 1
                
                if matches > 0:
                    similarity = score / matches
                    if similarity > 0.5:  # Only keep high similarity matches
                        results.append({
                            "id1": row1["unique_id"],
                            "id2": row2["unique_id"],
                            "similarity": similarity
                        })
        
        return pd.DataFrame(results)
    
    def deduplicate(
        self,
        dataset_id: str,
        settings: Dict[str, Any],
        progress_callback: Optional[callable] = None
    ) -> Dict[str, Any]:
        """
        Perform deduplication on a single dataset using simplified approach
        """
        start_time = datetime.utcnow()
        
        try:
            # Load dataset
            df = self._load_dataset(dataset_id)
            records_count = len(df)
            
            if progress_callback:
                progress_callback(10, "Dataset loaded")
            
            # Get comparison columns
            comparison_columns = settings.get("comparison_columns", ["first_name", "last_name", "email"])
            threshold = settings.get("threshold", 0.9)
            
            if progress_callback:
                progress_callback(30, "Settings configured")
            
            # Register dataframe with DuckDB
            self.conn.register("dataset", df)
            
            # Find potential duplicates using SQL
            duplicate_query = f"""
            SELECT 
                a.unique_id as id1,
                b.unique_id as id2,
                CASE 
                    WHEN a.first_name = b.first_name THEN 1 ELSE 0 
                END +
                CASE 
                    WHEN a.last_name = b.last_name THEN 1 ELSE 0 
                END +
                CASE 
                    WHEN a.email = b.email THEN 1 ELSE 0 
                END as match_score
            FROM dataset a, dataset b
            WHERE a.unique_id < b.unique_id
                AND (a.first_name = b.first_name 
                    OR a.last_name = b.last_name 
                    OR a.email = b.email)
            """
            
            if progress_callback:
                progress_callback(50, "Finding duplicates")
            
            # Execute query
            duplicates_df = self.conn.execute(duplicate_query).fetchdf()
            
            # Calculate match probability
            if not duplicates_df.empty:
                duplicates_df["match_probability"] = duplicates_df["match_score"] / 3.0
                high_confidence_matches = duplicates_df[duplicates_df["match_probability"] >= threshold]
                duplicates_found = len(high_confidence_matches)
                confidence_scores = high_confidence_matches["match_probability"].head(10).tolist()
            else:
                duplicates_found = 0
                confidence_scores = []
            
            if progress_callback:
                progress_callback(80, "Calculating statistics")
            
            # Calculate unique entities (approximate)
            unique_entities = records_count - duplicates_found
            
            if progress_callback:
                progress_callback(100, "Deduplication complete")
            
            # Calculate processing time
            processing_time = (datetime.utcnow() - start_time).total_seconds()
            
            return {
                "success": True,
                "duplicates_found": duplicates_found,
                "unique_entities": unique_entities,
                "records_processed": records_count,
                "confidence_scores": confidence_scores,
                "processing_time": f"{processing_time:.2f}s",
                "threshold_used": threshold,
                "backend": self.backend
            }
            
        except Exception as e:
            logger.error(f"Deduplication failed: {str(e)}")
            return {
                "success": False,
                "error": str(e),
                "processing_time": f"{(datetime.utcnow() - start_time).total_seconds():.2f}s"
            }
    
    def link_datasets(
        self,
        dataset1_id: str,
        dataset2_id: str,
        settings: Dict[str, Any],
        progress_callback: Optional[callable] = None
    ) -> Dict[str, Any]:
        """
        Link records between two datasets using simplified approach
        """
        start_time = datetime.utcnow()
        
        try:
            # Load datasets
            df1 = self._load_dataset(dataset1_id)
            df2 = self._load_dataset(dataset2_id)
            
            total_records = len(df1) + len(df2)
            
            if progress_callback:
                progress_callback(10, "Datasets loaded")
            
            # Get comparison columns
            comparison_columns = settings.get("comparison_columns", ["first_name", "last_name", "email"])
            threshold = settings.get("threshold", 0.9)
            
            if progress_callback:
                progress_callback(30, "Settings configured")
            
            # Register dataframes with DuckDB
            self.conn.register("dataset1", df1)
            self.conn.register("dataset2", df2)
            
            # Find potential matches using SQL
            linkage_query = f"""
            SELECT 
                a.unique_id as id1,
                b.unique_id as id2,
                CASE 
                    WHEN a.first_name = b.first_name THEN 1 ELSE 0 
                END +
                CASE 
                    WHEN a.last_name = b.last_name THEN 1 ELSE 0 
                END +
                CASE 
                    WHEN a.email = b.email THEN 1 ELSE 0 
                END as match_score
            FROM dataset1 a, dataset2 b
            WHERE a.first_name = b.first_name 
                OR a.last_name = b.last_name 
                OR a.email = b.email
            LIMIT 5000
            """
            
            if progress_callback:
                progress_callback(50, "Finding matches")
            
            # Execute query
            matches_df = self.conn.execute(linkage_query).fetchdf()
            
            # Calculate match probability
            if not matches_df.empty:
                matches_df["match_probability"] = matches_df["match_score"] / 3.0
                high_confidence_matches = matches_df[matches_df["match_probability"] >= threshold]
                matches_found = len(high_confidence_matches)
                confidence_scores = high_confidence_matches["match_probability"].head(10).tolist()
            else:
                matches_found = 0
                confidence_scores = []
            
            if progress_callback:
                progress_callback(80, "Calculating statistics")
            
            if progress_callback:
                progress_callback(100, "Linkage complete")
            
            # Calculate processing time
            processing_time = (datetime.utcnow() - start_time).total_seconds()
            
            return {
                "success": True,
                "matches_found": matches_found,
                "records_compared": total_records,
                "confidence_scores": confidence_scores,
                "processing_time": f"{processing_time:.2f}s",
                "threshold_used": threshold,
                "backend": self.backend
            }
            
        except Exception as e:
            logger.error(f"Linkage failed: {str(e)}")
            return {
                "success": False,
                "error": str(e),
                "processing_time": f"{(datetime.utcnow() - start_time).total_seconds():.2f}s"
            }
    
    def estimate_parameters(
        self,
        dataset_id: str,
        settings: Dict[str, Any],
        progress_callback: Optional[callable] = None
    ) -> Dict[str, Any]:
        """
        Estimate linkage parameters using simplified statistics
        """
        start_time = datetime.utcnow()
        
        try:
            # Load dataset
            df = self._load_dataset(dataset_id)
            
            if progress_callback:
                progress_callback(20, "Dataset loaded")
            
            # Register dataframe with DuckDB
            self.conn.register("dataset", df)
            
            if progress_callback:
                progress_callback(40, "Analyzing data distribution")
            
            # Calculate field statistics
            stats_query = """
            SELECT 
                COUNT(DISTINCT first_name) as unique_first_names,
                COUNT(DISTINCT last_name) as unique_last_names,
                COUNT(DISTINCT email) as unique_emails,
                COUNT(*) as total_records
            FROM dataset
            """
            
            stats = self.conn.execute(stats_query).fetchone()
            
            if progress_callback:
                progress_callback(60, "Calculating probabilities")
            
            # Estimate m-probabilities (probability of field agreement given records match)
            # These would normally be calculated using EM algorithm
            m_probabilities = [
                0.95,  # First name match probability
                0.93,  # Last name match probability
                0.98,  # Email match probability
                0.85,  # City match probability
                0.90   # Date of birth match probability
            ]
            
            # Estimate u-probabilities (probability of field agreement given records don't match)
            # Based on uniqueness of fields
            total = stats[3]
            u_probabilities = [
                stats[0] / total if total > 0 else 0.1,  # First name
                stats[1] / total if total > 0 else 0.1,  # Last name
                stats[2] / total if total > 0 else 0.05, # Email
                0.15,  # City (fixed estimate)
                0.10   # Date of birth (fixed estimate)
            ]
            
            # Normalize u-probabilities
            u_probabilities = [min(0.2, max(0.01, u)) for u in u_probabilities]
            
            # Estimate lambda (proportion of matches in dataset)
            # This is a rough estimate based on duplicate detection
            duplicate_rate_query = """
            SELECT COUNT(*) as duplicates
            FROM (
                SELECT email, COUNT(*) as cnt
                FROM dataset
                GROUP BY email
                HAVING COUNT(*) > 1
            )
            """
            
            duplicates = self.conn.execute(duplicate_rate_query).fetchone()[0]
            lambda_value = min(0.2, duplicates / total) if total > 0 else 0.1
            
            if progress_callback:
                progress_callback(80, "Finalizing estimates")
            
            if progress_callback:
                progress_callback(100, "Estimation complete")
            
            # Calculate processing time
            processing_time = (datetime.utcnow() - start_time).total_seconds()
            
            return {
                "success": True,
                "m_probabilities": m_probabilities[:5],
                "u_probabilities": u_probabilities[:5],
                "lambda": lambda_value,
                "iterations": 10,  # Simulated EM iterations
                "processing_time": f"{processing_time:.2f}s",
                "backend": self.backend,
                "statistics": {
                    "total_records": total,
                    "unique_first_names": stats[0],
                    "unique_last_names": stats[1],
                    "unique_emails": stats[2]
                }
            }
            
        except Exception as e:
            logger.error(f"Parameter estimation failed: {str(e)}")
            return {
                "success": False,
                "error": str(e),
                "processing_time": f"{(datetime.utcnow() - start_time).total_seconds():.2f}s"
            }

# Singleton instance
_engine_instance = None

def get_splink_engine(backend: str = "duckdb", data_dir: str = "/data") -> SpinkEngine:
    """Get or create the Splink engine singleton"""
    global _engine_instance
    if _engine_instance is None:
        _engine_instance = SpinkEngine(backend, data_dir)
    return _engine_instance