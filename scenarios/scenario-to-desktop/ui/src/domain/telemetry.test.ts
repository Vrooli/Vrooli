import { describe, it, expect } from "vitest";
import {
  generateTelemetryPaths,
  getTelemetryPathForOS,
  parseJsonlContent,
  validateTelemetryEvents,
  isStandardEvent,
  processTelemetryContent,
  formatEventPreview,
  generateExampleEvent,
  TELEMETRY_FILE_NAME,
  STANDARD_EVENTS
} from "./telemetry";

describe("telemetry domain", () => {
  describe("generateTelemetryPaths", () => {
    it("generates paths for all platforms", () => {
      const paths = generateTelemetryPaths("MyApp");

      expect(paths).toHaveLength(3);
      expect(paths.map((p) => p.os)).toEqual(["Windows", "macOS", "Linux"]);
    });

    it("includes app name in paths", () => {
      const paths = generateTelemetryPaths("TestApp");

      expect(paths.find((p) => p.os === "Windows")?.path).toContain("TestApp");
      expect(paths.find((p) => p.os === "macOS")?.path).toContain("TestApp");
      expect(paths.find((p) => p.os === "Linux")?.path).toContain("TestApp");
    });

    it("includes telemetry filename in paths", () => {
      const paths = generateTelemetryPaths("App");

      paths.forEach((p) => {
        expect(p.path).toContain(TELEMETRY_FILE_NAME);
      });
    });

    it("uses correct platform-specific locations", () => {
      const paths = generateTelemetryPaths("App");

      expect(paths.find((p) => p.os === "Windows")?.path).toContain("%APPDATA%");
      expect(paths.find((p) => p.os === "macOS")?.path).toContain("Library/Application Support");
      expect(paths.find((p) => p.os === "Linux")?.path).toContain(".config");
    });

    it("throws for empty app name", () => {
      expect(() => generateTelemetryPaths("")).toThrow("appName is required");
    });
  });

  describe("getTelemetryPathForOS", () => {
    it("returns correct path for each OS", () => {
      expect(getTelemetryPathForOS("App", "Windows")).toContain("%APPDATA%");
      expect(getTelemetryPathForOS("App", "macOS")).toContain("Library");
      expect(getTelemetryPathForOS("App", "Linux")).toContain(".config");
    });

    it("returns empty string for invalid OS", () => {
      expect(getTelemetryPathForOS("App", "FreeBSD" as never)).toBe("");
    });
  });

  describe("parseJsonlContent", () => {
    it("parses valid JSONL content", () => {
      const content = '{"event":"test1"}\n{"event":"test2"}';
      const result = parseJsonlContent(content);

      expect(result.success).toBe(true);
      expect(result.events).toHaveLength(2);
      expect(result.events[0].event).toBe("test1");
      expect(result.events[1].event).toBe("test2");
    });

    it("handles Windows line endings", () => {
      const content = '{"event":"test1"}\r\n{"event":"test2"}';
      const result = parseJsonlContent(content);

      expect(result.success).toBe(true);
      expect(result.events).toHaveLength(2);
    });

    it("skips empty lines", () => {
      const content = '{"event":"test1"}\n\n{"event":"test2"}\n';
      const result = parseJsonlContent(content);

      expect(result.events).toHaveLength(2);
    });

    it("reports errors for invalid JSON", () => {
      const content = '{"event":"test1"}\nnot json\n{"event":"test2"}';
      const result = parseJsonlContent(content);

      expect(result.success).toBe(false);
      expect(result.events).toHaveLength(2);
      expect(result.errors).toHaveLength(1);
      expect(result.errors[0].lineNumber).toBe(2);
    });

    it("rejects arrays as events", () => {
      const content = '{"event":"test"}\n[1,2,3]';
      const result = parseJsonlContent(content);

      expect(result.success).toBe(false);
      expect(result.errors[0].error).toContain("Expected a JSON object");
    });

    it("returns error for empty content", () => {
      const result = parseJsonlContent("");

      expect(result.success).toBe(false);
      expect(result.events).toHaveLength(0);
      expect(result.errors[0].error).toBe("File is empty");
    });

    it("returns error for whitespace-only content", () => {
      const result = parseJsonlContent("   \n\n   ");

      expect(result.success).toBe(false);
    });
  });

  describe("validateTelemetryEvents", () => {
    it("validates well-formed events", () => {
      const events = [
        { event: "runtime_start" },
        { event: "service_ready", service_id: "api" }
      ];
      const result = validateTelemetryEvents(events);

      expect(result.valid).toBe(true);
      expect(result.eventCount).toBe(2);
    });

    it("warns about missing event field", () => {
      const events = [{ some: "data" }];
      const result = validateTelemetryEvents(events);

      expect(result.valid).toBe(true); // Warnings don't invalidate
      expect(result.warnings).toContain("Event 1: Missing 'event' field");
    });

    it("warns about non-standard event types", () => {
      const events = [{ event: "custom_event" }, { event: "another_custom" }];
      const result = validateTelemetryEvents(events);

      expect(result.valid).toBe(true);
      expect(result.warnings.some((w) => w.includes("Non-standard event types"))).toBe(true);
    });

    it("returns invalid for empty events array", () => {
      const result = validateTelemetryEvents([]);

      expect(result.valid).toBe(false);
      expect(result.errors).toContain("No events to validate");
    });
  });

  describe("isStandardEvent", () => {
    it("recognizes standard events", () => {
      expect(isStandardEvent("runtime_start")).toBe(true);
      expect(isStandardEvent("service_ready")).toBe(true);
      expect(isStandardEvent("secrets_missing")).toBe(true);
    });

    it("rejects non-standard events", () => {
      expect(isStandardEvent("custom_event")).toBe(false);
      expect(isStandardEvent("unknown")).toBe(false);
    });
  });

  describe("processTelemetryContent", () => {
    it("processes valid content successfully", () => {
      const content = '{"event":"runtime_start"}\n{"event":"service_ready"}';
      const result = processTelemetryContent(content);

      expect(result.success).toBe(true);
      expect(result.events).toHaveLength(2);
      expect(result.error).toBeUndefined();
    });

    it("returns error for empty content", () => {
      const result = processTelemetryContent("");

      expect(result.success).toBe(false);
      expect(result.error).toBeDefined();
    });

    it("returns error for invalid JSON", () => {
      const content = "not valid json";
      const result = processTelemetryContent(content);

      expect(result.success).toBe(false);
      expect(result.error).toBeDefined();
    });

    it("includes warnings for non-standard events", () => {
      const content = '{"event":"custom_event"}';
      const result = processTelemetryContent(content);

      expect(result.success).toBe(true);
      expect(result.warnings).toBeDefined();
      expect(result.warnings!.some((w) => w.includes("Non-standard"))).toBe(true);
    });
  });

  describe("formatEventPreview", () => {
    it("formats events with all fields", () => {
      const event = {
        event: "test_event",
        timestamp: "2024-01-15T10:00:00Z",
        detail: "some detail"
      };
      const preview = formatEventPreview(event);

      expect(preview).toContain('event: "test_event"');
      expect(preview).toContain('timestamp: "2024-01-15T10:00:00Z"');
      expect(preview).toContain('detail: "some detail"');
    });

    it("handles missing fields gracefully", () => {
      const event = { event: "test" };
      const preview = formatEventPreview(event);

      expect(preview).toContain('event: "test"');
      expect(preview).not.toContain("timestamp");
      expect(preview).not.toContain("detail");
    });

    it("truncates long detail values", () => {
      const event = {
        event: "test",
        detail: "a".repeat(100)
      };
      const preview = formatEventPreview(event);

      expect(preview.length).toBeLessThan(200);
      expect(preview).toContain("...");
    });
  });

  describe("generateExampleEvent", () => {
    it("returns valid JSON", () => {
      const example = generateExampleEvent();

      expect(() => JSON.parse(example)).not.toThrow();
    });

    it("includes required event field", () => {
      const example = generateExampleEvent();
      const parsed = JSON.parse(example);

      expect(parsed.event).toBe("api_unreachable");
    });

    it("includes timestamp", () => {
      const example = generateExampleEvent();
      const parsed = JSON.parse(example);

      expect(parsed.timestamp).toBeDefined();
    });
  });

  describe("STANDARD_EVENTS constant", () => {
    it("includes core runtime events", () => {
      expect(STANDARD_EVENTS).toContain("runtime_start");
      expect(STANDARD_EVENTS).toContain("runtime_shutdown");
      expect(STANDARD_EVENTS).toContain("runtime_error");
    });

    it("includes service lifecycle events", () => {
      expect(STANDARD_EVENTS).toContain("service_start");
      expect(STANDARD_EVENTS).toContain("service_ready");
      expect(STANDARD_EVENTS).toContain("service_exit");
    });

    it("includes infrastructure events", () => {
      expect(STANDARD_EVENTS).toContain("gpu_status");
      expect(STANDARD_EVENTS).toContain("secrets_missing");
      expect(STANDARD_EVENTS).toContain("migration_applied");
    });
  });

  describe("TELEMETRY_FILE_NAME constant", () => {
    it("has the expected filename", () => {
      expect(TELEMETRY_FILE_NAME).toBe("deployment-telemetry.jsonl");
    });
  });
});
