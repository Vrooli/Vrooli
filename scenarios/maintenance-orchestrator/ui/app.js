const iframeParentOrigin = (() => {
    try {
        return document.referrer ? new URL(document.referrer).origin : undefined;
    } catch (error) {
        return undefined;
    }
})();

if (typeof window !== 'undefined' && window.parent !== window && typeof window.initIframeBridgeChild === 'function') {
    window.initIframeBridgeChild({ appId: 'maintenance-orchestrator', parentOrigin: iframeParentOrigin });
}
