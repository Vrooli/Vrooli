// Job Pipeline Dashboard Application

const iframeParentOrigin = (() => {
    try {
        return document.referrer ? new URL(document.referrer).origin : undefined;
    } catch (error) {
        return undefined;
    }
})();

if (typeof window !== 'undefined' && window.parent !== window && typeof window.initIframeBridgeChild === 'function') {
    window.initIframeBridgeChild({ appId: 'job-to-scenario-pipeline', parentOrigin: iframeParentOrigin });
}

const API_PORT = window.location.port || '15500';
const API_BASE = `http://localhost:${API_PORT}/api/v1`;

// State
let jobs = [];
let selectedJob = null;
let refreshInterval = null;

// Initialize app
document.addEventListener('DOMContentLoaded', () => {
    initializeEventListeners();
    loadJobs();
    startAutoRefresh();
});

// Event Listeners
function initializeEventListeners() {
    // Import button
    document.getElementById('importBtn').addEventListener('click', () => {
        showModal('importModal');
    });

    // Refresh button
    document.getElementById('refreshBtn').addEventListener('click', () => {
        loadJobs();
    });

    // Stats button
    document.getElementById('statsBtn').addEventListener('click', () => {
        showStats();
    });

    // Import tabs
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            switchTab(e.target.dataset.tab);
        });
    });

    // Manual import
    document.getElementById('importManualBtn').addEventListener('click', importManualJob);

    // Screenshot upload
    const dropZone = document.getElementById('dropZone');
    const fileInput = document.getElementById('fileInput');

    dropZone.addEventListener('click', () => fileInput.click());
    dropZone.addEventListener('dragover', handleDragOver);
    dropZone.addEventListener('dragleave', handleDragLeave);
    dropZone.addEventListener('drop', handleDrop);
    fileInput.addEventListener('change', handleFileSelect);

    document.getElementById('importScreenshotBtn').addEventListener('click', importScreenshotJob);

    // Research batch button
    document.querySelector('.research-batch').addEventListener('click', researchBatch);

    // Modal close buttons
    document.querySelectorAll('.close').forEach(btn => {
        btn.addEventListener('click', (e) => {
            e.target.closest('.modal').classList.remove('show');
        });
    });

    // Click outside modal to close
    window.addEventListener('click', (e) => {
        if (e.target.classList.contains('modal')) {
            e.target.classList.remove('show');
        }
    });
}

// Load and display jobs
async function loadJobs() {
    try {
        const response = await fetch(`${API_BASE}/jobs`);
        const data = await response.json();
        jobs = data.jobs || [];
        renderJobs();
        updateCounts();
    } catch (error) {
        console.error('Failed to load jobs:', error);
        showNotification('Failed to load jobs', 'error');
    }
}

// Render jobs in columns
function renderJobs() {
    const states = ['pending', 'researching', 'evaluated', 'approved', 'building', 'completed'];
    
    states.forEach(state => {
        const container = document.getElementById(`${state}-cards`);
        container.innerHTML = '';
        
        const stateJobs = jobs.filter(job => job.state === state);
        stateJobs.forEach(job => {
            container.appendChild(createJobCard(job));
        });
    });
}

// Create job card element
function createJobCard(job) {
    const card = document.createElement('div');
    card.className = 'job-card';
    card.dataset.jobId = job.id;
    
    // Title
    const title = document.createElement('div');
    title.className = 'job-card-title';
    title.textContent = job.title || 'Untitled Job';
    card.appendChild(title);
    
    // Metadata
    const meta = document.createElement('div');
    meta.className = 'job-card-meta';
    
    // Budget
    if (job.budget && (job.budget.min || job.budget.max)) {
        const budget = document.createElement('div');
        budget.className = 'job-card-budget';
        budget.textContent = `$${job.budget.min || 0} - $${job.budget.max || 0}`;
        meta.appendChild(budget);
    }
    
    // Source
    const source = document.createElement('div');
    source.textContent = `Source: ${job.source}`;
    meta.appendChild(source);
    
    // ID
    const id = document.createElement('div');
    id.textContent = job.id;
    id.style.fontSize = '0.65rem';
    meta.appendChild(id);
    
    card.appendChild(meta);
    
    // Evaluation badge
    if (job.research_report && job.research_report.evaluation) {
        const evaluation = document.createElement('div');
        evaluation.className = `job-card-evaluation evaluation-${job.research_report.evaluation.toLowerCase().replace('_', '-')}`;
        evaluation.textContent = job.research_report.evaluation.replace('_', ' ');
        card.appendChild(evaluation);
    }
    
    // Actions based on state
    const actions = document.createElement('div');
    actions.className = 'job-card-actions';
    
    switch (job.state) {
        case 'pending':
            actions.innerHTML = `
                <button onclick="researchJob('${job.id}')">Research</button>
                <button onclick="viewJob('${job.id}')">View</button>
            `;
            break;
        case 'evaluated':
            if (job.research_report && job.research_report.evaluation === 'RECOMMENDED') {
                actions.innerHTML = `
                    <button onclick="approveJob('${job.id}')">Approve</button>
                    <button onclick="viewJob('${job.id}')">View</button>
                `;
            } else {
                actions.innerHTML = `
                    <button onclick="viewJob('${job.id}')">View</button>
                `;
            }
            break;
        case 'completed':
            actions.innerHTML = `
                <button onclick="generateProposal('${job.id}')">Generate Proposal</button>
                <button onclick="viewJob('${job.id}')">View</button>
            `;
            break;
        default:
            actions.innerHTML = `
                <button onclick="viewJob('${job.id}')">View</button>
            `;
    }
    
    card.appendChild(actions);
    
    // Click to view details
    card.addEventListener('click', (e) => {
        if (!e.target.tagName === 'BUTTON') {
            viewJob(job.id);
        }
    });
    
    return card;
}

// Update column counts
function updateCounts() {
    const states = ['pending', 'researching', 'evaluated', 'approved', 'building', 'completed'];
    
    states.forEach(state => {
        const count = jobs.filter(job => job.state === state).length;
        const column = document.querySelector(`.column[data-state="${state}"]`);
        if (column) {
            column.querySelector('.count').textContent = count;
        }
    });
}

// View job details
async function viewJob(jobId) {
    try {
        const response = await fetch(`${API_BASE}/jobs/${jobId}`);
        const job = await response.json();
        
        const detailsHtml = `
            <h2>${job.title || 'Untitled Job'}</h2>
            <div class="job-details">
                <h3>Details</h3>
                <p><strong>ID:</strong> ${job.id}</p>
                <p><strong>Source:</strong> ${job.source}</p>
                <p><strong>State:</strong> ${job.state}</p>
                <p><strong>Budget:</strong> $${job.budget?.min || 0} - $${job.budget?.max || 0} ${job.budget?.currency || 'USD'}</p>
                <p><strong>Skills:</strong> ${job.skills_required?.join(', ') || 'None specified'}</p>
                
                <h3>Description</h3>
                <pre>${job.description || 'No description'}</pre>
                
                ${job.research_report ? `
                    <h3>Research Report</h3>
                    <p><strong>Evaluation:</strong> ${job.research_report.evaluation}</p>
                    <p><strong>Feasibility Score:</strong> ${job.research_report.feasibility_score || 0}</p>
                    <p><strong>Estimated Hours:</strong> ${job.research_report.estimated_hours || 'N/A'}</p>
                    <p><strong>Existing Scenarios:</strong> ${job.research_report.existing_scenarios?.join(', ') || 'None'}</p>
                    <p><strong>Required Scenarios:</strong> ${job.research_report.required_scenarios?.join(', ') || 'None'}</p>
                    <h4>Technical Analysis</h4>
                    <pre>${job.research_report.technical_analysis || 'No analysis available'}</pre>
                ` : ''}
                
                ${job.proposal ? `
                    <h3>Proposal</h3>
                    <p><strong>Price:</strong> $${job.proposal.price}</p>
                    <h4>Cover Letter</h4>
                    <pre>${job.proposal.cover_letter}</pre>
                    <h4>Technical Approach</h4>
                    <pre>${job.proposal.technical_approach}</pre>
                    <h4>Timeline</h4>
                    <ul>${job.proposal.timeline?.map(t => `<li>${t}</li>`).join('') || ''}</ul>
                    <h4>Deliverables</h4>
                    <ul>${job.proposal.deliverables?.map(d => `<li>${d}</li>`).join('') || ''}</ul>
                ` : ''}
            </div>
        `;
        
        document.getElementById('jobDetails').innerHTML = detailsHtml;
        showModal('detailsModal');
        
    } catch (error) {
        console.error('Failed to load job details:', error);
        showNotification('Failed to load job details', 'error');
    }
}

// Research job
async function researchJob(jobId) {
    try {
        const response = await fetch(`${API_BASE}/jobs/${jobId}/research`, {
            method: 'POST'
        });
        
        if (response.ok) {
            showNotification('Research started', 'success');
            setTimeout(() => loadJobs(), 2000);
        } else {
            showNotification('Failed to start research', 'error');
        }
    } catch (error) {
        console.error('Failed to research job:', error);
        showNotification('Failed to start research', 'error');
    }
}

// Research batch
async function researchBatch() {
    try {
        const pendingJobs = jobs.filter(j => j.state === 'pending').slice(0, 5);
        
        for (const job of pendingJobs) {
            await researchJob(job.id);
        }
        
        showNotification(`Started research for ${pendingJobs.length} jobs`, 'success');
    } catch (error) {
        console.error('Failed to research batch:', error);
        showNotification('Failed to research batch', 'error');
    }
}

// Approve job
async function approveJob(jobId) {
    try {
        const response = await fetch(`${API_BASE}/jobs/${jobId}/approve`, {
            method: 'POST'
        });
        
        if (response.ok) {
            showNotification('Job approved for building', 'success');
            setTimeout(() => loadJobs(), 2000);
        } else {
            showNotification('Failed to approve job', 'error');
        }
    } catch (error) {
        console.error('Failed to approve job:', error);
        showNotification('Failed to approve job', 'error');
    }
}

// Generate proposal
async function generateProposal(jobId) {
    try {
        showNotification('Generating proposal...', 'info');
        
        const response = await fetch(`${API_BASE}/jobs/${jobId}/proposal`, {
            method: 'POST'
        });
        
        if (response.ok) {
            const proposal = await response.json();
            showNotification('Proposal generated', 'success');
            viewJob(jobId);
        } else {
            showNotification('Failed to generate proposal', 'error');
        }
    } catch (error) {
        console.error('Failed to generate proposal:', error);
        showNotification('Failed to generate proposal', 'error');
    }
}

// Import manual job
async function importManualJob() {
    const description = document.getElementById('jobDescription').value;
    const title = document.getElementById('jobTitle').value;
    const budget = document.getElementById('jobBudget').value;
    
    if (!description) {
        showNotification('Please enter a job description', 'warning');
        return;
    }
    
    try {
        const jobData = title ? `${title}\n\n${description}` : description;
        
        const response = await fetch(`${API_BASE}/jobs/import`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                source: 'manual',
                data: jobData
            })
        });
        
        if (response.ok) {
            const result = await response.json();
            showNotification(`Job imported: ${result.job_id}`, 'success');
            closeModal('importModal');
            loadJobs();
            
            // Clear form
            document.getElementById('jobDescription').value = '';
            document.getElementById('jobTitle').value = '';
            document.getElementById('jobBudget').value = '';
        } else {
            showNotification('Failed to import job', 'error');
        }
    } catch (error) {
        console.error('Failed to import job:', error);
        showNotification('Failed to import job', 'error');
    }
}

// Import screenshot job
let screenshotData = null;

async function importScreenshotJob() {
    if (!screenshotData) {
        showNotification('Please select a screenshot', 'warning');
        return;
    }
    
    try {
        const response = await fetch(`${API_BASE}/jobs/import`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                source: 'screenshot',
                data: screenshotData
            })
        });
        
        if (response.ok) {
            const result = await response.json();
            showNotification(`Job imported from screenshot: ${result.job_id}`, 'success');
            closeModal('importModal');
            loadJobs();
            
            // Clear preview
            document.getElementById('previewContainer').innerHTML = '';
            document.getElementById('importScreenshotBtn').disabled = true;
            screenshotData = null;
        } else {
            showNotification('Failed to process screenshot', 'error');
        }
    } catch (error) {
        console.error('Failed to import screenshot:', error);
        showNotification('Failed to process screenshot', 'error');
    }
}

// File handling
function handleFileSelect(e) {
    const file = e.target.files[0];
    if (file) {
        processFile(file);
    }
}

function handleDragOver(e) {
    e.preventDefault();
    e.currentTarget.classList.add('dragover');
}

function handleDragLeave(e) {
    e.currentTarget.classList.remove('dragover');
}

function handleDrop(e) {
    e.preventDefault();
    e.currentTarget.classList.remove('dragover');
    
    const file = e.dataTransfer.files[0];
    if (file && file.type.startsWith('image/')) {
        processFile(file);
    }
}

function processFile(file) {
    const reader = new FileReader();
    
    reader.onload = (e) => {
        const base64 = e.target.result.split(',')[1];
        screenshotData = base64;
        
        // Show preview
        const preview = document.getElementById('previewContainer');
        preview.innerHTML = `<img src="${e.target.result}" alt="Screenshot preview">`;
        
        document.getElementById('importScreenshotBtn').disabled = false;
    };
    
    reader.readAsDataURL(file);
}

// Show statistics
async function showStats() {
    const stats = {
        total: jobs.length,
        pending: jobs.filter(j => j.state === 'pending').length,
        researching: jobs.filter(j => j.state === 'researching').length,
        evaluated: jobs.filter(j => j.state === 'evaluated').length,
        approved: jobs.filter(j => j.state === 'approved').length,
        building: jobs.filter(j => j.state === 'building').length,
        completed: jobs.filter(j => j.state === 'completed').length,
        recommended: jobs.filter(j => j.research_report?.evaluation === 'RECOMMENDED').length,
        notRecommended: jobs.filter(j => j.research_report?.evaluation === 'NOT_RECOMMENDED').length,
        alreadyDone: jobs.filter(j => j.research_report?.evaluation === 'ALREADY_DONE').length,
        noAction: jobs.filter(j => j.research_report?.evaluation === 'NO_ACTION').length,
    };
    
    const avgBudget = jobs.reduce((sum, job) => {
        const max = job.budget?.max || 0;
        return sum + max;
    }, 0) / (jobs.length || 1);
    
    const statsHtml = `
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-label">Total Jobs</div>
                <div class="stat-value">${stats.total}</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Average Budget</div>
                <div class="stat-value">$${Math.round(avgBudget)}</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Recommended</div>
                <div class="stat-value">${stats.recommended}</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Completed</div>
                <div class="stat-value">${stats.completed}</div>
            </div>
        </div>
        
        <h3>Pipeline Status</h3>
        <ul>
            <li>Pending: ${stats.pending}</li>
            <li>Researching: ${stats.researching}</li>
            <li>Evaluated: ${stats.evaluated}</li>
            <li>Approved: ${stats.approved}</li>
            <li>Building: ${stats.building}</li>
            <li>Completed: ${stats.completed}</li>
        </ul>
        
        <h3>Evaluation Results</h3>
        <ul>
            <li>Recommended: ${stats.recommended}</li>
            <li>Not Recommended: ${stats.notRecommended}</li>
            <li>Already Done: ${stats.alreadyDone}</li>
            <li>No Action: ${stats.noAction}</li>
        </ul>
    `;
    
    document.getElementById('statsContent').innerHTML = statsHtml;
    showModal('statsModal');
}

// Modal functions
function showModal(modalId) {
    document.getElementById(modalId).classList.add('show');
}

function closeModal(modalId) {
    document.getElementById(modalId).classList.remove('show');
}

function switchTab(tabName) {
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.remove('active');
    });
    
    document.querySelector(`.tab-btn[data-tab="${tabName}"]`).classList.add('active');
    document.getElementById(`${tabName}-tab`).classList.add('active');
}

// Notifications
function showNotification(message, type = 'info') {
    // Simple console notification for now
    console.log(`[${type.toUpperCase()}] ${message}`);
    
    // Could add a toast notification system here
}

// Auto refresh
function startAutoRefresh() {
    refreshInterval = setInterval(() => {
        loadJobs();
    }, 30000); // Refresh every 30 seconds
}

function stopAutoRefresh() {
    if (refreshInterval) {
        clearInterval(refreshInterval);
        refreshInterval = null;
    }
}

// Make functions globally accessible for onclick handlers
window.viewJob = viewJob;
window.researchJob = researchJob;
window.approveJob = approveJob;
window.generateProposal = generateProposal;
