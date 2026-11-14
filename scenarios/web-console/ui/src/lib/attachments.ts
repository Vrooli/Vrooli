/**
 * Shared attachment policy for the UI. This mirrors the server's upload policy
 * (api/upload_handler.go): native executables are refused, everything else is
 * accepted and grouped by category so the composer can pick a sensible preview.
 * The server remains the source of truth — this only avoids staging a file the
 * server is guaranteed to reject.
 */

export type AttachmentKind = "image" | "video" | "audio" | "file";

/** Extensions the server refuses outright (native executables / installers). */
const BLOCKED_EXTENSIONS = new Set([
  "exe", "dll", "so", "dylib", "msi", "com", "scr", "bat", "cmd",
  "cpl", "sys", "drv", "efi", "apk", "dmg", "pkg", "deb", "rpm", "jar",
]);

const IMAGE_EXTENSIONS = new Set([
  "png", "jpg", "jpeg", "gif", "webp", "bmp", "ico", "avif", "tiff", "tif", "svg",
]);
const VIDEO_EXTENSIONS = new Set(["mp4", "webm", "ogv", "mov", "m4v"]);
const AUDIO_EXTENSIONS = new Set(["mp3", "wav", "ogg", "oga", "m4a", "aac", "flac", "opus"]);

/** Accept token for a device-camera capture input. */
export const CAMERA_PICKER_ACCEPT = "image/*";

/** Lower-case extension without the dot, or "" when there is none. */
export function extensionOf(name: string): string {
  const idx = name.lastIndexOf(".");
  if (idx <= 0 || idx === name.length - 1) return "";
  return name.slice(idx + 1).toLowerCase();
}

interface MediaDevicesLike {
  getUserMedia?: (constraints: MediaStreamConstraints) => Promise<MediaStream>;
}

/**
 * Read navigator.mediaDevices through a cast. The DOM lib declares it as always
 * present, but older browsers and insecure contexts omit it.
 */
export function getCameraMediaDevices(): MediaDevicesLike | undefined {
  return (globalThis.navigator as unknown as { mediaDevices?: MediaDevicesLike }).mediaDevices;
}

/** True when this browser can open an in-app camera stream. */
export function cameraCaptureSupported(): boolean {
  return Boolean(getCameraMediaDevices()?.getUserMedia);
}

/** True when the server will reject this file as an executable/installer. */
export function isBlockedAttachment(file: File): boolean {
  return BLOCKED_EXTENSIONS.has(extensionOf(file.name));
}

/** Classify a file for preview purposes, preferring the MIME type. */
export function attachmentKind(file: File): AttachmentKind {
  const type = file.type.toLowerCase();
  if (type.startsWith("image/")) return "image";
  if (type.startsWith("video/")) return "video";
  if (type.startsWith("audio/")) return "audio";
  // Browsers sometimes omit the type (e.g. SVG, drag-in from some apps).
  return attachmentKindForName(file.name);
}

/** Classify by file name alone — for host files, which carry no MIME type. */
export function attachmentKindForName(name: string): AttachmentKind {
  const ext = extensionOf(name);
  if (IMAGE_EXTENSIONS.has(ext)) return "image";
  if (VIDEO_EXTENSIONS.has(ext)) return "video";
  if (AUDIO_EXTENSIONS.has(ext)) return "audio";
  return "file";
}

/** Human-readable byte size (binary units). */
export function formatFileSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return "";
  if (bytes < 1024) return `${String(bytes)} B`;
  const units = ["KiB", "MiB", "GiB", "TiB"];
  let value = bytes / 1024;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  const rounded = value >= 10 ? Math.round(value) : Math.round(value * 10) / 10;
  return `${String(rounded)} ${units[unit] ?? ""}`;
}

// ---------------------------------------------------------------------------
// Attach sources. The menu chooses the MACHINE a file comes from; the device's
// own picker chooses camera / photo library / files wherever the OS has one.
// ---------------------------------------------------------------------------

/**
 * - device: this device's native picker (its OS offers camera/library/files)
 * - upload: this device's file dialog (desktop browsers: no source chooser)
 * - camera: the in-app camera, for devices whose OS has no capture chooser
 * - host:   browse the machine the session runs on and attach by path
 */
export type AttachSourceId = "device" | "upload" | "camera" | "host";

/** Platforms whose file input opens an OS chooser with camera + library + files. */
export type NativeChooserPlatform = "iphone" | "ipad" | "android";

export interface AttachEnvironment {
  hostname: string;
  userAgent: string;
  maxTouchPoints: number;
}

export function readAttachEnvironment(): AttachEnvironment {
  const nav = typeof navigator === "undefined" ? undefined : navigator;
  return {
    hostname: typeof location === "undefined" ? "" : location.hostname,
    userAgent: nav?.userAgent ?? "",
    maxTouchPoints: nav?.maxTouchPoints ?? 0,
  };
}

/**
 * True when the page is served from this machine itself. Checked in the
 * browser because the server cannot tell: tunnel traffic reaches it from
 * loopback too.
 */
export function isLoopbackHostname(hostname: string): boolean {
  const host = hostname.toLowerCase().replace(/^\[|\]$/g, "");
  return host === "localhost" || host.endsWith(".localhost") || host === "::1" || /^127(?:\.\d{1,3}){3}$/.test(host);
}

/**
 * The platform whose OS shows its own source chooser for a plain file input,
 * or null. This asks about OS behavior, which no feature test reveals, so it
 * reads the user agent. iPadOS reports itself as a Mac; touch tells them
 * apart. A miss only costs the desktop menu, which still works.
 */
export function nativeChooserPlatform(env: AttachEnvironment): NativeChooserPlatform | null {
  const ua = env.userAgent;
  if (/iPhone|iPod/i.test(ua)) return "iphone";
  if (/iPad/i.test(ua) || (/Macintosh/i.test(ua) && env.maxTouchPoints > 1)) return "ipad";
  if (/Android/i.test(ua)) return "android";
  return null;
}

/** The ordered sources the attach menu offers in this environment. */
export function planAttachSources(
  env: AttachEnvironment,
  options: { hostAvailable: boolean; cameraAvailable: boolean },
): AttachSourceId[] {
  const local: AttachSourceId[] = nativeChooserPlatform(env)
    ? ["device"]
    : options.cameraAvailable ? ["upload", "camera"] : ["upload"];
  if (!options.hostAvailable) return local;
  // On the host itself both sources reach the same disk; browsing comes first
  // because it attaches the real path, where the file dialog can only copy.
  return isLoopbackHostname(env.hostname) ? ["host", ...local] : [...local, "host"];
}
