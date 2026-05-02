import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { GroupingRule } from "./FileList";
import { SettingsTabCredentials } from "./SettingsTabCredentials";
import { SettingsTabCredentialsSSH } from "./SettingsTabCredentialsSSH";
import { SettingsTabGrouping } from "./SettingsTabGrouping";
import { SettingsTabHealth } from "./SettingsTabHealth";
import { jsonResponse, renderWithQueryClient } from "../test-utils";

// AI_CHECK: GCT_TEST_ARCH=1 | LAST: 2026-05-01

function requestUrl(input: RequestInfo | URL) {
  if (input instanceof Request) return input.url;
  if (input instanceof URL) return input.toString();
  return input;
}

async function requestJson(init?: RequestInit) {
  if (typeof init?.body !== "string") return undefined;
  return JSON.parse(init.body) as unknown;
}

describe("Settings tab surfaces", () => {
  beforeEach(() => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    });
  });

  it("routes grouping rule edits through the supplied callbacks", () => {
    const onToggleGrouping = vi.fn();
    const onChangeRules = vi.fn();
    const rules: GroupingRule[] = [
      {
        id: "backend",
        label: "Backend",
        prefix: "api/",
        prefixes: ["api/"],
        mode: "prefix",
      },
    ];

    renderWithQueryClient(
      <SettingsTabGrouping
        groupingEnabled
        onToggleGrouping={onToggleGrouping}
        rules={rules}
        onChangeRules={onChangeRules}
        isMobile={false}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "On" }));
    expect(onToggleGrouping).toHaveBeenCalledOnce();

    fireEvent.change(screen.getByDisplayValue("Backend"), {
      target: { value: "API" },
    });
    expect(onChangeRules).toHaveBeenLastCalledWith([
      expect.objectContaining({ id: "backend", label: "API" }),
    ]);

    fireEvent.change(screen.getByDisplayValue("Prefix"), {
      target: { value: "segment" },
    });
    expect(onChangeRules).toHaveBeenLastCalledWith([
      expect.objectContaining({ id: "backend", mode: "segment" }),
    ]);

    fireEvent.change(screen.getByDisplayValue("api/"), {
      target: { value: "scenarios/" },
    });
    expect(onChangeRules).toHaveBeenLastCalledWith([
      expect.objectContaining({
        id: "backend",
        prefix: "scenarios/",
        prefixes: ["scenarios/"],
      }),
    ]);

    fireEvent.click(screen.getByRole("button", { name: /add prefix/i }));
    expect(onChangeRules).toHaveBeenLastCalledWith([
      expect.objectContaining({
        id: "backend",
        prefix: "api/",
        prefixes: ["api/", ""],
      }),
    ]);

    fireEvent.click(screen.getByRole("button", { name: "Remove group" }));
    expect(onChangeRules).toHaveBeenLastCalledWith([]);
  });

  it("renders gitignore health actions and sends repo-scoped move requests", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, _init?: RequestInit) => {
      const url = requestUrl(input);
      if (url.endsWith("/repo/grouping-rules")) {
        return jsonResponse({
          enabled: true,
          rules: [{ id: "api", label: "API", prefixes: ["api/"], mode: "prefix" }],
        });
      }
      if (url.endsWith("/repo/gitignore/move")) {
        return jsonResponse({ success: true });
      }
      return jsonResponse({
        root_entry_count: 4,
        suggestions: [
          {
            type: "single_group",
            line: 12,
            pattern: "api/tmp/",
            group_dir: "api/",
            group_label: "API",
            target_pattern: "tmp/",
            has_gitignore: false,
          },
          {
            type: "cross_group",
            line: 16,
            pattern: "*.log",
            group_dir: "",
            group_label: "workspace",
            target_pattern: "*.log",
            has_gitignore: true,
          },
        ],
      });
    });
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    renderWithQueryClient(<SettingsTabHealth isMobile={false} repoId="repo-1" />);

    expect(await screen.findByText(/1 entry could be moved/i)).toBeInTheDocument();
    expect(screen.getByText("api/tmp/")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Move" }));

    await waitFor(async () => {
      const moveCall = fetchMock.mock.calls.find(([input]) =>
        requestUrl(input).endsWith("/repo/gitignore/move"),
      );
      expect(moveCall).toBeDefined();
      expect(moveCall?.[1]).toEqual(
        expect.objectContaining({
          method: "POST",
          headers: expect.objectContaining({ "X-Repo-Id": "repo-1" }),
        }),
      );
      await expect(requestJson(moveCall?.[1])).resolves.toEqual({
        line: 12,
        pattern: "api/tmp/",
        group_dir: "api/",
        target_pattern: "tmp/",
      });
    });

    fireEvent.click(screen.getByTitle("Dismiss"));
    expect(window.localStorage.getItem("gct.gitignore.dismissals")).toContain("api/tmp/");
    expect(screen.getByText("1 dismissed")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /cross-group/i }));
    expect(screen.getByText("*.log")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /reset dismissals/i }));
    expect(window.localStorage.getItem("gct.gitignore.dismissals")).toBeNull();
  });

  it("saves HTTPS credentials, tests stored auth, and switches remote protocol", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = requestUrl(input);
      if (url.endsWith("/credentials") && init?.method === "POST") {
        return jsonResponse({ success: true });
      }
      if (url.endsWith("/credentials/test")) {
        return jsonResponse({
          success: true,
          authorized: true,
          reachable: true,
          message: "connected",
        });
      }
      if (url.endsWith("/repo/remote/url")) {
        return jsonResponse({
          success: true,
          url: "git@github.com:example/git-control-tower.git",
        });
      }
      return jsonResponse({
        credentials: [
          {
            remote: "origin",
            username: "octo",
            is_configured: true,
            token_masked: "ghp_****",
          },
        ],
      });
    });
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    renderWithQueryClient(
      <SettingsTabCredentials
        remoteUrl="https://github.com/example/git-control-tower.git"
        hasUpstream
        isMobile={false}
        repoId="repo-1"
      />,
    );

    expect(await screen.findByText(/authenticated/i)).toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText("GitHub username"), {
      target: { value: "octo" },
    });
    fireEvent.change(screen.getByPlaceholderText("ghp_xxxxxxxxxxxx"), {
      target: { value: "secret-token" },
    });
    fireEvent.click(screen.getByRole("button", { name: /save credentials/i }));

    await waitFor(async () => {
      const saveCall = fetchMock.mock.calls.find(([input, init]) =>
        requestUrl(input).endsWith("/credentials") && init?.method === "POST",
      );
      expect(saveCall).toBeDefined();
      expect(saveCall?.[1]).toEqual(
        expect.objectContaining({
          method: "POST",
          headers: expect.objectContaining({ "X-Repo-Id": "repo-1" }),
        }),
      );
      await expect(requestJson(saveCall?.[1])).resolves.toEqual({
        remote: "origin",
        username: "octo",
        token: "secret-token",
      });
    });

    fireEvent.click(screen.getByRole("button", { name: /test connection/i }));
    expect(await screen.findByText("Connection successful!")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /switch to ssh/i }));
    await waitFor(async () => {
      const switchCall = fetchMock.mock.calls.find(([input]) =>
        requestUrl(input).endsWith("/repo/remote/url"),
      );
      expect(switchCall).toBeDefined();
      await expect(requestJson(switchCall?.[1])).resolves.toEqual({
        remote: "origin",
        url: "git@github.com:example/git-control-tower.git",
      });
    });
  });

  it("manages SSH key selection, key material, testing, and generation", async () => {
    const onCredentialsSaved = vi.fn();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, _init?: RequestInit) => {
      const url = requestUrl(input);
      if (url.endsWith("/ssh/keys/public")) {
        return jsonResponse({ success: true, public_key: "ssh-ed25519 AAAA copied" });
      }
      if (url.endsWith("/ssh/keys/test")) {
        return jsonResponse({
          success: true,
          message: "Connected to GitHub",
          github_user: "octo",
        });
      }
      if (url.endsWith("/ssh/keys/generate")) {
        return jsonResponse({
          success: true,
          public_key: "ssh-rsa BBBB generated",
          key: { path: "/home/user/.ssh/github_rsa" },
        });
      }
      if (url.endsWith("/credentials")) {
        return jsonResponse({ success: true });
      }
      return jsonResponse({
        keys: [
          {
            path: "/home/user/.ssh/github_ed25519",
            filename: "github_ed25519",
            type: "ed25519",
            fingerprint: "SHA256:abc",
            comment: "octo@example.com",
            created_at: "2026-05-01T00:00:00Z",
            has_public: true,
          },
          {
            path: "/home/user/.ssh/github_rsa",
            filename: "github_rsa",
            type: "rsa",
            fingerprint: "SHA256:def",
            has_public: true,
          },
        ],
      });
    });
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    renderWithQueryClient(
      <SettingsTabCredentialsSSH
        isMobile={false}
        repoId="repo-1"
        inputClasses="test-input"
        buttonHeight="h-8"
        storedSSHKeyPath="/home/user/.ssh/github_ed25519"
        onCredentialsSaved={onCredentialsSaved}
      />,
    );

    fireEvent.click(await screen.findByText("github_ed25519"));
    expect(screen.getByText("SHA256:abc")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /copy public key/i }));
    await waitFor(() => {
      expect(navigator.clipboard.writeText).toHaveBeenCalledWith("ssh-ed25519 AAAA copied");
    });

    fireEvent.click(screen.getByRole("button", { name: /test connection/i }));
    expect(await screen.findByText("Connected to GitHub")).toBeInTheDocument();
    expect(screen.getByText("octo")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /save ssh key/i }));
    await waitFor(async () => {
      const saveCall = fetchMock.mock.calls.find(([input]) =>
        requestUrl(input).endsWith("/credentials"),
      );
      expect(saveCall).toBeDefined();
      await expect(requestJson(saveCall?.[1])).resolves.toEqual({
        remote: "origin",
        ssh_key_path: "/home/user/.ssh/github_ed25519",
      });
    });
    expect(onCredentialsSaved).toHaveBeenCalledOnce();

    fireEvent.click(screen.getByRole("button", { name: /generate new ssh key/i }));
    fireEvent.click(screen.getByRole("button", { name: /rsa/i }));
    fireEvent.change(screen.getByPlaceholderText("github_rsa"), {
      target: { value: "deploy_rsa" },
    });
    fireEvent.change(screen.getByPlaceholderText("your-email@example.com"), {
      target: { value: "deploy@example.com" },
    });
    fireEvent.click(screen.getByRole("button", { name: /^generate key$/i }));

    expect(await screen.findByText("New SSH Key Generated!")).toBeInTheDocument();
    expect(screen.getByText("ssh-rsa BBBB generated")).toBeInTheDocument();

    const generateCall = fetchMock.mock.calls.find(([input]) =>
      requestUrl(input).endsWith("/ssh/keys/generate"),
    );
    await expect(requestJson(generateCall?.[1])).resolves.toEqual({
      type: "rsa",
      filename: "deploy_rsa",
      comment: "deploy@example.com",
    });
  });
});
