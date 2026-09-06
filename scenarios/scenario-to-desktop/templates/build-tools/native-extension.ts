/** Versioned, closed registry. Extensions select owner-provided code, never imports. */
export interface NativeExtension {
    version: 1 | 2 | 3;
    module: 'presentation';
    permissions: string[];
    activation_shortcut?: string;
    platforms: ('linux' | 'mac' | 'win')[];
}

export function validateNativeExtension(value: unknown, framework: string, targets: string[] = []): asserts value is NativeExtension | undefined {
    if (value === undefined || value === null) return;
    if (typeof value !== 'object' || Array.isArray(value)) throw new Error('native_extension must be an object');
    const config = value as Record<string, unknown>;
    if (Object.keys(config).some(key => !['version', 'module', 'permissions', 'platforms', 'activation_shortcut'].includes(key))) {
        throw new Error('native_extension contains undeclared fields');
    }
    if (framework !== 'electron' || (config.version !== 1 && config.version !== 2 && config.version !== 3) || config.module !== 'presentation') {
        throw new Error('unsupported native extension contract, module or framework');
    }
    const expectedPermissions = config.version === 3 ? ['window.presentation', 'global-shortcut', 'desktop.context'] : config.version === 2 ? ['window.presentation', 'global-shortcut'] : ['window.presentation'];
    if (!Array.isArray(config.permissions) || config.permissions.length !== expectedPermissions.length || expectedPermissions.some(p => !(config.permissions as unknown[]).includes(p))) {
        throw new Error('native extension permissions do not match the selected contract');
    }
    const shortcut = config.activation_shortcut;
    if (config.version === 1 && shortcut !== undefined) throw new Error('shortcut requires native extension version 2');
    if ((config.version === 2 || config.version === 3) && (typeof shortcut !== 'string' || !/^(CommandOrControl|Control|Alt|Super)(\+(Shift|Alt))?\+(Space|[A-Z0-9])$/.test(shortcut) || new Set(shortcut.split('+')).size !== shortcut.split('+').length)) {
        throw new Error('version 2 requires a supported activation shortcut');
    }
    const platforms = config.platforms;
    const hostTargets: Record<string, string> = { darwin: 'mac', win32: 'win', linux: 'linux' };
    const effectiveTargets = targets.length ? targets : [hostTargets[process.platform] ?? 'unsupported'];
    if (!Array.isArray(platforms) || !platforms.length || new Set(platforms).size !== platforms.length ||
        platforms.some(target => !['linux', 'mac', 'win'].includes(target)) || effectiveTargets.some(target => {
            const match = /^(linux|darwin|windows)-(amd64|arm64)$/.exec(target);
            const osNames: Record<string, string> = { linux: 'linux', darwin: 'mac', windows: 'win' };
            return !platforms.includes(match ? osNames[match[1]] : target);
        })) {
        throw new Error('native extension platforms must cover all build targets');
    }
}
