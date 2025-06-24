/**
 * Email formatting utilities for notification system
 */

const EMAIL_STYLES = {
    paragraph: "Margin:0;-webkit-text-size-adjust:none;-ms-text-size-adjust:none;font-family:lato, 'helvetica neue', helvetica, arial, sans-serif;line-height:27px;color:#666666;font-size:18px",
    button: {
        border: "border-style:solid;border-color:#2074c5;background:1px;border-width:1px;display:inline-block;border-radius:2px;width:auto",
        link: "text-decoration:none;-webkit-text-size-adjust:none;-ms-text-size-adjust:none;color:#FFFFFF;font-size:20px;border-style:solid;border-color:#2074c5;border-width:15px 30px;display:inline-block;background:#2074c5;border-radius:2px;font-family:helvetica, 'helvetica neue', arial, verdana, sans-serif;font-weight:normal;font-style:normal;line-height:24px;width:auto;text-align:center",
    },
    row: "border-collapse:collapse",
    cell: "Margin:0;padding-left:10px;padding-right:10px;padding-top:20px;padding-bottom:20px",
} as const;

/**
 * Escapes HTML special characters to prevent injection
 */
function escapeHtml(text: string): string {
    if (!text) return "";
    
    const htmlEscapes: { [key: string]: string } = {
        "&": "&amp;",
        "<": "&lt;",
        ">": "&gt;",
        "\"": "&quot;",
        "'": "&#x27;",
        "/": "&#x2F;",
    };
    return text.replace(/[&<>"'/]/g, (match) => htmlEscapes[match]);
}

/**
 * Converts plain text to HTML, preserving line breaks and escaping HTML
 */
function textToHtml(text: string): string {
    if (!text) {
        return `<p style="${EMAIL_STYLES.paragraph}"></p>`;
    }
    
    // Escape HTML first
    const escaped = escapeHtml(text);
    // Convert line breaks to <br> tags
    const withBreaks = escaped.replace(/\n/g, "<br>");
    // Wrap in paragraph tags
    return `<p style="${EMAIL_STYLES.paragraph}">${withBreaks}</p>`;
}

/**
 * Generates the action button HTML if a link is provided
 */
function generateActionButton(link: string): string {
    return `
                     <tr style="${EMAIL_STYLES.row}"> 
                      <td align="center" style="${EMAIL_STYLES.cell}"><span class="es-button-border" style="${EMAIL_STYLES.button.border}"><a href="${escapeHtml(link)}" class="es-button" target="_blank" style="${EMAIL_STYLES.button.link}">View Details</a></span></td> 
                     </tr>`;
}

/**
 * Formats a notification email using the HTML template
 * @param template The HTML template with placeholders
 * @param title The notification title
 * @param body The notification body text
 * @param link Optional link for action button
 * @returns Formatted HTML email
 */
export function formatNotificationEmail(
    template: string,
    title: string,
    body: string,
    link?: string,
): string {
    if (!template) {
        throw new Error("Email template is required");
    }
    if (!title && !body) {
        throw new Error("Either title or body must be provided");
    }
    
    // Escape title for HTML
    const escapedTitle = escapeHtml(title || "");
    
    // Convert body to HTML format
    const htmlBody = textToHtml(body || "");
    
    // Generate action button if link is provided
    const actionButton = link ? generateActionButton(link) : "";
    
    // Replace placeholders in template
    const formattedHtml = template
        .replace(/\$\{TITLE\}/g, escapedTitle)
        .replace(/\$\{BODY\}/g, htmlBody)
        .replace(/\$\{ACTION_BUTTON\}/g, actionButton)
        .replace(/\$\{YEAR\}/g, new Date().getFullYear().toString());
    
    return formattedHtml;
}

/**
 * Validates that a template contains the required placeholders
 */
export function validateEmailTemplate(template: string): { isValid: boolean; missingPlaceholders: string[] } {
    const requiredPlaceholders = ["${TITLE}", "${BODY}", "${ACTION_BUTTON}", "${YEAR}"];
    const missingPlaceholders = requiredPlaceholders.filter(
        placeholder => !template.includes(placeholder),
    );
    
    return {
        isValid: missingPlaceholders.length === 0,
        missingPlaceholders,
    };
}
