import { useEffect } from 'react';
import {
  Users,
  Search,
  RefreshCw,
  ChevronLeft,
  ChevronRight,
  Monitor,
  CheckCircle2,
  XCircle,
  Clock,
  Ban,
  X,
} from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { LAYOUT } from '../config/layout.constants';
import { useUserManagement } from '../hooks/useUserManagement';
import {
  type UserAccount,
  type UserSession,
  formatSubscriptionStatus,
  formatCredits,
  parseUserAgent,
} from '../services/users.service';

/**
 * User Accounts - Admin page for managing user accounts and sessions.
 */
export function UserAccounts() {
  const {
    users,
    total,
    page,
    perPage,
    totalPages,
    search,
    setSearch,
    selectedUser,
    selectedUserSessions,
    loading,
    error,
    detailsLoading,
    sessionsLoading,
    actionLoading,
    loadUsers,
    setPage,
    selectUser,
    loadUserSessions,
    handleRevokeSession,
    handleRevokeAllSessions,
    clearError,
  } = useUserManagement();

  // Load sessions when user is selected
  useEffect(() => {
    if (selectedUser) {
      loadUserSessions(selectedUser.id);
    }
  }, [selectedUser, loadUserSessions]);

  return (
    <AdminLayout maxWidth="wide">
      <div className={LAYOUT.pageSpacing}>
        <PageHeader
          variant="icon-title"
          title="User Accounts"
          description="View and manage user accounts, subscriptions, and sessions"
          icon={Users}
          iconBgClass="bg-emerald-500/10"
          iconColorClass="text-emerald-400"
          testId="user-accounts-header"
          actions={
            <Button
              variant="ghost"
              size="sm"
              onClick={loadUsers}
              className="gap-2"
              disabled={loading}
            >
              <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
          }
        />

        {error && (
          <div className="flex items-center justify-between rounded-lg border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-rose-200">
            <span>{error}</span>
            <button onClick={clearError} className="text-rose-400 hover:text-rose-300">
              <X className="h-4 w-4" />
            </button>
          </div>
        )}

        <div className="grid gap-6 lg:grid-cols-3">
          {/* User List */}
          <div className="lg:col-span-2">
            <Card className={LAYOUT.card.base}>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>Users</CardTitle>
                    <CardDescription>
                      {total} total user{total !== 1 ? 's' : ''}
                    </CardDescription>
                  </div>
                  <div className="relative">
                    <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
                    <input
                      type="text"
                      value={search}
                      onChange={(e) => setSearch(e.target.value)}
                      placeholder="Search by email..."
                      className="w-64 rounded-lg border border-white/10 bg-slate-900/70 py-2 pl-10 pr-4 text-sm text-white placeholder:text-slate-500"
                    />
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                {loading && users.length === 0 ? (
                  <div className="flex items-center justify-center py-12 text-slate-400">
                    <RefreshCw className="h-5 w-5 animate-spin mr-2" />
                    Loading users...
                  </div>
                ) : users.length === 0 ? (
                  <div className="flex flex-col items-center justify-center py-12 text-slate-400">
                    <Users className="h-12 w-12 mb-4 opacity-50" />
                    <p>No users found</p>
                    {search && (
                      <p className="text-sm mt-1">
                        Try a different search term
                      </p>
                    )}
                  </div>
                ) : (
                  <>
                    <div className="overflow-hidden rounded-lg border border-white/10">
                      <table className="w-full">
                        <thead className="bg-slate-800/50">
                          <tr>
                            <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-400">
                              Email
                            </th>
                            <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-400">
                              Plan
                            </th>
                            <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-400">
                              Credits
                            </th>
                            <th className="px-4 py-3 text-left text-xs font-semibold uppercase tracking-wide text-slate-400">
                              Sessions
                            </th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-white/5">
                          {users.map((user) => (
                            <UserRow
                              key={user.id}
                              user={user}
                              selected={selectedUser?.id === user.id}
                              onSelect={() => selectUser(user)}
                            />
                          ))}
                        </tbody>
                      </table>
                    </div>

                    {/* Pagination */}
                    {totalPages > 1 && (
                      <div className="mt-4 flex items-center justify-between">
                        <p className="text-sm text-slate-400">
                          Showing {(page - 1) * perPage + 1} to{' '}
                          {Math.min(page * perPage, total)} of {total}
                        </p>
                        <div className="flex items-center gap-2">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setPage(page - 1)}
                            disabled={page <= 1}
                          >
                            <ChevronLeft className="h-4 w-4" />
                          </Button>
                          <span className="text-sm text-slate-400">
                            Page {page} of {totalPages}
                          </span>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setPage(page + 1)}
                            disabled={page >= totalPages}
                          >
                            <ChevronRight className="h-4 w-4" />
                          </Button>
                        </div>
                      </div>
                    )}
                  </>
                )}
              </CardContent>
            </Card>
          </div>

          {/* User Details Panel */}
          <div className="lg:col-span-1">
            {selectedUser ? (
              <UserDetailsPanel
                user={selectedUser}
                sessions={selectedUserSessions}
                loading={detailsLoading}
                sessionsLoading={sessionsLoading}
                actionLoading={actionLoading}
                onClose={() => selectUser(null)}
                onRevokeSession={handleRevokeSession}
                onRevokeAllSessions={handleRevokeAllSessions}
              />
            ) : (
              <Card className={LAYOUT.card.base}>
                <CardContent className="flex flex-col items-center justify-center py-12 text-slate-400">
                  <Users className="h-12 w-12 mb-4 opacity-50" />
                  <p>Select a user to view details</p>
                </CardContent>
              </Card>
            )}
          </div>
        </div>
      </div>
    </AdminLayout>
  );
}

interface UserRowProps {
  user: UserAccount;
  selected: boolean;
  onSelect: () => void;
}

function UserRow({ user, selected, onSelect }: UserRowProps) {
  return (
    <tr
      className={`cursor-pointer transition-colors hover:bg-white/5 ${
        selected ? 'bg-emerald-500/10' : ''
      }`}
      onClick={onSelect}
    >
      <td className="px-4 py-3">
        <div className="flex items-center gap-2">
          <span className="text-sm text-white">{user.email}</span>
          <span title={user.email_verified ? 'Email verified' : 'Email not verified'}>
            {user.email_verified ? (
              <CheckCircle2 className="h-4 w-4 text-emerald-400" />
            ) : (
              <XCircle className="h-4 w-4 text-slate-500" />
            )}
          </span>
        </div>
        {user.last_login_at && (
          <p className="text-xs text-slate-500">
            Last login: {new Date(user.last_login_at).toLocaleDateString()}
          </p>
        )}
      </td>
      <td className="px-4 py-3 text-sm text-slate-300">
        {formatSubscriptionStatus(user.subscription)}
      </td>
      <td className="px-4 py-3 text-sm text-slate-300">
        {formatCredits(user.credits)}
      </td>
      <td className="px-4 py-3 text-sm text-slate-300">
        {user.session_count} active
      </td>
    </tr>
  );
}

interface UserDetailsPanelProps {
  user: UserAccount;
  sessions: UserSession[];
  loading: boolean;
  sessionsLoading: boolean;
  actionLoading: string | null;
  onClose: () => void;
  onRevokeSession: (
    userId: string,
    sessionId: string
  ) => Promise<{ success: boolean; message?: string }>;
  onRevokeAllSessions: (
    userId: string
  ) => Promise<{ success: boolean; message?: string }>;
}

function UserDetailsPanel({
  user,
  sessions,
  loading,
  sessionsLoading,
  actionLoading,
  onClose,
  onRevokeSession,
  onRevokeAllSessions,
}: UserDetailsPanelProps) {
  const activeSessions = sessions.filter((s) => !s.revoked && new Date(s.expires_at) > new Date());

  return (
    <Card className={LAYOUT.card.base}>
      <CardHeader>
        <div className="flex items-center justify-between">
          <CardTitle className="text-lg">User Details</CardTitle>
          <button
            onClick={onClose}
            className="text-slate-400 hover:text-white"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
      </CardHeader>
      <CardContent className={LAYOUT.contentSpacing}>
        {loading ? (
          <div className="flex items-center justify-center py-8 text-slate-400">
            <RefreshCw className="h-5 w-5 animate-spin mr-2" />
            Loading...
          </div>
        ) : (
          <>
            {/* User Info */}
            <div className="space-y-3">
              <DetailRow label="Email" value={user.email} />
              <DetailRow
                label="Verified"
                value={
                  user.email_verified ? (
                    <span className="flex items-center gap-1 text-emerald-400">
                      <CheckCircle2 className="h-4 w-4" /> Yes
                    </span>
                  ) : (
                    <span className="flex items-center gap-1 text-slate-500">
                      <XCircle className="h-4 w-4" /> No
                    </span>
                  )
                }
              />
              <DetailRow
                label="Joined"
                value={new Date(user.created_at).toLocaleDateString()}
              />
              {user.last_login_at && (
                <DetailRow
                  label="Last Login"
                  value={new Date(user.last_login_at).toLocaleString()}
                />
              )}
            </div>

            {/* Subscription */}
            <div className="mt-6 rounded-lg border border-white/10 bg-slate-800/30 p-4">
              <h4 className="text-sm font-semibold text-white mb-3">Subscription</h4>
              {user.subscription ? (
                <div className="space-y-2">
                  <DetailRow
                    label="Plan"
                    value={user.subscription.plan_tier?.charAt(0).toUpperCase() + user.subscription.plan_tier?.slice(1)}
                  />
                  <DetailRow
                    label="Status"
                    value={
                      <span
                        className={
                          user.subscription.status === 'active'
                            ? 'text-emerald-400'
                            : 'text-amber-400'
                        }
                      >
                        {user.subscription.status}
                      </span>
                    }
                  />
                </div>
              ) : (
                <p className="text-sm text-slate-500">No active subscription</p>
              )}
            </div>

            {/* Credits */}
            <div className="mt-4 rounded-lg border border-white/10 bg-slate-800/30 p-4">
              <h4 className="text-sm font-semibold text-white mb-3">Credits</h4>
              {user.credits ? (
                <div className="space-y-2">
                  <DetailRow
                    label="Balance"
                    value={user.credits.balance.toLocaleString()}
                  />
                  <DetailRow
                    label="Bonus"
                    value={user.credits.bonus.toLocaleString()}
                  />
                  <DetailRow
                    label="Total"
                    value={(user.credits.balance + user.credits.bonus).toLocaleString()}
                  />
                </div>
              ) : (
                <p className="text-sm text-slate-500">No credit wallet</p>
              )}
            </div>

            {/* Sessions */}
            <div className="mt-6">
              <div className="flex items-center justify-between mb-3">
                <h4 className="text-sm font-semibold text-white">
                  Sessions ({activeSessions.length} active)
                </h4>
                {activeSessions.length > 0 && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => onRevokeAllSessions(user.id)}
                    disabled={actionLoading === 'all'}
                    className="text-rose-400 hover:text-rose-300 hover:bg-rose-500/10"
                  >
                    {actionLoading === 'all' ? (
                      <RefreshCw className="h-4 w-4 animate-spin mr-1" />
                    ) : (
                      <Ban className="h-4 w-4 mr-1" />
                    )}
                    Revoke All
                  </Button>
                )}
              </div>

              {sessionsLoading ? (
                <div className="flex items-center justify-center py-4 text-slate-400">
                  <RefreshCw className="h-4 w-4 animate-spin mr-2" />
                  Loading sessions...
                </div>
              ) : sessions.length === 0 ? (
                <p className="text-sm text-slate-500">No sessions found</p>
              ) : (
                <div className="space-y-2">
                  {sessions.map((session) => (
                    <SessionItem
                      key={session.id}
                      session={session}
                      userId={user.id}
                      loading={actionLoading === session.id}
                      onRevoke={onRevokeSession}
                    />
                  ))}
                </div>
              )}
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}

interface DetailRowProps {
  label: string;
  value: React.ReactNode;
}

function DetailRow({ label, value }: DetailRowProps) {
  return (
    <div className="flex justify-between items-center">
      <span className="text-xs text-slate-500 uppercase tracking-wide">{label}</span>
      <span className="text-sm text-slate-300">{value}</span>
    </div>
  );
}

interface SessionItemProps {
  session: UserSession;
  userId: string;
  loading: boolean;
  onRevoke: (
    userId: string,
    sessionId: string
  ) => Promise<{ success: boolean; message?: string }>;
}

function SessionItem({ session, userId, loading, onRevoke }: SessionItemProps) {
  const isExpired = new Date(session.expires_at) < new Date();
  const isActive = !session.revoked && !isExpired;

  return (
    <div
      className={`flex items-center justify-between rounded-lg border px-3 py-2 ${
        isActive
          ? 'border-white/10 bg-slate-800/30'
          : 'border-white/5 bg-slate-800/10 opacity-60'
      }`}
    >
      <div className="flex items-center gap-3">
        <Monitor className="h-4 w-4 text-slate-500" />
        <div>
          <p className="text-sm text-white">{parseUserAgent(session.user_agent)}</p>
          <div className="flex items-center gap-2 text-xs text-slate-500">
            {session.ip_address && <span>{session.ip_address}</span>}
            <span className="flex items-center gap-1">
              <Clock className="h-3 w-3" />
              {new Date(session.last_used_at).toLocaleDateString()}
            </span>
          </div>
        </div>
      </div>
      {isActive ? (
        <button
          onClick={() => onRevoke(userId, session.id)}
          disabled={loading}
          className="text-slate-400 hover:text-rose-400"
          title="Revoke session"
        >
          {loading ? (
            <RefreshCw className="h-4 w-4 animate-spin" />
          ) : (
            <Ban className="h-4 w-4" />
          )}
        </button>
      ) : (
        <span className="text-xs text-slate-500">
          {session.revoked ? 'Revoked' : 'Expired'}
        </span>
      )}
    </div>
  );
}
