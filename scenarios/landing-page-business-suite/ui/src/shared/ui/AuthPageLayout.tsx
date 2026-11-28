import { ReactNode } from 'react';
import { SiteShell } from '../../surfaces/public-landing/site/SiteShell';
import '../../surfaces/user-auth/auth.css';

interface AuthPageLayoutProps {
  children: ReactNode;
  /** Visible card heading, when the children do not render their own. */
  title?: string;
  subtitle?: string;
  /** Document title when the visible heading lives in the children. */
  pageTitle?: string;
  description?: string;
  /** Explanatory panel beside the card on wide screens. */
  aside?: ReactNode;
  chrome?: 'full' | 'minimal';
  /** Changes when the step changes, so the card animates between steps. */
  stepKey?: string;
}

/** Sign-in surfaces share the public site chrome and are never indexed. */
export function AuthPageLayout({ children, title, subtitle, pageTitle, description, aside, chrome = 'full', stepKey }: AuthPageLayoutProps) {
  return (
    <SiteShell
      meta={{ title: pageTitle ?? title ?? 'Sign in', description: description ?? subtitle ?? 'Sign in to your account.', noindex: true }}
      width={aside ? 'wide' : 'narrow'}
      chrome={chrome}
    >
      <div className={`auth-layout ${aside ? 'auth-layout-split' : 'auth-layout-single'}`}>
        {aside}
        <section className="site-card site-auth auth-card" key={stepKey} data-step={stepKey}>
          {(title || subtitle) && (
            <header className="site-auth-head auth-head">
              {title && <h1>{title}</h1>}
              {subtitle && <p>{subtitle}</p>}
            </header>
          )}
          {children}
        </section>
      </div>
    </SiteShell>
  );
}
