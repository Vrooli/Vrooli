// Enhanced connection rendering for Mind Maps
// This module extends the existing script.js with advanced visualization features

class EnhancedMindMapConnections {
    constructor(app) {
        this.app = app;
        this.connectionStyles = {
            'semantic_similarity': {
                color: '#8B5CF6',
                glow: true,
                animated: true,
                dashPattern: null,
                label: 'Similar'
            },
            'depends_on': {
                color: '#3B82F6',
                glow: false,
                animated: false,
                dashPattern: [10, 5],
                label: 'Depends On',
                arrow: 'large'
            },
            'conflicts_with': {
                color: '#EF4444',
                glow: false,
                animated: false,
                dashPattern: [5, 5],
                label: 'Conflicts',
                bidirectional: true
            },
            'related': {
                color: '#10B981',
                glow: false,
                animated: false,
                dashPattern: null,
                label: 'Related'
            },
            'parent_child': {
                color: '#6B7280',
                glow: false,
                animated: false,
                dashPattern: null,
                label: null,
                hierarchical: true
            }
        };
    }

    renderEnhancedConnections(ctx, connections, nodes, zoom, animationTime) {
        // Group connections by type for layered rendering
        const groupedConnections = this.groupConnectionsByType(connections);
        
        // Render each connection type layer
        Object.entries(groupedConnections).forEach(([type, conns]) => {
            this.renderConnectionLayer(ctx, conns, nodes, type, zoom, animationTime);
        });
    }

    groupConnectionsByType(connections) {
        const grouped = {};
        connections.forEach(conn => {
            const type = conn.relationship_type || 'related';
            if (!grouped[type]) grouped[type] = [];
            grouped[type].push(conn);
        });
        return grouped;
    }

    renderConnectionLayer(ctx, connections, nodes, type, zoom, animationTime) {
        const style = this.connectionStyles[type] || this.connectionStyles['related'];
        
        connections.forEach(conn => {
            const fromNode = nodes.get(conn.from || conn.from_node_id);
            const toNode = nodes.get(conn.to || conn.to_node_id);
            
            if (!fromNode || !toNode) return;
            
            // Calculate connection strength (0-1)
            const strength = conn.strength || 1.0;
            
            // Setup context for this connection
            ctx.save();
            
            // Apply glow effect for certain types
            if (style.glow) {
                ctx.shadowColor = style.color;
                ctx.shadowBlur = 10 * strength;
            }
            
            // Set line style based on connection type and strength
            ctx.strokeStyle = this.getColorWithAlpha(style.color, 0.3 + (strength * 0.7));
            ctx.lineWidth = (1 + strength * 2) / zoom;
            
            // Apply dash pattern if specified
            if (style.dashPattern) {
                ctx.setLineDash(style.dashPattern.map(d => d / zoom));
                if (style.animated) {
                    ctx.lineDashOffset = (animationTime / 50) % 20;
                }
            }
            
            // Draw the connection path
            this.drawAdvancedPath(ctx, fromNode, toNode, type, strength);
            
            // Draw connection label if specified
            if (style.label && zoom > 0.5) {
                this.drawConnectionLabel(ctx, fromNode, toNode, style.label, strength, zoom);
            }
            
            // Draw directional indicators
            if (!style.bidirectional) {
                this.drawEnhancedArrowhead(ctx, fromNode, toNode, style, strength, zoom);
            } else {
                // Draw arrows at both ends for bidirectional
                this.drawEnhancedArrowhead(ctx, fromNode, toNode, style, strength, zoom);
                this.drawEnhancedArrowhead(ctx, toNode, fromNode, style, strength, zoom);
            }
            
            ctx.restore();
        });
    }

    drawAdvancedPath(ctx, fromNode, toNode, type, strength) {
        const dx = toNode.x - fromNode.x;
        const dy = toNode.y - fromNode.y;
        const distance = Math.sqrt(dx * dx + dy * dy);
        
        ctx.beginPath();
        
        if (type === 'semantic_similarity') {
            // Draw a more organic, flowing curve for semantic connections
            const wobble = Math.sin(Date.now() / 1000) * 5 * strength;
            const cp1x = fromNode.x + dx * 0.3 + wobble;
            const cp1y = fromNode.y + dy * 0.3 - distance * 0.2 * strength;
            const cp2x = fromNode.x + dx * 0.7 - wobble;
            const cp2y = fromNode.y + dy * 0.7 - distance * 0.2 * strength;
            
            ctx.moveTo(fromNode.x, fromNode.y);
            ctx.bezierCurveTo(cp1x, cp1y, cp2x, cp2y, toNode.x, toNode.y);
        } else if (type === 'parent_child') {
            // Straight hierarchical line
            ctx.moveTo(fromNode.x, fromNode.y);
            ctx.lineTo(toNode.x, toNode.y);
        } else {
            // Standard curved connection
            const cp1x = fromNode.x + dx * 0.25;
            const cp1y = fromNode.y + dy * 0.25 - distance * 0.1;
            const cp2x = fromNode.x + dx * 0.75;
            const cp2y = fromNode.y + dy * 0.75 - distance * 0.1;
            
            ctx.moveTo(fromNode.x, fromNode.y);
            ctx.bezierCurveTo(cp1x, cp1y, cp2x, cp2y, toNode.x, toNode.y);
        }
        
        ctx.stroke();
    }

    drawConnectionLabel(ctx, fromNode, toNode, label, strength, zoom) {
        const midX = (fromNode.x + toNode.x) / 2;
        const midY = (fromNode.y + toNode.y) / 2;
        
        ctx.save();
        ctx.font = `${12 / zoom}px Inter`;
        ctx.fillStyle = this.getColorWithAlpha('#9CA3AF', strength);
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        
        // Add background for better readability
        const metrics = ctx.measureText(label);
        const padding = 4 / zoom;
        
        ctx.fillStyle = 'rgba(15, 15, 30, 0.8)';
        ctx.fillRect(
            midX - metrics.width / 2 - padding,
            midY - (12 / zoom) / 2 - padding,
            metrics.width + padding * 2,
            (12 / zoom) + padding * 2
        );
        
        ctx.fillStyle = this.getColorWithAlpha('#9CA3AF', strength);
        ctx.fillText(label, midX, midY);
        ctx.restore();
    }

    drawEnhancedArrowhead(ctx, fromNode, toNode, style, strength, zoom) {
        const dx = toNode.x - fromNode.x;
        const dy = toNode.y - fromNode.y;
        const angle = Math.atan2(dy, dx);
        
        const arrowSize = style.arrow === 'large' ? 12 : 8;
        const size = (arrowSize * strength) / zoom;
        
        ctx.save();
        ctx.translate(toNode.x, toNode.y);
        ctx.rotate(angle);
        
        ctx.beginPath();
        ctx.moveTo(-size, -size / 2);
        ctx.lineTo(0, 0);
        ctx.lineTo(-size, size / 2);
        
        ctx.strokeStyle = style.color;
        ctx.lineWidth = 2 / zoom;
        ctx.stroke();
        
        ctx.restore();
    }

    getColorWithAlpha(color, alpha) {
        // Convert hex to rgba
        const r = parseInt(color.slice(1, 3), 16);
        const g = parseInt(color.slice(3, 5), 16);
        const b = parseInt(color.slice(5, 7), 16);
        return `rgba(${r}, ${g}, ${b}, ${alpha})`;
    }

    // Method to highlight connections based on semantic similarity
    highlightSemanticConnections(connections, threshold = 0.7) {
        return connections.map(conn => {
            if (conn.relationship_type === 'semantic_similarity' && conn.strength >= threshold) {
                return {
                    ...conn,
                    highlighted: true,
                    pulseAnimation: true
                };
            }
            return conn;
        });
    }

    // Method to animate connection strength changes
    animateStrengthChange(connection, newStrength, duration = 1000) {
        const startStrength = connection.strength || 1.0;
        const startTime = Date.now();
        
        const animate = () => {
            const elapsed = Date.now() - startTime;
            const progress = Math.min(elapsed / duration, 1);
            
            // Ease-in-out animation
            const easeProgress = progress < 0.5 
                ? 2 * progress * progress 
                : -1 + (4 - 2 * progress) * progress;
            
            connection.strength = startStrength + (newStrength - startStrength) * easeProgress;
            
            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };
        
        animate();
    }
}

// Export for use in main script
if (typeof module !== 'undefined' && module.exports) {
    module.exports = EnhancedMindMapConnections;
}