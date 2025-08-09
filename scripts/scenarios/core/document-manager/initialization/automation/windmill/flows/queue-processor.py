#!/usr/bin/env python3
"""
Queue Processor Flow
Handles improvement queue operations including approval, denial, and batch processing
"""

import psycopg2
import json
import uuid
import requests
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any
import os

# Database connection helper
def get_db_connection():
    """Get PostgreSQL database connection"""
    conn = psycopg2.connect(
        host=os.getenv("POSTGRES_HOST", "localhost"),
        port=os.getenv("POSTGRES_PORT", "5433"),
        database=os.getenv("POSTGRES_DB", "document_manager"),
        user=os.getenv("POSTGRES_USER", "postgres"),
        password=os.getenv("POSTGRES_PASSWORD", "password")
    )
    return conn

def approve_improvement(
    improvement_id: str,
    reviewed_by: str,
    review_notes: Optional[str] = None,
    auto_apply: bool = False
) -> Dict[str, Any]:
    """Approve an improvement suggestion"""
    
    conn = get_db_connection()
    
    try:
        with conn.cursor() as cur:
            # Get improvement details
            cur.execute("""
                SELECT iq.*, a.name as app_name, ag.name as agent_name
                FROM improvement_queue iq
                JOIN applications a ON iq.application_id = a.id
                JOIN agents ag ON iq.agent_id = ag.id
                WHERE iq.id = %s AND iq.status = 'pending'
            """, (improvement_id,))
            
            improvement = cur.fetchone()
            if not improvement:
                raise ValueError(f"Improvement {improvement_id} not found or not in pending status")
            
            # Update status to approved
            cur.execute("""
                UPDATE improvement_queue
                SET status = 'approved', 
                    reviewed_at = %s, 
                    reviewed_by = %s, 
                    review_notes = %s,
                    updated_at = %s
                WHERE id = %s
                RETURNING *
            """, (datetime.now(), reviewed_by, review_notes, datetime.now(), improvement_id))
            
            updated_improvement = cur.fetchone()
            conn.commit()
            
            # Log the approval
            cur.execute("""
                INSERT INTO action_history (
                    queue_item_id, agent_id, action_type, action_data, 
                    result, created_at
                ) VALUES (%s, %s, 'approve', %s, 'success', %s)
            """, (
                improvement_id, 
                improvement[1],  # agent_id
                json.dumps({"reviewed_by": reviewed_by, "auto_apply": auto_apply}),
                datetime.now()
            ))
            conn.commit()
            
            # If auto-apply is enabled and the improvement supports it, apply immediately
            result = {
                "success": True,
                "message": f"Improvement '{improvement[5]}' approved successfully",
                "improvement_id": improvement_id,
                "auto_applied": False
            }
            
            if auto_apply and improvement[7]:  # suggested_fix exists
                apply_result = apply_improvement(improvement_id, reviewed_by)
                result["auto_applied"] = apply_result.get("success", False)
                result["apply_details"] = apply_result
            
            return result
            
    except Exception as e:
        conn.rollback()
        return {
            "success": False,
            "error": str(e)
        }
    finally:
        conn.close()

def deny_improvement(
    improvement_id: str,
    reviewed_by: str,
    review_notes: str
) -> Dict[str, Any]:
    """Deny an improvement suggestion"""
    
    if not review_notes or not review_notes.strip():
        return {
            "success": False,
            "error": "Review notes are required when denying an improvement"
        }
    
    conn = get_db_connection()
    
    try:
        with conn.cursor() as cur:
            # Verify improvement exists and is pending
            cur.execute("""
                SELECT title, agent_id 
                FROM improvement_queue 
                WHERE id = %s AND status = 'pending'
            """, (improvement_id,))
            
            improvement = cur.fetchone()
            if not improvement:
                raise ValueError(f"Improvement {improvement_id} not found or not in pending status")
            
            # Update status to denied
            cur.execute("""
                UPDATE improvement_queue
                SET status = 'denied', 
                    reviewed_at = %s, 
                    reviewed_by = %s, 
                    review_notes = %s,
                    updated_at = %s
                WHERE id = %s
            """, (datetime.now(), reviewed_by, review_notes.strip(), datetime.now(), improvement_id))
            
            conn.commit()
            
            # Log the denial
            cur.execute("""
                INSERT INTO action_history (
                    queue_item_id, agent_id, action_type, action_data, 
                    result, created_at
                ) VALUES (%s, %s, 'deny', %s, 'success', %s)
            """, (
                improvement_id, 
                improvement[1],  # agent_id
                json.dumps({"reviewed_by": reviewed_by, "reason": review_notes.strip()}),
                datetime.now()
            ))
            conn.commit()
            
            return {
                "success": True,
                "message": f"Improvement '{improvement[0]}' denied successfully",
                "improvement_id": improvement_id
            }
            
    except Exception as e:
        conn.rollback()
        return {
            "success": False,
            "error": str(e)
        }
    finally:
        conn.close()

def request_revision(
    improvement_id: str,
    reviewed_by: str,
    review_notes: str
) -> Dict[str, Any]:
    """Request revision for an improvement suggestion"""
    
    if not review_notes or not review_notes.strip():
        return {
            "success": False,
            "error": "Review notes are required when requesting revision"
        }
    
    conn = get_db_connection()
    
    try:
        with conn.cursor() as cur:
            # Verify improvement exists and is pending
            cur.execute("""
                SELECT title, agent_id 
                FROM improvement_queue 
                WHERE id = %s AND status = 'pending'
            """, (improvement_id,))
            
            improvement = cur.fetchone()
            if not improvement:
                raise ValueError(f"Improvement {improvement_id} not found or not in pending status")
            
            # Update status to revision_requested
            cur.execute("""
                UPDATE improvement_queue
                SET status = 'revision_requested', 
                    reviewed_at = %s, 
                    reviewed_by = %s, 
                    review_notes = %s,
                    updated_at = %s
                WHERE id = %s
            """, (datetime.now(), reviewed_by, review_notes.strip(), datetime.now(), improvement_id))
            
            conn.commit()
            
            # Log the revision request
            cur.execute("""
                INSERT INTO action_history (
                    queue_item_id, agent_id, action_type, action_data, 
                    result, created_at
                ) VALUES (%s, %s, 'request_revision', %s, 'success', %s)
            """, (
                improvement_id, 
                improvement[1],  # agent_id
                json.dumps({"reviewed_by": reviewed_by, "revision_notes": review_notes.strip()}),
                datetime.now()
            ))
            conn.commit()
            
            # TODO: In a real implementation, this would trigger the agent to regenerate the suggestion
            # For now, we just log the request
            
            return {
                "success": True,
                "message": f"Revision requested for '{improvement[0]}'",
                "improvement_id": improvement_id
            }
            
    except Exception as e:
        conn.rollback()
        return {
            "success": False,
            "error": str(e)
        }
    finally:
        conn.close()

def apply_improvement(
    improvement_id: str,
    applied_by: str
) -> Dict[str, Any]:
    """Apply an approved improvement"""
    
    conn = get_db_connection()
    
    try:
        with conn.cursor() as cur:
            # Get approved improvement details
            cur.execute("""
                SELECT iq.*, a.repository_url, a.documentation_path
                FROM improvement_queue iq
                JOIN applications a ON iq.application_id = a.id
                WHERE iq.id = %s AND iq.status = 'approved'
            """, (improvement_id,))
            
            improvement = cur.fetchone()
            if not improvement:
                raise ValueError(f"Improvement {improvement_id} not found or not approved")
            
            suggested_fix = json.loads(improvement[7]) if improvement[7] else {}
            
            # Simulate applying the improvement
            # In a real implementation, this would:
            # 1. Clone the repository
            # 2. Make the suggested changes
            # 3. Create a pull request or commit directly
            # 4. Verify the changes work correctly
            
            apply_result = {
                "files_modified": suggested_fix.get("files", []),
                "changes_count": len(suggested_fix.get("suggested_changes", [])),
                "confidence_score": suggested_fix.get("confidence", 0.5),
                "timestamp": datetime.now().isoformat(),
                "method": "simulated"  # In real implementation: "pull_request" or "direct_commit"
            }
            
            # Update improvement status to applied
            cur.execute("""
                UPDATE improvement_queue
                SET status = 'applied', 
                    applied_at = %s, 
                    applied_by = %s, 
                    applied_result = %s,
                    revert_possible = %s,
                    updated_at = %s
                WHERE id = %s
            """, (
                datetime.now(), applied_by, json.dumps(apply_result), 
                True, datetime.now(), improvement_id
            ))
            
            conn.commit()
            
            # Log the application
            cur.execute("""
                INSERT INTO action_history (
                    queue_item_id, agent_id, action_type, action_data, 
                    result, execution_time_ms, created_at
                ) VALUES (%s, %s, 'apply', %s, 'success', %s, %s)
            """, (
                improvement_id, 
                improvement[1],  # agent_id
                json.dumps(apply_result),
                1500,  # Simulated execution time
                datetime.now()
            ))
            conn.commit()
            
            return {
                "success": True,
                "message": f"Improvement '{improvement[5]}' applied successfully",
                "improvement_id": improvement_id,
                "apply_result": apply_result
            }
            
    except Exception as e:
        conn.rollback()
        return {
            "success": False,
            "error": str(e)
        }
    finally:
        conn.close()

def batch_process_improvements(
    improvement_ids: List[str],
    action: str,
    reviewed_by: str,
    review_notes: Optional[str] = None
) -> Dict[str, Any]:
    """Process multiple improvements in batch"""
    
    if action not in ["approve", "deny", "request_revision"]:
        return {
            "success": False,
            "error": "Invalid batch action. Must be 'approve', 'deny', or 'request_revision'"
        }
    
    if not improvement_ids:
        return {
            "success": False,
            "error": "No improvement IDs provided"
        }
    
    results = []
    success_count = 0
    error_count = 0
    
    for improvement_id in improvement_ids:
        try:
            if action == "approve":
                result = approve_improvement(improvement_id, reviewed_by, review_notes)
            elif action == "deny":
                result = deny_improvement(improvement_id, reviewed_by, review_notes or "Batch denial")
            else:  # request_revision
                result = request_revision(improvement_id, reviewed_by, review_notes or "Batch revision request")
            
            if result["success"]:
                success_count += 1
            else:
                error_count += 1
                
            results.append({
                "improvement_id": improvement_id,
                "success": result["success"],
                "message": result.get("message", ""),
                "error": result.get("error", "")
            })
            
        except Exception as e:
            error_count += 1
            results.append({
                "improvement_id": improvement_id,
                "success": False,
                "error": str(e)
            })
    
    return {
        "success": True,
        "message": f"Batch operation completed: {success_count} succeeded, {error_count} failed",
        "summary": {
            "total_processed": len(improvement_ids),
            "successful": success_count,
            "failed": error_count,
            "action": action
        },
        "results": results
    }

def get_queue_statistics(
    application_id: Optional[str] = None,
    days: int = 30
) -> Dict[str, Any]:
    """Get comprehensive queue statistics"""
    
    conn = get_db_connection()
    
    try:
        with conn.cursor() as cur:
            # Base WHERE clause
            where_clause = "WHERE iq.created_at >= %s"
            params = [datetime.now() - timedelta(days=days)]
            
            if application_id:
                where_clause += " AND iq.application_id = %s"
                params.append(application_id)
            
            # Get status distribution
            cur.execute(f"""
                SELECT iq.status, COUNT(*) as count
                FROM improvement_queue iq
                {where_clause}
                GROUP BY iq.status
                ORDER BY count DESC
            """, params)
            
            status_distribution = {row[0]: row[1] for row in cur.fetchall()}
            
            # Get severity distribution
            cur.execute(f"""
                SELECT iq.severity, COUNT(*) as count
                FROM improvement_queue iq
                {where_clause}
                GROUP BY iq.severity
                ORDER BY 
                    CASE iq.severity
                        WHEN 'critical' THEN 1
                        WHEN 'high' THEN 2
                        WHEN 'medium' THEN 3
                        WHEN 'low' THEN 4
                    END
            """, params)
            
            severity_distribution = {row[0]: row[1] for row in cur.fetchall()}
            
            # Get type distribution
            cur.execute(f"""
                SELECT iq.type, COUNT(*) as count
                FROM improvement_queue iq
                {where_clause}
                GROUP BY iq.type
                ORDER BY count DESC
            """, params)
            
            type_distribution = {row[0]: row[1] for row in cur.fetchall()}
            
            # Get processing time statistics (for completed items)
            cur.execute(f"""
                SELECT 
                    AVG(EXTRACT(EPOCH FROM (reviewed_at - created_at))/3600) as avg_review_time_hours,
                    MIN(EXTRACT(EPOCH FROM (reviewed_at - created_at))/3600) as min_review_time_hours,
                    MAX(EXTRACT(EPOCH FROM (reviewed_at - created_at))/3600) as max_review_time_hours
                FROM improvement_queue iq
                {where_clause}
                AND iq.reviewed_at IS NOT NULL
            """, params)
            
            timing_stats = cur.fetchone()
            
            # Get top agents by suggestions
            cur.execute(f"""
                SELECT ag.name, ag.type, COUNT(*) as suggestion_count
                FROM improvement_queue iq
                JOIN agents ag ON iq.agent_id = ag.id
                {where_clause}
                GROUP BY ag.id, ag.name, ag.type
                ORDER BY suggestion_count DESC
                LIMIT 10
            """, params)
            
            top_agents = [
                {"name": row[0], "type": row[1], "suggestions": row[2]}
                for row in cur.fetchall()
            ]
            
            return {
                "success": True,
                "period_days": days,
                "application_filter": application_id,
                "statistics": {
                    "status_distribution": status_distribution,
                    "severity_distribution": severity_distribution,
                    "type_distribution": type_distribution,
                    "total_items": sum(status_distribution.values()),
                    "pending_items": status_distribution.get("pending", 0),
                    "processing_times": {
                        "average_review_hours": round(timing_stats[0], 2) if timing_stats[0] else 0,
                        "min_review_hours": round(timing_stats[1], 2) if timing_stats[1] else 0,
                        "max_review_hours": round(timing_stats[2], 2) if timing_stats[2] else 0
                    } if timing_stats else {},
                    "top_agents": top_agents
                }
            }
            
    except Exception as e:
        return {
            "success": False,
            "error": str(e)
        }
    finally:
        conn.close()

def add_improvement_feedback(
    improvement_id: str,
    feedback_type: str,
    effectiveness_rating: int,
    feedback_text: str,
    would_revert: bool,
    created_by: str,
    additional_suggestions: Optional[str] = None
) -> Dict[str, Any]:
    """Add user feedback for an applied improvement"""
    
    if feedback_type not in ["positive", "negative", "neutral"]:
        return {"success": False, "error": "Invalid feedback type"}
    
    if not (1 <= effectiveness_rating <= 5):
        return {"success": False, "error": "Effectiveness rating must be between 1 and 5"}
    
    conn = get_db_connection()
    
    try:
        with conn.cursor() as cur:
            # Verify improvement exists and is applied
            cur.execute("""
                SELECT title FROM improvement_queue 
                WHERE id = %s AND status = 'applied'
            """, (improvement_id,))
            
            improvement = cur.fetchone()
            if not improvement:
                raise ValueError(f"Improvement {improvement_id} not found or not applied")
            
            # Add feedback
            feedback_id = str(uuid.uuid4())
            cur.execute("""
                INSERT INTO improvement_feedback (
                    id, improvement_id, feedback_type, effectiveness_rating,
                    feedback_text, would_revert, additional_suggestions, created_by, created_at
                ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)
            """, (
                feedback_id, improvement_id, feedback_type, effectiveness_rating,
                feedback_text, would_revert, additional_suggestions, created_by, datetime.now()
            ))
            
            conn.commit()
            
            return {
                "success": True,
                "message": f"Feedback added for improvement '{improvement[0]}'",
                "feedback_id": feedback_id
            }
            
    except Exception as e:
        conn.rollback()
        return {
            "success": False,
            "error": str(e)
        }
    finally:
        conn.close()

# Windmill flow entry points
def main(
    action: str,
    improvement_id: Optional[str] = None,
    improvement_ids: Optional[List[str]] = None,
    reviewed_by: Optional[str] = None,
    **kwargs
) -> Dict[str, Any]:
    """Main entry point for queue processing operations"""
    
    if action == "approve":
        return approve_improvement(improvement_id, reviewed_by, **kwargs)
    elif action == "deny":
        return deny_improvement(improvement_id, reviewed_by, kwargs.get("review_notes", ""))
    elif action == "request_revision":
        return request_revision(improvement_id, reviewed_by, kwargs.get("review_notes", ""))
    elif action == "apply":
        return apply_improvement(improvement_id, kwargs.get("applied_by", reviewed_by))
    elif action == "batch_process":
        return batch_process_improvements(improvement_ids, kwargs.get("batch_action"), reviewed_by, **kwargs)
    elif action == "statistics":
        return get_queue_statistics(**kwargs)
    elif action == "add_feedback":
        return add_improvement_feedback(improvement_id, **kwargs)
    else:
        return {
            "success": False,
            "error": f"Unknown action: {action}"
        }