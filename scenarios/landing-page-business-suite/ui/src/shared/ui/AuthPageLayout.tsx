import { ReactNode } from 'react';

interface AuthPageLayoutProps {
  children: ReactNode;
  title?: string;
  subtitle?: string;
}

export function AuthPageLayout({ children, title, subtitle }: AuthPageLayoutProps) {
  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-900 via-slate-800 to-slate-900 flex items-center justify-center p-4">
      <div className="w-full max-w-md">
        <div className="bg-slate-800/50 backdrop-blur-sm border border-slate-700/50 rounded-2xl p-8 shadow-2xl">
          {(title || subtitle) && (
            <div className="text-center mb-8">
              {title && (
                <h1 className="text-2xl font-bold text-white mb-2">{title}</h1>
              )}
              {subtitle && (
                <p className="text-slate-400">{subtitle}</p>
              )}
            </div>
          )}
          {children}
        </div>
      </div>
    </div>
  );
}
