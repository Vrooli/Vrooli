// SQLite Batch Operations Web UI
const API_BASE = window.location.origin;

// Tab switching
document.querySelectorAll('.tab').forEach(tab => {
    tab.addEventListener('click', () => {
        const targetTab = tab.dataset.tab;
        
        // Update active tab
        document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
        tab.classList.add('active');
        
        // Update active content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.remove('active');
        });
        document.getElementById(targetTab).classList.add('active');
        
        // Load data for the active tab
        if (targetTab === 'export') {
            loadDatabasesForExport();
        } else if (targetTab === 'stats') {
            loadDatabasesForStats();
        }
    });
});

// File upload handlers
document.getElementById('batch-file').addEventListener('change', function(e) {
    const file = e.target.files[0];
    if (file) {
        const info = document.getElementById('batch-file-info');
        info.textContent = `Selected: ${file.name} (${(file.size / 1024).toFixed(2)} KB)`;
        info.style.display = 'block';
        
        // Read file content
        const reader = new FileReader();
        reader.onload = function(event) {
            document.getElementById('batch-sql').value = event.target.result;
        };
        reader.readAsText(file);
    }
});

document.getElementById('import-file').addEventListener('change', function(e) {
    const file = e.target.files[0];
    if (file) {
        const info = document.getElementById('import-file-info');
        info.textContent = `Selected: ${file.name} (${(file.size / 1024).toFixed(2)} KB)`;
        info.style.display = 'block';
    }
});

// Load databases on page load
window.addEventListener('DOMContentLoaded', () => {
    loadDatabases();
});

// Show status message
function showStatus(message, type = 'info') {
    const status = document.getElementById('status');
    status.className = `status ${type}`;
    status.textContent = message;
    status.style.display = 'block';
    
    if (type !== 'error') {
        setTimeout(() => {
            status.style.display = 'none';
        }, 5000);
    }
}

// Load available databases
async function loadDatabases() {
    try {
        const response = await fetch(`${API_BASE}/api/databases`);
        const databases = await response.json();
        
        // Update all database selects
        const selects = ['batch-database', 'import-database', 'export-database', 'stats-database'];
        selects.forEach(id => {
            const select = document.getElementById(id);
            if (select) {
                select.innerHTML = '<option value="">Select a database...</option>';
                databases.forEach(db => {
                    const option = document.createElement('option');
                    option.value = db.name;
                    option.textContent = `${db.name} (${db.size})`;
                    select.appendChild(option);
                });
            }
        });
    } catch (error) {
        console.error('Failed to load databases:', error);
        showStatus('Failed to load databases', 'error');
    }
}

// Load databases for export tab with table loading
async function loadDatabasesForExport() {
    await loadDatabases();
    
    document.getElementById('export-database').addEventListener('change', async function() {
        const dbName = this.value;
        const tableSelect = document.getElementById('export-table');
        
        if (!dbName) {
            tableSelect.innerHTML = '<option value="">Select a database first</option>';
            return;
        }
        
        try {
            const response = await fetch(`${API_BASE}/api/databases/${dbName}/tables`);
            const tables = await response.json();
            
            tableSelect.innerHTML = '<option value="">Select a table...</option>';
            tables.forEach(table => {
                const option = document.createElement('option');
                option.value = table.name;
                option.textContent = `${table.name} (${table.rows} rows)`;
                tableSelect.appendChild(option);
            });
        } catch (error) {
            console.error('Failed to load tables:', error);
            showStatus('Failed to load tables', 'error');
        }
    });
}

// Load databases for stats tab
async function loadDatabasesForStats() {
    await loadDatabases();
}

// Execute batch SQL
async function executeBatch() {
    const database = document.getElementById('batch-database').value;
    const sql = document.getElementById('batch-sql').value;
    
    if (!database) {
        showStatus('Please select a database', 'error');
        return;
    }
    
    if (!sql.trim()) {
        showStatus('Please enter SQL commands', 'error');
        return;
    }
    
    const progressDiv = document.getElementById('batch-progress');
    const progressBar = document.getElementById('batch-progress-bar');
    progressDiv.style.display = 'block';
    progressBar.style.width = '0%';
    progressBar.textContent = '0%';
    
    try {
        // Simulate progress
        let progress = 0;
        const progressInterval = setInterval(() => {
            if (progress < 90) {
                progress += 10;
                progressBar.style.width = `${progress}%`;
                progressBar.textContent = `${progress}%`;
            }
        }, 100);
        
        const response = await fetch(`${API_BASE}/api/batch`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                database: database,
                sql: sql
            })
        });
        
        clearInterval(progressInterval);
        progressBar.style.width = '100%';
        progressBar.textContent = '100%';
        
        const result = await response.json();
        
        if (response.ok) {
            showStatus(`Batch execution completed in ${result.duration}ms`, 'success');
            
            const resultsDiv = document.getElementById('batch-results');
            const output = document.getElementById('batch-output');
            output.textContent = result.output || 'Batch operations completed successfully';
            resultsDiv.style.display = 'block';
        } else {
            throw new Error(result.error || 'Batch execution failed');
        }
    } catch (error) {
        showStatus(error.message, 'error');
    } finally {
        setTimeout(() => {
            progressDiv.style.display = 'none';
        }, 1000);
    }
}

// Import CSV
async function importCSV() {
    const database = document.getElementById('import-database').value;
    const table = document.getElementById('import-table').value;
    const fileInput = document.getElementById('import-file');
    const hasHeader = document.getElementById('import-header').checked;
    
    if (!database) {
        showStatus('Please select a database', 'error');
        return;
    }
    
    if (!table.trim()) {
        showStatus('Please enter a table name', 'error');
        return;
    }
    
    if (!fileInput.files[0]) {
        showStatus('Please select a CSV file', 'error');
        return;
    }
    
    const progressDiv = document.getElementById('import-progress');
    const progressBar = document.getElementById('import-progress-bar');
    progressDiv.style.display = 'block';
    progressBar.style.width = '0%';
    progressBar.textContent = '0%';
    
    try {
        const formData = new FormData();
        formData.append('database', database);
        formData.append('table', table);
        formData.append('file', fileInput.files[0]);
        formData.append('hasHeader', hasHeader);
        
        // Simulate progress
        let progress = 0;
        const progressInterval = setInterval(() => {
            if (progress < 90) {
                progress += 10;
                progressBar.style.width = `${progress}%`;
                progressBar.textContent = `${progress}%`;
            }
        }, 200);
        
        const response = await fetch(`${API_BASE}/api/import-csv`, {
            method: 'POST',
            body: formData
        });
        
        clearInterval(progressInterval);
        progressBar.style.width = '100%';
        progressBar.textContent = '100%';
        
        const result = await response.json();
        
        if (response.ok) {
            showStatus(`Successfully imported ${result.rows} rows`, 'success');
            
            const resultsDiv = document.getElementById('import-results');
            const output = document.getElementById('import-output');
            output.textContent = `Imported ${result.rows} rows into table '${table}'
Database: ${database}
File: ${fileInput.files[0].name}
Duration: ${result.duration}ms`;
            resultsDiv.style.display = 'block';
        } else {
            throw new Error(result.error || 'Import failed');
        }
    } catch (error) {
        showStatus(error.message, 'error');
    } finally {
        setTimeout(() => {
            progressDiv.style.display = 'none';
        }, 1000);
    }
}

// Export CSV
async function exportCSV() {
    const database = document.getElementById('export-database').value;
    const table = document.getElementById('export-table').value;
    const filename = document.getElementById('export-filename').value;
    
    if (!database) {
        showStatus('Please select a database', 'error');
        return;
    }
    
    if (!table) {
        showStatus('Please select a table', 'error');
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE}/api/export-csv`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                database: database,
                table: table,
                filename: filename
            })
        });
        
        const result = await response.json();
        
        if (response.ok) {
            showStatus(`Successfully exported ${result.rows} rows`, 'success');
            
            const resultsDiv = document.getElementById('export-results');
            const output = document.getElementById('export-output');
            output.textContent = `Exported ${result.rows} rows from table '${table}'
Database: ${database}
Output file: ${result.filename}
File size: ${result.size}`;
            resultsDiv.style.display = 'block';
            
            // Show download button
            const downloadBtn = document.getElementById('download-csv');
            downloadBtn.style.display = 'block';
            downloadBtn.onclick = () => {
                window.location.href = `${API_BASE}/api/download/${result.filename}`;
            };
        } else {
            throw new Error(result.error || 'Export failed');
        }
    } catch (error) {
        showStatus(error.message, 'error');
    }
}

// Load database statistics
async function loadStats() {
    const database = document.getElementById('stats-database').value;
    
    if (!database) {
        showStatus('Please select a database', 'error');
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE}/api/stats/${database}`);
        const stats = await response.json();
        
        if (response.ok) {
            // Update stat cards
            document.getElementById('stat-tables').textContent = stats.tables || 0;
            document.getElementById('stat-rows').textContent = stats.totalRows || 0;
            document.getElementById('stat-size').textContent = stats.size || '0 KB';
            document.getElementById('stat-indexes').textContent = stats.indexes || 0;
            document.getElementById('stats-grid').style.display = 'grid';
            
            // Show detailed stats
            if (stats.details) {
                const detailsDiv = document.getElementById('stats-details');
                const output = document.getElementById('stats-output');
                output.textContent = JSON.stringify(stats.details, null, 2);
                detailsDiv.style.display = 'block';
            }
            
            showStatus('Statistics loaded', 'success');
        } else {
            throw new Error(stats.error || 'Failed to load statistics');
        }
    } catch (error) {
        showStatus(error.message, 'error');
    }
}