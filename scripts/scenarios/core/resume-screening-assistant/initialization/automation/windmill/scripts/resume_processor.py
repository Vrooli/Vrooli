#!/usr/bin/env python3
"""
Resume Processor Script for Resume Screening Assistant
Advanced resume analysis, scoring, and processing capabilities
"""

import json
import requests
import re
from typing import List, Dict, Any, Optional, Tuple
from datetime import datetime
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Resource URLs - should match Vrooli resource ports
UNSTRUCTURED_URL = "http://localhost:11450"
OLLAMA_URL = "http://localhost:11434"
QDRANT_URL = "http://localhost:6333"
POSTGRES_URL = "http://localhost:5433"

class ResumeProcessor:
    """
    Advanced resume processing and analysis class
    """
    
    def __init__(self):
        self.embedding_model = "nomic-embed-text"
        self.analysis_model = "llama3.1:8b"
    
    def extract_text_from_file(self, file_content: bytes, filename: str) -> Dict[str, Any]:
        """
        Extract and parse text from resume files using Unstructured-IO
        """
        try:
            files = {"files": (filename, file_content)}
            data = {
                "strategy": "fast",
                "output_format": "application/json",
                "coordinates": "false"
            }
            
            response = requests.post(
                f"{UNSTRUCTURED_URL}/general/v0/general",
                files=files,
                data=data,
                timeout=60
            )
            
            if response.status_code == 200:
                elements = response.json()
                
                # Extract text from elements
                text_content = ""
                if isinstance(elements, list):
                    text_content = "\n".join([
                        elem.get("text", "") for elem in elements
                        if elem.get("type") in ["NarrativeText", "Title", "ListItem", "Header"]
                    ])
                
                return {
                    "success": True,
                    "text": text_content,
                    "elements": elements,
                    "word_count": len(text_content.split())
                }
            else:
                logger.error(f"Unstructured-IO error: {response.status_code} - {response.text}")
                return {"success": False, "error": f"Document parsing failed: {response.status_code}"}
                
        except Exception as e:
            logger.error(f"Error extracting text: {e}")
            return {"success": False, "error": str(e)}
    
    def extract_contact_info(self, text: str) -> Dict[str, str]:
        """
        Extract contact information using regex patterns
        """
        contact_info = {
            "email": "",
            "phone": "",
            "linkedin": "",
            "github": "",
            "website": ""
        }
        
        try:
            # Email extraction
            email_pattern = r'\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b'
            emails = re.findall(email_pattern, text)
            if emails:
                contact_info["email"] = emails[0]
            
            # Phone extraction
            phone_pattern = r'[\+]?[\d\s\-\(\)]{10,}'
            phones = re.findall(phone_pattern, text)
            if phones:
                # Clean up phone number
                phone = re.sub(r'[^\d\+]', '', phones[0])
                if len(phone) >= 10:
                    contact_info["phone"] = phone
            
            # LinkedIn extraction
            linkedin_pattern = r'linkedin\.com/in/[\w\-]+'
            linkedin_matches = re.findall(linkedin_pattern, text, re.IGNORECASE)
            if linkedin_matches:
                contact_info["linkedin"] = f"https://{linkedin_matches[0]}"
            
            # GitHub extraction
            github_pattern = r'github\.com/[\w\-]+'
            github_matches = re.findall(github_pattern, text, re.IGNORECASE)
            if github_matches:
                contact_info["github"] = f"https://{github_matches[0]}"
            
            # Website extraction (basic)
            website_pattern = r'https?://[^\s]+\.[^\s]+'
            websites = re.findall(website_pattern, text)
            for website in websites:
                if not any(domain in website.lower() for domain in ['linkedin', 'github', 'facebook', 'twitter']):
                    contact_info["website"] = website
                    break
        
        except Exception as e:
            logger.error(f"Error extracting contact info: {e}")
        
        return contact_info
    
    def analyze_with_ai(self, resume_text: str, job_context: str = None) -> Dict[str, Any]:
        """
        Use AI to analyze resume content and extract structured information
        """
        try:
            # Construct analysis prompt
            context_prompt = ""
            if job_context:
                context_prompt = f"\n\nJob Context for Analysis:\n{job_context}\n"
            
            prompt = f"""Analyze this resume and provide a comprehensive assessment in JSON format.{context_prompt}

Resume Text:
{resume_text}

Provide a JSON response with exactly these fields:
{{
    "candidate_name": "Full name of candidate",
    "summary": "Brief professional summary (2-3 sentences)",
    "skills": ["list", "of", "technical", "and", "soft", "skills"],
    "experience_years": 0,
    "education": {{
        "level": "Bachelor's/Master's/PhD/High School/Other",
        "field": "Field of study",
        "institution": "School/University name"
    }},
    "work_experience": [
        {{
            "title": "Job title",
            "company": "Company name",
            "duration": "Time period",
            "key_achievements": ["achievement 1", "achievement 2"]
        }}
    ],
    "key_achievements": ["Notable accomplishments or projects"],
    "certifications": ["Professional certifications or licenses"],
    "languages": ["English", "Spanish", "etc"],
    "overall_score": 85,
    "strengths": ["Top 3-5 strengths"],
    "areas_for_improvement": ["Potential gaps or areas to develop"],
    "job_fit_analysis": {{
        "fit_score": 0,
        "matching_skills": ["skills that match job requirements"],
        "missing_skills": ["skills mentioned in job but not in resume"],
        "recommendation": "Highly Recommended/Recommended/Maybe/Not Recommended"
    }}
}}

Ensure the response is valid JSON only."""
            
            response = requests.post(
                f"{OLLAMA_URL}/api/generate",
                json={
                    "model": self.analysis_model,
                    "prompt": prompt,
                    "stream": False,
                    "format": "json"
                },
                timeout=60
            )
            
            if response.status_code == 200:
                ai_response = response.json().get("response", "{}")
                try:
                    analysis = json.loads(ai_response)
                    
                    # Validate and sanitize the response
                    analysis = self._validate_ai_analysis(analysis)
                    
                    return {
                        "success": True,
                        "analysis": analysis
                    }
                except json.JSONDecodeError as e:
                    logger.error(f"Failed to parse AI response as JSON: {e}")
                    return {
                        "success": False,
                        "error": "AI response was not valid JSON",
                        "raw_response": ai_response
                    }
            else:
                return {
                    "success": False,
                    "error": f"AI analysis failed: {response.status_code}"
                }
                
        except Exception as e:
            logger.error(f"Error in AI analysis: {e}")
            return {
                "success": False,
                "error": str(e)
            }
    
    def _validate_ai_analysis(self, analysis: Dict[str, Any]) -> Dict[str, Any]:
        """
        Validate and sanitize AI analysis response
        """
        validated = {
            "candidate_name": str(analysis.get("candidate_name", "Unknown")).strip(),
            "summary": str(analysis.get("summary", "")).strip(),
            "skills": analysis.get("skills", []) if isinstance(analysis.get("skills"), list) else [],
            "experience_years": max(0, int(analysis.get("experience_years", 0) or 0)),
            "education": analysis.get("education", {}) if isinstance(analysis.get("education"), dict) else {},
            "work_experience": analysis.get("work_experience", []) if isinstance(analysis.get("work_experience"), list) else [],
            "key_achievements": analysis.get("key_achievements", []) if isinstance(analysis.get("key_achievements"), list) else [],
            "certifications": analysis.get("certifications", []) if isinstance(analysis.get("certifications"), list) else [],
            "languages": analysis.get("languages", []) if isinstance(analysis.get("languages"), list) else [],
            "overall_score": max(0, min(100, int(analysis.get("overall_score", 0) or 0))),
            "strengths": analysis.get("strengths", []) if isinstance(analysis.get("strengths"), list) else [],
            "areas_for_improvement": analysis.get("areas_for_improvement", []) if isinstance(analysis.get("areas_for_improvement"), list) else [],
            "job_fit_analysis": analysis.get("job_fit_analysis", {}) if isinstance(analysis.get("job_fit_analysis"), dict) else {}
        }
        
        # Ensure job_fit_analysis has required fields
        if not validated["job_fit_analysis"]:
            validated["job_fit_analysis"] = {
                "fit_score": 0,
                "matching_skills": [],
                "missing_skills": [],
                "recommendation": "Not Recommended"
            }
        
        return validated
    
    def generate_embedding(self, text: str) -> Optional[List[float]]:
        """
        Generate embedding vector for resume text
        """
        try:
            response = requests.post(
                f"{OLLAMA_URL}/api/embeddings",
                json={
                    "model": self.embedding_model,
                    "prompt": text
                },
                timeout=30
            )
            
            if response.status_code == 200:
                return response.json().get("embedding", [])
        except Exception as e:
            logger.error(f"Error generating embedding: {e}")
        return None
    
    def calculate_compatibility_score(
        self, 
        resume_analysis: Dict[str, Any], 
        job_requirements: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Calculate detailed compatibility score between resume and job
        """
        try:
            resume_skills = set([skill.lower().strip() for skill in resume_analysis.get("skills", [])])
            required_skills = set([skill.lower().strip() for skill in job_requirements.get("required_skills", [])])
            preferred_skills = set([skill.lower().strip() for skill in job_requirements.get("preferred_skills", [])])
            
            # Skills matching
            matching_required = resume_skills.intersection(required_skills)
            matching_preferred = resume_skills.intersection(preferred_skills)
            missing_required = required_skills - resume_skills
            
            skills_score = 0
            if required_skills:
                skills_score = (len(matching_required) / len(required_skills)) * 70
            if preferred_skills:
                skills_score += (len(matching_preferred) / len(preferred_skills)) * 30
            
            # Experience matching
            resume_exp = resume_analysis.get("experience_years", 0)
            required_exp = job_requirements.get("experience_required", 0)
            
            if required_exp == 0:
                experience_score = 100
            elif resume_exp >= required_exp:
                experience_score = 100
            else:
                experience_score = max(0, (resume_exp / required_exp) * 100)
            
            # Education matching (basic)
            education_levels = {
                "high school": 1,
                "associate": 2,
                "bachelor": 3,
                "master": 4,
                "phd": 5,
                "doctorate": 5
            }
            
            resume_education = resume_analysis.get("education", {}).get("level", "").lower()
            resume_edu_score = education_levels.get(resume_education, 2)
            education_score = min(100, (resume_edu_score / 3) * 100)  # Normalize to Bachelor's = 100
            
            # Overall compatibility score
            overall_score = (
                skills_score * 0.5 +
                experience_score * 0.3 +
                education_score * 0.2
            )
            
            # Determine recommendation
            if overall_score >= 85:
                recommendation = "Highly Recommended"
            elif overall_score >= 70:
                recommendation = "Recommended"
            elif overall_score >= 55:
                recommendation = "Maybe"
            else:
                recommendation = "Not Recommended"
            
            return {
                "overall_score": round(overall_score, 1),
                "skills_score": round(skills_score, 1),
                "experience_score": round(experience_score, 1),
                "education_score": round(education_score, 1),
                "matching_required_skills": list(matching_required),
                "matching_preferred_skills": list(matching_preferred),
                "missing_required_skills": list(missing_required),
                "recommendation": recommendation,
                "details": {
                    "resume_experience": resume_exp,
                    "required_experience": required_exp,
                    "education_match": resume_education
                }
            }
            
        except Exception as e:
            logger.error(f"Error calculating compatibility: {e}")
            return {
                "overall_score": 0,
                "skills_score": 0,
                "experience_score": 0,
                "education_score": 0,
                "matching_required_skills": [],
                "matching_preferred_skills": [],
                "missing_required_skills": [],
                "recommendation": "Not Recommended",
                "error": str(e)
            }

def process_resume_file(
    file_content: bytes,
    filename: str,
    job_id: Optional[int] = None,
    job_context: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """
    Main function to process a resume file
    
    Args:
        file_content: Binary content of the resume file
        filename: Name of the uploaded file
        job_id: Optional job ID for context-aware analysis
        job_context: Optional job requirements for compatibility analysis
    
    Returns:
        Dictionary with processing results
    """
    processor = ResumeProcessor()
    start_time = datetime.now()
    
    logger.info(f"Starting resume processing: {filename}")
    
    try:
        # Step 1: Extract text from file
        extraction_result = processor.extract_text_from_file(file_content, filename)
        if not extraction_result.get("success"):
            return {
                "success": False,
                "error": extraction_result.get("error", "Text extraction failed"),
                "stage": "text_extraction"
            }
        
        resume_text = extraction_result["text"]
        if not resume_text or len(resume_text.strip()) < 50:
            return {
                "success": False,
                "error": "Extracted text is too short or empty",
                "stage": "text_validation"
            }
        
        # Step 2: Extract contact information
        contact_info = processor.extract_contact_info(resume_text)
        
        # Step 3: AI analysis
        job_context_text = None
        if job_context:
            job_context_text = f"""
            Job Title: {job_context.get('job_title', '')}
            Company: {job_context.get('company_name', '')}
            Required Skills: {', '.join(job_context.get('required_skills', []))}
            Experience Required: {job_context.get('experience_required', 0)} years
            Description: {job_context.get('description', '')}
            """
        
        analysis_result = processor.analyze_with_ai(resume_text, job_context_text)
        if not analysis_result.get("success"):
            return {
                "success": False,
                "error": analysis_result.get("error", "AI analysis failed"),
                "stage": "ai_analysis"
            }
        
        analysis = analysis_result["analysis"]
        
        # Step 4: Generate embedding
        embedding = processor.generate_embedding(resume_text)
        if not embedding:
            logger.warning("Failed to generate embedding, but continuing without it")
        
        # Step 5: Calculate compatibility score if job context provided
        compatibility = {}
        if job_context:
            compatibility = processor.calculate_compatibility_score(analysis, job_context)
        
        # Step 6: Prepare final result
        processing_time = (datetime.now() - start_time).total_seconds() * 1000
        
        result = {
            "success": True,
            "processing_time_ms": int(processing_time),
            "file_info": {
                "filename": filename,
                "word_count": extraction_result.get("word_count", 0),
                "file_size": len(file_content)
            },
            "contact_info": contact_info,
            "analysis": analysis,
            "embedding": embedding,
            "compatibility": compatibility,
            "job_id": job_id,
            "processed_at": datetime.now().isoformat(),
            "metadata": {
                "processor_version": "1.0.0",
                "models_used": {
                    "analysis": processor.analysis_model,
                    "embedding": processor.embedding_model
                }
            }
        }
        
        logger.info(f"Resume processing completed in {processing_time:.0f}ms")
        return result
        
    except Exception as e:
        logger.error(f"Error processing resume: {e}")
        return {
            "success": False,
            "error": str(e),
            "stage": "unknown"
        }

# Windmill entry point
def run(
    file_content_base64: str,
    filename: str,
    job_id: Optional[int] = None,
    job_context: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """
    Windmill-compatible entry point for resume processing
    """
    try:
        import base64
        file_content = base64.b64decode(file_content_base64)
        
        return process_resume_file(
            file_content=file_content,
            filename=filename,
            job_id=job_id,
            job_context=job_context
        )
    except Exception as e:
        return {
            "success": False,
            "error": f"Failed to decode file content: {str(e)}"
        }

if __name__ == "__main__":
    # Test with sample data
    sample_text = """
    John Doe
    john.doe@email.com
    (555) 123-4567
    
    Senior Software Engineer with 5 years of experience in Python, JavaScript, React, and Node.js.
    Master's degree in Computer Science from MIT.
    
    Experience:
    - Senior Developer at Tech Corp (2019-2024)
    - Built scalable web applications using React and Node.js
    - Led team of 4 developers
    
    Skills: Python, JavaScript, React, Node.js, PostgreSQL, Docker
    """
    
    # Mock file content
    file_bytes = sample_text.encode('utf-8')
    
    result = process_resume_file(
        file_content=file_bytes,
        filename="john_doe_resume.txt",
        job_id=1,
        job_context={
            "job_title": "Senior Software Engineer",
            "company_name": "Tech Innovations Inc",
            "required_skills": ["JavaScript", "React", "Node.js", "PostgreSQL"],
            "preferred_skills": ["Python", "Docker"],
            "experience_required": 5,
            "description": "Looking for experienced full-stack developer"
        }
    )
    
    print(json.dumps(result, indent=2))