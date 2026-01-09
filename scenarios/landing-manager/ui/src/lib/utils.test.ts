import { describe, it, expect } from 'vitest';
import { cn, slugify } from './utils';

describe('cn utility', () => {
  it('should merge class names correctly', () => {
    const result = cn('text-sm', 'font-bold');
    expect(result).toContain('text-sm');
    expect(result).toContain('font-bold');
  });

  it('should handle conditional classes', () => {
    const isActive = true;
    const result = cn('base-class', isActive && 'active-class');
    expect(result).toContain('base-class');
    expect(result).toContain('active-class');
  });

  it('should filter out false/null/undefined values', () => {
    const result = cn('valid', false, null, undefined, 'also-valid');
    expect(result).toContain('valid');
    expect(result).toContain('also-valid');
    expect(result).not.toContain('false');
    expect(result).not.toContain('null');
  });

  it('should handle empty input', () => {
    const result = cn();
    expect(result).toBe('');
  });

  it('should merge Tailwind classes with conflicts', () => {
    // The clsx + tailwind-merge combination should handle conflicts
    const result = cn('px-2', 'px-4');
    // Should keep only px-4 (later class wins)
    expect(result).toBe('px-4');
  });
});

describe('slugify utility', () => {
  it('should convert to lowercase', () => {
    expect(slugify('Hello World')).toBe('hello-world');
  });

  it('should replace spaces with hyphens', () => {
    expect(slugify('My New Product')).toBe('my-new-product');
  });

  it('should remove special characters', () => {
    expect(slugify('Product @#$ Name!')).toBe('product-name');
  });

  it('should remove leading and trailing hyphens', () => {
    expect(slugify('--product-name--')).toBe('product-name');
  });

  it('should collapse multiple hyphens', () => {
    expect(slugify('product   ---   name')).toBe('product-name');
  });

  it('should truncate to 60 characters', () => {
    const longInput = 'a'.repeat(100);
    const result = slugify(longInput);
    expect(result.length).toBe(60);
  });

  it('should handle empty string', () => {
    expect(slugify('')).toBe('');
  });

  it('should handle numbers', () => {
    expect(slugify('Product 123')).toBe('product-123');
  });
});
