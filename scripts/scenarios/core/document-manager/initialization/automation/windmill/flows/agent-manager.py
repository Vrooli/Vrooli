#!/usr/bin/env python3
"""
Agent Management Flow
Handles CRUD operations for documentation agents
"""

import psycopg2
import json
import uuid
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

def create_agent(
    name: str,
    agent_type: str,
    application_id: str,
    config: Dict[str, Any],
    schedule_cron: Optional[str] = None,
    auto_apply_threshold: float = 0.0,
    enabled: bool = True
) -> Dict[str, Any]:
    """Create a new documentation agent"""
    
    agent_id = str(uuid.uuid4())
    conn = get_db_connection()
    
    try:
        with conn.cursor() as cur:
            # Validate application exists
            cur.execute("SELECT id FROM applications WHERE id = %s AND active = true", (application_id,))
            if not cur.fetchone():
                raise ValueError(f"Application {application_id} not found or inactive")
            
            # Calculate next run time if schedule provided
            next_run = None
            if schedule_cron and enabled:
                # Simple next run calculation - in real implementation use croniter
                next_run = datetime.now() + timedelta(hours=1)
            
            # Insert new agent
            cur.execute("""
                INSERT INTO agents (
                    id, name, type, application_id, config, schedule_cron, 
                    auto_apply_threshold, enabled, created_at, next_run
                ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                RETURNING *
            """, (
                agent_id, name, agent_type, application_id, json.dumps(config),
                schedule_cron, auto_apply_threshold, enabled, datetime.now(), next_run
            ))
            
            agent = cur.fetchone()
            conn.commit()
            
            # Log creation
            cur.execute("""
                INSERT INTO audit_log (entity_type, entity_id, action, new_values, performed_by, created_at)
                VALUES ('agent', %s, 'create', %s, %s, %s)
            """, (agent_id, json.dumps({"name": name, "type": agent_type}), "system", datetime.now()))
            conn.commit()
            
            return {
                "success": True,
                "agent_id": agent_id,
                "message": f"Agent '{name}' created successfully"
            }
            
    except Exception as e:
        conn.rollback()
        return {
            "success": False,
            "error": str(e)
        }
    finally:
        conn.close()

def update_agent(
    agent_id: str,
    name: Optional[str] = None,
    config: Optional[Dict[str, Any]] = None,
    schedule_cron: Optional[str] = None,
    auto_apply_threshold: Optional[float] = None,
    enabled: Optional[bool] = None
) -> Dict[str, Any]:
    """Update an existing agent"""
    
    conn = get_db_connection()
    
    try:
        with conn.cursor() as cur:
            # Get current agent data for audit log
            cur.execute("SELECT * FROM agents WHERE id = %s", (agent_id,))
            current_agent = cur.fetchone()
            if not current_agent:
                raise ValueError(f"Agent {agent_id} not found")
            
            # Build update query dynamically
            update_fields = []
            update_values = []
            
            if name is not None:
                update_fields.append("name = %s")
                update_values.append(name)
                
            if config is not None:
                update_fields.append("config = %s")
                update_values.append(json.dumps(config))
                
            if schedule_cron is not None:
                update_fields.append("schedule_cron = %s")
                update_values.append(schedule_cron)
                
            if auto_apply_threshold is not None:
                update_fields.append("auto_apply_threshold = %s")
                update_values.append(auto_apply_threshold)
                
            if enabled is not None:
                update_fields.append("enabled = %s")
                update_values.append(enabled)
                
                # Update next_run based on enabled status
                if enabled and schedule_cron:
                    update_fields.append("next_run = %s")
                    update_values.append(datetime.now() + timedelta(hours=1))
                elif not enabled:
                    update_fields.append("next_run = %s")
                    update_values.append(None)
            
            if not update_fields:
                return {"success": True, "message": "No changes to update"}
            
            # Add updated_at
            update_fields.append("updated_at = %s")
            update_values.append(datetime.now())
            update_values.append(agent_id)
            
            # Execute update
            cur.execute(f"""
                UPDATE agents 
                SET {', '.join(update_fields)}
                WHERE id = %s
                RETURNING *
            """, update_values)
            
            updated_agent = cur.fetchone()
            conn.commit()
            
            # Log update
            changes = {}
            if name: changes["name"] = name
            if config: changes["config"] = config
            if enabled is not None: changes["enabled"] = enabled
            
            cur.execute("""
                INSERT INTO audit_log (entity_type, entity_id, action, new_values, performed_by, created_at)
                VALUES ('agent', %s, 'update', %s, %s, %s)
            """, (agent_id, json.dumps(changes), "system", datetime.now()))
            conn.commit()
            
            return {
                "success": True,
                "message": f"Agent updated successfully",
                "agent": dict(zip([desc[0] for desc in cur.description], updated_agent))
            }
            
    except Exception as e:
        conn.rollback()
        return {
            "success": False,
            "error": str(e)
        }
    finally:
        conn.close()

def delete_agent(agent_id: str) -> Dict[str, Any]:
    """Delete an agent (soft delete by disabling)"""
    
    conn = get_db_connection()
    
    try:
        with conn.cursor() as cur:
            # Check if agent exists and get name for logging
            cur.execute("SELECT name FROM agents WHERE id = %s", (agent_id,))
            agent = cur.fetchone()
            if not agent:
                raise ValueError(f"Agent {agent_id} not found")
            
            agent_name = agent[0]
            
            # Soft delete by disabling
            cur.execute("""
                UPDATE agents 
                SET enabled = false, next_run = NULL, updated_at = %s 
                WHERE id = %s
            """, (datetime.now(), agent_id))
            
            conn.commit()
            
            # Log deletion
            cur.execute("""
                INSERT INTO audit_log (entity_type, entity_id, action, old_values, performed_by, created_at)
                VALUES ('agent', %s, 'delete', %s, %s, %s)
            """, (agent_id, json.dumps({"name": agent_name}), "system", datetime.now()))
            conn.commit()
            
            return {
                "success": True,
                "message": f"Agent '{agent_name}' deleted successfully"
            }
            
    except Exception as e:
        conn.rollback()
        return {
            "success": False,
            "error": str(e)
        }
    finally:
        conn.close()

def get_agent_performance_metrics(agent_id: str, days: int = 30) -> Dict[str, Any]:
    """Get performance metrics for an agent"""
    
    conn = get_db_connection()
    
    try:
        with conn.cursor() as cur:
            # Get agent info
            cur.execute("SELECT name, type FROM agents WHERE id = %s", (agent_id,))
            agent_info = cur.fetchone()
            if not agent_info:
                raise ValueError(f"Agent {agent_id} not found")
            
            # Get metrics for the specified period
            cur.execute("""
                SELECT 
                    metric_date,
                    runs_count,
                    suggestions_count,
                    approved_count,
                    denied_count,
                    revision_count,
                    avg_execution_time_ms,
                    error_count
                FROM agent_metrics 
                WHERE agent_id = %s 
                AND metric_date >= %s 
                ORDER BY metric_date DESC
            """, (agent_id, datetime.now().date() - timedelta(days=days)))
            
            metrics = cur.fetchall()
            
            # Calculate summary stats
            total_runs = sum(m[1] for m in metrics)
            total_suggestions = sum(m[2] for m in metrics)
            total_approved = sum(m[3] for m in metrics)
            total_errors = sum(m[7] for m in metrics)
            
            approval_rate = (total_approved / total_suggestions * 100) if total_suggestions > 0 else 0
            error_rate = (total_errors / total_runs * 100) if total_runs > 0 else 0
            
            return {
                "success": True,
                "agent_name": agent_info[0],
                "agent_type": agent_info[1],
                "period_days": days,
                "summary": {
                    "total_runs": total_runs,
                    "total_suggestions": total_suggestions,
                    "total_approved": total_approved,
                    "approval_rate": round(approval_rate, 2),
                    "error_rate": round(error_rate, 2)
                },
                "daily_metrics": [
                    {
                        "date": m[0].isoformat(),
                        "runs": m[1],
                        "suggestions": m[2],
                        "approved": m[3],
                        "denied": m[4],
                        "revisions": m[5],
                        "avg_execution_ms": m[6],
                        "errors": m[7]
                    } for m in metrics
                ]
            }
            
    except Exception as e:
        return {
            "success": False,
            "error": str(e)
        }
    finally:
        conn.close()

def clone_agent_from_template(
    template_id: str,
    application_id: str,
    name: str,
    customizations: Dict[str, Any] = None
) -> Dict[str, Any]:
    """Create a new agent from a template"""
    
    conn = get_db_connection()
    
    try:
        with conn.cursor() as cur:
            # Get template
            cur.execute("""
                SELECT name, type, default_config, default_schedule 
                FROM agent_templates 
                WHERE id = %s AND active = true
            """, (template_id,))
            
            template = cur.fetchone()
            if not template:
                raise ValueError(f"Template {template_id} not found or inactive")
            
            # Merge template config with customizations
            template_config = json.loads(template[2])
            if customizations:
                template_config.update(customizations)
            
            # Create agent using template
            return create_agent(
                name=name,
                agent_type=template[1],
                application_id=application_id,
                config=template_config,
                schedule_cron=template[3],
                enabled=True
            )
            
    except Exception as e:
        return {
            "success": False,
            "error": str(e)
        }
    finally:
        conn.close()

# Windmill flow entry points
def main(
    action: str,
    agent_id: Optional[str] = None,
    **kwargs
) -> Dict[str, Any]:
    """Main entry point for agent management operations"""
    
    if action == "create":
        return create_agent(**kwargs)
    elif action == "update":
        return update_agent(agent_id, **kwargs)
    elif action == "delete":
        return delete_agent(agent_id)
    elif action == "metrics":
        return get_agent_performance_metrics(agent_id, kwargs.get("days", 30))
    elif action == "clone_from_template":
        return clone_agent_from_template(**kwargs)
    else:
        return {
            "success": False,
            "error": f"Unknown action: {action}"
        }