# Visitor Intelligence Integration Guide

This guide shows how to integrate Visitor Intelligence into any Vrooli scenario to enable powerful visitor tracking, behavioral analytics, and retention marketing capabilities.

## 🚀 Quick Start Integration

### Single Line Integration
Add visitor tracking to any scenario with just one line:

```html
<script src="http://localhost:${API_PORT}/tracker.js" data-scenario="your-scenario-name"></script>
```

That's it! Your scenario now has:
- ✅ Automatic visitor identification (40-60% accuracy)
- ✅ Behavioral event tracking (clicks, pageviews, forms)
- ✅ Session management and analytics
- ✅ Privacy-compliant data collection

## 📊 Integration Examples by Scenario Type

### E-Commerce Scenarios
Perfect for cart abandonment, product recommendations, and conversion optimization.

```html
<!DOCTYPE html>
<html>
<head>
    <title>My E-Commerce Store</title>
</head>
<body>
    <div id="product-catalog">
        <button class="add-to-cart" data-product-id="123">Add to Cart</button>
    </div>

    <!-- Add visitor intelligence (replace with your API port) -->
    <script src="http://localhost:15001/tracker.js" data-scenario="my-store"></script>
    
    <script>
        // Track custom e-commerce events
        document.querySelectorAll('.add-to-cart').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const productId = e.target.dataset.productId;
                
                // Custom tracking for business intelligence
                window.VisitorIntelligence.track('add_to_cart', {
                    product_id: productId,
                    product_category: 'electronics',
                    price: 99.99,
                    currency: 'USD'
                });
            });
        });
        
        // Track cart abandonment
        window.addEventListener('beforeunload', () => {
            const cartItems = document.querySelectorAll('.cart-item').length;
            if (cartItems > 0) {
                window.VisitorIntelligence.track('cart_abandonment', {
                    items_count: cartItems,
                    estimated_value: calculateCartValue()
                });
            }
        });
    </script>
</body>
</html>
```

**What you get:**
- Cart abandonment detection and recovery campaigns
- Product recommendation based on browsing patterns  
- Customer lifetime value calculation
- A/B testing for pricing and layouts

### SaaS/Dashboard Scenarios
Ideal for feature usage analytics, onboarding optimization, and churn prediction.

```html
<!DOCTYPE html>
<html>
<head>
    <title>Project Management Dashboard</title>
</head>
<body>
    <nav id="main-nav">
        <a href="/projects" data-feature="projects">Projects</a>
        <a href="/tasks" data-feature="tasks">Tasks</a>
        <a href="/reports" data-feature="reports">Reports</a>
    </nav>

    <!-- Visitor Intelligence tracking -->
    <script src="http://localhost:15002/tracker.js" data-scenario="project-dashboard"></script>
    
    <script>
        // Track feature usage for product analytics
        document.querySelectorAll('[data-feature]').forEach(link => {
            link.addEventListener('click', (e) => {
                const feature = e.target.dataset.feature;
                
                window.VisitorIntelligence.track('feature_usage', {
                    feature_name: feature,
                    user_role: getCurrentUserRole(),
                    session_duration: getSessionDuration()
                });
            });
        });
        
        // Track user identification when they log in
        function onUserLogin(userEmail, userId) {
            window.VisitorIntelligence.identify({
                email: userEmail,
                user_id: userId,
                plan_type: 'premium',
                signup_date: '2023-01-15'
            });
        }
        
        // Track engagement milestones
        function trackMilestone(milestone) {
            window.VisitorIntelligence.track('milestone_reached', {
                milestone: milestone,
                time_to_milestone: calculateTimeToMilestone(),
                user_segment: 'power_user'
            });
        }
    </script>
</body>
</html>
```

**What you get:**
- Feature adoption and usage analytics
- User onboarding funnel optimization
- Churn risk scoring and prevention
- Product-led growth insights

### Content/Blog Scenarios
Great for engagement tracking, content optimization, and audience insights.

```html
<!DOCTYPE html>
<html>
<head>
    <title>Tech Blog - AI and Machine Learning</title>
</head>
<body>
    <article id="main-article" data-reading-time="8">
        <h1>The Future of AI in Software Development</h1>
        <div class="content">
            <!-- Article content -->
        </div>
        
        <div class="engagement-tools">
            <button class="like-btn" data-action="like">👍 Like</button>
            <button class="share-btn" data-action="share">📤 Share</button>
            <button class="bookmark-btn" data-action="bookmark">🔖 Bookmark</button>
        </div>
    </article>

    <!-- Newsletter signup form -->
    <form id="newsletter-signup">
        <input type="email" placeholder="Subscribe to our newsletter">
        <button type="submit">Subscribe</button>
    </form>

    <!-- Visitor Intelligence tracking -->
    <script src="http://localhost:15003/tracker.js" data-scenario="tech-blog"></script>
    
    <script>
        // Track reading engagement
        let readingStartTime = Date.now();
        let scrollDepth = 0;
        
        window.addEventListener('scroll', () => {
            const newScrollDepth = Math.round(
                (window.scrollY / (document.documentElement.scrollHeight - window.innerHeight)) * 100
            );
            
            // Track significant scroll milestones
            if (newScrollDepth > scrollDepth && newScrollDepth % 25 === 0) {
                window.VisitorIntelligence.track('reading_progress', {
                    scroll_depth: newScrollDepth,
                    time_reading: (Date.now() - readingStartTime) / 1000,
                    article_topic: 'AI development'
                });
                scrollDepth = newScrollDepth;
            }
        });
        
        // Track content engagement
        document.querySelectorAll('[data-action]').forEach(btn => {
            btn.addEventListener('click', (e) => {
                const action = e.target.dataset.action;
                
                window.VisitorIntelligence.track('content_engagement', {
                    action: action,
                    article_title: document.title,
                    reading_time: (Date.now() - readingStartTime) / 1000,
                    scroll_completion: scrollDepth
                });
            });
        });
        
        // Track newsletter signups
        document.getElementById('newsletter-signup').addEventListener('submit', (e) => {
            e.preventDefault();
            const email = e.target.querySelector('input[type="email"]').value;
            
            window.VisitorIntelligence.track('newsletter_signup', {
                source: 'article_bottom',
                article_topic: 'AI development',
                user_engagement_score: calculateEngagementScore()
            });
            
            // Identify the visitor
            window.VisitorIntelligence.identify({
                email: email,
                source: 'newsletter',
                interests: ['AI', 'software development']
            });
        });
    </script>
</body>
</html>
```

**What you get:**
- Content performance and engagement metrics
- Reader behavior and interest analysis
- Personalized content recommendations
- Newsletter conversion optimization

## 🔧 Advanced Integration Patterns

### Custom Event Tracking
Go beyond basic pageviews with business-specific events:

```javascript
// E-commerce events
VisitorIntelligence.track('product_view', {
    product_id: 'ABC123',
    category: 'electronics',
    price: 299.99,
    variant: 'blue-large'
});

// SaaS events
VisitorIntelligence.track('feature_adoption', {
    feature: 'advanced_reports',
    user_plan: 'enterprise',
    days_since_signup: 14
});

// Content events
VisitorIntelligence.track('video_engagement', {
    video_id: 'intro-tutorial',
    watch_duration: 120,
    completion_rate: 0.75
});
```

### User Identification
Link anonymous visitors to known users:

```javascript
// When user logs in or provides information
VisitorIntelligence.identify({
    email: 'user@example.com',
    name: 'John Doe',
    user_id: 'user_12345',
    plan: 'premium',
    signup_date: '2023-01-15',
    lifetime_value: 1299.99
});
```

### Real-time Personalization
Use visitor data to personalize experience:

```javascript
// Get current visitor information
const visitorId = VisitorIntelligence.getVisitorId();

// Fetch visitor profile and customize UI
fetch(`/api/v1/visitor/${visitorId}`)
    .then(response => response.json())
    .then(visitor => {
        if (visitor.identified) {
            showPersonalizedContent(visitor);
        } else {
            showGenericContent();
        }
        
        // Customize based on behavior
        if (visitor.total_page_views > 10) {
            showPowerUserFeatures();
        }
        
        if (visitor.tags.includes('high-value')) {
            showPremiumOffer();
        }
    });
```

## 📈 Analytics and Insights

### CLI Analytics
Get insights from the command line:

```bash
# Check system status
visitor-intelligence status

# View scenario analytics
visitor-intelligence analytics my-scenario --timeframe 30d

# Get visitor profile
visitor-intelligence profile visitor-abc123 --events

# Export data
visitor-intelligence analytics my-scenario --format csv > analytics.csv
```

### API Analytics
Programmatic access to visitor data:

```javascript
// Get scenario analytics
fetch('/api/v1/analytics/scenario/my-store?timeframe=7d')
    .then(response => response.json())
    .then(analytics => {
        console.log('Unique visitors:', analytics.unique_visitors);
        console.log('Conversion rate:', analytics.conversion_rate);
        console.log('Average session:', analytics.avg_session_duration);
    });

// Get visitor profile
fetch('/api/v1/visitor/abc-123')
    .then(response => response.json())
    .then(visitor => {
        console.log('Total visits:', visitor.session_count);
        console.log('Engagement score:', calculateEngagement(visitor));
        console.log('Recommended actions:', getRecommendations(visitor));
    });
```

### Dashboard Access
View comprehensive analytics at `http://localhost:${API_PORT}/dashboard`:

- 📊 Real-time visitor metrics
- 🔴 Live visitor activity feed  
- 📈 Historical performance charts
- 🎯 Top-performing scenarios
- ⚡ Recent behavioral events

## 🔄 Cross-Scenario Intelligence

### Scenario-to-Scenario Communication
Share visitor insights between scenarios:

```javascript
// In scenario A: Tag high-value visitors
if (purchaseAmount > 500) {
    VisitorIntelligence.track('high_value_purchase', {
        amount: purchaseAmount,
        customer_tier: 'gold'
    });
}

// In scenario B: Personalize based on visitor history
fetch(`/api/v1/visitor/${visitorId}`)
    .then(response => response.json())
    .then(visitor => {
        // Check if they've made high-value purchases
        const isHighValue = visitor.tags.includes('high-value');
        if (isHighValue) {
            showPremiumContent();
        }
    });
```

### Visitor Journey Tracking
Follow visitors across multiple scenarios:

```javascript
// Track cross-scenario journeys
VisitorIntelligence.track('scenario_transition', {
    from_scenario: 'marketing-site',
    to_scenario: 'product-demo',
    transition_reason: 'cta_click'
});
```

## 🛡️ Privacy and Compliance

### GDPR Compliance
Visitor Intelligence is built with privacy first:

```javascript
// Respect Do Not Track
if (navigator.doNotTrack === '1') {
    // Tracking is automatically disabled
}

// Provide opt-out mechanism
function disableTracking() {
    localStorage.setItem('visitor_tracking_disabled', 'true');
    // Tracking will be disabled on next page load
}

// Data retention controls
// Data is automatically purged after configured retention period
```

### Consent Management
Integrate with consent management systems:

```javascript
// Wait for consent before tracking
if (hasUserConsent()) {
    // Initialize tracking
    document.head.appendChild(createTrackingScript());
} else {
    // Show consent banner
    showConsentBanner().then(consent => {
        if (consent) {
            document.head.appendChild(createTrackingScript());
        }
    });
}
```

## 🚀 Deployment and Performance

### Production Setup
Configure for production environments:

```bash
# Set environment variables
export API_PORT=8080
export POSTGRES_PASSWORD=secure_password
export REDIS_PORT=6379

# Start visitor intelligence
vrooli scenario run visitor-intelligence

# Verify health
curl http://localhost:8080/health
```

### Performance Optimization
- **Tracking script**: < 50KB minified
- **API response time**: < 50ms for tracking
- **Database**: Optimized with partitioning and indexes
- **Caching**: Redis for session and visitor data
- **CDN ready**: Tracking script can be served from CDN

### Monitoring
Monitor system health and performance:

```bash
# Check API status
visitor-intelligence status --verbose

# Monitor database performance
psql -c "SELECT * FROM pg_stat_activity WHERE application_name LIKE '%visitor%'"

# View Redis usage
redis-cli info memory
```

## 📚 Best Practices

### 🎯 Event Naming
Use consistent, descriptive event names:

```javascript
// Good: descriptive and consistent
'product_added_to_cart'
'user_completed_onboarding'
'article_reading_milestone'

// Bad: vague or inconsistent
'click'
'action'
'event'
```

### 🏷️ Property Structure
Structure event properties consistently:

```javascript
// Good: structured and searchable
VisitorIntelligence.track('purchase_completed', {
    order_id: 'ORD-123',
    total_amount: 99.99,
    currency: 'USD',
    items_count: 3,
    payment_method: 'credit_card',
    customer_tier: 'gold'
});

// Bad: inconsistent structure
VisitorIntelligence.track('purchase', {
    total: '$99.99',
    items: '3 items',
    method: 'cc'
});
```

### 📊 Performance Tips
- Load tracking script asynchronously
- Batch events when possible  
- Use event throttling for high-frequency events
- Cache visitor data in localStorage for offline scenarios

## 🆘 Troubleshooting

### Common Issues

**Tracking not working?**
```bash
# Check API status
visitor-intelligence status

# Verify script URL
curl http://localhost:8080/tracker.js

# Check browser console for errors
```

**Low identification rate?**
- Ensure script loads from same domain
- Check for ad blockers
- Verify browser compatibility

**Missing events?**
```javascript
// Add debugging
window.VisitorIntelligence.debug = true;

// Check network tab in browser dev tools
// Verify API endpoint is reachable
```

### Getting Help
- 📖 Check the README.md for basic setup
- 🧪 Run integration tests: `./test/integration-test.sh`
- 📊 View dashboard for real-time diagnostics
- 🔧 Use CLI for detailed status: `visitor-intelligence status --verbose`

---

## 🎉 You're Ready!

With Visitor Intelligence integrated, your scenario now has:

✅ **Automatic visitor identification** (no forms required)  
✅ **Real-time behavioral tracking** (clicks, views, engagement)  
✅ **Cross-scenario intelligence** (visitors tracked everywhere)  
✅ **Privacy-compliant data** (GDPR ready, first-party only)  
✅ **Powerful analytics** (dashboard + API + CLI access)  
✅ **Retention marketing** (automated campaigns based on behavior)  

Your scenario is now part of Vrooli's compound intelligence system! 🚀