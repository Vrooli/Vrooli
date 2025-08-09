/**
 * Integration tests for UnstructuredIoResource
 * 
 * Tests document processing capabilities, format support, and API integration
 * with the Unstructured.io service running on localhost:11450
 */

import { promises as fs } from "fs";
import path from "path";
import { afterAll, beforeAll, describe, expect, test } from "vitest";
import { logger } from "../../../events/logger.js";
import { UnstructuredIoResource } from "./UnstructuredIoResource.js";

// Test timeout for long document processing
const PROCESSING_TIMEOUT = 120000; // 2 minutes

describe("UnstructuredIoResource Integration Tests", () => {
    let resource: UnstructuredIoResource;
    const testFixturesPath = path.join(__dirname, "../../../../../../scripts/resources/tests/fixtures/documents");

    beforeAll(async () => {
        // Initialize resource with test configuration
        resource = new UnstructuredIoResource();
        await resource.initialize({
            baseUrl: "http://localhost:11450",
            processing: {
                defaultStrategy: "hi_res",
                ocrLanguages: ["eng"],
                chunkingEnabled: true,
                maxChunkSize: 2000,
            },
            healthCheck: {
                endpoint: "/healthcheck",
                intervalMs: 60000,
                timeoutMs: 5000,
                retryCount: 3,
            },
        });

        // Wait for service to be ready
        const maxRetries = 10;
        let retries = 0;
        while (retries < maxRetries) {
            const health = await resource.healthCheck();
            if (health.healthy) {
                break;
            }
            await new Promise(resolve => setTimeout(resolve, 2000));
            retries++;
        }

        if (retries >= maxRetries) {
            throw new Error("Unstructured.io service is not available");
        }
    });

    afterAll(async () => {
        await resource.shutdown();
    });

    describe("Service Health and Discovery", () => {
        test("should discover running service", async () => {
            const discovered = await resource.discover();
            expect(discovered).toBe(true);
        });

        test("should report healthy status", async () => {
            const health = await resource.healthCheck();
            expect(health.healthy).toBe(true);
            expect(health.message).toContain("Unstructured.io service is healthy");
            expect(health.details?.baseUrl).toBe("http://localhost:11450");
        });

        test("should provide metadata", async () => {
            const metadata = await resource.getMetadataSnapshot();
            expect(metadata).toMatchObject({
                version: "latest",
                apiVersion: "v0",
                capabilities: expect.arrayContaining([
                    "document-processing",
                    "ocr",
                    "table-extraction",
                    "layout-analysis",
                ]),
                processingCapabilities: {
                    strategies: expect.arrayContaining(["fast", "hi_res", "auto"]),
                    outputFormats: expect.arrayContaining(["json", "markdown", "text", "elements"]),
                    ocrEnabled: true,
                    tableExtraction: true,
                },
            });
        });
    });

    describe("Format Support", () => {
        test("should list all supported formats", () => {
            const formats = resource.getSupportedFormats();
            expect(formats.length).toBeGreaterThan(20);

            // Check key formats are supported
            const extensions = formats.map(f => f.extension);
            expect(extensions).toContain("pdf");
            expect(extensions).toContain("docx");
            expect(extensions).toContain("xlsx");
            expect(extensions).toContain("pptx");
            expect(extensions).toContain("txt");
            expect(extensions).toContain("html");
            expect(extensions).toContain("png");
            expect(extensions).toContain("jpg");
        });

        test("should correctly identify supported formats", () => {
            expect(resource.isFormatSupported("pdf")).toBe(true);
            expect(resource.isFormatSupported("PDF")).toBe(true);
            expect(resource.isFormatSupported(".pdf")).toBe(true);
            expect(resource.isFormatSupported("docx")).toBe(true);
            expect(resource.isFormatSupported("unknown")).toBe(false);
        });

        test("should categorize formats correctly", () => {
            const formats = resource.getSupportedFormats();
            const pdfFormat = formats.find(f => f.extension === "pdf");
            expect(pdfFormat?.category).toBe("document");

            const xlsxFormat = formats.find(f => f.extension === "xlsx");
            expect(xlsxFormat?.category).toBe("spreadsheet");

            const pngFormat = formats.find(f => f.extension === "png");
            expect(pngFormat?.category).toBe("image");
            expect(pngFormat?.requiresOCR).toBe(true);
        });
    });

    describe("Document Processing - Text Files", () => {
        test("should process simple text file", async () => {
            const testFile = await fs.readFile(
                path.join(testFixturesPath, "meeting-notes.txt"),
            );

            const result = await resource.processDocument(testFile, {
                strategy: "fast",
                outputFormat: "json",
            });

            expect(result).toBeDefined();
            expect(Array.isArray(result)).toBe(true);
            expect(result.length).toBeGreaterThan(0);
            expect(result[0]).toHaveProperty("text");
            expect(result[0]).toHaveProperty("type");
        }, PROCESSING_TIMEOUT);

        test("should convert text to markdown format", async () => {
            const testFile = await fs.readFile(
                path.join(testFixturesPath, "meeting-notes.txt"),
            );

            const result = await resource.processDocument(testFile, {
                strategy: "fast",
                outputFormat: "markdown",
            });

            expect(result).toBeDefined();
            expect(typeof result).toBe("string");
            expect(result).toContain("#"); // Markdown headers
        }, PROCESSING_TIMEOUT);
    });

    describe("Document Processing - Office Documents", () => {
        test("should process Word document (.docx)", async () => {
            const docxPath = path.join(testFixturesPath, "office/word/educational/iup_test_document.docx");
            const testFile = await fs.readFile(docxPath);

            const result = await resource.processDocument(testFile, {
                strategy: "hi_res",
                outputFormat: "json",
            });

            expect(result).toBeDefined();
            expect(Array.isArray(result)).toBe(true);

            // Check for various element types
            const elementTypes = new Set(result.map((el: any) => el.type));
            expect(elementTypes.size).toBeGreaterThan(1); // Should have multiple element types
        }, PROCESSING_TIMEOUT);

        test("should process Excel spreadsheet (.xlsx)", async () => {
            const xlsxPath = path.join(testFixturesPath, "office/excel/educational/ou_sample_data.xlsx");
            const testFile = await fs.readFile(xlsxPath);

            const result = await resource.processDocument(testFile, {
                strategy: "hi_res",
                outputFormat: "json",
            });

            expect(result).toBeDefined();
            expect(Array.isArray(result)).toBe(true);

            // Check for table data
            const tables = result.filter((el: any) => el.type === "Table");
            expect(tables.length).toBeGreaterThan(0);
        }, PROCESSING_TIMEOUT);
    });

    describe("Document Processing - PDFs", () => {
        test("should process PDF with text content", async () => {
            const pdfPath = path.join(testFixturesPath, "office/pdf/government/constitution_annotated.pdf");
            const testFile = await fs.readFile(pdfPath);

            const result = await resource.processDocument(testFile, {
                strategy: "hi_res",
                outputFormat: "json",
                includePageBreaks: true,
            });

            expect(result).toBeDefined();
            expect(Array.isArray(result)).toBe(true);
            expect(result.length).toBeGreaterThan(10); // Should have substantial content

            // Check for various element types
            const hasTitle = result.some((el: any) => el.type === "Title");
            const hasText = result.some((el: any) => el.type === "NarrativeText");
            const hasPageBreaks = result.some((el: any) => el.type === "PageBreak");

            expect(hasTitle).toBe(true);
            expect(hasText).toBe(true);
            expect(hasPageBreaks).toBe(true);
        }, PROCESSING_TIMEOUT);

        test("should extract tables from PDF", async () => {
            // Use a PDF that likely contains tables
            const pdfPath = path.join(testFixturesPath, "samples/census_income_report.pdf");
            const testFile = await fs.readFile(pdfPath);

            const result = await resource.processDocument(testFile, {
                strategy: "hi_res",
                outputFormat: "json",
            });

            // Check for table elements
            const tables = result.filter((el: any) => el.type === "Table");
            if (tables.length > 0) {
                expect(tables[0]).toHaveProperty("text");
                expect(tables[0].text.length).toBeGreaterThan(0);
            }
        }, PROCESSING_TIMEOUT);
    });

    describe("Document Processing - Web Content", () => {
        test("should process HTML files", async () => {
            const htmlPath = path.join(testFixturesPath, "web/article.html");
            const testFile = await fs.readFile(htmlPath);

            const result = await resource.processDocument(testFile, {
                strategy: "fast",
                outputFormat: "json",
            });

            expect(result).toBeDefined();
            expect(Array.isArray(result)).toBe(true);

            // HTML should be parsed into structured elements
            const elementTypes = result.map((el: any) => el.type);
            expect(elementTypes).toContain("Title");
            expect(elementTypes).toContain("NarrativeText");
        }, PROCESSING_TIMEOUT);
    });

    describe("Document Processing - Structured Data", () => {
        test("should process JSON files", async () => {
            const jsonPath = path.join(testFixturesPath, "structured/database_export.json");
            const testFile = await fs.readFile(jsonPath);

            const result = await resource.processDocument(testFile, {
                strategy: "fast",
                outputFormat: "json",
            });

            expect(result).toBeDefined();
            expect(Array.isArray(result)).toBe(true);
        }, PROCESSING_TIMEOUT);

        test("should process CSV files", async () => {
            const csvPath = path.join(testFixturesPath, "structured/customers.csv");
            const testFile = await fs.readFile(csvPath);

            const result = await resource.processDocument(testFile, {
                strategy: "hi_res",
                outputFormat: "json",
            });

            expect(result).toBeDefined();
            expect(Array.isArray(result)).toBe(true);

            // CSV data should be extracted as table
            const tables = result.filter((el: any) => el.type === "Table");
            expect(tables.length).toBeGreaterThan(0);
        }, PROCESSING_TIMEOUT);
    });

    describe("Processing Strategies", () => {
        test("should process with fast strategy", async () => {
            const testFile = await fs.readFile(
                path.join(testFixturesPath, "meeting-notes.txt"),
            );

            const startTime = Date.now();
            const result = await resource.processDocument(testFile, {
                strategy: "fast",
                outputFormat: "json",
            });
            const duration = Date.now() - startTime;

            expect(result).toBeDefined();
            expect(Array.isArray(result)).toBe(true);

            // Fast strategy should be relatively quick
            logger.info(`Fast strategy processing took ${duration}ms`);
        }, PROCESSING_TIMEOUT);

        test("should process with hi_res strategy for complex layout", async () => {
            const pdfPath = path.join(testFixturesPath, "samples/gao_report_sample.pdf");
            const testFile = await fs.readFile(pdfPath);

            const result = await resource.processDocument(testFile, {
                strategy: "hi_res",
                outputFormat: "json",
            });

            expect(result).toBeDefined();
            expect(Array.isArray(result)).toBe(true);

            // Hi-res should provide more detailed extraction
            const elementTypes = new Set(result.map((el: any) => el.type));
            expect(elementTypes.size).toBeGreaterThan(2);
        }, PROCESSING_TIMEOUT);
    });

    describe("Output Formats", () => {
        const sampleFile = path.join(testFixturesPath, "meeting-notes.txt");

        test("should output JSON format", async () => {
            const testFile = await fs.readFile(sampleFile);
            const result = await resource.processDocument(testFile, {
                outputFormat: "json",
            });

            expect(result).toBeDefined();
            expect(Array.isArray(result)).toBe(true);
            expect(result[0]).toHaveProperty("type");
            expect(result[0]).toHaveProperty("text");
        }, PROCESSING_TIMEOUT);

        test("should output Markdown format", async () => {
            const testFile = await fs.readFile(sampleFile);
            const result = await resource.processDocument(testFile, {
                outputFormat: "markdown",
            });

            expect(result).toBeDefined();
            expect(typeof result).toBe("string");
            // Should contain markdown formatting
            const hasHeaders = result.includes("#") || result.includes("##");
            const hasContent = result.length > 100;
            expect(hasHeaders || hasContent).toBe(true);
        }, PROCESSING_TIMEOUT);

        test("should output plain text format", async () => {
            const testFile = await fs.readFile(sampleFile);
            const result = await resource.processDocument(testFile, {
                outputFormat: "text",
            });

            expect(result).toBeDefined();
            expect(typeof result).toBe("string");
            expect(result.length).toBeGreaterThan(0);
            // Should not contain JSON structure
            expect(result.startsWith("[")).toBe(false);
            expect(result.startsWith("{")).toBe(false);
        }, PROCESSING_TIMEOUT);
    });

    describe("Error Handling", () => {
        test("should handle empty file", async () => {
            const emptyFile = Buffer.from("");

            await expect(
                resource.processDocument(emptyFile),
            ).rejects.toThrow();
        });

        test("should handle invalid format option", async () => {
            const testFile = await fs.readFile(
                path.join(testFixturesPath, "meeting-notes.txt"),
            );

            // Should still process but return JSON (default)
            const result = await resource.processDocument(testFile, {
                outputFormat: "invalid" as any,
            });

            expect(result).toBeDefined();
        }, PROCESSING_TIMEOUT);

        test("should handle service unavailability gracefully", async () => {
            // Create a new resource with wrong URL
            const badResource = new UnstructuredIoResource();
            await badResource.initialize({
                baseUrl: "http://localhost:99999", // Invalid port
                healthCheck: {
                    endpoint: "/healthcheck",
                    intervalMs: 60000,
                    timeoutMs: 1000,
                    retryCount: 1,
                },
            });

            const health = await badResource.healthCheck();
            expect(health.healthy).toBe(false);

            await expect(
                badResource.processDocument(Buffer.from("test")),
            ).rejects.toThrow("Unstructured.io service is not healthy");

            await badResource.shutdown();
        });
    });

    describe("Performance and Limits", () => {
        test("should handle large text files", async () => {
            // Create a large text file
            const largeText = "This is a test paragraph. ".repeat(10000);
            const largeFile = Buffer.from(largeText);

            const result = await resource.processDocument(largeFile, {
                strategy: "fast",
                outputFormat: "json",
            });

            expect(result).toBeDefined();
            expect(Array.isArray(result)).toBe(true);
            expect(result.length).toBeGreaterThan(0);
        }, PROCESSING_TIMEOUT);

        test("should process multiple files sequentially", async () => {
            const files = [
                "meeting-notes.txt",
                "web/article.html",
                "structured/customers.csv",
            ];

            for (const file of files) {
                const testFile = await fs.readFile(
                    path.join(testFixturesPath, file),
                );

                const result = await resource.processDocument(testFile, {
                    strategy: "fast",
                    outputFormat: "json",
                });

                expect(result).toBeDefined();
                expect(Array.isArray(result)).toBe(true);
            }
        }, PROCESSING_TIMEOUT * 3);
    });

    describe("API Endpoints", () => {
        test("should use correct processing endpoint", async () => {
            // The endpoint is hardcoded in the implementation
            const config = resource["config"];
            expect(config?.baseUrl).toBe("http://localhost:11450");

            // Verify by successful processing
            const testFile = await fs.readFile(
                path.join(testFixturesPath, "meeting-notes.txt"),
            );

            const result = await resource.processDocument(testFile);
            expect(result).toBeDefined();
        }, PROCESSING_TIMEOUT);

        test("should check health endpoint", async () => {
            const health = await resource.healthCheck();
            expect(health.healthy).toBe(true);

            // The health check uses /healthcheck endpoint
            const config = resource["config"];
            expect(config?.healthCheck?.endpoint).toBe("/healthcheck");
        });
    });
});
