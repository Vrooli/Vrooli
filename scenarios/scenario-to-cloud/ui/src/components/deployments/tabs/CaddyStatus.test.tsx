import "@testing-library/jest-dom";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import type { CaddyState, DNSCheckResponse, TLSInfoResponse } from "../../../lib/api";

const hooks = vi.hoisted(() => ({
  useDNSCheck: vi.fn(),
  useCaddyControl: vi.fn(),
  useTLSInfo: vi.fn(),
  useTLSRenew: vi.fn(),
}));

vi.mock("../../../hooks/useLiveState", async () => {
  const actual = await vi.importActual<typeof import("../../../hooks/useLiveState")>("../../../hooks/useLiveState");
  return { ...actual, ...hooks };
});

import { CaddyStatus } from "./CaddyStatus";
import { renderWithProviders } from "../../../test-utils/renderWithProviders";

const caddy: CaddyState = {
  running: true,
  domain: "app.example.com",
  tls: {
    valid: true,
    issuer: "Let's Encrypt",
    expires: "2026-09-30",
    days_remaining: 45,
    alpn: { status: "pass", message: "TLS-ALPN challenge ready", protocol: "h2" },
  },
  routes: [
    { path: "/", upstream: "127.0.0.1:3000" },
    { path: "/api", upstream: "127.0.0.1:8080" },
  ],
};

const dns: DNSCheckResponse = {
  ok: true,
  vps_host: "203.0.113.10",
  vps_ips: ["203.0.113.10"],
  domains: [
    {
      domain: "app.example.com",
      role: "apex",
      ok: true,
      domain_ips: ["203.0.113.10"],
      points_to_vps: true,
      proxied: true,
      message: "Domain points to VPS",
    },
  ],
  message: "DNS checks passed",
  timestamp: "2026-08-14T12:00:00.000Z",
};

const tlsInfo: TLSInfoResponse = {
  ok: true,
  domain: "app.example.com",
  valid: true,
  validation: "full",
  issuer: "Let's Encrypt",
  subject: "app.example.com",
  not_before: "2026-07-01",
  not_after: "2026-09-30",
  days_remaining: 45,
  serial_number: "serial-1",
  sans: ["app.example.com", "www.example.com"],
  alpn: { status: "pass", message: "ALPN ready", protocol: "h2" },
  timestamp: "2026-08-14T12:00:00.000Z",
};

function setup(options?: {
  caddyState?: CaddyState;
  dnsResult?: DNSCheckResponse | null;
  dnsLoading?: boolean;
  tlsResult?: TLSInfoResponse | null;
  tlsLoading?: boolean;
  caddyControl?: { mutateAsync?: ReturnType<typeof vi.fn>; isPending?: boolean; data?: { ok: boolean; message: string; output?: string } };
  tlsRenew?: { mutateAsync: ReturnType<typeof vi.fn>; isPending: boolean; data?: { ok: boolean; message: string } };
}) {
  const caddyControl = {
    mutateAsync: vi.fn().mockResolvedValue({ ok: true, action: "restart", message: "Caddy restarted" }),
    isPending: false,
    data: undefined,
    ...options?.caddyControl,
  };
  const tlsRenewControl = options?.tlsRenew ?? {
    mutateAsync: vi.fn().mockResolvedValue({ ok: true, message: "Certificate renewed" }),
    isPending: false,
    data: undefined,
  };
  const refetchDNS = vi.fn();
  const refetchTLS = vi.fn();
  hooks.useDNSCheck.mockReturnValue({ data: options?.dnsResult === null ? undefined : options?.dnsResult ?? dns, isLoading: options?.dnsLoading ?? false, refetch: refetchDNS });
  hooks.useCaddyControl.mockReturnValue(caddyControl);
  hooks.useTLSInfo.mockReturnValue({ data: options?.tlsResult === null ? undefined : options?.tlsResult ?? tlsInfo, isLoading: options?.tlsLoading ?? false, refetch: refetchTLS });
  hooks.useTLSRenew.mockReturnValue(tlsRenewControl);
  renderWithProviders(<CaddyStatus caddy={options?.caddyState ?? caddy} deploymentId="deployment-1" />);
  return { caddyControl, tlsRenewControl, refetchDNS, refetchTLS };
}

describe("CaddyStatus", () => {
  beforeEach(() => vi.clearAllMocks());

  it("controls a running Caddy service and refreshes DNS/TLS details", async () => {
    const { caddyControl, tlsRenewControl, refetchDNS, refetchTLS } = setup({
      caddyControl: { data: { ok: true, message: "Caddy restarted", output: "reload complete" } },
      tlsRenew: { mutateAsync: vi.fn().mockResolvedValue({ ok: true }), isPending: false, data: { ok: true, message: "Certificate renewed" } },
    });

    expect(screen.getByText("Running")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Visit Site/ })).toHaveAttribute("href", "https://app.example.com");
    expect(screen.getByText("DNS checks passed")).toBeInTheDocument();
    expect(screen.getByText("Validation: full (chain + hostname verified)")).toBeInTheDocument();
    expect(screen.getAllByText("app.example.com").length).toBeGreaterThan(0);
    expect(screen.getByText("Certificate renewed")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Restart" }));
    fireEvent.click(screen.getByRole("button", { name: "Reload" }));
    fireEvent.click(screen.getByRole("button", { name: "Stop" }));
    fireEvent.click(screen.getByRole("button", { name: "Renew" }));
    fireEvent.click(screen.getByRole("button", { name: "Refresh DNS status" }));
    fireEvent.click(screen.getByRole("button", { name: "Refresh TLS status" }));

    await waitFor(() => {
      expect(caddyControl.mutateAsync).toHaveBeenNthCalledWith(1, "restart");
      expect(caddyControl.mutateAsync).toHaveBeenNthCalledWith(2, "reload");
      expect(caddyControl.mutateAsync).toHaveBeenNthCalledWith(3, "stop");
      expect(tlsRenewControl.mutateAsync).toHaveBeenCalledOnce();
    });
    expect(refetchDNS).toHaveBeenCalledOnce();
    expect(refetchTLS).toHaveBeenCalledOnce();
  });

  it("shows DNS remediation hints, route collapse, and stopped fallback TLS state", () => {
    const warningDNS: DNSCheckResponse = {
      ...dns,
      ok: false,
      vps_ips: [],
      domains: [{
        domain: "app.example.com",
        role: "apex",
        ok: false,
        proxied: false,
        domain_ips: ["198.51.100.20"],
        points_to_vps: false,
        message: "Domain does not point to VPS",
        hint: "Create an A record pointing to 203.0.113.10",
      }],
    };
    const stopped: CaddyState = { ...caddy, running: false, domain: "", routes: [], tls: { valid: false } };
    const { caddyControl } = setup({ caddyState: stopped, dnsResult: warningDNS, tlsResult: null });

    expect(screen.getByText("Stopped")).toBeInTheDocument();
    expect(screen.getByText("Not configured")).toBeInTheDocument();
    expect(screen.getByText("DNS checks need attention")).toBeInTheDocument();
    expect(screen.getByText("Domain does not point to VPS")).toBeInTheDocument();
    expect(screen.getByText("Certificate not valid")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Renew" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Start" })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /Visit Site/ })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Show how to fix/ }));
    expect(screen.getByText("Create an A record pointing to 203.0.113.10")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Hide how to fix/ }));
    expect(screen.queryByText("Create an A record pointing to 203.0.113.10")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Start" }));
    expect(caddyControl.mutateAsync).toHaveBeenCalledWith("start");
  });

  it("covers loading and detailed TLS warning branches", () => {
    const warningTLS: TLSInfoResponse = {
      ...tlsInfo,
      valid: true,
      validation: "time_only",
      days_remaining: 10,
      issuer: undefined,
      not_before: undefined,
      not_after: undefined,
      sans: [],
      alpn: { status: "warn", message: "ALPN needs attention", hint: "Open port 443", protocol: undefined },
    };
    setup({ dnsResult: undefined, dnsLoading: true, tlsResult: warningTLS });
    expect(screen.getByText("Checking DNS...")).toBeInTheDocument();
    expect(screen.getByText("Validation: time-only (chain + hostname not verified)")).toBeInTheDocument();
    expect(screen.getByText("Check")).toBeInTheDocument();
    expect(screen.getByText("ALPN needs attention")).toBeInTheDocument();
    expect(screen.getByText("Open port 443")).toBeInTheDocument();

    cleanup();
    setup({ dnsResult: null, tlsResult: null, tlsLoading: true });
    expect(screen.getByText("Checking certificate...")).toBeInTheDocument();
  });

  it("covers fallback TLS details, route collapse, and alternate DNS roles", () => {
    const fallback: CaddyState = {
      ...caddy,
      domain: "",
      routes: [{ path: "/", upstream: "127.0.0.1:3000" }],
      tls: { valid: true, days_remaining: 5, alpn: { status: "warn", message: "ALPN check" } },
    };
    const alternateDNS: DNSCheckResponse = {
      ...dns,
      vps_ips: [],
      domains: [
        { domain: "www.example.com", role: "www", ok: true, domain_ips: [], points_to_vps: true, proxied: false, message: "WWW ok" },
        { domain: "origin.example.com", role: "origin", ok: true, domain_ips: [], points_to_vps: true, proxied: false, message: "Origin ok" },
      ],
    };
    setup({ caddyState: fallback, dnsResult: alternateDNS, tlsResult: null });
    expect(screen.getByText("Valid certificate")).toBeInTheDocument();
    expect(screen.getByText("5")).toBeInTheDocument();
    expect(screen.getByText("WWW")).toBeInTheDocument();
    expect(screen.getByText("Origin")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Routes \(1\)/ }));
    expect(screen.queryByText("127.0.0.1:3000")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Routes \(1\)/ }));
    expect(screen.getByText("127.0.0.1:3000")).toBeInTheDocument();
  });

  it("renders detailed TLS error, invalid, and expiry bands", () => {
    setup({ tlsResult: { ...tlsInfo, error: "TLS inspection unavailable" } });
    expect(screen.getByText("TLS inspection unavailable")).toBeInTheDocument();
    cleanup();
    setup({ tlsResult: { ...tlsInfo, valid: false, error: undefined } });
    expect(screen.getByText("Certificate expired or invalid")).toBeInTheDocument();
    cleanup();
    setup({ tlsResult: { ...tlsInfo, days_remaining: 20, validation: undefined, sans: [], alpn: undefined } });
    expect(screen.getByText("20")).toBeInTheDocument();
    cleanup();
    setup({ tlsResult: { ...tlsInfo, days_remaining: 40, validation: "other", alpn: { status: "pass", message: "Ready", error: "ignored" } } });
    expect(screen.getByText("40")).toBeInTheDocument();
  });
});
