import { describe, expect, it, vi } from "vitest";
import { formatNotificationEmail } from "./formatters.js";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

describe("Notification Email Integration", () => {
    it("should format email with actual template", () => {
        const dirname = path.dirname(fileURLToPath(import.meta.url));
        const templatePath = path.join(dirname, "../tasks/email/templates/notification.html");
        
        // Skip test if template doesn't exist
        if (!fs.existsSync(templatePath)) {
            console.log("Skipping integration test - template file not found");
            return;
        }
        
        const template = fs.readFileSync(templatePath, "utf8");
        
        const result = formatNotificationEmail(
            template,
            "Test Notification",
            "This is a test notification message.\n\nIt has multiple lines.",
            "https://vrooli.com/notifications/123",
        );
        
        // Verify all key elements are present
        expect(result).toContain("<title>Test Notification</title>");
        expect(result).toContain("<h1 style=");
        expect(result).toContain("Test Notification</h1>");
        expect(result).toContain("This is a test notification message.<br><br>It has multiple lines.");
        expect(result).toContain("View Details");
        expect(result).toContain("href=\"https:&#x2F;&#x2F;vrooli.com&#x2F;notifications&#x2F;123\"");
        expect(result).toContain(`© ${new Date().getFullYear()} Vrooli`);
        expect(result).toContain("Best regards,<br>The Vrooli Team");
    });
    
    it("should format email without action button", () => {
        const dirname = path.dirname(fileURLToPath(import.meta.url));
        const templatePath = path.join(dirname, "../tasks/email/templates/notification.html");
        
        if (!fs.existsSync(templatePath)) {
            console.log("Skipping integration test - template file not found");
            return;
        }
        
        const template = fs.readFileSync(templatePath, "utf8");
        
        const result = formatNotificationEmail(
            template,
            "Simple Notification",
            "Just a simple message without any action required.",
        );
        
        expect(result).toContain("Simple Notification");
        expect(result).toContain("Just a simple message");
        expect(result).not.toContain("View Details");
        expect(result).toContain(`© ${new Date().getFullYear()} Vrooli`);
    });
});
