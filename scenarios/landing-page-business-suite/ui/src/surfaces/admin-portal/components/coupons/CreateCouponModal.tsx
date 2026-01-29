import { useState, useCallback } from 'react';
import { X, RefreshCw, DollarSign, Percent } from 'lucide-react';
import { Button } from '../../../../shared/ui/button';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../../../../shared/ui/select';
import { inputClassName } from '../FormField';
import type { CreateCouponPayload } from '../../../../shared/api/billing';

interface CreateCouponModalProps {
  isOpen: boolean;
  onClose: () => void;
  onCreate: (payload: CreateCouponPayload) => Promise<{ success: boolean; error?: string }>;
  creating: boolean;
  error: string | null;
}

type DiscountType = 'percent' | 'amount';
type Duration = 'once' | 'repeating' | 'forever';

export function CreateCouponModal({ isOpen, onClose, onCreate, creating, error }: CreateCouponModalProps) {
  const [id, setId] = useState('');
  const [name, setName] = useState('');
  const [discountType, setDiscountType] = useState<DiscountType>('percent');
  const [percentOff, setPercentOff] = useState('');
  const [amountOff, setAmountOff] = useState('');
  const [currency, setCurrency] = useState('usd');
  const [duration, setDuration] = useState<Duration>('once');
  const [durationInMonths, setDurationInMonths] = useState('');
  const [maxRedemptions, setMaxRedemptions] = useState('');
  const [redeemByDate, setRedeemByDate] = useState('');

  const resetForm = useCallback(() => {
    setId('');
    setName('');
    setDiscountType('percent');
    setPercentOff('');
    setAmountOff('');
    setCurrency('usd');
    setDuration('once');
    setDurationInMonths('');
    setMaxRedemptions('');
    setRedeemByDate('');
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const payload: CreateCouponPayload = {
      duration,
    };

    if (id.trim()) {
      payload.id = id.trim();
    }
    if (name.trim()) {
      payload.name = name.trim();
    }

    if (discountType === 'percent') {
      const pct = parseFloat(percentOff);
      if (isNaN(pct) || pct <= 0 || pct > 100) {
        return;
      }
      payload.percent_off = pct;
    } else {
      const amt = parseFloat(amountOff);
      if (isNaN(amt) || amt <= 0) {
        return;
      }
      payload.amount_off = Math.round(amt * 100); // Convert to cents
      payload.currency = currency.toLowerCase();
    }

    if (duration === 'repeating') {
      const months = parseInt(durationInMonths, 10);
      if (isNaN(months) || months <= 0) {
        return;
      }
      payload.duration_in_months = months;
    }

    if (maxRedemptions.trim()) {
      const max = parseInt(maxRedemptions, 10);
      if (!isNaN(max) && max > 0) {
        payload.max_redemptions = max;
      }
    }

    if (redeemByDate) {
      const timestamp = Math.floor(new Date(redeemByDate).getTime() / 1000);
      if (timestamp > 0) {
        payload.redeem_by = timestamp;
      }
    }

    const result = await onCreate(payload);
    if (result.success) {
      resetForm();
    }
  };

  const handleClose = () => {
    resetForm();
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={handleClose} />

      {/* Modal */}
      <div className="relative bg-slate-900 border border-white/10 rounded-xl shadow-xl w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-white/10">
          <h2 className="text-lg font-semibold text-white">Create Coupon</h2>
          <button
            onClick={handleClose}
            className="text-slate-400 hover:text-white transition-colors"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="p-4 space-y-4">
          {/* Error display */}
          {error && (
            <div className="rounded-lg border border-red-500/20 bg-red-500/10 p-3 text-sm text-red-300">
              {error}
            </div>
          )}

          {/* ID (optional) */}
          <div>
            <label className="block text-sm text-slate-400 mb-1">
              Coupon ID <span className="text-slate-500">(optional, auto-generated if blank)</span>
            </label>
            <input
              type="text"
              value={id}
              onChange={(e) => setId(e.target.value.toUpperCase().replace(/[^A-Z0-9_-]/g, ''))}
              placeholder="e.g., SUMMER_SALE_20"
              className={inputClassName}
            />
          </div>

          {/* Name (optional) */}
          <div>
            <label className="block text-sm text-slate-400 mb-1">
              Display Name <span className="text-slate-500">(optional)</span>
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., Summer Sale 20% Off"
              className={inputClassName}
            />
          </div>

          {/* Discount type toggle */}
          <div>
            <label className="block text-sm text-slate-400 mb-2">Discount Type</label>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setDiscountType('percent')}
                className={`flex-1 flex items-center justify-center gap-2 py-2 px-3 rounded-lg border transition-colors ${
                  discountType === 'percent'
                    ? 'bg-emerald-500/20 border-emerald-500/50 text-emerald-300'
                    : 'bg-white/5 border-white/10 text-slate-400 hover:bg-white/10'
                }`}
              >
                <Percent className="h-4 w-4" />
                Percent Off
              </button>
              <button
                type="button"
                onClick={() => setDiscountType('amount')}
                className={`flex-1 flex items-center justify-center gap-2 py-2 px-3 rounded-lg border transition-colors ${
                  discountType === 'amount'
                    ? 'bg-emerald-500/20 border-emerald-500/50 text-emerald-300'
                    : 'bg-white/5 border-white/10 text-slate-400 hover:bg-white/10'
                }`}
              >
                <DollarSign className="h-4 w-4" />
                Amount Off
              </button>
            </div>
          </div>

          {/* Discount value */}
          {discountType === 'percent' ? (
            <div>
              <label className="block text-sm text-slate-400 mb-1">Percent Off (0-100)</label>
              <input
                type="number"
                value={percentOff}
                onChange={(e) => setPercentOff(e.target.value)}
                placeholder="e.g., 20"
                min="0.01"
                max="100"
                step="0.01"
                required
                className={inputClassName}
              />
            </div>
          ) : (
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm text-slate-400 mb-1">Amount Off</label>
                <input
                  type="number"
                  value={amountOff}
                  onChange={(e) => setAmountOff(e.target.value)}
                  placeholder="e.g., 10.00"
                  min="0.01"
                  step="0.01"
                  required
                  className={inputClassName}
                />
              </div>
              <div>
                <label className="block text-sm text-slate-400 mb-1">Currency</label>
                <Select value={currency} onValueChange={setCurrency}>
                  <SelectTrigger className="w-full bg-white/5 border-white/10">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="usd">USD</SelectItem>
                    <SelectItem value="eur">EUR</SelectItem>
                    <SelectItem value="gbp">GBP</SelectItem>
                    <SelectItem value="cad">CAD</SelectItem>
                    <SelectItem value="aud">AUD</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          )}

          {/* Duration */}
          <div>
            <label className="block text-sm text-slate-400 mb-1">Duration</label>
            <Select value={duration} onValueChange={(v) => setDuration(v as Duration)}>
              <SelectTrigger className="w-full bg-white/5 border-white/10">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="once">Once (applies to first invoice only)</SelectItem>
                <SelectItem value="repeating">Repeating (applies for N months)</SelectItem>
                <SelectItem value="forever">Forever (applies to all invoices)</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {/* Duration in months (conditional) */}
          {duration === 'repeating' && (
            <div>
              <label className="block text-sm text-slate-400 mb-1">Duration in Months</label>
              <input
                type="number"
                value={durationInMonths}
                onChange={(e) => setDurationInMonths(e.target.value)}
                placeholder="e.g., 3"
                min="1"
                required
                className={inputClassName}
              />
            </div>
          )}

          {/* Max redemptions (optional) */}
          <div>
            <label className="block text-sm text-slate-400 mb-1">
              Max Redemptions <span className="text-slate-500">(optional)</span>
            </label>
            <input
              type="number"
              value={maxRedemptions}
              onChange={(e) => setMaxRedemptions(e.target.value)}
              placeholder="Unlimited if blank"
              min="1"
              className={inputClassName}
            />
          </div>

          {/* Expiration date (optional) */}
          <div>
            <label className="block text-sm text-slate-400 mb-1">
              Expiration Date <span className="text-slate-500">(optional)</span>
            </label>
            <input
              type="date"
              value={redeemByDate}
              onChange={(e) => setRedeemByDate(e.target.value)}
              min={new Date().toISOString().split('T')[0]}
              className={inputClassName}
            />
          </div>

          {/* Actions */}
          <div className="flex items-center justify-end gap-3 pt-4 border-t border-white/10">
            <Button type="button" variant="ghost" onClick={handleClose} disabled={creating}>
              Cancel
            </Button>
            <Button type="submit" disabled={creating} className="gap-2">
              {creating && <RefreshCw className="h-4 w-4 animate-spin" />}
              Create Coupon
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
