# Swarm Templates

This directory contains template configurations for different types of swarms in the Vrooli ecosystem. Each template provides a proven starting point for creating swarms optimized for specific business models and organizational tiers.

## Template Categories

### Director Templates (`director/`)
Meta-coordination swarms that manage the overall fleet, resource allocation, and business optimization across all other swarms.

### Infrastructure Templates (`infrastructure/`)
Platform optimization swarms focused on improving Vrooli itself, user experience, research capabilities, and marketing automation.

### Profit Templates (`profit/`)
Revenue-generating swarms designed to deliver specific services and generate target profit levels through various business models.

### Appliance Templates (`appliance/`)
Edge deployment swarms optimized for customer-premises installations with specific industry vertical configurations.

## Template Usage

### 1. Select Base Template
Choose the appropriate template based on:
- **Business Model**: inference-farm, SaaS, or appliance
- **Organizational Tier**: director, infrastructure, profit, or appliance
- **Industry Vertical**: dental, landscaping, consulting, etc. (if applicable)

### 2. Customize Configuration
Modify template parameters for your specific use case:
- Economic targets (profit goals, cost limits)
- Resource allocation (GPU, RAM, CPU quotas)
- Isolation requirements (security boundaries)
- KPI thresholds (success metrics)
- Vertical-specific workflows

### 3. Validate and Deploy
Use validation tools to ensure:
- Economic viability of the configuration
- Proper resource allocation and isolation
- Compliance with business model requirements
- Integration with existing swarm ecosystem

## Template Structure

Each template follows this standard structure:

```json
{
  "__version": "1.0",
  "identity": {
    "name": "template-name",
    "displayName": "Human Readable Name",
    "version": "1.0.0",
    "description": "Template description and use case"
  },
  "businessModel": {
    "type": "inference-farm|saas|appliance",
    "tier": "director|infrastructure|profit|appliance"
  },
  "economics": {
    "targetProfitPerMonth": 0,
    "maxMonthlyCost": 0,
    "revenueModel": "optimization-yield|saas-subscription|etc"
  },
  "isolation": {
    "sandboxLevel": "none|user-namespace|full-container",
    "resourceQuotas": {
      "gpuQuota": 0,
      "ramQuota": 0,
      "cpuQuota": 0
    }
  },
  "kpis": {
    "primary": {
      "profitPerAgentHour": 0,
      "successRate": 0
    }
  },
  "deployment": {
    "hardwareProfile": "swarminator|appliance|cloud",
    "customerTier": "enterprise|standard|self-service"
  },
  "baseConfig": {
    "goal": "Specific swarm objective",
    "limits": { /* Resource limits */ },
    "scheduling": { /* Tool scheduling */ },
    "policy": { /* Access policy */ }
  }
}
```

## Economic Modeling

Templates include economic modeling to ensure viability:

### Profit Calculations
- **Revenue Projections**: Based on historical data and market analysis
- **Cost Structure**: Computational, infrastructure, and operational costs
- **Break-Even Analysis**: Time to profitability and ROI calculations
- **Risk Assessment**: Sensitivity analysis for key variables

### Resource Optimization
- **Hardware Utilization**: Optimal GPU/CPU/RAM allocation
- **Scaling Parameters**: Auto-scaling triggers and limits
- **Cost Controls**: Automated shutdown and optimization triggers
- **Performance Targets**: Efficiency and utilization goals

## Customization Guidelines

### Business Model Adaptations

#### Inference Farm
- Focus on computational efficiency and resource utilization
- Higher profit targets to justify hardware investment
- Full isolation for client work separation
- Aggressive cost optimization and monitoring

#### SaaS
- Emphasis on user experience and feature development
- Lower individual swarm costs with volume scaling
- Shared resource pools for efficiency
- Customer satisfaction and retention metrics

#### Appliance
- Edge processing optimization and minimal resource usage
- Self-service configuration and monitoring
- Industry-specific compliance and workflows
- Customer success and support cost minimization

### Vertical Customizations

#### Healthcare/Dental
- HIPAA compliance requirements
- Enhanced data encryption and audit logging
- Patient-focused workflows and satisfaction metrics
- Integration with medical practice management systems

#### Professional Services
- Project-based workflows and deliverable tracking
- Client communication and satisfaction focus
- Time tracking and billing optimization
- Knowledge capture and reuse systems

#### Retail/E-commerce
- Inventory and order management automation
- Customer service and support optimization
- Pricing and competition monitoring
- Sales and conversion optimization

## Template Testing

### Validation Checklist
- [ ] Economic model validation (profit targets achievable)
- [ ] Resource allocation feasibility (hardware capacity)
- [ ] Isolation and security compliance (regulatory requirements)
- [ ] KPI benchmarking (realistic and measurable targets)
- [ ] Integration compatibility (existing swarm ecosystem)

### Performance Testing
- [ ] Load testing under expected workloads
- [ ] Resource utilization monitoring
- [ ] Cost tracking and optimization validation
- [ ] Customer satisfaction simulation
- [ ] Failure mode and recovery testing

## Template Evolution

Templates are living documents that evolve based on:
- **Performance Data**: Real-world deployment results and optimization
- **Market Changes**: Shifts in pricing, competition, and demand
- **Technology Updates**: New capabilities and efficiency improvements
- **Customer Feedback**: User experience and satisfaction improvements
- **Regulatory Changes**: Compliance and security requirement updates

### Version Management
- **Semantic Versioning**: Major.Minor.Patch version scheme
- **Backward Compatibility**: Migration paths for existing deployments
- **Testing Requirements**: Validation before template updates
- **Documentation**: Change logs and upgrade guidance

## Contributing

When creating or modifying templates:

1. **Start with Proven Patterns**: Base new templates on successful deployments
2. **Validate Economics**: Ensure profit targets and cost models are realistic
3. **Test Thoroughly**: Validate all aspects before making templates available
4. **Document Clearly**: Provide clear usage guidance and customization options
5. **Monitor Performance**: Track template success rates and optimization opportunities

## Related Documentation

- [`../schemas/`](../schemas/) - Configuration schemas and validation rules
- [`../README.md`](../README.md) - Swarm creation system overview
- [`/docs/architecture/execution/`](../../architecture/execution/) - Swarm execution architecture
- [`/docs/plans/swarm-orchestration.md`](../../plans/swarm-orchestration.md) - Swarm coordination strategies