import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { FormSection } from '../components/FormSection';
import { Button } from '../../../shared/ui/button';
import { ToggleSwitch } from '../../../shared/ui/ToggleSwitch';
import { InlineAlert } from '../../../shared/ui/InlineAlert';
import { RefreshCw, Trash2, Download, Mail, Users, Clock, ExternalLink } from 'lucide-react';
import { useWaitlistForm } from '../hooks/useWaitlistForm';
import { formatDateTime } from '../../../shared/lib/dateFormatters';
import { LAYOUT } from '../config/layout.constants';

export function WaitlistManagement() {
  const {
    emails,
    comingSoonEnabled,
    stats,
    loading,
    error,
    deleting,
    togglingComingSoon,
    loadData,
    handleDelete,
    handleToggleComingSoon,
    handleExport,
  } = useWaitlistForm();

  return (
    <AdminLayout maxWidth="default">
      <div className={LAYOUT.pageSpacing}>
        <PageHeader
          title="Manage Waitlist Signups"
          description="View and manage email addresses collected from the coming soon page."
          icon={Mail}
          iconBgClass="bg-cyan-500/10"
          iconColorClass="text-cyan-400"
          testId="waitlist-header"
          actions={
            <>
              {emails.length > 0 && (
                <Button variant="outline" size="sm" onClick={handleExport} className="gap-2">
                  <Download className="h-4 w-4" />
                  Export CSV
                </Button>
              )}
              <Button variant="ghost" size="sm" onClick={() => { void loadData(); }} className="gap-2">
                <RefreshCw className="h-4 w-4" />
                Refresh
              </Button>
            </>
          }
        />

        {/* Stats */}
        {!loading && (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <div className="flex items-center gap-3 rounded-xl border border-white/10 bg-slate-900/50 px-4 py-3">
              <Users className="h-5 w-5 text-blue-400" />
              <div>
                <p className="text-2xl font-bold text-white">{stats.totalSignups}</p>
                <p className="text-xs text-slate-400">Total signups</p>
              </div>
            </div>
            <div className="flex items-center gap-3 rounded-xl border border-white/10 bg-slate-900/50 px-4 py-3">
              <Mail className="h-5 w-5 text-emerald-400" />
              <div>
                <p className="text-2xl font-bold text-white">{stats.comingSoonCount}</p>
                <p className="text-xs text-slate-400">From coming soon page</p>
              </div>
            </div>
          </div>
        )}

        {/* Coming Soon Mode Toggle */}
        <FormSection
          title="Coming Soon Mode"
          description="Control whether visitors see a coming soon page with email signup"
          icon={Clock}
          iconColorClass="text-amber-300"
          testId="waitlist-coming-soon-section"
        >
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
                <ToggleSwitch
                  checked={comingSoonEnabled}
                  onToggle={() => { void handleToggleComingSoon(); }}
                  loading={togglingComingSoon}
                  disabled={togglingComingSoon}
                  aria-label="Toggle coming soon mode"
                />
              </div>
            </div>
        </FormSection>

        {/* Error */}
        {error && (
          <InlineAlert
            severity="error"
            message={error}
            dismissible={false}
            className="mt-4"
            data-testid="waitlist-error"
          />
        )}

        {/* Email List */}
        <FormSection
          title="Collected Emails"
          description="Emails collected from visitors who signed up for updates"
          icon={Mail}
          iconColorClass="text-blue-300"
          testId="waitlist-emails-section"
        >
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
                          {formatDateTime(email.created_at, 'full')}
                        </td>
                        <td className="py-3 px-4 text-right">
                          <Button
                            variant="ghost"
                            size="sm"
                            aria-label={`Delete waitlist signup for ${email.email}`}
                            onClick={() => { void handleDelete(email.id); }}
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
        </FormSection>
      </div>
    </AdminLayout>
  );
}
