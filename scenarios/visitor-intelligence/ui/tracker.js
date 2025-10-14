import { initIframeBridgeChild } from '@vrooli/iframe-bridge/child';

(function bootstrapVisitorIntelligenceBridge() {
    if (typeof window === 'undefined' || window.parent === window || window.__visitorIntelligenceBridgeInitialized) {
        return;
    }

    let parentOrigin;
    try {
        if (document.referrer) {
            parentOrigin = new URL(document.referrer).origin;
        }
    } catch (error) {
        console.warn('[VisitorIntelligence] Unable to determine parent origin for iframe bridge', error);
    }

    initIframeBridgeChild({ parentOrigin, appId: 'visitor-intelligence' });
    window.__visitorIntelligenceBridgeInitialized = true;
})();

/**
 * Vrooli Visitor Intelligence Tracker
 * Privacy-first visitor identification and behavioral tracking
 */
(function(window, document) {
    'use strict';

    // Configuration
    const config = {
        apiEndpoint: '/api/v1/visitor/track',
        sessionTimeout: 30 * 60 * 1000, // 30 minutes
        heartbeatInterval: 5 * 60 * 1000, // 5 minutes
        maxRetries: 3,
        retryDelay: 1000,
        respectDNT: true
    };

    // Check for Do Not Track
    if (config.respectDNT && (navigator.doNotTrack === '1' || window.doNotTrack === '1')) {
        return;
    }

    // Visitor Intelligence class
    class VisitorIntelligence {
        constructor(scenario) {
            this.scenario = scenario;
            this.sessionId = this.generateSessionId();
            this.fingerprint = null;
            this.visitorId = null;
            this.sessionStartTime = Date.now();
            this.lastActivity = Date.now();
            this.pageViews = 0;
            this.events = [];
            
            // Initialize
            this.init();
        }

        init() {
            // Generate fingerprint
            this.generateFingerprint().then(fingerprint => {
                this.fingerprint = fingerprint;
                this.trackPageView();
                this.setupEventListeners();
                this.startHeartbeat();
            });
        }

        // Generate browser fingerprint using multiple techniques
        async generateFingerprint() {
            const components = [];

            // Screen properties
            components.push(screen.width + 'x' + screen.height);
            components.push(screen.colorDepth);
            components.push(screen.pixelDepth);

            // Browser properties
            components.push(navigator.userAgent);
            components.push(navigator.language);
            components.push(navigator.platform);
            components.push(navigator.cookieEnabled);
            components.push(navigator.onLine);

            // Timezone
            try {
                components.push(Intl.DateTimeFormat().resolvedOptions().timeZone);
            } catch (e) {
                components.push(new Date().getTimezoneOffset());
            }

            // Canvas fingerprinting (simplified)
            try {
                const canvas = document.createElement('canvas');
                const ctx = canvas.getContext('2d');
                ctx.textBaseline = 'top';
                ctx.font = '14px Arial';
                ctx.fillText('Vrooli visitor tracking', 2, 2);
                components.push(canvas.toDataURL());
            } catch (e) {
                components.push('canvas-blocked');
            }

            // WebGL fingerprinting (basic)
            try {
                const canvas = document.createElement('canvas');
                const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
                if (gl) {
                    const debugInfo = gl.getExtension('WEBGL_debug_renderer_info');
                    components.push(gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL));
                    components.push(gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL));
                }
            } catch (e) {
                components.push('webgl-blocked');
            }

            // Font detection (simplified list)
            const testFonts = ['Arial', 'Times New Roman', 'Courier New', 'Helvetica', 'Georgia'];
            const availableFonts = testFonts.filter(font => this.isFontAvailable(font));
            components.push(availableFonts.join(','));

            // Audio context fingerprinting (basic)
            try {
                const audioCtx = new (window.AudioContext || window.webkitAudioContext)();
                const oscillator = audioCtx.createOscillator();
                oscillator.type = 'triangle';
                oscillator.frequency.value = 1000;
                const compressor = audioCtx.createDynamicsCompressor();
                oscillator.connect(compressor);
                compressor.connect(audioCtx.destination);
                components.push(compressor.threshold.value.toString());
                audioCtx.close();
            } catch (e) {
                components.push('audio-blocked');
            }

            // Generate hash of components
            return this.hashString(components.join('|'));
        }

        // Simple font availability check
        isFontAvailable(fontName) {
            const testString = 'mmmmmmmmmmlli';
            const testSize = '72px';
            const baselineFont = 'monospace';

            const canvas = document.createElement('canvas');
            const ctx = canvas.getContext('2d');

            ctx.font = testSize + ' ' + baselineFont;
            const baselineWidth = ctx.measureText(testString).width;

            ctx.font = testSize + ' ' + fontName + ', ' + baselineFont;
            const fontWidth = ctx.measureText(testString).width;

            return fontWidth !== baselineWidth;
        }

        // Simple hash function
        hashString(str) {
            let hash = 0;
            for (let i = 0; i < str.length; i++) {
                const char = str.charCodeAt(i);
                hash = ((hash << 5) - hash) + char;
                hash = hash & hash; // Convert to 32-bit integer
            }
            return Math.abs(hash).toString(36);
        }

        // Generate session ID
        generateSessionId() {
            return Date.now().toString(36) + Math.random().toString(36).substr(2);
        }

        // Setup event listeners for behavioral tracking
        setupEventListeners() {
            // Page visibility
            document.addEventListener('visibilitychange', () => {
                if (document.visibilityState === 'visible') {
                    this.trackEvent('page_focus');
                } else {
                    this.trackEvent('page_blur');
                }
            });

            // Scroll tracking (throttled)
            let scrollTimeout;
            window.addEventListener('scroll', () => {
                if (scrollTimeout) return;
                scrollTimeout = setTimeout(() => {
                    const scrollPercentage = Math.round(
                        (window.scrollY / (document.documentElement.scrollHeight - window.innerHeight)) * 100
                    );
                    this.trackEvent('scroll', { percentage: scrollPercentage });
                    scrollTimeout = null;
                }, 1000);
            });

            // Click tracking
            document.addEventListener('click', (e) => {
                const target = e.target;
                const tagName = target.tagName.toLowerCase();
                const properties = {
                    tag: tagName,
                    x: e.clientX,
                    y: e.clientY
                };

                // Add additional context for important elements
                if (tagName === 'a') {
                    properties.href = target.href;
                } else if (tagName === 'button') {
                    properties.text = target.textContent.trim().substr(0, 50);
                } else if (target.id) {
                    properties.id = target.id;
                } else if (target.className) {
                    properties.class = target.className;
                }

                this.trackEvent('click', properties);
            });

            // Form interactions
            document.addEventListener('focusin', (e) => {
                if (e.target.tagName.toLowerCase() === 'input' || e.target.tagName.toLowerCase() === 'textarea') {
                    this.trackEvent('form_focus', {
                        type: e.target.type || 'textarea',
                        name: e.target.name
                    });
                }
            });

            // Before unload (page exit)
            window.addEventListener('beforeunload', () => {
                this.trackEvent('page_exit', {
                    duration: Date.now() - this.sessionStartTime,
                    pageViews: this.pageViews
                });
                this.flush();
            });

            // Update last activity timestamp
            ['click', 'mousemove', 'keydown', 'scroll', 'touchstart'].forEach(eventType => {
                document.addEventListener(eventType, () => {
                    this.lastActivity = Date.now();
                }, { passive: true });
            });
        }

        // Track page view
        trackPageView() {
            this.pageViews++;
            this.trackEvent('pageview', {
                url: window.location.href,
                title: document.title,
                referrer: document.referrer,
                pageViews: this.pageViews
            });
        }

        // Track custom event
        trackEvent(eventType, properties = {}) {
            if (!this.fingerprint) {
                return; // Not initialized yet
            }

            const event = {
                fingerprint: this.fingerprint,
                session_id: this.sessionId,
                scenario: this.scenario,
                event_type: eventType,
                page_url: window.location.href,
                timestamp: new Date().toISOString(),
                properties: {
                    ...properties,
                    user_agent: navigator.userAgent,
                    viewport: window.innerWidth + 'x' + window.innerHeight,
                    screen: screen.width + 'x' + screen.height
                }
            };

            this.events.push(event);
            this.sendEvent(event);
        }

        // Send event to API with retry logic
        async sendEvent(event, retryCount = 0) {
            try {
                const response = await fetch(config.apiEndpoint, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify(event)
                });

                if (response.ok) {
                    const result = await response.json();
                    if (result.visitor_id && !this.visitorId) {
                        this.visitorId = result.visitor_id;
                    }
                } else if (response.status >= 500 && retryCount < config.maxRetries) {
                    // Retry on server errors
                    setTimeout(() => {
                        this.sendEvent(event, retryCount + 1);
                    }, config.retryDelay * Math.pow(2, retryCount));
                }
            } catch (error) {
                // Retry on network errors
                if (retryCount < config.maxRetries) {
                    setTimeout(() => {
                        this.sendEvent(event, retryCount + 1);
                    }, config.retryDelay * Math.pow(2, retryCount));
                }
            }
        }

        // Send heartbeat to maintain session
        startHeartbeat() {
            setInterval(() => {
                // Only send heartbeat if user was recently active
                if (Date.now() - this.lastActivity < config.sessionTimeout) {
                    this.trackEvent('heartbeat', {
                        duration: Date.now() - this.sessionStartTime
                    });
                }
            }, config.heartbeatInterval);
        }

        // Flush any pending events
        flush() {
            // Use sendBeacon if available for more reliable delivery
            if (navigator.sendBeacon && this.events.length > 0) {
                const payload = JSON.stringify({
                    events: this.events.slice(-10) // Send last 10 events
                });
                navigator.sendBeacon(config.apiEndpoint + '/batch', payload);
            }
        }

        // Public API
        identify(properties) {
            this.trackEvent('identify', properties);
        }

        track(eventType, properties) {
            this.trackEvent(eventType, properties);
        }

        getVisitorId() {
            return this.visitorId || this.fingerprint;
        }
    }

    // Auto-initialize from script tag data attributes
    function autoInit() {
        const scripts = document.querySelectorAll('script[data-scenario]');
        scripts.forEach(script => {
            const scenario = script.getAttribute('data-scenario');
            if (scenario) {
                window.VisitorIntelligence = new VisitorIntelligence(scenario);
            }
        });
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', autoInit);
    } else {
        autoInit();
    }

    // Expose API globally
    window.VisitorIntelligence = VisitorIntelligence;

})(window, document);
