import { useCallback, useState } from 'react';
import { Check, Loader2, Plus, Search, X } from 'lucide-react';
import { Button } from '../../../../shared/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../../../../shared/ui/dialog';
import { Input } from '../../../../shared/ui/input';
import { Label } from '../../../../shared/ui/label';
import { Textarea } from '../../../../shared/ui/textarea';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../../../../shared/ui/select';
import { Switch } from '../../../../shared/ui/switch';
import { Callout } from '../Callout';
import { createBundlePrice, verifyStripePrice, type CreateBundlePricePayload } from '../../../../shared/api/billing';

interface AddPlanModalProps {
  bundleKey: string;
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}

interface FormState {
  stripePriceId: string;
  planName: string;
  planTier: string;
  billingInterval: string;
  amountCents: string;
  currency: string;
  displayWeight: string;
  displayEnabled: boolean;
  monthlyIncludedCredits: string;
  subtitle: string;
  badge: string;
  ctaLabel: string;
  highlight: boolean;
  featuresText: string;
}

const DEFAULT_FORM: FormState = {
  stripePriceId: '',
  planName: '',
  planTier: 'pro',
  billingInterval: 'month',
  amountCents: '',
  currency: 'usd',
  displayWeight: '10',
  displayEnabled: true,
  monthlyIncludedCredits: '',
  subtitle: '',
  badge: '',
  ctaLabel: '',
  highlight: false,
  featuresText: '',
};

export function AddPlanModal({ bundleKey, isOpen, onClose, onSuccess }: AddPlanModalProps) {
  const [form, setForm] = useState<FormState>(DEFAULT_FORM);
  const [verifying, setVerifying] = useState(false);
  const [verified, setVerified] = useState(false);
  const [verifyError, setVerifyError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleChange = useCallback(
    <K extends keyof FormState>(field: K, value: FormState[K]) => {
      setForm((prev) => ({ ...prev, [field]: value }));
      // Reset verification when price ID changes
      if (field === 'stripePriceId') {
        setVerified(false);
        setVerifyError(null);
      }
    },
    []
  );

  const handleVerify = useCallback(async () => {
    const priceId = form.stripePriceId.trim();
    if (!priceId) {
      setVerifyError('Enter a Stripe price ID');
      return;
    }

    setVerifying(true);
    setVerifyError(null);
    setVerified(false);

    try {
      const info = await verifyStripePrice(priceId);
      setVerified(true);

      // Auto-fill fields from Stripe
      setForm((prev) => ({
        ...prev,
        amountCents: info.amount_cents?.toString() || prev.amountCents,
        currency: info.currency || prev.currency,
        billingInterval: info.interval || prev.billingInterval,
        planName: info.product || prev.planName,
      }));
    } catch (err) {
      setVerifyError(err instanceof Error ? err.message : 'Verification failed');
    } finally {
      setVerifying(false);
    }
  }, [form.stripePriceId]);

  const handleSave = useCallback(async () => {
    setError(null);

    // Validate required fields
    if (!form.stripePriceId.trim()) {
      setError('Stripe Price ID is required');
      return;
    }
    if (!form.planName.trim()) {
      setError('Plan Name is required');
      return;
    }
    if (!form.planTier) {
      setError('Plan Tier is required');
      return;
    }
    if (!form.billingInterval) {
      setError('Billing Interval is required');
      return;
    }

    setSaving(true);

    try {
      const features = form.featuresText
        .split('\n')
        .map((f) => f.trim())
        .filter((f) => f.length > 0);

      const payload: CreateBundlePricePayload = {
        stripe_price_id: form.stripePriceId.trim(),
        plan_name: form.planName.trim(),
        plan_tier: form.planTier,
        billing_interval: form.billingInterval,
        currency: form.currency || 'usd',
        display_weight: parseInt(form.displayWeight, 10) || 10,
        display_enabled: form.displayEnabled,
        highlight: form.highlight,
        features: features.length > 0 ? features : undefined,
      };

      if (form.amountCents) {
        payload.amount_cents = parseInt(form.amountCents, 10) || 0;
      }
      if (form.monthlyIncludedCredits) {
        payload.monthly_included_credits = parseInt(form.monthlyIncludedCredits, 10) || 0;
      }
      if (form.subtitle.trim()) {
        payload.subtitle = form.subtitle.trim();
      }
      if (form.badge.trim()) {
        payload.badge = form.badge.trim();
      }
      if (form.ctaLabel.trim()) {
        payload.cta_label = form.ctaLabel.trim();
      }

      await createBundlePrice(bundleKey, payload);
      setForm(DEFAULT_FORM);
      setVerified(false);
      onSuccess();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create plan');
    } finally {
      setSaving(false);
    }
  }, [form, bundleKey, onSuccess, onClose]);

  const handleClose = useCallback(() => {
    setForm(DEFAULT_FORM);
    setVerified(false);
    setVerifyError(null);
    setError(null);
    onClose();
  }, [onClose]);

  return (
    <Dialog open={isOpen} onOpenChange={(open: boolean) => !open && handleClose()}>
      <DialogContent className="max-w-lg max-h-[85vh] flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Plus className="h-5 w-5" />
            Add New Plan
          </DialogTitle>
          <DialogDescription>
            Create a new pricing plan. Enter a Stripe Price ID and verify it to auto-fill details.
          </DialogDescription>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto space-y-4 py-4">
          {error && <Callout type="error" message={error} />}

          {/* Stripe Price ID with Verify */}
          <div className="space-y-2">
            <Label htmlFor="stripe-price-id">Stripe Price ID *</Label>
            <div className="flex gap-2">
              <Input
                id="stripe-price-id"
                placeholder="price_..."
                value={form.stripePriceId}
                onChange={(e) => handleChange('stripePriceId', e.target.value)}
                className="flex-1 font-mono text-sm"
              />
              <Button
                type="button"
                variant="outline"
                onClick={handleVerify}
                disabled={verifying || !form.stripePriceId.trim()}
              >
                {verifying ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : verified ? (
                  <Check className="h-4 w-4 text-emerald-400" />
                ) : (
                  <Search className="h-4 w-4" />
                )}
              </Button>
            </div>
            {verifyError && (
              <p className="text-xs text-red-400">{verifyError}</p>
            )}
            {verified && (
              <p className="text-xs text-emerald-400">Price verified and fields auto-filled</p>
            )}
          </div>

          {/* Plan Name */}
          <div className="space-y-2">
            <Label htmlFor="plan-name">Plan Name *</Label>
            <Input
              id="plan-name"
              placeholder="Pro Monthly"
              value={form.planName}
              onChange={(e) => handleChange('planName', e.target.value)}
            />
          </div>

          {/* Plan Tier and Billing Interval */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Plan Tier *</Label>
              <Select value={form.planTier} onValueChange={(v) => handleChange('planTier', v)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="free">Free</SelectItem>
                  <SelectItem value="solo">Solo</SelectItem>
                  <SelectItem value="pro">Pro</SelectItem>
                  <SelectItem value="studio">Studio</SelectItem>
                  <SelectItem value="business">Business</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Billing Interval *</Label>
              <Select value={form.billingInterval} onValueChange={(v) => handleChange('billingInterval', v)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="month">Monthly</SelectItem>
                  <SelectItem value="year">Yearly</SelectItem>
                  <SelectItem value="one_time">One-time</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Amount and Currency */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="amount">Amount (cents)</Label>
              <Input
                id="amount"
                type="number"
                placeholder="7900"
                value={form.amountCents}
                onChange={(e) => handleChange('amountCents', e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label>Currency</Label>
              <Select value={form.currency} onValueChange={(v) => handleChange('currency', v)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="usd">USD</SelectItem>
                  <SelectItem value="eur">EUR</SelectItem>
                  <SelectItem value="gbp">GBP</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Display Weight */}
          <div className="space-y-2">
            <Label htmlFor="display-weight">Display Weight (higher = first)</Label>
            <Input
              id="display-weight"
              type="number"
              placeholder="10"
              value={form.displayWeight}
              onChange={(e) => handleChange('displayWeight', e.target.value)}
            />
          </div>

          {/* Toggles */}
          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>Display Enabled</Label>
              <p className="text-xs text-slate-400">Show this plan on the pricing page</p>
            </div>
            <Switch
              checked={form.displayEnabled}
              onCheckedChange={(checked: boolean) => handleChange('displayEnabled', checked)}
            />
          </div>

          <div className="flex items-center justify-between">
            <div className="space-y-0.5">
              <Label>Highlight</Label>
              <p className="text-xs text-slate-400">Feature this plan as recommended</p>
            </div>
            <Switch
              checked={form.highlight}
              onCheckedChange={(checked: boolean) => handleChange('highlight', checked)}
            />
          </div>

          {/* Marketing Copy */}
          <div className="space-y-2">
            <Label htmlFor="subtitle">Subtitle</Label>
            <Input
              id="subtitle"
              placeholder="For growing teams"
              value={form.subtitle}
              onChange={(e) => handleChange('subtitle', e.target.value)}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="badge">Badge</Label>
              <Input
                id="badge"
                placeholder="Popular"
                value={form.badge}
                onChange={(e) => handleChange('badge', e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="cta-label">CTA Label</Label>
              <Input
                id="cta-label"
                placeholder="Get Started"
                value={form.ctaLabel}
                onChange={(e) => handleChange('ctaLabel', e.target.value)}
              />
            </div>
          </div>

          {/* Features */}
          <div className="space-y-2">
            <Label htmlFor="features">Features (one per line)</Label>
            <Textarea
              id="features"
              placeholder="Unlimited projects&#10;Priority support&#10;Advanced analytics"
              value={form.featuresText}
              onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => handleChange('featuresText', e.target.value)}
              rows={4}
            />
          </div>
        </div>

        <DialogFooter className="border-t border-slate-700 pt-4">
          <Button variant="outline" onClick={handleClose} disabled={saving}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Creating...
              </>
            ) : (
              <>
                <Plus className="mr-2 h-4 w-4" />
                Create Plan
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
