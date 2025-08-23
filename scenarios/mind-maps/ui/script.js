// Mind Maps - Interactive Knowledge Visualization
class MindMapApp {
    constructor() {
        this.canvas = null;
        this.ctx = null;
        this.currentMapId = null;
        this.nodes = new Map();
        this.connections = [];
        this.selectedNode = null;
        this.hoveredNode = null;
        this.currentTool = 'select';
        this.zoom = 1;
        this.pan = { x: 0, y: 0 };
        this.isDragging = false;
        this.isConnecting = false;
        this.connectionStart = null;
        this.mousePos = { x: 0, y: 0 };
        this.animationFrame = null;
        this.particles = [];
        this.searchMode = 'semantic';
        
        // Color schemes for different node types
        this.nodeColors = {
            root: { bg: '#7c3aed', border: '#6d28d9', text: '#ffffff' },
            category: { bg: '#a78bfa', border: '#8b5cf6', text: '#ffffff' },
            concept: { bg: '#60a5fa', border: '#3b82f6', text: '#ffffff' },
            detail: { bg: '#34d399', border: '#10b981', text: '#ffffff' },
            note: { bg: '#fbbf24', border: '#f59e0b', text: '#1f2937' },
            question: { bg: '#f87171', border: '#ef4444', text: '#ffffff' }
        };
        
        this.init();
    }
    
    async init() {
        await this.setupCanvas();
        this.setupEventListeners();
        this.setupUI();
        this.startAnimation();
        this.loadInitialMap();
        this.createFloatingParticles();
    }
    
    setupCanvas() {
        return new Promise(resolve => {
            this.canvas = document.createElement('canvas');
            this.canvas.id = 'mindMapCanvas';
            this.canvas.className = 'mind-map-canvas';
            
            const viewport = document.querySelector('.canvas-viewport');
            if (!viewport) {
                console.error('Canvas viewport not found');
                return;
            }
            
            viewport.appendChild(this.canvas);
            this.ctx = this.canvas.getContext('2d');
            
            this.resizeCanvas();
            window.addEventListener('resize', () => this.resizeCanvas());
            
            resolve();
        });
    }
    
    resizeCanvas() {
        const viewport = document.querySelector('.canvas-viewport');
        this.canvas.width = viewport.clientWidth;
        this.canvas.height = viewport.clientHeight;
        this.render();
    }
    
    setupEventListeners() {
        // Canvas events
        this.canvas.addEventListener('mousedown', e => this.handleMouseDown(e));
        this.canvas.addEventListener('mousemove', e => this.handleMouseMove(e));
        this.canvas.addEventListener('mouseup', e => this.handleMouseUp(e));
        this.canvas.addEventListener('wheel', e => this.handleWheel(e));
        this.canvas.addEventListener('dblclick', e => this.handleDoubleClick(e));
        
        // Keyboard events
        document.addEventListener('keydown', e => this.handleKeyDown(e));
        document.addEventListener('keyup', e => this.handleKeyUp(e));
    }
    
    setupUI() {
        // Toolbar
        document.querySelectorAll('.tool').forEach(tool => {
            tool.addEventListener('click', e => {
                const toolId = e.currentTarget.id;
                this.handleToolClick(toolId);
            });
        });
        
        // Search
        const searchInput = document.getElementById('searchInput');
        if (searchInput) {
            searchInput.addEventListener('input', e => this.handleSearch(e.target.value));
        }
        
        // Search mode toggle
        document.querySelectorAll('.mode-toggle').forEach(toggle => {
            toggle.addEventListener('click', e => {
                document.querySelectorAll('.mode-toggle').forEach(t => t.classList.remove('active'));
                e.target.classList.add('active');
                this.searchMode = e.target.dataset.mode;
            });
        });
        
        // Sidebar actions
        const newMapBtn = document.getElementById('newMapBtn');
        if (newMapBtn) {
            newMapBtn.addEventListener('click', () => this.createNewMap());
        }
        
        const saveBtn = document.getElementById('saveBtn');
        if (saveBtn) {
            saveBtn.addEventListener('click', () => this.saveMap());
        }
        
        const shareBtn = document.getElementById('shareBtn');
        if (shareBtn) {
            shareBtn.addEventListener('click', () => this.shareMap());
        }
        
        const helpBtn = document.getElementById('helpBtn');
        if (helpBtn) {
            helpBtn.addEventListener('click', () => this.showShortcuts());
        }
        
        // AI suggestions
        const aiSuggestBtn = document.querySelector('.ai-suggest-btn');
        if (aiSuggestBtn) {
            aiSuggestBtn.addEventListener('click', () => this.getAISuggestions());
        }
        
        // Shortcuts panel
        const closeShortcutsBtn = document.getElementById('closeShortcutsBtn');
        if (closeShortcutsBtn) {
            closeShortcutsBtn.addEventListener('click', () => this.hideShortcuts());
        }
        
        // Close shortcuts when clicking outside
        const shortcutsOverlay = document.getElementById('shortcutsOverlay');
        if (shortcutsOverlay) {
            shortcutsOverlay.addEventListener('click', (e) => {
                if (e.target === shortcutsOverlay) {
                    this.hideShortcuts();
                }
            });
        }
    }
    
    handleToolClick(toolId) {
        switch(toolId) {
            case 'selectTool':
                this.setTool('select');
                break;
            case 'nodeTool':
                this.setTool('node');
                break;
            case 'connectTool':
                this.setTool('connect');
                break;
            case 'textTool':
                this.setTool('text');
                break;
            case 'zoomInBtn':
                this.zoomIn();
                break;
            case 'zoomOutBtn':
                this.zoomOut();
                break;
            case 'fitViewBtn':
                this.fitToView();
                break;
            case 'autoOrganizeBtn':
                this.autoOrganize();
                break;
            case 'exportBtn':
                this.exportMap();
                break;
        }
    }
    
    setTool(tool) {
        this.currentTool = tool;
        document.querySelectorAll('.tool').forEach(t => t.classList.remove('active'));
        
        const toolElement = {
            'select': 'selectTool',
            'node': 'nodeTool',
            'connect': 'connectTool',
            'text': 'textTool'
        }[tool];
        
        if (toolElement) {
            const el = document.getElementById(toolElement);
            if (el) el.classList.add('active');
        }
        
        // Update cursor
        this.canvas.style.cursor = {
            'select': 'default',
            'node': 'crosshair',
            'connect': 'crosshair',
            'text': 'text'
        }[tool] || 'default';
    }
    
    handleMouseDown(e) {
        const point = this.getCanvasPoint(e);
        const node = this.getNodeAt(point.x, point.y);
        
        if (this.currentTool === 'select') {
            if (node) {
                this.selectNode(node);
                this.isDragging = true;
                this.dragOffset = {
                    x: point.x - node.x,
                    y: point.y - node.y
                };
            } else {
                this.isDragging = true;
                this.panStart = { x: point.x - this.pan.x, y: point.y - this.pan.y };
                this.canvas.style.cursor = 'grabbing';
            }
        } else if (this.currentTool === 'connect' && node) {
            this.isConnecting = true;
            this.connectionStart = node;
        } else if (this.currentTool === 'node' && !node) {
            this.createNode(point.x, point.y);
        }
    }
    
    handleMouseMove(e) {
        const point = this.getCanvasPoint(e);
        this.mousePos = point;
        
        const node = this.getNodeAt(point.x, point.y);
        this.hoveredNode = node;
        
        if (this.isDragging && this.currentTool === 'select') {
            if (this.selectedNode) {
                this.selectedNode.x = point.x - this.dragOffset.x;
                this.selectedNode.y = point.y - this.dragOffset.y;
            } else if (this.panStart) {
                this.pan.x = point.x - this.panStart.x;
                this.pan.y = point.y - this.panStart.y;
            }
        }
        
        // Update cursor based on hover
        if (!this.isDragging && this.currentTool === 'select') {
            this.canvas.style.cursor = node ? 'pointer' : 'grab';
        }
    }
    
    handleMouseUp(e) {
        const point = this.getCanvasPoint(e);
        
        if (this.isConnecting && this.connectionStart) {
            const endNode = this.getNodeAt(point.x, point.y);
            if (endNode && endNode !== this.connectionStart) {
                this.createConnection(this.connectionStart, endNode);
            }
        }
        
        this.isDragging = false;
        this.isConnecting = false;
        this.connectionStart = null;
        this.panStart = null;
        
        if (this.currentTool === 'select') {
            this.canvas.style.cursor = 'grab';
        }
        
        // Save node position if it was moved
        if (this.selectedNode) {
            this.saveNodePosition(this.selectedNode);
        }
    }
    
    handleWheel(e) {
        e.preventDefault();
        const delta = e.deltaY > 0 ? 0.9 : 1.1;
        const newZoom = Math.max(0.1, Math.min(5, this.zoom * delta));
        
        // Zoom towards mouse position
        const point = this.getCanvasPoint(e);
        const zoomPoint = {
            x: (point.x - this.pan.x) / this.zoom,
            y: (point.y - this.pan.y) / this.zoom
        };
        
        this.zoom = newZoom;
        
        this.pan.x = point.x - zoomPoint.x * this.zoom;
        this.pan.y = point.y - zoomPoint.y * this.zoom;
        
        this.updateZoomIndicator();
    }
    
    handleDoubleClick(e) {
        const point = this.getCanvasPoint(e);
        const node = this.getNodeAt(point.x, point.y);
        
        if (node) {
            this.editNode(node);
        } else if (this.currentTool === 'select') {
            this.createNode(point.x, point.y);
        }
    }
    
    handleKeyDown(e) {
        // Keyboard shortcuts
        if (e.ctrlKey || e.metaKey) {
            switch(e.key) {
                case 'z':
                    e.preventDefault();
                    this.undo();
                    break;
                case 'y':
                    e.preventDefault();
                    this.redo();
                    break;
                case 's':
                    e.preventDefault();
                    this.saveMap();
                    break;
                case 'a':
                    e.preventDefault();
                    this.selectAll();
                    break;
                case 'n':
                    e.preventDefault();
                    this.createNewMap();
                    break;
                case 'f':
                    e.preventDefault();
                    this.focusSearch();
                    break;
                case '+':
                case '=':
                    e.preventDefault();
                    this.zoomIn();
                    break;
                case '-':
                    e.preventDefault();
                    this.zoomOut();
                    break;
                case '/':
                    e.preventDefault();
                    this.toggleShortcuts();
                    break;
            }
        } else {
            switch(e.key) {
                case 'Delete':
                    if (this.selectedNode) {
                        this.deleteNode(this.selectedNode);
                    }
                    break;
                case 'Escape':
                    this.hideShortcuts();
                    this.deselectAll();
                    this.setTool('select');
                    break;
                case 'n':
                    this.setTool('node');
                    break;
                case 'c':
                    this.setTool('connect');
                    break;
                case 't':
                    this.setTool('text');
                    break;
                case 'v':
                    this.setTool('select');
                    break;
                case '?':
                    e.preventDefault();
                    this.toggleShortcuts();
                    break;
                case '0':
                    this.fitToView();
                    break;
                case '+':
                case '=':
                    this.zoomIn();
                    break;
                case '-':
                    this.zoomOut();
                    break;
                case ' ':
                    e.preventDefault();
                    // Toggle pan mode or similar functionality
                    break;
            }
        }
    }
    
    handleKeyUp(e) {
        // Handle key release if needed
    }
    
    getCanvasPoint(e) {
        const rect = this.canvas.getBoundingClientRect();
        return {
            x: e.clientX - rect.left,
            y: e.clientY - rect.top
        };
    }
    
    getNodeAt(x, y) {
        // Transform to world coordinates
        const worldX = (x - this.pan.x) / this.zoom;
        const worldY = (y - this.pan.y) / this.zoom;
        
        for (const [id, node] of this.nodes) {
            const dx = worldX - node.x;
            const dy = worldY - node.y;
            const distance = Math.sqrt(dx * dx + dy * dy);
            
            if (distance <= node.radius) {
                return node;
            }
        }
        return null;
    }
    
    createNode(x, y, type = 'concept') {
        const id = `node_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
        
        const worldX = (x - this.pan.x) / this.zoom;
        const worldY = (y - this.pan.y) / this.zoom;
        
        const node = {
            id,
            x: worldX,
            y: worldY,
            radius: 40,
            type,
            title: 'New Node',
            content: '',
            color: this.nodeColors[type] || this.nodeColors.concept,
            tags: [],
            metadata: {
                created: new Date().toISOString(),
                modified: new Date().toISOString()
            }
        };
        
        this.nodes.set(id, node);
        this.selectNode(node);
        this.editNode(node);
        
        // Animate node creation
        this.animateNodeCreation(node);
        
        return node;
    }
    
    animateNodeCreation(node) {
        const startRadius = 0;
        const targetRadius = node.radius;
        const duration = 300;
        const startTime = Date.now();
        
        const animate = () => {
            const elapsed = Date.now() - startTime;
            const progress = Math.min(elapsed / duration, 1);
            
            // Elastic easing
            const easing = 1 - Math.pow(2, -10 * progress) * Math.cos(progress * Math.PI * 2);
            node.radius = startRadius + (targetRadius - startRadius) * easing;
            
            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };
        
        animate();
    }
    
    editNode(node) {
        const input = document.createElement('input');
        input.type = 'text';
        input.value = node.title;
        input.style.position = 'absolute';
        
        const rect = this.canvas.getBoundingClientRect();
        const screenX = node.x * this.zoom + this.pan.x + rect.left;
        const screenY = node.y * this.zoom + this.pan.y + rect.top;
        
        input.style.left = `${screenX - 50}px`;
        input.style.top = `${screenY - 10}px`;
        input.style.width = '100px';
        input.style.textAlign = 'center';
        input.style.border = '2px solid #7c3aed';
        input.style.borderRadius = '4px';
        input.style.padding = '4px';
        input.style.background = 'white';
        input.style.zIndex = '1000';
        
        document.body.appendChild(input);
        input.focus();
        input.select();
        
        const save = () => {
            node.title = input.value || 'Untitled';
            document.body.removeChild(input);
            this.saveNode(node);
        };
        
        input.addEventListener('blur', save);
        input.addEventListener('keydown', e => {
            if (e.key === 'Enter') {
                save();
            } else if (e.key === 'Escape') {
                document.body.removeChild(input);
            }
        });
    }
    
    deleteNode(node) {
        // Remove connections
        this.connections = this.connections.filter(conn => 
            conn.from !== node.id && conn.to !== node.id
        );
        
        // Remove node
        this.nodes.delete(node.id);
        
        if (this.selectedNode === node) {
            this.selectedNode = null;
        }
        
        // Animate deletion
        this.animateNodeDeletion(node);
    }
    
    animateNodeDeletion(node) {
        const duration = 300;
        const startTime = Date.now();
        const startRadius = node.radius;
        
        const animate = () => {
            const elapsed = Date.now() - startTime;
            const progress = Math.min(elapsed / duration, 1);
            
            node.radius = startRadius * (1 - progress);
            node.opacity = 1 - progress;
            
            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };
        
        animate();
    }
    
    createConnection(fromNode, toNode) {
        const connection = {
            id: `conn_${Date.now()}`,
            from: fromNode.id,
            to: toNode.id,
            type: 'default',
            animated: true
        };
        
        this.connections.push(connection);
        this.saveConnection(connection);
    }
    
    selectNode(node) {
        this.selectedNode = node;
        this.showProperties(node);
    }
    
    deselectAll() {
        this.selectedNode = null;
        this.hideProperties();
    }
    
    showProperties(node) {
        const panel = document.querySelector('.properties-panel');
        if (!panel) return;
        
        panel.classList.add('open');
        
        // Update property fields
        const titleInput = panel.querySelector('#nodeTitle');
        if (titleInput) titleInput.value = node.title;
        
        const contentInput = panel.querySelector('#nodeContent');
        if (contentInput) contentInput.value = node.content || '';
        
        const typeSelect = panel.querySelector('#nodeType');
        if (typeSelect) typeSelect.value = node.type;
        
        // Update color picker
        this.updateColorPicker(node);
    }
    
    hideProperties() {
        const panel = document.querySelector('.properties-panel');
        if (panel) panel.classList.remove('open');
    }
    
    updateColorPicker(node) {
        // Implementation for color picker
    }
    
    async handleSearch(query) {
        if (!query) return;
        
        try {
            const response = await fetch('/api/mindmaps/search', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    query,
                    mode: this.searchMode,
                    mindmap_id: this.currentMapId
                })
            });
            
            const results = await response.json();
            this.displaySearchResults(results);
        } catch (error) {
            console.error('Search failed:', error);
        }
    }
    
    displaySearchResults(results) {
        // Highlight matching nodes
        results.forEach(result => {
            const node = this.nodes.get(result.node_id);
            if (node) {
                node.highlighted = true;
                node.highlightStrength = result.score;
            }
        });
    }
    
    async getAISuggestions() {
        if (!this.selectedNode) return;
        
        try {
            const response = await fetch('/api/mindmaps/auto-organize', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    mindmap_id: this.currentMapId,
                    content: this.selectedNode.title + ' ' + this.selectedNode.content,
                    auto_categorize: true,
                    suggest_connections: true
                })
            });
            
            const suggestions = await response.json();
            this.displaySuggestions(suggestions);
        } catch (error) {
            console.error('Failed to get AI suggestions:', error);
        }
    }
    
    displaySuggestions(suggestions) {
        const list = document.querySelector('.suggestions-list');
        if (!list) return;
        
        list.innerHTML = '';
        
        if (suggestions.connections) {
            suggestions.connections.forEach(conn => {
                const item = document.createElement('div');
                item.className = 'suggestion-item';
                item.textContent = `Connect to: ${conn.title}`;
                item.onclick = () => {
                    const targetNode = Array.from(this.nodes.values()).find(n => n.title === conn.title);
                    if (targetNode) {
                        this.createConnection(this.selectedNode, targetNode);
                    }
                };
                list.appendChild(item);
            });
        }
        
        if (suggestions.tags) {
            suggestions.tags.forEach(tag => {
                const item = document.createElement('div');
                item.className = 'suggestion-item';
                item.textContent = `Add tag: ${tag}`;
                item.onclick = () => {
                    if (!this.selectedNode.tags.includes(tag)) {
                        this.selectedNode.tags.push(tag);
                        this.saveNode(this.selectedNode);
                    }
                };
                list.appendChild(item);
            });
        }
    }
    
    async autoOrganize() {
        try {
            const response = await fetch('/api/mindmaps/auto-organize', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    mindmap_id: this.currentMapId,
                    apply_changes: true
                })
            });
            
            const result = await response.json();
            if (result.success) {
                this.loadMap(this.currentMapId);
                this.showNotification('Map reorganized successfully');
            }
        } catch (error) {
            console.error('Auto-organize failed:', error);
        }
    }
    
    zoomIn() {
        this.zoom = Math.min(5, this.zoom * 1.2);
        this.updateZoomIndicator();
    }
    
    zoomOut() {
        this.zoom = Math.max(0.1, this.zoom / 1.2);
        this.updateZoomIndicator();
    }
    
    fitToView() {
        if (this.nodes.size === 0) return;
        
        let minX = Infinity, minY = Infinity;
        let maxX = -Infinity, maxY = -Infinity;
        
        this.nodes.forEach(node => {
            minX = Math.min(minX, node.x - node.radius);
            minY = Math.min(minY, node.y - node.radius);
            maxX = Math.max(maxX, node.x + node.radius);
            maxY = Math.max(maxY, node.y + node.radius);
        });
        
        const width = maxX - minX;
        const height = maxY - minY;
        const centerX = (minX + maxX) / 2;
        const centerY = (minY + maxY) / 2;
        
        const scaleX = (this.canvas.width - 100) / width;
        const scaleY = (this.canvas.height - 100) / height;
        this.zoom = Math.min(scaleX, scaleY, 2);
        
        this.pan.x = this.canvas.width / 2 - centerX * this.zoom;
        this.pan.y = this.canvas.height / 2 - centerY * this.zoom;
        
        this.updateZoomIndicator();
    }
    
    updateZoomIndicator() {
        const indicator = document.querySelector('.zoom-indicator');
        if (indicator) {
            indicator.textContent = `${Math.round(this.zoom * 100)}%`;
        }
    }
    
    createFloatingParticles() {
        const particleCount = 20;
        for (let i = 0; i < particleCount; i++) {
            this.particles.push({
                x: Math.random() * this.canvas.width,
                y: Math.random() * this.canvas.height,
                vx: (Math.random() - 0.5) * 0.5,
                vy: (Math.random() - 0.5) * 0.5,
                radius: Math.random() * 3 + 1,
                opacity: Math.random() * 0.3 + 0.1
            });
        }
    }
    
    updateParticles() {
        this.particles.forEach(particle => {
            particle.x += particle.vx;
            particle.y += particle.vy;
            
            // Wrap around edges
            if (particle.x < 0) particle.x = this.canvas.width;
            if (particle.x > this.canvas.width) particle.x = 0;
            if (particle.y < 0) particle.y = this.canvas.height;
            if (particle.y > this.canvas.height) particle.y = 0;
        });
    }
    
    startAnimation() {
        const animate = () => {
            this.render();
            this.animationFrame = requestAnimationFrame(animate);
        };
        animate();
    }
    
    render() {
        if (!this.ctx) return;
        
        // Clear canvas
        this.ctx.clearRect(0, 0, this.canvas.width, this.canvas.height);
        
        // Save context
        this.ctx.save();
        
        // Apply zoom and pan
        this.ctx.translate(this.pan.x, this.pan.y);
        this.ctx.scale(this.zoom, this.zoom);
        
        // Render connections
        this.renderConnections();
        
        // Render nodes
        this.renderNodes();
        
        // Restore context
        this.ctx.restore();
        
        // Render particles (not affected by zoom/pan)
        this.renderParticles();
        
        // Render UI overlays
        this.renderOverlays();
        
        // Update particles
        this.updateParticles();
    }
    
    renderConnections() {
        this.ctx.strokeStyle = '#cbd5e0';
        this.ctx.lineWidth = 2 / this.zoom;
        
        this.connections.forEach(conn => {
            const fromNode = this.nodes.get(conn.from);
            const toNode = this.nodes.get(conn.to);
            
            if (!fromNode || !toNode) return;
            
            this.ctx.beginPath();
            
            // Calculate control points for bezier curve
            const dx = toNode.x - fromNode.x;
            const dy = toNode.y - fromNode.y;
            const distance = Math.sqrt(dx * dx + dy * dy);
            
            const cp1x = fromNode.x + dx * 0.25;
            const cp1y = fromNode.y + dy * 0.25 - distance * 0.1;
            const cp2x = fromNode.x + dx * 0.75;
            const cp2y = fromNode.y + dy * 0.75 - distance * 0.1;
            
            this.ctx.moveTo(fromNode.x, fromNode.y);
            this.ctx.bezierCurveTo(cp1x, cp1y, cp2x, cp2y, toNode.x, toNode.y);
            
            if (conn.animated) {
                this.ctx.setLineDash([5, 5]);
                this.ctx.lineDashOffset = (Date.now() / 50) % 10;
            }
            
            this.ctx.stroke();
            this.ctx.setLineDash([]);
            
            // Draw arrowhead
            this.drawArrowhead(cp2x, cp2y, toNode.x, toNode.y);
        });
        
        // Draw temporary connection while connecting
        if (this.isConnecting && this.connectionStart) {
            this.ctx.strokeStyle = '#7c3aed';
            this.ctx.lineWidth = 2 / this.zoom;
            this.ctx.setLineDash([5, 5]);
            
            this.ctx.beginPath();
            this.ctx.moveTo(this.connectionStart.x, this.connectionStart.y);
            
            const worldX = (this.mousePos.x - this.pan.x) / this.zoom;
            const worldY = (this.mousePos.y - this.pan.y) / this.zoom;
            this.ctx.lineTo(worldX, worldY);
            this.ctx.stroke();
            this.ctx.setLineDash([]);
        }
    }
    
    drawArrowhead(fromX, fromY, toX, toY) {
        const angle = Math.atan2(toY - fromY, toX - fromX);
        const arrowLength = 10 / this.zoom;
        const arrowAngle = Math.PI / 6;
        
        this.ctx.beginPath();
        this.ctx.moveTo(toX, toY);
        this.ctx.lineTo(
            toX - arrowLength * Math.cos(angle - arrowAngle),
            toY - arrowLength * Math.sin(angle - arrowAngle)
        );
        this.ctx.moveTo(toX, toY);
        this.ctx.lineTo(
            toX - arrowLength * Math.cos(angle + arrowAngle),
            toY - arrowLength * Math.sin(angle + arrowAngle)
        );
        this.ctx.stroke();
    }
    
    renderNodes() {
        this.nodes.forEach(node => {
            if (node.opacity !== undefined && node.opacity <= 0) return;
            
            const isSelected = node === this.selectedNode;
            const isHovered = node === this.hoveredNode;
            
            // Node shadow
            if (isSelected || isHovered) {
                this.ctx.shadowColor = 'rgba(124, 58, 237, 0.3)';
                this.ctx.shadowBlur = 20 / this.zoom;
                this.ctx.shadowOffsetX = 0;
                this.ctx.shadowOffsetY = 4 / this.zoom;
            }
            
            // Node background
            const gradient = this.ctx.createRadialGradient(
                node.x - node.radius / 3, node.y - node.radius / 3, 0,
                node.x, node.y, node.radius
            );
            gradient.addColorStop(0, this.lightenColor(node.color.bg, 20));
            gradient.addColorStop(1, node.color.bg);
            
            this.ctx.fillStyle = gradient;
            this.ctx.globalAlpha = node.opacity || 1;
            
            this.ctx.beginPath();
            this.ctx.arc(node.x, node.y, node.radius, 0, Math.PI * 2);
            this.ctx.fill();
            
            // Node border
            this.ctx.strokeStyle = node.color.border;
            this.ctx.lineWidth = (isSelected ? 3 : 2) / this.zoom;
            this.ctx.stroke();
            
            // Reset shadow
            this.ctx.shadowColor = 'transparent';
            this.ctx.shadowBlur = 0;
            
            // Node text
            this.ctx.fillStyle = node.color.text;
            this.ctx.font = `${14 / this.zoom}px Inter, sans-serif`;
            this.ctx.textAlign = 'center';
            this.ctx.textBaseline = 'middle';
            
            // Wrap text if too long
            const maxWidth = node.radius * 1.5;
            const words = node.title.split(' ');
            const lines = [];
            let currentLine = '';
            
            words.forEach(word => {
                const testLine = currentLine ? `${currentLine} ${word}` : word;
                const metrics = this.ctx.measureText(testLine);
                
                if (metrics.width > maxWidth && currentLine) {
                    lines.push(currentLine);
                    currentLine = word;
                } else {
                    currentLine = testLine;
                }
            });
            if (currentLine) lines.push(currentLine);
            
            const lineHeight = 16 / this.zoom;
            const startY = node.y - ((lines.length - 1) * lineHeight) / 2;
            
            lines.forEach((line, index) => {
                this.ctx.fillText(line, node.x, startY + index * lineHeight);
            });
            
            // Node type indicator
            if (node.type !== 'concept') {
                this.ctx.font = `${10 / this.zoom}px Inter, sans-serif`;
                this.ctx.fillStyle = node.color.border;
                this.ctx.fillText(node.type.toUpperCase(), node.x, node.y + node.radius + 10 / this.zoom);
            }
            
            // Highlight for search results
            if (node.highlighted) {
                this.ctx.strokeStyle = '#f59e0b';
                this.ctx.lineWidth = 4 / this.zoom;
                this.ctx.setLineDash([5, 5]);
                this.ctx.stroke();
                this.ctx.setLineDash([]);
            }
            
            this.ctx.globalAlpha = 1;
        });
    }
    
    renderParticles() {
        this.ctx.fillStyle = 'rgba(124, 58, 237, 0.2)';
        
        this.particles.forEach(particle => {
            this.ctx.globalAlpha = particle.opacity;
            this.ctx.beginPath();
            this.ctx.arc(particle.x, particle.y, particle.radius, 0, Math.PI * 2);
            this.ctx.fill();
        });
        
        this.ctx.globalAlpha = 1;
    }
    
    renderOverlays() {
        // Render any UI overlays here
    }
    
    lightenColor(color, percent) {
        const num = parseInt(color.replace('#', ''), 16);
        const amt = Math.round(2.55 * percent);
        const R = (num >> 16) + amt;
        const G = (num >> 8 & 0x00FF) + amt;
        const B = (num & 0x0000FF) + amt;
        
        return '#' + (0x1000000 + (R < 255 ? R < 1 ? 0 : R : 255) * 0x10000 +
            (G < 255 ? G < 1 ? 0 : G : 255) * 0x100 +
            (B < 255 ? B < 1 ? 0 : B : 255))
            .toString(16).slice(1);
    }
    
    async loadInitialMap() {
        // Create a welcome map or load the last used map
        this.createWelcomeMap();
    }
    
    createWelcomeMap() {
        // Create root node
        const root = this.createNode(400, 300, 'root');
        root.title = 'My Knowledge Map';
        
        // Create some initial nodes
        const ideas = this.createNode(200, 200, 'category');
        ideas.title = 'Ideas';
        
        const projects = this.createNode(600, 200, 'category');
        projects.title = 'Projects';
        
        const notes = this.createNode(200, 400, 'category');
        notes.title = 'Notes';
        
        const research = this.createNode(600, 400, 'category');
        research.title = 'Research';
        
        // Create connections
        this.createConnection(root, ideas);
        this.createConnection(root, projects);
        this.createConnection(root, notes);
        this.createConnection(root, research);
        
        // Fit to view
        setTimeout(() => this.fitToView(), 100);
    }
    
    async saveNode(node) {
        try {
            await fetch(`/api/mindmaps/${this.currentMapId}/nodes`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(node)
            });
        } catch (error) {
            console.error('Failed to save node:', error);
        }
    }
    
    async saveNodePosition(node) {
        // Save node position to backend
        this.saveNode(node);
    }
    
    async saveConnection(connection) {
        try {
            await fetch(`/api/mindmaps/${this.currentMapId}/connections`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(connection)
            });
        } catch (error) {
            console.error('Failed to save connection:', error);
        }
    }
    
    async saveMap() {
        const mapData = {
            id: this.currentMapId,
            nodes: Array.from(this.nodes.values()),
            connections: this.connections,
            metadata: {
                zoom: this.zoom,
                pan: this.pan,
                modified: new Date().toISOString()
            }
        };
        
        try {
            await fetch(`/api/mindmaps/${this.currentMapId}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(mapData)
            });
            
            this.showNotification('Map saved successfully');
        } catch (error) {
            console.error('Failed to save map:', error);
            this.showNotification('Failed to save map', 'error');
        }
    }
    
    async loadMap(mapId) {
        try {
            const response = await fetch(`/api/mindmaps/${mapId}`);
            const mapData = await response.json();
            
            this.currentMapId = mapId;
            this.nodes.clear();
            this.connections = [];
            
            mapData.nodes.forEach(node => {
                this.nodes.set(node.id, node);
            });
            
            this.connections = mapData.connections || [];
            
            if (mapData.metadata) {
                this.zoom = mapData.metadata.zoom || 1;
                this.pan = mapData.metadata.pan || { x: 0, y: 0 };
            }
            
            this.fitToView();
        } catch (error) {
            console.error('Failed to load map:', error);
        }
    }
    
    async createNewMap() {
        const name = prompt('Enter map name:');
        if (!name) return;
        
        try {
            const response = await fetch('/api/mindmaps', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ title: name })
            });
            
            const newMap = await response.json();
            this.loadMap(newMap.id);
        } catch (error) {
            console.error('Failed to create map:', error);
        }
    }
    
    async exportMap() {
        const format = prompt('Export format (json, markdown, dot, opml):');
        if (!format) return;
        
        try {
            const response = await fetch(`/api/mindmaps/${this.currentMapId}/export?format=${format}`);
            const data = await response.json();
            
            // Download file
            const blob = new Blob([data.content], { type: 'text/plain' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = data.filename;
            a.click();
            URL.revokeObjectURL(url);
        } catch (error) {
            console.error('Failed to export map:', error);
        }
    }
    
    shareMap() {
        const shareUrl = `${window.location.origin}/mindmaps/${this.currentMapId}`;
        navigator.clipboard.writeText(shareUrl);
        this.showNotification('Share link copied to clipboard');
    }
    
    showNotification(message, type = 'success') {
        const notification = document.createElement('div');
        notification.className = `notification ${type}`;
        notification.textContent = message;
        notification.style.cssText = `
            position: fixed;
            bottom: 20px;
            right: 20px;
            padding: 12px 20px;
            background: ${type === 'error' ? '#ef4444' : '#10b981'};
            color: white;
            border-radius: 8px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
            z-index: 10000;
            animation: slideInUp 0.3s ease;
        `;
        
        document.body.appendChild(notification);
        
        setTimeout(() => {
            notification.style.animation = 'fadeOut 0.3s ease';
            setTimeout(() => document.body.removeChild(notification), 300);
        }, 3000);
    }
    
    // Keyboard shortcuts panel functions
    toggleShortcuts() {
        const overlay = document.getElementById('shortcutsOverlay');
        if (overlay) {
            const isVisible = overlay.classList.contains('active');
            if (isVisible) {
                this.hideShortcuts();
            } else {
                this.showShortcuts();
            }
        }
    }
    
    showShortcuts() {
        const overlay = document.getElementById('shortcutsOverlay');
        if (overlay) {
            overlay.classList.add('active');
            // Focus management for accessibility
            const closeBtn = document.getElementById('closeShortcutsBtn');
            if (closeBtn) {
                closeBtn.focus();
            }
        }
    }
    
    hideShortcuts() {
        const overlay = document.getElementById('shortcutsOverlay');
        if (overlay) {
            overlay.classList.remove('active');
        }
    }
    
    focusSearch() {
        const searchInput = document.getElementById('searchInput');
        if (searchInput) {
            searchInput.focus();
            searchInput.select();
        }
    }
    
    undo() {
        // Implement undo functionality
        console.log('Undo');
    }
    
    redo() {
        // Implement redo functionality
        console.log('Redo');
    }
    
    selectAll() {
        // Select all nodes
        console.log('Select all');
    }
}

// Initialize app when DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    window.mindMapApp = new MindMapApp();
});

// Add CSS animations
const style = document.createElement('style');
style.textContent = `
@keyframes slideInUp {
    from {
        transform: translateY(20px);
        opacity: 0;
    }
    to {
        transform: translateY(0);
        opacity: 1;
    }
}

@keyframes fadeOut {
    from {
        opacity: 1;
    }
    to {
        opacity: 0;
    }
}

.notification {
    animation: slideInUp 0.3s ease;
}
`;
document.head.appendChild(style);