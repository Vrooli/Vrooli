/**
 * Utility formatters for AI Model Orchestra Controller
 */

export const formatters = {
    /**
     * Format bytes to human readable size
     */
    formatBytes(bytes, decimals = 2) {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const dm = decimals < 0 ? 0 : decimals;
        const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
    },

    /**
     * Format memory in GB
     */
    formatMemoryGB(gb, decimals = 2) {
        return `${gb.toFixed(decimals)} GB`;
    },

    /**
     * Format percentage
     */
    formatPercent(value, decimals = 1) {
        return `${value.toFixed(decimals)}%`;
    },

    /**
     * Format response time
     */
    formatResponseTime(ms) {
        if (ms < 1000) {
            return `${ms.toFixed(0)}ms`;
        }
        return `${(ms / 1000).toFixed(2)}s`;
    },

    /**
     * Format timestamp to relative time
     */
    formatRelativeTime(timestamp) {
        const now = new Date();
        const date = new Date(timestamp);
        const seconds = Math.floor((now - date) / 1000);

        if (seconds < 60) return `${seconds}s ago`;
        if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
        if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
        if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`;
        return date.toLocaleDateString();
    },

    /**
     * Format cost
     */
    formatCost(amount, currency = '$') {
        return `${currency}${amount.toFixed(4)}`;
    },

    /**
     * Get status color class
     */
    getStatusColor(status) {
        const statusColors = {
            'healthy': 'text-green-500',
            'degraded': 'text-yellow-500',
            'unhealthy': 'text-red-500',
            'ok': 'text-green-500',
            'error': 'text-red-500',
            'warning': 'text-yellow-500',
            'active': 'text-green-500',
            'inactive': 'text-gray-500'
        };
        return statusColors[status.toLowerCase()] || 'text-gray-500';
    },

    /**
     * Get model tier badge
     */
    getModelTierBadge(tier) {
        const tierColors = {
            'basic': { bg: 'bg-gray-100', text: 'text-gray-800' },
            'good': { bg: 'bg-blue-100', text: 'text-blue-800' },
            'high': { bg: 'bg-purple-100', text: 'text-purple-800' },
            'exceptional': { bg: 'bg-yellow-100', text: 'text-yellow-800' }
        };
        const colors = tierColors[tier] || tierColors['basic'];
        return `<span class="px-2 py-1 text-xs font-medium rounded ${colors.bg} ${colors.text}">${tier}</span>`;
    },

    /**
     * Format model capabilities
     */
    formatCapabilities(capabilities) {
        if (!capabilities || capabilities.length === 0) return 'None';
        return capabilities.map(cap => 
            `<span class="px-2 py-1 text-xs bg-blue-50 text-blue-700 rounded mr-1">${cap}</span>`
        ).join('');
    },

    /**
     * Calculate memory pressure level
     */
    getMemoryPressureLevel(pressure) {
        if (pressure < 0.5) return { level: 'low', color: 'text-green-500' };
        if (pressure < 0.7) return { level: 'moderate', color: 'text-yellow-500' };
        if (pressure < 0.9) return { level: 'high', color: 'text-orange-500' };
        return { level: 'critical', color: 'text-red-500' };
    },

    /**
     * Format duration
     */
    formatDuration(ms) {
        const seconds = Math.floor(ms / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        
        if (hours > 0) {
            return `${hours}h ${minutes % 60}m`;
        }
        if (minutes > 0) {
            return `${minutes}m ${seconds % 60}s`;
        }
        return `${seconds}s`;
    }
};

export default formatters;