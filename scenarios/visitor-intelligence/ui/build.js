const fs = require('fs');
const path = require('path');

// Simple build script for the tracking library
async function build() {
    const srcFile = path.join(__dirname, 'tracker.js');
    const distFile = path.join(__dirname, 'tracker.min.js');
    
    console.log('Building visitor intelligence tracker...');
    
    try {
        // Read source
        const source = fs.readFileSync(srcFile, 'utf8');
        
        // Try to use terser if available, otherwise just copy
        let minified;
        try {
            const { minify } = require('terser');
            const result = await minify(source, {
                compress: {
                    dead_code: true,
                    drop_console: false, // Keep console for debugging
                    drop_debugger: true,
                    keep_fargs: false,
                    unused: true
                },
                mangle: {
                    toplevel: true,
                    reserved: ['VisitorIntelligence'] // Keep main class name
                },
                format: {
                    comments: false,
                    max_line_len: 200
                }
            });
            minified = result.code;
            console.log(`Minified: ${source.length} → ${minified.length} bytes (${Math.round((1 - minified.length / source.length) * 100)}% reduction)`);
        } catch (e) {
            console.log('Terser not available, copying source as-is');
            minified = source;
        }
        
        // Write minified version
        fs.writeFileSync(distFile, minified);
        console.log(`Built: ${distFile}`);
        
        // Create a version with embedded config for self-hosting
        const embeddedConfig = minified.replace(
            'apiEndpoint: \'/api/v1/visitor/track\'',
            'apiEndpoint: window.location.origin + \'/api/v1/visitor/track\''
        );
        
        const embedFile = path.join(__dirname, 'tracker.embed.js');
        fs.writeFileSync(embedFile, embeddedConfig);
        console.log(`Built embedded version: ${embedFile}`);
        
    } catch (error) {
        console.error('Build failed:', error);
        process.exit(1);
    }
}

// Run build
build().catch(console.error);