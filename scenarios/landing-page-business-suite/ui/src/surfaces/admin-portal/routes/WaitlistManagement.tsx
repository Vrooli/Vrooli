import { useCallback, useEffect, useState } from 'react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import {
  getWaitlistEmails,
  deleteWaitlistEmail,
  getWaitlistExportUrl,
  getBranding,
  updateBranding,
  type WaitlistEmail,
} from '../../../shared/api';
import { RefreshCw, Trash2, Download, Mail, Users, AlertCircle, Clock, ExternalLink } from 'lucide-react';

export function WaitlistManagement() {
  const [emails, setEmails] = useState<WaitlistEmail[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<number | null>(null);
  const [comingSoonEnabled, setComingSoonEnabled] = useState(false);
  const [togglingComingSoon, setTogglingComingSoon] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [emailsData, brandingData] = await Promise.all([
        getWaitlistEmails(),
        getBranding(),
      ]);
      setEmails(emailsData || []);
      setComingSoonEnabled(brandingData?.coming_soon_enabled ?? false);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadData();
  }, [loadData]);

  const handleToggleComingSoon = async () => {
    setTogglingComingSoon(true);
    try {
      const newValue = !comingSoonEnabled;
      await updateBranding({ coming_soon_enabled: newValue });
      setComingSoonEnabled(newValue);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to toggle coming soon mode');
    } finally {
      setTogglingComingSoon(false);
    }
  };

  const handleDelete = async (id: number) => {
    setDeleting(id);
    try {
      await deleteWaitlistEmail(id);
      setEmails((prev) => prev.filter((e) => e.id !== id));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete email');
    } finally {
      setDeleting(null);
    }
  };

  const handleExport = () => {
    window.open(getWaitlistExportUrl(), '_blank');
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleString();
  };

  return (
    <AdminLayout>
      <div className="space-y-8">
        {/* Header */}
        <div className="rounded-2xl border border-white/10 bg-gradient-to-br from-slate-900/80 via-slate-900/40 to-slate-900/90 p-6">
          <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
            <div className="flex-1">
              <p className="text-xs uppercase tracking-[0.3em] text-slate-500">Waitlist</p>
              <h1 className="text-2xl font-bold text-white mt-1">Manage Waitlist Signups</h1>
              <p className="text-slate-400 text-sm mt-2">
                View and manage email addresses collected from the coming soon page.
              </p>
            </div>
            <div className="flex flex-wrap gap-2">
              {emails.length > 0 && (
                <Button variant="outline" size="sm" onClick={handleExport} className="gap-2">
                  <Download className="h-4 w-4" />
                  Export CSV
                </Button>
              )}
              <Button variant="ghost" size="sm" onClick={loadData} className="gap-2">
                <RefreshCw className="h-4 w-4" />
                Refresh
              </Button>
            </div>
          </div>

          {/* Stats */}
          {!loading && (
            <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              <div className="flex items-center gap-3 rounded-xl border border-white/10 bg-slate-900/50 px-4 py-3">
                <Users className="h-5 w-5 text-blue-400" />
                <div>
                  <p className="text-2xl font-bold text-white">{emails.length}</p>
                  <p className="text-xs text-slate-400">Total signups</p>
                </div>
              </div>
              <div className="flex items-center gap-3 rounded-xl border border-white/10 bg-slate-900/50 px-4 py-3">
                <Mail className="h-5 w-5 text-emerald-400" />
                <div>
                  <p className="text-2xl font-bold text-white">
                    {emails.filter((e) => e.source === 'coming_soon').length}
                  </p>
                  <p className="text-xs text-slate-400">From coming soon page</p>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Coming Soon Mode Toggle */}
        <Card className="border-white/10 bg-slate-900/60">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Clock className="h-5 w-5 text-amber-300" /> Coming Soon Mode
            </CardTitle>
            <CardDescription>
              Control whether visitors see a "coming soon" page with email signup
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between">
              <div className="flex-1">
                <p className="text-sm text-white">
                  {comingSoonEnabled ? (
                    <span className="flex items-center gap-2">
                      <span className="h-2 w-2 rounded-full bg-amber-400 animate-pulse" />
                      Coming soon mode is active
                    </span>
                  ) : (
                    <span className="flex items-center gap-2">
                      <span className="h-2 w-2 rounded-full bg-slate-500" />
                      Coming soon mode is disabled
                    </span>
                  )}
                </p>
                <p className="text-xs text-slate-400 mt-1">
                  {comingSoonEnabled
                    ? 'Visitors see a coming soon page and can sign up for notifications'
                    : 'Visitors see your normal landing page'}
                </p>
              </div>
              <div className="flex items-center gap-3">
                <a
                  href="/admin/branding"
                  className="text-xs text-blue-400 hover:text-blue-300 flex items-center gap-1"
                >
                  Customize message
                  <ExternalLink className="h-3 w-3" />
                </a>
                <button
                  type="button"
                  role="switch"
                  aria-checked={comingSoonEnabled}
                  disabled={togglingComingSoon}
                  onClick={handleToggleComingSoon}
                  className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors disabled:opacity-50 ${
                    comingSoonEnabled ? 'bg-amber-500' : 'bg-slate-700'
                  }`}
                >
                  {togglingComingSoon ? (
                    <RefreshCw className="h-4 w-4 text-white absolute left-1/2 -translate-x-1/2 animate-spin" />
                  ) : (
                    <span
                      className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                        comingSoonEnabled ? 'translate-x-6' : 'translate-x-1'
                      }`}
                    />
                  )}
                </button>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Error */}
        {error && (
          <div className="flex items-center gap-2 rounded-lg border border-rose-500/30 bg-rose-500/10 p-4">
            <AlertCircle className="h-5 w-5 text-rose-400" />
            <p className="text-sm text-rose-300">{error}</p>
          </div>
        )}

        {/* Email List */}
        <Card className="border-white/10 bg-slate-900/60">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Mail className="h-5 w-5 text-blue-300" /> Collected Emails
            </CardTitle>
            <CardDescription>
              Emails collected from visitors who signed up for updates
            </CardDescription>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="text-slate-400 py-8 text-center">Loading...</div>
            ) : emails.length === 0 ? (
              <div className="text-slate-400 py-8 text-center">
                <Mail className="h-12 w-12 mx-auto mb-4 opacity-30" />
                <p>No emails collected yet</p>
                <p className="text-sm mt-2">
                  Enable coming soon mode in Branding Settings to start collecting emails.
                </p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b border-white/10">
                      <th className="text-left py-3 px-4 text-xs font-semibold uppercase tracking-wide text-slate-400">
                        Email
                      </th>
                      <th className="text-left py-3 px-4 text-xs font-semibold uppercase tracking-wide text-slate-400">
                        Source
                      </th>
                      <th className="text-left py-3 px-4 text-xs font-semibold uppercase tracking-wide text-slate-400">
                        Date
                      </th>
                      <th className="text-right py-3 px-4 text-xs font-semibold uppercase tracking-wide text-slate-400">
                        Actions
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {emails.map((email) => (
                      <tr key={email.id} className="border-b border-white/5 hover:bg-slate-800/50">
                        <td className="py-3 px-4 text-sm text-white">{email.email}</td>
                        <td className="py-3 px-4">
                          <span className="inline-flex items-center px-2 py-1 rounded-full text-xs bg-slate-700 text-slate-300">
                            {email.source}
                          </span>
                        </td>
                        <td className="py-3 px-4 text-sm text-slate-400">
                          {formatDate(email.created_at)}
                        </td>
                        <td className="py-3 px-4 text-right">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleDelete(email.id)}
                            disabled={deleting === email.id}
                            className="text-rose-400 hover:text-rose-300 hover:bg-rose-500/10"
                          >
                            {deleting === email.id ? (
                              <RefreshCw className="h-4 w-4 animate-spin" />
                            ) : (
                              <Trash2 className="h-4 w-4" />
                            )}
                          </Button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </AdminLayout>
  );
}
