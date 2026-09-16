/** Shared system states, not product/marketing fallback content. */
export const presentationSystemUi = {
  loading: 'Loading page…', unavailable: 'This page is currently unavailable.',
  unavailableDetail: 'Published presentation content could not be loaded. Please try again later.',
  notFound: 'Page not found', notFoundDetail: 'This page is not available at this address.',
  retry: 'Try again', leave: 'Discard unsaved presentation changes and leave this page?',
  signIn: 'Sign in', legalNav: 'Legal and contact', contact: 'Contact', privacy: 'Privacy', terms: 'Terms',
};

/** Neutral download/commerce UI vocabulary; product claims remain page-owned. */
export const downloadSystemUi = {
  title: 'Downloads', back: 'Back to app', platform: 'Platform', release: 'Release',
  noInstallers: 'No installers are currently available for this app.',
  choose: 'Choose a release', prepare: 'Prepare download', preparing: 'Preparing download…',
  ready: 'Your authorized download link is ready.', open: 'Download file',
  unavailable: 'Downloads are currently unavailable.', failed: 'Unable to authorize this download. Please try again.',
  changed: 'This release has changed. Reload the page before downloading.',
  invalidId: 'This release has no valid download identifier. Reload the page before downloading.',
  duplicateId: 'This release has a duplicate download identifier. Reload the page before downloading.',
  signIn: 'Sign in', signInRequired: 'Sign in to check access to this download.',
  denied: 'Your account does not currently have access to this download.',
  checking: 'Checking session…', recheck: 'Recheck session', signedIn: 'Signed in',
  access: 'Download access is verified when you prepare a download.',
  returnNote: 'After signing in, return to this page and recheck your session.',
  checksum: 'Checksum', notes: 'Release notes', file: 'File',
  signingNotice: { summary: 'Release trust', linkFallback: 'Learn more' },
  billing: { month: 'Billed monthly', year: 'Billed yearly', one_time: 'One-time payment' },
  standardPrice: 'Standard price shown. Any applicable introductory terms are confirmed at checkout.',
};
