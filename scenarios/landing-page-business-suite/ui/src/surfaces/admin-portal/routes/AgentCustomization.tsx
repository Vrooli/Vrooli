import { useNavigate } from 'react-router-dom';
import { Sparkles, ArrowLeft, Info } from 'lucide-react';
import { AdminLayout } from '../components/AdminLayout';
import { PageHeader } from '../components/PageHeader';
import { FormSection } from '../components/FormSection';
import { FormField, textareaClassName } from '../components/FormField';
import { LAYOUT } from '../config/layout.constants';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '../../../shared/ui/card';
import { Button } from '../../../shared/ui/button';
import { Textarea } from '../../../shared/ui/input';
import { InlineAlert } from '../../../shared/ui/InlineAlert';
import { useAgentForm } from '../hooks/useAgentForm';

/**
 * Agent Customization Trigger
 * Implements OT-P0-005 (Trigger agent customization run)
 * Implements OT-P0-006 (Structured agent input: text + assets + goals)
 *
 * [REQ:AGENT-TRIGGER] [REQ:AGENT-INPUT]
 */
export function AgentCustomization() {
  const navigate = useNavigate();
  const {
    form,
    result,
    submitting,
    error,
    validationError,
    setBrief,
    setAssets,
    setPreview,
    handleSubmit,
    clearResult,
    clearValidationError,
  } = useAgentForm();

  const onSubmit = async () => {
    await handleSubmit();
  };

  return (
    <AdminLayout maxWidth="narrow">
      <div className={LAYOUT.pageSpacing}>
        <PageHeader
          variant="icon-title"
          title="Agent Customization"
          description="Trigger AI-powered customization of your landing page"
          icon={Sparkles}
          iconBgClass="bg-violet-500/10"
          iconColorClass="text-violet-400"
          testId="agent-header"
          actions={
            <Button
              variant="ghost"
              size="sm"
              onClick={() => navigate('/admin/customization')}
              className="gap-2"
            >
              <ArrowLeft className="h-4 w-4" />
              Back
            </Button>
          }
        />

        {error && (
          <div className="rounded-xl border border-red-500/20 bg-red-500/10 p-4 mb-6">
            <p className="text-red-400">{error}</p>
          </div>
        )}

        {validationError && (
          <InlineAlert
            severity="warning"
            message={validationError.message}
            title={validationError.title}
            onDismiss={clearValidationError}
            className="mb-6"
            data-testid="agent-validation-alert"
          />
        )}

        {result ? (
          <Card className={LAYOUT.card.base}>
            <CardHeader>
              <CardTitle className="text-green-400">Agent Customization Triggered</CardTitle>
              <CardDescription className="text-slate-400">
                Your customization request has been queued
              </CardDescription>
            </CardHeader>
            <CardContent className={LAYOUT.contentSpacing}>
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
                <Button onClick={clearResult} variant="outline" className="flex-1">
                  Create Another Request
                </Button>
                <Button onClick={() => navigate('/admin/customization')} className="flex-1">
                  Back to Customization
                </Button>
              </div>
            </CardContent>
          </Card>
        ) : (
          <FormSection
            title="Customization Brief"
            description="Provide structured input for the AI agent to customize your landing page"
            icon={Sparkles}
            iconColorClass="text-violet-300"
          >
            <FormField
              label="Brief *"
              htmlFor="brief"
              helpText="Be specific about your goals, target audience, and desired changes"
            >
              <Textarea
                id="brief"
                value={form.brief}
                onChange={(e) => setBrief(e.target.value)}
                className={textareaClassName}
                rows={8}
                placeholder="Describe what you want the agent to customize. Examples:&#10;• Make the hero section more compelling for B2B SaaS&#10;• Add social proof with customer logos&#10;• Improve CTA button copy for higher conversions&#10;• Optimize pricing section for enterprise customers"
                data-testid="agent-brief-input"
              />
            </FormField>

            <FormField
              label="Assets (optional)"
              htmlFor="assets"
              helpText="URLs to images, logos, or other assets the agent should use"
            >
              <Textarea
                id="assets"
                value={form.assets}
                onChange={(e) => setAssets(e.target.value)}
                className={textareaClassName}
                rows={4}
                placeholder="Asset URLs (one per line):&#10;https://example.com/logo.png&#10;https://example.com/hero-image.jpg"
                data-testid="agent-assets-input"
              />
            </FormField>

              {/* Preview Mode */}
              <div className="flex items-center gap-3">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={form.preview}
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
                onClick={onSubmit}
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
          </FormSection>
        )}

        <FormSection
          title="How it works"
          description="The agent will analyze your brief, apply customizations to the selected variant, and provide a preview. You can review changes before they go live."
          icon={Info}
          iconColorClass="text-blue-300"
          className="bg-blue-500/10 border-blue-500/20"
        >
          <div className="hidden" />
        </FormSection>
      </div>
    </AdminLayout>
  );
}
