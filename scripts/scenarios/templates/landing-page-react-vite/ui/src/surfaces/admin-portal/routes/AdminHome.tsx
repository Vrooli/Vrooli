import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AdminLayout } from "../components/AdminLayout";
import { Button } from "../../../shared/ui/button";
import { Activity, BarChart3, Palette, History } from "lucide-react";
import { getAdminExperienceSnapshot, type AdminExperienceSnapshot } from "../../../shared/lib/adminExperience";

/**
 * Admin home page - implements ADMIN-MODES requirement (OT-P0-009)
 *
 * Shows exactly two modes:
 * 1. Analytics / Metrics
 * 2. Customization
 *
 * Navigation efficiency: ≤ 3 clicks to any customization card (OT-P0-010)
 *
 * [REQ:ADMIN-MODES] [REQ:ADMIN-NAV]
 */
export function AdminHome() {
  const navigate = useNavigate();
  const [experience, setExperience] = useState<AdminExperienceSnapshot | null>(null);

  useEffect(() => {
    setExperience(getAdminExperienceSnapshot());
  }, []);

  const resumeVariant = experience?.lastVariant;
  const resumeAnalytics = experience?.lastAnalytics;

  const handleResumeVariant = () => {
    if (!resumeVariant) return;
    const basePath =
      resumeVariant.surface === "section" && resumeVariant.sectionId
        ? `/admin/customization/variants/${resumeVariant.slug}/sections/${resumeVariant.sectionId}`
        : `/admin/customization/variants/${resumeVariant.slug}`;
    navigate(basePath);
  };

  const handleResumeAnalytics = () => {
    if (!resumeAnalytics) return;
    const params = new URLSearchParams();
    if (resumeAnalytics.variantSlug) {
      params.set("variant", resumeAnalytics.variantSlug);
    }
    if (resumeAnalytics.timeRangeDays && resumeAnalytics.timeRangeDays !== 7) {
      params.set("range", String(resumeAnalytics.timeRangeDays));
    }
    const query = params.toString() ? `?${params.toString()}` : "";
    navigate(`/admin/analytics${query}`);
  };

  return (
    <AdminLayout>
      <div className="max-w-4xl mx-auto">
        <h1 className="text-3xl font-semibold mb-8">Landing Manager Admin</h1>

        <div className="grid md:grid-cols-2 gap-6">
          {/* Analytics / Metrics Mode */}
          <button
            onClick={() => navigate("/admin/analytics")}
            className="group relative rounded-2xl border border-white/10 bg-white/5 p-8 shadow-xl backdrop-blur hover:bg-white/10 transition-all text-left"
            data-testid="admin-mode-analytics"
          >
            <div className="flex items-center gap-4 mb-4">
              <div className="p-3 rounded-xl bg-blue-500/20">
                <BarChart3 className="h-8 w-8 text-blue-400" />
              </div>
              <h2 className="text-2xl font-semibold">Analytics / Metrics</h2>
            </div>
            <p className="text-slate-300 mb-4">
              View conversion rates, A/B test results, visitor metrics, and performance data across all variants.
            </p>
            <Button className="mt-4 group-hover:translate-x-1 transition-transform">
              View Analytics →
            </Button>
          </button>

          {/* Customization Mode */}
          <button
            onClick={() => navigate("/admin/customization")}
            className="group relative rounded-2xl border border-white/10 bg-white/5 p-8 shadow-xl backdrop-blur hover:bg-white/10 transition-all text-left"
            data-testid="admin-mode-customization"
          >
            <div className="flex items-center gap-4 mb-4">
              <div className="p-3 rounded-xl bg-purple-500/20">
                <Palette className="h-8 w-8 text-purple-400" />
              </div>
              <h2 className="text-2xl font-semibold">Customization</h2>
            </div>
            <p className="text-slate-300 mb-4">
              Customize landing page content, trigger agent-based improvements, manage A/B test variants, and configure site settings.
            </p>
            <Button className="mt-4 group-hover:translate-x-1 transition-transform">
              Customize Site →
            </Button>
          </button>
        </div>

        {(resumeVariant || resumeAnalytics) && (
          <div className="mt-10 space-y-4" data-testid="admin-resume-panel">
            <p className="text-sm uppercase tracking-[0.3em] text-slate-500">Continue where you left off</p>
            <div className="grid gap-4 md:grid-cols-2">
              {resumeVariant && (
                <div className="rounded-2xl border border-white/10 bg-white/5 p-5" data-testid="admin-resume-card">
                  <div className="flex items-center gap-3 mb-3">
                    <div className="p-2 rounded-xl bg-purple-500/15">
                      <History className="h-5 w-5 text-purple-300" />
                    </div>
                    <div>
                      <p className="text-sm text-slate-400">Last customization</p>
                      <p className="text-lg font-semibold">
                        {resumeVariant.name ?? resumeVariant.slug}
                        {resumeVariant.surface === "section" && resumeVariant.sectionType && (
                          <span className="text-sm font-normal text-slate-400"> · {resumeVariant.sectionType}</span>
                        )}
                      </p>
                    </div>
                  </div>
                  <p className="text-sm text-slate-400 mb-4">
                    {resumeVariant.surface === "section" ? "Resume editing the section preview you left open." : "Jump back into the variant settings and section list."}
                  </p>
                  <Button onClick={handleResumeVariant} className="w-full gap-2" data-testid="admin-resume-customization">
                    Return to {resumeVariant.surface === "section" ? "Section" : "Variant"}
                  </Button>
                </div>
              )}

              {resumeAnalytics && (
                <div className="rounded-2xl border border-white/10 bg-white/5 p-5" data-testid="admin-resume-analytics-card">
                  <div className="flex items-center gap-3 mb-3">
                    <div className="p-2 rounded-xl bg-blue-500/15">
                      <Activity className="h-5 w-5 text-blue-300" />
                    </div>
                    <div>
                      <p className="text-sm text-slate-400">Last analytics view</p>
                      <p className="text-lg font-semibold">
                        {resumeAnalytics.variantName ?? (resumeAnalytics.variantSlug ?? "All variants")}
                      </p>
                    </div>
                  </div>
                  <p className="text-sm text-slate-400 mb-4">
                    Showing last {resumeAnalytics.timeRangeDays} day{resumeAnalytics.timeRangeDays === 1 ? "" : "s"} window.
                  </p>
                  <Button onClick={handleResumeAnalytics} variant="outline" className="w-full gap-2" data-testid="admin-resume-analytics">
                    Reopen Analytics
                  </Button>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </AdminLayout>
  );
}
