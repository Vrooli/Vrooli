-- SaaS Landing Manager Seed Data
-- Base templates and initial configuration for the landing page system

-- Insert base templates for different SaaS types
INSERT INTO templates (id, name, category, saas_type, industry, html_content, css_content, js_content, config_schema, preview_url, usage_count, rating) VALUES

-- B2B Tool Template
(
    'b2b-tool-template',
    'Professional B2B SaaS',
    'base',
    'b2b_tool',
    'general',
    '<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{title}} | Professional {{industry}} Solution</title>
    <meta name="description" content="{{description}}">
    <link rel="stylesheet" href="styles.css">
</head>
<body>
    <header class="header">
        <nav class="nav">
            <div class="logo">{{company_name}}</div>
            <div class="nav-links">
                <a href="#features">Features</a>
                <a href="#pricing">Pricing</a>
                <a href="#contact">Contact</a>
            </div>
        </nav>
    </header>
    
    <section class="hero">
        <div class="hero-content">
            <h1>{{hero_title}}</h1>
            <p class="hero-subtitle">{{hero_description}}</p>
            <div class="cta-buttons">
                <button class="cta-primary" onclick="trackClick(''primary_cta'')">{{primary_cta_text}}</button>
                <button class="cta-secondary" onclick="trackClick(''secondary_cta'')">{{secondary_cta_text}}</button>
            </div>
        </div>
        <div class="hero-image">
            <img src="{{hero_image_url}}" alt="{{company_name}} Dashboard">
        </div>
    </section>
    
    <section id="features" class="features">
        <h2>Key Features</h2>
        <div class="features-grid">
            {{#each features}}
            <div class="feature-card">
                <div class="feature-icon">{{icon}}</div>
                <h3>{{title}}</h3>
                <p>{{description}}</p>
            </div>
            {{/each}}
        </div>
    </section>
    
    <section id="pricing" class="pricing">
        <h2>Simple, Transparent Pricing</h2>
        <div class="pricing-cards">
            {{#each pricing_tiers}}
            <div class="pricing-card {{#if featured}}featured{{/if}}">
                <h3>{{name}}</h3>
                <div class="price">${{price}}<span>/month</span></div>
                <ul class="features-list">
                    {{#each features}}
                    <li>{{this}}</li>
                    {{/each}}
                </ul>
                <button class="cta-pricing" onclick="trackClick(''pricing_cta'', ''{{name}}'')">Get Started</button>
            </div>
            {{/each}}
        </div>
    </section>
    
    <section id="contact" class="contact">
        <h2>Ready to Get Started?</h2>
        <form class="contact-form" onsubmit="trackConversion(event)">
            <input type="email" placeholder="Your email" required>
            <button type="submit">Request Demo</button>
        </form>
    </section>
    
    <footer class="footer">
        <p>&copy; 2024 {{company_name}}. All rights reserved.</p>
    </footer>
    
    <script src="script.js"></script>
</body>
</html>',
    
    '/* B2B SaaS Professional Styles */
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: "Inter", -apple-system, BlinkMacSystemFont, sans-serif;
    line-height: 1.6;
    color: #333;
}

.header {
    background: #fff;
    box-shadow: 0 2px 10px rgba(0,0,0,0.1);
    position: sticky;
    top: 0;
    z-index: 100;
}

.nav {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 5%;
    max-width: 1200px;
    margin: 0 auto;
}

.logo {
    font-size: 1.5rem;
    font-weight: bold;
    color: #2563eb;
}

.nav-links a {
    text-decoration: none;
    color: #333;
    margin-left: 2rem;
    transition: color 0.3s;
}

.nav-links a:hover {
    color: #2563eb;
}

.hero {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 4rem;
    padding: 5rem 5%;
    max-width: 1200px;
    margin: 0 auto;
    align-items: center;
}

.hero h1 {
    font-size: 3rem;
    font-weight: bold;
    color: #1f2937;
    margin-bottom: 1rem;
}

.hero-subtitle {
    font-size: 1.2rem;
    color: #6b7280;
    margin-bottom: 2rem;
}

.cta-buttons {
    display: flex;
    gap: 1rem;
}

.cta-primary, .cta-secondary, .cta-pricing {
    padding: 0.75rem 2rem;
    border-radius: 0.5rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s;
}

.cta-primary {
    background: #2563eb;
    color: white;
    border: none;
}

.cta-primary:hover {
    background: #1d4ed8;
    transform: translateY(-2px);
}

.cta-secondary {
    background: transparent;
    color: #2563eb;
    border: 2px solid #2563eb;
}

.features {
    padding: 5rem 5%;
    background: #f9fafb;
}

.features h2 {
    text-align: center;
    font-size: 2.5rem;
    margin-bottom: 3rem;
    color: #1f2937;
}

.features-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 2rem;
    max-width: 1200px;
    margin: 0 auto;
}

.feature-card {
    background: white;
    padding: 2rem;
    border-radius: 0.5rem;
    box-shadow: 0 4px 6px rgba(0,0,0,0.1);
    text-align: center;
}

.pricing {
    padding: 5rem 5%;
}

.pricing h2 {
    text-align: center;
    font-size: 2.5rem;
    margin-bottom: 3rem;
}

.pricing-cards {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 2rem;
    max-width: 1000px;
    margin: 0 auto;
}

.pricing-card {
    background: white;
    border: 2px solid #e5e7eb;
    border-radius: 0.5rem;
    padding: 2rem;
    text-align: center;
    transition: transform 0.3s;
}

.pricing-card.featured {
    border-color: #2563eb;
    transform: scale(1.05);
}

.pricing-card:hover {
    transform: translateY(-5px);
}

.contact {
    padding: 5rem 5%;
    background: #2563eb;
    color: white;
    text-align: center;
}

.contact-form {
    display: flex;
    gap: 1rem;
    justify-content: center;
    margin-top: 2rem;
}

.contact-form input {
    padding: 0.75rem;
    border: none;
    border-radius: 0.5rem;
    width: 300px;
}

.contact-form button {
    background: white;
    color: #2563eb;
    border: none;
    padding: 0.75rem 2rem;
    border-radius: 0.5rem;
    font-weight: 600;
    cursor: pointer;
}

.footer {
    padding: 2rem;
    text-align: center;
    background: #1f2937;
    color: white;
}

@media (max-width: 768px) {
    .hero {
        grid-template-columns: 1fr;
        gap: 2rem;
        text-align: center;
    }
    
    .hero h1 {
        font-size: 2rem;
    }
    
    .contact-form {
        flex-direction: column;
        align-items: center;
    }
    
    .contact-form input {
        width: 100%;
        max-width: 300px;
    }
}',
    
    '// B2B SaaS Landing Page JavaScript
let analytics = {
    page_views: 0,
    clicks: {},
    conversions: 0
};

// Track page view
analytics.page_views++;

function trackClick(element, variant = null) {
    const key = variant ? `${element}_${variant}` : element;
    analytics.clicks[key] = (analytics.clicks[key] || 0) + 1;
    
    // Send to analytics endpoint
    fetch("/api/v1/analytics/track", {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify({
            event: "click",
            element: key,
            timestamp: new Date().toISOString()
        })
    }).catch(console.error);
}

function trackConversion(event) {
    event.preventDefault();
    analytics.conversions++;
    
    // Send conversion event
    fetch("/api/v1/analytics/track", {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify({
            event: "conversion",
            form: "contact",
            timestamp: new Date().toISOString()
        })
    }).then(() => {
        alert("Thank you! We''ll be in touch soon.");
    }).catch(console.error);
}

// A/B Test Traffic Routing
function getVariant() {
    const hash = location.hash.replace("#", "");
    if (hash && ["a", "b", "control"].includes(hash)) {
        return hash;
    }
    
    // Random assignment for new visitors
    const variants = ["control", "a", "b"];
    const random = Math.random();
    if (random < 0.33) return "control";
    if (random < 0.66) return "a";
    return "b";
}

// Load variant-specific content
const variant = getVariant();
document.body.setAttribute("data-variant", variant);

// Performance monitoring
window.addEventListener("load", function() {
    // Core Web Vitals tracking would go here
    const loadTime = performance.now();
    
    fetch("/api/v1/analytics/performance", {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify({
            load_time: loadTime,
            variant: variant,
            timestamp: new Date().toISOString()
        })
    }).catch(console.error);
});',
    
    '{
        "title": {"type": "string", "required": true, "default": "Professional SaaS Solution"},
        "description": {"type": "string", "required": true, "default": "Streamline your business operations with our powerful platform"},
        "company_name": {"type": "string", "required": true, "default": "Your Company"},
        "hero_title": {"type": "string", "required": true, "default": "Transform Your Business Operations"},
        "hero_description": {"type": "string", "required": true, "default": "Powerful, intuitive tools designed for modern businesses"},
        "hero_image_url": {"type": "string", "default": "/images/dashboard-preview.png"},
        "primary_cta_text": {"type": "string", "default": "Start Free Trial"},
        "secondary_cta_text": {"type": "string", "default": "View Demo"},
        "features": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "icon": {"type": "string"},
                    "title": {"type": "string"},
                    "description": {"type": "string"}
                }
            }
        },
        "pricing_tiers": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "name": {"type": "string"},
                    "price": {"type": "number"},
                    "features": {"type": "array", "items": {"type": "string"}},
                    "featured": {"type": "boolean", "default": false}
                }
            }
        }
    }',
    '/preview/b2b-tool-template',
    0,
    0.0
),

-- B2C App Template
(
    'b2c-app-template',
    'Consumer App Landing',
    'base',
    'b2c_app',
    'general',
    '<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{title}} | {{tagline}}</title>
    <meta name="description" content="{{description}}">
    <link rel="stylesheet" href="styles.css">
</head>
<body>
    <section class="hero">
        <div class="hero-content">
            <h1 class="hero-title">{{hero_title}}</h1>
            <p class="hero-subtitle">{{hero_subtitle}}</p>
            <div class="app-badges">
                <a href="{{ios_url}}" class="app-badge">
                    <img src="/images/app-store-badge.png" alt="Download on App Store">
                </a>
                <a href="{{android_url}}" class="app-badge">
                    <img src="/images/google-play-badge.png" alt="Get it on Google Play">
                </a>
            </div>
        </div>
        <div class="hero-visual">
            <img src="{{app_preview_url}}" alt="{{title}} App Preview" class="app-preview">
        </div>
    </section>
    
    <section class="features">
        <h2>Why You''ll Love {{title}}</h2>
        <div class="features-list">
            {{#each features}}
            <div class="feature-item">
                <div class="feature-icon">{{emoji}}</div>
                <div class="feature-content">
                    <h3>{{title}}</h3>
                    <p>{{description}}</p>
                </div>
            </div>
            {{/each}}
        </div>
    </section>
    
    <section class="social-proof">
        <h2>Join Thousands of Happy Users</h2>
        <div class="stats">
            <div class="stat">
                <div class="stat-number">{{download_count}}</div>
                <div class="stat-label">Downloads</div>
            </div>
            <div class="stat">
                <div class="stat-number">{{rating}}</div>
                <div class="stat-label">App Store Rating</div>
            </div>
            <div class="stat">
                <div class="stat-number">{{user_count}}</div>
                <div class="stat-label">Active Users</div>
            </div>
        </div>
    </section>
    
    <section class="cta-section">
        <h2>Ready to Get Started?</h2>
        <p>Download {{title}} today and see the difference!</p>
        <div class="app-badges">
            <a href="{{ios_url}}" class="app-badge" onclick="trackClick(''ios_download'')">
                <img src="/images/app-store-badge.png" alt="Download on App Store">
            </a>
            <a href="{{android_url}}" class="app-badge" onclick="trackClick(''android_download'')">
                <img src="/images/google-play-badge.png" alt="Get it on Google Play">
            </a>
        </div>
    </section>
    
    <script src="script.js"></script>
</body>
</html>',
    
    '/* B2C App Landing Page Styles */
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: "SF Pro Display", -apple-system, BlinkMacSystemFont, sans-serif;
    line-height: 1.6;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: #333;
}

.hero {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 5rem 5%;
    max-width: 1200px;
    margin: 0 auto;
    color: white;
}

.hero-title {
    font-size: 3.5rem;
    font-weight: 800;
    margin-bottom: 1rem;
    background: linear-gradient(45deg, #fff, #f0f0f0);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
}

.hero-subtitle {
    font-size: 1.3rem;
    margin-bottom: 2rem;
    opacity: 0.9;
}

.app-badges {
    display: flex;
    gap: 1rem;
}

.app-badge img {
    height: 60px;
    transition: transform 0.3s;
}

.app-badge:hover img {
    transform: scale(1.05);
}

.app-preview {
    max-width: 400px;
    height: auto;
    border-radius: 20px;
    box-shadow: 0 20px 40px rgba(0,0,0,0.3);
}

.features {
    background: white;
    padding: 5rem 5%;
    text-align: center;
}

.features h2 {
    font-size: 2.5rem;
    margin-bottom: 3rem;
    color: #333;
}

.features-list {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 3rem;
    max-width: 1000px;
    margin: 0 auto;
}

.feature-item {
    display: flex;
    align-items: center;
    text-align: left;
    gap: 1rem;
}

.feature-icon {
    font-size: 3rem;
    flex-shrink: 0;
}

.social-proof {
    background: #f8f9fa;
    padding: 5rem 5%;
    text-align: center;
}

.social-proof h2 {
    font-size: 2.5rem;
    margin-bottom: 3rem;
}

.stats {
    display: flex;
    justify-content: center;
    gap: 4rem;
    max-width: 800px;
    margin: 0 auto;
}

.stat-number {
    font-size: 3rem;
    font-weight: bold;
    color: #667eea;
}

.stat-label {
    font-size: 1.1rem;
    color: #666;
}

.cta-section {
    background: linear-gradient(135deg, #764ba2 0%, #667eea 100%);
    padding: 5rem 5%;
    text-align: center;
    color: white;
}

.cta-section h2 {
    font-size: 2.5rem;
    margin-bottom: 1rem;
}

.cta-section p {
    font-size: 1.2rem;
    margin-bottom: 2rem;
    opacity: 0.9;
}

@media (max-width: 768px) {
    .hero {
        flex-direction: column;
        text-align: center;
        gap: 3rem;
    }
    
    .hero-title {
        font-size: 2.5rem;
    }
    
    .stats {
        flex-direction: column;
        gap: 2rem;
    }
    
    .app-badges {
        flex-direction: column;
        align-items: center;
    }
}',
    
    '// B2C App Landing Page JavaScript
function trackClick(action) {
    // Analytics tracking
    fetch("/api/v1/analytics/track", {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify({
            event: "app_download",
            platform: action.includes("ios") ? "ios" : "android",
            timestamp: new Date().toISOString()
        })
    }).catch(console.error);
}

// Smooth animations on scroll
const observerOptions = {
    threshold: 0.1,
    rootMargin: "0px 0px -50px 0px"
};

const observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
        if (entry.isIntersecting) {
            entry.target.style.opacity = "1";
            entry.target.style.transform = "translateY(0)";
        }
    });
}, observerOptions);

// Animate elements on load
document.addEventListener("DOMContentLoaded", function() {
    const animatedElements = document.querySelectorAll(".feature-item, .stat");
    animatedElements.forEach(el => {
        el.style.opacity = "0";
        el.style.transform = "translateY(20px)";
        el.style.transition = "all 0.6s ease-out";
        observer.observe(el);
    });
});',
    
    '{
        "title": {"type": "string", "required": true, "default": "Amazing App"},
        "tagline": {"type": "string", "required": true, "default": "Life Made Simple"},
        "description": {"type": "string", "required": true, "default": "The most intuitive app for your daily needs"},
        "hero_title": {"type": "string", "required": true, "default": "Life Made Simple"},
        "hero_subtitle": {"type": "string", "required": true, "default": "Everything you need, right at your fingertips"},
        "app_preview_url": {"type": "string", "default": "/images/app-preview.png"},
        "ios_url": {"type": "string", "default": "#"},
        "android_url": {"type": "string", "default": "#"},
        "download_count": {"type": "string", "default": "10K+"},
        "rating": {"type": "string", "default": "4.9★"},
        "user_count": {"type": "string", "default": "5K+"},
        "features": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "emoji": {"type": "string"},
                    "title": {"type": "string"},
                    "description": {"type": "string"}
                }
            }
        }
    }',
    '/preview/b2c-app-template',
    0,
    0.0
),

-- API Service Template
(
    'api-service-template',
    'Developer API Platform',
    'base',
    'api_service',
    'technology',
    '<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{title}} API | {{tagline}}</title>
    <meta name="description" content="{{description}}">
    <link rel="stylesheet" href="styles.css">
</head>
<body>
    <header class="header">
        <nav class="nav">
            <div class="logo">{{api_name}} API</div>
            <div class="nav-links">
                <a href="#docs">Docs</a>
                <a href="#pricing">Pricing</a>
                <a href="#playground">Playground</a>
                <a href="{{dashboard_url}}">Dashboard</a>
            </div>
        </nav>
    </header>
    
    <section class="hero">
        <div class="hero-content">
            <h1>{{hero_title}}</h1>
            <p class="hero-subtitle">{{hero_description}}</p>
            <div class="hero-actions">
                <button class="cta-primary" onclick="trackClick(''get_api_key'')">Get API Key</button>
                <a href="#docs" class="cta-secondary">View Documentation</a>
            </div>
            <div class="code-preview">
                <pre><code>curl -X POST {{base_url}}/v1/{{primary_endpoint}} \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d ''{{sample_request}}''</code></pre>
            </div>
        </div>
    </section>
    
    <section id="features" class="features">
        <h2>Why Developers Choose {{api_name}}</h2>
        <div class="features-grid">
            {{#each features}}
            <div class="feature-card">
                <div class="feature-icon">{{icon}}</div>
                <h3>{{title}}</h3>
                <p>{{description}}</p>
            </div>
            {{/each}}
        </div>
    </section>
    
    <section id="pricing" class="pricing">
        <h2>Simple API Pricing</h2>
        <div class="pricing-table">
            {{#each pricing_tiers}}
            <div class="pricing-tier {{#if popular}}popular{{/if}}">
                <h3>{{name}}</h3>
                <div class="price">${{price}}<span>/month</span></div>
                <div class="limits">
                    <div class="limit-item">{{requests_per_month}} requests/month</div>
                    <div class="limit-item">{{rate_limit}} requests/second</div>
                </div>
                <button class="pricing-cta" onclick="trackClick(''pricing_select'', ''{{name}}'')">
                    {{#if free}}Get Started Free{{else}}Choose Plan{{/if}}
                </button>
            </div>
            {{/each}}
        </div>
    </section>
    
    <section id="playground" class="playground">
        <h2>Try It Now</h2>
        <div class="api-playground">
            <div class="playground-controls">
                <select id="endpoint-select">
                    {{#each endpoints}}
                    <option value="{{path}}">{{method}} {{path}}</option>
                    {{/each}}
                </select>
                <button onclick="testAPI()" class="test-btn">Test API</button>
            </div>
            <div class="playground-result">
                <pre id="api-response">Click "Test API" to see results...</pre>
            </div>
        </div>
    </section>
    
    <script src="script.js"></script>
</body>
</html>',
    
    '/* API Service Landing Page Styles */
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: "Monaco", "Menlo", "Ubuntu Mono", monospace;
    line-height: 1.6;
    color: #e2e8f0;
    background: #0f172a;
}

.header {
    background: rgba(15, 23, 42, 0.95);
    backdrop-filter: blur(10px);
    position: sticky;
    top: 0;
    z-index: 100;
    border-bottom: 1px solid #1e293b;
}

.nav {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 5%;
    max-width: 1200px;
    margin: 0 auto;
}

.logo {
    font-size: 1.5rem;
    font-weight: bold;
    color: #10b981;
}

.nav-links a {
    text-decoration: none;
    color: #cbd5e1;
    margin-left: 2rem;
    transition: color 0.3s;
}

.nav-links a:hover {
    color: #10b981;
}

.hero {
    padding: 5rem 5%;
    max-width: 1200px;
    margin: 0 auto;
}

.hero h1 {
    font-size: 3rem;
    font-weight: bold;
    margin-bottom: 1rem;
    background: linear-gradient(45deg, #10b981, #3b82f6);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
}

.hero-subtitle {
    font-size: 1.2rem;
    color: #94a3b8;
    margin-bottom: 2rem;
}

.hero-actions {
    display: flex;
    gap: 1rem;
    margin-bottom: 3rem;
}

.cta-primary {
    background: #10b981;
    color: white;
    padding: 0.75rem 2rem;
    border: none;
    border-radius: 0.5rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s;
}

.cta-primary:hover {
    background: #059669;
    transform: translateY(-2px);
}

.cta-secondary {
    color: #10b981;
    text-decoration: none;
    padding: 0.75rem 2rem;
    border: 2px solid #10b981;
    border-radius: 0.5rem;
    font-weight: 600;
    transition: all 0.3s;
}

.cta-secondary:hover {
    background: #10b981;
    color: white;
}

.code-preview {
    background: #1e293b;
    padding: 1.5rem;
    border-radius: 0.5rem;
    border-left: 4px solid #10b981;
    overflow-x: auto;
}

.code-preview code {
    color: #e2e8f0;
    font-size: 0.9rem;
}

.features {
    padding: 5rem 5%;
    background: #1e293b;
}

.features h2 {
    text-align: center;
    font-size: 2.5rem;
    margin-bottom: 3rem;
    color: #f1f5f9;
}

.features-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 2rem;
    max-width: 1200px;
    margin: 0 auto;
}

.feature-card {
    background: #0f172a;
    padding: 2rem;
    border-radius: 0.5rem;
    border: 1px solid #334155;
    transition: transform 0.3s;
}

.feature-card:hover {
    transform: translateY(-5px);
    border-color: #10b981;
}

.pricing {
    padding: 5rem 5%;
}

.pricing h2 {
    text-align: center;
    font-size: 2.5rem;
    margin-bottom: 3rem;
    color: #f1f5f9;
}

.pricing-table {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 2rem;
    max-width: 1000px;
    margin: 0 auto;
}

.pricing-tier {
    background: #1e293b;
    border: 2px solid #334155;
    border-radius: 0.5rem;
    padding: 2rem;
    text-align: center;
    position: relative;
}

.pricing-tier.popular {
    border-color: #10b981;
}

.pricing-tier.popular::before {
    content: "Most Popular";
    background: #10b981;
    color: white;
    padding: 0.5rem 1rem;
    border-radius: 0.25rem;
    position: absolute;
    top: -0.75rem;
    left: 50%;
    transform: translateX(-50%);
    font-size: 0.8rem;
    font-weight: 600;
}

.playground {
    padding: 5rem 5%;
    background: #1e293b;
}

.playground h2 {
    text-align: center;
    font-size: 2.5rem;
    margin-bottom: 3rem;
    color: #f1f5f9;
}

.api-playground {
    max-width: 800px;
    margin: 0 auto;
    background: #0f172a;
    border-radius: 0.5rem;
    padding: 2rem;
}

.playground-controls {
    display: flex;
    gap: 1rem;
    margin-bottom: 2rem;
}

#endpoint-select {
    flex: 1;
    padding: 0.75rem;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 0.5rem;
    color: #e2e8f0;
}

.test-btn {
    background: #3b82f6;
    color: white;
    border: none;
    padding: 0.75rem 2rem;
    border-radius: 0.5rem;
    cursor: pointer;
    font-weight: 600;
}

.playground-result {
    background: #000;
    border-radius: 0.5rem;
    padding: 1.5rem;
    min-height: 200px;
}

#api-response {
    color: #10b981;
    font-size: 0.9rem;
}

@media (max-width: 768px) {
    .hero h1 {
        font-size: 2rem;
    }
    
    .hero-actions {
        flex-direction: column;
    }
    
    .playground-controls {
        flex-direction: column;
    }
}',
    
    '// API Service Landing Page JavaScript
function trackClick(action, plan = null) {
    const data = {
        event: action,
        timestamp: new Date().toISOString()
    };
    
    if (plan) {
        data.plan = plan;
    }
    
    fetch("/api/v1/analytics/track", {
        method: "POST",
        headers: {
            "Content-Type": "application/json"
        },
        body: JSON.stringify(data)
    }).catch(console.error);
}

function testAPI() {
    const endpoint = document.getElementById("endpoint-select").value;
    const responseElement = document.getElementById("api-response");
    
    responseElement.textContent = "Loading...";
    
    // Simulate API response
    setTimeout(() => {
        const sampleResponse = {
            "status": "success",
            "data": {
                "message": "API call successful",
                "endpoint": endpoint,
                "timestamp": new Date().toISOString()
            },
            "meta": {
                "version": "1.0",
                "rate_limit_remaining": 999
            }
        };
        
        responseElement.textContent = JSON.stringify(sampleResponse, null, 2);
        
        trackClick("api_playground_test");
    }, 1000);
}

// Syntax highlighting for code blocks
document.addEventListener("DOMContentLoaded", function() {
    const codeBlocks = document.querySelectorAll("pre code");
    codeBlocks.forEach(block => {
        // Simple syntax highlighting
        let html = block.innerHTML;
        html = html.replace(/(curl|POST|GET|PUT|DELETE)/g, '<span style="color: #3b82f6;">$1</span>');
        html = html.replace(/(".*?")/g, '<span style="color: #10b981;">$1</span>');
        html = html.replace(/(https?:\/\/[^\s]+)/g, '<span style="color: #f59e0b;">$1</span>');
        block.innerHTML = html;
    });
});',
    
    '{
        "title": {"type": "string", "required": true, "default": "Developer API"},
        "tagline": {"type": "string", "required": true, "default": "Powerful APIs for Developers"},
        "description": {"type": "string", "required": true, "default": "Simple, reliable APIs that scale with your business"},
        "api_name": {"type": "string", "required": true, "default": "YourAPI"},
        "hero_title": {"type": "string", "required": true, "default": "Build Amazing Apps with Our API"},
        "hero_description": {"type": "string", "required": true, "default": "Simple, powerful APIs designed for developers who demand reliability and performance"},
        "base_url": {"type": "string", "default": "https://api.yourservice.com"},
        "primary_endpoint": {"type": "string", "default": "data"},
        "sample_request": {"type": "string", "default": "{\"query\": \"hello world\"}"},
        "dashboard_url": {"type": "string", "default": "/dashboard"},
        "features": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "icon": {"type": "string"},
                    "title": {"type": "string"},
                    "description": {"type": "string"}
                }
            }
        },
        "pricing_tiers": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "name": {"type": "string"},
                    "price": {"type": "number"},
                    "requests_per_month": {"type": "string"},
                    "rate_limit": {"type": "string"},
                    "free": {"type": "boolean", "default": false},
                    "popular": {"type": "boolean", "default": false}
                }
            }
        },
        "endpoints": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "method": {"type": "string"},
                    "path": {"type": "string"}
                }
            }
        }
    }',
    '/preview/api-service-template',
    0,
    0.0
);

-- Update template usage count trigger
CREATE OR REPLACE FUNCTION increment_template_usage()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE templates 
    SET usage_count = usage_count + 1 
    WHERE id = NEW.template_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER increment_template_usage_trigger
    AFTER INSERT ON landing_pages
    FOR EACH ROW
    EXECUTE FUNCTION increment_template_usage();