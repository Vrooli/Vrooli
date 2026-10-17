import { TwoFactorSettings } from '../components/TwoFactorSettings';
import { AuthPageLayout } from '../../../shared/ui/AuthPageLayout';
import { ShieldCheck, LockKeyhole, Smartphone } from 'lucide-react';

export function AdminEnrollSecondFactor() {
  return (
    <AuthPageLayout pageTitle="Secure your administrator account" chrome="minimal" stepKey="admin-mfa-enrollment" aside={<aside className="auth-aside" aria-label="Why use two-factor authentication"><p className="eyebrow auth-eyebrow">ADMIN SECURITY</p><h2 className="auth-aside-title">Protect the work behind <span>your site.</span></h2><p className="auth-aside-lede">Add a second layer of protection before you start managing your customers, content, and payments.</p><ul className="auth-points"><li><span className="auth-point-icon"><ShieldCheck /></span><span><strong>Stronger sign-in</strong><span>A stolen password alone won’t be enough.</span></span></li><li><span className="auth-point-icon"><LockKeyhole /></span><span><strong>Built for administrators</strong><span>Keep high-impact tools reserved for you.</span></span></li><li><span className="auth-point-icon"><Smartphone /></span><span><strong>Quick to finish</strong><span>Scan once, confirm a code, and you’re ready.</span></span></li></ul></aside>}>
      <div className="auth-step auth-mfa-page">
        <span className="auth-badge" aria-hidden="true"><ShieldCheck /></span>
        <header className="auth-head"><h1 id="admin-enroll-title">Secure your administrator account</h1><p>Set up two-factor authentication now, or confirm that you want to continue for this session and we’ll remind you next time.</p></header>
        <TwoFactorSettings enrollmentOnly onDeferred={() => { window.location.assign('/admin'); }} onEnrollmentComplete={() => { window.location.assign('/admin'); }} />
      </div>
    </AuthPageLayout>
  );
}
