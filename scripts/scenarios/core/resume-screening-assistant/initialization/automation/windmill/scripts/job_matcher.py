#!/usr/bin/env python3
"""
Job Matcher Script for Resume Screening Assistant
Matches candidates to job postings using vector similarity and AI assessment
"""

import json
import requests
from typing import List, Dict, Any, Optional
from datetime import datetime
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Resource URLs
QDRANT_URL = "http://localhost:6333"
OLLAMA_URL = "http://localhost:11434"

def fetch_job_requirements(job_id: str) -> Optional[Dict[str, Any]]:
    """
    Fetch job requirements from the database
    """
    try:
        response = requests.get(
            f"{QDRANT_URL}/collections/job_requirements/points/{job_id}"
        )
        if response.status_code == 200:
            return response.json().get("result", {})
    except Exception as e:
        logger.error(f"Error fetching job requirements: {e}")
    return None

def search_candidates(
    job_vector: List[float], 
    limit: int = 10,
    score_threshold: float = 0.7
) -> List[Dict[str, Any]]:
    """
    Search for matching candidates using vector similarity
    """
    try:
        search_payload = {
            "vector": job_vector,
            "limit": limit,
            "score_threshold": score_threshold,
            "with_payload": True
        }
        
        response = requests.post(
            f"{QDRANT_URL}/collections/candidate_profiles/points/search",
            json=search_payload
        )
        
        if response.status_code == 200:
            return response.json().get("result", [])
    except Exception as e:
        logger.error(f"Error searching candidates: {e}")
    return []

def assess_candidate_fit(
    candidate: Dict[str, Any],
    job_requirements: str,
    model: str = "llama3.1:8b"
) -> Dict[str, Any]:
    """
    Use AI to assess candidate fit for the job
    """
    try:
        prompt = f"""
        Assess this candidate's fit for the job based on the following:
        
        Job Requirements:
        {job_requirements}
        
        Candidate Profile:
        {json.dumps(candidate.get('payload', {}), indent=2)}
        
        Provide a detailed assessment including:
        1. Overall fit score (0-100)
        2. Strengths (list 3-5)
        3. Gaps or concerns (list any)
        4. Hiring recommendation (Highly Recommended/Recommended/Maybe/Not Recommended)
        5. Interview focus areas
        
        Format as JSON.
        """
        
        response = requests.post(
            f"{OLLAMA_URL}/api/generate",
            json={
                "model": model,
                "prompt": prompt,
                "stream": False,
                "format": "json"
            }
        )
        
        if response.status_code == 200:
            assessment_text = response.json().get("response", "{}")
            try:
                return json.loads(assessment_text)
            except json.JSONDecodeError:
                return {
                    "fit_score": 0,
                    "assessment": assessment_text,
                    "error": "Failed to parse AI response as JSON"
                }
    except Exception as e:
        logger.error(f"Error assessing candidate: {e}")
        return {"error": str(e)}

def rank_candidates(
    candidates: List[Dict[str, Any]]
) -> List[Dict[str, Any]]:
    """
    Rank candidates based on their assessment scores
    """
    def get_score(candidate):
        return candidate.get("assessment", {}).get("fit_score", 0)
    
    return sorted(candidates, key=get_score, reverse=True)

def generate_shortlist(
    ranked_candidates: List[Dict[str, Any]],
    max_candidates: int = 5
) -> Dict[str, Any]:
    """
    Generate a shortlist report
    """
    shortlist = ranked_candidates[:max_candidates]
    
    report = {
        "generated_at": datetime.now().isoformat(),
        "total_candidates_evaluated": len(ranked_candidates),
        "shortlist_size": len(shortlist),
        "candidates": []
    }
    
    for i, candidate in enumerate(shortlist, 1):
        candidate_summary = {
            "rank": i,
            "id": candidate.get("id"),
            "score": candidate.get("score", 0),
            "fit_score": candidate.get("assessment", {}).get("fit_score", 0),
            "recommendation": candidate.get("assessment", {}).get("recommendation", "Unknown"),
            "strengths": candidate.get("assessment", {}).get("strengths", []),
            "concerns": candidate.get("assessment", {}).get("gaps", []),
            "interview_focus": candidate.get("assessment", {}).get("interview_focus_areas", [])
        }
        report["candidates"].append(candidate_summary)
    
    return report

def main(
    job_id: str,
    job_description: str = None,
    max_candidates: int = 20,
    shortlist_size: int = 5,
    score_threshold: float = 0.6
) -> Dict[str, Any]:
    """
    Main job matching workflow
    """
    logger.info(f"Starting job matching for job_id: {job_id}")
    
    # Step 1: Get job requirements
    job_data = fetch_job_requirements(job_id) if job_id else {}
    
    if not job_data and not job_description:
        return {
            "error": "No job data found and no job description provided",
            "job_id": job_id
        }
    
    # Use provided description or fetched data
    job_requirements = job_description or job_data.get("payload", {}).get("description", "")
    
    # Step 2: Generate job embedding if not available
    if "vector" not in job_data:
        logger.info("Generating job embedding...")
        response = requests.post(
            f"{OLLAMA_URL}/api/embeddings",
            json={
                "model": "llama3.1:8b",
                "prompt": job_requirements
            }
        )
        if response.status_code == 200:
            job_vector = response.json().get("embedding", [])
        else:
            return {"error": "Failed to generate job embedding"}
    else:
        job_vector = job_data["vector"]
    
    # Step 3: Search for matching candidates
    logger.info(f"Searching for candidates (threshold: {score_threshold})...")
    matching_candidates = search_candidates(
        job_vector, 
        limit=max_candidates,
        score_threshold=score_threshold
    )
    
    if not matching_candidates:
        return {
            "job_id": job_id,
            "message": "No matching candidates found",
            "suggestion": "Consider lowering the score threshold or expanding search criteria"
        }
    
    logger.info(f"Found {len(matching_candidates)} matching candidates")
    
    # Step 4: Assess each candidate
    logger.info("Assessing candidate fit...")
    assessed_candidates = []
    for candidate in matching_candidates:
        assessment = assess_candidate_fit(candidate, job_requirements)
        candidate["assessment"] = assessment
        assessed_candidates.append(candidate)
    
    # Step 5: Rank candidates
    logger.info("Ranking candidates...")
    ranked_candidates = rank_candidates(assessed_candidates)
    
    # Step 6: Generate shortlist
    logger.info(f"Generating shortlist (top {shortlist_size})...")
    shortlist = generate_shortlist(ranked_candidates, shortlist_size)
    
    # Step 7: Store results
    result = {
        "job_id": job_id,
        "timestamp": datetime.now().isoformat(),
        "candidates_evaluated": len(matching_candidates),
        "shortlist": shortlist,
        "status": "success"
    }
    
    logger.info(f"Job matching complete. Shortlisted {len(shortlist['candidates'])} candidates")
    
    return result

# Windmill entry point
def run(
    job_id: str = "test_job_001",
    job_description: str = None,
    max_candidates: int = 20,
    shortlist_size: int = 5,
    score_threshold: float = 0.6
) -> Dict[str, Any]:
    """
    Windmill-compatible entry point
    """
    return main(
        job_id=job_id,
        job_description=job_description,
        max_candidates=max_candidates,
        shortlist_size=shortlist_size,
        score_threshold=score_threshold
    )

if __name__ == "__main__":
    # Test execution
    result = main(
        job_id="test_job_001",
        job_description="Senior Software Engineer with Python, React, and machine learning experience",
        max_candidates=10,
        shortlist_size=3
    )
    print(json.dumps(result, indent=2))