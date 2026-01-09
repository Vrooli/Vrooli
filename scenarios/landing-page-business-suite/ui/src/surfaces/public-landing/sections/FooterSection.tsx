import { Github, Twitter, Linkedin, Mail } from 'lucide-react';

interface FooterLink {
  label: string;
  url: string;
}

interface FooterColumn {
  title: string;
  links: FooterLink[];
}

interface FooterSectionProps {
  content: {
    company_name?: string;
    tagline?: string;
    columns?: FooterColumn[];
    social_links?: {
      github?: string;
      twitter?: string;
      linkedin?: string;
      email?: string;
    };
    copyright?: string;
  };
}

export function FooterSection({ content }: FooterSectionProps) {
  const defaultColumns: FooterColumn[] = [
    {
      title: 'Product',
      links: [
        { label: 'Features', url: '#features' },
        { label: 'Pricing', url: '#pricing' },
        { label: 'Downloads', url: '#downloads-section' },
      ],
    },
    {
      title: 'Company',
      links: [
        { label: 'Docs', url: '/docs' },
        { label: 'Roadmap', url: '/roadmap' },
        { label: 'Support', url: '/support' },
      ],
    },
    {
      title: 'Legal',
      links: [
        { label: 'Privacy', url: '/privacy' },
        { label: 'Terms', url: '/terms' },
        { label: 'Security', url: '/security' },
      ],
    },
  ];

  const columns = content.columns || defaultColumns;
  const socialLinks = content.social_links || {};

  return (
    <footer className="border-t border-white/10 bg-[#07090F]">
      <div className="container mx-auto px-6 py-16">
        <div className="grid gap-12 md:grid-cols-2 lg:grid-cols-5">
          <div className="space-y-4 lg:col-span-2">
            <h3 className="text-2xl font-semibold text-white">
              {content.company_name || 'Silent Founder OS'}
            </h3>
            <p className="text-slate-400">
              {content.tagline || 'Build and market your business without a team. Vrooli Ascension leads, with more Vrooli apps joining the bundle soon.'}
            </p>
            <div className="flex gap-3 pt-2">
              {socialLinks.github && (
                <FooterIcon href={socialLinks.github} label="GitHub">
                  <Github className="h-5 w-5" />
                </FooterIcon>
              )}
              {socialLinks.twitter && (
                <FooterIcon href={socialLinks.twitter} label="Twitter">
                  <Twitter className="h-5 w-5" />
                </FooterIcon>
              )}
              {socialLinks.linkedin && (
                <FooterIcon href={socialLinks.linkedin} label="LinkedIn">
                  <Linkedin className="h-5 w-5" />
                </FooterIcon>
              )}
              {socialLinks.email && (
                <FooterIcon href={`mailto:${socialLinks.email}`} label="Email">
                  <Mail className="h-5 w-5" />
                </FooterIcon>
              )}
            </div>
          </div>
          {columns.map((column, index) => (
            <div key={index}>
              <h4 className="mb-3 text-sm font-semibold uppercase tracking-[0.25em] text-slate-500">{column.title}</h4>
              <ul className="space-y-2">
                {column.links.map((link, linkIndex) => (
                  <li key={linkIndex}>
                    <a href={link.url} className="text-slate-400 transition-colors hover:text-white">
                      {link.label}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
        <div className="mt-12 flex flex-col gap-4 border-t border-white/10 pt-6 text-sm text-slate-500 md:flex-row md:items-center md:justify-between">
          <p>{content.copyright || `© ${new Date().getFullYear()} Silent Founder OS by Vrooli. All rights reserved.`}</p>
          <div className="flex gap-6">
            <a href="/privacy" className="hover:text-white">
              Privacy Policy
            </a>
            <a href="/terms" className="hover:text-white">
              Terms of Service
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}

function FooterIcon({ href, label, children }: { href: string; label: string; children: React.ReactNode }) {
  return (
    <a
      href={href}
      aria-label={label}
      target={href.startsWith('http') ? '_blank' : undefined}
      rel={href.startsWith('http') ? 'noopener noreferrer' : undefined}
      className="flex h-10 w-10 items-center justify-center rounded-full border border-white/10 text-slate-400 transition-colors hover:border-white/40 hover:text-white"
    >
      {children}
    </a>
  );
}
