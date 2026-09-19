/** Neutral site vocabulary. Business facts come from Branding, never from here. */
export const siteCopy = {
  skip: 'Skip to content', menu: 'Menu', primaryNavLabel: 'Site', footerLabel: 'Site footer', footerNavLabel: 'Company',
  home: 'Home', contact: 'Contact', privacy: 'Privacy', terms: 'Terms', signIn: 'Sign in',
  company: 'Company', reachUs: 'Reach us', sendMessage: 'Send us a message',

  contactPage: {
    title: 'Contact', eyebrow: 'Contact',
    heading: 'How can we help?',
    lede: 'Questions, bug reports, refund requests, ideas — every message is read by a person.',
    description: 'Get in touch for support, refunds, bug reports, or feature requests.',
    formHeading: 'Send a message', responseTime: 'We reply within 1–2 business days.',
    emailHeading: 'Email', addressHeading: 'Mailing address', refundHeading: '30-day refunds',
    refundBody: 'Not the right fit? Ask for a refund within 30 days of your first payment.',
    topicLegend: 'What is this about?',
    email: 'Your email', emailHint: 'We only use this to reply.',
    orderId: 'Order or subscription ID', optional: 'optional', orderHint: 'Found in your Stripe receipt email.',
    subject: 'Subject', message: 'Message',
    submit: 'Send message', submitting: 'Sending…',
    errors: {
      emailRequired: 'Enter your email address.',
      emailInvalid: 'Enter an email address like name@example.com.',
      subjectRequired: 'Add a short subject.',
      subjectLong: 'Keep the subject under 200 characters.',
      messageRequired: 'Tell us a little more.',
      messageShort: 'Add a few more details so we can help (at least 10 characters).',
      timeout: 'That took too long. Check your connection and try again.',
      network: 'We could not reach the server. Check your connection and try again.',
      server: 'Something went wrong on our side. Your message was not sent — please try again.',
    },
  },
  topics: {
    general: { label: 'General question', description: 'Anything else on your mind' },
    bug: { label: 'Report a bug', description: 'Something is not working' },
    feature: { label: 'Suggest a feature', description: 'An idea that would help you' },
    refund: { label: 'Request a refund', description: 'Within 30 days of purchase' },
  },

  thankYou: {
    title: 'Thank you', description: 'Thank you for getting in touch.',
    eyebrow: 'All set',
    message: { heading: 'Message received.', body: 'Thanks for writing in. We reply within 1–2 business days — keep an eye on your inbox.' },
    refund: { heading: 'Refund request received.', body: 'We will confirm by email within 1–2 business days. Approved refunds reach your account in 5–7 business days.' },
    checkout: { heading: 'You are all set.', body: 'Your payment went through. A receipt is on its way to your inbox, and your plan is active now.' },
    waitlist: { heading: 'You are on the list.', body: 'We will email you once — when it is ready.' },
    home: 'Back to home', contact: 'Contact us', signIn: 'Sign in to your account',
    questions: 'Questions?',
  },

  legal: {
    privacy: { title: 'Privacy Policy', description: 'How we collect, use, and protect your information.' },
    terms: { title: 'Terms and Conditions', description: 'The terms that govern use of our website, apps, and subscriptions.' },
    eyebrow: 'Legal', effective: 'Effective', contents: 'On this page', questions: 'Questions about this document?', contact: 'Contact us',
  },

  notFound: {
    title: 'Page not found', description: 'The page you were looking for does not exist.',
    code: '404', heading: 'Page not found',
    body: 'This page drifted out of orbit — the link may be broken, or the page may have moved. Here are some places to start instead.',
    home: 'Go to home', contact: 'Contact us',
  },
  unavailable: {
    title: 'Temporarily unavailable', heading: 'This page is currently unavailable.',
    body: 'This page could not be loaded just now. Please try again in a moment.', retry: 'Try again', contact: 'Contact us',
  },
  loading: 'Loading page…',
} as const;
