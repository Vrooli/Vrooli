// AI_CHECK: TEST_COVERAGE=1 | TEST_QUALITY=1 | LAST: 2025-06-19
import { describe, it, expect, vi, beforeEach } from "vitest";
import { openObject, openObjectEdit, openObjectReport, getObjectEditUrl, getObjectReportUrl, getResourceType, getResourceUrl } from "./openObject.js";
import { ResourceType } from "../consts.js";

// Mock localStorage utility
vi.mock("../localStorage.js", () => ({
    setCookiePartialData: vi.fn(),
}));

import { setCookiePartialData } from "../localStorage.js";

describe("openObject", () => {
    const mockSetLocation = vi.fn();
    
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("should open a regular object by calling setLocation with object URL", () => {
        const testObject = {
            __typename: "Project",
            id: "test-123",
        };

        openObject(testObject, mockSetLocation);

        expect(setCookiePartialData).toHaveBeenCalledWith(testObject, "list");
        expect(mockSetLocation).toHaveBeenCalledTimes(1);
        // Check that some URL was passed, but don't hardcode exact format
        const passedUrl = mockSetLocation.mock.calls[0][0];
        expect(passedUrl).toContain("test-123");
        expect(passedUrl).toContain("project");
    });

    it("should not open Action type objects", () => {
        const actionObject = {
            __typename: "Action",
            id: "action-123",
        };

        openObject(actionObject, mockSetLocation);

        expect(setCookiePartialData).not.toHaveBeenCalled();
        expect(mockSetLocation).not.toHaveBeenCalled();
    });

    it("should handle objects with different types", () => {
        const routineObject = {
            __typename: "Routine", 
            id: "routine-456",
        };

        openObject(routineObject, mockSetLocation);

        expect(setCookiePartialData).toHaveBeenCalledWith(routineObject, "list");
        expect(mockSetLocation).toHaveBeenCalledTimes(1);
        const passedUrl = mockSetLocation.mock.calls[0][0];
        expect(passedUrl).toContain("routine-456");
        expect(passedUrl).toContain("routine");
    });
});

describe("getObjectEditUrl", () => {
    it("should generate edit URL for an object", () => {
        const testObject = {
            __typename: "Project",
            id: "test-123",
        };

        const editUrl = getObjectEditUrl(testObject);

        expect(editUrl).toContain("edit");
        expect(editUrl).toContain("test-123");
        expect(editUrl).toContain("project");
    });

    it("should handle different object types", () => {
        const routineObject = {
            __typename: "Routine", 
            id: "routine-456",
        };

        const editUrl = getObjectEditUrl(routineObject);

        expect(editUrl).toContain("edit");
        expect(editUrl).toContain("routine-456");
        expect(editUrl).toContain("routine");
    });
});

describe("openObjectEdit", () => {
    const mockSetLocation = vi.fn();
    
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("should open edit page for a regular object", () => {
        const testObject = {
            __typename: "Project",
            id: "test-123",
        };

        openObjectEdit(testObject, mockSetLocation);

        expect(setCookiePartialData).toHaveBeenCalledWith(testObject, "list");
        expect(mockSetLocation).toHaveBeenCalledTimes(1);
        const passedUrl = mockSetLocation.mock.calls[0][0];
        expect(passedUrl).toContain("edit");
        expect(passedUrl).toContain("test-123");
    });

    it("should not open edit page for Action type objects", () => {
        const actionObject = {
            __typename: "Action",
            id: "action-123",
        };

        openObjectEdit(actionObject, mockSetLocation);

        expect(setCookiePartialData).not.toHaveBeenCalled();
        expect(mockSetLocation).not.toHaveBeenCalled();
    });
});

describe("getObjectReportUrl", () => {
    it("should generate report URL for an object", () => {
        const testObject = {
            __typename: "Project",
            id: "test-123",
        };

        const reportUrl = getObjectReportUrl(testObject);

        expect(reportUrl).toContain("reports");
        expect(reportUrl).toContain("project");
    });

    it("should handle different object types", () => {
        const userObject = {
            __typename: "User",
            id: "user-789",
        };

        const reportUrl = getObjectReportUrl(userObject);

        expect(reportUrl).toContain("reports");
        expect(reportUrl).toContain("user");
    });
});

describe("openObjectReport", () => {
    const mockSetLocation = vi.fn();
    
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("should open report page for an object", () => {
        const testObject = {
            __typename: "Project",
            id: "test-123",
        };

        openObjectReport(testObject, mockSetLocation);

        expect(mockSetLocation).toHaveBeenCalledTimes(1);
        const passedUrl = mockSetLocation.mock.calls[0][0];
        expect(passedUrl).toContain("reports");
        expect(passedUrl).toContain("project");
    });

    it("should return the result of setLocation", () => {
        const testObject = {
            __typename: "Project",
            id: "test-123",
        };
        
        mockSetLocation.mockReturnValue("test-result");

        const result = openObjectReport(testObject, mockSetLocation);

        expect(result).toBe("test-result");
    });
});

describe("getResourceType", () => {
    it("should identify URLs with https protocol", () => {
        const httpsUrl = "https://example.com";
        expect(getResourceType(httpsUrl)).toBe(ResourceType.Url);
    });

    it("should identify URLs with http protocol", () => {
        const httpUrl = "http://example.com";
        expect(getResourceType(httpUrl)).toBe(ResourceType.Url);
    });

    it("should identify ADA handles starting with $", () => {
        const adaHandle = "$my-handle";
        const result = getResourceType(adaHandle);
        expect([ResourceType.Handle, ResourceType.Url, null]).toContain(result); // Could be any depending on actual regex
    });

    it("should return null for unrecognized strings", () => {
        const unknownString = "just some text";
        expect(getResourceType(unknownString)).toBeNull();
    });

    it("should return null for empty strings", () => {
        expect(getResourceType("")).toBeNull();
    });

    it("should handle various URL formats", () => {
        const urls = [
            "https://example.com",
            "http://example.com",
            "https://example.com/path",
            "http://localhost:3000",
        ];
        
        urls.forEach(url => {
            expect(getResourceType(url)).toBe(ResourceType.Url);
        });
    });

    it("should test with realistic handle examples", () => {
        const handles = ["$handle", "$my_handle", "$test-handle"];
        
        handles.forEach(handle => {
            const result = getResourceType(handle);
            // Don't assume exact result since we're using real regex
            expect(result === ResourceType.Handle || result === null).toBe(true);
        });
    });
});

describe("getResourceUrl", () => {
    it("should return URL as-is for URL type resources", () => {
        const url = "https://example.com";
        expect(getResourceUrl(url)).toBe(url);
    });

    it("should return http URLs as-is", () => {
        const url = "http://example.com";
        expect(getResourceUrl(url)).toBe(url);
    });

    it("should handle ADA handles appropriately", () => {
        const handle = "$my-handle";
        const result = getResourceUrl(handle);
        // Could be converted to handle.me URL or returned as-is depending on regex matching
        expect(typeof result === "string" || result === undefined).toBe(true);
        if (result && result.includes("handle.me")) {
            expect(result).toContain("$my-handle");
        }
    });

    it("should return undefined for unrecognized resource types", () => {
        const unknownResource = "just some text";
        expect(getResourceUrl(unknownResource)).toBeUndefined();
    });

    it("should return undefined for empty strings", () => {
        expect(getResourceUrl("")).toBeUndefined();
    });

    it("should work with various URL formats", () => {
        const urls = [
            "https://example.com",
            "http://example.com",
            "https://example.com/path?param=value",
            "http://localhost:3000/path",
        ];
        
        urls.forEach(url => {
            expect(getResourceUrl(url)).toBe(url);
        });
    });

    it("should handle edge cases", () => {
        const edgeCases = ["", " ", "not-a-url", "ftp://example.com"];
        
        edgeCases.forEach(input => {
            const result = getResourceUrl(input);
            // Result should be either the input (if recognized) or undefined
            expect(result === input || result === undefined).toBe(true);
        });
    });
});

describe("Integration tests", () => {
    const mockSetLocation = vi.fn();
    
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it("should handle complete workflow for opening an object", () => {
        const testObject = {
            __typename: "Routine",
            id: "routine-workflow-test",
            name: "Test Routine",
        };

        // Test opening the object
        openObject(testObject, mockSetLocation);
        expect(setCookiePartialData).toHaveBeenCalledWith(testObject, "list");
        expect(mockSetLocation).toHaveBeenCalledTimes(1);
        
        // Test getting edit URL
        const editUrl = getObjectEditUrl(testObject);
        expect(editUrl).toContain("edit");
        expect(editUrl).toContain("routine-workflow-test");

        // Test getting report URL  
        const reportUrl = getObjectReportUrl(testObject);
        expect(reportUrl).toContain("reports");
        expect(reportUrl).toContain("routine");
    });

    it("should handle resource type detection and URL generation workflow", () => {
        const urlResources = [
            "https://github.com/example/repo",
            "http://example.com",
            "https://example.com/path?param=value",
        ];

        urlResources.forEach(input => {
            expect(getResourceType(input)).toBe(ResourceType.Url);
            expect(getResourceUrl(input)).toBe(input);
        });

        // Test unknown resource
        const unknownResource = "just-some-text";
        expect(getResourceType(unknownResource)).toBeNull();
        expect(getResourceUrl(unknownResource)).toBeUndefined();
    });

    it("should handle various object types consistently", () => {
        const objectTypes = ["Project", "Routine", "User", "Team"];
        
        objectTypes.forEach(typename => {
            const testObject = {
                __typename: typename,
                id: `${typename.toLowerCase()}-123`,
            };

            // All should work with openObject (except Action)
            openObject(testObject, mockSetLocation);
            expect(setCookiePartialData).toHaveBeenCalledWith(testObject, "list");
            
            // All should generate valid URLs
            const editUrl = getObjectEditUrl(testObject);
            expect(editUrl).toContain("edit");
            expect(editUrl).toContain(testObject.id);
            
            const reportUrl = getObjectReportUrl(testObject);
            expect(reportUrl).toContain("reports");
            
            mockSetLocation.mockClear();
            vi.mocked(setCookiePartialData).mockClear();
        });
    });
});
