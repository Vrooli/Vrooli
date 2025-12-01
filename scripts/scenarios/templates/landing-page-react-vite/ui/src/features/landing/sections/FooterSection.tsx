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
        { label: 'FAQ', url: '#faq' },
      ],
    },
    {
      title: 'Company',
      links: [
        { label: 'About', url: '/about' },
        { label: 'Blog', url: '/blog' },
        { label: 'Careers', url: '/careers' },
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
    <footer className="bg-slate-950 border-t border-white/10 relative overflow-hidden">
      {/* Background decoration */}
      <div className="absolute top-0 left-0 w-96 h-96 bg-purple-500/5 rounded-full blur-3xl"></div>
      <div className="absolute bottom-0 right-0 w-96 h-96 bg-blue-500/5 rounded-full blur-3xl"></div>

      <div className="container relative z-10 mx-auto px-6 py-16">
        <div className="max-w-6xl mx-auto">
          {/* Main footer content */}
          <div className="grid md:grid-cols-2 lg:grid-cols-5 gap-12 mb-12">
            {/* Brand column */}
            <div className="lg:col-span-2 space-y-4">
              <h3 className="text-2xl font-bold bg-gradient-to-r from-purple-400 to-blue-400 bg-clip-text text-transparent">
                {content.company_name || 'Landing Manager'}
              </h3>
              <p className="text-slate-400 leading-relaxed max-w-sm">
                {content.tagline || 'Build, launch, and optimize high-converting landing pages with AI-powered tools and built-in A/B testing.'}
              </p>

              {/* Social links */}
              <div className="flex gap-4 pt-4">
                {socialLinks.github && (
                  <a
                    href={socialLinks.github}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="w-10 h-10 rounded-lg bg-white/5 hover:bg-white/10 flex items-center justify-center transition-colors"
                    aria-label="GitHub"
                  >
                    <Github className="w-5 h-5 text-slate-400 hover:text-white transition-colors" />
                  </a>
                )}
                {socialLinks.twitter && (
                  <a
                    href={socialLinks.twitter}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="w-10 h-10 rounded-lg bg-white/5 hover:bg-white/10 flex items-center justify-center transition-colors"
                    aria-label="Twitter"
                  >
                    <Twitter className="w-5 h-5 text-slate-400 hover:text-white transition-colors" />
                  </a>
                )}
                {socialLinks.linkedin && (
                  <a
                    href={socialLinks.linkedin}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="w-10 h-10 rounded-lg bg-white/5 hover:bg-white/10 flex items-center justify-center transition-colors"
                    aria-label="LinkedIn"
                  >
                    <Linkedin className="w-5 h-5 text-slate-400 hover:text-white transition-colors" />
                  </a>
                )}
                {socialLinks.email && (
                  <a
                    href={`mailto:${socialLinks.email}`}
                    className="w-10 h-10 rounded-lg bg-white/5 hover:bg-white/10 flex items-center justify-center transition-colors"
                    aria-label="Email"
                  >
                    <Mail className="w-5 h-5 text-slate-400 hover:text-white transition-colors" />
                  </a>
                )}
              </div>
            </div>

            {/* Link columns */}
            {columns.map((column, index) => (
              <div key={index}>
                <h4 className="text-white font-semibold mb-4">{column.title}</h4>
                <ul className="space-y-3">
                  {column.links.map((link, linkIndex) => (
                    <li key={linkIndex}>
                      <a
                        href={link.url}
                        className="text-slate-400 hover:text-white transition-colors"
                      >
                        {link.label}
                      </a>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>

          {/* Bottom bar */}
          <div className="pt-8 border-t border-white/10">
            <div className="flex flex-col md:flex-row items-center justify-between gap-4">
              <p className="text-slate-400 text-sm">
                {content.copyright || `© ${new Date().getFullYear()} Landing Manager. All rights reserved.`}
              </p>
              <div className="flex gap-6 text-sm">
                <a href="/privacy" className="text-slate-400 hover:text-white transition-colors">
                  Privacy Policy
                </a>
                <a href="/terms" className="text-slate-400 hover:text-white transition-colors">
                  Terms of Service
                </a>
              </div>
            </div>
          </div>
        </div>
      </div>
    </footer>
  );
}
