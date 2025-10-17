import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

const BRIDGE_FLAG = '__mindMapsNeuralBridgeInitialized';

function bootstrapIframeBridge() {
    if (typeof window === 'undefined' || window[BRIDGE_FLAG]) {
        return;
    }

    if (window.parent !== window) {
        const options = { appId: 'mind-maps-neural' };
        try {
            if (document.referrer) {
                options.parentOrigin = new URL(document.referrer).origin;
            }
        } catch (error) {
            console.warn('[MindMapsNeural] Unable to determine parent origin for iframe bridge', error);
        }

        initIframeBridgeChild(options);
        window[BRIDGE_FLAG] = true;
    }
}

bootstrapIframeBridge();

// Neural Mind Maps - Interactive Neural Network Visualization
class NeuralMindMap {
    constructor() {
        this.canvas = document.getElementById('mind-map-canvas');
        this.ctx = this.canvas.getContext('2d');
        this.nodes = new Map();
        this.connections = new Map();
        this.selectedNode = null;
        this.mode = 'view';
        this.zoom = 1;
        this.offset = { x: 0, y: 0 };
        this.isDragging = false;
        this.dragStart = null;
        this.neuralActivity = [];
        this.apiUrl = window.location.hostname === 'localhost' 
            ? `http://localhost:${window.API_PORT || 8100}/api` 
            : '/api';
        
        this.init();
    }

    init() {
        this.setupCanvas();
        this.setupEventListeners();
        this.setupNeuralBackground();
        this.loadNetworks();
        this.startNeuralAnimation();
    }

    setupCanvas() {
        const resizeCanvas = () => {
            const container = this.canvas.parentElement;
            this.canvas.width = container.clientWidth;
            this.canvas.height = container.clientHeight;
            this.render();
        };
        
        resizeCanvas();
        window.addEventListener('resize', resizeCanvas);
    }

    setupNeuralBackground() {
        const bgCanvas = document.getElementById('neuralBackground');
        if (!bgCanvas) return;
        
        const bgCtx = bgCanvas.getContext('2d');
        bgCanvas.width = window.innerWidth;
        bgCanvas.height = window.innerHeight;
        
        const particles = [];
        const particleCount = 50;
        
        for (let i = 0; i < particleCount; i++) {
            particles.push({
                x: Math.random() * bgCanvas.width,
                y: Math.random() * bgCanvas.height,
                vx: (Math.random() - 0.5) * 0.5,
                vy: (Math.random() - 0.5) * 0.5,
                radius: Math.random() * 2 + 1,
                opacity: Math.random() * 0.5 + 0.2
            });
        }
        
        const animateBackground = () => {
            bgCtx.fillStyle = 'rgba(10, 14, 27, 0.05)';
            bgCtx.fillRect(0, 0, bgCanvas.width, bgCanvas.height);
            
            particles.forEach((particle, i) => {
                particle.x += particle.vx;
                particle.y += particle.vy;
                
                if (particle.x < 0 || particle.x > bgCanvas.width) particle.vx *= -1;
                if (particle.y < 0 || particle.y > bgCanvas.height) particle.vy *= -1;
                
                bgCtx.beginPath();
                bgCtx.arc(particle.x, particle.y, particle.radius, 0, Math.PI * 2);
                bgCtx.fillStyle = `rgba(0, 255, 204, ${particle.opacity})`;
                bgCtx.fill();
                
                // Draw connections between nearby particles
                particles.slice(i + 1).forEach(other => {
                    const distance = Math.hypot(other.x - particle.x, other.y - particle.y);
                    if (distance < 100) {
                        bgCtx.beginPath();
                        bgCtx.moveTo(particle.x, particle.y);
                        bgCtx.lineTo(other.x, other.y);
                        bgCtx.strokeStyle = `rgba(0, 255, 204, ${0.1 * (1 - distance / 100)})`;
                        bgCtx.stroke();
                    }
                });
            });
            
            requestAnimationFrame(animateBackground);
        };
        
        animateBackground();
    }

    setupEventListeners() {
        // Canvas interactions
        this.canvas.addEventListener('mousedown', this.handleMouseDown.bind(this));
        this.canvas.addEventListener('mousemove', this.handleMouseMove.bind(this));
        this.canvas.addEventListener('mouseup', this.handleMouseUp.bind(this));
        this.canvas.addEventListener('wheel', this.handleWheel.bind(this));
        this.canvas.addEventListener('dblclick', this.handleDoubleClick.bind(this));
        
        // Toolbar buttons
        document.getElementById('addNodeBtn')?.addEventListener('click', () => {
            this.mode = this.mode === 'add' ? 'view' : 'add';
            this.updateToolbarState();
        });
        
        document.getElementById('connectBtn')?.addEventListener('click', () => {
            this.mode = this.mode === 'connect' ? 'view' : 'connect';
            this.updateToolbarState();
        });
        
        document.getElementById('autoOrganizeBtn')?.addEventListener('click', () => {
            this.autoOrganize();
        });
        
        document.getElementById('visualizeBtn')?.addEventListener('click', () => {
            this.visualizeConnections();
        });
        
        document.getElementById('zoomInBtn')?.addEventListener('click', () => {
            this.zoom = Math.min(this.zoom * 1.2, 5);
            this.render();
        });
        
        document.getElementById('zoomOutBtn')?.addEventListener('click', () => {
            this.zoom = Math.max(this.zoom / 1.2, 0.2);
            this.render();
        });
        
        document.getElementById('resetViewBtn')?.addEventListener('click', () => {
            this.zoom = 1;
            this.offset = { x: 0, y: 0 };
            this.render();
        });
        
        // Search functionality
        document.getElementById('searchInput')?.addEventListener('input', (e) => {
            this.handleSearch(e.target.value);
        });
        
        // Search mode toggles
        document.querySelectorAll('.mode-toggle').forEach(btn => {
            btn.addEventListener('click', (e) => {
                document.querySelectorAll('.mode-toggle').forEach(b => b.classList.remove('active'));
                e.target.classList.add('active');
                this.searchMode = e.target.dataset.mode;
            });
        });
        
        // Pattern cards
        document.querySelectorAll('.pattern-card').forEach(card => {
            card.addEventListener('click', (e) => {
                const pattern = e.currentTarget.dataset.pattern;
                this.applyPattern(pattern);
            });
        });
        
        // Network creation
        document.querySelector('.create-network-btn')?.addEventListener('click', () => {
            this.createNewNetwork();
        });
    }

    handleMouseDown(e) {
        const rect = this.canvas.getBoundingClientRect();
        const x = (e.clientX - rect.left - this.offset.x) / this.zoom;
        const y = (e.clientY - rect.top - this.offset.y) / this.zoom;
        
        const clickedNode = this.getNodeAt(x, y);
        
        if (this.mode === 'add' && !clickedNode) {
            this.addNode(x, y);
        } else if (this.mode === 'connect' && clickedNode) {
            if (this.selectedNode && this.selectedNode !== clickedNode) {
                this.connectNodes(this.selectedNode, clickedNode);
                this.selectedNode = null;
            } else {
                this.selectedNode = clickedNode;
            }
        } else if (clickedNode) {
            this.selectedNode = clickedNode;
            this.showInspector(clickedNode);
            this.isDragging = true;
            this.dragStart = { x: e.clientX - clickedNode.x * this.zoom, y: e.clientY - clickedNode.y * this.zoom };
        } else {
            this.isDragging = true;
            this.dragStart = { x: e.clientX - this.offset.x, y: e.clientY - this.offset.y };
        }
        
        this.render();
    }

    handleMouseMove(e) {
        if (this.isDragging) {
            if (this.selectedNode && this.dragStart) {
                this.selectedNode.x = (e.clientX - this.dragStart.x) / this.zoom;
                this.selectedNode.y = (e.clientY - this.dragStart.y) / this.zoom;
            } else if (this.dragStart) {
                this.offset.x = e.clientX - this.dragStart.x;
                this.offset.y = e.clientY - this.dragStart.y;
            }
            this.render();
        }
    }

    handleMouseUp() {
        this.isDragging = false;
        this.dragStart = null;
        if (this.selectedNode && this.mode !== 'connect') {
            this.saveNodePosition(this.selectedNode);
        }
    }

    handleWheel(e) {
        e.preventDefault();
        const delta = e.deltaY > 0 ? 0.9 : 1.1;
        const newZoom = Math.max(0.2, Math.min(5, this.zoom * delta));
        
        const rect = this.canvas.getBoundingClientRect();
        const mouseX = e.clientX - rect.left;
        const mouseY = e.clientY - rect.top;
        
        this.offset.x = mouseX - (mouseX - this.offset.x) * (newZoom / this.zoom);
        this.offset.y = mouseY - (mouseY - this.offset.y) * (newZoom / this.zoom);
        
        this.zoom = newZoom;
        this.render();
    }

    handleDoubleClick(e) {
        const rect = this.canvas.getBoundingClientRect();
        const x = (e.clientX - rect.left - this.offset.x) / this.zoom;
        const y = (e.clientY - rect.top - this.offset.y) / this.zoom;
        
        const clickedNode = this.getNodeAt(x, y);
        if (clickedNode) {
            this.editNode(clickedNode);
        } else {
            this.addNode(x, y);
        }
    }

    getNodeAt(x, y) {
        for (const node of this.nodes.values()) {
            const distance = Math.hypot(node.x - x, node.y - y);
            if (distance < 30) {
                return node;
            }
        }
        return null;
    }

    addNode(x, y) {
        const id = `node_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
        const node = {
            id,
            x,
            y,
            label: 'New Thought',
            category: 'concept',
            weight: 5,
            color: '#00ffcc',
            connections: new Set()
        };
        
        this.nodes.set(id, node);
        this.updateMetrics();
        this.render();
        this.sendNeuralPulse(node);
        
        // Save to backend
        this.saveNode(node);
    }

    connectNodes(node1, node2) {
        const connectionId = `${node1.id}_${node2.id}`;
        const reverseConnectionId = `${node2.id}_${node1.id}`;
        
        if (!this.connections.has(connectionId) && !this.connections.has(reverseConnectionId)) {
            this.connections.set(connectionId, {
                source: node1.id,
                target: node2.id,
                strength: 1,
                type: 'synapse'
            });
            
            node1.connections.add(node2.id);
            node2.connections.add(node1.id);
            
            this.updateMetrics();
            this.render();
            this.animateConnection(node1, node2);
            
            // Save connection to backend
            this.saveConnection(connectionId);
        }
    }

    render() {
        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
        
        this.ctx.save();
        this.ctx.translate(this.offset.x, this.offset.y);
        this.ctx.scale(this.zoom, this.zoom);
        
        // Draw connections
        this.connections.forEach(connection => {
            const source = this.nodes.get(connection.source);
            const target = this.nodes.get(connection.target);
            
            if (source && target) {
                this.drawConnection(source, target, connection);
            }
        });
        
        // Draw nodes
        this.nodes.forEach(node => {
            this.drawNode(node);
        });
        
        // Draw neural activity
        this.renderNeuralActivity();
        
        this.ctx.restore();
    }

    drawNode(node) {
        const isSelected = node === this.selectedNode;
        const radius = 20 + node.weight;
        
        // Outer glow
        if (isSelected) {
            const gradient = this.ctx.createRadialGradient(node.x, node.y, 0, node.x, node.y, radius * 2);
            gradient.addColorStop(0, `${node.color}40`);
            gradient.addColorStop(1, 'transparent');
            this.ctx.fillStyle = gradient;
            this.ctx.fillRect(node.x - radius * 2, node.y - radius * 2, radius * 4, radius * 4);
        }
        
        // Node body
        this.ctx.beginPath();
        this.ctx.arc(node.x, node.y, radius, 0, Math.PI * 2);
        
        const nodeGradient = this.ctx.createRadialGradient(node.x, node.y, 0, node.x, node.y, radius);
        nodeGradient.addColorStop(0, `${node.color}80`);
        nodeGradient.addColorStop(0.7, `${node.color}40`);
        nodeGradient.addColorStop(1, `${node.color}20`);
        
        this.ctx.fillStyle = nodeGradient;
        this.ctx.fill();
        
        // Border
        this.ctx.strokeStyle = node.color;
        this.ctx.lineWidth = isSelected ? 3 : 2;
        this.ctx.stroke();
        
        // Inner core
        this.ctx.beginPath();
        this.ctx.arc(node.x, node.y, radius * 0.3, 0, Math.PI * 2);
        this.ctx.fillStyle = node.color;
        this.ctx.fill();
        
        // Label
        this.ctx.fillStyle = '#ffffff';
        this.ctx.font = `${12 + node.weight / 2}px Space Grotesk`;
        this.ctx.textAlign = 'center';
        this.ctx.textBaseline = 'middle';
        this.ctx.fillText(node.label, node.x, node.y + radius + 15);
    }

    drawConnection(source, target, connection) {
        const gradient = this.ctx.createLinearGradient(source.x, source.y, target.x, target.y);
        gradient.addColorStop(0, `${source.color}60`);
        gradient.addColorStop(0.5, '#00ffcc40');
        gradient.addColorStop(1, `${target.color}60`);
        
        this.ctx.strokeStyle = gradient;
        this.ctx.lineWidth = 1 + connection.strength;
        this.ctx.beginPath();
        
        // Draw curved connection
        const midX = (source.x + target.x) / 2;
        const midY = (source.y + target.y) / 2;
        const curve = 20;
        
        this.ctx.moveTo(source.x, source.y);
        this.ctx.quadraticCurveTo(midX + curve, midY - curve, target.x, target.y);
        this.ctx.stroke();
        
        // Draw energy pulses along connection
        if (this.neuralActivity.length > 0) {
            this.neuralActivity.forEach(pulse => {
                if (pulse.connection === connection) {
                    const t = pulse.progress;
                    const x = source.x * (1 - t) + target.x * t;
                    const y = source.y * (1 - t) + target.y * t;
                    
                    this.ctx.beginPath();
                    this.ctx.arc(x, y, 3, 0, Math.PI * 2);
                    this.ctx.fillStyle = '#00ffcc';
                    this.ctx.fill();
                }
            });
        }
    }

    renderNeuralActivity() {
        this.neuralActivity = this.neuralActivity.filter(pulse => {
            pulse.progress += 0.02;
            return pulse.progress <= 1;
        });
    }

    startNeuralAnimation() {
        setInterval(() => {
            // Random neural firing
            if (Math.random() < 0.1 && this.connections.size > 0) {
                const connectionArray = Array.from(this.connections.values());
                const randomConnection = connectionArray[Math.floor(Math.random() * connectionArray.length)];
                this.neuralActivity.push({
                    connection: randomConnection,
                    progress: 0
                });
            }
            
            this.render();
        }, 50);
    }

    animateConnection(node1, node2) {
        const steps = 30;
        let step = 0;
        
        const animate = () => {
            if (step < steps) {
                const progress = step / steps;
                const x = node1.x * (1 - progress) + node2.x * progress;
                const y = node1.y * (1 - progress) + node2.y * progress;
                
                this.ctx.beginPath();
                this.ctx.arc(x, y, 5, 0, Math.PI * 2);
                this.ctx.fillStyle = '#00ffcc';
                this.ctx.fill();
                
                step++;
                requestAnimationFrame(animate);
            }
        };
        
        animate();
    }

    sendNeuralPulse(node) {
        const pulseElement = document.createElement('div');
        pulseElement.className = 'neural-pulse';
        pulseElement.style.left = `${node.x}px`;
        pulseElement.style.top = `${node.y}px`;
        document.body.appendChild(pulseElement);
        
        setTimeout(() => pulseElement.remove(), 2000);
    }

    updateMetrics() {
        document.getElementById('nodeCount').textContent = this.nodes.size;
        document.getElementById('synapseCount').textContent = this.connections.size;
    }

    updateToolbarState() {
        document.querySelectorAll('.toolbar-btn').forEach(btn => {
            btn.classList.remove('active');
        });
        
        if (this.mode === 'add') {
            document.getElementById('addNodeBtn')?.classList.add('active');
        } else if (this.mode === 'connect') {
            document.getElementById('connectBtn')?.classList.add('active');
        }
    }

    showInspector(node) {
        const inspector = document.getElementById('inspectorPanel');
        if (!inspector) return;
        
        inspector.style.display = 'block';
        document.getElementById('nodeLabel').value = node.label;
        document.getElementById('nodeCategory').value = node.category;
        document.getElementById('nodeWeight').value = node.weight;
        
        // Update connections list
        const connectionList = document.getElementById('connectionList');
        connectionList.innerHTML = '';
        
        node.connections.forEach(connectedId => {
            const connectedNode = this.nodes.get(connectedId);
            if (connectedNode) {
                const item = document.createElement('div');
                item.className = 'connection-item';
                item.textContent = connectedNode.label;
                connectionList.appendChild(item);
            }
        });
    }

    async autoOrganize() {
        this.showLoader('Organizing neural network...');
        
        try {
            const response = await fetch(`${this.apiUrl}/mindmaps/auto-organize`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    nodes: Array.from(this.nodes.values()),
                    connections: Array.from(this.connections.values()),
                    pattern: 'organic'
                })
            });
            
            if (response.ok) {
                const result = await response.json();
                this.applyOrganization(result);
                this.showNotification('Neural network organized successfully', 'success');
            }
        } catch (error) {
            console.error('Auto-organize failed:', error);
            this.showNotification('Failed to organize network', 'error');
        } finally {
            this.hideLoader();
        }
    }

    applyPattern(pattern) {
        const centerX = this.canvas.width / 2 / this.zoom;
        const centerY = this.canvas.height / 2 / this.zoom;
        const radius = 200;
        
        const nodeArray = Array.from(this.nodes.values());
        
        switch (pattern) {
            case 'hierarchical':
                this.arrangeHierarchical(nodeArray, centerX, centerY);
                break;
            case 'radial':
                this.arrangeRadial(nodeArray, centerX, centerY, radius);
                break;
            case 'organic':
                this.arrangeOrganic(nodeArray, centerX, centerY);
                break;
            case 'matrix':
                this.arrangeMatrix(nodeArray, centerX, centerY);
                break;
        }
        
        this.render();
    }

    arrangeRadial(nodes, centerX, centerY, radius) {
        nodes.forEach((node, index) => {
            const angle = (index / nodes.length) * Math.PI * 2;
            node.x = centerX + Math.cos(angle) * radius;
            node.y = centerY + Math.sin(angle) * radius;
        });
    }

    arrangeHierarchical(nodes, centerX, startY) {
        const levels = this.calculateHierarchy(nodes);
        const levelHeight = 100;
        const nodeSpacing = 80;
        
        levels.forEach((level, levelIndex) => {
            const y = startY + levelIndex * levelHeight;
            const totalWidth = level.length * nodeSpacing;
            const startX = centerX - totalWidth / 2;
            
            level.forEach((node, nodeIndex) => {
                node.x = startX + nodeIndex * nodeSpacing;
                node.y = y;
            });
        });
    }

    arrangeOrganic(nodes, centerX, centerY) {
        // Simulate organic growth pattern
        nodes.forEach((node, index) => {
            const angle = (index / nodes.length) * Math.PI * 2 + Math.random() * 0.5;
            const distance = 100 + Math.random() * 150;
            node.x = centerX + Math.cos(angle) * distance;
            node.y = centerY + Math.sin(angle) * distance;
        });
    }

    arrangeMatrix(nodes, centerX, centerY) {
        const cols = Math.ceil(Math.sqrt(nodes.length));
        const rows = Math.ceil(nodes.length / cols);
        const spacing = 80;
        
        nodes.forEach((node, index) => {
            const col = index % cols;
            const row = Math.floor(index / cols);
            node.x = centerX - (cols * spacing) / 2 + col * spacing;
            node.y = centerY - (rows * spacing) / 2 + row * spacing;
        });
    }

    calculateHierarchy(nodes) {
        // Simple hierarchy calculation based on connections
        const levels = [];
        const visited = new Set();
        
        // Find root nodes (nodes with no incoming connections)
        const roots = nodes.filter(node => {
            const hasIncoming = Array.from(this.connections.values()).some(
                conn => conn.target === node.id
            );
            return !hasIncoming;
        });
        
        if (roots.length === 0) roots.push(nodes[0]);
        
        let currentLevel = roots;
        while (currentLevel.length > 0) {
            levels.push(currentLevel);
            currentLevel.forEach(node => visited.add(node.id));
            
            const nextLevel = [];
            currentLevel.forEach(node => {
                node.connections.forEach(connId => {
                    if (!visited.has(connId)) {
                        const connNode = this.nodes.get(connId);
                        if (connNode && !nextLevel.includes(connNode)) {
                            nextLevel.push(connNode);
                        }
                    }
                });
            });
            
            currentLevel = nextLevel;
        }
        
        return levels;
    }

    async handleSearch(query) {
        if (!query) return;
        
        const mode = document.querySelector('.mode-toggle.active').dataset.mode;
        
        try {
            const response = await fetch(`${this.apiUrl}/mindmaps/search`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ query, mode })
            });
            
            if (response.ok) {
                const results = await response.json();
                this.highlightSearchResults(results);
            }
        } catch (error) {
            console.error('Search failed:', error);
        }
    }

    highlightSearchResults(results) {
        // Reset all nodes
        this.nodes.forEach(node => {
            node.highlighted = false;
        });
        
        // Highlight matching nodes
        results.forEach(result => {
            const node = this.nodes.get(result.id);
            if (node) {
                node.highlighted = true;
                node.searchScore = result.score;
            }
        });
        
        this.render();
    }

    async loadNetworks() {
        try {
            const response = await fetch(`${this.apiUrl}/mindmaps`);
            if (response.ok) {
                const networks = await response.json();
                this.displayNetworks(networks);
            }
        } catch (error) {
            console.error('Failed to load networks:', error);
        }
    }

    displayNetworks(networks) {
        const container = document.getElementById('networkList');
        if (!container) return;
        
        container.innerHTML = '';
        networks.forEach(network => {
            const item = document.createElement('div');
            item.className = 'network-item';
            item.innerHTML = `
                <div class="network-icon">🧠</div>
                <div class="network-info">
                    <div class="network-name">${network.name}</div>
                    <div class="network-stats">${network.nodeCount} nodes • ${network.connectionCount} synapses</div>
                </div>
            `;
            item.addEventListener('click', () => this.loadNetwork(network.id));
            container.appendChild(item);
        });
    }

    async loadNetwork(networkId) {
        this.showLoader('Loading neural network...');
        
        try {
            const response = await fetch(`${this.apiUrl}/mindmaps/${networkId}`);
            if (response.ok) {
                const data = await response.json();
                this.nodes.clear();
                this.connections.clear();
                
                data.nodes.forEach(node => {
                    this.nodes.set(node.id, node);
                });
                
                data.connections.forEach(conn => {
                    this.connections.set(`${conn.source}_${conn.target}`, conn);
                });
                
                this.updateMetrics();
                this.render();
                this.showNotification('Network loaded successfully', 'success');
            }
        } catch (error) {
            console.error('Failed to load network:', error);
            this.showNotification('Failed to load network', 'error');
        } finally {
            this.hideLoader();
        }
    }

    async createNewNetwork() {
        const name = prompt('Enter network name:');
        if (!name) return;
        
        try {
            const response = await fetch(`${this.apiUrl}/mindmaps`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name })
            });
            
            if (response.ok) {
                const network = await response.json();
                this.nodes.clear();
                this.connections.clear();
                this.updateMetrics();
                this.render();
                this.showNotification(`Network "${name}" created`, 'success');
                this.loadNetworks();
            }
        } catch (error) {
            console.error('Failed to create network:', error);
            this.showNotification('Failed to create network', 'error');
        }
    }

    async saveNode(node) {
        try {
            await fetch(`${this.apiUrl}/mindmaps/nodes`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(node)
            });
        } catch (error) {
            console.error('Failed to save node:', error);
        }
    }

    async saveConnection(connectionId) {
        const connection = this.connections.get(connectionId);
        if (!connection) return;
        
        try {
            await fetch(`${this.apiUrl}/mindmaps/connections`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(connection)
            });
        } catch (error) {
            console.error('Failed to save connection:', error);
        }
    }

    async saveNodePosition(node) {
        try {
            await fetch(`${this.apiUrl}/mindmaps/nodes/${node.id}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ x: node.x, y: node.y })
            });
        } catch (error) {
            console.error('Failed to save node position:', error);
        }
    }

    showLoader(message = 'Processing...') {
        const loader = document.getElementById('loader');
        if (loader) {
            loader.style.display = 'flex';
            const loaderText = loader.querySelector('.loader-text');
            if (loaderText) loaderText.textContent = message;
        }
    }

    hideLoader() {
        const loader = document.getElementById('loader');
        if (loader) {
            loader.style.display = 'none';
        }
    }

    showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `neural-notification ${type}`;
        notification.textContent = message;
        document.body.appendChild(notification);
        
        setTimeout(() => {
            notification.style.animation = 'notificationSlide 0.5s ease-out reverse';
            setTimeout(() => notification.remove(), 500);
        }, 3000);
    }

    visualizeConnections() {
        // Create a pulse effect through all connections
        this.connections.forEach((connection, id) => {
            setTimeout(() => {
                this.neuralActivity.push({
                    connection,
                    progress: 0
                });
            }, Math.random() * 1000);
        });
        
        this.showNotification('Visualizing neural pathways...', 'info');
    }

    editNode(node) {
        const newLabel = prompt('Edit node label:', node.label);
        if (newLabel && newLabel !== node.label) {
            node.label = newLabel;
            this.saveNode(node);
            this.render();
        }
    }
}

// Initialize the application
document.addEventListener('DOMContentLoaded', () => {
    const app = new NeuralMindMap();
    
    // Set initial status
    document.getElementById('processingStatus').textContent = 'Active';
    
    // Animate status indicator
    setInterval(() => {
        const dot = document.querySelector('.status-dot');
        if (dot) {
            dot.style.animation = 'none';
            setTimeout(() => {
                dot.style.animation = 'pulse 2s infinite';
            }, 10);
        }
    }, 5000);
    
    // Close inspector
    document.getElementById('closeInspector')?.addEventListener('click', () => {
        document.getElementById('inspectorPanel').style.display = 'none';
    });
    
    // Property changes
    document.getElementById('nodeLabel')?.addEventListener('change', (e) => {
        if (app.selectedNode) {
            app.selectedNode.label = e.target.value;
            app.saveNode(app.selectedNode);
            app.render();
        }
    });
    
    document.getElementById('nodeCategory')?.addEventListener('change', (e) => {
        if (app.selectedNode) {
            app.selectedNode.category = e.target.value;
            app.saveNode(app.selectedNode);
            app.render();
        }
    });
    
    document.getElementById('nodeWeight')?.addEventListener('input', (e) => {
        if (app.selectedNode) {
            app.selectedNode.weight = parseInt(e.target.value);
            document.querySelector('.slider-value').textContent = e.target.value;
            app.saveNode(app.selectedNode);
            app.render();
        }
    });
    
    // Color selection
    document.querySelectorAll('.color-option').forEach(btn => {
        btn.style.backgroundColor = btn.dataset.color;
        btn.addEventListener('click', (e) => {
            if (app.selectedNode) {
                app.selectedNode.color = e.target.dataset.color;
                app.saveNode(app.selectedNode);
                app.render();
            }
        });
    });
    
    // Export to global scope for debugging
    window.neuralMindMap = app;
});
