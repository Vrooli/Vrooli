import { useState, useEffect } from 'react';
import { Loader2, Tag } from 'lucide-react';
import { Button } from '../../../../shared/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '../../../../shared/ui/dialog';
import { FormField } from '../FormField';
import { inputClassName } from '../formFieldClasses';
import { Callout } from '../Callout';
import type { StripeCoupon } from '../../../../shared/api/billing';

interface EditCouponModalProps {
  coupon: StripeCoupon | null;
  isOpen: boolean;
  onClose: () => void;
  onSave: (couponId: string, name: string) => Promise<{ success: boolean; error?: string }>;
  saving: boolean;
}

/**
 * Format discount amount for display
 */
function formatDiscount(coupon: StripeCoupon): string {
  if (coupon.percent_off != null) {
    return `${String(coupon.percent_off)}% off`;
  }
  if (coupon.amount_off != null) {
    const currency = (coupon.currency || 'usd').toUpperCase();
    const amount = coupon.amount_off / 100;
    return `$${amount.toFixed(2)} ${currency} off`;
  }
  return 'Unknown discount';
}

export function EditCouponModal({
  coupon,
  isOpen,
  onClose,
  onSave,
  saving,
}: EditCouponModalProps) {
  const [name, setName] = useState('');
  const [error, setError] = useState<string | null>(null);

  // Reset form when coupon changes
  useEffect(() => {
    if (coupon) {
      setName(coupon.name || '');
      setError(null);
    }
  }, [coupon]);

  const handleSave = async () => {
    if (!coupon) return;
    setError(null);

    const result = await onSave(coupon.id, name);
    if (result.success) {
      onClose();
    } else {
      setError(result.error || 'Failed to update coupon');
    }
  };

  const handleClose = () => {
    if (!saving) {
      setError(null);
      onClose();
    }
  };

  if (!coupon) return null;

  return (
    <Dialog open={isOpen} onOpenChange={(open) => { if (!open) { handleClose(); } }}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Tag className="h-5 w-5" />
            Edit Coupon
          </DialogTitle>
          <DialogDescription>
            Update the display name for this coupon. Note: Discount amount, duration, and other
            settings cannot be changed after creation.
          </DialogDescription>
        </DialogHeader>

        <div className="py-4 space-y-4">
          {/* Coupon ID (read-only) */}
          <FormField label="Coupon ID">
            <input
              type="text"
              value={coupon.id}
              disabled
              className={`${inputClassName} opacity-60 cursor-not-allowed`}
            />
          </FormField>

          {/* Discount info (read-only) */}
          <div className="p-3 rounded-md bg-slate-800/50 border border-slate-700/50">
            <p className="text-sm text-slate-400">Discount</p>
            <p className="text-lg font-semibold text-emerald-400">{formatDiscount(coupon)}</p>
            <p className="text-xs text-slate-500 mt-1">
              Duration: {coupon.duration}
              {coupon.duration === 'repeating' && coupon.duration_in_months
                ? ` (${String(coupon.duration_in_months)} months)`
                : ''}
            </p>
          </div>

          {/* Editable name */}
          <FormField label="Display Name">
            <input
              type="text"
              value={name}
              onChange={(e) => { setName(e.target.value); }}
              placeholder="Enter a display name (optional)"
              className={inputClassName}
              disabled={saving}
            />
            <p className="mt-1 text-xs text-slate-400">
              A friendly name shown to customers. Leave blank to use the coupon ID.
            </p>
          </FormField>

          {error && <Callout type="error" message={error} />}
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={handleClose} disabled={saving}>
            Cancel
          </Button>
          <Button onClick={() => { void handleSave(); }} disabled={saving} className="gap-2">
            {saving && <Loader2 className="h-4 w-4 animate-spin" />}
            {saving ? 'Saving...' : 'Save Changes'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
