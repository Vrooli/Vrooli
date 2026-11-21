/** Authority is the window/frame and renderer origin assigned by the shell. */
export interface RendererSender {
    sender: unknown;
    senderFrame?: { url: string } | null;
}
export interface OwnedRenderer {
    sender: unknown;
    mainFrame: { url: string };
    url: string;
}

export function isOwnedRendererURL(expected: string, candidate: string): boolean {
    try {
        const owner = new URL(expected);
        const target = new URL(candidate);
        if (target.username || target.password || owner.protocol !== target.protocol) return false;
        if (owner.protocol === "file:") {
            owner.hash = "";
            target.hash = "";
            return owner.href === target.href;
        }
        if (owner.protocol !== "http:" && owner.protocol !== "https:" && owner.protocol !== "vrooli-desktop:") return false;
        return owner.host === target.host;
    } catch { return false; }
}

export function createRendererAuthority(owned: () => OwnedRenderer | null): (event: RendererSender) => boolean {
    return (event) => {
        const renderer = owned();
        return Boolean(renderer && event.sender === renderer.sender && event.senderFrame === renderer.mainFrame &&
            event.senderFrame && isOwnedRendererURL(renderer.url, event.senderFrame.url));
    };
}

interface NavigationEvent { preventDefault(): void; url?: string; isMainFrame?: boolean }
interface NavigationContents { on(event: string, listener: (event: NavigationEvent, url?: string) => void): unknown }

export function protectRendererNavigation(contents: NavigationContents, rendererURL: string): void {
    const guard = (event: NavigationEvent, url?: string) => {
        if (event.isMainFrame === false) return;
        if (!isOwnedRendererURL(rendererURL, url ?? event.url ?? "")) event.preventDefault();
    };
    contents.on("will-navigate", guard);
    contents.on("will-redirect", guard);
    contents.on("will-frame-navigate", guard);
}

export function isExternalBrowserURL(raw: string): boolean {
    try { const url = new URL(raw); return (url.protocol === "https:" || url.protocol === "http:") && !url.username && !url.password; }
    catch { return false; }
}
