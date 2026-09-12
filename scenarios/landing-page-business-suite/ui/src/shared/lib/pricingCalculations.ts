/**
 * Shared pricing calculation utilities for admin and landing page.
 * Provides consistent discount calculation and formatting across the app.
 */

import type { StripeCoupon } from '../api/schemas/billing.schema';

/**
 * Result of a discount calculation.
 */
export interface DiscountResult {
  /** Original price in cents */
  originalCents: number;
  /** Discounted price in cents */
  discountedCents: number;
  /** Amount saved in cents */
  savingsCents: number;
  /** Percentage saved (0-100) */
  savingsPercent: number;
  /** Whether a discount was applied */
  hasDiscount: boolean;
}

/**
 * Calculates the discounted price given an original amount and a coupon.
 *
 * @param amountCents - Original price in cents
 * @param coupon - Stripe coupon object (or null if no coupon)
 * @returns Discount calculation result
 */
export function calculateDiscountedPrice(
  amountCents: number,
  coupon: StripeCoupon | null | undefined,
): DiscountResult {
  const result: DiscountResult = {
    originalCents: amountCents,
    discountedCents: amountCents,
    savingsCents: 0,
    savingsPercent: 0,
    hasDiscount: false,
  };

  if (!coupon || !coupon.valid || amountCents <= 0) {
    return result;
  }

  let discountedCents = amountCents;

  if (typeof coupon.percent_off === 'number' && coupon.percent_off > 0) {
    // Percentage discount
    discountedCents = Math.round(amountCents * (1 - coupon.percent_off / 100));
  } else if (typeof coupon.amount_off === 'number' && coupon.amount_off > 0) {
    // Fixed amount discount (already in cents)
    discountedCents = Math.max(0, amountCents - coupon.amount_off);
  }

  const savingsCents = amountCents - discountedCents;
  const savingsPercent = amountCents > 0 ? Math.round((savingsCents / amountCents) * 100) : 0;

  return {
    originalCents: amountCents,
    discountedCents,
    savingsCents,
    savingsPercent,
    hasDiscount: savingsCents > 0,
  };
}

/**
 * Formats a price in cents as a currency string.
 *
 * @param amountCents - Amount in cents
 * @param currency - ISO currency code (default: 'usd')
 * @returns Formatted currency string (e.g., "$79")
 */
export function formatCurrency(amountCents: number, currency = 'usd'): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: currency.toUpperCase(),
    maximumFractionDigits: 0,
  }).format(amountCents / 100);
}

/**
 * Formats a discount badge string based on the coupon.
 *
 * @param amountCents - Original price in cents
 * @param coupon - Stripe coupon object (or null if no coupon)
 * @param interval - Billing interval ('month' | 'year')
 * @returns Badge text (e.g., "First month $1") or undefined if no discount
 */
export function formatDiscountBadge(
  amountCents: number,
  coupon: StripeCoupon | null | undefined,
  interval: 'month' | 'year',
): string | undefined {
  if (!coupon || !coupon.valid || amountCents <= 0) {
    return undefined;
  }

  const result = calculateDiscountedPrice(amountCents, coupon);
  if (!result.hasDiscount) {
    return undefined;
  }

  // Build a descriptive badge based on coupon type and duration
  const discountedPrice = formatCurrency(result.discountedCents, 'usd');
  const durationText = getDurationText(coupon, interval);

  if (coupon.percent_off && coupon.percent_off > 0) {
    // Percentage discount
    if (coupon.duration === 'once') {
      return `First ${interval} ${String(Math.round(coupon.percent_off))}% off`;
    }
    if (coupon.duration === 'forever') {
      return `${String(Math.round(coupon.percent_off))}% off`;
    }
    return `${String(Math.round(coupon.percent_off))}% off ${durationText}`;
  }

  if (coupon.amount_off && coupon.amount_off > 0) {
    // Fixed amount discount
    if (coupon.duration === 'once') {
      return `First ${interval} ${discountedPrice}`;
    }
    return `${discountedPrice} ${durationText}`;
  }

  return undefined;
}

/**
 * Gets duration text for a coupon.
 */
function getDurationText(coupon: StripeCoupon, interval: 'month' | 'year'): string {
  switch (coupon.duration) {
    case 'once':
      return `for 1 ${interval}`;
    case 'forever':
      return 'forever';
    case 'repeating':
      if (coupon.duration_in_months && coupon.duration_in_months > 0) {
        const months = coupon.duration_in_months;
        if (interval === 'year') {
          const years = Math.floor(months / 12);
          return years > 1 ? `for ${String(years)} years` : 'for 1 year';
        }
        return months > 1 ? `for ${String(months)} months` : 'for 1 month';
      }
      return '';
    default:
      return '';
  }
}

/**
 * Formats a brief discount preview for admin UI.
 * Shows the discounted price and savings.
 *
 * @param amountCents - Original price in cents
 * @param coupon - Stripe coupon object (or null if no coupon)
 * @returns Preview text (e.g., "$1 (save $78)") or undefined if no discount
 */
export function formatDiscountPreview(
  amountCents: number,
  coupon: StripeCoupon | null | undefined,
): string | undefined {
  if (!coupon || !coupon.valid || amountCents <= 0) {
    return undefined;
  }

  const result = calculateDiscountedPrice(amountCents, coupon);
  if (!result.hasDiscount) {
    return undefined;
  }

  const discountedPrice = formatCurrency(result.discountedCents, 'usd');
  const savings = formatCurrency(result.savingsCents, 'usd');

  return `${discountedPrice} (save ${savings})`;
}

/**
 * Gets a coupon summary for display.
 *
 * @param coupon - Stripe coupon object
 * @returns Summary text (e.g., "50% off once" or "$10 off forever")
 */
export function getCouponSummary(coupon: StripeCoupon): string {
  let discountText = '';

  if (typeof coupon.percent_off === 'number' && coupon.percent_off > 0) {
    discountText = `${String(Math.round(coupon.percent_off))}% off`;
  } else if (typeof coupon.amount_off === 'number' && coupon.amount_off > 0) {
    discountText = `${formatCurrency(coupon.amount_off, coupon.currency ?? 'usd')} off`;
  } else {
    return coupon.name || coupon.id;
  }

  switch (coupon.duration) {
    case 'once':
      return `${discountText} (first payment)`;
    case 'forever':
      return `${discountText} (forever)`;
    case 'repeating':
      if (coupon.duration_in_months && coupon.duration_in_months > 0) {
        return `${discountText} (${String(coupon.duration_in_months)} months)`;
      }
      return discountText;
    default:
      return discountText;
  }
}
