import { DesktopConfig, DesktopTemplateGenerator } from "./template-generator";
import * as fs from "fs-extra";
import * as path from "path";

const toggleFields = [
    "shared_runtime_readiness",
    "secure_custom_scheme_ui",
    "v8_snapshot_code_cache",
    "electron_fuses",
] as const;

function makeConfig(profile?: Record<(typeof toggleFields)[number], boolean>, strategyID?: string): DesktopConfig {
    return {
        app_name: "hello-desktop",
        app_display_name: "Hello Desktop",
        app_description: "Optimization profile fixture",
        version: "1.0.0",
        author: "Vrooli",
        license: "MIT",
        app_id: "com.vrooli.hello-desktop",
        server_type: "node",
        server_path: "server.js",
        api_endpoint: "http://127.0.0.1:3000",
        framework: "electron",
        strategy_id: strategyID,
        optimization_profile: profile,
        template_type: "basic",
        platforms: ["linux"],
        output_path: "/tmp/e3-optimization-profile-test",
    } as DesktopConfig;
}

function templateVariables(config: DesktopConfig): Record<string, any> {
    return (new DesktopTemplateGenerator(config) as any).buildTemplateVariables({});
}

async function renderedPackage(config: DesktopConfig): Promise<Record<string, any>> {
    const generator = new DesktopTemplateGenerator(config);
    const template = await fs.readFile(path.resolve(__dirname, "../vanilla/package.json.template"), "utf8");
    const rendered = await (generator as any).processTemplateContent(template, {});
    return JSON.parse(rendered);
}

describe("Electron optimization profile projection", () => {
    it("keeps electron-sidecar as the all-off default", () => {
        const variables = templateVariables(makeConfig());
        expect(variables.PACKAGING_STRATEGY).toBe("electron-sidecar");
        expect(variables.OPT_SHARED_RUNTIME_READINESS).toBe(false);
        expect(variables.OPT_SECURE_CUSTOM_SCHEME_UI).toBe(false);
        expect(variables.OPT_V8_SNAPSHOT_CODE_CACHE).toBe(false);
        expect(variables.OPT_ELECTRON_FUSES).toBe(false);
        expect(JSON.parse(variables.ELECTRON_FUSES)).toBeNull();
    });

    it("projects each optimized toggle independently", () => {
        for (const enabledToggle of toggleFields) {
            const profile = Object.fromEntries(toggleFields.map((field) => [field, field === enabledToggle])) as Record<(typeof toggleFields)[number], boolean>;
            const variables = templateVariables(makeConfig(profile, "electron-optimized"));
            const actual = {
                shared_runtime_readiness: variables.OPT_SHARED_RUNTIME_READINESS,
                secure_custom_scheme_ui: variables.OPT_SECURE_CUSTOM_SCHEME_UI,
                v8_snapshot_code_cache: variables.OPT_V8_SNAPSHOT_CODE_CACHE,
                electron_fuses: variables.OPT_ELECTRON_FUSES,
            };
            expect(actual).toEqual(profile);
            if (enabledToggle === "electron_fuses") {
                expect(JSON.parse(variables.ELECTRON_FUSES)).toEqual({
                    runAsNode: false,
                    enableNodeOptionsEnvironmentVariable: false,
                    enableNodeCliInspectArguments: false,
                    enableEmbeddedAsarIntegrityValidation: true,
                    onlyLoadAppFromAsar: true,
                    grantFileProtocolExtraPrivileges: false,
                });
            } else if (enabledToggle === "v8_snapshot_code_cache") {
                expect(JSON.parse(variables.ELECTRON_FUSES)).toEqual({
                    runAsNode: false,
                    enableNodeOptionsEnvironmentVariable: false,
                    enableNodeCliInspectArguments: false,
                    enableEmbeddedAsarIntegrityValidation: true,
                    onlyLoadAppFromAsar: true,
                    grantFileProtocolExtraPrivileges: false,
                    loadBrowserProcessSpecificV8Snapshot: true,
                });
                expect(variables.V8_SNAPSHOT_BEFORE_PACK).toBe('"scripts/build-v8-snapshot.js"');
                expect(JSON.parse(variables.EXTRA_RESOURCES)).toContainEqual({
                    from: "snapshot-output/browser_v8_context_snapshot.bin",
                    to: "browser_v8_context_snapshot.bin",
                });
            } else {
                expect(JSON.parse(variables.ELECTRON_FUSES)).toBeNull();
            }
        }
    });

    it("passes Electron fuse selections into the generated Electron Builder config", async () => {
        const sidecar = await renderedPackage(makeConfig());
        expect(sidecar.build.electronFuses).toBeNull();
        expect(sidecar.build.beforePack).toBeNull();
        expect(sidecar.build.extraResources).toHaveLength(1);
        expect(sidecar.devDependencies["electron-mksnapshot"]).toBeUndefined();

        const fusesOnly = Object.fromEntries(toggleFields.map((field) => [field, field === "electron_fuses"])) as Record<(typeof toggleFields)[number], boolean>;
        const optimized = await renderedPackage(makeConfig(fusesOnly, "electron-optimized"));
        expect(optimized.build.electronFuses).toEqual({
            runAsNode: false,
            enableNodeOptionsEnvironmentVariable: false,
            enableNodeCliInspectArguments: false,
            enableEmbeddedAsarIntegrityValidation: true,
            onlyLoadAppFromAsar: true,
            grantFileProtocolExtraPrivileges: false,
        });

        const snapshotOnly = Object.fromEntries(toggleFields.map((field) => [field, field === "v8_snapshot_code_cache"])) as Record<(typeof toggleFields)[number], boolean>;
        const snapshot = await renderedPackage(makeConfig(snapshotOnly, "electron-optimized"));
        expect(snapshot.build.beforePack).toBe("scripts/build-v8-snapshot.js");
        expect(snapshot.build.electronFuses.loadBrowserProcessSpecificV8Snapshot).toBe(true);
        expect(snapshot.build.extraResources).toHaveLength(2);
        expect(snapshot.devDependencies["electron-mksnapshot"]).toBe("44.7.2");
    });
});
