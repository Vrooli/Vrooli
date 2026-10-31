export interface MailShortcut {
  label: string;
  href: string;
}

const PROVIDERS: { domains: string[]; label: string; href: string }[] = [
  { domains: ['gmail.com', 'googlemail.com'], label: 'Open Gmail', href: 'https://mail.google.com/mail/u/0/#search/in%3Aanywhere+newer_than%3A1h' },
  { domains: ['outlook.com', 'hotmail.com', 'live.com', 'msn.com'], label: 'Open Outlook', href: 'https://outlook.live.com/mail/0/' },
  { domains: ['yahoo.com', 'ymail.com'], label: 'Open Yahoo Mail', href: 'https://mail.yahoo.com/' },
  { domains: ['icloud.com', 'me.com', 'mac.com'], label: 'Open iCloud Mail', href: 'https://www.icloud.com/mail' },
  { domains: ['proton.me', 'protonmail.com', 'pm.me'], label: 'Open Proton Mail', href: 'https://mail.proton.me/' },
];

/** A webmail shortcut for well-known providers; custom domains get none. */
export function mailShortcutFor(email: string): MailShortcut | null {
  const domain = email.split('@')[1]?.toLowerCase();
  if (!domain) return null;
  const provider = PROVIDERS.find((entry) => entry.domains.includes(domain));
  return provider ? { label: provider.label, href: provider.href } : null;
}
