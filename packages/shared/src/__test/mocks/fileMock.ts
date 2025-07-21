/**
 * File API mock for Node.js test environments
 * Extracted from integration package and enhanced for shared use
 */

export class MockFile {
    public bits: any[];
    public name: string;
    public type: string;
    public lastModified: number;
    public size: number;

    constructor(bits: any[], name: string, options: { type?: string; lastModified?: number } = {}) {
        this.bits = bits;
        this.name = name;
        this.type = options.type || "";
        this.lastModified = options.lastModified || Date.now();
        
        // Calculate size from bits
        this.size = bits.reduce((acc, bit) => {
            if (typeof bit === "string") return acc + bit.length;
            if (bit instanceof ArrayBuffer) return acc + bit.byteLength;
            if (bit instanceof Uint8Array) return acc + bit.length;
            return acc;
        }, 0);
    }

    // Mock File methods that might be used in tests
    async text(): Promise<string> {
        return this.bits.map(bit => {
            if (typeof bit === "string") return bit;
            if (bit instanceof ArrayBuffer || bit instanceof Uint8Array) {
                return new TextDecoder().decode(bit);
            }
            return String(bit);
        }).join("");
    }

    slice(start?: number, end?: number, contentType?: string): MockFile {
        // Simple slice implementation
        const text = this.bits.join("");
        const sliced = text.slice(start, end);
        return new MockFile([sliced], this.name, { type: contentType || this.type });
    }

    // Add arrayBuffer method for completeness
    async arrayBuffer(): Promise<ArrayBuffer> {
        const text = await this.text();
        const encoder = new TextEncoder();
        return encoder.encode(text).buffer;
    }

    // Add stream method stub
    stream(): ReadableStream {
        throw new Error("stream() method not implemented in File mock");
    }
}

export class MockBlob {
    public bits: any[];
    public type: string;
    public size: number;

    constructor(bits: any[] = [], options: { type?: string } = {}) {
        this.bits = bits;
        this.type = options.type || "";
        this.size = bits.reduce((acc, bit) => {
            if (typeof bit === "string") return acc + bit.length;
            if (bit instanceof ArrayBuffer) return acc + bit.byteLength;
            if (bit instanceof Uint8Array) return acc + bit.length;
            return acc;
        }, 0);
    }

    async text(): Promise<string> {
        return this.bits.map(bit => {
            if (typeof bit === "string") return bit;
            if (bit instanceof ArrayBuffer || bit instanceof Uint8Array) {
                return new TextDecoder().decode(bit);
            }
            return String(bit);
        }).join("");
    }

    slice(start?: number, end?: number, contentType?: string): MockBlob {
        const text = this.bits.join("");
        const sliced = text.slice(start, end);
        return new MockBlob([sliced], { type: contentType || this.type });
    }

    async arrayBuffer(): Promise<ArrayBuffer> {
        const text = await this.text();
        const encoder = new TextEncoder();
        return encoder.encode(text).buffer;
    }
}

/**
 * Sets up File and Blob mocks in the global scope for Node.js test environments
 * Safe to call multiple times - checks if mocks are already present
 */
export function setupFileMock(): void {
    // Only setup if we're in a Node.js environment without File API
    if (typeof globalThis.File === "undefined") {
        (globalThis as any).File = MockFile;
    }
    
    if (typeof globalThis.Blob === "undefined") {
        (globalThis as any).Blob = MockBlob;
    }
}

/**
 * Cleans up File and Blob mocks from global scope
 * Used in test teardown to prevent pollution between tests
 */
export function teardownFileMock(): void {
    // Only remove if they're our mocks (check constructor name to avoid type issues)
    if (typeof globalThis.File !== "undefined" && (globalThis.File as any).name === "MockFile") {
        delete (globalThis as any).File;
    }
    
    if (typeof globalThis.Blob !== "undefined" && (globalThis.Blob as any).name === "MockBlob") {
        delete (globalThis as any).Blob;
    }
}