import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Sparkles, ArrowLeft, Upload } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../components/ui/card';
import { Button } from '../../../components/ui/button';
import { triggerAgentCustomization } from '../../../services/api';

/**
 * Agent Customization Trigger
 * Implements OT-P0-005 (Trigger agent customization run)
 * Implements OT-P0-006 (Structured agent input: text + assets + goals)
 *
 * [REQ:AGENT-TRIGGER] [REQ:AGENT-INPUT]
 */
export function AgentCustomization() {
  const navigate = useNavigate();
  const [brief, setBrief] = useState('');
  const [assets, setAssets] = useState('');
  const [preview, setPreview] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<{ job_id: string; status: string; agent_id: string } | null>(null);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    if (!brief.trim()) {
      alert('Please provide a brief for the agent');
      return;
    }

    try {
      setSubmitting(true);
      setError(null);

      const assetList = assets
        .split('\n')
        .map(a => a.trim())
        .filter(a => a.length > 0);

      const response = await triggerAgentCustomization(
        'landing-page', // scenario ID
        brief.trim(),
        assetList,
        preview
      );

      setResult(response);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to trigger agent customization');
      console.error('Agent customization error:', err);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AdminLayout>
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="flex items-center gap-4 mb-8">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate('/admin/customization')}
            className="gap-2"
          >
            <ArrowLeft className="h-4 w-4" />
            Back
          </Button>
          <div className="flex-1">
            <h1 className="text-3xl font-semibold flex items-center gap-3">
              <Sparkles className="h-8 w-8 text-blue-400" />
              Agent Customization
            </h1>
            <p className="text-slate-400 mt-1">
              Trigger AI-powered customization of your landing page
            </p>
          </div>
        </div>

        {error && (
          <div className="rounded-xl border border-red-500/20 bg-red-500/10 p-4 mb-6">
            <p className="text-red-400">{error}</p>
          </div>
        )}

        {result ? (
          <Card className="bg-white/5 border-white/10">
            <CardHeader>
              <CardTitle className="text-green-400">Agent Customization Triggered</CardTitle>
              <CardDescription className="text-slate-400">
                Your customization request has been queued
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="bg-slate-900 rounded-lg p-4 space-y-2">
                <div className="flex justify-between">
                  <span className="text-slate-400">Job ID:</span>
                  <span className="font-mono">{result.job_id}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">Status:</span>
                  <span className="capitalize">{result.status}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">Agent ID:</span>
                  <span className="font-mono">{result.agent_id}</span>
                </div>
              </div>

              <div className="flex gap-3">
                <Button
                  onClick={() => {
                    setResult(null);
                    setBrief('');
                    setAssets('');
                  }}
                  variant="outline"
                  className="flex-1"
                >
                  Create Another Request
                </Button>
                <Button
                  onClick={() => navigate('/admin/customization')}
                  className="flex-1"
                >
                  Back to Customization
                </Button>
              </div>
            </CardContent>
          </Card>
        ) : (
          <Card className="bg-white/5 border-white/10">
            <CardHeader>
              <CardTitle>Customization Brief</CardTitle>
              <CardDescription className="text-slate-400">
                Provide structured input for the AI agent to customize your landing page
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {/* Brief */}
              <div>
                <label htmlFor="brief" className="block text-sm font-medium text-slate-300 mb-2">
                  Brief <span className="text-red-400">*</span>
                </label>
                <textarea
                  id="brief"
                  value={brief}
                  onChange={(e) => setBrief(e.target.value)}
                  className="w-full px-4 py-3 bg-slate-900 border border-white/10 rounded-lg focus:border-blue-500 focus:outline-none"
                  rows={8}
                  placeholder="Describe what you want the agent to customize. Examples:&#10;• Make the hero section more compelling for B2B SaaS&#10;• Add social proof with customer logos&#10;• Improve CTA button copy for higher conversions&#10;• Optimize pricing section for enterprise customers"
                  data-testid="agent-brief-input"
                />
                <p className="text-xs text-slate-500 mt-1">
                  Be specific about your goals, target audience, and desired changes
                </p>
              </div>

              {/* Assets */}
              <div>
                <label htmlFor="assets" className="block text-sm font-medium text-slate-300 mb-2">
                  Assets (optional)
                </label>
                <textarea
                  id="assets"
                  value={assets}
                  onChange={(e) => setAssets(e.target.value)}
                  className="w-full px-4 py-3 bg-slate-900 border border-white/10 rounded-lg focus:border-blue-500 focus:outline-none"
                  rows={4}
                  placeholder="Asset URLs (one per line):&#10;https://example.com/logo.png&#10;https://example.com/hero-image.jpg"
                  data-testid="agent-assets-input"
                />
                <p className="text-xs text-slate-500 mt-1">
                  URLs to images, logos, or other assets the agent should use
                </p>
              </div>

              {/* Preview Mode */}
              <div className="flex items-center gap-3">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={preview}
                    onChange={(e) => setPreview(e.target.checked)}
                    className="w-4 h-4"
                    data-testid="agent-preview-input"
                  />
                  <span className="text-sm text-slate-300">Preview mode</span>
                </label>
                <span className="text-xs text-slate-500">
                  (recommended: review changes before applying)
                </span>
              </div>

              {/* Submit */}
              <div className="flex gap-3 pt-4 border-t border-white/10">
                <Button
                  onClick={() => navigate('/admin/customization')}
                  variant="outline"
                  className="flex-1"
                  disabled={submitting}
                >
                  Cancel
                </Button>
                <Button
                  onClick={handleSubmit}
                  className="flex-1 gap-2"
                  disabled={submitting}
                  data-testid="agent-submit"
                >
                  {submitting ? (
                    <>
                      <div className="animate-spin h-4 w-4 border-2 border-white/20 border-t-white rounded-full" />
                      Submitting...
                    </>
                  ) : (
                    <>
                      <Sparkles className="h-4 w-4" />
                      Trigger Agent Customization
                    </>
                  )}
                </Button>
              </div>
            </CardContent>
          </Card>
        )}

        {/* Info Card */}
        <Card className="bg-blue-500/10 border-blue-500/20 mt-6">
          <CardContent className="py-4">
            <p className="text-sm text-blue-200">
              <strong>How it works:</strong> The agent will analyze your brief, apply customizations
              to the selected variant, and provide a preview. You can review changes before they go live.
            </p>
          </CardContent>
        </Card>
      </div>
    </AdminLayout>
  );
}
