#!/usr/bin/env python3
"""
Semantic Search Script for Resume Screening Assistant
Provides advanced semantic search capabilities across candidates and jobs
"""

import json
import requests
from typing import List, Dict, Any, Optional, Union
from datetime import datetime, timedelta
import logging
import psycopg2
import psycopg2.extras
from contextlib import contextmanager
import hashlib
import time
from functools import lru_cache

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Resource URLs - should match Vrooli resource ports
QDRANT_URL = "http://localhost:6333"
OLLAMA_URL = "http://localhost:11434"
POSTGRES_HOST = "localhost"
POSTGRES_PORT = 5432
POSTGRES_DB = "resume_screening_assistant"
POSTGRES_USER = "postgres"
POSTGRES_PASSWORD = "password"

# Connection pooling with requests sessions
qdrant_session = requests.Session()
ollama_session = requests.Session()

# Configure session adapters for connection pooling
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry

# Setup retry strategy and connection pooling
retry_strategy = Retry(
    total=3,
    backoff_factor=1,
    status_forcelist=[429, 500, 502, 503, 504]
)

adapter = HTTPAdapter(
    pool_connections=10,
    pool_maxsize=20,
    max_retries=retry_strategy
)

qdrant_session.mount("http://", adapter)
qdrant_session.mount("https://", adapter)
ollama_session.mount("http://", adapter)
ollama_session.mount("https://", adapter)

# Simple in-memory cache
class SimpleCache:
    def __init__(self, default_ttl=900):  # 15 minutes default TTL
        self.cache = {}
        self.default_ttl = default_ttl
    
    def _get_cache_key(self, *args, **kwargs):
        """Generate a cache key from arguments"""
        key_data = str(args) + str(sorted(kwargs.items()))
        return hashlib.md5(key_data.encode()).hexdigest()
    
    def get(self, key):
        """Get cached value if not expired"""
        if key in self.cache:
            value, expiry = self.cache[key]
            if datetime.now() < expiry:
                return value
            else:
                del self.cache[key]
        return None
    
    def set(self, key, value, ttl=None):
        """Set cached value with TTL"""
        ttl = ttl or self.default_ttl
        expiry = datetime.now() + timedelta(seconds=ttl)
        self.cache[key] = (value, expiry)
    
    def clear(self):
        """Clear all cached values"""
        self.cache.clear()

# Global cache instances
embedding_cache = SimpleCache(ttl=3600)  # 1 hour for embeddings
search_cache = SimpleCache(ttl=900)  # 15 minutes for search results

@contextmanager
def get_database_connection():
    """
    Context manager for database connections with proper cleanup
    """
    conn = None
    try:
        conn = psycopg2.connect(
            host=POSTGRES_HOST,
            port=POSTGRES_PORT,
            database=POSTGRES_DB,
            user=POSTGRES_USER,
            password=POSTGRES_PASSWORD
        )
        yield conn
    except Exception as e:
        logger.error(f"Database connection error: {e}")
        if conn:
            conn.rollback()
        raise
    finally:
        if conn:
            conn.close()

def generate_query_embedding(query: str, model: str = "nomic-embed-text") -> Optional[List[float]]:
    """
    Generate embedding vector for search query with caching
    """
    # Check cache first
    cache_key = embedding_cache._get_cache_key(query, model)
    cached_embedding = embedding_cache.get(cache_key)
    if cached_embedding:
        logger.debug(f"Using cached embedding for query: {query[:50]}...")
        return cached_embedding
    
    try:
        response = ollama_session.post(
            f"{OLLAMA_URL}/api/embeddings",
            json={
                "model": model,
                "prompt": query
            },
            timeout=45
        )
        
        if response.status_code == 200:
            embedding = response.json().get("embedding", [])
            # Cache the embedding
            embedding_cache.set(cache_key, embedding, ttl=3600)  # Cache for 1 hour
            return embedding
    except Exception as e:
        logger.error(f"Error generating query embedding: {e}")
    return None

def search_candidates_vector(
    query_vector: List[float],
    limit: int = 20,
    score_threshold: float = 0.6,
    filters: Dict[str, Any] = None
) -> List[Dict[str, Any]]:
    """
    Search candidates using vector similarity
    """
    try:
        search_payload = {
            "vector": query_vector,
            "limit": limit,
            "score_threshold": score_threshold,
            "with_payload": True,
            "with_vector": False
        }
        
        # Add filters if provided
        if filters:
            search_payload["filter"] = filters
        
        response = qdrant_session.post(
            f"{QDRANT_URL}/collections/resumes/points/search",
            json=search_payload,
            timeout=45
        )
        
        if response.status_code == 200:
            data = response.json()
            return data.get("result", [])
    except Exception as e:
        logger.error(f"Error searching candidates: {e}")
    return []

def search_jobs_vector(
    query_vector: List[float],
    limit: int = 20,
    score_threshold: float = 0.6,
    filters: Dict[str, Any] = None
) -> List[Dict[str, Any]]:
    """
    Search jobs using vector similarity
    """
    try:
        search_payload = {
            "vector": query_vector,
            "limit": limit,
            "score_threshold": score_threshold,
            "with_payload": True,
            "with_vector": False
        }
        
        # Add filters if provided
        if filters:
            search_payload["filter"] = filters
        
        response = qdrant_session.post(
            f"{QDRANT_URL}/collections/job-descriptions/points/search",
            json=search_payload,
            timeout=45
        )
        
        if response.status_code == 200:
            data = response.json()
            return data.get("result", [])
    except Exception as e:
        logger.error(f"Error searching jobs: {e}")
    return []

def search_database_text(
    query: str,
    search_type: str = "both",
    limit: int = 20
) -> Dict[str, List[Dict[str, Any]]]:
    """
    Fallback text-based search in database
    """
    results = {"candidates": [], "jobs": []}
    
    try:
        with get_database_connection() as conn:
            cursor = conn.cursor(cursor_factory=psycopg2.extras.RealDictCursor)
            
            if search_type in ["candidates", "both"]:
                # Search candidates by name, skills, or resume text
                candidate_query = """
                SELECT id, candidate_name, email, parsed_skills, experience_years, 
                       education_level, score, status
                FROM resumes 
                WHERE LOWER(candidate_name) LIKE LOWER(%s) 
                   OR LOWER(resume_text) LIKE LOWER(%s)
                   OR LOWER(parsed_skills::text) LIKE LOWER(%s)
                ORDER BY score DESC
                LIMIT %s
                """
                query_pattern = f"%{query}%"
                cursor.execute(candidate_query, (query_pattern, query_pattern, query_pattern, limit))
                candidate_rows = cursor.fetchall()
                
                for row in candidate_rows:
                    results["candidates"].append({
                        "id": row["id"],
                        "type": "candidate",
                        "candidate_name": row["candidate_name"],
                        "email": row["email"],
                        "skills": json.loads(row["parsed_skills"]) if row["parsed_skills"] else [],
                        "experience_years": row["experience_years"],
                        "education_level": row["education_level"],
                        "score": row["score"],
                        "status": row["status"]
                    })
                
            if search_type in ["jobs", "both"]:
                # Search jobs by title, company, description, or skills
                job_query = """
                SELECT id, job_title, company_name, description, required_skills,
                       experience_required, location, salary_range
                FROM job_descriptions 
                WHERE status = 'active' 
                  AND (LOWER(job_title) LIKE LOWER(%s) 
                       OR LOWER(company_name) LIKE LOWER(%s)
                       OR LOWER(description) LIKE LOWER(%s)
                       OR LOWER(required_skills::text) LIKE LOWER(%s))
                ORDER BY created_at DESC
                LIMIT %s
                """
                query_pattern = f"%{query}%"
                cursor.execute(job_query, (query_pattern, query_pattern, query_pattern, query_pattern, limit))
                job_rows = cursor.fetchall()
                
                for row in job_rows:
                    results["jobs"].append({
                        "id": row["id"],
                        "type": "job",
                        "job_title": row["job_title"],
                        "company_name": row["company_name"],
                        "description": row["description"],
                        "required_skills": json.loads(row["required_skills"]) if row["required_skills"] else [],
                        "experience_required": row["experience_required"],
                        "location": row["location"],
                        "salary_range": row["salary_range"]
                    })
            
            cursor.close()
            
    except Exception as e:
        logger.error(f"Error in database text search: {e}")
    
    return results

def format_candidate_result(candidate: Dict[str, Any], score: float) -> Dict[str, Any]:
    """
    Format candidate search result for display
    """
    payload = candidate.get("payload", {})
    skills = payload.get("skills", [])
    
    return {
        "type": "candidate",
        "id": candidate.get("id"),
        "score": round(score, 3),
        "candidate_name": payload.get("candidate_name", "Unknown"),
        "skills": skills[:5] if isinstance(skills, list) else [],
        "experience_years": payload.get("experience_years", 0),
        "education_level": payload.get("education_level", "Not specified"),
        "overall_score": payload.get("overall_score", 0),
        "match_details": {
            "similarity_score": round(score, 3),
            "key_skills": skills[:3] if isinstance(skills, list) else [],
            "experience_match": payload.get("experience_years", 0) >= 2
        }
    }

def format_job_result(job: Dict[str, Any], score: float) -> Dict[str, Any]:
    """
    Format job search result for display
    """
    payload = job.get("payload", {})
    required_skills = payload.get("required_skills", [])
    
    return {
        "type": "job",
        "id": job.get("id"),
        "score": round(score, 3),
        "job_title": payload.get("job_title", "Unknown"),
        "company_name": payload.get("company_name", "Unknown"),
        "required_skills": required_skills[:5] if isinstance(required_skills, list) else [],
        "experience_required": payload.get("experience_required", 0),
        "location": payload.get("location", "Remote"),
        "match_details": {
            "similarity_score": round(score, 3),
            "key_requirements": required_skills[:3] if isinstance(required_skills, list) else [],
            "experience_level": f"{payload.get('experience_required', 0)}+ years"
        }
    }

def log_search_query(
    query: str,
    search_type: str,
    results_count: int,
    execution_time_ms: int,
    filters: Dict[str, Any] = None
):
    """
    Log search query for analytics
    """
    try:
        with get_database_connection() as conn:
            cursor = conn.cursor()
            
            # Insert into search_history table
            insert_query = """
            INSERT INTO search_history (
                query_text, query_type, results_count, execution_time_ms, filters
            ) VALUES (%s, %s, %s, %s, %s)
            """
            
            cursor.execute(insert_query, (
                query,
                "semantic",
                results_count,
                execution_time_ms,
                json.dumps(filters) if filters else None
            ))
            
            conn.commit()
            cursor.close()
            
            logger.info(f"Search query logged: '{query}' ({results_count} results, {execution_time_ms}ms)")
            
    except Exception as e:
        logger.error(f"Error logging search: {e}")
        # Fall back to console logging
        log_data = {
            "query_text": query,
            "query_type": "semantic",
            "search_type": search_type,
            "results_count": results_count,
            "execution_time_ms": execution_time_ms,
            "filters": json.dumps(filters) if filters else None,
            "timestamp": datetime.now().isoformat()
        }
        logger.info(f"Search logged (fallback): {log_data}")

def main(
    query: str,
    search_type: str = "both",
    limit: int = 20,
    score_threshold: float = 0.6,
    filters: Dict[str, Any] = None,
    use_vector_search: bool = True
) -> Dict[str, Any]:
    """
    Main semantic search function with caching
    
    Args:
        query: Search query text
        search_type: "candidates", "jobs", or "both"
        limit: Maximum number of results per type
        score_threshold: Minimum similarity score
        filters: Additional filters to apply
        use_vector_search: Whether to use vector search (True) or text search (False)
    
    Returns:
        Dictionary with search results and metadata
    """
    start_time = datetime.now()
    
    # Check cache for this exact search
    cache_key = search_cache._get_cache_key(
        query, search_type, limit, score_threshold, filters, use_vector_search
    )
    cached_result = search_cache.get(cache_key)
    if cached_result:
        logger.debug(f"Using cached search result for: {query[:50]}...")
        # Update timestamp but keep other data
        cached_result["metadata"]["timestamp"] = datetime.now().isoformat()
        cached_result["metadata"]["cached"] = True
        return cached_result
    
    # Validate inputs
    if not query or len(query.strip()) < 2:
        return {
            "success": False,
            "error": "Query must be at least 2 characters long",
            "results": []
        }
    
    if search_type not in ["candidates", "jobs", "both"]:
        return {
            "success": False,
            "error": "search_type must be 'candidates', 'jobs', or 'both'",
            "results": []
        }
    
    query = query.strip()
    limit = min(max(1, limit), 50)  # Clamp between 1-50
    score_threshold = max(0.0, min(1.0, score_threshold))  # Clamp between 0-1
    
    logger.info(f"Starting semantic search: '{query}' (type: {search_type}, limit: {limit})")
    
    results = []
    candidate_count = 0
    job_count = 0
    
    try:
        if use_vector_search:
            # Vector-based semantic search
            query_vector = generate_query_embedding(query)
            
            if not query_vector:
                logger.warning("Failed to generate query embedding, falling back to text search")
                use_vector_search = False
            else:
                # Search candidates
                if search_type in ["candidates", "both"]:
                    candidate_results = search_candidates_vector(
                        query_vector, limit, score_threshold, filters
                    )
                    for result in candidate_results:
                        formatted = format_candidate_result(result, result.get("score", 0))
                        results.append(formatted)
                    candidate_count = len(candidate_results)
                
                # Search jobs
                if search_type in ["jobs", "both"]:
                    job_results = search_jobs_vector(
                        query_vector, limit, score_threshold, filters
                    )
                    for result in job_results:
                        formatted = format_job_result(result, result.get("score", 0))
                        results.append(formatted)
                    job_count = len(job_results)
        
        if not use_vector_search:
            # Fallback to text-based search
            text_results = search_database_text(query, search_type, limit)
            candidate_count = len(text_results.get("candidates", []))
            job_count = len(text_results.get("jobs", []))
            
            # Convert text results to standard format
            for candidate in text_results.get("candidates", []):
                formatted = {
                    "type": "candidate",
                    "id": candidate.get("id"),
                    "score": candidate.get("score", 50) / 100.0,  # Convert to 0-1 scale
                    "candidate_name": candidate.get("candidate_name", "Unknown"),
                    "skills": candidate.get("skills", [])[:5],
                    "experience_years": candidate.get("experience_years", 0),
                    "education_level": candidate.get("education_level", "Not specified"),
                    "overall_score": candidate.get("score", 0),
                    "match_details": {
                        "similarity_score": candidate.get("score", 50) / 100.0,
                        "key_skills": candidate.get("skills", [])[:3],
                        "experience_match": candidate.get("experience_years", 0) >= 2
                    }
                }
                results.append(formatted)
            
            for job in text_results.get("jobs", []):
                formatted = {
                    "type": "job",
                    "id": job.get("id"),
                    "score": 0.8,  # Default relevance score for text matches
                    "job_title": job.get("job_title", "Unknown"),
                    "company_name": job.get("company_name", "Unknown"),
                    "required_skills": job.get("required_skills", [])[:5],
                    "experience_required": job.get("experience_required", 0),
                    "location": job.get("location", "Remote"),
                    "match_details": {
                        "similarity_score": 0.8,
                        "key_requirements": job.get("required_skills", [])[:3],
                        "experience_level": f"{job.get('experience_required', 0)}+ years"
                    }
                }
                results.append(formatted)
        
        # Sort all results by score
        results.sort(key=lambda x: x.get("score", 0), reverse=True)
        
        # Apply final limit if searching both types
        if search_type == "both":
            results = results[:limit]
        
        # Calculate execution time
        execution_time = (datetime.now() - start_time).total_seconds() * 1000
        
        # Log search query
        log_search_query(
            query, search_type, len(results), 
            int(execution_time), filters
        )
        
        logger.info(f"Search completed: {len(results)} results in {execution_time:.0f}ms")
        
        result = {
            "success": True,
            "query": query,
            "search_type": search_type,
            "total_results": len(results),
            "candidate_count": candidate_count,
            "job_count": job_count,
            "score_threshold": score_threshold,
            "execution_time_ms": int(execution_time),
            "results": results,
            "metadata": {
                "vector_search_used": use_vector_search,
                "timestamp": datetime.now().isoformat(),
                "cached": False
            }
        }
        
        # Cache the result for future searches
        search_cache.set(cache_key, result, ttl=900)  # Cache for 15 minutes
        
        return result
        
    except Exception as e:
        logger.error(f"Error in semantic search: {e}")
        return {
            "success": False,
            "error": f"Search failed: {str(e)}",
            "results": []
        }

# Windmill entry point
def run(
    query: str,
    search_type: str = "both",
    limit: int = 20,
    score_threshold: float = 0.6,
    filters: Dict[str, Any] = None
) -> Dict[str, Any]:
    """
    Windmill-compatible entry point for semantic search
    """
    return main(
        query=query,
        search_type=search_type,
        limit=limit,
        score_threshold=score_threshold,
        filters=filters or {}
    )

if __name__ == "__main__":
    # Test execution
    test_queries = [
        "Python machine learning",
        "Senior software engineer React",
        "Data scientist with PhD",
        "Frontend developer JavaScript"
    ]
    
    for test_query in test_queries:
        print(f"\n--- Testing: {test_query} ---")
        result = main(
            query=test_query,
            search_type="both",
            limit=5,
            score_threshold=0.7
        )
        print(json.dumps(result, indent=2))