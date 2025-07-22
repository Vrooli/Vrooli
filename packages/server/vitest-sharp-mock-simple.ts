/**
 * Simple sharp and nodejs-snowflake mocks for vitest
 */

import { vi } from 'vitest';

// Mock sharp to prevent native module issues
vi.doMock('sharp', () => {
    const makeChain = () => {
        const chain: any = {};
        const pass = () => chain;
        
        // Only include essential methods to avoid duplicates
        Object.assign(chain, {
            resize: pass,
            rotate: pass,
            jpeg: pass,
            png: pass,
            webp: pass,
            toBuffer: async () => Buffer.alloc(0),
            toFile: async () => ({ size: 0 }),
            metadata: async () => ({ width: 100, height: 100, format: "jpeg" }),
        });
        
        return chain;
    };
    
    const sharp = (input?: any) => makeChain();
    
    sharp.format = {
        jpeg: { id: 'jpeg' },
        png: { id: 'png' },
        webp: { id: 'webp' },
    };
    
    sharp.versions = { sharp: '0.32.6-mocked' };
    sharp.cache = vi.fn();
    sharp.concurrency = vi.fn();
    sharp.counters = vi.fn(() => ({ queue: 0, process: 0 }));
    sharp.simd = vi.fn();
    
    return {
        __esModule: true,
        default: sharp,
        sharp,
    };
});

// Mock nodejs-snowflake as fallback
vi.doMock('nodejs-snowflake', () => {
    class MockSnowflake {
        constructor(options: any) {}
        
        getUniqueID() {
            return BigInt(Date.now()) * 1000n + BigInt(Math.floor(Math.random() * 1000));
        }
    }
    
    return {
        __esModule: true,
        default: MockSnowflake,
        Snowflake: MockSnowflake,
    };
});

// Mock bcrypt and bcryptjs to prevent native module loading issues
vi.doMock('bcrypt', () => {
    const mockBcrypt = {
        hash: async (data: string, rounds: number): Promise<string> => `mocked_hash_${data}`,
        hashSync: (data: string, rounds: number): string => `mocked_hash_${data}`,
        compare: async (data: string, encrypted: string): Promise<boolean> => encrypted === `mocked_hash_${data}`,
        compareSync: (data: string, encrypted: string): boolean => encrypted === `mocked_hash_${data}`,
    };
    
    return {
        __esModule: true,
        default: mockBcrypt,
        ...mockBcrypt,
    };
});

vi.doMock('bcryptjs', () => {
    const mockBcryptjs = {
        hash: async (data: string, rounds: number): Promise<string> => `mocked_hash_${data}`,
        hashSync: (data: string, rounds: number): string => `mocked_hash_${data}`,
        compare: async (data: string, encrypted: string): Promise<boolean> => encrypted === `mocked_hash_${data}`,
        compareSync: (data: string, encrypted: string): boolean => encrypted === `mocked_hash_${data}`,
    };
    
    return {
        __esModule: true,
        default: mockBcryptjs,
        ...mockBcryptjs,
    };
});

console.log('[VITEST] Sharp, nodejs-snowflake, and bcrypt modules mocked');