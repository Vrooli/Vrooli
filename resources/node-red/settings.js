module.exports = {
    // Flow file settings
    flowFile: 'flows.json',
    flowFilePretty: true,
    
    // User directory
    userDir: '/data/',
    
    // Node-RED settings
    uiPort: process.env.PORT || 1880,
    
    // Logging
    logging: {
        console: {
            level: "info",
            metrics: false,
            audit: false
        }
    },
    
    // Editor theme
    editorTheme: {
        theme: "dark",
        projects: {
            enabled: false
        }
    },
    
    // Function node settings
    functionGlobalContext: {
        // Add global libraries here
        // os: require('os'),
    },
    
    // Allow external npm modules in function nodes
    functionExternalModules: true,
    
    // Debug settings
    debugMaxLength: 1000,
    
    // Exec node settings
    execMaxBufferSize: 10000000, // 10MB
    
    // HTTP request timeout
    httpRequestTimeout: 120000, // 2 minutes
}