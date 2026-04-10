/**
 * Filesystem Mocks
 *
 * DOC: docs/internal/SEAMS.md#filesystem-mocks
 *
 * Provides mock implementations for filesystem operations.
 * These mocks allow testing file-based modules without actual disk access.
 */

import { vi, type Mock } from "vitest";

/**
 * Mock filesystem interface matching the IFileSystem seam used in modules.
 */
export interface MockFileSystem {
    readFile: Mock;
    writeFile: Mock;
    exists: Mock;
    mkdir: Mock;
    unlink: Mock;
    rm: Mock;
    readdir: Mock;
    stat: Mock;
    access: Mock;
    appendFile: Mock;

    // Internal state for testing
    _files: Map<string, string | Buffer>;
    _directories: Set<string>;
    _setFile: (path: string, content: string | Buffer) => void;
    _getFile: (path: string) => string | Buffer | undefined;
    _clear: () => void;
}

/**
 * Create a mock filesystem with in-memory storage.
 */
export function createMockFs(initialFiles?: Record<string, string | Buffer>): MockFileSystem {
    const files = new Map<string, string | Buffer>();
    const directories = new Set<string>();

    // Initialize with provided files
    if (initialFiles) {
        for (const [path, content] of Object.entries(initialFiles)) {
            files.set(path, content);
            // Add parent directories
            const parts = path.split("/");
            for (let i = 1; i < parts.length; i++) {
                directories.add(parts.slice(0, i).join("/"));
            }
        }
    }

    // Helper to get directory contents
    const getDirContents = (dirPath: string): Array<{ name: string; isDirectory: boolean; isFile: boolean }> => {
        const normalizedDir = dirPath.endsWith("/") ? dirPath.slice(0, -1) : dirPath;
        const entries: Array<{ name: string; isDirectory: boolean; isFile: boolean }> = [];
        const seen = new Set<string>();

        // Check files
        for (const filePath of files.keys()) {
            if (filePath.startsWith(normalizedDir + "/")) {
                const relativePath = filePath.slice(normalizedDir.length + 1);
                const namePart = relativePath.split("/")[0];
                if (namePart !== undefined && !seen.has(namePart)) {
                    seen.add(namePart);
                    const isDir = relativePath.includes("/");
                    entries.push({ name: namePart, isDirectory: isDir, isFile: !isDir });
                }
            }
        }

        // Check directories
        for (const dir of directories) {
            if (dir.startsWith(normalizedDir + "/")) {
                const relativePath = dir.slice(normalizedDir.length + 1);
                const namePart = relativePath.split("/")[0];
                if (namePart !== undefined && !seen.has(namePart)) {
                    seen.add(namePart);
                    entries.push({ name: namePart, isDirectory: true, isFile: false });
                }
            }
        }

        return entries;
    };

    const mockFs: MockFileSystem = {
        readFile: vi.fn(async (path: string, encoding?: string) => {
            const content = files.get(path);
            if (content === undefined) {
                const error = new Error(`ENOENT: no such file or directory, open '${path}'`);
                (error as NodeJS.ErrnoException).code = "ENOENT";
                throw error;
            }
            if (encoding === "utf-8" || encoding === "utf8") {
                return typeof content === "string" ? content : content.toString("utf-8");
            }
            return typeof content === "string" ? Buffer.from(content) : content;
        }),

        writeFile: vi.fn(async (path: string, content: string | Buffer) => {
            files.set(path, content);
            // Add parent directories
            const parts = path.split("/");
            for (let i = 1; i < parts.length; i++) {
                directories.add(parts.slice(0, i).join("/"));
            }
        }),

        exists: vi.fn(async (path: string) => {
            return files.has(path) || directories.has(path);
        }),

        mkdir: vi.fn(async (path: string, _options?: { recursive?: boolean }) => {
            directories.add(path);
            // Add parent directories
            const parts = path.split("/");
            for (let i = 1; i < parts.length; i++) {
                directories.add(parts.slice(0, i).join("/"));
            }
        }),

        unlink: vi.fn(async (path: string) => {
            if (!files.has(path)) {
                const error = new Error(`ENOENT: no such file or directory, unlink '${path}'`);
                (error as NodeJS.ErrnoException).code = "ENOENT";
                throw error;
            }
            files.delete(path);
        }),

        rm: vi.fn(async (path: string, _options?: { recursive?: boolean; force?: boolean }) => {
            // Remove file
            files.delete(path);
            // Remove directory and all contents
            directories.delete(path);
            for (const filePath of files.keys()) {
                if (filePath.startsWith(path + "/")) {
                    files.delete(filePath);
                }
            }
            for (const dir of directories) {
                if (dir.startsWith(path + "/")) {
                    directories.delete(dir);
                }
            }
        }),

        readdir: vi.fn(async (path: string, options?: { withFileTypes?: boolean }) => {
            if (!directories.has(path) && !files.has(path)) {
                const error = new Error(`ENOENT: no such file or directory, scandir '${path}'`);
                (error as NodeJS.ErrnoException).code = "ENOENT";
                throw error;
            }

            const contents = getDirContents(path);

            if (options?.withFileTypes) {
                return contents.map((entry) => ({
                    name: entry.name,
                    isDirectory: () => entry.isDirectory,
                    isFile: () => entry.isFile,
                    isSymbolicLink: () => false,
                }));
            }

            return contents.map((entry) => entry.name);
        }),

        stat: vi.fn(async (path: string) => {
            if (files.has(path)) {
                const content = files.get(path)!;
                return {
                    size: typeof content === "string" ? content.length : content.length,
                    isFile: () => true,
                    isDirectory: () => false,
                    birthtimeMs: Date.now(),
                    mtimeMs: Date.now(),
                };
            }
            if (directories.has(path)) {
                return {
                    size: 0,
                    isFile: () => false,
                    isDirectory: () => true,
                    birthtimeMs: Date.now(),
                    mtimeMs: Date.now(),
                };
            }
            const error = new Error(`ENOENT: no such file or directory, stat '${path}'`);
            (error as NodeJS.ErrnoException).code = "ENOENT";
            throw error;
        }),

        access: vi.fn(async (path: string) => {
            if (!files.has(path) && !directories.has(path)) {
                const error = new Error(`ENOENT: no such file or directory, access '${path}'`);
                (error as NodeJS.ErrnoException).code = "ENOENT";
                throw error;
            }
        }),

        appendFile: vi.fn(async (path: string, content: string) => {
            const existing = files.get(path) ?? "";
            const existingStr = typeof existing === "string" ? existing : existing.toString();
            files.set(path, existingStr + content);
        }),

        // Internal helpers
        _files: files,
        _directories: directories,
        _setFile: (path: string, content: string | Buffer) => {
            files.set(path, content);
        },
        _getFile: (path: string) => files.get(path),
        _clear: () => {
            files.clear();
            directories.clear();
        },
    };

    return mockFs;
}

/**
 * Create a mock Node.js fs module with promises interface.
 * This matches the shape of `import * as fs from "node:fs"`.
 */
export function createMockNodeFs(initialFiles?: Record<string, string | Buffer>) {
    const mockFs = createMockFs(initialFiles);

    return {
        promises: {
            readFile: mockFs.readFile,
            writeFile: mockFs.writeFile,
            mkdir: mockFs.mkdir,
            unlink: mockFs.unlink,
            rm: mockFs.rm,
            readdir: mockFs.readdir,
            stat: mockFs.stat,
            access: mockFs.access,
            appendFile: mockFs.appendFile,
        },
        // Sync versions (rarely used, but available)
        readFileSync: vi.fn((path: string, encoding?: string) => {
            const content = mockFs._files.get(path);
            if (content === undefined) {
                const error = new Error(`ENOENT: no such file or directory, open '${path}'`);
                (error as NodeJS.ErrnoException).code = "ENOENT";
                throw error;
            }
            if (encoding === "utf-8" || encoding === "utf8") {
                return typeof content === "string" ? content : content.toString("utf-8");
            }
            return typeof content === "string" ? Buffer.from(content) : content;
        }),
        existsSync: vi.fn((path: string) => mockFs._files.has(path) || mockFs._directories.has(path)),
        // Expose internal state
        _mock: mockFs,
    };
}

/**
 * Create a mock path module.
 */
export function createMockPath() {
    return {
        join: vi.fn((...segments: string[]) => segments.filter(Boolean).join("/").replace(/\/+/g, "/")),
        resolve: vi.fn((...segments: string[]) => {
            const result = segments.filter(Boolean).join("/").replace(/\/+/g, "/");
            return result.startsWith("/") ? result : "/" + result;
        }),
        dirname: vi.fn((p: string) => {
            const parts = p.split("/");
            parts.pop();
            return parts.join("/") || "/";
        }),
        basename: vi.fn((p: string, ext?: string) => {
            const base = p.split("/").pop() || "";
            if (ext && base.endsWith(ext)) {
                return base.slice(0, -ext.length);
            }
            return base;
        }),
        extname: vi.fn((p: string) => {
            const base = p.split("/").pop() || "";
            const dotIndex = base.lastIndexOf(".");
            return dotIndex > 0 ? base.slice(dotIndex) : "";
        }),
        isAbsolute: vi.fn((p: string) => p.startsWith("/")),
        normalize: vi.fn((p: string) => {
            const parts = p.split("/").filter((part) => part !== "." && part !== "");
            const result: string[] = [];
            for (const part of parts) {
                if (part === "..") {
                    result.pop();
                } else {
                    result.push(part);
                }
            }
            return (p.startsWith("/") ? "/" : "") + result.join("/");
        }),
        relative: vi.fn((from: string, to: string) => {
            const fromParts = from.split("/").filter(Boolean);
            const toParts = to.split("/").filter(Boolean);

            let commonLength = 0;
            while (
                commonLength < fromParts.length &&
                commonLength < toParts.length &&
                fromParts[commonLength] === toParts[commonLength]
            ) {
                commonLength++;
            }

            const upCount = fromParts.length - commonLength;
            const relativeParts = [...Array(upCount).fill(".."), ...toParts.slice(commonLength)];
            return relativeParts.join("/") || ".";
        }),
        sep: "/",
    };
}
