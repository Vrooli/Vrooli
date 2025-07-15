import { getBcryptService } from "./bcryptWrapper.js";
import * as crypto from "crypto";
import { randomString } from "./codes.js";

const IV_LENGTH = 16;
const MIN_KEY_LENGTH = 1;
const MAX_KEY_LENGTH = 2048;
const SALT_ROUNDS = 10;
const SITE_KEY_LENGTH = 32;
const SITE_KEY_CHARS = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

/**
 * Service for encrypting and decrypting site and user-provided API keys. 
 * 
 * These are important for letting users add their own API keys to the app.
 * This has two main benefits:
 * 1. Users can try out the app without having to buy credits or a subscription.
 * 2. Users can add API keys for paid external services that we don't natively support.
 * 
 * The encryption is done using AES-256-CBC with a random initialization vector.
 * The initialization vector is prepended to the encrypted data and the whole is base64 encoded.
 */
export class ApiKeyEncryptionService {
    /**
     * The length of the initialization vector used for the AES-256-CBC encryption.
     * This is 16 bytes, as AES-256-CBC uses a 16-byte initialization vector.
     */
    private static IV_LENGTH = IV_LENGTH;
    /**
     * The minimum length of a valid API key.
     */
    private static MIN_KEY_LENGTH = MIN_KEY_LENGTH;
    /**
     * The maximum length of a valid API key.
     */
    private static MAX_KEY_LENGTH = MAX_KEY_LENGTH;
    /**
     * How many salt rounds to use for bcrypt hashing.
     */
    private static SALT_ROUNDS = SALT_ROUNDS;
    /**
     * The length of the site key.
     */
    private static SITE_KEY_LENGTH = SITE_KEY_LENGTH;
    /**
     * Acceptable characters for a site key.
     */
    private static SITE_KEY_CHARS = SITE_KEY_CHARS;

    /**
     * The singleton instance of the service.
     */
    private static instance: ApiKeyEncryptionService;

    /**
     * The encryption key buffer (32 bytes for AES-256).
     */
    private readonly keyBuffer: Buffer;

    /**
     * Private constructor to enforce singleton pattern and initialize key.
     */
    private constructor() {
        // Generate the key once and convert to Buffer for AES-256
        const siteKey = ApiKeyEncryptionService.generateSiteKey();
        this.keyBuffer = Buffer.from(siteKey, "utf8");
    }

    /**
     * Gets the singleton instance of the service.
     */
    public static get(): ApiKeyEncryptionService {
        if (!ApiKeyEncryptionService.instance) {
            ApiKeyEncryptionService.instance = new ApiKeyEncryptionService();
        }
        return ApiKeyEncryptionService.instance;
    }

    /**
     * Checks if a key is valid for encryption or hashing.
     * 
     * @param key The key to validate.
     * @returns True if the key is valid, false otherwise.
     */
    static isValidKey(key: string): boolean {
        // Key must be non-empty and ≤ 2048 characters
        return key.length >= this.MIN_KEY_LENGTH && key.length <= this.MAX_KEY_LENGTH;
    }

    /**
     * Encrypts a plaintext string using AES-256-CBC.
     * Used exclusively for encrypting user-provided API keys.
     * 
     * @param plaintext String to encrypt
     * @returns Base64-encoded string containing IV and ciphertext
     */
    public encryptExternal(plaintext: string): string {
        // Generate a random 16-byte initialization vector
        const iv = crypto.randomBytes(ApiKeyEncryptionService.IV_LENGTH);

        // Convert Buffer to Uint8Array for proper type compatibility
        const keyArray = new Uint8Array(this.keyBuffer);
        const ivArray = new Uint8Array(iv);

        // Create cipher with AES-256-CBC algorithm
        const cipher = crypto.createCipheriv("aes-256-cbc", keyArray, ivArray);

        // Encrypt the plaintext
        const updateResult = cipher.update(plaintext, "utf8");
        const finalResult = cipher.final();

        // Combine encrypted parts
        const encrypted = new Uint8Array(updateResult.length + finalResult.length);
        encrypted.set(updateResult, 0);
        encrypted.set(finalResult, updateResult.length);

        // Combine IV and encrypted data
        const result = new Uint8Array(iv.length + encrypted.length);
        result.set(iv, 0);
        result.set(encrypted, iv.length);

        return Buffer.from(result).toString("base64");
    }

    /**
     * Decrypts a base64-encoded string containing IV and ciphertext.
     * Used exclusively for decrypting user-provided API keys.
     * 
     * @param encrypted Base64-encoded string to decrypt
     * @returns Decrypted plaintext string
     */
    public decryptExternal(encrypted: string): string {
        // Decode base64 string to buffer
        const buffer = Buffer.from(encrypted, "base64");

        // Extract IV (first IV_LENGTH bytes) and ciphertext (remaining bytes)
        const iv = buffer.slice(0, ApiKeyEncryptionService.IV_LENGTH);
        const ciphertext = buffer.slice(ApiKeyEncryptionService.IV_LENGTH);

        // Convert Buffer to Uint8Array for proper type compatibility
        const keyArray = new Uint8Array(this.keyBuffer);
        const ivArray = new Uint8Array(iv);
        const ciphertextArray = new Uint8Array(ciphertext);

        // Create decipher
        const decipher = crypto.createDecipheriv("aes-256-cbc", keyArray, ivArray);

        // Decrypt the ciphertext
        const updateResult = decipher.update(ciphertextArray);
        const finalResult = decipher.final();

        // Combine decrypted parts
        const decrypted = new Uint8Array(updateResult.length + finalResult.length);
        decrypted.set(updateResult, 0);
        decrypted.set(finalResult, updateResult.length);

        return Buffer.from(decrypted).toString("utf8");
    }

    /**
     * Generate a new site key.
     * 
     * @returns A new site key.
     */
    static generateSiteKey(): string {
        return randomString(this.SITE_KEY_LENGTH, this.SITE_KEY_CHARS);
    }

    /**
     * Hashes a plaintext API key using bcrypt.
     * Used for hashing the site's own API keys.
     *
     * @param plaintext The plain API key to hash.
     * @returns The hashed API key.
     * @throws Error if the key is invalid or hashing fails.
     */
    static async hashSiteKey(plaintext: string): Promise<string> {
        if (!this.isValidKey(plaintext)) {
            throw new Error("Invalid API key for hashing.");
        }
        return await getBcryptService().hash(plaintext, this.SALT_ROUNDS);
    }

    /**
     * Verifies a plaintext API key against a stored hash.
     * Used for verifying incoming API requests.
     *
     * @param plaintext The plain API key to verify.
     * @param storedHash The stored hash to compare against.
     * @returns True if the key matches the hash, false otherwise.
     */
    static async verifySiteKey(plaintext: string, storedHash: string): Promise<boolean> {
        if (!this.isValidKey(plaintext)) {
            return false;
        }
        return await getBcryptService().compare(plaintext, storedHash);
    }
}
