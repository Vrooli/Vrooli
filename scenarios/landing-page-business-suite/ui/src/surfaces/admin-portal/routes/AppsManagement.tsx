import { useNavigate } from 'react-router-dom';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { Button } from '../../../shared/ui/button';
import {
  AppWindow,
  Download,
  Activity,
  Gauge,
  RefreshCw,
  AlertTriangle,
  Package,
  Info,
} from 'lucide-react';
import { LAYOUT } from '../config/layout.constants';
import { useAdminHome } from '../hooks/useAdminHome';

/**
 * Apps Dashboard - Entry point for app and usage management
 *
 * Provides quick flows, stats, and navigation for:
 * - Download configuration
 * - Usage monitoring
 * - Credit limits (tier and app)
 */
export function AppsManagement() {
  const navigate = useNavigate();

  const {
    downloadsHealth,
    downloadsLoading,
    refreshDownloadsHealth,
  } = useAdminHome();

  return (
    <AdminLayout maxWidth="wide">
      <div className={LAYOUT.sectionSpacing}>
        <PageHeader
          title="Apps Dashboard"
          icon={AppWindow}
          iconBgClass="bg-blue-500/10"
          iconColorClass="text-blue-400"
          testId="apps-dashboard-header"
        />

        {/* Quick Flows */}
        <div className="mb-8" data-testid="apps-quick-flows">
          <p className="text-xs uppercase tracking-[0.3em] text-slate-500 mb-4">
            Quick flows
          </p>
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <QuickFlowCard
              title="Configure downloads"
              description="Set up app registry, platforms, and installers"
              icon={Download}
              iconBg="bg-blue-500/20"
              iconColor="text-blue-300"
              onClick={() => { navigate('/admin/downloads'); }}
              testId="flow-downloads"
            />
            <QuickFlowCard
              title="View usage"
              description="Monitor credit consumption across apps"
              icon={Activity}
              iconBg="bg-purple-500/20"
              iconColor="text-purple-300"
              onClick={() => { navigate('/admin/usage'); }}
              testId="flow-usage"
            />
            <QuickFlowCard
              title="Tier limits"
              description="Set credit limits per subscription tier"
              icon={Gauge}
              iconBg="bg-amber-500/20"
              iconColor="text-amber-300"
              onClick={() => { navigate('/admin/tier-limits'); }}
              testId="flow-tier-limits"
            />
            <QuickFlowCard
              title="App limits"
              description="Configure per-app usage limits"
              icon={Gauge}
              iconBg="bg-emerald-500/20"
              iconColor="text-emerald-300"
              onClick={() => { navigate('/admin/app-limits'); }}
              testId="flow-app-limits"
            />
          </div>
        </div>

        {/* Downloads Health */}
        <div
          className="mb-8 rounded-2xl border border-white/10 bg-white/5 p-6"
          data-testid="apps-downloads-health"
        >
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between mb-6">
            <div>
              <p className="text-xs uppercase tracking-[0.3em] text-slate-500">
                Downloads status
              </p>
              <h2 className="text-xl font-semibold text-white mt-1">
                App registry and platform coverage
              </h2>
            </div>
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  void refreshDownloadsHealth();
                }}
                className="gap-2"
                data-testid="apps-downloads-refresh"
              >
                <RefreshCw className="h-4 w-4" />
                Refresh
              </Button>
              <Button size="sm" onClick={() => { navigate('/admin/downloads'); }}>
                Configure downloads
              </Button>
            </div>
          </div>

          {downloadsLoading ? (
            <div className="grid gap-4 md:grid-cols-3">
              {[0, 1, 2].map((i) => (
                <div key={i} className="h-24 rounded-xl bg-white/5 animate-pulse" />
              ))}
            </div>
          ) : downloadsHealth ? (
            <div className="grid gap-4 md:grid-cols-3">
              <StatCard
                label="Apps configured"
                value={downloadsHealth.appCount}
                icon={Package}
                iconColor="text-blue-300"
                iconBg="bg-blue-500/20"
                description={
                  downloadsHealth.hasApps
                    ? `${String(downloadsHealth.appCount)} app${downloadsHealth.appCount !== 1 ? 's' : ''} in registry`
                    : 'No apps configured'
                }
              />
              <StatCard
                label="Platforms"
                value={downloadsHealth.platformsConfigured}
                icon={Download}
                iconColor="text-purple-300"
                iconBg="bg-purple-500/20"
                description={
                  downloadsHealth.platformsConfigured > 0
                    ? `${String(downloadsHealth.platformsConfigured)} platform${downloadsHealth.platformsConfigured !== 1 ? 's' : ''} with installers`
                    : 'No platforms configured'
                }
              />
              <StatCard
                label="Store links"
                value={downloadsHealth.storefrontsConfigured}
                icon={AppWindow}
                iconColor="text-emerald-300"
                iconBg="bg-emerald-500/20"
                description={
                  downloadsHealth.storefrontsConfigured > 0
                    ? `${String(downloadsHealth.storefrontsConfigured)} storefront${downloadsHealth.storefrontsConfigured !== 1 ? 's' : ''} linked`
                    : 'No store links'
                }
              />
            </div>
          ) : (
            <div className="rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 py-3 text-sm text-rose-100 flex items-center gap-3">
              <AlertTriangle className="h-4 w-4" />
              <span>Unable to load downloads status</span>
              <Button
                size="sm"
                variant="ghost"
                onClick={() => {
                  void refreshDownloadsHealth();
                }}
              >
                Retry
              </Button>
            </div>
          )}
        </div>

        {/* Guidance */}
        {downloadsHealth && !downloadsHealth.hasApps && (
          <div
            className="rounded-2xl border border-white/10 bg-gradient-to-br from-slate-900/80 via-slate-900/40 to-slate-900/90 p-6"
            data-testid="apps-guidance"
          >
            <div className="flex items-start gap-4">
              <div className="p-3 rounded-xl bg-blue-500/20">
                <Info className="h-6 w-6 text-blue-300" />
              </div>
              <div className="flex-1">
                <h3 className="text-lg font-semibold text-white mb-2">
                  Add your first app
                </h3>
                <p className="text-sm text-slate-400 mb-4">
                  Configure downloadable apps to enable the download section on your landing page.
                  You can add multiple apps with platform-specific installers and store links.
                </p>
                <Button
                  size="sm"
                  onClick={() => { navigate('/admin/downloads'); }}
                  data-testid="apps-add-first"
                >
                  Configure downloads
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Usage Info */}
        <div
          className="mt-8 rounded-2xl border border-white/10 bg-white/5 p-6"
          data-testid="apps-usage-info"
        >
          <div className="flex items-start gap-4">
            <div className="p-3 rounded-xl bg-purple-500/20">
              <Activity className="h-6 w-6 text-purple-300" />
            </div>
            <div className="flex-1">
              <h3 className="text-lg font-semibold text-white mb-2">
                Credit usage monitoring
              </h3>
              <p className="text-sm text-slate-400 mb-4">
                Track how users consume credits across your apps. Set tier-level limits to control
                subscription entitlements, and app-level limits to balance load across services.
              </p>
              <div className="flex flex-wrap gap-3">
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => { navigate('/admin/usage'); }}
                >
                  View usage dashboard
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => { navigate('/admin/tier-limits'); }}
                >
                  Configure tier limits
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </AdminLayout>
  );
}

interface QuickFlowCardProps {
  title: string;
  description: string;
  icon: React.ElementType;
  iconBg: string;
  iconColor: string;
  onClick: () => void;
  testId: string;
}

function QuickFlowCard({
  title,
  description,
  icon: Icon,
  iconBg,
  iconColor,
  onClick,
  testId,
}: QuickFlowCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="group rounded-xl border border-white/10 bg-white/5 p-4 text-left hover:bg-white/10 transition-all"
      data-testid={testId}
    >
      <div className="flex items-center gap-3 mb-2">
        <div className={`p-2 rounded-lg ${iconBg}`}>
          <Icon className={`h-4 w-4 ${iconColor}`} />
        </div>
        <p className="font-semibold text-white">{title}</p>
      </div>
      <p className="text-sm text-slate-400">{description}</p>
      <span className="mt-3 inline-flex items-center gap-1 text-xs font-semibold text-slate-300 group-hover:translate-x-1 transition-transform">
        Go →
      </span>
    </button>
  );
}

interface StatCardProps {
  label: string;
  value: number;
  icon: React.ElementType;
  iconColor: string;
  iconBg: string;
  description: string;
}

function StatCard({
  label,
  value,
  icon: Icon,
  iconColor,
  iconBg,
  description,
}: StatCardProps) {
  return (
    <div className="rounded-xl border border-white/10 bg-slate-900/40 p-4">
      <div className="flex items-center gap-3 mb-3">
        <div className={`p-2 rounded-lg ${iconBg}`}>
          <Icon className={`h-5 w-5 ${iconColor}`} />
        </div>
        <p className="text-xs uppercase tracking-[0.3em] text-slate-500">{label}</p>
      </div>
      <p className="text-3xl font-semibold text-white mb-1">{value}</p>
      <p className="text-xs text-slate-400">{description}</p>
    </div>
  );
}
