#!/usr/bin/env python3
"""
Application Management Flow
Handles CRUD operations for monitored applications
"""

import psycopg2
import json
import uuid
import requests
from datetime import datetime
from typing import Dict, List, Optional, Any
import os
import re

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

def validate_repository_url(url: str) -> Dict[str, Any]:
    """Validate that a repository URL is accessible"""
    try:
        # Basic URL format validation
        if not re.match(r'^https?://', url):
            return {"valid": False, "error": "URL must start with http:// or https://"}
        
        # Try to access the repository (basic check)
        response = requests.head(url, timeout=10, allow_redirects=True)
        
        if response.status_code == 200:
            return {"valid": True}
        elif response.status_code == 404:
            return {"valid": False, "error": "Repository not found (404)"}
        else:
            return {"valid": False, "error": f"Repository returned status code {response.status_code}"}
            
    except requests.exceptions.RequestException as e:
        return {"valid": False, "error": f"Unable to access repository: {str(e)}"}

def create_application(
    name: str,
    repository_url: str,
    documentation_path: str = "/docs",
    notification_settings: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """Create a new application for monitoring"""
    
    # Validate inputs
    if not name or not name.strip():
        return {"success": False, "error": "Application name is required"}
    
    if not repository_url or not repository_url.strip():
        return {"success": False, "error": "Repository URL is required"}
    
    # Validate repository URL
    url_validation = validate_repository_url(repository_url)
    if not url_validation["valid"]:
        return {"success": False, "error": f"Repository validation failed: {url_validation['error']}"}
    
    app_id = str(uuid.uuid4())
    conn = get_db_connection()
    
    # Default notification settings
    if notification_settings is None:
        notification_settings = {
            "email": False,
            "slack": True,
            "severity_threshold": "medium"
        }
    
    try:
        with conn.cursor() as cur:
            # Check if application with same name already exists
            cur.execute("SELECT id FROM applications WHERE LOWER(name) = LOWER(%s) AND active = true", (name,))
            if cur.fetchone():
                raise ValueError(f"Application '{name}' already exists")
            
            # Create application
            cur.execute("""
                INSERT INTO applications (
                    id, name, repository_url, documentation_path, 
                    notification_settings, created_at, updated_at, active
                ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
                RETURNING *
            """, (
                app_id, name.strip(), repository_url.strip(), documentation_path.strip(),
                json.dumps(notification_settings), datetime.now(), datetime.now(), True
            ))
            
            app = cur.fetchone()
            conn.commit()
            
            # Create default app settings
            cur.execute("""
                INSERT INTO app_settings (
                    application_id, auto_apply_enabled, auto_apply_max_severity,
                    notification_channels, scan_frequency_hours, retention_days, custom_rules
                ) VALUES (%s, %s, %s, %s, %s, %s, %s)
            """, (
                app_id, False, "low", json.dumps(["slack"]), 24, 90, json.dumps({})
            ))
            conn.commit()
            
            # Log creation
            cur.execute("""
                INSERT INTO audit_log (entity_type, entity_id, action, new_values, performed_by, created_at)
                VALUES ('application', %s, 'create', %s, %s, %s)
            """, (app_id, json.dumps({"name": name, "repository_url": repository_url}), "system", datetime.now()))
            conn.commit()
            
            return {
                "success": True,
                "application_id": app_id,
                "message": f"Application '{name}' created successfully"
            }
            
    except Exception as e:
        conn.rollback()
        return {
            "success": False,
            "error": str(e)
        }
    finally:
        conn.close()

def update_application(
    app_id: str,
    name: Optional[str] = None,
    repository_url: Optional[str] = None,
    documentation_path: Optional[str] = None,
    notification_settings: Optional[Dict[str, Any]] = None,
    health_score: Optional[float] = None
) -> Dict[str, Any]:
    """Update an existing application"""
    
    conn = get_db_connection()
    
    try:
        with conn.cursor() as cur:
            # Verify application exists
            cur.execute("SELECT name FROM applications WHERE id = %s AND active = true", (app_id,))
            current_app = cur.fetchone()
            if not current_app:
                raise ValueError(f"Application {app_id} not found or inactive")
            
            # Validate repository URL if provided
            if repository_url:
                url_validation = validate_repository_url(repository_url)
                if not url_validation["valid"]:
                    return {"success": False, "error": f"Repository validation failed: {url_validation['error']}"}
            
            # Build update query dynamically
            update_fields = []
            update_values = []
            
            if name is not None and name.strip():
                # Check for name conflicts
                cur.execute("SELECT id FROM applications WHERE LOWER(name) = LOWER(%s) AND id != %s AND active = true", (name.strip(), app_id))
                if cur.fetchone():
                    raise ValueError(f"Application name '{name}' is already in use")
                    
                update_fields.append("name = %s")
                update_values.append(name.strip())
                
            if repository_url is not None:
                update_fields.append("repository_url = %s")
                update_values.append(repository_url.strip())
                
            if documentation_path is not None:
                update_fields.append("documentation_path = %s")
                update_values.append(documentation_path.strip())
                
            if notification_settings is not None:
                update_fields.append("notification_settings = %s")
                update_values.append(json.dumps(notification_settings))
                
            if health_score is not None:
                if not (0.0 <= health_score <= 1.0):
                    raise ValueError("Health score must be between 0.0 and 1.0")
                update_fields.append("health_score = %s")
                update_values.append(health_score)
            
            if not update_fields:
                return {"success": True, "message": "No changes to update"}
            
            # Add updated_at timestamp
            update_fields.append("updated_at = %s")
            update_values.append(datetime.now())
            update_values.append(app_id)
            
            # Execute update
            cur.execute(f"""
                UPDATE applications 
                SET {', '.join(update_fields)}
                WHERE id = %s
                RETURNING *
            """, update_values)
            
            updated_app = cur.fetchone()
            conn.commit()
            
            # Log update
            changes = {}
            if name: changes["name"] = name
            if repository_url: changes["repository_url"] = repository_url
            if health_score is not None: changes["health_score"] = health_score
            
            cur.execute("""
                INSERT INTO audit_log (entity_type, entity_id, action, new_values, performed_by, created_at)
                VALUES ('application', %s, 'update', %s, %s, %s)
            """, (app_id, json.dumps(changes), "system", datetime.now()))
            conn.commit()
            
            return {
                "success": True,
                "message": "Application updated successfully",
                "application": dict(zip([desc[0] for desc in cur.description], updated_app))
            }
            
    except Exception as e:
        conn.rollback()
        return {
            "success": False,
            "error": str(e)
        }
    finally:
        conn.close()

def deactivate_application(app_id: str) -> Dict[str, Any]:
    """Deactivate an application (soft delete)"""
    
    conn = get_db_connection()
    
    try:
        with conn.cursor() as cur:
            # Get application name for logging
            cur.execute("SELECT name FROM applications WHERE id = %s AND active = true", (app_id,))
            app = cur.fetchone()
            if not app:
                raise ValueError(f"Application {app_id} not found or already inactive")
            
            app_name = app[0]
            
            # Deactivate application
            cur.execute("""
                UPDATE applications 
                SET active = false, updated_at = %s 
                WHERE id = %s
            """, (datetime.now(), app_id))
            
            # Disable all agents for this application
            cur.execute("""
                UPDATE agents 
                SET enabled = false, next_run = NULL, updated_at = %s 
                WHERE application_id = %s
            """, (datetime.now(), app_id))
            
            conn.commit()
            
            # Log deactivation
            cur.execute("""
                INSERT INTO audit_log (entity_type, entity_id, action, old_values, performed_by, created_at)
                VALUES ('application', %s, 'deactivate', %s, %s, %s)
            """, (app_id, json.dumps({"name": app_name}), "system", datetime.now()))
            conn.commit()
            
            return {
                "success": True,
                "message": f"Application '{app_name}' deactivated successfully"
            }
            
    except Exception as e:
        conn.rollback()
        return {
            "success": False,
            "error": str(e)
        }
    finally:
        conn.close()

def get_application_summary(app_id: str) -> Dict[str, Any]:
    """Get comprehensive summary of an application"""
    
    conn = get_db_connection()
    
    try:
        with conn.cursor() as cur:
            # Get application details
            cur.execute("""
                SELECT a.*, 
                       COUNT(DISTINCT ag.id) as total_agents,
                       COUNT(DISTINCT CASE WHEN ag.enabled = true THEN ag.id END) as active_agents,
                       COUNT(DISTINCT iq.id) as pending_improvements,
                       MAX(dc.coverage_percentage) as latest_coverage
                FROM applications a
                LEFT JOIN agents ag ON a.id = ag.application_id
                LEFT JOIN improvement_queue iq ON a.id = iq.application_id AND iq.status = 'pending'
                LEFT JOIN documentation_coverage dc ON a.id = dc.application_id
                WHERE a.id = %s AND a.active = true
                GROUP BY a.id
            """, (app_id,))
            
            app_data = cur.fetchone()
            if not app_data:
                raise ValueError(f"Application {app_id} not found or inactive")
            
            # Get recent activity
            cur.execute("""
                SELECT 
                    'improvement' as type,
                    iq.title as description,
                    iq.severity,
                    iq.created_at as timestamp
                FROM improvement_queue iq
                WHERE iq.application_id = %s
                
                UNION ALL
                
                SELECT 
                    'agent_run' as type,
                    'Agent ' || ag.name || ' completed run' as description,
                    CASE WHEN ah.result = 'success' THEN 'low' ELSE 'high' END as severity,
                    ah.created_at as timestamp
                FROM action_history ah
                JOIN agents ag ON ah.agent_id = ag.id
                WHERE ag.application_id = %s AND ah.created_at >= %s
                
                ORDER BY timestamp DESC
                LIMIT 10
            """, (app_id, app_id, datetime.now() - timedelta(days=7)))
            
            recent_activity = cur.fetchall()
            
            # Get app settings
            cur.execute("SELECT * FROM app_settings WHERE application_id = %s", (app_id,))
            settings = cur.fetchone()
            
            return {
                "success": True,
                "application": {
                    "id": app_data[0],
                    "name": app_data[1],
                    "repository_url": app_data[2],
                    "documentation_path": app_data[3],
                    "health_score": float(app_data[4]) if app_data[4] else 0.0,
                    "notification_settings": json.loads(app_data[6]) if app_data[6] else {},
                    "created_at": app_data[7].isoformat(),
                    "updated_at": app_data[8].isoformat(),
                    "statistics": {
                        "total_agents": app_data[10],
                        "active_agents": app_data[11],
                        "pending_improvements": app_data[12],
                        "latest_coverage": float(app_data[13]) if app_data[13] else 0.0
                    }
                },
                "settings": {
                    "auto_apply_enabled": settings[2] if settings else False,
                    "auto_apply_max_severity": settings[3] if settings else "low",
                    "scan_frequency_hours": settings[5] if settings else 24,
                    "retention_days": settings[6] if settings else 90
                } if settings else {},
                "recent_activity": [
                    {
                        "type": activity[0],
                        "description": activity[1],
                        "severity": activity[2],
                        "timestamp": activity[3].isoformat()
                    } for activity in recent_activity
                ]
            }
            
    except Exception as e:
        return {
            "success": False,
            "error": str(e)
        }
    finally:
        conn.close()

def trigger_application_scan(app_id: str, scan_type: str = "full") -> Dict[str, Any]:
    """Trigger an immediate scan of an application"""
    
    conn = get_db_connection()
    
    try:
        with conn.cursor() as cur:
            # Verify application exists and is active
            cur.execute("SELECT name FROM applications WHERE id = %s AND active = true", (app_id,))
            app = cur.fetchone()
            if not app:
                raise ValueError(f"Application {app_id} not found or inactive")
            
            app_name = app[0]
            
            # Get active agents for this application
            cur.execute("""
                SELECT id, name, type 
                FROM agents 
                WHERE application_id = %s AND enabled = true
            """, (app_id,))
            
            agents = cur.fetchall()
            if not agents:
                return {
                    "success": False,
                    "error": "No active agents found for this application"
                }
            
            # Update next_run for all agents to trigger immediate execution
            cur.execute("""
                UPDATE agents 
                SET next_run = %s, updated_at = %s
                WHERE application_id = %s AND enabled = true
            """, (datetime.now(), datetime.now(), app_id))
            
            conn.commit()
            
            # Log the scan trigger
            cur.execute("""
                INSERT INTO audit_log (entity_type, entity_id, action, new_values, performed_by, created_at)
                VALUES ('application', %s, 'scan_triggered', %s, %s, %s)
            """, (app_id, json.dumps({"scan_type": scan_type, "agents_triggered": len(agents)}), "system", datetime.now()))
            conn.commit()
            
            return {
                "success": True,
                "message": f"Scan triggered for '{app_name}' ({len(agents)} agents scheduled)",
                "agents_triggered": [{"id": a[0], "name": a[1], "type": a[2]} for a in agents]
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
    app_id: Optional[str] = None,
    **kwargs
) -> Dict[str, Any]:
    """Main entry point for application management operations"""
    
    if action == "create":
        return create_application(**kwargs)
    elif action == "update":
        return update_application(app_id, **kwargs)
    elif action == "deactivate":
        return deactivate_application(app_id)
    elif action == "summary":
        return get_application_summary(app_id)
    elif action == "trigger_scan":
        return trigger_application_scan(app_id, kwargs.get("scan_type", "full"))
    else:
        return {
            "success": False,
            "error": f"Unknown action: {action}"
        }