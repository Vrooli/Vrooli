#!/bin/bash
# Compliance Checklist Generator
# Generates jurisdiction-specific compliance checklists for legal document implementation

set -euo pipefail

# Function to generate GDPR compliance checklist
generate_gdpr_checklist() {
    local business_name="${1:-Your Business}"
    cat <<EOF
# GDPR Compliance Checklist for ${business_name}

## Required Actions After Document Generation

### Immediate Actions (Before Going Live)
- [ ] Review generated privacy policy with legal counsel
- [ ] Implement cookie consent mechanism on website
- [ ] Set up data processing records (Article 30)
- [ ] Designate Data Protection Officer (if required by Article 37)
- [ ] Establish procedures for data subject rights requests

### Technical Implementation
- [ ] Add cookie banner with explicit consent options
- [ ] Implement "Accept" and "Reject" buttons for cookies
- [ ] Provide granular cookie preferences (essential, analytics, marketing)
- [ ] Ensure data minimization in collection forms
- [ ] Implement secure data storage and encryption

### Documentation Requirements
- [ ] Document legal basis for each type of data processing
- [ ] Create data retention policy and schedules
- [ ] Establish data breach notification procedures
- [ ] Document third-party data processors and agreements
- [ ] Maintain records of consent

### User Rights Implementation
- [ ] Build "Access My Data" feature (Article 15)
- [ ] Implement "Delete My Data" functionality (Article 17 - Right to be Forgotten)
- [ ] Create "Export My Data" capability (Article 20 - Data Portability)
- [ ] Set up "Correct My Data" process (Article 16)
- [ ] Establish "Object to Processing" mechanism (Article 21)

### Ongoing Compliance
- [ ] Conduct Data Protection Impact Assessment (if high-risk processing)
- [ ] Review and update privacy policy annually
- [ ] Train staff on GDPR compliance
- [ ] Monitor third-party processor compliance
- [ ] Maintain audit trail of compliance activities

### Links to Privacy Policy
- [ ] Link to privacy policy in website footer
- [ ] Include link in signup/registration flows
- [ ] Reference in email communications
- [ ] Display during checkout/payment processes
- [ ] Include in mobile app settings

## Penalties for Non-Compliance
⚠️  GDPR fines can reach up to €20 million or 4% of annual global turnover, whichever is higher.

## Resources
- GDPR Official Text: https://gdpr-info.eu/
- ICO Guide: https://ico.org.uk/for-organisations/
- GDPR Checklist: https://gdpr.eu/checklist/
EOF
}

# Function to generate CCPA compliance checklist
generate_ccpa_checklist() {
    local business_name="${1:-Your Business}"
    cat <<EOF
# CCPA Compliance Checklist for ${business_name}

## Required Actions After Document Generation

### Immediate Actions (Before Going Live)
- [ ] Review generated privacy policy with legal counsel
- [ ] Add "Do Not Sell My Personal Information" link to website
- [ ] Implement consumer rights request mechanisms
- [ ] Update privacy policy to include CCPA-specific disclosures
- [ ] Determine if you're a "business" under CCPA definition

### Technical Implementation
- [ ] Add "Do Not Sell" link in website footer
- [ ] Create consumer request portal
- [ ] Implement request verification system (2-step process)
- [ ] Set up systems to respond within 45 days
- [ ] Build capability to provide data in portable format

### Required Disclosures
- [ ] Categories of personal information collected
- [ ] Sources of personal information
- [ ] Business or commercial purposes for collection
- [ ] Categories of third parties with whom data is shared
- [ ] Specific pieces of information collected

### Consumer Rights Implementation
- [ ] Right to Know - Build data disclosure process
- [ ] Right to Delete - Implement deletion mechanism
- [ ] Right to Opt-Out - Create opt-out of sale mechanism
- [ ] Right to Non-Discrimination - Ensure equal service regardless of exercise of rights

### Notice Requirements
- [ ] Provide notice at or before data collection
- [ ] Update privacy policy with CCPA-specific sections
- [ ] Post "Do Not Sell" link prominently
- [ ] Include toll-free number for requests (if online business)
- [ ] Provide notice of financial incentives (if applicable)

### Ongoing Compliance
- [ ] Respond to consumer requests within 45 days (90 day extension if needed)
- [ ] Train employees on CCPA requirements
- [ ] Review and update privacy policy annually
- [ ] Maintain records of consumer requests
- [ ] Monitor third-party data practices

## Who CCPA Applies To
CCPA applies if your business:
- Has annual gross revenues over \$25 million, OR
- Processes personal information of 100,000+ California consumers, OR
- Derives 50%+ of revenue from selling personal information

## Penalties for Non-Compliance
⚠️  CCPA penalties: \$2,500 per violation, \$7,500 per intentional violation

## Resources
- CCPA Official Text: https://oag.ca.gov/privacy/ccpa
- CCPA Regulations: https://www.oag.ca.gov/privacy/ccpa/regs
- California AG Guidance: https://oag.ca.gov/privacy/ccpa
EOF
}

# Function to generate UK-GDPR compliance checklist
generate_uk_gdpr_checklist() {
    local business_name="${1:-Your Business}"
    cat <<EOF
# UK GDPR Compliance Checklist for ${business_name}

## Required Actions After Document Generation

### Immediate Actions
- [ ] Review generated privacy policy with UK legal counsel
- [ ] Implement cookie consent (ICO guidance)
- [ ] Register with ICO if required (data controller)
- [ ] Set up data processing records
- [ ] Designate Data Protection Officer if required

### Technical Implementation (Same as EU GDPR)
- [ ] Cookie consent banner with Accept/Reject
- [ ] Granular cookie preferences
- [ ] Data minimization in forms
- [ ] Secure storage and encryption
- [ ] Access controls

### UK-Specific Requirements
- [ ] Register with ICO (£40-60 annual fee for most)
- [ ] Appoint UK representative if based outside UK
- [ ] Ensure international data transfers comply with UK adequacy decisions
- [ ] Follow ICO guidance on cookies and consent
- [ ] Implement Subject Access Request (SAR) process

### Individual Rights (Same core rights as GDPR)
- [ ] Right of access
- [ ] Right to rectification
- [ ] Right to erasure
- [ ] Right to restrict processing
- [ ] Right to data portability
- [ ] Right to object

### ICO Registration Exemptions
You may be exempt from ICO registration if you only process data for:
- Staff administration
- Advertising, marketing and public relations
- Accounts and records

### Ongoing Compliance
- [ ] Conduct DPIAs for high-risk processing
- [ ] Maintain records of processing activities
- [ ] Report data breaches to ICO within 72 hours
- [ ] Notify affected individuals of breaches when required
- [ ] Review privacy policy annually

## Penalties
⚠️  Up to £17.5 million or 4% of annual global turnover, whichever is higher

## Resources
- ICO Website: https://ico.org.uk/
- UK GDPR Guidance: https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/
- Cookie Guidance: https://ico.org.uk/for-organisations/direct-marketing-and-privacy-and-electronic-communications/guide-to-pecr/cookies-and-similar-technologies/
EOF
}

# Function to generate PIPEDA compliance checklist (Canada)
generate_pipeda_checklist() {
    local business_name="${1:-Your Business}"
    cat <<EOF
# PIPEDA Compliance Checklist for ${business_name}

## Required Actions After Document Generation

### Immediate Actions
- [ ] Review generated privacy policy with Canadian legal counsel
- [ ] Implement consent mechanisms (meaningful, informed consent)
- [ ] Designate Privacy Officer or Data Protection Officer
- [ ] Establish procedures for access requests
- [ ] Create data retention and disposal policies

### PIPEDA's 10 Fair Information Principles
- [ ] Accountability - Designate someone responsible for compliance
- [ ] Identifying Purposes - Explain why you collect data
- [ ] Consent - Obtain meaningful consent before collection
- [ ] Limiting Collection - Only collect what's necessary
- [ ] Limiting Use, Disclosure, and Retention - Use only for stated purposes
- [ ] Accuracy - Keep personal information accurate and up-to-date
- [ ] Safeguards - Protect personal information appropriately
- [ ] Openness - Be transparent about privacy practices
- [ ] Individual Access - Provide access to personal information
- [ ] Challenging Compliance - Allow individuals to challenge compliance

### Consent Requirements
- [ ] Make consent clear and understandable
- [ ] Provide opt-out mechanisms for marketing
- [ ] Obtain express consent for sensitive information
- [ ] Allow withdrawal of consent
- [ ] Document consent obtained

### Access Request Process
- [ ] Respond to access requests within 30 days
- [ ] Provide information in an accessible format
- [ ] Charge reasonable fees (if any)
- [ ] Correct inaccurate information when requested
- [ ] Document reasons if access is denied

### Data Breach Requirements (PIPEDA Breach Regulations)
- [ ] Report breaches to Privacy Commissioner if real risk of significant harm
- [ ] Notify affected individuals if real risk of significant harm
- [ ] Keep records of all breaches
- [ ] Report as soon as feasible

### Ongoing Compliance
- [ ] Designate Privacy Officer
- [ ] Train staff on privacy practices
- [ ] Review privacy policy annually
- [ ] Conduct privacy audits
- [ ] Update practices as needed

## Resources
- Office of the Privacy Commissioner: https://www.priv.gc.ca/
- PIPEDA Overview: https://www.priv.gc.ca/en/privacy-topics/privacy-laws-in-canada/the-personal-information-protection-and-electronic-documents-act-pipeda/
- Breach Guidance: https://www.priv.gc.ca/en/privacy-topics/business-privacy/safeguards-and-breaches/privacy-breaches/respond-to-a-privacy-breach-at-your-business/
EOF
}

# Function to generate general compliance checklist
generate_general_checklist() {
    local business_name="${1:-Your Business}"
    cat <<EOF
# General Compliance Checklist for ${business_name}

## After Generating Legal Documents

### Document Review
- [ ] Have legal counsel review all generated documents
- [ ] Customize documents for your specific business needs
- [ ] Ensure all placeholder information is replaced
- [ ] Verify contact information is correct

### Website Implementation
- [ ] Add privacy policy to website footer
- [ ] Link to terms of service during signup
- [ ] Display cookie policy if using cookies
- [ ] Make documents easily accessible (2 clicks or less)

### User Consent
- [ ] Implement consent mechanisms during data collection
- [ ] Provide opt-in for marketing communications
- [ ] Document consent obtained
- [ ] Allow users to withdraw consent

### Data Management
- [ ] Create data retention policies
- [ ] Implement secure data storage
- [ ] Establish data deletion procedures
- [ ] Set up backup systems

### Best Practices
- [ ] Be transparent about data practices
- [ ] Minimize data collection
- [ ] Use encryption for sensitive data
- [ ] Train staff on privacy practices
- [ ] Review policies annually

### Next Steps
1. Implement documents on your website
2. Update data collection forms
3. Train your team
4. Monitor compliance
5. Update as needed

⚠️  Remember: Generated documents are templates. Always consult with legal counsel before use.
EOF
}

# Main function
main() {
    local jurisdiction="${1:-general}"
    local business_name="${2:-Your Business}"

    case "$jurisdiction" in
        EU|eu|gdpr|GDPR)
            generate_gdpr_checklist "$business_name"
            ;;
        US|us|ccpa|CCPA)
            generate_ccpa_checklist "$business_name"
            ;;
        UK|uk)
            generate_uk_gdpr_checklist "$business_name"
            ;;
        CA|ca|canada|pipeda|PIPEDA)
            generate_pipeda_checklist "$business_name"
            ;;
        general|*)
            generate_general_checklist "$business_name"
            ;;
    esac
}

# Run main if executed directly
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main "$@"
fi
