import { describe, expect, it } from "vitest";
import { formatNotificationEmail, validateEmailTemplate } from "./formatters.js";

describe("formatNotificationEmail", () => {
    const mockTemplate = `
        <html>
            <body>
                <h1>\${TITLE}</h1>
                <div>\${BODY}</div>
                \${ACTION_BUTTON}
            </body>
        </html>
    `;

    it("should format basic notification without link", () => {
        const result = formatNotificationEmail(
            mockTemplate,
            "Test Title",
            "Test body content",
        );

        expect(result).toContain("<h1>Test Title</h1>");
        expect(result).toContain("Test body content");
        expect(result).not.toContain("View Details");
    });

    it("should format notification with link", () => {
        const result = formatNotificationEmail(
            mockTemplate,
            "Test Title",
            "Test body content",
            "https://example.com/action",
        );

        expect(result).toContain("<h1>Test Title</h1>");
        expect(result).toContain("Test body content");
        expect(result).toContain("View Details");
        // URL will be escaped in href attribute
        expect(result).toContain("href=\"https:&#x2F;&#x2F;example.com&#x2F;action\"");
    });

    it("should escape HTML in title and body", () => {
        const result = formatNotificationEmail(
            mockTemplate,
            "<script>alert('xss')</script>",
            "Body with <strong>HTML</strong> & special chars",
        );

        expect(result).toContain("&lt;script&gt;alert(&#x27;xss&#x27;)&lt;&#x2F;script&gt;");
        expect(result).toContain("&lt;strong&gt;HTML&lt;&#x2F;strong&gt;");
        expect(result).toContain("&amp; special chars");
    });

    it("should convert line breaks to HTML breaks", () => {
        const result = formatNotificationEmail(
            mockTemplate,
            "Multi-line Title",
            "Line 1\nLine 2\nLine 3",
        );

        expect(result).toContain("Line 1<br>Line 2<br>Line 3");
    });

    it("should handle empty body gracefully", () => {
        const result = formatNotificationEmail(
            mockTemplate,
            "Title Only",
            "",
        );

        expect(result).toContain("<h1>Title Only</h1>");
        expect(result).toContain("<p style=");
    });

    it("should escape HTML in link URL", () => {
        const result = formatNotificationEmail(
            mockTemplate,
            "Test Title",
            "Test body",
            "https://example.com/action?param=<script>alert('xss')</script>",
        );

        // Check that the script tag is properly escaped in the href attribute
        expect(result).toContain("&lt;script&gt;alert(&#x27;xss&#x27;)&lt;&#x2F;script&gt;");
        expect(result).not.toContain("<script>alert('xss')</script>");
    });
});

describe("formatNotificationEmail edge cases", () => {
    const mockTemplate = "<html><h1>${TITLE}</h1><div>${BODY}</div>${ACTION_BUTTON}</html>";

    it("should handle empty title gracefully", () => {
        const result = formatNotificationEmail(mockTemplate, "", "Body text");
        expect(result).toContain("<h1></h1>");
        expect(result).toContain("Body text");
    });
    
    it("should handle empty body gracefully", () => {
        const result = formatNotificationEmail(mockTemplate, "Title", "");
        expect(result).toContain("<h1>Title</h1>");
        expect(result).toContain("<p style=");
        expect(result).toContain("</p>");
    });
    
    it("should throw on missing template", () => {
        expect(() => formatNotificationEmail("", "Title", "Body"))
            .toThrow("Email template is required");
    });

    it("should throw when both title and body are empty", () => {
        expect(() => formatNotificationEmail(mockTemplate, "", ""))
            .toThrow("Either title or body must be provided");
    });

    it("should handle very long title and body", () => {
        const longText = "a".repeat(1000);
        const result = formatNotificationEmail(mockTemplate, longText, longText);
        expect(result).toContain(longText);
    });

    it("should handle multiple line breaks", () => {
        const multilineBody = "Line 1\n\nLine 2\n\n\nLine 3";
        const result = formatNotificationEmail(mockTemplate, "Title", multilineBody);
        expect(result).toContain("Line 1<br><br>Line 2<br><br><br>Line 3");
    });

    it("should replace YEAR placeholder with current year", () => {
        const templateWithYear = "<html>${TITLE} ${BODY} ${ACTION_BUTTON} Copyright ${YEAR}</html>";
        const result = formatNotificationEmail(templateWithYear, "Title", "Body");
        const currentYear = new Date().getFullYear().toString();
        expect(result).toContain(`Copyright ${currentYear}`);
    });
});

describe("validateEmailTemplate", () => {
    it("should validate template with all placeholders", () => {
        const template = "<html>\${TITLE}\${BODY}\${ACTION_BUTTON}\${YEAR}</html>";
        const validation = validateEmailTemplate(template);
        expect(validation.isValid).toBe(true);
        expect(validation.missingPlaceholders).toHaveLength(0);
    });
    
    it("should identify missing placeholders", () => {
        const badTemplate = "<html><body>Missing placeholders</body></html>";
        const validation = validateEmailTemplate(badTemplate);
        expect(validation.isValid).toBe(false);
        expect(validation.missingPlaceholders).toContain("${TITLE}");
        expect(validation.missingPlaceholders).toContain("${BODY}");
        expect(validation.missingPlaceholders).toContain("${ACTION_BUTTON}");
        expect(validation.missingPlaceholders).toContain("${YEAR}");
    });

    it("should identify partially missing placeholders", () => {
        const partialTemplate = "<html>\${TITLE}Only title</html>";
        const validation = validateEmailTemplate(partialTemplate);
        expect(validation.isValid).toBe(false);
        expect(validation.missingPlaceholders).toHaveLength(3);
        expect(validation.missingPlaceholders).toContain("${BODY}");
        expect(validation.missingPlaceholders).toContain("${ACTION_BUTTON}");
        expect(validation.missingPlaceholders).toContain("${YEAR}");
    });
});
