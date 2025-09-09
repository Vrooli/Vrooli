# Comment System Integration Guide

The Universal Comment System provides multiple integration methods to add collaborative discussion features to any Vrooli scenario. This guide covers all integration patterns and best practices.

## Quick Start (Recommended)

### 1. Drop-in Widget

The fastest way to add comments to your scenario:

```html
<!-- Add anywhere in your HTML -->
<div data-vrooli-comments="your-scenario-name"></div>

<!-- Include the SDK -->
<script src="http://localhost:3100/sdk/vrooli-comments.js"></script>
```

That's it! Comments will automatically appear with default settings.

### 2. Custom Configuration

```html
<div 
  data-vrooli-comments="your-scenario-name"
  data-theme="dark"
  data-sort-by="threaded"
  data-page-size="10"
  data-allow-anonymous="true"
  data-show-replies="true">
</div>
```

### 3. JavaScript Initialization

```javascript
VrooliComments.init({
    scenarioId: 'your-scenario-name',
    container: '#comments-container',
    theme: 'default',
    authToken: userSessionToken,
    sortBy: 'newest',
    pageSize: 20
});
```

## Integration Methods

### Method 1: Widget Integration (Easiest)

Perfect for most scenarios that just need comments.

```html
<!DOCTYPE html>
<html>
<head>
    <title>My Scenario</title>
</head>
<body>
    <h1>My Awesome Scenario</h1>
    <p>Main content here...</p>
    
    <!-- Comments section -->
    <section class="comments-section">
        <div data-vrooli-comments="my-scenario"></div>
    </section>
    
    <!-- Include SDK at end of body -->
    <script src="http://localhost:3100/sdk/vrooli-comments.js"></script>
</body>
</html>
```

### Method 2: JavaScript SDK (Flexible)

Use this when you need programmatic control.

```javascript
// Initialize comment widget
const comments = VrooliComments.init({
    scenarioId: 'my-scenario',
    container: '#custom-comments',
    theme: 'default',
    authToken: getCurrentUserToken(),
    
    // Callbacks
    onCommentAdded: (comment) => {
        console.log('New comment:', comment);
        updateCommentCount();
    },
    
    onError: (error) => {
        console.error('Comment error:', error);
        showErrorMessage(error);
    }
});

// Update authentication when user logs in
function onUserLogin(token) {
    VrooliComments.setAuth(token);
}

// Refresh comments when needed
function refreshComments() {
    VrooliComments.refresh();
}

// Add comment programmatically
function addQuickComment(text) {
    VrooliComments.addComment('my-scenario', text);
}
```

### Method 3: Direct API Integration (Advanced)

For custom implementations or non-web scenarios.

```javascript
// Get comments
async function getComments(scenarioName, page = 0, limit = 20) {
    const response = await fetch(
        `http://localhost:8080/api/v1/comments/${scenarioName}?limit=${limit}&offset=${page * limit}`
    );
    
    if (!response.ok) {
        throw new Error(`Failed to fetch comments: ${response.statusText}`);
    }
    
    return await response.json();
}

// Create comment
async function createComment(scenarioName, content, authToken, parentId = null) {
    const payload = {
        content: content,
        content_type: 'markdown',
        author_token: authToken
    };
    
    if (parentId) {
        payload.parent_id = parentId;
    }
    
    const response = await fetch(`http://localhost:8080/api/v1/comments/${scenarioName}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify(payload)
    });
    
    if (!response.ok) {
        throw new Error(`Failed to create comment: ${response.statusText}`);
    }
    
    return await response.json();
}

// Get scenario configuration
async function getScenarioConfig(scenarioName) {
    const response = await fetch(`http://localhost:8080/api/v1/config/${scenarioName}`);
    
    if (!response.ok) {
        throw new Error(`Failed to get config: ${response.statusText}`);
    }
    
    return await response.json();
}
```

### Method 4: CLI Integration (Server-side)

Integrate comments into server-side scenarios or automation.

```bash
#!/bin/bash

# Check if comments are configured for scenario
if comment-system config my-scenario --json | jq -e '.config.auth_required'; then
    echo "Authentication required for comments"
fi

# List recent comments
echo "Recent comments:"
comment-system list my-scenario table 5

# Create a comment from automation
comment-system create my-scenario "Automated status update: Process completed successfully"

# Moderate flagged comments
comment-system moderate spam-comment-id delete
```

## Configuration Options

### Scenario Configuration

Configure comment behavior per scenario via the admin dashboard or API:

```javascript
// Update scenario configuration
const config = {
    auth_required: true,        // Require user authentication
    allow_anonymous: false,     // Allow anonymous comments  
    allow_rich_media: true,     // Enable image/file attachments
    moderation_level: 'manual', // none, manual, ai_assisted
    theme_config: {
        theme: 'default',
        show_avatars: true,
        enable_voting: false
    },
    notification_settings: {
        mentions: true,         // Notify on @mentions
        replies: true,          // Notify on replies
        new_comments: false     // Notify on all new comments
    }
};

await fetch(`http://localhost:8080/api/v1/config/my-scenario`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config)
});
```

### Widget Configuration

Customize the comment widget appearance and behavior:

```javascript
VrooliComments.init({
    scenarioId: 'my-scenario',
    container: '#comments',
    
    // Authentication
    authToken: 'user-session-token',
    allowAnonymous: false,
    loginUrl: '/login',
    
    // Display options
    theme: 'default',           // default, dark, custom
    sortBy: 'newest',          // newest, oldest, threaded
    pageSize: 20,              // Comments per page
    showReplies: true,         // Show reply functionality
    maxDepth: 5,               // Maximum reply nesting
    
    // Content options
    enableMarkdown: true,      // Allow markdown formatting
    enableRichMedia: true,     // Allow file attachments
    placeholder: 'Share your thoughts...',
    
    // Real-time options
    autoRefresh: false,        // Auto-refresh comments
    refreshInterval: 30000,    // Refresh every 30 seconds
    
    // Event callbacks
    onCommentAdded: (comment) => console.log('Added:', comment),
    onCommentUpdated: (comment) => console.log('Updated:', comment),  
    onCommentDeleted: (commentId) => console.log('Deleted:', commentId),
    onError: (error) => console.error('Error:', error)
});
```

## Authentication Integration

The comment system integrates with the `session-authenticator` scenario for user management.

### Getting User Tokens

```javascript
// Assuming session-authenticator provides these functions
async function getUserSession() {
    const response = await fetch('/api/auth/session');
    const session = await response.json();
    return session.token;
}

// Initialize comments with user token
const userToken = await getUserSession();
VrooliComments.setAuth(userToken);
```

### Handling Authentication State Changes

```javascript
// Listen for authentication events
window.addEventListener('user-login', (event) => {
    VrooliComments.setAuth(event.detail.token);
});

window.addEventListener('user-logout', (event) => {
    VrooliComments.setAuth(null);
    VrooliComments.refresh(); // Refresh to show anonymous state
});
```

### Anonymous Comments

Configure scenarios to allow anonymous commenting:

```javascript
// Enable anonymous comments for a scenario
await fetch(`http://localhost:8080/api/v1/config/public-demo`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
        auth_required: false,
        allow_anonymous: true
    })
});
```

## Theming and Customization

### Built-in Themes

```javascript
// Available themes
VrooliComments.init({
    scenarioId: 'my-scenario',
    container: '#comments',
    theme: 'default'  // default, dark, light, minimal
});
```

### Custom CSS

Override styles with custom CSS:

```css
/* Customize comment widget appearance */
.vrooli-comments.theme-custom {
    --primary-color: #your-brand-color;
    --background-color: #your-bg-color;
    --text-color: #your-text-color;
    --border-color: #your-border-color;
}

.vrooli-comments.theme-custom .comment {
    border-left: 3px solid var(--primary-color);
    padding-left: 1rem;
}

.vrooli-comments.theme-custom .comment-author {
    color: var(--primary-color);
    font-weight: 600;
}
```

### Dynamic Theming

```javascript
// Change theme programmatically
VrooliComments.configure({
    theme: 'dark'
});

// Match your scenario's theme
function updateCommentsTheme(isDarkMode) {
    VrooliComments.configure({
        theme: isDarkMode ? 'dark' : 'default'
    });
}
```

## Advanced Features

### Real-time Updates

```javascript
VrooliComments.init({
    scenarioId: 'my-scenario',
    container: '#comments',
    autoRefresh: true,
    refreshInterval: 15000, // 15 seconds
    
    onCommentAdded: (comment) => {
        // Show notification for new comments
        if (comment.author_id !== getCurrentUserId()) {
            showNotification(`New comment from ${comment.author_name}`);
        }
    }
});
```

### Comment Moderation

```javascript
// Admin moderation interface
if (isUserAdmin()) {
    VrooliComments.init({
        scenarioId: 'my-scenario',
        container: '#comments',
        showModerationTools: true,
        
        onModerationAction: (commentId, action) => {
            console.log(`Comment ${commentId} ${action}`);
            logModerationAction(commentId, action);
        }
    });
}
```

### Analytics Integration

```javascript
VrooliComments.init({
    scenarioId: 'my-scenario',
    container: '#comments',
    
    onCommentAdded: (comment) => {
        // Track engagement
        analytics.track('Comment Added', {
            scenarioId: 'my-scenario',
            commentLength: comment.content.length,
            isReply: !!comment.parent_id
        });
    }
});
```

## Troubleshooting

### Common Issues

**1. Comments not loading**
```javascript
// Check API connectivity
VrooliComments.init({
    scenarioId: 'my-scenario',
    container: '#comments',
    onError: (error) => {
        console.error('Comment system error:', error);
        // Check if API is running and accessible
    }
});
```

**2. Authentication not working**
```javascript
// Verify token is valid
const token = getCurrentUserToken();
console.log('Using token:', token);

// Test authentication
fetch('http://localhost:8080/api/v1/comments/test', {
    headers: { 'Authorization': `Bearer ${token}` }
}).then(response => {
    if (!response.ok) {
        console.error('Authentication failed');
    }
});
```

**3. Styling conflicts**
```css
/* Ensure comment widget styles take precedence */
.vrooli-comments {
    all: initial;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}

.vrooli-comments * {
    box-sizing: border-box;
}
```

### Debug Mode

```javascript
// Enable debug logging
window.VROOLI_COMMENTS_DEBUG = true;

VrooliComments.init({
    scenarioId: 'my-scenario',
    container: '#comments',
    debug: true  // Enable detailed logging
});
```

### Health Checks

```bash
# Check system health
comment-system status --verbose

# Test specific scenario
comment-system list my-scenario json 1

# Verify configuration
comment-system config my-scenario
```

## Best Practices

### 1. Scenario Setup

```yaml
# In your scenario's .vrooli/service.json
{
  "dependencies": {
    "scenarios": [
      {
        "name": "comment-system",
        "required": true,
        "purpose": "Collaborative discussion features"
      },
      {
        "name": "session-authenticator", 
        "required": true,
        "purpose": "User authentication for comments"
      }
    ]
  }
}
```

### 2. Error Handling

```javascript
VrooliComments.init({
    scenarioId: 'my-scenario',
    container: '#comments',
    
    onError: (error) => {
        // Graceful error handling
        const errorContainer = document.getElementById('comment-errors');
        errorContainer.innerHTML = `
            <div class="alert alert-warning">
                <strong>Comments temporarily unavailable.</strong>
                <br>Please try refreshing the page.
                <button onclick="location.reload()">Refresh</button>
            </div>
        `;
    }
});
```

### 3. Performance

```javascript
// Lazy load comments for better initial page load
const commentsContainer = document.getElementById('comments');
const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
        if (entry.isIntersecting) {
            VrooliComments.init({
                scenarioId: 'my-scenario',
                container: '#comments'
            });
            observer.disconnect();
        }
    });
});
observer.observe(commentsContainer);
```

### 4. Accessibility

```html
<!-- Ensure proper semantic structure -->
<section aria-label="Comments" role="region">
    <h2 id="comments-heading">Discussion</h2>
    <div 
        data-vrooli-comments="my-scenario"
        aria-labelledby="comments-heading">
    </div>
</section>
```

### 5. Content Security

```javascript
// Sanitize user content (handled automatically by SDK)
VrooliComments.init({
    scenarioId: 'my-scenario',
    container: '#comments',
    enableMarkdown: true,  // Safe markdown rendering
    allowHTML: false,      // Block raw HTML
    sanitizeContent: true  // Enable content sanitization
});
```

## API Reference

### REST Endpoints

```
GET    /api/v1/comments/{scenario}     # List comments
POST   /api/v1/comments/{scenario}     # Create comment
PUT    /api/v1/comments/{id}           # Update comment
DELETE /api/v1/comments/{id}           # Delete comment
GET    /api/v1/config/{scenario}       # Get configuration
POST   /api/v1/config/{scenario}       # Update configuration
```

### CLI Commands

```bash
comment-system status                  # System health
comment-system list <scenario>         # List comments
comment-system create <scenario> <content>  # Create comment
comment-system config <scenario>       # Manage configuration
comment-system moderate <id> <action>  # Moderate comments
```

### JavaScript SDK Methods

```javascript
VrooliComments.init(options)           // Initialize widget
VrooliComments.configure(newConfig)    // Update configuration
VrooliComments.setAuth(token)          // Set authentication
VrooliComments.refresh()               // Refresh all widgets
VrooliComments.getInstance(container)  // Get widget instance
VrooliComments.addComment(scenario, content)  // Add comment programmatically
```

## Support

For issues, questions, or feature requests:

1. Check the [Comment System Admin Dashboard](http://localhost:3100)
2. Review API health at `/health` endpoint
3. Enable debug logging for detailed error information
4. Check scenario configuration via CLI or admin dashboard

The Universal Comment System is designed to be robust and self-healing, with graceful fallbacks for common failure scenarios.