import { TwoFactorSettings } from '../components/TwoFactorSettings';

export function AdminEnrollSecondFactor() {
  return (
    <main className="admin-page" aria-labelledby="admin-enroll-title">
      <h1 id="admin-enroll-title">Secure your administrator account</h1>
      <p>Two-factor authentication is required before administrator tools are available.</p>
      <TwoFactorSettings />
    </main>
  );
}
