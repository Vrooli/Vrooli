import { describe, expect, it } from 'vitest';
import type { StripeCoupon } from '../api/schemas/billing.schema';
import {
  calculateDiscountedPrice,
  formatDiscountBadge,
  formatDiscountPreview,
  getCouponSummary,
} from './pricingCalculations';

const coupon = (overrides: Partial<StripeCoupon> = {}): StripeCoupon => ({
  id: 'coupon_1',
  duration: 'once',
  times_redeemed: 0,
  valid: true,
  created: 1,
  is_intro_coupon: false,
  ...overrides,
});

describe('pricingCalculations', () => {
  it('calculates percent and fixed discounts without negative totals', () => {
    expect(calculateDiscountedPrice(1000, coupon({ percent_off: 25 }))).toMatchObject({
      discountedCents: 750,
      savingsCents: 250,
      savingsPercent: 25,
      hasDiscount: true,
    });
    expect(calculateDiscountedPrice(500, coupon({ amount_off: 1000 }))).toMatchObject({
      discountedCents: 0,
      savingsCents: 500,
    });
  });

  it('formats duration-aware purchase messaging', () => {
    const repeating = coupon({ percent_off: 50, duration: 'repeating', duration_in_months: 24 });
    expect(formatDiscountBadge(2000, repeating, 'year')).toBe('50% off for 2 years');
    expect(formatDiscountPreview(2000, repeating)).toBe('$10 (save $10)');
    expect(getCouponSummary(repeating)).toBe('50% off (24 months)');
  });

  it('does not advertise invalid or non-discounting coupons', () => {
    expect(formatDiscountBadge(2000, coupon({ valid: false, percent_off: 50 }), 'month')).toBeUndefined();
    expect(formatDiscountBadge(2000, coupon(), 'month')).toBeUndefined();
  });

  it('preserves the original price when a coupon is absent, invalid, or has no usable discount', () => {
    expect(calculateDiscountedPrice(1000, null)).toMatchObject({ discountedCents: 1000, savingsCents: 0, hasDiscount: false });
    expect(calculateDiscountedPrice(1000, coupon({ valid: false, percent_off: 25 }))).toMatchObject({ discountedCents: 1000, hasDiscount: false });
    expect(calculateDiscountedPrice(0, coupon({ percent_off: 25 }))).toMatchObject({ discountedCents: 0, hasDiscount: false });
    expect(calculateDiscountedPrice(1000, coupon({ percent_off: 0, amount_off: 0 }))).toMatchObject({ discountedCents: 1000, hasDiscount: false });
    expect(formatDiscountPreview(0, coupon({ percent_off: 25 }))).toBeUndefined();
    expect(formatDiscountPreview(1000, coupon({ percent_off: 0 }))).toBeUndefined();
  });

  it('formats each supported percent and fixed-amount duration without overstating its term', () => {
    expect(formatDiscountBadge(2000, coupon({ percent_off: 20, duration: 'once' }), 'month')).toBe('First month 20% off');
    expect(formatDiscountBadge(2000, coupon({ percent_off: 20, duration: 'forever' }), 'month')).toBe('20% off');
    expect(formatDiscountBadge(2000, coupon({ percent_off: 20, duration: 'repeating' }), 'month')).toBe('20% off ');
    expect(formatDiscountBadge(2000, coupon({ amount_off: 500, duration: 'once' }), 'year')).toBe('First year $15');
    expect(formatDiscountBadge(2000, coupon({ amount_off: 500, duration: 'repeating', duration_in_months: 1 }), 'month')).toBe('$15 for 1 month');
    expect(formatDiscountBadge(2000, coupon({ amount_off: 500, duration: 'repeating', duration_in_months: 12 }), 'year')).toBe('$15 for 1 year');
  });

  it('summarizes coupon amounts and all duration variants for an operator', () => {
    expect(getCouponSummary(coupon({ amount_off: 500, currency: 'usd', duration: 'once' }))).toBe('$5 off (first payment)');
    expect(getCouponSummary(coupon({ amount_off: 500, currency: 'usd', duration: 'forever' }))).toBe('$5 off (forever)');
    expect(getCouponSummary(coupon({ amount_off: 500, currency: 'usd', duration: 'repeating' }))).toBe('$5 off');
    expect(getCouponSummary(coupon({ percent_off: 10, duration: 'unknown' as never }))).toBe('10% off');
    expect(getCouponSummary(coupon({ name: 'No promotion' }))).toBe('No promotion');
    expect(getCouponSummary(coupon({ id: 'coupon_without_discount' }))).toBe('coupon_without_discount');
  });
});
