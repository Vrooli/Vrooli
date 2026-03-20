declare module "@novnc/novnc/lib/rfb" {
  export default class RFB {
    constructor(target: HTMLElement, urlOrChannel: string | WebSocket, options?: {
      shared?: boolean;
      credentials?: { username?: string; password?: string; target?: string };
      wsProtocols?: string[];
    });

    scaleViewport: boolean;
    resizeSession: boolean;
    clipViewport: boolean;
    readonly capabilities: { power: boolean };

    disconnect(): void;
    sendCredentials(credentials: { username?: string; password?: string; target?: string }): void;
    sendKey(keysym: number, code: string | null, down?: boolean): void;
    sendCtrlAltDel(): void;
    focus(): void;
    blur(): void;
    machineShutdown(): void;
    machineReboot(): void;
    machineReset(): void;
    clipboardPasteFrom(text: string): void;

    addEventListener(type: "connect", listener: () => void): void;
    addEventListener(type: "disconnect", listener: (e: CustomEvent<{ clean: boolean }>) => void): void;
    addEventListener(type: "credentialsrequired", listener: (e: CustomEvent<{ types: string[] }>) => void): void;
    addEventListener(type: "securityfailure", listener: (e: CustomEvent<{ status: number; reason: string }>) => void): void;
    addEventListener(type: "clipboard", listener: (e: CustomEvent<{ text: string }>) => void): void;
    addEventListener(type: "bell", listener: () => void): void;
    addEventListener(type: "desktopname", listener: (e: CustomEvent<{ name: string }>) => void): void;
    addEventListener(type: "capabilities", listener: (e: CustomEvent<{ capabilities: { power: boolean } }>) => void): void;
  }
}
