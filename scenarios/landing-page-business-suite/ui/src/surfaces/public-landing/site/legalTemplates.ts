/**
 * Default legal documents. Operators replace either document from
 * Admin → Branding → Legal pages; an unset document falls back to these.
 *
 * Templates use {{token}} placeholders so edited documents keep tracking the
 * business identity configured in Branding. They describe what this suite
 * actually does (magic-link sign-in, Stripe checkout, first-party analytics,
 * support messages, app downloads) and are a starting point, not legal advice.
 */

export type LegalDocumentKind = 'privacy' | 'terms';

export interface LegalIdentity {
  businessName: string;
  siteName: string;
  contactEmail?: string;
  contactAddress?: string;
  website?: string;
  effectiveDate?: string;
}

export const LEGAL_TOKENS = ['business_name', 'site_name', 'contact_email', 'contact_address', 'website', 'effective_date'] as const;

export const DEFAULT_PRIVACY_MARKDOWN = `This Privacy Policy explains how {{business_name}} ("we", "us", or "our") collects, uses, and protects information when you visit {{website}}, create an account, buy a subscription, download our apps, or contact us.

## Information we collect

**Information you give us**

- **Account details.** Your email address when you sign in. We use passwordless sign-in links, so we never collect or store a password.
- **Messages.** The email address, subject, message, and any order reference you include when you contact us or send feedback.
- **Waitlist sign-ups.** Your email address if you ask to be told when something launches.

**Information collected automatically**

- **Usage information.** Pages viewed, buttons selected, the page that referred you, and campaign parameters in the link you followed (such as \`utm_source\`). We use this to understand which pages are helpful.
- **Device identifiers.** A random visitor identifier and session identifier stored in your browser, so we can count visits and keep the page you see consistent between visits.
- **Download records.** Which app, version, and platform you downloaded, so we can deliver the right installer and confirm that you have access.

**Information from payment providers**

Payments are processed by Stripe. We never see or store your full card number. Stripe shares with us the details needed to manage your subscription, such as your email address, plan, payment status, and billing country.

## How we use information

- To provide, operate, and secure the site and our apps.
- To sign you in and confirm what your account can access.
- To process payments, subscriptions, and refunds.
- To answer your messages and support requests.
- To understand and improve how the site performs.
- To prevent fraud and abuse, and to meet our legal obligations.

We do not sell your personal information, and we do not use it for third-party advertising.

## Cookies and browser storage

We use a small amount of browser storage to keep you signed in, remember your visitor and session identifiers, and remember interface preferences. We do not use third-party advertising cookies. You can clear this storage at any time in your browser settings; some features, such as staying signed in, will stop working until you sign in again.

## Sharing

We share information only with service providers that help us run the service, and only as needed for that purpose:

- **Stripe** for payments and billing.
- **Email delivery providers** to send sign-in links and replies to your messages.
- **Hosting and storage providers** that run our servers and deliver downloads.

We may also disclose information when required by law, to protect our rights or the safety of others, or as part of a merger, acquisition, or sale of assets, in which case this policy continues to apply to your information.

## Retention

We keep account and billing records for as long as your account is active and afterwards as required for tax, accounting, and legal purposes. Messages are kept for as long as needed to resolve them. Usage information is kept in aggregate or deleted when no longer needed.

## Your rights and choices

Depending on where you live, you may have the right to access, correct, export, or delete your personal information, or to object to or restrict certain uses of it. To make a request, contact us at {{contact_email}}. We will respond within the time required by applicable law and may need to confirm your identity first.

## Security

We protect information with encryption in transit, access controls, and passwordless authentication. No method of transmission or storage is completely secure, so we cannot guarantee absolute security.

## Children

The service is not directed to children under 13 (or the minimum age in your country), and we do not knowingly collect their personal information.

## International transfers

We and our service providers may process information in countries other than your own. Where required, we use appropriate safeguards for those transfers.

## Changes to this policy

We may update this policy from time to time. When we do, we will change the effective date at the top of this page, and for significant changes we will provide additional notice.

## Contact us

{{business_name}}
{{contact_address}}

Email: {{contact_email}}
`;

export const DEFAULT_TERMS_MARKDOWN = `These Terms and Conditions ("Terms") govern your use of {{website}} and the apps, downloads, and subscriptions offered by {{business_name}} ("we", "us", or "our"). By using the service you agree to these Terms. If you do not agree, do not use the service.

## Accounts

You sign in with a link sent to your email address. You are responsible for keeping access to that email account secure and for activity that happens under your account. Tell us promptly at {{contact_email}} if you believe your account has been used without permission.

## Subscriptions and payments

- Paid plans are billed in advance, monthly or yearly, or as a one-time purchase, as shown at checkout. Payments are processed by Stripe.
- Subscriptions renew automatically at the end of each billing period until you cancel. You can cancel at any time, and your plan stays active until the end of the period you have paid for.
- Prices may change. We will tell you before a price change applies to your subscription, and you can cancel before it takes effect.
- You are responsible for any taxes that apply to your purchase, unless they are included in the price shown.

## Refunds

If you are not satisfied, contact us within 30 days of your first payment for a full refund. After 30 days, payments are non-refundable except where required by law. To request a refund, use our contact page or email {{contact_email}}.

## Downloads and license

When you download one of our apps, we grant you a personal, non-exclusive, non-transferable, revocable license to install and use it for as long as your account or plan allows. You may not resell, sublicense, reverse engineer (except where the law allows it), or redistribute the apps. Open-source components are covered by their own licenses.

## Acceptable use

You agree not to:

- break the law or infringe anyone else's rights while using the service;
- try to gain unauthorized access to the service, other accounts, or our systems;
- interfere with or disrupt the service, including by sending automated traffic that places an unreasonable load on it;
- use the service to send spam or distribute malware.

We may suspend or end access for anyone who breaks these rules.

## Your content

You keep ownership of anything you create or upload with our apps. You give us only the permissions we need to operate the service for you.

## Our intellectual property

The service, including its software, design, text, and trademarks, belongs to {{business_name}} or its licensors and is protected by law. These Terms do not give you any rights to our trademarks.

## Feedback

If you send us ideas or suggestions, we may use them without any obligation to you.

## Disclaimers

The service is provided "as is" and "as available". To the fullest extent permitted by law, we disclaim all warranties, express or implied, including warranties of merchantability, fitness for a particular purpose, and non-infringement. We do not promise that the service will be uninterrupted or error-free.

## Limitation of liability

To the fullest extent permitted by law, {{business_name}} will not be liable for any indirect, incidental, special, consequential, or punitive damages, or for any loss of profits, data, or goodwill. Our total liability for any claim related to the service is limited to the amount you paid us in the 12 months before the claim arose.

## Termination

You can stop using the service and cancel your subscription at any time. We may suspend or end your access if you break these Terms or if we stop offering the service; if we stop offering a paid service, we will refund any prepaid, unused portion.

## Changes to these Terms

We may update these Terms from time to time. When we do, we will change the effective date at the top of this page, and for significant changes we will provide additional notice. Continuing to use the service after a change takes effect means you accept the updated Terms.

## Governing law

These Terms are governed by the laws of the place where {{business_name}} is established, without regard to conflict-of-law rules, except where the law of your country of residence requires otherwise.

## Contact us

{{business_name}}
{{contact_address}}

Email: {{contact_email}}
`;

export function defaultLegalMarkdown(kind: LegalDocumentKind): string {
  return kind === 'privacy' ? DEFAULT_PRIVACY_MARKDOWN : DEFAULT_TERMS_MARKDOWN;
}

/**
 * Substitutes identity tokens. Unconfigured values degrade to wording that still
 * reads correctly instead of leaking a raw placeholder to visitors.
 */
export function renderLegalTokens(markdown: string, identity: LegalIdentity): string {
  const values: Record<(typeof LEGAL_TOKENS)[number], string> = {
    business_name: identity.businessName,
    site_name: identity.siteName,
    contact_email: identity.contactEmail || 'the address on our contact page',
    contact_address: identity.contactAddress ?? '',
    website: identity.website ? identity.website.replace(/^https?:\/\//, '').replace(/\/$/, '') : 'this website',
    effective_date: identity.effectiveDate ?? '',
  };
  return markdown
    .replace(/\{\{\s*([a-z_]+)\s*\}\}/g, (match, token: string) => (token in values ? values[token as keyof typeof values] : match))
    // An unconfigured address leaves an empty line inside the contact block.
    .replace(/\n{3,}/g, '\n\n');
}
