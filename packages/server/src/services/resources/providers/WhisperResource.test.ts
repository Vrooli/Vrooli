/**
 * Integration tests for WhisperResource
 * 
 * Tests speech-to-text transcription capabilities, language detection,
 * and API integration with the Whisper service running on localhost:8090
 */

import { promises as fs } from "fs";
import path from "path";
import { afterAll, beforeAll, describe, expect, test } from "vitest";
import { logger } from "../../../events/logger.js";
import { WhisperResource } from "./WhisperResource.js";

// Test timeout for transcription operations
const TRANSCRIPTION_TIMEOUT = 60000; // 1 minute

describe("WhisperResource Integration Tests", () => {
    let resource: WhisperResource;
    const testAudioPath = path.join(__dirname, "../../../../../../scripts/resources/ai/whisper/tests/audio");

    beforeAll(async () => {
        // Initialize resource with test configuration
        resource = new WhisperResource();
        await resource.initialize({
            baseUrl: "http://localhost:8090",
            modelSize: "large",
            healthCheck: {
                endpoint: "/",
                intervalMs: 60000,
                timeoutMs: 5000,
                retryCount: 3,
            },
        });

        // Wait for service to be ready
        const maxRetries = 10;
        let retries = 0;
        while (retries < maxRetries) {
            const health = await resource.checkHealth();
            if (health.healthy) {
                break;
            }
            await new Promise(resolve => setTimeout(resolve, 2000));
            retries++;
        }

        if (retries >= maxRetries) {
            throw new Error("Whisper service is not available on port 8090");
        }
    });

    afterAll(async () => {
        await resource.cleanup();
    });

    describe("Service Health and Discovery", () => {
        test("should discover running service", async () => {
            const discovered = await resource.discover();
            expect(discovered).toBe(true);
        });

        test("should report healthy status with 307 redirect", async () => {
            const health = await resource.checkHealth();
            expect(health.healthy).toBe(true);
            expect(health.message).toContain("Whisper service is healthy");
            expect(health.details?.baseUrl).toBe("http://localhost:8090");
            expect(health.details?.modelSize).toBe("large");
        });

        test("should provide metadata with capabilities", async () => {
            const metadata = await resource.getMetadataSnapshot();
            expect(metadata).toMatchObject({
                version: "whisper.cpp",
                capabilities: expect.arrayContaining([
                    "transcription",
                    "translation", 
                    "speech-to-text",
                    "language-detection",
                ]),
                supportedModels: expect.arrayContaining([
                    "tiny",
                    "base",
                    "small",
                    "medium",
                    "large",
                    "large-v3",
                ]),
            });
        });
    });

    describe("Model Management", () => {
        test("should list default models when endpoint unavailable", async () => {
            const models = await resource.listModels();
            expect(models).toContain("tiny");
            expect(models).toContain("base");
            expect(models).toContain("small");
            expect(models).toContain("medium");
            expect(models).toContain("large");
            expect(models).toContain("large-v3");
        });

        test("should check if model is available", async () => {
            expect(await resource.hasModel("large")).toBe(true);
            expect(await resource.hasModel("large-v3")).toBe(true);
            expect(await resource.hasModel("nonexistent")).toBe(false);
        });
    });

    describe("Transcription Tests", () => {
        test("should transcribe speech audio file", async () => {
            // Skip if test audio doesn't exist
            const audioFile = path.join(testAudioPath, "test_speech.mp3");
            try {
                await fs.access(audioFile);
            } catch {
                console.log("Skipping test - test_speech.mp3 not found");
                return;
            }

            const audioBuffer = await fs.readFile(audioFile);
            
            // Make direct API call to test transcription
            const response = await fetch("http://localhost:8090/asr?output=json", {
                method: "POST",
                body: createFormData(audioBuffer, "test_speech.mp3"),
            });

            expect(response.ok).toBe(true);
            const result = await response.json();
            
            expect(result).toHaveProperty("text");
            expect(result).toHaveProperty("language");
            expect(result.text.length).toBeGreaterThan(10);
            expect(result.language).toBe("en");
        }, TRANSCRIPTION_TIMEOUT);

        test("should handle short audio file", async () => {
            // Skip if test audio doesn't exist
            const audioFile = path.join(testAudioPath, "test_short.mp3");
            try {
                await fs.access(audioFile);
            } catch {
                console.log("Skipping test - test_short.mp3 not found");
                return;
            }

            const audioBuffer = await fs.readFile(audioFile);
            
            const response = await fetch("http://localhost:8090/asr?output=json", {
                method: "POST", 
                body: createFormData(audioBuffer, "test_short.mp3"),
            });

            expect(response.ok).toBe(true);
            const result = await response.json();
            
            expect(result).toHaveProperty("text");
            expect(result).toHaveProperty("segments");
        }, TRANSCRIPTION_TIMEOUT);
    });

    describe("Language Detection", () => {
        test("should detect language from audio", async () => {
            // Skip if test audio doesn't exist
            const audioFile = path.join(testAudioPath, "test_speech.mp3");
            try {
                await fs.access(audioFile);
            } catch {
                console.log("Skipping test - test_speech.mp3 not found");
                return;
            }

            const audioBuffer = await fs.readFile(audioFile);
            
            const response = await fetch("http://localhost:8090/detect-language", {
                method: "POST",
                body: createFormData(audioBuffer, "test_speech.mp3"),
            });

            expect(response.ok).toBe(true);
            const result = await response.json();
            
            expect(result).toHaveProperty("detected_language");
            expect(result).toHaveProperty("language_code");
            expect(result).toHaveProperty("confidence");
            expect(result.language_code).toBe("en");
        }, TRANSCRIPTION_TIMEOUT);
    });

    describe("API Behavior", () => {
        test("should handle 307 redirect on root endpoint", async () => {
            const response = await fetch("http://localhost:8090/", {
                redirect: "manual",
            });
            
            expect(response.status).toBe(307);
            expect(response.headers.get("location")).toBe("/docs");
        });

        test("should provide OpenAPI spec", async () => {
            const response = await fetch("http://localhost:8090/openapi.json");
            expect(response.ok).toBe(true);
            
            const spec = await response.json();
            expect(spec).toHaveProperty("openapi");
            expect(spec).toHaveProperty("paths");
            expect(spec.paths).toHaveProperty("/asr");
            expect(spec.paths).toHaveProperty("/detect-language");
        });
    });

    describe("Error Handling", () => {
        test("should handle missing audio file", async () => {
            const response = await fetch("http://localhost:8090/asr", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({}),
            });

            expect(response.ok).toBe(false);
            expect(response.status).toBe(405); // Method Not Allowed
        });

        test("should handle unavailable service gracefully", async () => {
            // Create a new resource with wrong URL
            const badResource = new WhisperResource();
            await badResource.initialize({
                baseUrl: "http://localhost:99999", // Invalid port
                healthCheck: {
                    endpoint: "/",
                    intervalMs: 60000,
                    timeoutMs: 1000,
                    retryCount: 1,
                },
            });

            const health = await badResource.checkHealth();
            expect(health.healthy).toBe(false);
            expect(health.message).toContain("failed");

            await badResource.cleanup();
        });
    });
});

// Helper function to create form data
function createFormData(buffer: Buffer, filename: string): FormData {
    const formData = new FormData();
    const blob = new Blob([buffer], { type: "audio/mpeg" });
    formData.append("audio_file", blob, filename);
    return formData;
}