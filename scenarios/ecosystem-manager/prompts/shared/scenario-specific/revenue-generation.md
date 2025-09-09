# Revenue Generation

## Purpose
Many scenarios are meant to be products or services that generate revenue. If this is one of those cases, consider ways to make it a good product/service that people will want to use and pay for.

## Building with Microservices

The most effective way to build feature-rich applications is to use other scenarios as microservices. This approach:
- **Speeds up development** by reusing proven components
- **Makes code cleaner** by focusing on core functionality  
- **Enables rapid prototyping** with established building blocks

### Core Microservice Scenarios
**scenario-authenticator** - Add multi-tenancy and user management  
**comment-system** - Add commenting functionality to any interface  
**recommendation-engine** - Add personalized recommendations  
**notification-system** - Handle email, SMS, and push notifications  
**payment-processor** - Accept payments and manage billing  
**analytics-dashboard** - Track usage and generate insights  

Using these as microservices means you can focus on your unique value proposition while leveraging battle-tested components for common needs.

## Deployment and Enhancement Helpers

Some scenarios specialize in modifying existing scenarios with new features or deployment targets:

**scenario-to-android** - Convert web scenarios to Android apps  
**saas-landing-manager** - Add marketing landing pages  
**api-documentation** - Generate interactive API docs  
**monitoring-integration** - Add observability and alerts  

## Planning Approach

When creating a revenue-generating scenario:

1. **Focus on core value** - What unique problem does this solve?
2. **Identify microservice needs** - Which common features can be delegated?
3. **Plan for helpers** - What deployment/enhancement needs come later?

In your PRD.md, consider specifying:
```markdown
## Future Enhancements
- Mobile app will be handled later by scenario-to-android
- Marketing site will be handled later by saas-landing-manager  
- Advanced analytics will be handled later by analytics-dashboard
```

This lets you stay focused on your core functionality while acknowledging future needs will be addressed by specialized helper scenarios.