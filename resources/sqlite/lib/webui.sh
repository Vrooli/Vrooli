#!/bin/bash

# SQLite Web UI Server
# Provides a web interface for batch operations, CSV import/export, and database statistics

# Source core functionality
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Only source if not already in Vrooli context
if [[ -z "${VROOLI_COMMON_SOURCED:-}" ]]; then
    if [[ -f "${SCRIPT_DIR}/../../../scripts/common/source_all.sh" ]]; then
        source "${SCRIPT_DIR}/../../../scripts/common/source_all.sh"
    fi
fi

# Source core library if available
if [[ -f "${SCRIPT_DIR}/core.sh" ]]; then
    source "${SCRIPT_DIR}/core.sh"
fi

# Configuration  
readonly SQLITE_UI_PORT="${SQLITE_UI_PORT:-8297}"
readonly SQLITE_UI_HOST="${SQLITE_UI_HOST:-127.0.0.1}"
readonly SQLITE_UI_DIR="${SCRIPT_DIR}/../ui"
readonly SQLITE_UI_PID_FILE="${VROOLI_TMP:-/tmp}/sqlite-ui.pid"
readonly SQLITE_UI_LOG="${VROOLI_LOGS:-/tmp}/sqlite-ui.log"

# Start web UI server
sqlite::webui::start() {
    log::info "Starting SQLite Web UI on port ${SQLITE_UI_PORT}..."
    
    # Check if already running
    if [[ -f "$SQLITE_UI_PID_FILE" ]]; then
        local pid
        pid=$(cat "$SQLITE_UI_PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            log::warning "SQLite Web UI already running (PID: $pid)"
            return 1
        fi
    fi
    
    # Ensure UI files exist
    if [[ ! -f "${SQLITE_UI_DIR}/index.html" ]]; then
        log::error "Web UI files not found in ${SQLITE_UI_DIR}"
        return 1
    fi
    
    # Start Python HTTP server with custom handler
    cat > "${VROOLI_TMP:-/tmp}/sqlite_webui_server.py" << 'EOF'
#!/usr/bin/env python3
import json
import os
import subprocess
import sys
from http.server import HTTPServer, SimpleHTTPRequestHandler
from urllib.parse import urlparse, parse_qs
import mimetypes
import tempfile
import shutil

# Configuration from environment
PORT = int(os.environ.get('SQLITE_UI_PORT', '8197'))
HOST = os.environ.get('SQLITE_UI_HOST', '127.0.0.1')
UI_DIR = os.environ.get('SQLITE_UI_DIR', '')
SQLITE_CLI = os.environ.get('SQLITE_CLI_PATH', 'resource-sqlite')
DATABASE_PATH = os.environ.get('SQLITE_DATABASE_PATH', '')

class SQLiteAPIHandler(SimpleHTTPRequestHandler):
    def __init__(self, *args, **kwargs):
        super().__init__(*args, directory=UI_DIR, **kwargs)
    
    def do_GET(self):
        parsed_path = urlparse(self.path)
        
        # API endpoints
        if parsed_path.path == '/api/databases':
            self.handle_get_databases()
        elif parsed_path.path.startswith('/api/databases/') and parsed_path.path.endswith('/tables'):
            db_name = parsed_path.path.split('/')[3]
            self.handle_get_tables(db_name)
        elif parsed_path.path.startswith('/api/stats/'):
            db_name = parsed_path.path.split('/')[3]
            self.handle_get_stats(db_name)
        elif parsed_path.path.startswith('/api/download/'):
            filename = parsed_path.path.split('/')[3]
            self.handle_download(filename)
        else:
            # Serve static files
            super().do_GET()
    
    def do_POST(self):
        parsed_path = urlparse(self.path)
        
        if parsed_path.path == '/api/batch':
            self.handle_batch_sql()
        elif parsed_path.path == '/api/import-csv':
            self.handle_import_csv()
        elif parsed_path.path == '/api/export-csv':
            self.handle_export_csv()
        else:
            self.send_error(404)
    
    def handle_get_databases(self):
        """Get list of available databases"""
        try:
            result = subprocess.run(
                [SQLITE_CLI, 'content', 'list'],
                capture_output=True,
                text=True,
                timeout=10
            )
            
            databases = []
            for line in result.stdout.splitlines()[2:]:  # Skip header lines
                if line.strip() and not line.startswith('No databases'):
                    parts = line.split('(')
                    if len(parts) >= 2:
                        name = parts[0].strip()
                        size = parts[1].split(')')[0] if ')' in parts[1] else 'unknown'
                        databases.append({'name': name, 'size': size})
            
            self.send_json_response(databases)
        except Exception as e:
            self.send_error_response(str(e))
    
    def handle_get_tables(self, db_name):
        """Get list of tables in a database"""
        try:
            # Get table list using SQLite
            query = "SELECT name FROM sqlite_master WHERE type='table';"
            result = subprocess.run(
                [SQLITE_CLI, 'content', 'execute', db_name, query],
                capture_output=True,
                text=True,
                timeout=10
            )
            
            tables = []
            for line in result.stdout.splitlines():
                if line.strip() and not line.startswith('['):
                    # Get row count for each table
                    count_query = f"SELECT COUNT(*) FROM {line.strip()};"
                    count_result = subprocess.run(
                        [SQLITE_CLI, 'content', 'execute', db_name, count_query],
                        capture_output=True,
                        text=True,
                        timeout=5
                    )
                    rows = count_result.stdout.strip() if count_result.returncode == 0 else '0'
                    tables.append({'name': line.strip(), 'rows': rows})
            
            self.send_json_response(tables)
        except Exception as e:
            self.send_error_response(str(e))
    
    def handle_get_stats(self, db_name):
        """Get database statistics"""
        try:
            # Get stats using the SQLite resource stats command
            result = subprocess.run(
                [SQLITE_CLI, 'stats', 'analyze', db_name],
                capture_output=True,
                text=True,
                timeout=30
            )
            
            # Parse the output
            stats = {
                'tables': 0,
                'totalRows': 0,
                'size': 'unknown',
                'indexes': 0,
                'details': result.stdout
            }
            
            # Extract key metrics from output
            for line in result.stdout.splitlines():
                if 'Tables:' in line:
                    stats['tables'] = int(line.split(':')[1].strip())
                elif 'Total Rows:' in line:
                    stats['totalRows'] = int(line.split(':')[1].strip())
                elif 'Database Size:' in line:
                    stats['size'] = line.split(':')[1].strip()
                elif 'Indexes:' in line:
                    stats['indexes'] = int(line.split(':')[1].strip())
            
            self.send_json_response(stats)
        except Exception as e:
            self.send_error_response(str(e))
    
    def handle_batch_sql(self):
        """Execute batch SQL operations"""
        try:
            content_length = int(self.headers['Content-Length'])
            post_data = json.loads(self.rfile.read(content_length))
            
            database = post_data.get('database')
            sql = post_data.get('sql')
            
            if not database or not sql:
                self.send_error_response('Missing database or SQL')
                return
            
            # Write SQL to temp file
            with tempfile.NamedTemporaryFile(mode='w', suffix='.sql', delete=False) as f:
                f.write(sql)
                sql_file = f.name
            
            try:
                # Execute batch operations
                start_time = os.times()
                result = subprocess.run(
                    [SQLITE_CLI, 'content', 'batch', database, sql_file],
                    capture_output=True,
                    text=True,
                    timeout=120
                )
                end_time = os.times()
                duration = int((end_time.elapsed - start_time.elapsed) * 1000)
                
                if result.returncode == 0:
                    self.send_json_response({
                        'success': True,
                        'output': result.stdout,
                        'duration': duration
                    })
                else:
                    self.send_error_response(result.stderr or 'Batch execution failed')
            finally:
                os.unlink(sql_file)
        except Exception as e:
            self.send_error_response(str(e))
    
    def handle_import_csv(self):
        """Import CSV file to database"""
        try:
            # Parse multipart form data
            content_type = self.headers.get('Content-Type')
            if not content_type or 'multipart/form-data' not in content_type:
                self.send_error_response('Invalid content type')
                return
            
            # This is a simplified implementation
            # In production, use a proper multipart parser
            self.send_error_response('CSV import via web UI not yet fully implemented')
        except Exception as e:
            self.send_error_response(str(e))
    
    def handle_export_csv(self):
        """Export table to CSV"""
        try:
            content_length = int(self.headers['Content-Length'])
            post_data = json.loads(self.rfile.read(content_length))
            
            database = post_data.get('database')
            table = post_data.get('table')
            filename = post_data.get('filename', '')
            
            if not database or not table:
                self.send_error_response('Missing database or table')
                return
            
            # Execute export
            result = subprocess.run(
                [SQLITE_CLI, 'content', 'export_csv', database, table, filename],
                capture_output=True,
                text=True,
                timeout=60
            )
            
            if result.returncode == 0:
                # Parse output to get filename and stats
                output_file = None
                row_count = 0
                for line in result.stdout.splitlines():
                    if 'Exported' in line and 'rows to:' in line:
                        parts = line.split('rows to:')
                        row_count = int(parts[0].split('Exported')[1].strip())
                        output_file = parts[1].strip()
                
                if output_file:
                    file_size = os.path.getsize(output_file) if os.path.exists(output_file) else 0
                    self.send_json_response({
                        'success': True,
                        'rows': row_count,
                        'filename': os.path.basename(output_file),
                        'size': f"{file_size / 1024:.2f} KB"
                    })
                else:
                    self.send_error_response('Export succeeded but could not parse output')
            else:
                self.send_error_response(result.stderr or 'Export failed')
        except Exception as e:
            self.send_error_response(str(e))
    
    def handle_download(self, filename):
        """Download exported CSV file"""
        try:
            file_path = os.path.join(DATABASE_PATH, filename)
            if not os.path.exists(file_path):
                self.send_error(404)
                return
            
            # Send file
            self.send_response(200)
            self.send_header('Content-Type', 'text/csv')
            self.send_header('Content-Disposition', f'attachment; filename="{filename}"')
            self.end_headers()
            
            with open(file_path, 'rb') as f:
                shutil.copyfileobj(f, self.wfile)
        except Exception as e:
            self.send_error(500)
    
    def send_json_response(self, data):
        """Send JSON response"""
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Access-Control-Allow-Origin', '*')
        self.end_headers()
        self.wfile.write(json.dumps(data).encode())
    
    def send_error_response(self, message):
        """Send error response"""
        self.send_response(400)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Access-Control-Allow-Origin', '*')
        self.end_headers()
        self.wfile.write(json.dumps({'error': message}).encode())
    
    def log_message(self, format, *args):
        """Override to suppress default logging"""
        pass

if __name__ == '__main__':
    print(f"Starting SQLite Web UI server on {HOST}:{PORT}")
    print(f"Serving files from: {UI_DIR}")
    print(f"Access the UI at: http://{HOST}:{PORT}/")
    
    httpd = HTTPServer((HOST, PORT), SQLiteAPIHandler)
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        print("\nShutting down server...")
        httpd.shutdown()
EOF
    
    # Make server script executable
    chmod +x "${VROOLI_TMP:-/tmp}/sqlite_webui_server.py"
    
    # Set environment variables for the server
    export SQLITE_UI_PORT
    export SQLITE_UI_HOST
    export SQLITE_UI_DIR
    export SQLITE_CLI_PATH="${SCRIPT_DIR}/../cli.sh"
    export SQLITE_DATABASE_PATH
    
    # Start the server in background
    nohup python3 "${VROOLI_TMP:-/tmp}/sqlite_webui_server.py" >> "$SQLITE_UI_LOG" 2>&1 &
    local pid=$!
    
    # Save PID
    echo "$pid" > "$SQLITE_UI_PID_FILE"
    
    # Wait for server to start
    local retries=10
    while [[ $retries -gt 0 ]]; do
        if timeout 1 curl -sf "http://${SQLITE_UI_HOST}:${SQLITE_UI_PORT}/" > /dev/null 2>&1; then
            log::success "SQLite Web UI started successfully"
            echo "Access the UI at: http://${SQLITE_UI_HOST}:${SQLITE_UI_PORT}/"
            return 0
        fi
        sleep 0.5
        retries=$((retries - 1))
    done
    
    log::error "Failed to start SQLite Web UI"
    kill "$pid" 2>/dev/null
    rm -f "$SQLITE_UI_PID_FILE"
    return 1
}

# Stop web UI server
sqlite::webui::stop() {
    log::info "Stopping SQLite Web UI..."
    
    if [[ ! -f "$SQLITE_UI_PID_FILE" ]]; then
        log::warning "SQLite Web UI is not running"
        return 1
    fi
    
    local pid
    pid=$(cat "$SQLITE_UI_PID_FILE")
    
    if kill -0 "$pid" 2>/dev/null; then
        kill "$pid"
        
        # Wait for process to terminate
        local retries=10
        while [[ $retries -gt 0 ]] && kill -0 "$pid" 2>/dev/null; do
            sleep 0.5
            retries=$((retries - 1))
        done
        
        if kill -0 "$pid" 2>/dev/null; then
            log::warning "Process didn't terminate gracefully, forcing..."
            kill -9 "$pid"
        fi
        
        log::success "SQLite Web UI stopped"
    else
        log::warning "Process not found (PID: $pid)"
    fi
    
    rm -f "$SQLITE_UI_PID_FILE"
    rm -f "${VROOLI_TMP:-/tmp}/sqlite_webui_server.py"
    return 0
}

# Get web UI status
sqlite::webui::status() {
    if [[ ! -f "$SQLITE_UI_PID_FILE" ]]; then
        echo "Status: Stopped"
        return 1
    fi
    
    local pid
    pid=$(cat "$SQLITE_UI_PID_FILE")
    
    if kill -0 "$pid" 2>/dev/null; then
        echo "Status: Running"
        echo "PID: $pid"
        echo "URL: http://${SQLITE_UI_HOST}:${SQLITE_UI_PORT}/"
        
        # Check if responsive
        if timeout 1 curl -sf "http://${SQLITE_UI_HOST}:${SQLITE_UI_PORT}/" > /dev/null 2>&1; then
            echo "Health: Responsive"
        else
            echo "Health: Not responding"
        fi
        return 0
    else
        echo "Status: Stopped (stale PID file)"
        rm -f "$SQLITE_UI_PID_FILE"
        return 1
    fi
}

# Restart web UI
sqlite::webui::restart() {
    sqlite::webui::stop
    sleep 1
    sqlite::webui::start
}

# Only execute main handler if run directly, not sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Main handler
    case "${1:-}" in
        start)
            sqlite::webui::start
            ;;
        stop)
            sqlite::webui::stop
            ;;
        restart)
            sqlite::webui::restart
            ;;
        status)
            sqlite::webui::status
            ;;
        *)
            echo "Usage: $0 {start|stop|restart|status}"
            exit 1
            ;;
    esac
fi