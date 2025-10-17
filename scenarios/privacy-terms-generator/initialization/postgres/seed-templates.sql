-- Seed Templates for Privacy & Terms Generator
-- Version: 1.0.0
-- Description: Initial legal templates and compliance requirements

SET search_path TO legal_generator;

-- Insert compliance requirements
INSERT INTO compliance_requirements (jurisdiction, requirement_type, description, mandatory_clauses, effective_date, source_url) VALUES
('EU', 'GDPR', 'General Data Protection Regulation', 
 ARRAY['data_controller_identity', 'purpose_of_processing', 'legal_basis', 'data_subject_rights', 'retention_period', 'data_transfers', 'dpo_contact'],
 '2018-05-25', 'https://gdpr.eu/'),
('US', 'CCPA', 'California Consumer Privacy Act',
 ARRAY['categories_collected', 'purposes_of_use', 'right_to_delete', 'right_to_know', 'right_to_opt_out', 'non_discrimination'],
 '2020-01-01', 'https://oag.ca.gov/privacy/ccpa'),
('US', 'COPPA', 'Childrens Online Privacy Protection Act',
 ARRAY['parental_consent', 'data_collection_disclosure', 'parental_review_rights', 'data_deletion', 'security_measures'],
 '2000-04-21', 'https://www.ftc.gov/legal-library/browse/rules/childrens-online-privacy-protection-rule-coppa'),
('UK', 'UK-GDPR', 'UK General Data Protection Regulation',
 ARRAY['data_controller_identity', 'purpose_of_processing', 'legal_basis', 'data_subject_rights', 'retention_period', 'ico_contact'],
 '2021-01-01', 'https://ico.org.uk/'),
('CA', 'PIPEDA', 'Personal Information Protection and Electronic Documents Act',
 ARRAY['purposes_of_collection', 'consent', 'limiting_collection', 'safeguards', 'openness', 'individual_access'],
 '2000-04-13', 'https://www.priv.gc.ca/en/privacy-topics/privacy-laws-in-canada/the-personal-information-protection-and-electronic-documents-act-pipeda/')
ON CONFLICT (jurisdiction, requirement_type) DO NOTHING;

-- Insert basic privacy policy template (GDPR compliant)
INSERT INTO legal_templates (template_type, jurisdiction, industry, version, content, sections, requirements) VALUES
('privacy', 'EU', NULL, '1.0.0', 
'# Privacy Policy for {{business_name}}

**Last Updated:** {{generated_date}}

## 1. Introduction

{{business_name}} ("we", "our", or "us") is committed to protecting your privacy. This Privacy Policy explains how we collect, use, disclose, and safeguard your information when you use our {{service_description}}.

## 2. Data Controller Information

**Company:** {{business_name}}
**Address:** {{business_address}}
**Email:** {{contact_email}}
**Data Protection Officer:** {{dpo_contact}}

## 3. Information We Collect

We collect the following categories of personal data:
{{data_categories}}

## 4. Legal Basis for Processing

We process your personal data based on the following legal bases:
- **Consent:** Where you have given clear consent for us to process your personal data
- **Contract:** Where processing is necessary for the performance of a contract
- **Legal Obligation:** Where processing is necessary for compliance with a legal obligation
- **Legitimate Interests:** Where processing is necessary for our legitimate interests

## 5. How We Use Your Information

We use your information for the following purposes:
{{use_purposes}}

## 6. Data Sharing and Disclosure

We may share your information with:
{{data_recipients}}

## 7. International Data Transfers

{{international_transfers}}

## 8. Data Retention

We retain your personal data for as long as necessary to fulfill the purposes outlined in this Privacy Policy, unless a longer retention period is required by law.

## 9. Your Rights

Under GDPR, you have the following rights:
- Right to access your personal data
- Right to rectification
- Right to erasure ("right to be forgotten")
- Right to restrict processing
- Right to data portability
- Right to object
- Rights related to automated decision-making and profiling

To exercise these rights, please contact us at {{contact_email}}.

## 10. Data Security

We implement appropriate technical and organizational measures to protect your personal data against unauthorized access, alteration, disclosure, or destruction.

## 11. Children\'s Privacy

{{children_policy}}

## 12. Updates to This Policy

We may update this Privacy Policy from time to time. The updated version will be indicated by an updated "Last Updated" date.

## 13. Contact Us

If you have questions about this Privacy Policy, please contact us at:
**Email:** {{contact_email}}
**Data Protection Officer:** {{dpo_contact}}

## 14. Supervisory Authority

You have the right to lodge a complaint with a supervisory authority if you believe our processing of your personal data violates applicable law.',
JSONB_BUILD_OBJECT(
    'introduction', 'Company introduction and commitment to privacy',
    'controller_info', 'Data controller contact details',
    'data_collection', 'Types of data collected',
    'legal_basis', 'Legal grounds for processing',
    'data_usage', 'How collected data is used',
    'data_sharing', 'Third parties data is shared with',
    'international_transfers', 'Cross-border data transfers',
    'retention', 'Data retention periods',
    'user_rights', 'GDPR data subject rights',
    'security', 'Security measures',
    'children', 'Children privacy protection',
    'updates', 'Policy update procedures',
    'contact', 'Contact information',
    'supervisory', 'Supervisory authority information'
),
JSONB_BUILD_OBJECT('GDPR', true, 'compliant', true))
ON CONFLICT (template_type, jurisdiction, industry, version) DO NOTHING;

-- Insert basic terms of service template
INSERT INTO legal_templates (template_type, jurisdiction, industry, version, content, sections, requirements) VALUES
('terms', 'US', 'saas', '1.0.0',
'# Terms of Service

**Last Updated:** {{generated_date}}
**Effective Date:** {{effective_date}}

## 1. Agreement to Terms

By accessing or using {{service_name}} ("Service") operated by {{business_name}} ("Company", "we", "us", or "our"), you agree to be bound by these Terms of Service ("Terms").

## 2. Description of Service

{{service_description}}

## 3. Account Registration

To access certain features of the Service, you must register for an account. You agree to:
- Provide accurate and complete information
- Maintain the security of your password
- Accept responsibility for all activities under your account
- Notify us immediately of any unauthorized use

## 4. Subscription and Payment

### 4.1 Billing
{{billing_terms}}

### 4.2 Free Trial
{{free_trial_terms}}

### 4.3 Refunds
{{refund_policy}}

## 5. Acceptable Use

You agree NOT to:
- Violate any laws or regulations
- Infringe on intellectual property rights
- Transmit malware or harmful code
- Attempt to gain unauthorized access
- Harass, abuse, or harm others
- Use the Service for illegal purposes

## 6. Intellectual Property

### 6.1 Our Property
The Service and its original content remain the exclusive property of {{business_name}} and its licensors.

### 6.2 Your Content
You retain ownership of content you submit, but grant us a license to use, modify, and display it in connection with the Service.

## 7. Privacy

Your use of the Service is subject to our Privacy Policy, available at {{privacy_policy_url}}.

## 8. Disclaimers and Warranties

THE SERVICE IS PROVIDED "AS IS" WITHOUT WARRANTIES OF ANY KIND. WE DISCLAIM ALL WARRANTIES, EXPRESS OR IMPLIED, INCLUDING MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE, AND NON-INFRINGEMENT.

## 9. Limitation of Liability

TO THE MAXIMUM EXTENT PERMITTED BY LAW, {{business_name}} SHALL NOT BE LIABLE FOR ANY INDIRECT, INCIDENTAL, SPECIAL, CONSEQUENTIAL, OR PUNITIVE DAMAGES ARISING FROM YOUR USE OF THE SERVICE.

## 10. Indemnification

You agree to indemnify and hold harmless {{business_name}} from any claims, damages, or expenses arising from your use of the Service or violation of these Terms.

## 11. Termination

We may terminate or suspend your account immediately, without prior notice, for any reason, including breach of these Terms.

## 12. Governing Law

These Terms shall be governed by the laws of {{governing_law_jurisdiction}}, without regard to conflict of law provisions.

## 13. Dispute Resolution

{{dispute_resolution_terms}}

## 14. Changes to Terms

We reserve the right to modify these Terms at any time. We will notify you of changes by posting the new Terms on this page.

## 15. Contact Information

For questions about these Terms, contact us at:
**Email:** {{contact_email}}
**Address:** {{business_address}}',
JSONB_BUILD_OBJECT(
    'agreement', 'Agreement to terms',
    'service_description', 'What the service provides',
    'registration', 'Account creation requirements',
    'payment', 'Subscription and billing terms',
    'acceptable_use', 'Usage restrictions',
    'intellectual_property', 'IP ownership',
    'privacy', 'Privacy policy reference',
    'disclaimers', 'Warranty disclaimers',
    'liability', 'Liability limitations',
    'indemnification', 'User indemnification',
    'termination', 'Account termination',
    'governing_law', 'Applicable law',
    'dispute_resolution', 'How disputes are handled',
    'changes', 'Terms modification',
    'contact', 'Contact information'
),
JSONB_BUILD_OBJECT('saas', true, 'standard', true))
ON CONFLICT (template_type, jurisdiction, industry, version) DO NOTHING;

-- Insert reusable template clauses
INSERT INTO template_clauses (clause_type, jurisdiction, content, tags) VALUES
('cookies_essential', 'EU', 'We use essential cookies that are necessary for the operation of our website. These cookies do not require your consent as they are essential for providing you with services available through our website.', ARRAY['cookies', 'gdpr', 'essential']),
('cookies_analytics', 'EU', 'We use analytics cookies to understand how visitors interact with our website. These cookies help us improve our website\'s functionality and your user experience.', ARRAY['cookies', 'gdpr', 'analytics']),
('data_breach_notification', 'EU', 'In the event of a personal data breach, we will notify the relevant supervisory authority within 72 hours of becoming aware of the breach. If the breach is likely to result in a high risk to your rights and freedoms, we will notify you without undue delay.', ARRAY['gdpr', 'data_breach', 'notification']),
('california_rights', 'US', 'If you are a California resident, you have the right to request disclosure of the personal information we collect, use, and disclose. You may also request deletion of your personal information, subject to certain exceptions.', ARRAY['ccpa', 'california', 'rights']),
('children_under_13', 'US', 'Our Service is not directed to children under 13. We do not knowingly collect personal information from children under 13. If we become aware that we have collected personal information from a child under 13, we will take steps to delete such information.', ARRAY['coppa', 'children', 'privacy']),
('liability_cap', 'US', 'Our total liability to you for any damages arising out of or related to these Terms or the Service shall not exceed the amount you paid us in the twelve (12) months preceding the claim.', ARRAY['liability', 'terms', 'limitation']),
('arbitration', 'US', 'Any dispute arising from these Terms shall be resolved through binding arbitration in accordance with the rules of the American Arbitration Association. The arbitration shall be conducted in {{arbitration_location}}.', ARRAY['dispute', 'arbitration', 'resolution'])
ON CONFLICT DO NOTHING;

-- Update template freshness tracking
INSERT INTO template_freshness (template_id, checked_at, is_current, update_priority)
SELECT id, CURRENT_TIMESTAMP, true, 'low'
FROM legal_templates
WHERE is_active = true;