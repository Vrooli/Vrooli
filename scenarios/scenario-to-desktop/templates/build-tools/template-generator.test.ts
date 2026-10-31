/**
 * Tests for DesktopTemplateGenerator update configuration behavior.
 *
 * These tests validate that UPDATE_PROVIDER correctly reflects whether
 * electron-builder will generate an app-update.yml file. When publish
 * config is null, UPDATE_PROVIDER must be "none" to prevent ENOENT errors
 * when the app tries to check for updates.
 *
 * Root cause: When generic provider has no URL, getPublishConfig() returns null
 * (no app-update.yml), but UPDATE_PROVIDER was still "generic", causing
 * electron-updater to fail with ENOENT.
 */

import { DesktopConfig, DesktopTemplateGenerator } from './template-generator';
import * as fs from 'fs-extra';
import * as path from 'path';
import * as os from 'os';
import { DesktopConfigValidator } from './config-validator';

describe('DesktopTemplateGenerator update configuration', () => {
    let tempDir: string;

    beforeEach(async () => {
        tempDir = await fs.mkdtemp(path.join(os.tmpdir(), 'template-gen-test-'));
    });

    afterEach(async () => {
        if (tempDir) {
            await fs.remove(tempDir);
        }
    });

    function createMinimalConfig(overrides: Partial<DesktopConfig> = {}): DesktopConfig {
        return {
            app_name: 'test-app',
            app_display_name: 'Test App',
            app_description: 'Test description',
            version: '1.0.0',
            author: 'Test Author',
            author_email: 'test@example.com',
            license: 'MIT',
            app_id: 'com.test.app',
            server_type: 'static',
            api_endpoint: 'http://localhost:3000',
            framework: 'electron',
            template_type: 'basic',
            output_path: tempDir,
            scenario_dist_path: '/fake/dist',
            ...overrides,
        } as DesktopConfig;
    }

    describe('explicit configuration precedence', () => {
        const templateDefaults = {
            WINDOW_WIDTH: 1200, WINDOW_HEIGHT: 800, WINDOW_BACKGROUND: '#ffffff',
            ENABLE_SPLASH: true, ENABLE_MENU: true, ENABLE_SYSTEM_TRAY: false,
            ENABLE_AUTO_UPDATER: true, ENABLE_SINGLE_INSTANCE: true, ENABLE_DEV_TOOLS: true,
            SPLASH_BACKGROUND_START: '#4a90e2', SPLASH_BACKGROUND_END: '#357abd',
            SPLASH_TEXT_COLOR: '#ffffff', SPLASH_ACCENT_COLOR: '#64b5f6',
            TEMPLATE_ONLY_CAPABILITY: 'retained',
        };

        it('preserves explicit window, feature false values and splash styling over template defaults', () => {
            const config = createMinimalConfig({
                window: { width: 1400, height: 900, background: '#2e2c29' },
                features: { splash: false, menu: false, systemTray: true, autoUpdater: false, singleInstance: false, devTools: false },
                styling: { splashBackgroundStart: '#1a1a1a', splashBackgroundEnd: '#2d2d2d', splashTextColor: '#eeeeee', splashAccentColor: '#00d4ff' },
            });
            const values = (new DesktopTemplateGenerator(config) as any).buildTemplateVariables({ template_variables: templateDefaults });
            expect(values).toMatchObject({
                WINDOW_WIDTH: 1400, WINDOW_HEIGHT: 900, WINDOW_BACKGROUND: '#2e2c29',
                ENABLE_SPLASH: false, ENABLE_MENU: false, ENABLE_SYSTEM_TRAY: true,
                ENABLE_AUTO_UPDATER: false, ENABLE_SINGLE_INSTANCE: false, ENABLE_DEV_TOOLS: false,
                SPLASH_BACKGROUND_START: '#1a1a1a', SPLASH_BACKGROUND_END: '#2d2d2d', SPLASH_TEXT_COLOR: '#eeeeee', SPLASH_ACCENT_COLOR: '#00d4ff',
                TEMPLATE_ONLY_CAPABILITY: 'retained',
            });
            expect(templateDefaults.WINDOW_WIDTH).toBe(1200);
        });

        it('retains selected-template defaults and template-only variables when caller omits those fields', () => {
            const config = createMinimalConfig();
            const values = (new DesktopTemplateGenerator(config) as any).buildTemplateVariables({ template_variables: {
                WINDOW_WIDTH: 1440, WINDOW_HEIGHT: 960, WINDOW_BACKGROUND: '#000000',
                ENABLE_MENU: false, ENABLE_SPLASH: false, ENABLE_DEV_TOOLS: false,
                ENABLE_SYSTEM_TRAY: true, SPLASH_BACKGROUND_START: '#112233', ENABLE_KIOSK_MODE: true,
                APP_ID: 'template-must-not-replace-user-identity',
            } });
            expect(values).toMatchObject({
                WINDOW_WIDTH: 1440, WINDOW_HEIGHT: 960, WINDOW_BACKGROUND: '#000000',
                ENABLE_MENU: false, ENABLE_SPLASH: false, ENABLE_DEV_TOOLS: false,
                ENABLE_SYSTEM_TRAY: true, SPLASH_BACKGROUND_START: '#112233', ENABLE_KIOSK_MODE: true,
                APP_ID: 'com.test.app',
            });
        });

        it('keeps safe owner fallbacks when neither caller nor template specifies fields', () => {
            const values = (new DesktopTemplateGenerator(createMinimalConfig()) as any).buildTemplateVariables({});
            expect(values).toMatchObject({ WINDOW_WIDTH: 1200, WINDOW_HEIGHT: 800, WINDOW_BACKGROUND: '#f5f5f5', ENABLE_SINGLE_INSTANCE: true, ENABLE_DEV_TOOLS: true });
        });
    });

    it('renders identical deterministic port configuration regardless of input insertion order', () => {
        const entries = [
            ['ui', { env_var: 'UI_PORT', port: 3000 }],
            ['api', { env_var: 'API_PORT', port: 8080 }],
            ['driver', { env_var: 'DRIVER_PORT', port: 4000 }],
        ];
        const render = (ports: DesktopConfig['ports']) => (new DesktopTemplateGenerator(createMinimalConfig({ ports })) as any).buildTemplateVariables({}).PORTS_CONFIG;
        const first = render(Object.fromEntries(entries));
        const second = render(Object.fromEntries([...entries].reverse()));
        expect(first).toBe(second);
        expect(first).toBe('{"api":{"envVar":"API_PORT","port":8080},"driver":{"envVar":"DRIVER_PORT","port":4000},"ui":{"envVar":"UI_PORT","port":3000}}');
    });

    describe('validator preserves caller-owned nested configuration', () => {
        function validatorConfig() {
            return { appName: 'test-app', appDisplayName: 'Test App', appDescription: 'Fixture', version: '1.0.0', author: 'Tester', appId: 'com.test.app', serverType: 'external', serverPath: 'https://example.test', apiEndpoint: 'https://example.test/api', framework: 'electron', templateType: 'basic', outputPath: 'output/app', platforms: ['linux'] };
        }

        it('copies explicit feature and window values without mutating even frozen nested inputs', () => {
            const config = { ...validatorConfig(), features: Object.freeze({ splash: false, systemTray: true }), window: Object.freeze({ width: 1400, height: 900, background: '#2e2c29' }) };
            const before = JSON.stringify(config);
            const result = new DesktopConfigValidator().validate(config);
            expect(result.isValid).toBe(true);
            expect(result.sanitizedConfig.features).toMatchObject({ splash: false, systemTray: true, autoUpdater: true });
            expect(result.sanitizedConfig.window).toEqual({ width: 1400, height: 900, background: '#2e2c29' });
            expect(result.sanitizedConfig.features).not.toBe(config.features);
            expect(result.sanitizedConfig.window).not.toBe(config.window);
            expect(JSON.stringify(config)).toBe(before);
        });

        it('fills omitted values, preserves template safety warnings and still rejects invalid dimensions', () => {
            const validator = new DesktopConfigValidator();
            const defaults = validator.validate(validatorConfig());
            expect(defaults.isValid).toBe(true);
            expect(defaults.sanitizedConfig.features).toMatchObject({ splash: true, systemTray: false });
            expect(defaults.sanitizedConfig.window).toEqual({ width: 1200, height: 800, background: '#f5f5f5' });
            const invalid = validator.validate({ ...validatorConfig(), window: { width: 100, height: 100 } });
            expect(invalid.isValid).toBe(false);
            expect(invalid.errors).toEqual(expect.arrayContaining([expect.objectContaining({ field: 'window.width', severity: 'error' }), expect.objectContaining({ field: 'window.height', severity: 'error' })]));
            const kiosk = validator.validate({ ...validatorConfig(), templateType: 'kiosk', features: { systemTray: true, devTools: true } });
            expect(kiosk.errors).toEqual(expect.arrayContaining([expect.objectContaining({ field: 'features.systemTray', severity: 'warning' }), expect.objectContaining({ field: 'features.devTools', severity: 'warning' })]));
        });
    });

    describe('getEffectiveUpdateProvider behavior', () => {
        it('returns "none" when generic provider has no URL configured', () => {
            const config = createMinimalConfig({
                update_config: {
                    provider: 'generic',
                    // No generic.url configured
                },
            });

            const generator = new DesktopTemplateGenerator(config);
            // Access private method via any cast for testing
            const effectiveProvider = (generator as any).getEffectiveUpdateProvider();

            expect(effectiveProvider).toBe('none');
        });

        it('returns "generic" when generic provider has URL configured', () => {
            const config = createMinimalConfig({
                update_config: {
                    provider: 'generic',
                    generic: {
                        url: 'https://updates.example.com/my-app',
                    },
                },
            });

            const generator = new DesktopTemplateGenerator(config);
            const effectiveProvider = (generator as any).getEffectiveUpdateProvider();

            expect(effectiveProvider).toBe('generic');
        });

        it('returns "github" when github provider is configured', () => {
            const config = createMinimalConfig({
                update_config: {
                    provider: 'github',
                    github: {
                        owner: 'myorg',
                        repo: 'myrepo',
                    },
                },
            });

            const generator = new DesktopTemplateGenerator(config);
            const effectiveProvider = (generator as any).getEffectiveUpdateProvider();

            expect(effectiveProvider).toBe('github');
        });

        it('returns "none" when provider is explicitly "none"', () => {
            const config = createMinimalConfig({
                update_config: {
                    provider: 'none',
                },
            });

            const generator = new DesktopTemplateGenerator(config);
            const effectiveProvider = (generator as any).getEffectiveUpdateProvider();

            expect(effectiveProvider).toBe('none');
        });

        it('returns "none" when no update_config at all (default to generic without URL)', () => {
            const config = createMinimalConfig({
                // No update_config
            });

            const generator = new DesktopTemplateGenerator(config);
            const effectiveProvider = (generator as any).getEffectiveUpdateProvider();

            expect(effectiveProvider).toBe('none');
        });
    });

    describe('getPublishConfig and getEffectiveUpdateProvider consistency', () => {
        it('both return appropriate values when URL is missing', () => {
            const config = createMinimalConfig({
                update_config: {
                    provider: 'generic',
                    // No URL
                },
            });

            const generator = new DesktopTemplateGenerator(config);
            const publishConfig = (generator as any).getPublishConfig();
            const effectiveProvider = (generator as any).getEffectiveUpdateProvider();

            // When publish config is null, provider must be "none"
            expect(publishConfig).toBeNull();
            expect(effectiveProvider).toBe('none');
        });

        it('both return appropriate values when URL is configured', () => {
            const config = createMinimalConfig({
                update_config: {
                    provider: 'generic',
                    channel: 'stable',
                    generic: {
                        url: 'https://updates.example.com',
                    },
                },
            });

            const generator = new DesktopTemplateGenerator(config);
            const publishConfig = (generator as any).getPublishConfig();
            const effectiveProvider = (generator as any).getEffectiveUpdateProvider();

            expect(publishConfig).not.toBeNull();
            expect(publishConfig.provider).toBe('generic');
            expect(publishConfig.url).toBe('https://updates.example.com/stable');
            expect(effectiveProvider).toBe('generic');
        });

        it('github provider always has publish config', () => {
            const config = createMinimalConfig({
                update_config: {
                    provider: 'github',
                    github: {
                        owner: 'test',
                        repo: 'test',
                    },
                },
            });

            const generator = new DesktopTemplateGenerator(config);
            const publishConfig = (generator as any).getPublishConfig();
            const effectiveProvider = (generator as any).getEffectiveUpdateProvider();

            expect(publishConfig).not.toBeNull();
            expect(publishConfig.provider).toBe('github');
            expect(effectiveProvider).toBe('github');
        });
    });

    it('wires the Linux artifact signer only when Linux signing is enabled', () => {
        const unsigned = new DesktopTemplateGenerator(createMinimalConfig());
        expect((unsigned as any).buildTemplateVariables({}).LINUX_ARTIFACT_SIGNER_HOOK).toBe('null');

        const signed = new DesktopTemplateGenerator(createMinimalConfig({
            code_signing: { enabled: true, linux: { gpg_key_id: 'ABC123' } },
        }));
        expect((signed as any).buildTemplateVariables({}).LINUX_ARTIFACT_SIGNER_HOOK)
            .toBe(JSON.stringify('scripts/sign-linux-artifacts.js'));
    });

    it('rejects unresolved tokens before writing a rendered file', async () => {
        const generator = new DesktopTemplateGenerator(createMinimalConfig());
        const sourcePath = path.join(tempDir, 'input.ts');
        await fs.writeFile(sourcePath, '{{NOT_A_VARIABLE}}');

        await expect((generator as any).processTemplateFile({
            sourcePath,
            targetPath: 'output.ts',
            isTemplate: true,
        }, {})).rejects.toThrow(/output\.ts.*\{\{NOT_A_VARIABLE\}\}/);
        expect(await fs.pathExists(path.join(tempDir, 'output.ts'))).toBe(false);
    });

    it('rejects invalid JSON before writing a rendered file', async () => {
        const generator = new DesktopTemplateGenerator(createMinimalConfig());
        const sourcePath = path.join(tempDir, 'input.json');
        await fs.writeFile(sourcePath, '{ invalid json }');

        await expect((generator as any).processTemplateFile({
            sourcePath,
            targetPath: 'output.json',
            isTemplate: true,
        }, {})).rejects.toThrow(/invalid JSON generated for output\.json/);
        expect(await fs.pathExists(path.join(tempDir, 'output.json'))).toBe(false);
    });

    it('does not inject fixed Tier 1 ports into bundled runtime launches', async () => {
        const template = await fs.readFile(path.resolve(__dirname, '../vanilla/main.ts'), 'utf8');
        expect(template).toContain('if (!isBundledMode)');
        expect(template).toContain('defeat the bundle allocator\'s role bands');
        expect(template).not.toContain('WC_SESSION_STATE_ROOT');
        expect(template).not.toContain('web-console-sessions');
    });

    it('pins generated projects to the governed Electron runtime version', async () => {
        const developerPackage = JSON.parse(await fs.readFile(path.resolve(__dirname, '../vanilla/package.json'), 'utf8'));
        const generatedTemplate = await fs.readFile(path.resolve(__dirname, '../vanilla/package.json.template'), 'utf8');
        const generatedElectronVersion = generatedTemplate.match(/"electron"\s*:\s*"([^"]+)"/)?.[1];

        expect(generatedElectronVersion).toBe(developerPackage.devDependencies.electron.replace(/^[~^]/, ''));
        expect(generatedElectronVersion).toBe('44.7.2');
        expect(generatedTemplate).toContain('{{ELECTRON_MKSNAPSHOT_DEPENDENCY}}');
    });

    it.each(['electron-sidecar', 'electron-optimized'] as const)(
        'generates the %s shell from the canonical vanilla source', async (strategy_id) => {
            const optimization_profile = strategy_id === 'electron-optimized'
                ? { shared_runtime_readiness: true, secure_custom_scheme_ui: false, v8_snapshot_code_cache: false, electron_fuses: false }
                : undefined;
            const generator = new DesktopTemplateGenerator(createMinimalConfig({ strategy_id, optimization_profile }));
            const canonicalMain = path.resolve(__dirname, '../vanilla/main.ts');
            const canonicalRoot = path.resolve(__dirname, '../vanilla');
            const mainFile = (await (generator as any).getTemplateFiles(canonicalRoot, canonicalRoot))
                .find((file: { targetPath: string }) => file.targetPath === 'main.ts');

            expect(mainFile).toBeDefined();
            expect(await fs.realpath(mainFile.sourcePath)).toBe(await fs.realpath(canonicalMain));

            await (generator as any).processTemplateFiles({ strategy_id });
            await (generator as any).createSourceStructure();

            const rendered = await fs.readFile(path.join(tempDir, 'src', 'main.ts'), 'utf8');
            const source = await fs.readFile(canonicalMain, 'utf8');
            expect(rendered).toBe(await (generator as any).processTemplateContent(source, { strategy_id }));
        },
    );

    it('includes startup snapshot build files only when the profile enables snapshots', async () => {
        const canonicalRoot = path.resolve(__dirname, '../vanilla');
        const sidecar = new DesktopTemplateGenerator(createMinimalConfig());
        const sidecarFiles = await (sidecar as any).getTemplateFiles(canonicalRoot, canonicalRoot);
        expect(sidecarFiles.some((file: { targetPath: string }) => file.targetPath.startsWith('scripts/') && file.targetPath.includes('snapshot'))).toBe(false);

        const optimized = new DesktopTemplateGenerator(createMinimalConfig({
            strategy_id: 'electron-optimized',
            optimization_profile: { shared_runtime_readiness: false, secure_custom_scheme_ui: false, v8_snapshot_code_cache: true, electron_fuses: false },
        }));
        const optimizedFiles = await (optimized as any).getTemplateFiles(canonicalRoot, canonicalRoot);
        expect(optimizedFiles.map((file: { targetPath: string }) => file.targetPath)).toEqual(expect.arrayContaining([
            'scripts/build-v8-snapshot.js',
            'scripts/v8-startup-snapshot.js',
        ]));
    });
});
