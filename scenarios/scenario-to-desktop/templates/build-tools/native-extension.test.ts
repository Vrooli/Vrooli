import * as fs from 'fs/promises';
import * as path from 'path';
import * as os from 'os';
import ts from 'typescript';
import { NativeExtensionCompatibilityError, validateNativeExtension } from './native-extension';
import { DesktopTemplateGenerator, DesktopConfig } from './template-generator';

const valid = { version: 1, module: 'presentation', permissions: ['window.presentation'], platforms: ['linux', 'win', 'mac'] };
describe('native extension admission', () => {
    it('requires version 3 and explicit context permission for native capture', () => {
        const extension = { ...valid, version: 3, permissions: ['window.presentation', 'global-shortcut', 'desktop.context'], activation_shortcut: 'Control+Shift+Space' };
        expect(() => validateNativeExtension(extension, 'electron', ['linux'])).not.toThrow();
        expect(() => validateNativeExtension({...extension, version: 2}, 'electron', ['linux'])).toThrow();
        expect(() => validateNativeExtension({...extension, permissions: ['window.presentation', 'global-shortcut']}, 'electron', ['linux'])).toThrow();
    });
    it('requires version 2 and its permission for activation', () => {
        const extension = { ...valid, version: 2, permissions: ['window.presentation','global-shortcut'], activation_shortcut: 'CommandOrControl+Shift+Space' };
        expect(() => validateNativeExtension(extension,'electron',['linux'])).not.toThrow();
        for (const invalid of [{...extension,version:1}, {...extension,permissions:['window.presentation']}, {...extension,activation_shortcut:'Alt+Alt+Space'}, {...extension,activation_shortcut:'code.js'}]) {
            expect(() => validateNativeExtension(invalid,'electron',['linux'])).toThrow();
        }
    });
    it('accepts canonical pipeline targets and refuses unknown architectures', () => {
        expect(() => validateNativeExtension(valid, 'electron', ['linux-amd64','linux-arm64','darwin-amd64','darwin-arm64','windows-amd64','windows-arm64'])).not.toThrow();
        expect(() => validateNativeExtension(valid, 'electron', ['linux-unknown'])).toThrow();
    });
    it('keeps vanilla configurations and admits the built-in module', () => {
        expect(() => validateNativeExtension(undefined, 'electron', ['linux'])).not.toThrow();
        expect(() => validateNativeExtension(valid, 'electron', ['linux', 'mac'])).not.toThrow();
    });
    it('returns a stable typed reason for an unsupported extension contract', () => {
        try {
            validateNativeExtension({ ...valid, version: 4 }, 'electron', ['linux']);
            throw new Error('expected extension admission to fail');
        } catch (error) {
            expect(error).toBeInstanceOf(NativeExtensionCompatibilityError);
            expect((error as NativeExtensionCompatibilityError).code).toBe('NATIVE_EXTENSION_UNSUPPORTED_CONTRACT');
        }
    });
    it('identifies a target mismatch without exposing arbitrary paths', () => {
        const linuxOnly = { ...valid, platforms: ['linux'] as ('linux' | 'mac' | 'win')[] };
        expect(() => validateNativeExtension(linuxOnly, 'electron', ['windows-arm64'])).toThrowError(NativeExtensionCompatibilityError);
        try {
            validateNativeExtension(linuxOnly, 'electron', ['windows-arm64']);
        } catch (error) {
            expect((error as NativeExtensionCompatibilityError).code).toBe('NATIVE_EXTENSION_TARGET_UNSUPPORTED');
            expect((error as NativeExtensionCompatibilityError).target).toBe('windows-arm64');
        }
    });
    it.each([
        { ...valid, version: 3 }, { ...valid, module: '../custom' },
        { ...valid, entrypoint: '/tmp/custom.js' }, { ...valid, permissions: ['filesystem'] },
        { ...valid, platforms: ['mac'] }, { ...valid, platforms: ['linux', 'linux'] },
    ])('rejects unsupported contracts before generation: %j', native_extension => {
        expect(() => new DesktopTemplateGenerator({ framework: 'electron', platforms: ['linux'], native_extension } as unknown as DesktopConfig)).toThrow();
    });
    it('admits only the governed desktop-session helper provider', () => {
        expect(() => validateNativeExtension({ ...valid, helper_providers: [{ owner: 'device-control', capability: 'desktop.session' }] }, 'electron', ['linux'])).not.toThrow();
        expect(() => validateNativeExtension({ ...valid, helper_providers: [{ owner: 'device-control', capability: 'arbitrary.exec' }] }, 'electron', ['linux'])).toThrowError(NativeExtensionCompatibilityError);
        expect(() => validateNativeExtension({ ...valid, helper_providers: [{ owner: 'device-control', capability: 'desktop.session', path: '/tmp/helper' }] }, 'electron', ['linux'])).toThrowError(NativeExtensionCompatibilityError);
    });
    it('renders the declaration as data', async () => {
        const generator = new DesktopTemplateGenerator({ framework: 'electron', output_path: '/tmp/native-generation-test', platforms: ['linux'], native_extension: valid, features: {}, window: {} } as unknown as DesktopConfig);
        const rendered = await (generator as any).processTemplateContent('JSON.parse("{{NATIVE_EXTENSION_CONFIG}}")', {});
        expect(JSON.parse(JSON.parse(rendered.slice('JSON.parse('.length, -1)))).toEqual(valid);
    });
});


describe('generated native extension layout', () => {
    it.each([false, true])('packages executable template hooks (enabled=%s)', async enabled => {
        const output = await fs.mkdtemp(path.join(os.tmpdir(), 'native-layout-'));
        try {
            const generator = new DesktopTemplateGenerator({ app_name: 'example', app_display_name: 'Example', app_description: 'Example',
                version: '1.0.0', author: 'test', license: 'MIT', app_id: 'test.example', framework: 'electron', template_type: 'basic',
                server_type: 'external', server_path: '', api_endpoint: '', output_path: output, platforms: ['linux'],
                features: {}, window: {}, update_config: { provider: 'none' },
                ...(enabled ? { native_extension: valid } : {}),
            } as DesktopConfig);
            const internal = generator as any;
            internal.templateBasePath = path.resolve(__dirname, '..');
            const template = await internal.loadTemplateConfig();
            await internal.processTemplateFiles(template);
            await internal.generateAdditionalFiles(template);
            const main = await fs.readFile(path.join(output, 'src/main.ts'), 'utf8');
            const preload = await fs.readFile(path.join(output, 'src/preload.ts'), 'utf8');
            expect(main).toContain('from "./native/presentation"');
            // PKG-08: remote scenario content runs in a sandboxed renderer and
            // cannot reach Node or an unrestricted host IPC surface.
            expect(main).toContain('nodeIntegration: false');
            expect(main).toContain('contextIsolation: true');
            expect(main).toContain('sandbox: true');
            expect(main).toContain('return { action: "deny" }');
            expect(main).not.toContain('{{NATIVE_EXTENSION_CONFIG}}');
            expect(preload).not.toContain('{{NATIVE_EXTENSION_CONFIG}}');
            expect((await fs.readFile(path.join(output, 'src/native/presentation.ts'), 'utf8'))).toContain('installPresentation');
            await expect(fs.stat(path.join(output, 'src/native/__tests__'))).rejects.toThrow();
            for (const source of [main, preload]) {
                const result = ts.transpileModule(source, { reportDiagnostics: true, compilerOptions: { target: ts.ScriptTarget.ES2020, module: ts.ModuleKind.CommonJS } });
                expect(result.diagnostics).toEqual([]);
            }
            if (enabled) {
                const metadata = JSON.parse(await fs.readFile(path.join(output, 'native-extension.json'), 'utf8'));
                expect(metadata.module).toBe('presentation');
                expect(metadata.main_entrypoint).toBe('dist/native/presentation.js');
                const pkg = JSON.parse(await fs.readFile(path.join(output, 'package.json'), 'utf8'));
                expect(pkg.build.files).toContain('native-extension.json');
                expect(metadata.native_dependencies).toEqual([]);
                expect(metadata.helper_providers).toEqual([]);
            } else {
                expect(preload).toContain('if (JSON.parse("null"))');
                await expect(fs.stat(path.join(output, 'native-extension.json'))).rejects.toThrow();
            }
        } finally { await fs.rm(output, { recursive: true, force: true }); }
    });
});
