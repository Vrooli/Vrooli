/** Versioned, closed registry. Extensions select owner-provided code, never imports. */
export interface NativeExtension {
    version: 1 | 2 | 3;
    module: 'presentation';
    permissions: string[];
    activation_shortcut?: string;
    platforms: ('linux' | 'mac' | 'win')[];
    helper_providers?: { owner: 'device-control'; capability: 'desktop.session' }[];
}

/** Stable machine-readable reason returned when extension admission fails. */
export type NativeExtensionCompatibilityCode =
    | 'NATIVE_EXTENSION_UNDECLARED_FIELD'
    | 'NATIVE_EXTENSION_UNSUPPORTED_CONTRACT'
    | 'NATIVE_EXTENSION_PERMISSION_MISMATCH'
    | 'NATIVE_EXTENSION_SHORTCUT_INVALID'
    | 'NATIVE_EXTENSION_PLATFORM_INVALID'
    | 'NATIVE_EXTENSION_TARGET_UNSUPPORTED'
    | 'NATIVE_EXTENSION_HELPER_PROVIDER_INVALID';

export class NativeExtensionCompatibilityError extends Error {
    readonly code: NativeExtensionCompatibilityCode;
    readonly field?: string;
    readonly target?: string;

    constructor(code: NativeExtensionCompatibilityCode, message: string, details: { field?: string; target?: string } = {}) {
        super(message);
        this.name = 'NativeExtensionCompatibilityError';
        this.code = code;
        this.field = details.field;
        this.target = details.target;
    }
}

function reject(code: NativeExtensionCompatibilityCode, message: string, details: { field?: string; target?: string } = {}): never {
    throw new NativeExtensionCompatibilityError(code, message, details);
}

export function validateNativeExtension(value: unknown, framework: string, targets: string[] = []): asserts value is NativeExtension | undefined {
    if (value === undefined || value === null) return;
    if (typeof value !== 'object' || Array.isArray(value)) reject('NATIVE_EXTENSION_UNSUPPORTED_CONTRACT', 'native_extension must be an object');
    const config = value as Record<string, unknown>;
    const undeclared = Object.keys(config).find(key => !['version', 'module', 'permissions', 'platforms', 'activation_shortcut', 'helper_providers'].includes(key));
    if (undeclared) {
        reject('NATIVE_EXTENSION_UNDECLARED_FIELD', 'native_extension contains undeclared fields', { field: undeclared });
    }
    if (framework !== 'electron' || (config.version !== 1 && config.version !== 2 && config.version !== 3) || config.module !== 'presentation') {
        reject('NATIVE_EXTENSION_UNSUPPORTED_CONTRACT', 'unsupported native extension contract, module or framework');
    }
    const expectedPermissions = config.version === 3 ? ['window.presentation', 'global-shortcut', 'desktop.context'] : config.version === 2 ? ['window.presentation', 'global-shortcut'] : ['window.presentation'];
    if (!Array.isArray(config.permissions) || config.permissions.length !== expectedPermissions.length || expectedPermissions.some(p => !(config.permissions as unknown[]).includes(p))) {
        reject('NATIVE_EXTENSION_PERMISSION_MISMATCH', 'native extension permissions do not match the selected contract', { field: 'permissions' });
    }
    const shortcut = config.activation_shortcut;
    if (config.version === 1 && shortcut !== undefined) reject('NATIVE_EXTENSION_SHORTCUT_INVALID', 'shortcut requires native extension version 2', { field: 'activation_shortcut' });
    if ((config.version === 2 || config.version === 3) && (typeof shortcut !== 'string' || !/^(CommandOrControl|Control|Alt|Super)(\+(Shift|Alt))?\+(Space|[A-Z0-9])$/.test(shortcut) || new Set(shortcut.split('+')).size !== shortcut.split('+').length)) {
        reject('NATIVE_EXTENSION_SHORTCUT_INVALID', 'version 2 requires a supported activation shortcut', { field: 'activation_shortcut' });
    }
    const providers = config.helper_providers;
    if (providers !== undefined) {
        if (!Array.isArray(providers)) reject('NATIVE_EXTENSION_HELPER_PROVIDER_INVALID', 'helper_providers must be an array', { field: 'helper_providers' });
        const seen = new Set<string>();
        for (const provider of providers as unknown[]) {
            if (!provider || typeof provider !== 'object' || Array.isArray(provider)) reject('NATIVE_EXTENSION_HELPER_PROVIDER_INVALID', 'helper provider must be an object', { field: 'helper_providers' });
            const value = provider as Record<string, unknown>;
            const undeclaredProviderField = Object.keys(value).find(key => key !== 'owner' && key !== 'capability');
            if (undeclaredProviderField) reject('NATIVE_EXTENSION_HELPER_PROVIDER_INVALID', 'helper provider contains undeclared fields', { field: 'helper_providers' });
            if (value.owner !== 'device-control' || value.capability !== 'desktop.session') reject('NATIVE_EXTENSION_HELPER_PROVIDER_INVALID', 'helper provider is not in the governed registry', { field: 'helper_providers' });
            const key = `${value.owner}/${value.capability}`;
            if (seen.has(key)) reject('NATIVE_EXTENSION_HELPER_PROVIDER_INVALID', 'duplicate helper provider', { field: 'helper_providers' });
            seen.add(key);
        }
    }
    const platforms = config.platforms;
    const declaredPlatforms = Array.isArray(platforms) ? platforms : [];
    const hostTargets: Record<string, string> = { darwin: 'mac', win32: 'win', linux: 'linux' };
    const effectiveTargets = targets.length ? targets : [hostTargets[process.platform] ?? 'unsupported'];
    if (!declaredPlatforms.length || new Set(declaredPlatforms).size !== declaredPlatforms.length ||
        declaredPlatforms.some(target => !['linux', 'mac', 'win'].includes(target)) || effectiveTargets.some(target => {
        const match = /^(linux|darwin|windows)-(amd64|arm64)$/.exec(target);
        const osNames: Record<string, string> = { linux: 'linux', darwin: 'mac', windows: 'win' };
        return !declaredPlatforms.includes(match ? osNames[match[1]] : target);
    })) {
        const invalid = effectiveTargets.find(target => {
            const match = /^(linux|darwin|windows)-(amd64|arm64)$/.exec(target);
            const osNames: Record<string, string> = { linux: 'linux', darwin: 'mac', windows: 'win' };
            return !declaredPlatforms.includes(match ? osNames[match[1]] : target);
        });
        reject(invalid ? 'NATIVE_EXTENSION_TARGET_UNSUPPORTED' : 'NATIVE_EXTENSION_PLATFORM_INVALID', 'native extension platforms must cover all build targets', { field: 'platforms', target: invalid });
    }
}
