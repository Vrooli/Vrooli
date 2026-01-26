declare module 'ws' {
  const WebSocket: unknown;
  export default WebSocket;
}

declare module '@ghostery/adblocker-playwright' {
  import type { Page as RebrowserPage } from 'rebrowser-playwright';

  export class PlaywrightBlocker {
    static fromPrebuiltAdsOnly(fetchFn: typeof fetch): Promise<PlaywrightBlocker>;
    static fromPrebuiltAdsAndTracking(fetchFn: typeof fetch): Promise<PlaywrightBlocker>;
    enableBlockingInPage(page: RebrowserPage): Promise<void>;
  }
}
