import { useNavigate } from "react-router-dom";
import { AdminLayout } from "../components/AdminLayout";
import { Button } from "../../../shared/ui/button";
import { BarChart3, Palette } from "lucide-react";

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
      </div>
    </AdminLayout>
  );
}
