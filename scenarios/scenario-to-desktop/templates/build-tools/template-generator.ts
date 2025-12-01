#!/usr/bin/env node

/**
 * Desktop Application Template Generator
 * Part of scenario-to-desktop system
 * 
 * This script generates desktop applications from templates and configuration.
 */

import * as fs from 'fs/promises';
import * as path from 'path';
import { execSync } from 'child_process';

interface DesktopConfig {
    // Application identity (matching Go JSON tags)
    app_name: string;
    app_display_name: string;
    app_description: string;
    version: string;
    author: string;
    author_email?: string;
    homepage?: string;
    license: string;
    app_id: string;
    app_url?: string;

    // Server configuration
    server_type: 'node' | 'static' | 'external' | 'executable';
    server_port?: number;
    server_path: string;
    api_endpoint: string;
    scenario_dist_path?: string;
    deployment_mode?: 'external-server' | 'cloud-api' | 'bundled';
    scenario_name?: string;
    auto_manage_vrooli?: boolean;
    auto_manage_tier1?: boolean; // Legacy alias from early builds
    proxy_url?: string;
    vrooli_binary_path?: string;

    // Template configuration
    framework: 'electron' | 'tauri' | 'neutralino';
    template_type: 'basic' | 'universal' | 'advanced' | 'kiosk' | 'multi_window';

    // Features
    features: {
        splash?: boolean;
        systemTray?: boolean;
        autoUpdater?: boolean;
        devTools?: boolean;
        singleInstance?: boolean;
        [key: string]: any;
    };

    // Window configuration
    window: {
        width?: number;
        height?: number;
        background?: string;
        [key: string]: any;
    };

    // Platform targets
    platforms: ('win' | 'mac' | 'linux')[];

    // Output configuration
    output_path: string;
    
    // Styling (for splash, etc.)
    styling?: {
        splashBackgroundStart?: string;
        splashBackgroundEnd?: string;
        splashTextColor?: string;
        splashAccentColor?: string;
        [key: string]: any;
    };
}

interface TemplateFile {
    sourcePath: string;
    targetPath: string;
    isTemplate: boolean;
}

class DesktopTemplateGenerator {
    private config: DesktopConfig;
    private templateBasePath: string;
    private outputPath: string;
    
    constructor(config: DesktopConfig) {
        this.config = config;
        // SECURITY: __dirname is controlled by the build system (not user input).
        // This path traversal is safe as it navigates from build-tools/dist/ to templates/.
        // The path.resolve() ensures this becomes an absolute path without traversal.
        this.templateBasePath = path.resolve(__dirname, '../../');

        // Determine output path - default to standard platforms/electron/ location
        let outputPath: string;

        if (!config.output_path || config.output_path === '') {
            // Use standard location: <vrooli-root>/scenarios/<scenario-name>/platforms/electron/
            const vrooliRoot = process.env.VROOLI_ROOT || path.resolve(__dirname, '../../../../../');
            outputPath = path.join(
                vrooliRoot,
                'scenarios',
                config.app_name,
                'platforms',
                'electron'
            );
            console.log(`📁 Using standard location: ${outputPath}`);
        } else {
            // Validate and sanitize user-provided output path to prevent path traversal
            const resolvedPath = path.resolve(config.output_path);
            const normalizedPath = path.normalize(resolvedPath);

            // Ensure the path doesn't contain traversal patterns after normalization
            if (normalizedPath.includes('..')) {
                throw new Error('Invalid output path: path traversal detected');
            }

            outputPath = normalizedPath;
        }

        this.outputPath = outputPath;
    }
    
    async generate(): Promise<void> {
        console.log(`🚀 Generating desktop application: ${this.config.app_display_name}`);
        console.log(`📁 Output directory: ${this.outputPath}`);
        console.log(`🎨 Template type: ${this.config.template_type}`);
        console.log(`⚡ Framework: ${this.config.framework}`);
        
        try {
            // Load template configuration
            const templateConfig = await this.loadTemplateConfig();
            
            // Create output directory
            await this.ensureOutputDirectory();
            
            // Copy and process template files
            await this.processTemplateFiles(templateConfig);
            
            // Generate additional files based on template
            await this.generateAdditionalFiles(templateConfig);
            
            // Create assets directory and icons
            await this.setupAssets();

            // Copy UI dist files if scenario_dist_path is provided
            await this.copyUIDistFiles();

            // Install dependencies
            await this.installDependencies();
            
            console.log(`✅ Desktop application generated successfully!`);
            console.log(`📝 Next steps:`);
            console.log(`   cd ${this.outputPath}`);
            console.log(`   npm run dev`);
            
        } catch (error) {
            console.error(`❌ Failed to generate desktop application:`, error);
            throw error;
        }
    }
    
    private async loadTemplateConfig(): Promise<any> {
        // Map template types to actual filenames
        const templateFiles: Record<string, string> = {
            'basic': 'universal-app.json',        // 'basic' is an alias for 'universal'
            'universal': 'universal-app.json',    // Default universal template
            'advanced': 'advanced-app.json',
            'multi_window': 'multi-window.json',
            'kiosk': 'kiosk-mode.json'
        };

        const filename = templateFiles[this.config.template_type];
        if (!filename) {
            throw new Error(`Invalid template type: ${this.config.template_type}`);
        }

        const templateConfigPath = path.join(
            this.templateBasePath,
            'advanced',
            filename
        );

        try {
            const configContent = await fs.readFile(templateConfigPath, 'utf-8');
            return JSON.parse(configContent);
        } catch (error) {
            throw new Error(`Failed to load template configuration: ${templateConfigPath}`);
        }
    }
    
    private async ensureOutputDirectory(): Promise<void> {
        try {
            await fs.access(this.outputPath);
            console.log(`⚠️  Output directory exists: ${this.outputPath}`);
        } catch {
            await fs.mkdir(this.outputPath, { recursive: true });
            console.log(`📁 Created output directory: ${this.outputPath}`);
        }
    }
    
    private async processTemplateFiles(templateConfig: any): Promise<void> {
        const vanillaPath = path.join(this.templateBasePath, 'vanilla');
        const files = await this.getTemplateFiles(vanillaPath);
        
        for (const file of files) {
            await this.processTemplateFile(file, templateConfig);
        }
    }
    
    private async getTemplateFiles(dirPath: string): Promise<TemplateFile[]> {
        const files: TemplateFile[] = [];
        const entries = await fs.readdir(dirPath, { withFileTypes: true });
        
        for (const entry of entries) {
            const fullPath = path.join(dirPath, entry.name);
            
            if (entry.isDirectory()) {
                const subFiles = await this.getTemplateFiles(fullPath);
                files.push(...subFiles);
            } else {
                const relativePath = path.relative(path.join(this.templateBasePath, 'vanilla'), fullPath);
                const targetPath = entry.name.endsWith('.template') 
                    ? relativePath.replace('.template', '')
                    : relativePath;
                
                // Determine if file should be processed as template
                // Always process: .template files, and common code/config files that may contain variables
                const templateExtensions = ['.ts', '.js', '.json', '.html', '.md', '.yml', '.yaml', '.xml', '.sh'];
                const isTemplateByExtension = templateExtensions.some(ext => entry.name.endsWith(ext));
                const isTemplateByName = entry.name.includes('.template') || entry.name.includes('{{');

                files.push({
                    sourcePath: fullPath,
                    targetPath,
                    isTemplate: isTemplateByName || isTemplateByExtension
                });
            }
        }
        
        return files;
    }
    
    private async processTemplateFile(file: TemplateFile, templateConfig: any): Promise<void> {
        const targetPath = path.join(this.outputPath, file.targetPath);
        const targetDir = path.dirname(targetPath);
        
        // Ensure target directory exists
        await fs.mkdir(targetDir, { recursive: true });
        
        let content = await fs.readFile(file.sourcePath, 'utf-8');
        
        if (file.isTemplate) {
            content = await this.processTemplateContent(content, templateConfig);
        }
        
        await fs.writeFile(targetPath, content);
        console.log(`📄 Processed: ${file.targetPath}`);
    }
    
    private async processTemplateContent(content: string, templateConfig: any): Promise<string> {
        const variables = this.buildTemplateVariables(templateConfig);
        
        let processedContent = content;
        
        // Replace template variables
        for (const [key, value] of Object.entries(variables)) {
            const regex = new RegExp(`{{${key}}}`, 'g');
            processedContent = processedContent.replace(regex, String(value));
        }
        
        return processedContent;
    }
    
    private buildTemplateVariables(templateConfig: any): Record<string, any> {
        const currentYear = new Date().getFullYear();
        
        const variables: Record<string, any> = {
            // Basic app info
            APP_NAME: this.config.app_name,
            APP_DISPLAY_NAME: this.config.app_display_name,
            APP_DESCRIPTION: this.config.app_description,
            VERSION: this.config.version,
            AUTHOR: this.config.author,
            AUTHOR_EMAIL: this.config.author_email || 'noreply@vrooli.com',
            HOMEPAGE: this.config.homepage || this.config.app_url || 'https://vrooli.com',
            LICENSE: this.config.license,
            APP_ID: this.config.app_id,
            APP_URL: this.config.app_url || '',
            YEAR: currentYear,
            
            // Server config
            SERVER_TYPE: this.config.server_type,
            SERVER_PORT: this.config.server_port || 3000,
            SERVER_PATH: this.config.server_path,
            API_ENDPOINT: this.config.api_endpoint,
            SCENARIO_DIST_PATH: this.config.scenario_dist_path || '../ui/dist',
            DEPLOYMENT_MODE: this.config.deployment_mode || 'external-server',
            SCENARIO_NAME: this.config.scenario_name || this.config.app_name,
            AUTO_MANAGE_VROOLI: this.config.auto_manage_vrooli === true || this.config.auto_manage_tier1 === true,
            VROOLI_BINARY_PATH: this.config.vrooli_binary_path || 'vrooli',

            // Bundled runtime metadata
            BUNDLED_RUNTIME_SUPPORTED: this.config.deployment_mode === 'bundled',
            BUNDLED_RUNTIME_ROOT: this.config.bundle_runtime_root || 'bundle',
            BUNDLED_RUNTIME_IPC_HOST: this.config.bundle_ipc?.host || '127.0.0.1',
            BUNDLED_RUNTIME_IPC_PORT: this.config.bundle_ipc?.port || 47710,
            BUNDLED_RUNTIME_TOKEN_PATH: this.config.bundle_ipc?.auth_token_path || 'runtime/auth-token',
            BUNDLED_RUNTIME_UI_SERVICE: this.config.bundle_ui_service_id || '',
            BUNDLED_RUNTIME_UI_PORT_NAME: this.config.bundle_ui_port_name || 'http',
            BUNDLED_RUNTIME_TELEMETRY_UPLOAD_URL: this.config.bundle_telemetry_upload_url || '',
            
            // Window config (use optional chaining for safety)
            WINDOW_WIDTH: this.config.window?.width || 1200,
            WINDOW_HEIGHT: this.config.window?.height || 800,
            WINDOW_BACKGROUND: this.config.window?.background || '#f5f5f5',

            // Features (boolean values) (use optional chaining for safety)
            ENABLE_SPLASH: this.config.features?.splash ?? true,
            ENABLE_MENU: this.config.features?.menu ?? true,
            ENABLE_SYSTEM_TRAY: this.config.features?.systemTray ?? false,
            ENABLE_AUTO_UPDATER: this.config.features?.autoUpdater ?? true,
            ENABLE_SINGLE_INSTANCE: this.config.features?.singleInstance ?? true,
            ENABLE_DEV_TOOLS: this.config.features?.devTools ?? true,
            
            // Styling
            SPLASH_BACKGROUND_START: this.config.styling?.splashBackgroundStart || '#4a90e2',
            SPLASH_BACKGROUND_END: this.config.styling?.splashBackgroundEnd || '#357abd',
            SPLASH_TEXT_COLOR: this.config.styling?.splashTextColor || '#ffffff',
            SPLASH_ACCENT_COLOR: this.config.styling?.splashAccentColor || '#64b5f6',
            APP_ICON_PATH: 'assets/icon.png',
            
            // Distribution
            PUBLISHER_NAME: this.config.author,
            PUBLISH_CONFIG: JSON.stringify(this.getPublishConfig(), null, 2),
        };
        
        // Merge template-specific variables
        if (templateConfig.template_variables) {
            Object.assign(variables, templateConfig.template_variables);
        }
        
        return variables;
    }
    
    private getPublishConfig(): any {
        // Default publish configuration - can be customized based on deployment needs
        return {
            provider: "github",
            owner: "your-organization",
            repo: `${this.config.app_name}-desktop`
        };
    }
    
    private async generateAdditionalFiles(templateConfig: any): Promise<void> {
        // Generate README.md
        await this.generateReadme();
        
        // Generate .gitignore
        await this.generateGitIgnore();
        
        // Generate build scripts
        await this.generateBuildScripts(templateConfig);
        
        // Create src directory structure if needed
        await this.createSourceStructure();
    }
    
    private async generateReadme(): Promise<void> {
        const readme = `# ${this.config.app_display_name}

${this.config.app_description}

## 🚀 Quick Start

### Development
\`\`\`bash
# Install dependencies
npm install

# Start in development mode
npm run dev

# Build TypeScript
npm run build
\`\`\`

### Building for Distribution
\`\`\`bash
# Build for current platform
npm run dist

# Build for all platforms
npm run dist:all
\`\`\`

## 📦 Generated Application Details

- **Framework**: ${this.config.framework}
- **Template**: ${this.config.template_type}
- **Server Type**: ${this.config.server_type}
- **Target Platforms**: ${this.config.platforms.join(', ')}

## 🛠️ Development

This desktop application was generated using **scenario-to-desktop**.

### Project Structure
\`\`\`
${this.config.app_name}-desktop/
├── src/
│   ├── main.ts          # Main Electron process
│   ├── preload.ts       # Renderer bridge
│   └── splash.html      # Splash screen
├── assets/              # Icons and resources
├── dist/                # Built TypeScript
└── dist-electron/       # Final application builds
\`\`\`

### Customization

- **Main Process**: Edit \`src/main.ts\` for window management and native features
- **Preload Script**: Edit \`src/preload.ts\` for secure renderer APIs
- **Splash Screen**: Edit \`src/splash.html\` for startup branding
- **Build Config**: Edit \`package.json\` build section for distribution settings

## 📝 License

${this.config.license}

---

Generated by [scenario-to-desktop](https://github.com/vrooli/vrooli) v1.0.0
`;

        await fs.writeFile(path.join(this.outputPath, 'README.md'), readme);
        console.log('📄 Generated: README.md');
    }
    
    private async generateGitIgnore(): Promise<void> {
        const gitignore = `# Dependencies
node_modules/

# Build outputs
dist/
dist-electron/

# Logs
*.log
npm-debug.log*

# OS generated files
.DS_Store
Thumbs.db

# IDE files
.vscode/
.idea/
*.swp
*.swo

# Electron builder
app-update.yml
dev-app-update.yml

# Environment variables
.env
.env.local

# Temporary files
*.tmp
*.temp
`;

        await fs.writeFile(path.join(this.outputPath, '.gitignore'), gitignore);
        console.log('📄 Generated: .gitignore');
    }
    
    private async generateBuildScripts(templateConfig: any): Promise<void> {
        // Create scripts directory
        const scriptsDir = path.join(this.outputPath, 'scripts');
        await fs.mkdir(scriptsDir, { recursive: true });
        
        // Generate notarization script for macOS
        const notarizeScript = `const { notarize } = require('@electron/notarize');

exports.default = async function notarizing(context) {
    const { electronPlatformName, appOutDir } = context;
    if (electronPlatformName !== 'darwin') {
        return;
    }

    const appName = context.packager.appInfo.productFilename;

    return await notarize({
        appBundleId: '${this.config.app_id}',
        appPath: \`\${appOutDir}/\${appName}.app\`,
        appleId: process.env.APPLE_ID,
        appleIdPassword: process.env.APPLE_ID_PASSWORD,
        teamId: process.env.APPLE_TEAM_ID,
    });
};`;

        await fs.writeFile(path.join(scriptsDir, 'notarize.js'), notarizeScript);
        console.log('📄 Generated: scripts/notarize.js');
    }
    
    private async createSourceStructure(): Promise<void> {
        const srcDir = path.join(this.outputPath, 'src');
        await fs.mkdir(srcDir, { recursive: true });
        
        // Move template files to src/ if they exist in root
        const filesToMove = ['main.ts', 'preload.ts', 'splash.html'];
        
        for (const fileName of filesToMove) {
            const rootPath = path.join(this.outputPath, fileName);
            const srcPath = path.join(srcDir, fileName);
            
            try {
                await fs.access(rootPath);
                await fs.rename(rootPath, srcPath);
                console.log(`📁 Moved ${fileName} to src/`);
            } catch {
                // File doesn't exist or already in src, skip
            }
        }
    }
    
    private async setupAssets(): Promise<void> {
        // Validate and normalize output path to prevent path traversal
        const normalizedOutput = path.normalize(this.outputPath);
        const assetsDir = path.join(normalizedOutput, 'assets');

        // Ensure assetsDir is within the output path
        if (!assetsDir.startsWith(normalizedOutput)) {
            throw new Error('Invalid output path: potential path traversal detected');
        }

        await fs.mkdir(assetsDir, { recursive: true });

        // Create placeholder icon files (fixed size list prevents injection)
        const iconSizes = [16, 32, 48, 128, 256, 512, 1024];

        for (const size of iconSizes) {
            const iconData = this.generatePlaceholderIcon(size);
            const iconPath = path.join(assetsDir, `icon-${size}x${size}.png`);

            // Verify the icon path is still within assetsDir
            if (!iconPath.startsWith(assetsDir)) {
                throw new Error('Invalid icon path detected');
            }

            await fs.writeFile(iconPath, iconData);
        }

        // Create main icon.png (512x512) - electron-builder will convert to ICO/ICNS as needed
        // We don't generate .ico or .icns placeholders because electron-builder requires
        // specific multi-resolution formats that are complex to generate properly
        const mainIconPath = path.join(assetsDir, 'icon.png');
        if (!mainIconPath.startsWith(assetsDir)) {
            throw new Error('Invalid platform icon path detected');
        }
        await fs.writeFile(mainIconPath, this.generatePlaceholderIcon(512));
        
        // Create license file
        const licenseContent = `${this.config.app_display_name}
Copyright (c) ${new Date().getFullYear()} ${this.config.author}

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.`;

        await fs.writeFile(path.join(assetsDir, 'license.txt'), licenseContent);
        
        console.log('🎨 Generated assets and icons');
    }
    
    private generatePlaceholderIcon(size: number): Buffer {
        // Generate a minimal valid 256x256 PNG (minimum size for electron-builder)
        // This creates a simple gray square that electron-builder can process and convert

        // For sizes < 256, we still generate 256x256 (electron-builder minimum)
        // For sizes >= 256, we generate the requested size
        const actualSize = Math.max(size, 256);

        // Create a simple PNG with a solid color
        // We'll create the smallest valid PNG possible for the requested dimensions
        const png = this.createMinimalPNG(actualSize, actualSize);
        return png;
    }

    private createMinimalPNG(width: number, height: number): Buffer {
        // Create a minimal PNG file with specified dimensions
        // This generates a solid gray image that's valid for electron-builder

        // PNG uses scanlines with a filter byte, so each row is (width * 4 + 1) bytes for RGBA
        const bytesPerPixel = 4; // RGBA
        const bytesPerRow = width * bytesPerPixel + 1; // +1 for filter byte
        const rawData = Buffer.alloc(height * bytesPerRow);

        // Fill with filter byte 0 (None) and gray pixels (0x80808080 = medium gray, opaque)
        for (let y = 0; y < height; y++) {
            const rowStart = y * bytesPerRow;
            rawData[rowStart] = 0; // Filter byte: None

            for (let x = 0; x < width; x++) {
                const pixelStart = rowStart + 1 + x * bytesPerPixel;
                rawData[pixelStart] = 0x80;     // R: medium gray
                rawData[pixelStart + 1] = 0x80; // G: medium gray
                rawData[pixelStart + 2] = 0x80; // B: medium gray
                rawData[pixelStart + 3] = 0xFF; // A: fully opaque
            }
        }

        // Compress the raw data using deflate (zlib)
        const zlib = require('zlib');
        const compressedData = zlib.deflateSync(rawData, { level: 9 });

        // Build PNG file structure
        const chunks: Buffer[] = [];

        // PNG signature
        chunks.push(Buffer.from([0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A]));

        // IHDR chunk
        chunks.push(this.createPNGChunk('IHDR', this.createIHDR(width, height)));

        // IDAT chunk (compressed image data)
        chunks.push(this.createPNGChunk('IDAT', compressedData));

        // IEND chunk
        chunks.push(this.createPNGChunk('IEND', Buffer.alloc(0)));

        return Buffer.concat(chunks);
    }

    private createIHDR(width: number, height: number): Buffer {
        const ihdr = Buffer.alloc(13);
        ihdr.writeUInt32BE(width, 0);       // Width
        ihdr.writeUInt32BE(height, 4);      // Height
        ihdr.writeUInt8(8, 8);              // Bit depth
        ihdr.writeUInt8(6, 9);              // Color type: RGBA
        ihdr.writeUInt8(0, 10);             // Compression method
        ihdr.writeUInt8(0, 11);             // Filter method
        ihdr.writeUInt8(0, 12);             // Interlace method
        return ihdr;
    }

    private createPNGChunk(type: string, data: Buffer): Buffer {
        const length = Buffer.alloc(4);
        length.writeUInt32BE(data.length, 0);

        const typeBuffer = Buffer.from(type, 'ascii');
        const crc = this.crc32(Buffer.concat([typeBuffer, data]));
        const crcBuffer = Buffer.alloc(4);
        crcBuffer.writeUInt32BE(crc >>> 0, 0);

        return Buffer.concat([length, typeBuffer, data, crcBuffer]);
    }

    private crc32(data: Buffer): number {
        let crc = 0xFFFFFFFF;
        for (let i = 0; i < data.length; i++) {
            crc = crc ^ data[i];
            for (let j = 0; j < 8; j++) {
                crc = (crc >>> 1) ^ (0xEDB88320 & -(crc & 1));
            }
        }
        return crc ^ 0xFFFFFFFF;
    }

    private async copyUIDistFiles(): Promise<void> {
        // Check if scenario_dist_path is provided and valid
        const distPath = this.config.scenario_dist_path;
        if (!distPath || distPath === '../ui/dist') {
            console.log('ℹ️  No UI dist path provided - desktop app will load from external server');
            return;
        }

        try {
            // Validate source path exists
            const stat = await fs.stat(distPath);
            if (!stat.isDirectory()) {
                console.log(`⚠️  UI dist path is not a directory: ${distPath}`);
                return;
            }

            // Create renderer directory in output
            const rendererDir = path.join(this.outputPath, 'renderer');
            await fs.mkdir(rendererDir, { recursive: true });

            // Copy all files from dist to renderer
            console.log(`📋 Copying UI files from ${distPath}...`);
            await this.copyDirectory(distPath, rendererDir);
            console.log(`✅ Copied UI dist files to renderer/`);

            // Update the template to reference the renderer directory
            // This happens automatically via the SCENARIO_DIST_PATH variable substitution

        } catch (error) {
            console.log(`⚠️  Failed to copy UI dist files: ${error}`);
            console.log('   Desktop app will attempt to load from external server instead');
        }
    }

    private async copyDirectory(source: string, destination: string): Promise<void> {
        await fs.mkdir(destination, { recursive: true });

        const entries = await fs.readdir(source, { withFileTypes: true });

        for (const entry of entries) {
            const sourcePath = path.join(source, entry.name);
            const destPath = path.join(destination, entry.name);

            if (entry.isDirectory()) {
                await this.copyDirectory(sourcePath, destPath);
            } else {
                await fs.copyFile(sourcePath, destPath);
            }
        }
    }

    private async installDependencies(): Promise<void> {
        if (process.env.SKIP_DESKTOP_DEPENDENCY_INSTALL === '1') {
            console.log('⏭️  Skipping dependency installation (SKIP_DESKTOP_DEPENDENCY_INSTALL=1)');
            return;
        }

        console.log('📦 Installing dependencies...');
        
        try {
            execSync('npm install', { 
                cwd: this.outputPath,
                stdio: 'inherit'
            });
            console.log('✅ Dependencies installed successfully');
        } catch (error) {
            console.warn('⚠️  Failed to install dependencies automatically');
            console.log('💡 Run "npm install" manually in the output directory');
        }
    }
}

// CLI interface
async function main() {
    const args = process.argv.slice(2);
    
    if (args.length === 0) {
        console.log('Usage: template-generator <config-file>');
        process.exit(1);
    }
    
    const configPath = path.resolve(args[0]);
    
    try {
        const configContent = await fs.readFile(configPath, 'utf-8');
        const config: DesktopConfig = JSON.parse(configContent);
        
        const generator = new DesktopTemplateGenerator(config);
        await generator.generate();
        
    } catch (error) {
        console.error('❌ Error:', error);
        process.exit(1);
    }
}

// Export for programmatic use
export { DesktopTemplateGenerator, DesktopConfig };

// Run if called directly
if (require.main === module) {
    main();
}
