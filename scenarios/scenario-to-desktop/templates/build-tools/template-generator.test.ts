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
});
