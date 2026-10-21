export {};

declare global {
  interface DeviceStudioDesktopLinkStatus {
    state: "linked" | "unlinked" | "expired" | "revoked" | "unavailable";
    localPrincipal?: string;
    businessAccountId?: string;
    scopes: string[];
    message?: string;
  }
  interface DeviceStudioCreditSessionStatus {
    state: "shared" | "not_shared" | "signed_out" | "unavailable";
    localCleared?: boolean;
    remoteRevocation?: "uncertain";
  }

  interface Window {
    desktop?: {
      auth?: {
        signIn?(): Promise<{ state: string }>;
        connectDesktop(options: { resource: string; audience: string; scopes: string[] }): Promise<DeviceStudioDesktopLinkStatus>;
        getDesktopLinkStatus(options: { resource: string }): Promise<DeviceStudioDesktopLinkStatus>;
        disconnectDesktop(options: { resource: string }): Promise<{ localCleared: boolean; remoteRevocation: "confirmed" | "uncertain" }>;
        getCreditSessionShareStatus?(): Promise<DeviceStudioCreditSessionStatus>;
        setCreditSessionSharing?(enabled: boolean): Promise<DeviceStudioCreditSessionStatus>;
      };
    };
  }
}
