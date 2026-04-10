import { describe, it, expect } from 'vitest';
import { isFormDirty, isFormDirtyNormalized } from './formUtils';

describe('formUtils', () => {
  describe('isFormDirty', () => {
    it('returns false when objects are identical', () => {
      const current = { name: 'test', value: 123 };
      const original = { name: 'test', value: 123 };
      expect(isFormDirty(current, original)).toBe(false);
    });

    it('returns true when string field differs', () => {
      const current = { name: 'changed' };
      const original = { name: 'original' };
      expect(isFormDirty(current, original)).toBe(true);
    });

    it('returns true when number field differs', () => {
      const current = { count: 10 };
      const original = { count: 5 };
      expect(isFormDirty(current, original)).toBe(true);
    });

    it('returns true when boolean field differs', () => {
      const current = { enabled: true };
      const original = { enabled: false };
      expect(isFormDirty(current, original)).toBe(true);
    });

    it('handles nested objects', () => {
      const current = { nested: { value: 'changed' } };
      const original = { nested: { value: 'original' } };
      expect(isFormDirty(current, original)).toBe(true);
    });

    it('returns false for nested objects that match', () => {
      const current = { nested: { value: 'same' } };
      const original = { nested: { value: 'same' } };
      expect(isFormDirty(current, original)).toBe(false);
    });

    it('handles arrays', () => {
      const current = { items: [1, 2, 3] };
      const original = { items: [1, 2] };
      expect(isFormDirty(current, original)).toBe(true);
    });

    it('returns false for arrays that match', () => {
      const current = { items: ['a', 'b'] };
      const original = { items: ['a', 'b'] };
      expect(isFormDirty(current, original)).toBe(false);
    });

    it('handles null values', () => {
      const current = { value: null as string | null };
      const original = { value: 'test' as string | null };
      expect(isFormDirty(current, original)).toBe(true);
    });

    it('returns false when both are null', () => {
      const current = { value: null };
      const original = { value: null };
      expect(isFormDirty(current, original)).toBe(false);
    });

    it('handles undefined values', () => {
      const current = { value: undefined as string | undefined };
      const original = { value: 'test' as string | undefined };
      expect(isFormDirty(current, original)).toBe(true);
    });

    it('works with empty objects', () => {
      expect(isFormDirty({}, {})).toBe(false);
    });

    it('detects field addition', () => {
      const current = { name: 'test', extra: 'field' };
      const original = { name: 'test' };
      expect(isFormDirty(current, original)).toBe(true);
    });

    it('handles complex form state', () => {
      const current = {
        name: 'My Site',
        tagline: 'Welcome',
        enabled: true,
        settings: {
          theme: 'dark',
          notifications: ['email', 'sms'],
        },
      };
      const original = {
        name: 'My Site',
        tagline: 'Welcome',
        enabled: true,
        settings: {
          theme: 'dark',
          notifications: ['email', 'sms'],
        },
      };
      expect(isFormDirty(current, original)).toBe(false);
    });
  });

  describe('isFormDirtyNormalized', () => {
    const trimNormalizer = (form: { name: string }) => ({
      name: form.name.trim(),
    });

    it('returns false when normalized values match', () => {
      const current = { name: '  test  ' };
      const original = { name: 'test' };
      expect(isFormDirtyNormalized(current, original, trimNormalizer)).toBe(false);
    });

    it('returns true when normalized values differ', () => {
      const current = { name: '  changed  ' };
      const original = { name: 'original' };
      expect(isFormDirtyNormalized(current, original, trimNormalizer)).toBe(true);
    });

    it('applies normalizer to both forms', () => {
      const current = { name: '  same  ' };
      const original = { name: '  same  ' };
      expect(isFormDirtyNormalized(current, original, trimNormalizer)).toBe(false);
    });

    it('works with complex normalizers', () => {
      interface ComplexForm {
        items: string[];
        count: number;
      }

      const normalizer = (form: ComplexForm) => ({
        items: form.items.map((s) => s.toLowerCase().trim()).sort(),
        count: form.count,
      });

      const current = { items: ['  B  ', 'A'], count: 5 };
      const original = { items: ['a', 'b'], count: 5 };
      expect(isFormDirtyNormalized(current, original, normalizer)).toBe(false);
    });

    it('detects changes after normalization', () => {
      interface ComplexForm {
        items: string[];
        count: number;
      }

      const normalizer = (form: ComplexForm) => ({
        items: form.items.map((s) => s.toLowerCase().trim()).sort(),
        count: form.count,
      });

      const current = { items: ['a', 'b'], count: 10 };
      const original = { items: ['a', 'b'], count: 5 };
      expect(isFormDirtyNormalized(current, original, normalizer)).toBe(true);
    });

    it('supports different input and output types', () => {
      interface FormInput {
        price: string;
        quantity: string;
      }

      interface FormOutput {
        price: number;
        quantity: number;
      }

      const normalizer = (form: FormInput): FormOutput => ({
        price: parseFloat(form.price) || 0,
        quantity: parseInt(form.quantity, 10) || 0,
      });

      const current = { price: '10.50', quantity: '5' };
      const original = { price: '10.5', quantity: '5' };
      expect(isFormDirtyNormalized(current, original, normalizer)).toBe(false);
    });
  });
});
