import { useState, useEffect } from 'react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { useToast } from '../../../shared/ui/Toast';
import { Activity, Users, Server, Calendar, RefreshCw, TrendingUp, ChevronLeft, ChevronRight } from 'lucide-react';
import {
  getAdminUsageSummary,
  AdminUsageSummary,
  formatCredits,
} from '../../../shared/api';

export function UsageDashboard() {
  const { addToast } = useToast();
  const [summary, setSummary] = useState<AdminUsageSummary | null>(null);
  const [loading, setLoading] = useState(true);
  const [billingPeriod, setBillingPeriod] = useState(() => {
    const now = new Date();
    return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
  });

  const fetchSummary = async () => {
    try {
      setLoading(true);
      const data = await getAdminUsageSummary(billingPeriod);
      setSummary(data);
    } catch (error) {
      addToast({
        type: 'error',
        message: error instanceof Error ? error.message : 'Failed to load usage data',
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchSummary();
  }, [billingPeriod]);

  const navigateMonth = (delta: number) => {
    const [year, month] = billingPeriod.split('-').map(Number);
    const date = new Date(year, month - 1 + delta, 1);
    setBillingPeriod(`${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`);
  };

  const formatPeriod = (period: string) => {
    const [year, month] = period.split('-').map(Number);
    return new Date(year, month - 1, 1).toLocaleDateString('en-US', {
      month: 'long',
      year: 'numeric',
    });
  };

  const totalUsage = summary
    ? Object.values(summary.user_totals).reduce((sum, val) => sum + val, 0)
    : 0;

  return (
    <AdminLayout>
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold">Usage Dashboard</h1>
            <p className="text-slate-400 mt-1">Monitor AI credit usage across all users</p>
          </div>
          <Button onClick={fetchSummary} disabled={loading} className="gap-2" variant="outline">
            <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </Button>
        </div>

        {/* Period Selector */}
        <Card>
          <CardContent className="py-4">
            <div className="flex items-center justify-center gap-4">
              <Button variant="ghost" size="sm" onClick={() => navigateMonth(-1)}>
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <div className="flex items-center gap-2">
                <Calendar className="h-5 w-5 text-slate-400" />
                <span className="text-lg font-medium">{formatPeriod(billingPeriod)}</span>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => navigateMonth(1)}
                disabled={
                  billingPeriod ===
                  `${new Date().getFullYear()}-${String(new Date().getMonth() + 1).padStart(2, '0')}`
                }
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </CardContent>
        </Card>

        {loading ? (
          <div className="text-center py-8 text-slate-400">Loading usage data...</div>
        ) : !summary ? (
          <Card className="border-dashed">
            <CardContent className="pt-6 text-center">
              <Activity className="h-12 w-12 text-slate-500 mx-auto mb-4" />
              <p className="text-slate-400">No usage data available</p>
            </CardContent>
          </Card>
        ) : (
          <>
            {/* Overview Stats */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Card>
                <CardContent className="pt-6">
                  <div className="flex items-center gap-4">
                    <div className="p-3 rounded-lg bg-blue-500/20">
                      <Users className="h-6 w-6 text-blue-400" />
                    </div>
                    <div>
                      <p className="text-sm text-slate-400">Active Users</p>
                      <p className="text-2xl font-bold">{summary.total_users}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardContent className="pt-6">
                  <div className="flex items-center gap-4">
                    <div className="p-3 rounded-lg bg-purple-500/20">
                      <TrendingUp className="h-6 w-6 text-purple-400" />
                    </div>
                    <div>
                      <p className="text-sm text-slate-400">Total Credits Used</p>
                      <p className="text-2xl font-bold">{formatCredits(totalUsage)}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>

              <Card>
                <CardContent className="pt-6">
                  <div className="flex items-center gap-4">
                    <div className="p-3 rounded-lg bg-emerald-500/20">
                      <Activity className="h-6 w-6 text-emerald-400" />
                    </div>
                    <div>
                      <p className="text-sm text-slate-400">Total Operations</p>
                      <p className="text-2xl font-bold">{summary.total_records}</p>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>

            {/* Usage by App */}
            {Object.keys(summary.app_totals).length > 0 && (
              <Card>
                <CardHeader>
                  <div className="flex items-center gap-3">
                    <Server className="h-5 w-5 text-slate-400" />
                    <div>
                      <CardTitle>Usage by App</CardTitle>
                      <CardDescription>Credit consumption per application</CardDescription>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    {Object.entries(summary.app_totals)
                      .sort(([, a], [, b]) => b - a)
                      .map(([app, usage]) => {
                        const percentage = totalUsage > 0 ? (usage / totalUsage) * 100 : 0;
                        return (
                          <div key={app} className="space-y-1">
                            <div className="flex items-center justify-between text-sm">
                              <span className="font-medium">{app}</span>
                              <span className="text-slate-400">
                                {formatCredits(usage)} ({percentage.toFixed(1)}%)
                              </span>
                            </div>
                            <div className="h-2 bg-slate-700 rounded-full overflow-hidden">
                              <div
                                className="h-full bg-blue-500 rounded-full transition-all"
                                style={{ width: `${percentage}%` }}
                              />
                            </div>
                          </div>
                        );
                      })}
                  </div>
                </CardContent>
              </Card>
            )}

            {/* Top Users */}
            {Object.keys(summary.user_totals).length > 0 && (
              <Card>
                <CardHeader>
                  <div className="flex items-center gap-3">
                    <Users className="h-5 w-5 text-slate-400" />
                    <div>
                      <CardTitle>Top Users</CardTitle>
                      <CardDescription>Highest credit consumers this period</CardDescription>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="divide-y divide-slate-700">
                    {Object.entries(summary.user_totals)
                      .sort(([, a], [, b]) => b - a)
                      .slice(0, 10)
                      .map(([user, usage], index) => (
                        <div key={user} className="flex items-center justify-between py-3">
                          <div className="flex items-center gap-3">
                            <span className="text-slate-500 w-6">{index + 1}.</span>
                            <span className="font-medium">{user}</span>
                          </div>
                          <span className="text-slate-400">{formatCredits(usage)} credits</span>
                        </div>
                      ))}
                  </div>
                </CardContent>
              </Card>
            )}

            {/* Recent Records */}
            {summary.records.length > 0 && (
              <Card>
                <CardHeader>
                  <div className="flex items-center gap-3">
                    <Activity className="h-5 w-5 text-slate-400" />
                    <div>
                      <CardTitle>Recent Activity</CardTitle>
                      <CardDescription>Latest usage records</CardDescription>
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="overflow-x-auto">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="text-left text-slate-400 border-b border-slate-700">
                          <th className="pb-3 pr-4">User</th>
                          <th className="pb-3 pr-4">Type</th>
                          <th className="pb-3 pr-4">App</th>
                          <th className="pb-3 pr-4 text-right">Credits</th>
                          <th className="pb-3 text-right">Last Activity</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-slate-700/50">
                        {summary.records.slice(0, 20).map((record) => (
                          <tr key={record.id}>
                            <td className="py-3 pr-4">{record.user_identity}</td>
                            <td className="py-3 pr-4">
                              <code className="text-xs bg-slate-800 px-2 py-0.5 rounded">
                                {record.limit_key}
                              </code>
                            </td>
                            <td className="py-3 pr-4 text-slate-400">
                              {record.app_bundle_key || '-'}
                            </td>
                            <td className="py-3 pr-4 text-right">
                              {formatCredits(record.usage_amount)}
                            </td>
                            <td className="py-3 text-right text-slate-400">
                              {record.last_operation_at
                                ? new Date(record.last_operation_at).toLocaleString()
                                : '-'}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </CardContent>
              </Card>
            )}
          </>
        )}
      </div>
    </AdminLayout>
  );
}
