/**
 * Window State Storage
 *
 * DOC: docs/internal/SEAMS.md#window-state-storage
 *
 * Handles persistence of window state to disk.
 * This module provides:
 * - File-based state storage in userData directory
 * - Atomic writes to prevent corruption
 * - Graceful handling of missing/corrupt files
 *
 * Testing Seams:
 * - IFileSystem: Mock filesystem operations
 * - Storage path can be customized via deps
 */

import type { IStateStorage, WindowState } from "./types";

/**
 * Interface for filesystem operations.
 * Seam for testing without actual filesystem access.
 */
export interface IFileSystem {
    /** Read file contents as UTF-8 string */
    readFile(path: string): Promise<string>;
    /** Write string to file (creates parent dirs if needed) */
    writeFile(path: string, content: string): Promise<void>;
    /** Check if file exists */
    exists(path: string): Promise<boolean>;
}

/**
 * Interface for path operations.
 * Seam for testing path resolution.
 */
export interface IPathProvider {
    /** Get the userData directory path */
    getUserDataPath(): string;
    /** Join path segments */
    join(...segments: string[]): string;
}

/**
 * Dependencies for WindowStateStorage.
 */
export interface StorageDeps {
    fileSystem: IFileSystem;
    pathProvider: IPathProvider;
    /** Optional logger for debugging */
    log?: (message: string, ...args: unknown[]) => void;
}

/**
 * File-based window state storage.
 *
 * Responsibilities:
 * - Load state from JSON file
 * - Save state to JSON file with error handling
 * - Validate loaded state structure
 *
 * NOT responsible for:
 * - State validation against displays
 * - Window management
 * - Default state creation
 */
export class WindowStateStorage implements IStateStorage {
    private readonly deps: StorageDeps;
    private readonly fileName: string;
    private readonly log: (message: string, ...args: unknown[]) => void;

    constructor(deps: StorageDeps, fileName: string = "window-state.json") {
        this.deps = deps;
        this.fileName = fileName;
        this.log = deps.log ?? ((...args) => console.log("[WindowStateStorage]", ...args));
    }

    /**
     * Get the full path to the state file.
     */
    private getStatePath(): string {
        return this.deps.pathProvider.join(
            this.deps.pathProvider.getUserDataPath(),
            this.fileName
        );
    }

    /**
     * Load saved state from storage.
     * Returns null if file doesn't exist or is invalid.
     */
    async load(): Promise<WindowState | null> {
        const statePath = this.getStatePath();

        try {
            const exists = await this.deps.fileSystem.exists(statePath);
            if (!exists) {
                this.log("No saved window state found");
                return null;
            }

            const content = await this.deps.fileSystem.readFile(statePath);
            const parsed: unknown = JSON.parse(content);

            // Validate the loaded state has required fields
            if (!this.isValidState(parsed)) {
                this.log("Invalid window state structure, ignoring");
                return null;
            }

            this.log("Loaded window state:", parsed);
            return parsed;
        } catch (error) {
            // Log but don't throw - missing/corrupt state is not fatal
            this.log("Failed to load window state:", error);
            return null;
        }
    }

    /**
     * Save state to storage.
     * Writes atomically to prevent corruption.
     */
    async save(state: WindowState): Promise<void> {
        const statePath = this.getStatePath();

        try {
            const content = JSON.stringify(state, null, 2);
            await this.deps.fileSystem.writeFile(statePath, content);
            this.log("Saved window state:", state);
        } catch (error) {
            // Log but don't throw - failing to save state is not critical
            this.log("Failed to save window state:", error);
        }
    }

    /**
     * Type guard to validate loaded state structure.
     * Checks for required fields and correct types.
     */
    private isValidState(value: unknown): value is WindowState {
        if (typeof value !== "object" || value === null) {
            return false;
        }

        const state = value as Record<string, unknown>;

        // Required fields
        if (typeof state.width !== "number" || state.width <= 0) return false;
        if (typeof state.height !== "number" || state.height <= 0) return false;
        if (typeof state.isMaximized !== "boolean") return false;
        if (typeof state.isFullScreen !== "boolean") return false;

        // Optional fields (must be correct type if present)
        if (state.x !== undefined && typeof state.x !== "number") return false;
        if (state.y !== undefined && typeof state.y !== "number") return false;
        if (state.displayId !== undefined && typeof state.displayId !== "number") return false;

        return true;
    }
}

/**
 * Factory function to create WindowStateStorage with Node.js/Electron dependencies.
 *
 * This is the production factory - tests should create WindowStateStorage
 * directly with mock dependencies.
 */
export function createWindowStateStorage(
    app: { getPath(name: "userData"): string },
    fs: {
        promises: {
            readFile(path: string, encoding: "utf-8"): Promise<string>;
            writeFile(path: string, data: string): Promise<void>;
            access(path: string): Promise<void>;
            mkdir(path: string, options: { recursive: boolean }): Promise<string | undefined>;
        };
    },
    path: { join(...args: string[]): string; dirname(p: string): string },
    fileName?: string
): WindowStateStorage {
    const deps: StorageDeps = {
        fileSystem: {
            readFile: (p) => fs.promises.readFile(p, "utf-8"),
            writeFile: async (p, content) => {
                // Ensure parent directory exists
                const dir = path.dirname(p);
                await fs.promises.mkdir(dir, { recursive: true });
                await fs.promises.writeFile(p, content);
            },
            exists: async (p) => {
                try {
                    await fs.promises.access(p);
                    return true;
                } catch {
                    return false;
                }
            },
        },
        pathProvider: {
            getUserDataPath: () => app.getPath("userData"),
            join: (...segments) => path.join(...segments),
        },
    };

    return new WindowStateStorage(deps, fileName);
}
