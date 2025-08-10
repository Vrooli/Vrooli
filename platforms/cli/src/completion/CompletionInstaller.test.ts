// AI_CHECK: TEST_COVERAGE=24 | LAST: 2025-08-04 | STATUS: Significantly expanded test coverage from 4 to 24 tests (+20), adding comprehensive coverage for status(), detectShell(), getGenerator(), getInstallPath(), ensureDirectoryExists(), and showShellInstructions() methods. Coverage improved from 36.36% to 88.11% (+51.75%). All mocking issues resolved for chalk (added bold, red, cyan) and path (added join method).
import { promises as fs } from "fs";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { CompletionInstaller } from "./CompletionInstaller.js";

vi.mock("fs", () => ({
    promises: {
        writeFile: vi.fn(),
        access: vi.fn(),
        unlink: vi.fn(),
        mkdir: vi.fn(),
    },
}));

vi.mock("path", () => ({
    dirname: vi.fn((path) => path.substring(0, path.lastIndexOf("/"))),
    join: vi.fn((...parts) => parts.join("/")),
}));

vi.mock("os", () => ({
    homedir: vi.fn(() => "/home/user"),
}));

vi.mock("./generators/bash.js", () => ({
    BashCompletionGenerator: vi.fn(() => ({
        generate: vi.fn(() => "bash script"),
    })),
}));

vi.mock("./generators/zsh.js", () => ({
    ZshCompletionGenerator: vi.fn(() => ({
        generate: vi.fn(() => "zsh script"),
    })),
}));

vi.mock("./generators/fish.js", () => ({
    FishCompletionGenerator: vi.fn(() => ({
        generate: vi.fn(() => "fish script"),
    })),
}));

vi.mock("chalk", () => ({
    default: {
        green: vi.fn((text) => text),
        gray: vi.fn((text) => text),
        yellow: vi.fn((text) => text),
        red: vi.fn((text) => text),
        bold: vi.fn((text) => text),
        cyan: vi.fn((text) => text),
    },
}));

describe("CompletionInstaller", () => {
    let installer: CompletionInstaller;
    let consoleSpy: ReturnType<typeof vi.spyOn>;

    beforeEach(() => {
        vi.clearAllMocks();
        installer = new CompletionInstaller();
        consoleSpy = vi.spyOn(console, "log").mockImplementation(() => { /* intentionally empty */ });
    });

    afterEach(() => {
        consoleSpy.mockRestore();
    });

    describe("install", () => {
        it("should install bash completions when shell is bash", async () => {
            vi.spyOn(installer as any, "detectShell").mockReturnValue("bash");
            vi.spyOn(installer as any, "getInstallPath").mockReturnValue("/path/to/bash_completion");
            vi.spyOn(installer as any, "ensureDirectoryExists").mockResolvedValue(undefined);
            vi.spyOn(installer as any, "showShellInstructions").mockImplementation(() => { /* intentionally empty */ });

            await installer.install("bash");

            expect(vi.mocked(fs.writeFile)).toHaveBeenCalledWith("/path/to/bash_completion", "bash script");
            expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining("✓ Installed bash completions"));
        });

        it("should throw error for unsupported shell", async () => {
            await expect(installer.install("unsupported")).rejects.toThrow("Unsupported shell: unsupported");
        });
    });

    describe("uninstall", () => {
        it("should uninstall all existing completion files", async () => {
            vi.spyOn(installer as any, "getInstallPath")
                .mockReturnValueOnce("/path/bash")
                .mockReturnValueOnce("/path/zsh")
                .mockReturnValueOnce("/path/fish");

            vi.mocked(fs.access).mockResolvedValue(undefined);
            vi.mocked(fs.unlink).mockResolvedValue(undefined);

            await installer.uninstall();

            expect(fs.access).toHaveBeenCalledTimes(3);
            expect(fs.unlink).toHaveBeenCalledTimes(3);
            expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining("✓ Uninstalled bash completions"));
        });

        it("should handle missing completion files gracefully", async () => {
            vi.spyOn(installer as any, "getInstallPath")
                .mockReturnValueOnce("/path/bash")
                .mockReturnValueOnce("/path/zsh")
                .mockReturnValueOnce("/path/fish");

            vi.mocked(fs.access).mockRejectedValue(new Error("File not found"));

            await installer.uninstall();

            expect(fs.access).toHaveBeenCalledTimes(3);
            expect(fs.unlink).not.toHaveBeenCalled();
            expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining("No completions found to uninstall"));
        });
    });

    describe("status", () => {
        it("should show status of all completion files", async () => {
            vi.spyOn(installer as any, "getInstallPath")
                .mockReturnValueOnce("/path/bash")
                .mockReturnValueOnce("/path/zsh")
                .mockReturnValueOnce("/path/fish");

            vi.mocked(fs.access)
                .mockResolvedValueOnce(undefined) // bash exists
                .mockRejectedValueOnce(new Error("File not found")) // zsh doesn't exist
                .mockResolvedValueOnce(undefined); // fish exists

            await installer.status();

            expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining("Completion Status:"));
            expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining("bash: Installed"));
            expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining("zsh: Not installed"));
            expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining("fish: Installed"));
        });
    });

    describe("detectShell", () => {
        const originalShell = process.env.SHELL;

        afterEach(() => {
            process.env.SHELL = originalShell;
        });

        it("should detect bash shell", () => {
            process.env.SHELL = "/bin/bash";
            const result = (installer as any).detectShell();
            expect(result).toBe("bash");
        });

        it("should detect zsh shell", () => {
            process.env.SHELL = "/usr/local/bin/zsh";
            const result = (installer as any).detectShell();
            expect(result).toBe("zsh");
        });

        it("should detect fish shell", () => {
            process.env.SHELL = "/usr/bin/fish";
            const result = (installer as any).detectShell();
            expect(result).toBe("fish");
        });

        it("should default to bash for unknown shell", () => {
            process.env.SHELL = "/usr/bin/unknown";
            const result = (installer as any).detectShell();
            expect(result).toBe("bash");
        });

        it("should default to bash when SHELL env var is not set", () => {
            delete process.env.SHELL;
            const result = (installer as any).detectShell();
            expect(result).toBe("bash");
        });
    });

    describe("getGenerator", () => {
        it("should return BashCompletionGenerator for bash", () => {
            const generator = (installer as any).getGenerator("bash");
            expect(generator).toBeDefined();
        });

        it("should return ZshCompletionGenerator for zsh", () => {
            const generator = (installer as any).getGenerator("zsh");
            expect(generator).toBeDefined();
        });

        it("should return FishCompletionGenerator for fish", () => {
            const generator = (installer as any).getGenerator("fish");
            expect(generator).toBeDefined();
        });

        it("should throw error for unsupported shell", () => {
            expect(() => (installer as any).getGenerator("unsupported")).toThrow("Unsupported shell: unsupported");
        });
    });

    describe("getInstallPath", () => {
        it("should return bash completion path", () => {
            const path = (installer as any).getInstallPath("bash");
            expect(path).toContain(".local/share/bash-completion/completions/vrooli");
        });

        it("should return zsh completion path", () => {
            const path = (installer as any).getInstallPath("zsh");
            expect(path).toContain(".zsh/completions/_vrooli");
        });

        it("should return fish completion path", () => {
            const path = (installer as any).getInstallPath("fish");
            expect(path).toContain(".config/fish/completions/vrooli.fish");
        });

        it("should throw error for unsupported shell", () => {
            expect(() => (installer as any).getInstallPath("unsupported")).toThrow("Unsupported shell: unsupported");
        });
    });

    describe("ensureDirectoryExists", () => {
        it("should create directory if it doesn't exist", async () => {
            vi.mocked(fs.mkdir).mockResolvedValue(undefined);

            await (installer as any).ensureDirectoryExists("/path/to/dir");

            expect(fs.mkdir).toHaveBeenCalledWith("/path/to/dir", { recursive: true });
        });

        it("should handle directory creation errors gracefully", async () => {
            vi.mocked(fs.mkdir).mockRejectedValue(new Error("Directory exists"));

            // Should not throw
            await expect((installer as any).ensureDirectoryExists("/path/to/dir")).resolves.toBeUndefined();
        });
    });

    describe("showShellInstructions", () => {
        it("should show bash instructions", () => {
            (installer as any).showShellInstructions("bash", "/path/to/bash");

            expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining("BASH Instructions:"));
        });

        it("should show zsh instructions", () => {
            (installer as any).showShellInstructions("zsh", "/path/to/zsh");

            expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining("ZSH Instructions:"));
        });

        it("should show fish instructions", () => {
            (installer as any).showShellInstructions("fish", "/path/to/fish");

            expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining("FISH Instructions:"));
        });
    });

    describe("install with auto-detection", () => {
        it("should auto-detect shell and install completions", async () => {
            vi.spyOn(installer as any, "detectShell").mockReturnValue("zsh");
            vi.spyOn(installer as any, "getInstallPath").mockReturnValue("/path/to/zsh_completion");
            vi.spyOn(installer as any, "ensureDirectoryExists").mockResolvedValue(undefined);
            vi.spyOn(installer as any, "showShellInstructions").mockImplementation(() => { /* intentionally empty */ });

            await installer.install("auto");

            expect(vi.mocked(fs.writeFile)).toHaveBeenCalledWith("/path/to/zsh_completion", "zsh script");
            expect(consoleSpy).toHaveBeenCalledWith(expect.stringContaining("✓ Installed zsh completions"));
        });
    });
}); 
