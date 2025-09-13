#!/bin/bash
# VirusTotal Resource Core Library
# Implements v2.0 contract core functionality

set -euo pipefail

# Resource configuration
RESOURCE_NAME="virustotal"
DEFAULT_PORT=8290
VIRUSTOTAL_PORT="${VIRUSTOTAL_PORT:-$DEFAULT_PORT}"
VIRUSTOTAL_API_KEY="${VIRUSTOTAL_API_KEY:-}"
VIRUSTOTAL_DATA_DIR="${VIRUSTOTAL_DATA_DIR:-${HOME}/.vrooli/resources/virustotal}"
CONTAINER_NAME="vrooli-virustotal"
IMAGE_NAME="vrooli/virustotal:latest"

# Paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "${SCRIPT_DIR}")"
CONFIG_DIR="${RESOURCE_DIR}/config"
TEST_DIR="${RESOURCE_DIR}/test"

# Source test library if needed
[[ -f "${SCRIPT_DIR}/test.sh" ]] && source "${SCRIPT_DIR}/test.sh"

# Show resource info from runtime.json
show_info() {
    local format="${1:-}"
    local runtime_file="${CONFIG_DIR}/runtime.json"
    
    if [[ ! -f "$runtime_file" ]]; then
        echo "Error: runtime.json not found at ${runtime_file}"
        exit 1
    fi
    
    if [[ "$format" == "--json" ]]; then
        cat "$runtime_file"
    else
        echo "VirusTotal Resource Information:"
        echo "================================"
        jq -r '
            "Startup Order: \(.startup_order)
Dependencies: \(.dependencies | join(", "))
Startup Timeout: \(.startup_timeout) seconds
Startup Time Estimate: \(.startup_time_estimate)
Recovery Attempts: \(.recovery_attempts)
Priority: \(.priority)"
        ' "$runtime_file"
    fi
}

# Handle lifecycle management commands
handle_manage() {
    local subcommand="${1:-}"
    shift || true
    
    case "${subcommand}" in
        install)
            install_resource "$@"
            ;;
        uninstall)
            uninstall_resource "$@"
            ;;
        start)
            start_resource "$@"
            ;;
        stop)
            stop_resource "$@"
            ;;
        restart)
            restart_resource "$@"
            ;;
        *)
            echo "Error: Unknown manage subcommand '${subcommand}'"
            echo "Valid subcommands: install, uninstall, start, stop, restart"
            exit 1
            ;;
    esac
}

# Install the resource
install_resource() {
    echo "Installing VirusTotal resource..."
    
    # Create data directory
    mkdir -p "${VIRUSTOTAL_DATA_DIR}"
    
    # Check for API key
    if [[ -z "${VIRUSTOTAL_API_KEY}" ]]; then
        echo "Warning: VIRUSTOTAL_API_KEY not set. You'll need to set it before starting."
    fi
    
    # Build Docker image with vt-cli
    echo "Building Docker image with vt-cli..."
    cat > "${VIRUSTOTAL_DATA_DIR}/Dockerfile" << 'EOF'
FROM python:3.11-slim

# Install vt-cli and dependencies
RUN pip install --no-cache-dir vt-py flask gunicorn aiohttp

# Create app directory
WORKDIR /app

# Copy API wrapper script
COPY api_wrapper.py /app/

# Expose API port
EXPOSE 8290

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD python -c "import requests; requests.get('http://localhost:8290/api/health').raise_for_status()" || exit 1

# Run the API wrapper
CMD ["gunicorn", "--bind", "0.0.0.0:8290", "--workers", "2", "--timeout", "120", "api_wrapper:app"]
EOF
    
    # Create API wrapper
    create_api_wrapper
    
    # Build image
    docker build -t "${IMAGE_NAME}" "${VIRUSTOTAL_DATA_DIR}"
    
    echo "VirusTotal resource installed successfully"
    return 0
}

# Create the API wrapper script
create_api_wrapper() {
    cat > "${VIRUSTOTAL_DATA_DIR}/api_wrapper.py" << 'EOF'
#!/usr/bin/env python3
"""
VirusTotal API Wrapper Service
Provides REST API interface to vt-py library with rate limiting
"""

import os
import json
import time
import hashlib
import sqlite3
import asyncio
from flask import Flask, request, jsonify, send_file
import vt
from threading import Lock
from collections import deque
from datetime import datetime, timedelta
from werkzeug.utils import secure_filename
import tempfile
import base64

app = Flask(__name__)

# Configuration
API_KEY = os.environ.get('VIRUSTOTAL_API_KEY', '')
RATE_LIMIT = 4  # requests per minute for free tier
DAILY_LIMIT = 500  # daily limit for free tier
CACHE_DIR = os.environ.get('VIRUSTOTAL_CACHE_DIR', '/data/cache')
DB_PATH = os.path.join(CACHE_DIR, 'virustotal.db')

# Rate limiting
request_times = deque(maxlen=RATE_LIMIT)
daily_count = 0
daily_reset = datetime.now().replace(hour=0, minute=0, second=0) + timedelta(days=1)
rate_lock = Lock()

# Initialize cache database
def init_db():
    """Initialize SQLite cache database"""
    os.makedirs(CACHE_DIR, exist_ok=True)
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS scan_cache (
            hash TEXT PRIMARY KEY,
            report TEXT,
            timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
            scan_date TEXT,
            positives INTEGER,
            total INTEGER
        )
    ''')
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS url_cache (
            url TEXT PRIMARY KEY,
            report TEXT,
            timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
            scan_date TEXT,
            positives INTEGER,
            total INTEGER
        )
    ''')
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS ip_cache (
            ip TEXT PRIMARY KEY,
            report TEXT,
            timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
            reputation_score INTEGER,
            last_seen TEXT
        )
    ''')
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS domain_cache (
            domain TEXT PRIMARY KEY,
            report TEXT,
            timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
            reputation_score INTEGER,
            last_seen TEXT
        )
    ''')
    cursor.execute('''
        CREATE TABLE IF NOT EXISTS webhooks (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            analysis_id TEXT,
            webhook_url TEXT,
            status TEXT DEFAULT 'pending',
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            completed_at DATETIME
        )
    ''')
    conn.commit()
    conn.close()

# Initialize database on startup
init_db()

def check_rate_limit():
    """Check if we're within rate limits"""
    global daily_count, daily_reset
    
    with rate_lock:
        now = datetime.now()
        
        # Reset daily counter if needed
        if now >= daily_reset:
            daily_count = 0
            daily_reset = now.replace(hour=0, minute=0, second=0) + timedelta(days=1)
        
        # Check daily limit
        if daily_count >= DAILY_LIMIT:
            return False, "Daily limit exceeded"
        
        # Check rate limit (per minute)
        if len(request_times) >= RATE_LIMIT:
            oldest = request_times[0]
            if (now - oldest).seconds < 60:
                wait_time = 60 - (now - oldest).seconds
                return False, f"Rate limit exceeded. Wait {wait_time} seconds"
        
        # Record this request
        request_times.append(now)
        daily_count += 1
        return True, None

def get_cached_report(hash_value):
    """Get cached report from database"""
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    cursor.execute('SELECT report FROM scan_cache WHERE hash = ?', (hash_value,))
    result = cursor.fetchone()
    conn.close()
    if result:
        return json.loads(result[0])
    return None

def cache_report(hash_value, report):
    """Cache report in database"""
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    
    # Extract key metrics for indexing
    positives = report.get('data', {}).get('attributes', {}).get('last_analysis_stats', {}).get('malicious', 0)
    total = sum(report.get('data', {}).get('attributes', {}).get('last_analysis_stats', {}).values())
    scan_date = report.get('data', {}).get('attributes', {}).get('last_analysis_date', '')
    
    cursor.execute('''
        INSERT OR REPLACE INTO scan_cache (hash, report, scan_date, positives, total)
        VALUES (?, ?, ?, ?, ?)
    ''', (hash_value, json.dumps(report), scan_date, positives, total))
    conn.commit()
    conn.close()

async def scan_file_async(file_path):
    """Scan file using VirusTotal API"""
    if not API_KEY:
        return None, "API key not configured"
    
    try:
        async with vt.Client(API_KEY) as client:
            with open(file_path, 'rb') as f:
                # Submit file for analysis
                analysis = await client.scan_file_async(f)
                
                # Wait for analysis to complete (with timeout)
                max_wait = 60  # seconds
                start_time = time.time()
                
                while analysis.status == "queued" and (time.time() - start_time) < max_wait:
                    await asyncio.sleep(2)
                    analysis = await client.get_object_async(f"/analyses/{analysis.id}")
                
                return {
                    'analysis_id': analysis.id,
                    'status': analysis.status,
                    'stats': analysis.stats if hasattr(analysis, 'stats') else {}
                }, None
                
    except Exception as e:
        return None, str(e)

async def get_file_report_async(hash_value):
    """Get file report from VirusTotal API"""
    if not API_KEY:
        return None, "API key not configured"
    
    # Check cache first
    cached = get_cached_report(hash_value)
    if cached:
        cached['from_cache'] = True
        return cached, None
    
    try:
        async with vt.Client(API_KEY) as client:
            file_obj = await client.get_object_async(f"/files/{hash_value}")
            
            report = {
                'data': {
                    'id': hash_value,
                    'type': 'file',
                    'attributes': {
                        'last_analysis_stats': file_obj.last_analysis_stats,
                        'last_analysis_date': file_obj.last_analysis_date.isoformat() if file_obj.last_analysis_date else None,
                        'names': file_obj.names if hasattr(file_obj, 'names') else [],
                        'size': file_obj.size if hasattr(file_obj, 'size') else 0,
                        'type_description': file_obj.type_description if hasattr(file_obj, 'type_description') else '',
                        'sha256': file_obj.sha256 if hasattr(file_obj, 'sha256') else hash_value,
                        'md5': file_obj.md5 if hasattr(file_obj, 'md5') else '',
                        'sha1': file_obj.sha1 if hasattr(file_obj, 'sha1') else ''
                    }
                },
                'from_cache': False
            }
            
            # Cache the report
            cache_report(hash_value, report)
            
            return report, None
            
    except vt.error.APIError as e:
        if e.code == "NotFoundError":
            return {'error': 'File not found', 'hash': hash_value}, None
        return None, str(e)
    except Exception as e:
        return None, str(e)

async def scan_url_async(url):
    """Scan URL using VirusTotal API"""
    if not API_KEY:
        return None, "API key not configured"
    
    try:
        async with vt.Client(API_KEY) as client:
            # Submit URL for analysis
            url_id = vt.url_id(url)
            analysis = await client.scan_url_async(url)
            
            return {
                'analysis_id': analysis.id,
                'url_id': url_id,
                'status': 'queued',
                'message': 'URL submitted for analysis'
            }, None
            
    except Exception as e:
        return None, str(e)

async def get_ip_reputation_async(ip):
    """Get IP reputation from VirusTotal API"""
    if not API_KEY:
        return None, "API key not configured"
    
    # Check cache first
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    cursor.execute('SELECT report FROM ip_cache WHERE ip = ?', (ip,))
    result = cursor.fetchone()
    conn.close()
    
    if result:
        cached = json.loads(result[0])
        cached['from_cache'] = True
        return cached, None
    
    try:
        async with vt.Client(API_KEY) as client:
            ip_obj = await client.get_object_async(f"/ip_addresses/{ip}")
            
            report = {
                'data': {
                    'id': ip,
                    'type': 'ip',
                    'attributes': {
                        'country': ip_obj.country if hasattr(ip_obj, 'country') else '',
                        'as_owner': ip_obj.as_owner if hasattr(ip_obj, 'as_owner') else '',
                        'last_analysis_stats': ip_obj.last_analysis_stats if hasattr(ip_obj, 'last_analysis_stats') else {},
                        'reputation': ip_obj.reputation if hasattr(ip_obj, 'reputation') else 0,
                        'tags': ip_obj.tags if hasattr(ip_obj, 'tags') else []
                    }
                },
                'from_cache': False
            }
            
            # Cache the report
            conn = sqlite3.connect(DB_PATH)
            cursor = conn.cursor()
            cursor.execute('''
                INSERT OR REPLACE INTO ip_cache (ip, report, reputation_score, last_seen)
                VALUES (?, ?, ?, ?)
            ''', (ip, json.dumps(report), report['data']['attributes']['reputation'], datetime.now().isoformat()))
            conn.commit()
            conn.close()
            
            return report, None
            
    except vt.error.APIError as e:
        if e.code == "NotFoundError":
            return {'error': 'IP not found', 'ip': ip}, None
        return None, str(e)
    except Exception as e:
        return None, str(e)

async def get_domain_reputation_async(domain):
    """Get domain reputation from VirusTotal API"""
    if not API_KEY:
        return None, "API key not configured"
    
    # Check cache first
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    cursor.execute('SELECT report FROM domain_cache WHERE domain = ?', (domain,))
    result = cursor.fetchone()
    conn.close()
    
    if result:
        cached = json.loads(result[0])
        cached['from_cache'] = True
        return cached, None
    
    try:
        async with vt.Client(API_KEY) as client:
            domain_obj = await client.get_object_async(f"/domains/{domain}")
            
            report = {
                'data': {
                    'id': domain,
                    'type': 'domain',
                    'attributes': {
                        'last_analysis_stats': domain_obj.last_analysis_stats if hasattr(domain_obj, 'last_analysis_stats') else {},
                        'reputation': domain_obj.reputation if hasattr(domain_obj, 'reputation') else 0,
                        'categories': domain_obj.categories if hasattr(domain_obj, 'categories') else {},
                        'tags': domain_obj.tags if hasattr(domain_obj, 'tags') else [],
                        'whois': domain_obj.whois if hasattr(domain_obj, 'whois') else '',
                        'last_dns_records': domain_obj.last_dns_records if hasattr(domain_obj, 'last_dns_records') else []
                    }
                },
                'from_cache': False
            }
            
            # Cache the report
            conn = sqlite3.connect(DB_PATH)
            cursor = conn.cursor()
            cursor.execute('''
                INSERT OR REPLACE INTO domain_cache (domain, report, reputation_score, last_seen)
                VALUES (?, ?, ?, ?)
            ''', (domain, json.dumps(report), report['data']['attributes']['reputation'], datetime.now().isoformat()))
            conn.commit()
            conn.close()
            
            return report, None
            
    except vt.error.APIError as e:
        if e.code == "NotFoundError":
            return {'error': 'Domain not found', 'domain': domain}, None
        return None, str(e)
    except Exception as e:
        return None, str(e)

def register_webhook(analysis_id, webhook_url):
    """Register a webhook for analysis completion"""
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    cursor.execute('''
        INSERT INTO webhooks (analysis_id, webhook_url, status)
        VALUES (?, ?, 'pending')
    ''', (analysis_id, webhook_url))
    webhook_id = cursor.lastrowid
    conn.commit()
    conn.close()
    return webhook_id

async def process_webhooks():
    """Process pending webhooks in the background"""
    import aiohttp
    
    while True:
        await asyncio.sleep(10)  # Check every 10 seconds
        
        conn = sqlite3.connect(DB_PATH)
        cursor = conn.cursor()
        cursor.execute('''
            SELECT id, analysis_id, webhook_url 
            FROM webhooks 
            WHERE status = 'pending' 
            LIMIT 10
        ''')
        webhooks = cursor.fetchall()
        conn.close()
        
        for webhook_id, analysis_id, webhook_url in webhooks:
            try:
                # Check if analysis is complete
                async with vt.Client(API_KEY) as client:
                    analysis = await client.get_object_async(f"/analyses/{analysis_id}")
                    
                    if analysis.status == "completed":
                        # Send webhook notification
                        async with aiohttp.ClientSession() as session:
                            payload = {
                                'analysis_id': analysis_id,
                                'status': 'completed',
                                'stats': analysis.stats if hasattr(analysis, 'stats') else {},
                                'timestamp': datetime.now().isoformat()
                            }
                            async with session.post(webhook_url, json=payload) as resp:
                                if resp.status == 200:
                                    # Mark webhook as completed
                                    conn = sqlite3.connect(DB_PATH)
                                    cursor = conn.cursor()
                                    cursor.execute('''
                                        UPDATE webhooks 
                                        SET status = 'completed', completed_at = ? 
                                        WHERE id = ?
                                    ''', (datetime.now(), webhook_id))
                                    conn.commit()
                                    conn.close()
            except Exception as e:
                print(f"Webhook error for {webhook_id}: {e}")

# Start webhook processor in background when API starts
import threading
def start_webhook_processor():
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
    loop.run_until_complete(process_webhooks())

webhook_thread = threading.Thread(target=start_webhook_processor, daemon=True)
webhook_thread.start()

@app.route('/api/health', methods=['GET'])
def health():
    """Health check endpoint"""
    status = {
        'status': 'healthy' if API_KEY else 'degraded',
        'api_key_configured': bool(API_KEY),
        'api_key_type': 'premium' if API_KEY and len(API_KEY) > 64 else 'free',
        'daily_remaining': DAILY_LIMIT - daily_count,
        'cache_enabled': os.path.exists(DB_PATH),
        'timestamp': datetime.now().isoformat()
    }
    return jsonify(status), 200 if API_KEY else 503

@app.route('/api/scan/file', methods=['POST'])
def scan_file():
    """Submit file for scanning"""
    if not API_KEY:
        return jsonify({'error': 'API key not configured'}), 503
    
    allowed, msg = check_rate_limit()
    if not allowed:
        return jsonify({'error': msg}), 429
    
    # Check if file was uploaded
    if 'file' not in request.files:
        return jsonify({'error': 'No file provided'}), 400
    
    file = request.files['file']
    if file.filename == '':
        return jsonify({'error': 'No file selected'}), 400
    
    # Get webhook URL if provided
    webhook = request.form.get('webhook')
    
    # Save file temporarily
    with tempfile.NamedTemporaryFile(delete=False) as tmp_file:
        file.save(tmp_file.name)
        
        # Calculate file hash
        hasher = hashlib.sha256()
        with open(tmp_file.name, 'rb') as f:
            for chunk in iter(lambda: f.read(4096), b""):
                hasher.update(chunk)
        file_hash = hasher.hexdigest()
        
        # Check if we already have a report for this hash
        cached = get_cached_report(file_hash)
        if cached:
            os.unlink(tmp_file.name)
            return jsonify({
                'status': 'completed',
                'hash': file_hash,
                'cached': True,
                'report': cached
            }), 200
        
        # Submit for scanning (async operation)
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        result, error = loop.run_until_complete(scan_file_async(tmp_file.name))
        loop.close()
        
        os.unlink(tmp_file.name)
        
        if error:
            return jsonify({'error': error}), 500
        
        response = {
            'status': 'submitted',
            'hash': file_hash,
            'analysis': result
        }
        
        # Register webhook if provided
        if webhook and 'analysis_id' in result:
            webhook_id = register_webhook(result['analysis_id'], webhook)
            response['webhook_id'] = webhook_id
        
        return jsonify(response), 202

@app.route('/api/scan/url', methods=['POST'])
def scan_url():
    """Submit URL for scanning"""
    if not API_KEY:
        return jsonify({'error': 'API key not configured'}), 503
    
    allowed, msg = check_rate_limit()
    if not allowed:
        return jsonify({'error': msg}), 429
    
    data = request.get_json()
    url = data.get('url') if data else None
    webhook = data.get('webhook') if data else None
    
    if not url:
        return jsonify({'error': 'No URL provided'}), 400
    
    # Submit URL for scanning
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
    result, error = loop.run_until_complete(scan_url_async(url))
    loop.close()
    
    if error:
        return jsonify({'error': error}), 500
    
    # Register webhook if provided
    if webhook and 'analysis_id' in result:
        webhook_id = register_webhook(result['analysis_id'], webhook)
        result['webhook_id'] = webhook_id
    
    return jsonify(result), 202

@app.route('/api/report/<hash_value>', methods=['GET'])
def get_report(hash_value):
    """Get file report by hash"""
    if not API_KEY:
        return jsonify({'error': 'API key not configured'}), 503
    
    # Check cache first (doesn't count against rate limit)
    cached = get_cached_report(hash_value)
    if cached:
        cached['from_cache'] = True
        return jsonify(cached), 200
    
    allowed, msg = check_rate_limit()
    if not allowed:
        return jsonify({'error': msg}), 429
    
    # Get report from API
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
    result, error = loop.run_until_complete(get_file_report_async(hash_value))
    loop.close()
    
    if error:
        return jsonify({'error': error}), 500
    
    return jsonify(result), 200

@app.route('/api/stats', methods=['GET'])
def stats():
    """Get usage statistics"""
    # Get cache statistics
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    cursor.execute('SELECT COUNT(*) FROM scan_cache')
    cached_files = cursor.fetchone()[0]
    cursor.execute('SELECT COUNT(*) FROM url_cache')
    cached_urls = cursor.fetchone()[0]
    conn.close()
    
    return jsonify({
        'daily_used': daily_count,
        'daily_limit': DAILY_LIMIT,
        'daily_remaining': DAILY_LIMIT - daily_count,
        'reset_time': daily_reset.isoformat(),
        'cache': {
            'files': cached_files,
            'urls': cached_urls,
            'total': cached_files + cached_urls
        }
    }), 200

@app.route('/api/cache/list', methods=['GET'])
def list_cache():
    """List cached reports"""
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    
    # Get recent cached files
    cursor.execute('''
        SELECT hash, scan_date, positives, total, timestamp 
        FROM scan_cache 
        ORDER BY timestamp DESC 
        LIMIT 100
    ''')
    
    files = []
    for row in cursor.fetchall():
        files.append({
            'hash': row[0],
            'scan_date': row[1],
            'positives': row[2],
            'total': row[3],
            'cached_at': row[4]
        })
    
    conn.close()
    
    return jsonify({'cached_files': files}), 200

@app.route('/api/cache/clear', methods=['DELETE'])
def clear_cache():
    """Clear all cached reports"""
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    cursor.execute('DELETE FROM scan_cache')
    cursor.execute('DELETE FROM url_cache')
    cursor.execute('DELETE FROM ip_cache')
    cursor.execute('DELETE FROM domain_cache')
    deleted = cursor.rowcount
    conn.commit()
    conn.close()
    
    return jsonify({'message': f'Cleared {deleted} cached entries'}), 200

@app.route('/api/reputation/ip/<ip>', methods=['GET'])
def get_ip_reputation(ip):
    """Get IP reputation check"""
    if not API_KEY:
        return jsonify({'error': 'API key not configured'}), 503
    
    # Check cache first (doesn't count against rate limit)
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    cursor.execute('SELECT report FROM ip_cache WHERE ip = ?', (ip,))
    result = cursor.fetchone()
    conn.close()
    
    if result:
        cached = json.loads(result[0])
        cached['from_cache'] = True
        return jsonify(cached), 200
    
    allowed, msg = check_rate_limit()
    if not allowed:
        return jsonify({'error': msg}), 429
    
    # Get report from API
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
    result, error = loop.run_until_complete(get_ip_reputation_async(ip))
    loop.close()
    
    if error:
        return jsonify({'error': error}), 500
    
    return jsonify(result), 200

@app.route('/api/reputation/domain/<domain>', methods=['GET'])
def get_domain_reputation(domain):
    """Get domain reputation check"""
    if not API_KEY:
        return jsonify({'error': 'API key not configured'}), 503
    
    # Check cache first (doesn't count against rate limit)
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    cursor.execute('SELECT report FROM domain_cache WHERE domain = ?', (domain,))
    result = cursor.fetchone()
    conn.close()
    
    if result:
        cached = json.loads(result[0])
        cached['from_cache'] = True
        return jsonify(cached), 200
    
    allowed, msg = check_rate_limit()
    if not allowed:
        return jsonify({'error': msg}), 429
    
    # Get report from API
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
    result, error = loop.run_until_complete(get_domain_reputation_async(domain))
    loop.close()
    
    if error:
        return jsonify({'error': error}), 500
    
    return jsonify(result), 200

@app.route('/api/webhooks', methods=['GET'])
def list_webhooks():
    """List webhook registrations"""
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    cursor.execute('''
        SELECT id, analysis_id, webhook_url, status, created_at, completed_at
        FROM webhooks
        ORDER BY created_at DESC
        LIMIT 100
    ''')
    
    webhooks = []
    for row in cursor.fetchall():
        webhooks.append({
            'id': row[0],
            'analysis_id': row[1],
            'webhook_url': row[2],
            'status': row[3],
            'created_at': row[4],
            'completed_at': row[5]
        })
    
    conn.close()
    
    return jsonify({'webhooks': webhooks}), 200

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8290)
EOF
}

# Uninstall the resource
uninstall_resource() {
    echo "Uninstalling VirusTotal resource..."
    
    # Stop container if running
    if docker ps -q -f name="${CONTAINER_NAME}" | grep -q .; then
        stop_resource
    fi
    
    # Remove container
    docker rm -f "${CONTAINER_NAME}" 2>/dev/null || true
    
    # Remove image
    docker rmi "${IMAGE_NAME}" 2>/dev/null || true
    
    # Optional: keep data
    local keep_data=false
    for arg in "$@"; do
        if [[ "$arg" == "--keep-data" ]]; then
            keep_data=true
            break
        fi
    done
    
    if [[ "$keep_data" == false ]]; then
        rm -rf "${VIRUSTOTAL_DATA_DIR}"
        echo "Data directory removed"
    else
        echo "Data directory preserved at ${VIRUSTOTAL_DATA_DIR}"
    fi
    
    echo "VirusTotal resource uninstalled"
    return 0
}

# Start the resource
start_resource() {
    echo "Starting VirusTotal resource..."
    
    # Check if already running
    if docker ps -q -f name="${CONTAINER_NAME}" | grep -q .; then
        echo "VirusTotal is already running"
        return 0
    fi
    
    # Check API key
    if [[ -z "${VIRUSTOTAL_API_KEY}" ]]; then
        echo "Error: VIRUSTOTAL_API_KEY environment variable not set"
        echo "Please set: export VIRUSTOTAL_API_KEY='your-api-key'"
        exit 1
    fi
    
    # Start container
    docker run -d \
        --name "${CONTAINER_NAME}" \
        --restart unless-stopped \
        -p "${VIRUSTOTAL_PORT}:8290" \
        -e "VIRUSTOTAL_API_KEY=${VIRUSTOTAL_API_KEY}" \
        -v "${VIRUSTOTAL_DATA_DIR}:/data" \
        "${IMAGE_NAME}"
    
    # Wait for health check
    local wait_flag=false
    for arg in "$@"; do
        if [[ "$arg" == "--wait" ]]; then
            wait_flag=true
            break
        fi
    done
    
    if [[ "$wait_flag" == true ]]; then
        echo "Waiting for service to be healthy..."
        local max_attempts=30
        local attempt=0
        
        while [ $attempt -lt $max_attempts ]; do
            if curl -sf "http://localhost:${VIRUSTOTAL_PORT}/api/health" >/dev/null 2>&1; then
                echo "VirusTotal service is healthy"
                return 0
            fi
            sleep 1
            attempt=$((attempt + 1))
        done
        
        echo "Warning: Service started but health check failed"
    fi
    
    echo "VirusTotal resource started on port ${VIRUSTOTAL_PORT}"
    return 0
}

# Stop the resource
stop_resource() {
    echo "Stopping VirusTotal resource..."
    
    if ! docker ps -q -f name="${CONTAINER_NAME}" | grep -q .; then
        echo "VirusTotal is not running"
        return 0
    fi
    
    docker stop "${CONTAINER_NAME}"
    docker rm "${CONTAINER_NAME}"
    
    echo "VirusTotal resource stopped"
    return 0
}

# Restart the resource
restart_resource() {
    echo "Restarting VirusTotal resource..."
    stop_resource
    start_resource "$@"
    return 0
}

# Handle test commands
handle_test() {
    local subcommand="${1:-all}"
    
    case "${subcommand}" in
        smoke)
            test_smoke
            ;;
        integration)
            test_integration
            ;;
        unit)
            test_unit
            ;;
        all)
            test_all
            ;;
        *)
            echo "Error: Unknown test subcommand '${subcommand}'"
            echo "Valid subcommands: smoke, integration, unit, all"
            exit 1
            ;;
    esac
}

# Handle content commands
handle_content() {
    local subcommand="${1:-}"
    shift || true
    
    case "${subcommand}" in
        add)
            content_add "$@"
            ;;
        list)
            content_list "$@"
            ;;
        get)
            content_get "$@"
            ;;
        remove)
            content_remove "$@"
            ;;
        execute)
            content_execute "$@"
            ;;
        *)
            echo "Error: Unknown content subcommand '${subcommand}'"
            echo "Valid subcommands: add, list, get, remove, execute"
            exit 1
            ;;
    esac
}

# Add content (submit for scanning)
content_add() {
    echo "Submitting content for scanning..."
    
    # Parse arguments
    local file=""
    local url=""
    local wait=false
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            --url)
                url="$2"
                shift 2
                ;;
            --wait)
                wait=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -n "$file" ]]; then
        if [[ ! -f "$file" ]]; then
            echo "Error: File not found: $file"
            exit 1
        fi
        
        echo "Scanning file: $file"
        
        # Calculate file hash for reference
        local file_hash=$(sha256sum "$file" | cut -d' ' -f1)
        echo "File SHA256: $file_hash"
        
        # Upload file for scanning
        local response=$(curl -sf -X POST \
            -F "file=@${file}" \
            "http://localhost:${VIRUSTOTAL_PORT}/api/scan/file" 2>/dev/null)
        
        if [[ $? -eq 0 ]]; then
            echo "$response" | jq . 2>/dev/null || echo "$response"
            
            if [[ "$wait" == true ]]; then
                echo "Waiting for scan to complete..."
                sleep 5
                # Retrieve the report
                content_get --hash "$file_hash"
            fi
        else
            echo "Error: Failed to submit file for scanning"
            exit 1
        fi
        
    elif [[ -n "$url" ]]; then
        echo "Scanning URL: $url"
        
        # Submit URL for scanning
        local response=$(curl -sf -X POST \
            -H "Content-Type: application/json" \
            -d "{\"url\": \"$url\"}" \
            "http://localhost:${VIRUSTOTAL_PORT}/api/scan/url" 2>/dev/null)
        
        if [[ $? -eq 0 ]]; then
            echo "$response" | jq . 2>/dev/null || echo "$response"
        else
            echo "Error: Failed to submit URL for scanning"
            exit 1
        fi
        
    else
        echo "Error: Specify --file or --url"
        echo "Usage: resource-virustotal content add --file <path> [--wait]"
        echo "       resource-virustotal content add --url <url>"
        exit 1
    fi
}

# List content (cached results)
content_list() {
    echo "Listing cached scan results..."
    
    local response=$(curl -sf "http://localhost:${VIRUSTOTAL_PORT}/api/cache/list" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq . 2>/dev/null || echo "$response"
    else
        echo "No cached results or service not running"
    fi
}

# Get content (retrieve report)
content_get() {
    local hash=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --hash)
                hash="$2"
                shift 2
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ -z "$hash" ]]; then
        echo "Error: --hash required"
        exit 1
    fi
    
    echo "Retrieving report for hash: $hash"
    curl -sf "http://localhost:${VIRUSTOTAL_PORT}/api/report/${hash}" | jq . || echo "Failed to retrieve report"
}

# Remove content (clear cache)
content_remove() {
    echo "Clearing cached scan results..."
    
    local response=$(curl -sf -X DELETE "http://localhost:${VIRUSTOTAL_PORT}/api/cache/clear" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        echo "$response" | jq -r '.message' 2>/dev/null || echo "Cache cleared"
    else
        echo "Error: Failed to clear cache"
        exit 1
    fi
}

# Execute content (run batch scan)
content_execute() {
    echo "Executing batch scan..."
    # TODO: Implement batch scanning
    echo "Batch scanning will be implemented in the next iteration"
}

# Show service status
show_status() {
    echo "VirusTotal Service Status"
    echo "========================="
    
    # Check container status
    if docker ps -q -f name="${CONTAINER_NAME}" | grep -q .; then
        echo "Service: Running"
        
        # Get health status
        if curl -sf "http://localhost:${VIRUSTOTAL_PORT}/api/health" 2>/dev/null | jq . 2>/dev/null; then
            echo ""
            # Get usage stats
            echo "Usage Statistics:"
            curl -sf "http://localhost:${VIRUSTOTAL_PORT}/api/stats" 2>/dev/null | jq . 2>/dev/null || echo "Stats unavailable"
        else
            echo "Health: Unhealthy or starting"
        fi
    else
        echo "Service: Stopped"
    fi
    
    # Check configuration
    echo ""
    echo "Configuration:"
    echo "  Port: ${VIRUSTOTAL_PORT}"
    echo "  API Key: $(if [[ -n "${VIRUSTOTAL_API_KEY}" ]]; then echo "Configured"; else echo "Not set"; fi)"
    echo "  Data Dir: ${VIRUSTOTAL_DATA_DIR}"
}

# Show logs
show_logs() {
    local follow=false
    local lines=50
    
    for arg in "$@"; do
        case "$arg" in
            -f|--follow)
                follow=true
                ;;
            --lines=*)
                lines="${arg#*=}"
                ;;
        esac
    done
    
    if [[ "$follow" == true ]]; then
        docker logs -f "${CONTAINER_NAME}"
    else
        docker logs --tail "$lines" "${CONTAINER_NAME}"
    fi
}

# Show credentials
show_credentials() {
    echo "VirusTotal API Configuration"
    echo "============================"
    
    if [[ -n "${VIRUSTOTAL_API_KEY}" ]]; then
        echo "API Key: ${VIRUSTOTAL_API_KEY:0:8}...${VIRUSTOTAL_API_KEY: -4}"
        echo "API Tier: $(if [[ ${#VIRUSTOTAL_API_KEY} -eq 64 ]]; then echo "Public (Free)"; else echo "Premium"; fi)"
    else
        echo "API Key: Not configured"
        echo ""
        echo "To configure:"
        echo "  export VIRUSTOTAL_API_KEY='your-api-key'"
        echo ""
        echo "Get a free API key at:"
        echo "  https://www.virustotal.com/gui/join-us"
    fi
    
    echo ""
    echo "API Limits:"
    echo "  Free Tier: 500 requests/day, 4 requests/minute"
    echo "  Premium: Unlimited (contact sales)"
}