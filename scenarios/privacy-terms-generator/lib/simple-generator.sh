#!/bin/bash
# Simple Template-Based Generator
# Falls back to basic templates when database is not available

set -euo pipefail

# Generate privacy policy
generate_privacy_policy() {
    local business_name="$1"
    local jurisdiction="$2"
    local email="${3:-privacy@example.com}"
    local website="${4:-https://example.com}"
    
    cat <<EOF
# Privacy Policy

**Last Updated:** $(date '+%B %d, %Y')  
**Effective Date:** $(date '+%B %d, %Y')

## 1. Introduction

Welcome to ${business_name}. We respect your privacy and are committed to protecting your personal data. This privacy policy will inform you about how we look after your personal data and tell you about your privacy rights.

## 2. Information We Collect

We may collect and process the following data about you:

### Personal Information
- Name and contact details (email address, phone number)
- Account credentials and profile information
- Payment and billing information
- Usage data and preferences

### Technical Information
- IP address and device information
- Browser type and version
- Time zone settings and location data
- Operating system and platform

## 3. How We Use Your Information

We use the information we collect to:
- Provide and maintain our services
- Process transactions and send related information
- Send administrative information and updates
- Respond to your comments and questions
- Monitor and analyze usage patterns
- Protect against fraudulent or illegal activity

## 4. Legal Basis for Processing (${jurisdiction} Specific)

$(case "$jurisdiction" in
    EU|UK)
        echo "Under GDPR, we process your data based on:
- **Consent**: Where you have given clear consent
- **Contract**: Processing necessary for a contract
- **Legal Obligation**: To comply with the law
- **Legitimate Interests**: Our business interests that don't override your rights"
        ;;
    US)
        echo "We process your data in accordance with applicable US privacy laws, including CCPA where applicable. You have the right to know what personal information we collect and how it's used."
        ;;
    CA)
        echo "We comply with Canadian privacy laws including PIPEDA. We collect, use, and disclose personal information only for purposes that a reasonable person would consider appropriate."
        ;;
    AU)
        echo "We comply with the Australian Privacy Principles (APPs) under the Privacy Act 1988. We handle personal information in an open and transparent way."
        ;;
    *)
        echo "We process your data in accordance with applicable privacy laws in your jurisdiction."
        ;;
esac)

## 5. Data Retention

We retain your personal data only for as long as necessary to fulfill the purposes for which it was collected, including satisfying any legal, accounting, or reporting requirements.

## 6. Your Rights

Depending on your location, you may have the following rights:
- Access your personal data
- Correct inaccurate data
- Request deletion of your data
- Object to processing of your data
- Request restriction of processing
- Data portability
- Withdraw consent

## 7. Data Security

We implement appropriate technical and organizational measures to protect your personal data against unauthorized or unlawful processing, accidental loss, destruction, or damage.

## 8. Third-Party Services

We may share your information with trusted third-party service providers who assist us in operating our website, conducting our business, or servicing you.

## 9. International Transfers

$(case "$jurisdiction" in
    EU|UK)
        echo "If we transfer your data outside the EEA/UK, we ensure appropriate safeguards are in place, such as Standard Contractual Clauses or adequacy decisions."
        ;;
    *)
        echo "Your information may be transferred to and maintained on servers located outside of your state, province, or country where privacy laws may differ."
        ;;
esac)

## 10. Children's Privacy

Our services are not intended for individuals under the age of 16. We do not knowingly collect personal data from children under 16.

## 11. Changes to This Policy

We may update this privacy policy from time to time. We will notify you of any changes by posting the new policy on this page and updating the "Last Updated" date.

## 12. Contact Us

If you have any questions about this privacy policy, please contact us:
- Email: ${email}
- Website: ${website}

---

© $(date '+%Y') ${business_name}. All rights reserved.
EOF
}

# Generate terms of service
generate_terms() {
    local business_name="$1"
    local jurisdiction="$2"
    local email="${3:-legal@example.com}"
    local website="${4:-https://example.com}"
    
    cat <<EOF
# Terms of Service

**Last Updated:** $(date '+%B %d, %Y')  
**Effective Date:** $(date '+%B %d, %Y')

## 1. Agreement to Terms

By accessing or using the services provided by ${business_name}, you agree to be bound by these Terms of Service.

## 2. Use of Service

You may use our service for lawful purposes only. You agree not to use the service:
- In any way that violates any applicable law or regulation
- To transmit any harmful or malicious code
- To attempt to gain unauthorized access to any part of the service
- In any way that could damage, disable, or impair the service

## 3. Intellectual Property

The service and its original content, features, and functionality are owned by ${business_name} and are protected by international copyright, trademark, patent, trade secret, and other intellectual property laws.

## 4. User Content

You retain ownership of content you submit. By submitting content, you grant us a worldwide, non-exclusive, royalty-free license to use, reproduce, adapt, publish, and distribute your content.

## 5. Privacy

Your use of our service is also governed by our Privacy Policy. Please review our Privacy Policy, which also governs the site and informs users of our data collection practices.

## 6. Termination

We may terminate or suspend your account and access to the service immediately, without prior notice or liability, for any reason, including breach of these Terms.

## 7. Disclaimers

The service is provided on an "AS IS" and "AS AVAILABLE" basis. ${business_name} makes no warranties, expressed or implied, and hereby disclaims all implied warranties.

## 8. Limitation of Liability

In no event shall ${business_name}, its directors, employees, partners, agents, suppliers, or affiliates, be liable for any indirect, incidental, special, consequential, or punitive damages.

## 9. Governing Law

These Terms shall be governed by the laws of $(case "$jurisdiction" in
    US) echo "the United States" ;;
    EU) echo "the European Union" ;;
    UK) echo "the United Kingdom" ;;
    CA) echo "Canada" ;;
    AU) echo "Australia" ;;
    *) echo "the applicable jurisdiction" ;;
esac), without regard to its conflict of law provisions.

## 10. Changes to Terms

We reserve the right to modify these terms at any time. If we make material changes, we will notify you by updating the date at the top of these terms.

## 11. Contact Information

For questions about these Terms, please contact us at:
- Email: ${email}
- Website: ${website}

---

© $(date '+%Y') ${business_name}. All rights reserved.
EOF
}

# Main function
case "${1:-}" in
    privacy)
        generate_privacy_policy "$2" "$3" "${4:-}" "${5:-}"
        ;;
    terms)
        generate_terms "$2" "$3" "${4:-}" "${5:-}"
        ;;
    *)
        echo "Usage: $0 {privacy|terms} <business_name> <jurisdiction> [email] [website]"
        exit 1
        ;;
esac