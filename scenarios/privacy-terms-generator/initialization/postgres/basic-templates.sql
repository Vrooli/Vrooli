-- Basic Templates for Privacy & Terms Generator
-- Version: 1.0.0

SET search_path TO legal_generator;

-- Clear existing templates for clean seeding
DELETE FROM legal_templates WHERE source_url = 'basic-seed';

-- Insert US Privacy Policy Template
INSERT INTO legal_templates (template_type, jurisdiction, version, source_url, content, is_active, fetched_at) VALUES
('privacy', 'US', '1.0.0', 'basic-seed', 
E'# Privacy Policy for {{business_name}}

**Last Updated:** {{generated_date}}  
**Effective Date:** {{effective_date}}

## 1. Introduction

Welcome to {{business_name}}. We respect your privacy and are committed to protecting your personal data. This privacy policy will inform you about how we look after your personal data and tell you about your privacy rights under US law.

## 2. Information We Collect

We may collect and process the following data about you:
- Name and contact details (email address, phone number)
- Account credentials and profile information
- Payment and billing information
- Usage data and preferences
- IP address and device information
- Browser type and version
- Time zone settings and location data

## 3. How We Use Your Information

We use the information we collect to:
- Provide and maintain our services
- Process transactions and send related information
- Send administrative information and updates
- Respond to your comments and questions
- Monitor and analyze usage patterns
- Protect against fraudulent or illegal activity

## 4. Data Security

We implement appropriate technical and organizational measures to protect your personal data against unauthorized or unlawful processing and against accidental loss, destruction, or damage.

## 5. Contact Information

For privacy-related questions, please contact us at:
- Email: {{contact_email}}
- Website: {{website}}',
true, CURRENT_TIMESTAMP - INTERVAL '5 days');

-- Insert EU/GDPR Privacy Policy Template
INSERT INTO legal_templates (template_type, jurisdiction, version, source_url, content, is_active, fetched_at) VALUES
('privacy', 'EU', '1.0.0', 'basic-seed',
E'# Privacy Policy for {{business_name}}

**Last Updated:** {{generated_date}}  
**Effective Date:** {{effective_date}}

## 1. Introduction

{{business_name}} ("we", "our", or "us") is committed to protecting your personal data in compliance with the General Data Protection Regulation (GDPR) (EU) 2016/679.

## 2. Data Controller

{{business_name}} is the data controller responsible for your personal data. You can contact our Data Protection Officer at: {{contact_email}}

## 3. Personal Data We Collect

### Data You Provide:
- Identity Data: name, username, title
- Contact Data: email address, telephone numbers, postal address
- Financial Data: payment card details, bank account details
- Profile Data: preferences, feedback, survey responses

### Data We Collect Automatically:
- Technical Data: IP address, browser type, device information
- Usage Data: information about how you use our services
- Marketing Data: preferences for receiving marketing

## 4. Legal Basis for Processing (Article 6 GDPR)

We process your personal data based on:
- **Article 6(1)(a)**: Consent - you have given consent for specific purposes
- **Article 6(1)(b)**: Contract - processing is necessary for contract performance
- **Article 6(1)(c)**: Legal obligation - processing is required by law
- **Article 6(1)(f)**: Legitimate interests - necessary for our legitimate interests

## 5. Your Rights (Articles 15-22 GDPR)

You have the right to:
- **Access** (Article 15): Obtain confirmation and access to your data
- **Rectification** (Article 16): Correct inaccurate personal data
- **Erasure** (Article 17): Request deletion of your data
- **Restriction** (Article 18): Restrict processing of your data
- **Portability** (Article 20): Receive your data in a portable format
- **Object** (Article 21): Object to processing based on legitimate interests
- **Withdraw consent**: Where processing is based on consent',
true, CURRENT_TIMESTAMP - INTERVAL '10 days');

-- Insert US Terms of Service Template
INSERT INTO legal_templates (template_type, jurisdiction, version, source_url, content, is_active, fetched_at) VALUES
('terms', 'US', '1.0.0', 'basic-seed',
E'# Terms of Service for {{business_name}}

**Last Updated:** {{generated_date}}  
**Effective Date:** {{effective_date}}

## 1. Agreement to Terms

By accessing or using {{business_name}} services, you agree to be bound by these Terms of Service and all applicable laws and regulations.

## 2. Use of Service

### Eligibility
You must be at least 18 years old to use our services. By using our services, you represent and warrant that you meet this requirement.

### Account Responsibilities
- You are responsible for maintaining the confidentiality of your account
- You are responsible for all activities under your account
- You must notify us immediately of any unauthorized use

## 3. Prohibited Uses

You may not use our services to:
- Violate any laws or regulations
- Infringe on intellectual property rights
- Transmit malicious code or viruses
- Engage in fraudulent or deceptive practices
- Harass, abuse, or harm others

## 4. Disclaimer of Warranties

THE SERVICE IS PROVIDED "AS IS" WITHOUT WARRANTIES OF ANY KIND, EXPRESS OR IMPLIED.

## 5. Limitation of Liability

IN NO EVENT SHALL {{business_name}} BE LIABLE FOR ANY INDIRECT, INCIDENTAL, SPECIAL, CONSEQUENTIAL, OR PUNITIVE DAMAGES.

## 6. Contact Information

Questions about these Terms should be sent to:
- Email: {{contact_email}}',
true, CURRENT_TIMESTAMP - INTERVAL '7 days');

-- Insert EU Terms of Service Template
INSERT INTO legal_templates (template_type, jurisdiction, version, source_url, content, is_active, fetched_at) VALUES
('terms', 'EU', '1.0.0', 'basic-seed',
E'# Terms of Service for {{business_name}}

**Last Updated:** {{generated_date}}  
**Effective Date:** {{effective_date}}

## 1. Scope and Acceptance

These Terms of Service constitute a legally binding agreement between you and {{business_name}}.

## 2. Consumer Rights (EU Directive 2011/83/EU)

### Right of Withdrawal
Consumers have 14 days to withdraw from the contract without giving any reason.

### Information Requirements
We provide clear information about:
- Main characteristics of the service
- Total price including taxes
- Duration of the contract
- Withdrawal procedures

## 3. Data Protection

Personal data processing is governed by our Privacy Policy in accordance with GDPR (EU) 2016/679.

## 4. Liability

We are liable for damages caused intentionally or by gross negligence. For slight negligence, we are only liable for breach of essential contractual obligations.

## 5. Contact

{{business_name}}  
Email: {{contact_email}}',
true, CURRENT_TIMESTAMP - INTERVAL '12 days');

-- Insert UK, Canada, and Australia templates  
INSERT INTO legal_templates (template_type, jurisdiction, version, source_url, content, is_active, fetched_at) VALUES
('privacy', 'UK', '1.0.0', 'basic-seed',
E'# Privacy Notice for {{business_name}}

**Last Updated:** {{generated_date}}

This privacy notice explains how {{business_name}} uses your personal data in accordance with the UK General Data Protection Regulation (UK GDPR) and the Data Protection Act 2018.

## Data Controller
{{business_name}} is the data controller. Contact us at {{contact_email}}.

## Personal Data We Process
- Identity and contact data
- Financial and transaction data
- Technical and usage data

## Your Rights
Under UK GDPR, you have rights to:
- Access your data
- Rectify inaccuracies
- Erase your data
- Restrict processing
- Data portability
- Object to processing

## Contact the ICO
You may lodge a complaint with the Information Commissioner''s Office at ico.org.uk.',
true, CURRENT_TIMESTAMP - INTERVAL '15 days'),

('privacy', 'CA', '1.0.0', 'basic-seed',
E'# Privacy Policy for {{business_name}}

**Last Updated:** {{generated_date}}

{{business_name}} is committed to protecting your privacy in accordance with the Personal Information Protection and Electronic Documents Act (PIPEDA).

## Accountability
{{business_name}} is responsible for personal information under our control.

## Consent
We obtain your consent for collection, use, and disclosure of personal information.

## Safeguards
We protect personal information with appropriate security safeguards.

## Individual Access
Upon request, you can access your personal information and challenge its accuracy.

## Contact
Contact our Privacy Officer at {{contact_email}} with any concerns.',
true, CURRENT_TIMESTAMP - INTERVAL '20 days'),

('privacy', 'AU', '1.0.0', 'basic-seed',
E'# Privacy Policy for {{business_name}}

**Last Updated:** {{generated_date}}

This privacy policy sets out how {{business_name}} complies with the Australian Privacy Principles (APPs) under the Privacy Act 1988 (Cth).

## Collection of Personal Information
We collect personal information that is reasonably necessary for our business functions and activities.

## Use and Disclosure
We use and disclose personal information only for the purposes for which it was collected.

## Data Security
We protect personal information from misuse, interference, loss, unauthorized access, modification, or disclosure.

## Access and Correction
You may request access to and correction of your personal information.

## Complaints
Contact our Privacy Officer at {{contact_email}} to make a complaint.',
true, CURRENT_TIMESTAMP - INTERVAL '25 days');