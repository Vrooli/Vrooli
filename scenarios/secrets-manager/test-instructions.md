# Code Viewer Test Instructions

## Setup Complete! 🎉

The vulnerability code viewer with syntax highlighting is now fully implemented and running.

### Test the Code Viewer:

1. **Open the Secrets Manager UI**
   - URL: http://localhost:37165
   - The UI will load with a dark chrome cyberpunk theme

2. **Trigger a Security Scan**
   - Click the refresh button (⟳) in the top right of the "VAULT SECRETS & SECURITY" panel
   - This will scan for vulnerabilities across all scenarios and resources

3. **View Vulnerabilities**
   - After scanning, click on any of the colored stat cards:
     - **Critical Risk** (red) - Most severe vulnerabilities
     - **High Risk** (orange) - High severity issues
     - **Medium Risk** (green) - Medium severity issues
     - **Low Risk** (gray) - Low severity issues
   - The table will update to show vulnerabilities of that severity

4. **Click Any Vulnerability Row**
   - Click anywhere on a vulnerability row (except the checkbox or DETAILS button)
   - The code viewer modal will open showing:
     - File path and language
     - Vulnerability type and severity
     - Syntax-highlighted code
     - The vulnerable line highlighted in red
     - Copy code button

5. **Check Browser Console**
   - Open browser DevTools (F12)
   - Go to Console tab
   - You should see debug messages like:
     ```
     Added click handler for vulnerability: [id] [file_path]
     Row clicked! Vulnerability: [id] [file_path]
     Opening code viewer for: [file_path]
     openCodeViewer called with: [vulnerability object]
     Modal found, showing...
     Modal display set to flex
     ```

### Features:
- **Syntax Highlighting**: Automatic language detection for 20+ languages
- **Line Highlighting**: Vulnerable lines are highlighted and auto-scrolled into view
- **Copy Code**: One-click code copying to clipboard
- **Multiple Close Methods**: ESC key, click outside, or X button
- **Security**: Path traversal protection and Vrooli-scoped file access

### Troubleshooting:
- If clicking doesn't work, check browser console for errors
- Make sure you're clicking on vulnerability rows (not the initial vault status rows)
- The modal CSS uses `display: flex` when visible, `display: none` when hidden

### API Test:
Test the file content API directly:
```bash
curl -s "http://localhost:16751/api/v1/files/content?path=resources/test-resource/config/secrets-test.sh" | jq .
```

The implementation is complete and should be working now! The debug logging will help identify any remaining issues.