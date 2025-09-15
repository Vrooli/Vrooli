/**
 * Agent Dashboard - Cyberpunk Radar Visualization
 * 
 * Displays agents as moving dots on a circular radar with:
 * - Rotating scanner sweep
 * - Color-coded agent health status  
 * - Smooth position animations
 * - Hover tooltips with agent details
 */

class AgentRadar {
    constructor(containerId) {
        this.container = document.getElementById(containerId);
        this.agents = new Map();
        this.radarSize = 200;
        this.centerX = this.radarSize / 2;
        this.centerY = this.radarSize / 2;
        this.maxRadius = this.radarSize / 2 - 10;
        this.animationSpeed = 0.008; // Reduced speed for smoother, slower agent movement
        
        this.init();
    }
    
    init() {
        // Create radar container
        this.container.innerHTML = `
            <div class="radar-display">
                <div class="radar-background">
                    <!-- Radar grid circles -->
                    <div class="radar-circle" style="width: 25%; height: 25%;"></div>
                    <div class="radar-circle" style="width: 50%; height: 50%;"></div>
                    <div class="radar-circle" style="width: 75%; height: 75%;"></div>
                    <div class="radar-circle" style="width: 100%; height: 100%;"></div>
                    
                    <!-- Radar cross lines -->
                    <div class="radar-line radar-line-h"></div>
                    <div class="radar-line radar-line-v"></div>
                </div>
                
                <!-- Rotating scanner sweep -->
                <div class="radar-sweep"></div>
                
                <!-- Agent dots container -->
                <div class="radar-agents" id="radarAgents"></div>
                
                <!-- Radar info overlay -->
                <div class="radar-info">
                    <div class="radar-title">AGENT SCAN</div>
                    <div class="radar-count" id="radarCount">0 CONTACTS</div>
                </div>
            </div>
            
            <!-- Tooltip for agent details -->
            <div class="radar-tooltip" id="radarTooltip" style="display: none;">
                <div class="tooltip-header" id="tooltipHeader"></div>
                <div class="tooltip-body" id="tooltipBody"></div>
            </div>
        `;
        
        this.agentsContainer = document.getElementById('radarAgents');
        this.countDisplay = document.getElementById('radarCount');
        this.tooltip = document.getElementById('radarTooltip');
        this.tooltipHeader = document.getElementById('tooltipHeader');
        this.tooltipBody = document.getElementById('tooltipBody');
        
        // Start animation loop
        this.startAnimationLoop();
        
        console.log('🎯 Agent Radar initialized');
    }
    
    updateAgents(agentsData) {
        // Clear existing agents
        this.agents.clear();
        this.agentsContainer.innerHTML = '';
        
        if (!agentsData || agentsData.length === 0) {
            this.countDisplay.textContent = '0 CONTACTS';
            return;
        }
        
        // Add new agents
        agentsData.forEach((agent, index) => {
            this.addAgent(agent, index);
        });
        
        this.countDisplay.textContent = `${agentsData.length} CONTACT${agentsData.length !== 1 ? 'S' : ''}`;
        
        console.log(`🎯 Radar updated with ${agentsData.length} agents`);
    }
    
    addAgent(agentData, index) {
        // Generate radar position if not provided
        let position = agentData.radar_position;
        if (!position) {
            position = this.generateRandomPosition(index);
        }
        
        // Create agent object
        const agent = {
            id: agentData.id,
            name: agentData.name,
            type: agentData.type,
            status: agentData.status,
            data: agentData,
            currentPos: { x: position.x, y: position.y },
            targetPos: { x: position.target_x || position.x, y: position.target_y || position.y },
            element: null
        };
        
        // Create visual element
        agent.element = this.createAgentElement(agent);
        this.agentsContainer.appendChild(agent.element);
        
        // Store agent
        this.agents.set(agent.id, agent);
        
        // Set new target position for continuous movement
        setTimeout(() => {
            this.setNewTarget(agent);
        }, Math.random() * 3000 + 1000); // Random delay 1-4 seconds
    }
    
    createAgentElement(agent) {
        const element = document.createElement('div');
        element.className = `radar-agent radar-agent-${agent.status}`;
        element.setAttribute('data-agent-id', agent.id);
        
        // Position agent on radar
        const screenPos = this.radarToScreenPos(agent.currentPos.x, agent.currentPos.y);
        element.style.left = `${screenPos.x}px`;
        element.style.top = `${screenPos.y}px`;
        
        // Add hover events
        element.addEventListener('mouseenter', (e) => this.showTooltip(e, agent));
        element.addEventListener('mouseleave', () => this.hideTooltip());
        element.addEventListener('mousemove', (e) => this.updateTooltipPosition(e));
        
        return element;
    }
    
    generateRandomPosition(index) {
        // Generate position in radar coordinates (0-100)
        const angle = (index / 18) * 2 * Math.PI; // Spread around circle
        const radius = 20 + Math.random() * 60; // 20-80% from center
        
        return {
            x: 50 + Math.cos(angle) * radius * 0.4,
            y: 50 + Math.sin(angle) * radius * 0.4,
            target_x: 50 + Math.cos(angle + Math.PI) * radius * 0.3,
            target_y: 50 + Math.sin(angle + Math.PI) * radius * 0.3
        };
    }
    
    setNewTarget(agent) {
        // Generate new target position for smooth movement
        const angle = Math.random() * 2 * Math.PI;
        const radius = 15 + Math.random() * 30; // Keep within reasonable bounds
        
        agent.targetPos = {
            x: Math.max(5, Math.min(95, 50 + Math.cos(angle) * radius)),
            y: Math.max(5, Math.min(95, 50 + Math.sin(angle) * radius))
        };
        
        // Schedule next target change
        setTimeout(() => {
            if (this.agents.has(agent.id)) {
                this.setNewTarget(agent);
            }
        }, 5000 + Math.random() * 10000); // 5-15 seconds
    }
    
    startAnimationLoop() {
        const animate = () => {
            this.updateAgentPositions();
            requestAnimationFrame(animate);
        };
        requestAnimationFrame(animate);
    }
    
    updateAgentPositions() {
        for (const agent of this.agents.values()) {
            // Smoothly move towards target
            const dx = agent.targetPos.x - agent.currentPos.x;
            const dy = agent.targetPos.y - agent.currentPos.y;
            
            agent.currentPos.x += dx * this.animationSpeed;
            agent.currentPos.y += dy * this.animationSpeed;
            
            // Update visual position
            const screenPos = this.radarToScreenPos(agent.currentPos.x, agent.currentPos.y);
            agent.element.style.left = `${screenPos.x}px`;
            agent.element.style.top = `${screenPos.y}px`;
        }
    }
    
    radarToScreenPos(radarX, radarY) {
        // Convert radar coordinates (0-100) to screen pixels
        return {
            x: (radarX / 100) * this.radarSize,
            y: (radarY / 100) * this.radarSize
        };
    }
    
    showTooltip(event, agent) {
        const uptime = this.formatUptime(agent.data.uptime);
        const lastSeen = this.formatTimestamp(agent.data.last_seen);
        
        this.tooltipHeader.innerHTML = `
            <span class="tooltip-agent-name">${agent.name}</span>
            <span class="tooltip-agent-type">${agent.type}</span>
        `;
        
        this.tooltipBody.innerHTML = `
            <div class="tooltip-row">
                <span class="tooltip-label">Status:</span>
                <span class="tooltip-value status-${agent.status}">${agent.status.toUpperCase()}</span>
            </div>
            <div class="tooltip-row">
                <span class="tooltip-label">PID:</span>
                <span class="tooltip-value">${agent.data.pid || 'N/A'}</span>
            </div>
            <div class="tooltip-row">
                <span class="tooltip-label">Uptime:</span>
                <span class="tooltip-value">${uptime}</span>
            </div>
            <div class="tooltip-row">
                <span class="tooltip-label">Last Seen:</span>
                <span class="tooltip-value">${lastSeen}</span>
            </div>
            ${agent.data.capabilities && agent.data.capabilities.length > 0 ? `
            <div class="tooltip-row">
                <span class="tooltip-label">Capabilities:</span>
                <div class="tooltip-capabilities">
                    ${agent.data.capabilities.slice(0, 3).map(cap => `<span class="capability-tag">${cap}</span>`).join('')}
                    ${agent.data.capabilities.length > 3 ? `<span class="capability-more">+${agent.data.capabilities.length - 3}</span>` : ''}
                </div>
            </div>
            ` : ''}
        `;
        
        this.tooltip.style.display = 'block';
        this.updateTooltipPosition(event);
    }
    
    hideTooltip() {
        this.tooltip.style.display = 'none';
    }
    
    updateTooltipPosition(event) {
        const rect = this.container.getBoundingClientRect();
        const x = event.clientX - rect.left + 10;
        const y = event.clientY - rect.top - 10;
        
        this.tooltip.style.left = `${x}px`;
        this.tooltip.style.top = `${y}px`;
    }
    
    formatUptime(uptime) {
        if (!uptime) return 'Unknown';
        return uptime;
    }
    
    formatTimestamp(timestamp) {
        if (!timestamp) return 'Unknown';
        const date = new Date(timestamp);
        return date.toLocaleTimeString();
    }
    
    // Public method to add radar styles
    static addStyles() {
        if (document.getElementById('radar-styles')) return; // Already added
        
        const style = document.createElement('style');
        style.id = 'radar-styles';
        style.textContent = `
            /* Radar container positioning */
            .radar-container {
                position: fixed;
                bottom: 20px;
                right: 20px;
                width: 200px;
                height: 200px;
                z-index: 100;
                pointer-events: auto;
            }
            
            .radar-display {
                position: relative;
                width: 100%;
                height: 100%;
                background: radial-gradient(circle, rgba(0, 255, 255, 0.1) 0%, rgba(0, 0, 0, 0.9) 70%);
                border: 2px solid var(--primary-cyan);
                border-radius: 50%;
                overflow: hidden;
            }
            
            .radar-background {
                position: absolute;
                top: 0;
                left: 0;
                width: 100%;
                height: 100%;
            }
            
            .radar-circle {
                position: absolute;
                top: 50%;
                left: 50%;
                transform: translate(-50%, -50%);
                border: 1px solid rgba(0, 255, 255, 0.2);
                border-radius: 50%;
                pointer-events: none;
            }
            
            .radar-line {
                position: absolute;
                background: rgba(0, 255, 255, 0.2);
                pointer-events: none;
            }
            
            .radar-line-h {
                top: 50%;
                left: 0;
                width: 100%;
                height: 1px;
                transform: translateY(-50%);
            }
            
            .radar-line-v {
                left: 50%;
                top: 0;
                width: 1px;
                height: 100%;
                transform: translateX(-50%);
            }
            
            .radar-sweep {
                position: absolute;
                top: 50%;
                left: 50%;
                width: 50%;
                height: 2px;
                background: linear-gradient(90deg, rgba(0, 255, 255, 0.8), transparent);
                transform-origin: left center;
                animation: radarSweep 4s linear infinite;
                pointer-events: none;
            }
            
            .radar-agents {
                position: absolute;
                top: 0;
                left: 0;
                width: 100%;
                height: 100%;
                pointer-events: none;
            }
            
            .radar-agent {
                position: absolute;
                width: 6px;
                height: 6px;
                border-radius: 50%;
                transform: translate(-50%, -50%);
                animation: agentPulse 2s infinite;
                cursor: pointer;
                pointer-events: auto;
                z-index: 10;
            }
            
            .radar-agent-active {
                background: var(--accent-green);
                box-shadow: 0 0 8px var(--accent-green);
            }
            
            .radar-agent-inactive {
                background: var(--text-secondary);
                box-shadow: 0 0 4px var(--text-secondary);
            }
            
            .radar-agent-error {
                background: var(--accent-red);
                box-shadow: 0 0 8px var(--accent-red);
            }
            
            .radar-info {
                position: absolute;
                bottom: 5px;
                left: 50%;
                transform: translateX(-50%);
                text-align: center;
                font-family: 'Share Tech Mono', monospace;
                font-size: 10px;
                pointer-events: none;
            }
            
            .radar-title {
                color: var(--primary-cyan);
                font-weight: 600;
                margin-bottom: 2px;
            }
            
            .radar-count {
                color: var(--accent-green);
                font-size: 9px;
            }
            
            /* Tooltip styles */
            .radar-tooltip {
                position: absolute;
                background: rgba(0, 0, 0, 0.95);
                border: 1px solid var(--primary-cyan);
                border-radius: 4px;
                padding: 8px;
                font-family: 'Share Tech Mono', monospace;
                font-size: 11px;
                color: var(--text-primary);
                z-index: 1000;
                max-width: 200px;
                box-shadow: 0 4px 12px rgba(0, 255, 255, 0.3);
                pointer-events: none;
            }
            
            .tooltip-header {
                display: flex;
                justify-content: space-between;
                align-items: center;
                margin-bottom: 6px;
                padding-bottom: 4px;
                border-bottom: 1px solid rgba(0, 255, 255, 0.3);
            }
            
            .tooltip-agent-name {
                color: var(--primary-cyan);
                font-weight: 600;
            }
            
            .tooltip-agent-type {
                color: var(--primary-yellow);
                font-size: 9px;
                text-transform: uppercase;
            }
            
            .tooltip-row {
                display: flex;
                justify-content: space-between;
                margin-bottom: 3px;
            }
            
            .tooltip-label {
                color: var(--text-secondary);
            }
            
            .tooltip-value {
                color: var(--text-primary);
            }
            
            .tooltip-value.status-active {
                color: var(--accent-green);
            }
            
            .tooltip-value.status-error {
                color: var(--accent-red);
            }
            
            .tooltip-value.status-inactive {
                color: var(--text-secondary);
            }
            
            .tooltip-capabilities {
                display: flex;
                flex-wrap: wrap;
                gap: 2px;
                margin-top: 2px;
            }
            
            .capability-tag {
                background: rgba(0, 255, 255, 0.2);
                color: var(--primary-cyan);
                padding: 1px 4px;
                border-radius: 2px;
                font-size: 8px;
            }
            
            .capability-more {
                color: var(--text-secondary);
                font-size: 8px;
            }
            
            /* Mobile responsive */
            @media (max-width: 768px) {
                .radar-container {
                    width: 150px;
                    height: 150px;
                    bottom: 10px;
                    right: 10px;
                }
                
                .radar-info {
                    font-size: 8px;
                }
                
                .radar-count {
                    font-size: 7px;
                }
            }
        `;
        
        document.head.appendChild(style);
    }
}

// Auto-initialize styles when script loads
AgentRadar.addStyles();

// Export for global use
window.AgentRadar = AgentRadar;