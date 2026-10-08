import { constants } from 'node:fs';
import { randomUUID } from 'node:crypto';
import { open } from 'node:fs/promises';
import { request } from 'node:http';
import { isAbsolute } from 'node:path';
import type { ActivationCapture } from './presentation';

export interface LocalActivationBinding {
    socket: string;
    includeImage?: boolean;
    session: {
        surface: { target: { ownerScenario: string; resourceId: string; hostNodeId: string }; ownerScenario: string; surfaceId: string };
        sessionId: string;
        desktopSessionId: string;
    };
}
const refused = () => new Error('Local desktop activation context unavailable');

// Electron's native handle is read in the main process, never accepted over IPC.
export function nativeWindowIdentity(handle: Buffer): string {
    if (process.platform !== 'linux' || (handle.length !== 4 && handle.length !== 8)) throw refused();
    const id = handle.length === 4 ? BigInt(handle.readUInt32LE()) : handle.readBigUInt64LE();
    if (id === 0n || id > 0xffffffffn) throw refused();
    return id.toString();
}

const ownerCall = async (socket: string, method: string, payload: object, signal: AbortSignal): Promise<unknown> => {
        signal.throwIfAborted();
        const body = JSON.stringify(payload);
        return new Promise((resolve, reject) => {
            const req = request({ socketPath: socket, path: `/vrooli.device_control.v1.desktop.DesktopOwnerService/${method}`,
                method: 'POST', signal, headers: { 'Content-Type': 'application/json', 'Connect-Protocol-Version': '1', 'Content-Length': Buffer.byteLength(body) } }, res => {
                const chunks: Buffer[] = []; let length = 0;
                res.on('data', (chunk: Buffer) => { length += chunk.length; if (length > (method === 'ReadActivationImage' ? 45 * 1024 * 1024 : 16 * 1024)) { req.destroy(refused()); return; } chunks.push(chunk); });
                res.on('error', reject);
                res.on('end', () => {
                    if (res.statusCode !== 200) { reject(refused()); return; }
                    try { const parsed: unknown = JSON.parse(Buffer.concat(chunks).toString('utf8')); resolve(parsed); } catch { reject(refused()); }
                });
            });
            req.on('error', reject); req.setTimeout(method === 'ReadActivationImage' ? 2000 : 700, () => req.destroy(refused())); req.end(body);
        });
    };

/** Binding is trusted native configuration, never a renderer URL. No network
 * discovery, automatic Open, account fallback or capture retry. */
export function localActivationCapture(binding: LocalActivationBinding, window: string): ActivationCapture {
    const value = JSON.parse(JSON.stringify(binding)) as LocalActivationBinding;
    const s = value.session;
    if (value.includeImage !== undefined && typeof value.includeImage !== 'boolean') throw refused();
    const fields = [s.sessionId, s.desktopSessionId, s.surface.ownerScenario, s.surface.surfaceId,
        s.surface.target.ownerScenario, s.surface.target.resourceId, s.surface.target.hostNodeId];
    if (!isAbsolute(value.socket) || value.socket.includes('\0') || fields.some(v => typeof v !== 'string' || !v || v.length > 256) ||
        !/^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$/.test(s.desktopSessionId) || s.surface.ownerScenario !== 'device-control') throw refused();
    if (!/^[1-9][0-9]{0,9}$/.test(window) || BigInt(window)>0xffffffffn) throw refused();
    const call = (method: string, payload: object, signal: AbortSignal) => ownerCall(value.socket, method, payload, signal);
    return { capture: async signal => {
        const response = await call('CaptureCompanionActivation', { session: s, companionWindow: window, ...(value.includeImage ? {includeImage:true} : {}) }, signal);
        signal.throwIfAborted();
        if (!response || typeof response !== 'object') throw refused();
        const ref = response as Record<string, unknown>;
        const expires = typeof ref.expiresAt === 'string' ? Date.parse(ref.expiresAt) : NaN;
        const captured = typeof ref.capturedAt === 'string' ? Date.parse(ref.capturedAt) : NaN;
        const now = Date.now();
        if (typeof ref.contextId !== 'string' || !/^[0-9a-f]{8}(-[0-9a-f]{4}){3}-[0-9a-f]{12}$/.test(ref.contextId) || ref.contextId === '00000000-0000-0000-0000-000000000000' || !Number.isFinite(expires) || !Number.isFinite(captured) || captured > now || expires <= now || expires > captured + 30000) throw refused();
        if (value.includeImage === true && (ref.hasImage !== true ||
            [ref.displayId, ref.geometryRevision].some(field => typeof field !== 'string' || !field || field.length > 128))) throw refused();
        const contextId = ref.contextId;
        const readImage = value.includeImage ? async (readSignal: AbortSignal) => {
            readSignal.throwIfAborted();
            if (Date.now() >= expires) throw refused();
            const response = await call('ReadActivationImage', {session:s,contextId}, readSignal);
            readSignal.throwIfAborted();
            if (Date.now() >= expires || !response || typeof response !== 'object') throw refused();
            const image = response as Record<string,unknown>;
            if (!image.reference || typeof image.reference !== 'object' || typeof image.png !== 'string' || image.png.length > 44739244 || (image.png.length % 4 !== 0 || !/^[A-Za-z0-9+/]*={0,2}$/.test(image.png))) throw refused();
            const current = image.reference as Record<string,unknown>;
            for (const field of ['contextId','capturedAt','expiresAt','displayId','geometryRevision','hasImage']) if (current[field] !== ref[field]) throw refused();
            const bounds = current.sourceBounds as {x?:number;y?:number;width:number;height:number} | undefined;
            const original = ref.sourceBounds as typeof bounds;
            if (!bounds || !original || ['x','y','width','height'].some(key => (bounds[key as keyof typeof bounds] ?? 0) !== (original[key as keyof typeof original] ?? 0)) ||
                !Number.isInteger(bounds.width) || !Number.isInteger(bounds.height) || bounds.width <= 0 || bounds.height <= 0 || bounds.width > 65535 || bounds.height > 65535 || bounds.width*bounds.height > 16*1024*1024 ||
                !Number.isInteger(bounds.x ?? 0) || !Number.isInteger(bounds.y ?? 0) || (bounds.x ?? 0) < -2147483648 || (bounds.y ?? 0) < -2147483648 || (bounds.x ?? 0)+bounds.width > 2147483647 || (bounds.y ?? 0)+bounds.height > 2147483647) throw refused();
            const png = Buffer.from(image.png,'base64');
            if (png.length > 32*1024*1024 || png.length < 33 || png.subarray(0,8).toString('hex') !== '89504e470d0a1a0a' || png.readUInt32BE(8) !== 13 || png.toString('ascii',12,16) !== 'IHDR' || png.readUInt32BE(16) !== bounds.width || png.readUInt32BE(20) !== bounds.height) throw refused();
            const sourceBounds = {x:bounds.x ?? 0,y:bounds.y ?? 0,width:bounds.width,height:bounds.height};
            // Construct known fields from the bound session and original capture.
            // No helper/session credentials or window identifiers cross IPC.
            const surface = {ownerScenario:s.surface.ownerScenario,surfaceId:s.surface.surfaceId,
                target:{ownerScenario:s.surface.target.ownerScenario,resourceId:s.surface.target.resourceId,hostNodeId:s.surface.target.hostNodeId}};
            return {contextId,expiresAt:expires,mimeType:'image/png' as const,dataUrl:`data:image/png;base64,${image.png}`,sourceBounds,
                source:{surface,captureId:contextId,displayId:ref.displayId as string,geometryRevision:ref.geometryRevision as string,
                    capturedAt:new Date(captured).toISOString(),bounds:{...sourceBounds}}};
        } : undefined;
        return { contextId, expiresAt: expires, ...(readImage ? {readImage} : {}), discard: async () => {
            await call('DeleteActivation', { session: s, contextId }, AbortSignal.timeout(700));
        } };
    } };
}

/** Private binding may be atomically renewed by its local owner. Read on each
 * activation; missing/invalid data fails capture without preventing UI startup. */
export function configuredLocalActivationCapture(configPath: string | undefined, nativeHandle: () => Buffer): ActivationCapture {
    return { capture: async signal => {
        signal.throwIfAborted();
        if (process.platform !== 'linux' || !process.getuid || !configPath || !isAbsolute(configPath)) throw refused();
        const file = await open(configPath, constants.O_RDONLY | constants.O_NOFOLLOW | constants.O_NONBLOCK);
        let binding: LocalActivationBinding | LocalActivationSource;
        try {
            const info = await file.stat();
            if (!info.isFile() || info.uid !== process.getuid() || (info.mode & 0o077) !== 0 || info.size > 16384) throw refused();
            const buffer = Buffer.alloc(16385);
            const { bytesRead } = await file.read(buffer, 0, buffer.length, 0);
            if (bytesRead > 16384) throw refused();
            binding = JSON.parse(buffer.subarray(0, bytesRead).toString('utf8')) as LocalActivationBinding | LocalActivationSource;
        } finally { await file.close(); }
        signal.throwIfAborted();
        return ("version" in binding
            ? admittedLocalActivationCapture(binding, nativeWindowIdentity(nativeHandle()))
            : localActivationCapture(binding as LocalActivationBinding, nativeWindowIdentity(nativeHandle()))).capture(signal);
    } };
}

/** Version 2 is an explicit private opt-in to a bounded observation admission
 * per activation. It contains a destination, never an input credential. */
export interface LocalActivationSource {
    version: 2;
    socket: string;
    includeImage?: boolean;
    surface: LocalActivationBinding['session']['surface'];
}

export function admittedLocalActivationCapture(source: LocalActivationSource, window: string): ActivationCapture {
    const value = JSON.parse(JSON.stringify(source)) as LocalActivationSource;
    if ((value as {version:unknown}).version !== 2) throw refused();
    // Reuse the exact destination/window validator before creating authority.
    localActivationCapture({socket:value.socket,...(value.includeImage !== undefined ? {includeImage:value.includeImage} : {}),session:{surface:value.surface,sessionId:'validate',desktopSessionId:'validate'}},window);
    const call = (method:string,payload:object,signal:AbortSignal) => ownerCall(value.socket,method,payload,signal);
    return {capture:async signal => {
        signal.throwIfAborted();
        const intent = {surface:value.surface,control:false,ttlSeconds:30,requestId:randomUUID()};
        let session: LocalActivationBinding['session'] | undefined;
        const exactSession = (response:unknown): LocalActivationBinding['session'] => {
            if (!response || typeof response !== 'object' || !('session' in response)) throw refused();
            const candidate = response.session as LocalActivationBinding['session'];
            localActivationCapture({socket:value.socket,session:candidate},window);
            const a=candidate.surface,b=value.surface;
            if (a.ownerScenario!==b.ownerScenario || a.surfaceId!==b.surfaceId || a.target.ownerScenario!==b.target.ownerScenario || a.target.resourceId!==b.target.resourceId || a.target.hostNodeId!==b.target.hostNodeId) throw refused();
            return candidate;
        };
        const stop = async () => {
            if (!session) return;
            const cleanup = AbortSignal.timeout(1500);
            try { await call('Stop',{session},cleanup); } catch { /* Exact cleanup read can confirm a lost Stop reply. */ }
            const receipt = await call('ReadCleanup',{session},cleanup);
            const confirmed = exactSession(receipt);
            if (!receipt || typeof receipt !== 'object' || !('released' in receipt) || receipt.released !== true || confirmed.sessionId !== session.sessionId || confirmed.desktopSessionId !== session.desktopSessionId) throw refused();
        };
        try {
            const opened = await call('Open',intent,signal);
            session = exactSession(opened);
            const admission = opened as Record<string,unknown>;
            const expiry = typeof admission.expiresAt === 'string' ? Date.parse(admission.expiresAt) : NaN;
            if (admission.control === true || !Number.isFinite(expiry) || expiry <= Date.now() || expiry > Date.now()+30000) throw refused();
            const result = await localActivationCapture({socket:value.socket,...(value.includeImage !== undefined ? {includeImage:value.includeImage} : {}),session},window).capture(signal);
            return {...result,discard:async () => {
                try { await result.discard?.(); } finally { await stop(); }
            }};
        } catch (error) {
            // One read/reconciliation for this exact intent, never a second Open.
            if (!session) {
                try {
                    const disposition = await call('ReconcileOpen',intent,AbortSignal.timeout(1500));
                    if (disposition && typeof disposition === 'object' && 'state' in disposition && disposition.state === 'forwarding') session=exactSession(disposition);
                } catch { /* Owner/helper expiry bounds an unreachable admission. */ }
            }
            try { await stop(); } catch { /* Do not replay cleanup; helper expiry still applies. */ }
            throw error;
        }
    }};
}
