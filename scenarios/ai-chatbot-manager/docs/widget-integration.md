# AI Chatbot Manager - Widget Integration Guide

## Overview

The AI Chatbot Manager provides embeddable JavaScript widgets that enable seamless integration of AI-powered chatbots into any website. This guide covers installation, configuration, customization, and advanced integration patterns.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Installation Methods](#installation-methods)
3. [Configuration Options](#configuration-options)
4. [Styling & Customization](#styling--customization)
5. [API Integration](#api-integration)
6. [Event Handling](#event-handling)
7. [Security Considerations](#security-considerations)
8. [Troubleshooting](#troubleshooting)
9. [Examples](#examples)

## Quick Start

### 1. Create Your Chatbot

First, create a chatbot using the CLI or API:

```bash
# Using CLI
visited-tracker create "Customer Support Bot" \
  --personality "You are a helpful customer support agent" \
  --model llama3.2

# Response will include chatbot ID and embed code
```

### 2. Embed the Widget

Add the generated embed code to your website's HTML:

```html
<!-- AI Chatbot Manager Widget -->
<script>
  window.AIChatbotConfig = {
    apiUrl: 'https://your-api-server.com',  // Replace with your API URL
    chatbotId: 'YOUR_CHATBOT_ID',          // Replace with your chatbot ID
    config: {
      theme: 'light',
      position: 'bottom-right',
      primaryColor: '#007bff'
    }
  };
</script>
<script src="https://your-api-server.com/api/v1/widget.js" async></script>
```

The widget will automatically appear in the configured position when the page loads.

## Installation Methods

### Method 1: Script Tag (Recommended)

The simplest installation method using a script tag:

```html
<script src="https://your-api-server.com/api/v1/widget.js" async></script>
```

### Method 2: NPM Package (Coming Soon)

For React/Vue/Angular applications:

```bash
npm install @vrooli/chatbot-widget
```

```javascript
import { ChatbotWidget } from '@vrooli/chatbot-widget';

ChatbotWidget.init({
  apiUrl: 'https://your-api-server.com',
  chatbotId: 'YOUR_CHATBOT_ID'
});
```

### Method 3: Dynamic Loading

Load the widget programmatically:

```javascript
function loadChatbot() {
  const script = document.createElement('script');
  script.src = 'https://your-api-server.com/api/v1/widget.js';
  script.async = true;
  script.onload = () => {
    console.log('Chatbot widget loaded successfully');
  };
  document.head.appendChild(script);
}

// Load when needed
document.addEventListener('DOMContentLoaded', loadChatbot);
```

## Configuration Options

### Basic Configuration

```javascript
window.AIChatbotConfig = {
  // Required
  apiUrl: 'https://your-api-server.com',
  chatbotId: 'abc123-def456',
  
  // Optional
  config: {
    // Widget appearance
    theme: 'light',              // 'light' | 'dark' | 'auto'
    position: 'bottom-right',    // 'bottom-right' | 'bottom-left' | 'top-right' | 'top-left'
    primaryColor: '#007bff',     // Any valid CSS color
    secondaryColor: '#6c757d',   // Any valid CSS color
    
    // Widget behavior
    autoOpen: false,              // Auto-open on page load
    openDelay: 3000,             // Delay before auto-open (ms)
    showLauncher: true,          // Show/hide launcher button
    fullscreen: false,           // Start in fullscreen mode
    
    // Chat settings
    placeholder: 'Type your message...',
    welcomeMessage: 'Hello! How can I help you today?',
    offlineMessage: 'We are currently offline. Please try again later.',
    
    // User settings
    requireEmail: false,         // Require email before chat
    requireName: false,         // Require name before chat
    persistSession: true,        // Remember conversation across pages
    
    // Advanced
    zIndex: 999999,             // Widget z-index
    enableSound: true,          // Enable notification sounds
    enableFileUpload: false,    // Allow file uploads
    maxMessageLength: 1000,     // Maximum message length
  }
};
```

### Advanced Configuration

```javascript
window.AIChatbotConfig = {
  apiUrl: 'https://your-api-server.com',
  chatbotId: 'abc123-def456',
  
  // Custom headers for API requests
  headers: {
    'X-API-Key': 'your-api-key',
    'X-Custom-Header': 'value'
  },
  
  // Custom metadata
  metadata: {
    userId: 'user-123',
    sessionId: 'session-456',
    pageUrl: window.location.href,
    referrer: document.referrer
  },
  
  // Callbacks
  onLoad: () => {
    console.log('Widget loaded');
  },
  onOpen: () => {
    console.log('Chat opened');
    // Track event in analytics
    gtag('event', 'chatbot_opened');
  },
  onClose: () => {
    console.log('Chat closed');
  },
  onMessage: (message) => {
    console.log('Message received:', message);
  },
  onError: (error) => {
    console.error('Widget error:', error);
  }
};
```

## Styling & Customization

### CSS Variables

Override CSS variables for fine-grained control:

```css
/* Add to your stylesheet */
.ai-chatbot-widget {
  --chatbot-primary-color: #007bff;
  --chatbot-secondary-color: #6c757d;
  --chatbot-text-color: #333333;
  --chatbot-bg-color: #ffffff;
  --chatbot-border-radius: 8px;
  --chatbot-font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  --chatbot-font-size: 14px;
  --chatbot-launcher-size: 60px;
  --chatbot-widget-width: 380px;
  --chatbot-widget-height: 600px;
}
```

### Custom Themes

Create completely custom themes:

```javascript
window.AIChatbotConfig = {
  // ... other config
  customTheme: {
    launcher: {
      backgroundColor: '#FF6B6B',
      iconColor: '#FFFFFF',
      size: '70px',
      borderRadius: '50%',
      boxShadow: '0 4px 12px rgba(0,0,0,0.15)'
    },
    header: {
      backgroundColor: '#4ECDC4',
      textColor: '#FFFFFF',
      height: '60px',
      fontSize: '18px',
      fontWeight: 'bold'
    },
    messages: {
      userBubbleColor: '#FF6B6B',
      userTextColor: '#FFFFFF',
      botBubbleColor: '#F7F7F7',
      botTextColor: '#333333',
      timestampColor: '#999999'
    },
    input: {
      backgroundColor: '#FFFFFF',
      borderColor: '#E0E0E0',
      textColor: '#333333',
      placeholderColor: '#999999',
      sendButtonColor: '#4ECDC4'
    }
  }
};
```

### Mobile Responsive Design

The widget automatically adapts to mobile devices:

```javascript
window.AIChatbotConfig = {
  // ... other config
  mobile: {
    fullscreen: true,           // Force fullscreen on mobile
    hideOnScroll: true,         // Hide launcher when scrolling
    breakpoint: 768,            // Mobile breakpoint (px)
    position: 'bottom-center'   // Mobile-specific position
  }
};
```

## API Integration

### JavaScript API

Control the widget programmatically:

```javascript
// Wait for widget to load
window.addEventListener('AIChatbotReady', () => {
  const chatbot = window.AIChatbot;
  
  // Open/close chat
  chatbot.open();
  chatbot.close();
  chatbot.toggle();
  
  // Send messages programmatically
  chatbot.sendMessage('Hello, I need help with my order');
  
  // Update configuration
  chatbot.updateConfig({
    theme: 'dark',
    primaryColor: '#FF6B6B'
  });
  
  // Set user information
  chatbot.setUser({
    name: 'John Doe',
    email: 'john@example.com',
    customerId: 'cust-123'
  });
  
  // Clear conversation
  chatbot.clearConversation();
  
  // Destroy widget
  chatbot.destroy();
});
```

### REST API Integration

Send messages directly via API:

```javascript
async function sendChatMessage(message) {
  const response = await fetch('https://your-api-server.com/api/v1/chat/YOUR_CHATBOT_ID', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-API-Key': 'your-api-key'
    },
    body: JSON.stringify({
      message: message,
      session_id: 'session-123',
      context: {
        page: window.location.pathname,
        user: 'user-123'
      }
    })
  });
  
  return response.json();
}
```

## Event Handling

### Available Events

Listen to widget events:

```javascript
// Widget loaded
window.addEventListener('AIChatbotReady', (event) => {
  console.log('Chatbot ready', event.detail);
});

// Chat opened
window.addEventListener('AIChatbotOpened', (event) => {
  console.log('Chat opened', event.detail);
});

// Chat closed
window.addEventListener('AIChatbotClosed', (event) => {
  console.log('Chat closed', event.detail);
});

// Message sent
window.addEventListener('AIChatbotMessageSent', (event) => {
  console.log('User message:', event.detail.message);
});

// Message received
window.addEventListener('AIChatbotMessageReceived', (event) => {
  console.log('Bot response:', event.detail.message);
});

// Lead captured
window.addEventListener('AIChatbotLeadCaptured', (event) => {
  console.log('Lead info:', event.detail.lead);
  // Send to your CRM
  sendToCRM(event.detail.lead);
});

// Error occurred
window.addEventListener('AIChatbotError', (event) => {
  console.error('Chatbot error:', event.detail.error);
});
```

### Custom Event Handlers

```javascript
window.AIChatbotConfig = {
  // ... other config
  events: {
    onConversationStart: (data) => {
      // Track conversation start
      analytics.track('Chatbot Conversation Started', data);
    },
    onConversationEnd: (data) => {
      // Track conversation end
      analytics.track('Chatbot Conversation Ended', {
        duration: data.duration,
        messageCount: data.messageCount
      });
    },
    onLeadQualified: (lead) => {
      // Handle qualified leads
      if (lead.qualificationScore > 0.7) {
        notifySalesTeam(lead);
      }
    },
    onEscalationRequired: (context) => {
      // Handle escalation to human agent
      createSupportTicket(context);
    }
  }
};
```

## Security Considerations

### Content Security Policy (CSP)

Add the following to your CSP headers:

```http
Content-Security-Policy: 
  script-src 'self' https://your-api-server.com;
  connect-src 'self' https://your-api-server.com wss://your-api-server.com;
  frame-src 'self' https://your-api-server.com;
  style-src 'self' 'unsafe-inline' https://your-api-server.com;
```

### CORS Configuration

Ensure your API server allows cross-origin requests:

```javascript
// API server configuration
app.use(cors({
  origin: ['https://your-website.com', 'https://www.your-website.com'],
  credentials: true,
  methods: ['GET', 'POST', 'OPTIONS'],
  allowedHeaders: ['Content-Type', 'X-API-Key']
}));
```

### API Key Security

Never expose API keys in client-side code:

```javascript
// ❌ BAD - API key exposed
window.AIChatbotConfig = {
  apiKey: 'sk-secret-key-123'  // Never do this!
};

// ✅ GOOD - Use proxy endpoint
window.AIChatbotConfig = {
  apiUrl: '/api/chatbot-proxy'  // Proxy handles authentication
};
```

### Data Privacy

Configure data handling preferences:

```javascript
window.AIChatbotConfig = {
  // ... other config
  privacy: {
    collectIP: false,           // Don't collect IP addresses
    collectUserAgent: false,    // Don't collect browser info
    anonymizeUser: true,        // Anonymize user data
    storageType: 'session',     // 'session' | 'local' | 'none'
    gdprCompliant: true,        // Enable GDPR features
    showPrivacyNotice: true,    // Show privacy notice
    privacyPolicyUrl: 'https://your-website.com/privacy'
  }
};
```

## Troubleshooting

### Common Issues

#### Widget Not Appearing

```javascript
// Check if config is set before script loads
console.log(window.AIChatbotConfig);  // Should show your config

// Check for errors
window.addEventListener('AIChatbotError', (e) => {
  console.error('Widget error:', e.detail);
});

// Verify API is accessible
fetch('https://your-api-server.com/health')
  .then(r => r.json())
  .then(console.log)
  .catch(console.error);
```

#### CORS Errors

```javascript
// Test CORS configuration
fetch('https://your-api-server.com/api/v1/chatbots/YOUR_CHATBOT_ID', {
  method: 'GET',
  headers: {
    'Content-Type': 'application/json'
  },
  mode: 'cors'
})
.then(response => console.log('CORS OK'))
.catch(error => console.error('CORS Error:', error));
```

#### WebSocket Connection Issues

```javascript
// Test WebSocket connection
const ws = new WebSocket('wss://your-api-server.com/api/v1/ws/YOUR_CHATBOT_ID');
ws.onopen = () => console.log('WebSocket connected');
ws.onerror = (error) => console.error('WebSocket error:', error);
ws.onclose = (event) => console.log('WebSocket closed:', event.code, event.reason);
```

### Debug Mode

Enable debug mode for detailed logging:

```javascript
window.AIChatbotConfig = {
  // ... other config
  debug: true,  // Enable debug logging
  logLevel: 'verbose'  // 'error' | 'warn' | 'info' | 'verbose'
};
```

## Examples

### E-commerce Integration

```html
<!DOCTYPE html>
<html>
<head>
  <title>Online Store</title>
</head>
<body>
  <!-- Your website content -->
  
  <script>
    // Configure chatbot for e-commerce
    window.AIChatbotConfig = {
      apiUrl: 'https://api.your-store.com',
      chatbotId: 'ecommerce-bot-123',
      config: {
        theme: 'light',
        position: 'bottom-right',
        primaryColor: '#28a745',
        welcomeMessage: 'Hi! Need help finding the perfect product?',
        placeholder: 'Ask about products, orders, or shipping...'
      },
      metadata: {
        cartValue: getCartValue(),
        customerId: getCustomerId(),
        currentProduct: getCurrentProduct()
      },
      events: {
        onLeadQualified: (lead) => {
          if (lead.purchaseIntent) {
            showPromoCode('CHAT10');
          }
        }
      }
    };
    
    function getCartValue() {
      // Return current cart value
      return localStorage.getItem('cartValue') || 0;
    }
    
    function getCustomerId() {
      // Return customer ID if logged in
      return localStorage.getItem('customerId') || null;
    }
    
    function getCurrentProduct() {
      // Return current product page info
      return {
        id: document.querySelector('[data-product-id]')?.dataset.productId,
        name: document.querySelector('h1.product-title')?.textContent,
        price: document.querySelector('.product-price')?.textContent
      };
    }
    
    function showPromoCode(code) {
      // Show promotional code to user
      alert(`Special offer! Use code ${code} for 10% off`);
    }
  </script>
  <script src="https://api.your-store.com/api/v1/widget.js" async></script>
</body>
</html>
```

### SaaS Platform Integration

```javascript
// Integration for SaaS dashboard
class ChatbotIntegration {
  constructor() {
    this.config = {
      apiUrl: process.env.REACT_APP_CHATBOT_API,
      chatbotId: process.env.REACT_APP_CHATBOT_ID,
      config: {
        theme: 'auto',  // Match app theme
        position: 'bottom-right',
        requireEmail: true,
        persistSession: true
      }
    };
    
    this.loadWidget();
    this.setupEventListeners();
  }
  
  loadWidget() {
    window.AIChatbotConfig = this.config;
    
    const script = document.createElement('script');
    script.src = `${this.config.apiUrl}/api/v1/widget.js`;
    script.async = true;
    document.head.appendChild(script);
  }
  
  setupEventListeners() {
    // Update user context when they log in
    window.addEventListener('userLogin', (event) => {
      if (window.AIChatbot) {
        window.AIChatbot.setUser({
          id: event.detail.userId,
          name: event.detail.userName,
          email: event.detail.userEmail,
          plan: event.detail.userPlan
        });
      }
    });
    
    // Track support interactions
    window.addEventListener('AIChatbotMessageSent', (event) => {
      analytics.track('Support Chat Message', {
        message: event.detail.message,
        timestamp: new Date().toISOString()
      });
    });
    
    // Handle escalations
    window.addEventListener('AIChatbotEscalationRequired', (event) => {
      this.createSupportTicket(event.detail);
    });
  }
  
  createSupportTicket(details) {
    fetch('/api/support/tickets', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${getAuthToken()}`
      },
      body: JSON.stringify({
        source: 'chatbot',
        conversation_id: details.conversationId,
        messages: details.messages,
        user_id: details.userId,
        priority: 'high'
      })
    });
  }
}

// Initialize on app load
document.addEventListener('DOMContentLoaded', () => {
  new ChatbotIntegration();
});
```

### WordPress Plugin Integration

```php
<?php
/**
 * AI Chatbot Manager WordPress Integration
 */

function ai_chatbot_enqueue_scripts() {
    // Get settings from WordPress options
    $api_url = get_option('ai_chatbot_api_url');
    $chatbot_id = get_option('ai_chatbot_id');
    $config = get_option('ai_chatbot_config', array());
    
    // Add configuration
    wp_add_inline_script('ai-chatbot-config', 
        'window.AIChatbotConfig = ' . json_encode(array(
            'apiUrl' => $api_url,
            'chatbotId' => $chatbot_id,
            'config' => $config,
            'metadata' => array(
                'site' => get_bloginfo('name'),
                'page' => get_the_title(),
                'user' => is_user_logged_in() ? wp_get_current_user()->user_email : null
            )
        )) . ';',
        'before'
    );
    
    // Load widget script
    wp_enqueue_script(
        'ai-chatbot-widget',
        $api_url . '/api/v1/widget.js',
        array(),
        '1.0.0',
        true
    );
}

add_action('wp_enqueue_scripts', 'ai_chatbot_enqueue_scripts');

// Add settings page
function ai_chatbot_settings_page() {
    ?>
    <div class="wrap">
        <h1>AI Chatbot Settings</h1>
        <form method="post" action="options.php">
            <?php settings_fields('ai_chatbot_settings'); ?>
            <table class="form-table">
                <tr>
                    <th scope="row">API URL</th>
                    <td>
                        <input type="text" name="ai_chatbot_api_url" 
                               value="<?php echo get_option('ai_chatbot_api_url'); ?>" />
                    </td>
                </tr>
                <tr>
                    <th scope="row">Chatbot ID</th>
                    <td>
                        <input type="text" name="ai_chatbot_id" 
                               value="<?php echo get_option('ai_chatbot_id'); ?>" />
                    </td>
                </tr>
            </table>
            <?php submit_button(); ?>
        </form>
    </div>
    <?php
}
?>
```

## Support & Resources

### Documentation
- [API Documentation](./API.md)
- [CLI Reference](../cli/README.md)
- [PRD Specification](../PRD.md)

### Getting Help
- CLI: `ai-chatbot-manager help`
- API Health Check: `GET /health`
- Debug Mode: Set `debug: true` in configuration

### Contributing
For bug reports and feature requests, please contact the development team or submit through the appropriate channels.

---

**Last Updated**: 2025-01-13  
**Version**: 1.0.0  
**Status**: Production Ready