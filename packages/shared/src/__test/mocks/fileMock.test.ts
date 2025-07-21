import { describe, test, expect, beforeEach } from "vitest";
import { MockFile, MockBlob, setupFileMock, teardownFileMock } from "./fileMock.js";

describe("File Mock Tests", () => {
    beforeEach(() => {
        // Ensure clean state for each test
        teardownFileMock();
        setupFileMock();
    });

    test("should create File objects in Node.js environment", () => {
        expect(typeof globalThis.File).toBe("function");
        
        const file = new File(["test content"], "test.txt", { type: "text/plain" });
        expect(file).toBeInstanceOf(File);
        expect(file.name).toBe("test.txt");
        expect(file.type).toBe("text/plain");
        expect(file.size).toBe(12); // "test content" is 12 characters
    });

    test("should create Blob objects in Node.js environment", () => {
        expect(typeof globalThis.Blob).toBe("function");
        
        const blob = new Blob(["test content"], { type: "text/plain" });
        expect(blob).toBeInstanceOf(Blob);
        expect(blob.type).toBe("text/plain");
        expect(blob.size).toBe(12);
    });

    test("should calculate file size correctly", () => {
        const file1 = new File(["hello"], "test1.txt");
        expect(file1.size).toBe(5);

        const file2 = new File(["hello", " ", "world"], "test2.txt");
        expect(file2.size).toBe(11);
    });

    test("should implement text() method", async () => {
        const file = new File(["hello", " ", "world"], "test.txt");
        const text = await file.text();
        expect(text).toBe("hello world");
    });

    test("should implement slice() method", () => {
        const file = new File(["hello world"], "test.txt", { type: "text/plain" });
        const sliced = file.slice(0, 5, "text/html");
        
        expect(sliced).toBeInstanceOf(File);
        expect(sliced.name).toBe("test.txt");
        expect(sliced.type).toBe("text/html");
    });

    test("should implement arrayBuffer() method", async () => {
        const file = new File(["test"], "test.txt");
        const buffer = await file.arrayBuffer();
        expect(buffer).toBeInstanceOf(ArrayBuffer);
        expect(buffer.byteLength).toBe(4);
    });
});

describe("Bot Fixtures Integration", () => {
    test("should allow botFixtures to use File constructor", async () => {
        // This test verifies that the File mock allows botFixtures to work
        const { botFixtures } = await import("../fixtures/api-inputs/botFixtures.js");
        
        expect(botFixtures).toBeDefined();
        expect(botFixtures.complete).toBeDefined();
        expect(botFixtures.complete.create).toBeDefined();
        
        const createData = botFixtures.complete.create;
        expect(createData.profileImage).toBeDefined();
        expect(createData.bannerImage).toBeDefined();
        
        // Verify the images are File objects
        expect(createData.profileImage).toBeInstanceOf(File);
        expect(createData.bannerImage).toBeInstanceOf(File);
        
        // Verify file properties
        expect(createData.profileImage.name).toBe("profile.jpg");
        expect(createData.profileImage.type).toBe("image/jpeg");
        expect(createData.bannerImage.name).toBe("banner.png");
        expect(createData.bannerImage.type).toBe("image/png");
    });
});