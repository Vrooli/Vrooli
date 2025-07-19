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
    },
}));

describe("CompletionInstaller", () => {
    let installer: CompletionInstaller;
    let consoleSpy: ReturnType<typeof vi.spyOn>;

    beforeEach(() => {
        vi.clearAllMocks();
        installer = new CompletionInstaller();
        consoleSpy = vi.spyOn(console, "log").mockImplementation(() => { });
    });

    afterEach(() => {
        consoleSpy.mockRestore();
    });

    describe("install", () => {
        it("should install bash completions when shell is bash", async () => {
            vi.spyOn(installer as any, "detectShell").mockReturnValue("bash");
            vi.spyOn(installer as any, "getInstallPath").mockReturnValue("/path/to/bash_completion");
            vi.spyOn(installer as any, "ensureDirectoryExists").mockResolvedValue(undefined);
            vi.spyOn(installer as any, "showShellInstructions").mockImplementation(() => { });

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
});