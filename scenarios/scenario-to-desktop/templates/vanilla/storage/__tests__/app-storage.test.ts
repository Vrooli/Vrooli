/**
 * App Storage Tests
 *
 * DOC: docs/internal/SEAMS.md#app-storage-tests
 *
 * Tests for the app storage system with focus on:
 * - Path traversal attack prevention
 * - CRUD operations
 * - Error handling
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import type {
    IStorageFileSystem,
    IStoragePathUtils,
    StorageConfig,
    DirentLike,
    StatsLike,
} from "../types";
import { createAppStorage } from "../app-storage";

// ===== Mock Factories =====

interface MockDirent {
    name: string;
    _isDir: boolean;
    _isFile: boolean;
    isDirectory(): boolean;
    isFile(): boolean;
}

function createMockDirent(name: string, isDir: boolean): MockDirent {
    return {
        name,
        _isDir: isDir,
        _isFile: !isDir,
        isDirectory() { return this._isDir; },
        isFile() { return this._isFile; },
    };
}

interface MockStats {
    size: number;
    birthtimeMs: number;
    mtimeMs: number;
    _isDir: boolean;
    _isFile: boolean;
    isDirectory(): boolean;
    isFile(): boolean;
}

function createMockStats(size: number, isDir = false): MockStats {
    return {
        size,
        birthtimeMs: 1704067200000, // 2024-01-01
        mtimeMs: 1704153600000, // 2024-01-02
        _isDir: isDir,
        _isFile: !isDir,
        isDirectory() { return this._isDir; },
        isFile() { return this._isFile; },
    };
}

interface MockFileSystem extends IStorageFileSystem {
    _files: Map<string, string | Buffer>;
    _dirs: Set<string>;
}

function createMockFileSystem(): MockFileSystem {
    const files = new Map<string, string | Buffer>();
    const dirs = new Set<string>(["/mock/userData", "/mock/userData/app-storage"]);

    const fs: MockFileSystem = {
        _files: files,
        _dirs: dirs,

        readFile: vi.fn(async (path: string, encoding?: "utf-8"): Promise<Buffer | string> => {
            const content = files.get(path);
            if (content === undefined) {
                const error = new Error(`ENOENT: no such file: ${path}`);
                (error as NodeJS.ErrnoException).code = "ENOENT";
                throw error;
            }
            if (encoding === "utf-8") {
                return Buffer.isBuffer(content) ? content.toString("utf-8") : content;
            }
            return Buffer.isBuffer(content) ? content : Buffer.from(content);
        }) as unknown as IStorageFileSystem["readFile"],

        writeFile: vi.fn(async (path: string, data: string | Buffer) => {
            files.set(path, data);
        }),

        mkdir: vi.fn(async (path: string, _options?: { recursive?: boolean }) => {
            dirs.add(path);
        }),

        readdir: vi.fn(async (dirPath: string, _options: { withFileTypes: true }): Promise<DirentLike[]> => {
            if (!dirs.has(dirPath)) {
                const error = new Error(`ENOENT: no such directory: ${dirPath}`);
                (error as NodeJS.ErrnoException).code = "ENOENT";
                throw error;
            }

            const entries: DirentLike[] = [];
            const prefix = dirPath.endsWith("/") ? dirPath : dirPath + "/";

            // Find immediate children in files
            for (const filePath of files.keys()) {
                if (filePath.startsWith(prefix)) {
                    const relativePath = filePath.slice(prefix.length);
                    if (!relativePath.includes("/")) {
                        entries.push(createMockDirent(relativePath, false));
                    }
                }
            }

            // Find immediate children in dirs
            for (const childDir of dirs) {
                if (childDir.startsWith(prefix) && childDir !== dirPath) {
                    const relativePath = childDir.slice(prefix.length);
                    if (!relativePath.includes("/")) {
                        entries.push(createMockDirent(relativePath, true));
                    }
                }
            }

            return entries;
        }),

        unlink: vi.fn(async (path: string) => {
            if (!files.has(path)) {
                const error = new Error(`ENOENT: no such file: ${path}`);
                (error as NodeJS.ErrnoException).code = "ENOENT";
                throw error;
            }
            files.delete(path);
        }),

        rm: vi.fn(async (path: string, options?: { recursive?: boolean; force?: boolean }) => {
            const deleted = files.delete(path) || dirs.delete(path);
            if (!deleted && !options?.force) {
                const error = new Error(`ENOENT: no such file or directory: ${path}`);
                (error as NodeJS.ErrnoException).code = "ENOENT";
                throw error;
            }
        }),

        stat: vi.fn(async (path: string): Promise<StatsLike> => {
            if (files.has(path)) {
                const content = files.get(path);
                const size = Buffer.isBuffer(content) ? content.length : Buffer.from(content ?? "").length;
                return createMockStats(size, false);
            }
            if (dirs.has(path)) {
                return createMockStats(0, true);
            }
            const error = new Error(`ENOENT: no such file or directory: ${path}`);
            (error as NodeJS.ErrnoException).code = "ENOENT";
            throw error;
        }),

        access: vi.fn(async (path: string) => {
            if (!files.has(path) && !dirs.has(path)) {
                const error = new Error(`ENOENT: no such file or directory: ${path}`);
                (error as NodeJS.ErrnoException).code = "ENOENT";
                throw error;
            }
        }),
    };

    return fs;
}

function createMockPathUtils(): IStoragePathUtils {
    return {
        join: (...segments) => segments.join("/").replace(/\/+/g, "/"),
        dirname: (p) => {
            const parts = p.split("/");
            parts.pop();
            return parts.join("/") || "/";
        },
        normalize: (p) => {
            // Accurate normalization that preserves leading ".." for relative paths
            const isAbsolute = p.startsWith("/");
            const parts = p.split("/");
            const result: string[] = [];

            for (const part of parts) {
                if (part === "..") {
                    // Can only pop if we have non-".." parts to pop
                    if (result.length > 0 && result[result.length - 1] !== "..") {
                        result.pop();
                    } else if (!isAbsolute) {
                        // For relative paths, keep leading ".."
                        result.push("..");
                    }
                    // For absolute paths, ".." at root is ignored
                } else if (part !== "." && part !== "") {
                    result.push(part);
                }
            }

            if (isAbsolute) {
                return "/" + result.join("/");
            }
            return result.join("/") || ".";
        },
        resolve: (...segments) => {
            // Accurate resolve: join paths and resolve ".." properly
            let result = "";
            for (const segment of segments) {
                if (segment.startsWith("/")) {
                    result = segment;
                } else {
                    result = result + "/" + segment;
                }
            }
            // Normalize the absolute path
            const parts = result.split("/");
            const normalized: string[] = [];
            for (const part of parts) {
                if (part === "..") {
                    // Pop from normalized (but not below root)
                    if (normalized.length > 0) {
                        normalized.pop();
                    }
                } else if (part !== "." && part !== "") {
                    normalized.push(part);
                }
            }
            return "/" + normalized.join("/");
        },
        isAbsolute: (p) => p.startsWith("/"),
        relative: (from, to) => {
            // Simple relative: strip common prefix
            const fromParts = from.split("/").filter(Boolean);
            const toParts = to.split("/").filter(Boolean);
            let common = 0;
            while (common < fromParts.length && common < toParts.length && fromParts[common] === toParts[common]) {
                common++;
            }
            return toParts.slice(common).join("/");
        },
        sep: "/",
    };
}

function createTestConfig(overrides?: Partial<StorageConfig>): StorageConfig {
    return {
        userDataPath: "/mock/userData",
        storageDirName: "app-storage",
        ...overrides,
    };
}

// ===== Tests =====

describe("createAppStorage", () => {
    let fs: MockFileSystem;
    let path: IStoragePathUtils;
    let config: StorageConfig;

    beforeEach(() => {
        fs = createMockFileSystem();
        path = createMockPathUtils();
        config = createTestConfig();
    });

    describe("getRoot", () => {
        it("returns storage root path", async () => {
            const storage = createAppStorage(fs, path, config);

            const root = await storage.getRoot();

            expect(root).toBe("/mock/userData/app-storage");
        });

        it("creates storage directory on first access", async () => {
            const storage = createAppStorage(fs, path, config);

            await storage.getRoot();

            expect(fs.mkdir).toHaveBeenCalledWith("/mock/userData/app-storage", { recursive: true });
        });

        it("only creates directory once", async () => {
            const storage = createAppStorage(fs, path, config);

            await storage.getRoot();
            await storage.getRoot();
            await storage.getRoot();

            expect(fs.mkdir).toHaveBeenCalledTimes(1);
        });

        it("uses custom storage directory name", async () => {
            config.storageDirName = "custom-storage";
            const storage = createAppStorage(fs, path, config);

            const root = await storage.getRoot();

            expect(root).toBe("/mock/userData/custom-storage");
        });
    });

    describe("resolvePath", () => {
        it("resolves valid relative paths", async () => {
            const storage = createAppStorage(fs, path, config);

            const resolved = await storage.resolvePath("documents/file.txt");

            expect(resolved).toBe("/mock/userData/app-storage/documents/file.txt");
        });

        it("resolves paths with current directory markers", async () => {
            const storage = createAppStorage(fs, path, config);

            const resolved = await storage.resolvePath("./docs/./readme.md");

            expect(resolved).toBe("/mock/userData/app-storage/docs/readme.md");
        });

        it("rejects absolute paths", async () => {
            const storage = createAppStorage(fs, path, config);

            const resolved = await storage.resolvePath("/etc/passwd");

            expect(resolved).toBeNull();
        });

        it("blocks simple path traversal", async () => {
            const storage = createAppStorage(fs, path, config);

            const resolved = await storage.resolvePath("../../../etc/passwd");

            expect(resolved).toBeNull();
        });

        it("blocks path traversal with intermediate directories", async () => {
            const storage = createAppStorage(fs, path, config);

            const resolved = await storage.resolvePath("docs/../../../etc/passwd");

            expect(resolved).toBeNull();
        });

        it("allows paths with .. that stay within root", async () => {
            const storage = createAppStorage(fs, path, config);

            const resolved = await storage.resolvePath("docs/nested/../file.txt");

            expect(resolved).toBe("/mock/userData/app-storage/docs/file.txt");
        });

        it("allows empty path (returns root)", async () => {
            const storage = createAppStorage(fs, path, config);

            const resolved = await storage.resolvePath("");

            expect(resolved).toBe("/mock/userData/app-storage");
        });
    });

    describe("ensureDir", () => {
        it("creates directory with recursive option", async () => {
            const storage = createAppStorage(fs, path, config);

            await storage.ensureDir("path/to/nested/dir");

            expect(fs.mkdir).toHaveBeenCalledWith(
                "/mock/userData/app-storage/path/to/nested/dir",
                { recursive: true }
            );
        });

        it("throws on invalid path", async () => {
            const storage = createAppStorage(fs, path, config);

            await expect(storage.ensureDir("../outside")).rejects.toThrow("Invalid storage path");
        });
    });

    describe("writeFile", () => {
        it("writes string data", async () => {
            const storage = createAppStorage(fs, path, config);

            await storage.writeFile("test.txt", "Hello, World!");

            expect(fs.writeFile).toHaveBeenCalledWith(
                "/mock/userData/app-storage/test.txt",
                "Hello, World!",
                "utf-8"
            );
        });

        it("writes Buffer data", async () => {
            const storage = createAppStorage(fs, path, config);
            const data = Buffer.from([0x00, 0x01, 0x02]);

            await storage.writeFile("binary.dat", data);

            expect(fs.writeFile).toHaveBeenCalledWith(
                "/mock/userData/app-storage/binary.dat",
                data
            );
        });

        it("creates parent directories", async () => {
            const storage = createAppStorage(fs, path, config);

            await storage.writeFile("nested/path/file.txt", "content");

            // Should create parent dir before writing
            expect(fs.mkdir).toHaveBeenCalledWith(
                "/mock/userData/app-storage/nested/path",
                { recursive: true }
            );
        });

        it("throws on invalid path", async () => {
            const storage = createAppStorage(fs, path, config);

            await expect(storage.writeFile("../escape.txt", "data")).rejects.toThrow("Invalid storage path");
        });
    });

    describe("readFile", () => {
        it("reads file as Buffer", async () => {
            const storage = createAppStorage(fs, path, config);
            fs._files.set("/mock/userData/app-storage/test.txt", "Hello");

            const result = await storage.readFile("test.txt");

            expect(result).toBeInstanceOf(Buffer);
            expect(result?.toString()).toBe("Hello");
        });

        it("returns null for non-existent file", async () => {
            const storage = createAppStorage(fs, path, config);

            const result = await storage.readFile("missing.txt");

            expect(result).toBeNull();
        });

        it("returns null for invalid path", async () => {
            const storage = createAppStorage(fs, path, config);

            const result = await storage.readFile("../escape.txt");

            expect(result).toBeNull();
        });

        it("propagates non-ENOENT errors", async () => {
            const storage = createAppStorage(fs, path, config);
            fs.readFile = vi.fn(async () => {
                const error = new Error("Permission denied");
                (error as NodeJS.ErrnoException).code = "EACCES";
                throw error;
            }) as MockFileSystem["readFile"];

            await expect(storage.readFile("test.txt")).rejects.toThrow("Permission denied");
        });
    });

    describe("readTextFile", () => {
        it("reads file as UTF-8 string", async () => {
            const storage = createAppStorage(fs, path, config);
            fs._files.set("/mock/userData/app-storage/test.txt", "Hello, World!");

            const result = await storage.readTextFile("test.txt");

            expect(result).toBe("Hello, World!");
        });

        it("returns null for non-existent file", async () => {
            const storage = createAppStorage(fs, path, config);

            const result = await storage.readTextFile("missing.txt");

            expect(result).toBeNull();
        });

        it("returns null for invalid path", async () => {
            const storage = createAppStorage(fs, path, config);

            const result = await storage.readTextFile("/etc/passwd");

            expect(result).toBeNull();
        });
    });

    describe("deleteFile", () => {
        it("deletes existing file", async () => {
            const storage = createAppStorage(fs, path, config);
            fs._files.set("/mock/userData/app-storage/test.txt", "content");

            const result = await storage.deleteFile("test.txt");

            expect(result).toBe(true);
            expect(fs.unlink).toHaveBeenCalledWith("/mock/userData/app-storage/test.txt");
        });

        it("returns false for non-existent file", async () => {
            const storage = createAppStorage(fs, path, config);

            const result = await storage.deleteFile("missing.txt");

            expect(result).toBe(false);
        });

        it("returns false for invalid path", async () => {
            const storage = createAppStorage(fs, path, config);

            const result = await storage.deleteFile("../escape.txt");

            expect(result).toBe(false);
        });

        it("propagates non-ENOENT errors", async () => {
            const storage = createAppStorage(fs, path, config);
            fs._files.set("/mock/userData/app-storage/test.txt", "content");
            fs.unlink = vi.fn(async () => {
                const error = new Error("Permission denied");
                (error as NodeJS.ErrnoException).code = "EACCES";
                throw error;
            });

            await expect(storage.deleteFile("test.txt")).rejects.toThrow("Permission denied");
        });
    });

    describe("deleteDir", () => {
        it("deletes directory recursively", async () => {
            const storage = createAppStorage(fs, path, config);
            fs._dirs.add("/mock/userData/app-storage/subdir");

            const result = await storage.deleteDir("subdir");

            expect(result).toBe(true);
            expect(fs.rm).toHaveBeenCalledWith(
                "/mock/userData/app-storage/subdir",
                { recursive: true, force: true }
            );
        });

        it("returns false for invalid path", async () => {
            const storage = createAppStorage(fs, path, config);

            const result = await storage.deleteDir("../escape");

            expect(result).toBe(false);
        });
    });

    describe("listDir", () => {
        it("lists directory contents", async () => {
            const storage = createAppStorage(fs, path, config);
            fs._files.set("/mock/userData/app-storage/docs/file1.txt", "content1");
            fs._files.set("/mock/userData/app-storage/docs/file2.txt", "content2");
            fs._dirs.add("/mock/userData/app-storage/docs");
            fs._dirs.add("/mock/userData/app-storage/docs/subdir");

            const result = await storage.listDir("docs");

            expect(result).not.toBeNull();
            expect(result?.length).toBe(3);

            const names = result?.map(e => e.name).sort();
            expect(names).toEqual(["file1.txt", "file2.txt", "subdir"]);
        });

        it("returns null for non-existent directory", async () => {
            const storage = createAppStorage(fs, path, config);

            const result = await storage.listDir("missing");

            expect(result).toBeNull();
        });

        it("returns null for invalid path", async () => {
            const storage = createAppStorage(fs, path, config);

            const result = await storage.listDir("../escape");

            expect(result).toBeNull();
        });

        it("includes file metadata", async () => {
            const storage = createAppStorage(fs, path, config);
            fs._files.set("/mock/userData/app-storage/docs/test.txt", "Hello");
            fs._dirs.add("/mock/userData/app-storage/docs");

            const result = await storage.listDir("docs");
            const file = result?.find(e => e.name === "test.txt");

            expect(file).toBeDefined();
            expect(file?.isFile).toBe(true);
            expect(file?.isDirectory).toBe(false);
            expect(file?.size).toBe(5);
            expect(file?.path).toBe("docs/test.txt");
        });
    });

    describe("exists", () => {
        it("returns true for existing file", async () => {
            const storage = createAppStorage(fs, path, config);
            fs._files.set("/mock/userData/app-storage/test.txt", "content");

            const result = await storage.exists("test.txt");

            expect(result).toBe(true);
        });

        it("returns true for existing directory", async () => {
            const storage = createAppStorage(fs, path, config);
            fs._dirs.add("/mock/userData/app-storage/subdir");

            const result = await storage.exists("subdir");

            expect(result).toBe(true);
        });

        it("returns false for non-existent path", async () => {
            const storage = createAppStorage(fs, path, config);

            const result = await storage.exists("missing.txt");

            expect(result).toBe(false);
        });

        it("returns false for invalid path", async () => {
            const storage = createAppStorage(fs, path, config);

            const result = await storage.exists("../escape");

            expect(result).toBe(false);
        });
    });

    describe("stat", () => {
        it("returns stats for file", async () => {
            const storage = createAppStorage(fs, path, config);
            fs._files.set("/mock/userData/app-storage/test.txt", "Hello World");

            const result = await storage.stat("test.txt");

            expect(result).not.toBeNull();
            expect(result?.isFile).toBe(true);
            expect(result?.isDirectory).toBe(false);
            expect(result?.size).toBe(11);
            expect(typeof result?.createdAt).toBe("number");
            expect(typeof result?.modifiedAt).toBe("number");
        });

        it("returns stats for directory", async () => {
            const storage = createAppStorage(fs, path, config);
            fs._dirs.add("/mock/userData/app-storage/subdir");

            const result = await storage.stat("subdir");

            expect(result).not.toBeNull();
            expect(result?.isFile).toBe(false);
            expect(result?.isDirectory).toBe(true);
        });

        it("returns null for non-existent path", async () => {
            const storage = createAppStorage(fs, path, config);

            const result = await storage.stat("missing.txt");

            expect(result).toBeNull();
        });

        it("returns null for invalid path", async () => {
            const storage = createAppStorage(fs, path, config);

            const result = await storage.stat("/etc/passwd");

            expect(result).toBeNull();
        });
    });

    describe("getInfo", () => {
        it("returns storage usage statistics", async () => {
            const storage = createAppStorage(fs, path, config);
            fs._files.set("/mock/userData/app-storage/file1.txt", "Hello");
            fs._files.set("/mock/userData/app-storage/file2.txt", "World!");

            const info = await storage.getInfo();

            expect(info.used).toBe(11); // 5 + 6 bytes
            expect(info.count).toBe(2);
            expect(info.available).toBe(10 * 1024 * 1024 * 1024); // 10GB estimate
        });

        it("handles nested files", async () => {
            const storage = createAppStorage(fs, path, config);
            fs._files.set("/mock/userData/app-storage/nested/deep/file.txt", "content");
            fs._dirs.add("/mock/userData/app-storage/nested");
            fs._dirs.add("/mock/userData/app-storage/nested/deep");

            const info = await storage.getInfo();

            expect(info.used).toBe(7); // "content" = 7 bytes
            expect(info.count).toBe(1);
        });
    });

    describe("path traversal attack vectors", () => {
        it("blocks URL-encoded path traversal when decoded before call", async () => {
            const storage = createAppStorage(fs, path, config);

            // URL decoding happens at the IPC layer before storage call
            // This simulates a decoded path traversal attempt
            const decoded = decodeURIComponent("..%2F..%2Fetc%2Fpasswd");
            const result = await storage.resolvePath(decoded);

            expect(result).toBeNull();
        });

        it("treats non-decoded URL-encoded paths as literal strings", async () => {
            const storage = createAppStorage(fs, path, config);

            // Non-decoded URL-encoded strings are treated literally (no path traversal)
            const result = await storage.resolvePath("..%2F..%2Fetc%2Fpasswd");

            // The literal string "..%2F..." is a valid filename, stays within storage
            expect(result).toBe("/mock/userData/app-storage/..%2F..%2Fetc%2Fpasswd");
        });

        it("handles paths containing null bytes", async () => {
            const storage = createAppStorage(fs, path, config);

            // Null bytes in paths - the actual behavior depends on the OS and Node.js version
            // In Node.js, null bytes in paths typically cause errors or are treated literally
            // The key is that even if the path is accepted, it shouldn't escape storage root
            const result = await storage.resolvePath("legit.txt\0../../../etc/passwd");

            // Either the path is rejected (null) or it stays within storage
            if (result !== null) {
                expect(result.startsWith("/mock/userData/app-storage")).toBe(true);
            }
            // If null, that's also acceptable - the path was rejected
        });

        it("blocks double-dot sequences with spaces", async () => {
            const storage = createAppStorage(fs, path, config);

            const result = await storage.resolvePath(". ./. ./etc/passwd");

            // Spaces make these not real ".." sequences, so treated as literal directory names
            // Should stay within storage root
            if (result !== null) {
                expect(result.startsWith("/mock/userData/app-storage")).toBe(true);
            }
        });
    });
});
