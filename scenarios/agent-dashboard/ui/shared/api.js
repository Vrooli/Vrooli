// Shared API Module
// Handles all API interactions across pages

class AgentAPI {
    constructor() {
        this.apiPort = window.API_PORT || '15000';
        this.baseUrl = `http://localhost:${this.apiPort}/api/v1`;
    }

    async fetchAgents() {
        try {
            const response = await fetch(`${this.baseUrl}/agents`);
            if (!response.ok) {
                throw new Error(`API responded with status: ${response.status}`);
            }
            const data = await response.json();
            return data.agents || [];
        } catch (error) {
            console.error('Failed to fetch agents:', error);
            throw error;
        }
    }

    async fetchAgentDetails(agentId) {
        try {
            const response = await fetch(`${this.baseUrl}/agents/${agentId}`);
            if (!response.ok) {
                throw new Error(`API responded with status: ${response.status}`);
            }
            return await response.json();
        } catch (error) {
            console.error(`Failed to fetch agent ${agentId}:`, error);
            throw error;
        }
    }

    async fetchAgentLogs(agentId, lines = 50) {
        try {
            const response = await fetch(`${this.baseUrl}/agents/${agentId}/logs?lines=${lines}`);
            if (!response.ok) {
                throw new Error(`API responded with status: ${response.status}`);
            }
            const data = await response.json();
            return data.data || data;
        } catch (error) {
            console.error(`Failed to fetch logs for agent ${agentId}:`, error);
            throw error;
        }
    }

    async fetchAgentMetrics(agentId) {
        try {
            const response = await fetch(`${this.baseUrl}/agents/${agentId}/metrics`);
            if (!response.ok) {
                throw new Error(`API responded with status: ${response.status}`);
            }
            return await response.json();
        } catch (error) {
            console.error(`Failed to fetch metrics for agent ${agentId}:`, error);
            throw error;
        }
    }

    async stopAgent(agentId) {
        try {
            const response = await fetch(`${this.baseUrl}/agents/${agentId}/stop`, {
                method: 'POST'
            });
            if (!response.ok) {
                throw new Error(`API responded with status: ${response.status}`);
            }
            return await response.json();
        } catch (error) {
            console.error(`Failed to stop agent ${agentId}:`, error);
            throw error;
        }
    }

    async startAgent(agentId) {
        try {
            const response = await fetch(`${this.baseUrl}/agents/${agentId}/start`, {
                method: 'POST'
            });
            if (!response.ok) {
                throw new Error(`API responded with status: ${response.status}`);
            }
            return await response.json();
        } catch (error) {
            console.error(`Failed to start agent ${agentId}:`, error);
            throw error;
        }
    }

    async triggerScan() {
        try {
            const response = await fetch(`${this.baseUrl}/scan`, {
                method: 'POST'
            });
            if (!response.ok) {
                throw new Error(`API responded with status: ${response.status}`);
            }
            return await response.json();
        } catch (error) {
            console.error('Failed to trigger scan:', error);
            throw error;
        }
    }

    async orchestrate(mode = 'auto') {
        try {
            const response = await fetch(`${this.baseUrl}/orchestrate`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify({ mode })
            });
            if (!response.ok) {
                throw new Error(`API responded with status: ${response.status}`);
            }
            return await response.json();
        } catch (error) {
            console.error('Failed to orchestrate:', error);
            throw error;
        }
    }
}

// Create global instance
window.agentAPI = new AgentAPI();

// Agent actions for button handlers
window.agentActions = {
    async stop(agentId) {
        try {
            const result = await window.agentAPI.stopAgent(agentId);
            if (result.success) {
                window.location.reload();
            }
        } catch (error) {
            alert(`Failed to stop agent: ${error.message}`);
        }
    },
    
    async start(agentId) {
        try {
            const result = await window.agentAPI.startAgent(agentId);
            if (result.success) {
                window.location.reload();
            }
        } catch (error) {
            alert(`Failed to start agent: ${error.message}`);
        }
    }
};

// Update agent count in header (used by all pages)
function updateAgentCount(count) {
    const agentCountElement = document.getElementById('agentCount');
    if (agentCountElement) {
        agentCountElement.textContent = `${count} AGENT${count !== 1 ? 'S' : ''} ACTIVE`;
        
        const statusIcon = agentCountElement.parentElement.querySelector('.status-icon');
        if (statusIcon) {
            statusIcon.classList.remove('active', 'warning', 'error');
            statusIcon.classList.add(count > 0 ? 'active' : 'warning');
        }
    }
}

// Make it available globally
window.updateAgentCount = updateAgentCount;

// Auto-update agent count on all pages
document.addEventListener('DOMContentLoaded', async () => {
    try {
        const agents = await window.agentAPI.fetchAgents();
        updateAgentCount(agents.length);
    } catch (error) {
        console.error('Failed to update agent count:', error);
    }
});