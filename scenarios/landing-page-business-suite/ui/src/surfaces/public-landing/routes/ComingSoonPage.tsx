import { useState, FormEvent } from 'react';
import { submitWaitlistEmail, type PublicBranding } from '../../../shared/api';
import { Button } from '../../../shared/ui/button';
import { Input } from '../../../shared/ui/input';
import { CheckCircle, AlertCircle, Loader2, Mail } from 'lucide-react';

interface ComingSoonPageProps {
  branding: PublicBranding;
}

export function ComingSoonPage({ branding }: ComingSoonPageProps) {
  const [email, setEmail] = useState('');
  const [status, setStatus] = useState<'idle' | 'loading' | 'success' | 'error'>('idle');
  const [errorMessage, setErrorMessage] = useState('');

  const primaryColor = branding.theme_primary_color;
  const primaryFallback = 'rgb(var(--color-primary-default))';
  const primarySolid = primaryColor || primaryFallback;
  const primaryGlow = primaryColor ? `${primaryColor}20` : 'rgba(var(--color-primary-default),0.12)';
  const backgroundColor = branding.theme_background_color || 'rgb(var(--color-bg-base))';
  const successColor = 'rgb(var(--color-success))';
  const message = branding.coming_soon_message || 'We are working hard to bring you something amazing. Stay tuned!';

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();

    if (!email.trim()) {
      setStatus('error');
      setErrorMessage('Please enter your email');
      return;
    }

    setStatus('loading');
    setErrorMessage('');

    try {
      await submitWaitlistEmail(email.trim());
      setStatus('success');
      setEmail('');
    } catch (err) {
      setStatus('error');
      setErrorMessage(err instanceof Error ? err.message : 'Failed to submit email');
    }
  };

  return (
    <div
      className="min-h-screen flex flex-col items-center justify-center px-4"
      style={{ backgroundColor }}
    >
      {/* Gradient background effect */}
      <div
        className="absolute inset-0 opacity-30 pointer-events-none"
        style={{
          background: `radial-gradient(ellipse at center, ${primaryGlow} 0%, transparent 70%)`,
        }}
      />

      <div className="relative z-10 w-full max-w-md">
        {/* Logo/Brand */}
        <div className="text-center mb-8">
          {branding.logo_url ? (
            <img
              src={branding.logo_url}
              alt={branding.site_name}
              className="h-16 mx-auto mb-4 object-contain"
            />
          ) : branding.logo_icon_url ? (
            <img
              src={branding.logo_icon_url}
              alt={branding.site_name}
              className="h-16 w-16 mx-auto mb-4 object-contain"
            />
          ) : null}
          <h1 className="text-3xl font-bold text-white mb-2">
            {branding.site_name}
          </h1>
          {branding.tagline && (
            <p className="text-slate-400 text-sm">{branding.tagline}</p>
          )}
        </div>

        {/* Main Card */}
        <div className="rounded-2xl border border-white/10 bg-slate-900/60 backdrop-blur-sm p-8">
          <div className="text-center mb-8">
            <h2 className="text-2xl font-semibold text-white mb-4">
              Coming Soon
            </h2>
            <p className="text-slate-300 leading-relaxed">
              {message}
            </p>
          </div>

          {/* Email Form */}
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label htmlFor="email" className="sr-only">
                Email address
              </label>
              <div className="relative">
                <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                  <Mail className="h-5 w-5 text-slate-500" />
                </div>
                <Input
                  type="email"
                  id="email"
                  name="email"
                  size="md"
                  value={email}
                  onChange={(e) => {
                    setEmail(e.target.value);
                    if (status === 'error') setStatus('idle');
                  }}
                  placeholder="Enter your email"
                  disabled={status === 'loading' || status === 'success'}
                  className="pl-10 pr-4 focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500/50"
                />
              </div>
            </div>

            <Button
              type="submit"
              disabled={status === 'loading' || status === 'success'}
              className="w-full rounded-lg gap-2"
              style={{
                backgroundColor: status === 'success' ? successColor : primarySolid,
              }}
            >
              {status === 'loading' && (
                <>
                  <Loader2 className="h-5 w-5 animate-spin" />
                  Subscribing...
                </>
              )}
              {status === 'success' && (
                <>
                  <CheckCircle className="h-5 w-5" />
                  You're on the list!
                </>
              )}
              {(status === 'idle' || status === 'error') && (
                'Notify me when ready'
              )}
            </Button>

            {/* Error Message */}
            {status === 'error' && errorMessage && (
              <div className="flex items-center gap-2 text-rose-400 text-sm">
                <AlertCircle className="h-4 w-4 flex-shrink-0" />
                <span>{errorMessage}</span>
              </div>
            )}

            {/* Success Message */}
            {status === 'success' && (
              <p className="text-emerald-400 text-sm text-center">
                We'll let you know when we launch. Thank you for your interest!
              </p>
            )}
          </form>
        </div>

        {/* Footer */}
        <p className="text-center text-slate-500 text-xs mt-8">
          We respect your privacy and won't spam you.
        </p>
      </div>
    </div>
  );
}
