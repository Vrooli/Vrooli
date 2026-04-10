import { describe, it, expect, vi, afterEach } from 'vitest';
import { z } from 'zod';
import {
  safeParse,
  parseOrThrow,
  parseOrNull,
  parseArrayFiltered,
  ValidationError,
  toValidationError,
  type ParseResult,
  type ParseFailure,
} from './safeParse';

describe('safeParse utilities', () => {
  // Mock console.error to avoid noise in test output
  const consoleErrorSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

  afterEach(() => {
    consoleErrorSpy.mockClear();
  });

  describe('safeParse', () => {
    const UserSchema = z.object({
      id: z.string(),
      name: z.string(),
      email: z.string().email(),
      age: z.number().optional(),
    });

    describe('successful parsing', () => {
      it('returns success with validated data', () => {
        const data = { id: '123', name: 'John', email: 'john@example.com' };
        const result = safeParse(UserSchema, data, 'User');

        expect(result.success).toBe(true);
        if (result.success) {
          expect(result.data).toEqual(data);
        }
      });

      it('includes optional fields when present', () => {
        const data = { id: '123', name: 'John', email: 'john@example.com', age: 30 };
        const result = safeParse(UserSchema, data, 'User');

        expect(result.success).toBe(true);
        if (result.success) {
          expect(result.data.age).toBe(30);
        }
      });

      it('strips extra fields not in schema', () => {
        const data = { id: '123', name: 'John', email: 'john@example.com', extra: 'ignored' };
        const result = safeParse(UserSchema, data, 'User');

        expect(result.success).toBe(true);
        if (result.success) {
          expect('extra' in result.data).toBe(false);
        }
      });

      it('does not log on success', () => {
        const data = { id: '123', name: 'John', email: 'john@example.com' };
        safeParse(UserSchema, data, 'User');

        expect(consoleErrorSpy).not.toHaveBeenCalled();
      });
    });

    describe('failed parsing', () => {
      it('returns failure with error message', () => {
        const data = { id: '123', name: 'John' }; // Missing email
        const result = safeParse(UserSchema, data, 'User');

        expect(result.success).toBe(false);
        if (!result.success) {
          expect(result.error).toContain('Invalid User response');
          expect(result.error).toContain('email');
        }
      });

      it('includes ZodError details', () => {
        const data = { id: '123', name: 'John' };
        const result = safeParse(UserSchema, data, 'User');

        expect(result.success).toBe(false);
        if (!result.success) {
          expect(result.details).toBeInstanceOf(z.ZodError);
          expect(result.details.errors).toHaveLength(1);
        }
      });

      it('includes raw data in failure', () => {
        const data = { id: '123', name: 'John' };
        const result = safeParse(UserSchema, data, 'User');

        expect(result.success).toBe(false);
        if (!result.success) {
          expect(result.raw).toEqual(data);
        }
      });

      it('logs error to console', () => {
        const data = { id: '123', name: 'John' };
        safeParse(UserSchema, data, 'User');

        expect(consoleErrorSpy).toHaveBeenCalledWith(
          '[API Validation] User:',
          expect.anything()
        );
      });

      it('handles multiple validation errors', () => {
        const data = { id: 123, name: null }; // Wrong types
        const result = safeParse(UserSchema, data, 'User');

        expect(result.success).toBe(false);
        if (!result.success) {
          expect(result.error).toContain('id');
          expect(result.error).toContain('name');
          expect(result.error).toContain('email');
        }
      });

      it('handles invalid email format', () => {
        const data = { id: '123', name: 'John', email: 'not-an-email' };
        const result = safeParse(UserSchema, data, 'User');

        expect(result.success).toBe(false);
        if (!result.success) {
          expect(result.error).toContain('email');
        }
      });

      it('formats nested path correctly', () => {
        const NestedSchema = z.object({
          user: z.object({
            profile: z.object({
              bio: z.string(),
            }),
          }),
        });

        const data = { user: { profile: { bio: 123 } } };
        const result = safeParse(NestedSchema, data, 'Nested');

        expect(result.success).toBe(false);
        if (!result.success) {
          expect(result.error).toContain('user.profile.bio');
        }
      });
    });

    describe('edge cases', () => {
      it('handles null input', () => {
        const result = safeParse(UserSchema, null, 'User');
        expect(result.success).toBe(false);
      });

      it('handles undefined input', () => {
        const result = safeParse(UserSchema, undefined, 'User');
        expect(result.success).toBe(false);
      });

      it('handles empty object', () => {
        const result = safeParse(UserSchema, {}, 'User');
        expect(result.success).toBe(false);
      });

      it('handles array input when object expected', () => {
        const result = safeParse(UserSchema, [], 'User');
        expect(result.success).toBe(false);
      });

      it('handles primitive input when object expected', () => {
        const result = safeParse(UserSchema, 'string', 'User');
        expect(result.success).toBe(false);
      });
    });
  });

  describe('parseOrThrow', () => {
    const NumberSchema = z.number();

    it('returns validated data on success', () => {
      const result = parseOrThrow(NumberSchema, 42, 'Number');
      expect(result).toBe(42);
    });

    it('throws error on validation failure', () => {
      expect(() => parseOrThrow(NumberSchema, 'not a number', 'Number')).toThrow(
        'Invalid Number response'
      );
    });

    it('throws with descriptive message', () => {
      expect(() => parseOrThrow(NumberSchema, null, 'Number')).toThrow(/root/);
    });
  });

  describe('parseOrNull', () => {
    const StringSchema = z.string();

    it('returns validated data on success', () => {
      const result = parseOrNull(StringSchema, 'hello', 'String');
      expect(result).toBe('hello');
    });

    it('returns null on validation failure', () => {
      const result = parseOrNull(StringSchema, 123, 'String');
      expect(result).toBeNull();
    });

    it('logs error on failure', () => {
      parseOrNull(StringSchema, 123, 'String');
      expect(consoleErrorSpy).toHaveBeenCalled();
    });
  });

  describe('parseArrayFiltered', () => {
    const ItemSchema = z.object({
      id: z.number(),
      name: z.string(),
    });

    it('returns all valid items', () => {
      const data = [
        { id: 1, name: 'One' },
        { id: 2, name: 'Two' },
        { id: 3, name: 'Three' },
      ];
      const result = parseArrayFiltered(ItemSchema, data, 'Item');

      expect(result).toHaveLength(3);
      expect(result).toEqual(data);
    });

    it('filters out invalid items', () => {
      const data = [
        { id: 1, name: 'One' },
        { id: 'invalid', name: 'Two' }, // Invalid id
        { id: 3, name: 'Three' },
      ];
      const result = parseArrayFiltered(ItemSchema, data, 'Item');

      expect(result).toHaveLength(2);
      expect(result[0]?.id).toBe(1);
      expect(result[1]?.id).toBe(3);
    });

    it('returns empty array when all items invalid', () => {
      const data = [
        { id: 'a' },
        { name: 'only name' },
        {},
      ];
      const result = parseArrayFiltered(ItemSchema, data, 'Item');

      expect(result).toHaveLength(0);
    });

    it('logs each invalid item with index', () => {
      const data = [
        { id: 1, name: 'Valid' },
        { id: 'bad' }, // Invalid at index 1
      ];
      parseArrayFiltered(ItemSchema, data, 'Item');

      expect(consoleErrorSpy).toHaveBeenCalledWith(
        '[API Validation] Item[1]:',
        expect.anything()
      );
    });

    it('handles empty array', () => {
      const result = parseArrayFiltered(ItemSchema, [], 'Item');
      expect(result).toEqual([]);
    });

    it('preserves order of valid items', () => {
      const data = [
        { id: 1, name: 'First' },
        { id: 'skip' },
        { id: 2, name: 'Second' },
        { name: 'skip' },
        { id: 3, name: 'Third' },
      ];
      const result = parseArrayFiltered(ItemSchema, data, 'Item');

      expect(result).toHaveLength(3);
      expect(result.map(r => r.id)).toEqual([1, 2, 3]);
    });
  });

  describe('ValidationError', () => {
    it('extends Error', () => {
      const error = new ValidationError('Test message', 'Context');
      expect(error).toBeInstanceOf(Error);
    });

    it('has correct name', () => {
      const error = new ValidationError('Test message', 'Context');
      expect(error.name).toBe('ValidationError');
    });

    it('stores context', () => {
      const error = new ValidationError('Test message', 'MyContext');
      expect(error.context).toBe('MyContext');
    });

    it('stores optional details', () => {
      const zodError = new z.ZodError([]);
      const error = new ValidationError('Test', 'Context', zodError);
      expect(error.details).toBe(zodError);
    });

    it('stores optional raw data', () => {
      const raw = { foo: 'bar' };
      const error = new ValidationError('Test', 'Context', undefined, raw);
      expect(error.raw).toBe(raw);
    });
  });

  describe('toValidationError', () => {
    it('creates ValidationError from ParseFailure', () => {
      const failure: ParseFailure = {
        success: false,
        error: 'Validation failed',
        details: new z.ZodError([]),
        raw: { some: 'data' },
      };

      const error = toValidationError(failure, 'TestContext');

      expect(error).toBeInstanceOf(ValidationError);
      expect(error.message).toBe('Validation failed');
      expect(error.context).toBe('TestContext');
      expect(error.details).toBe(failure.details);
      expect(error.raw).toEqual({ some: 'data' });
    });
  });

  describe('type narrowing', () => {
    const TestSchema = z.object({ value: z.string() });

    it('narrows to success type correctly', () => {
      const result: ParseResult<{ value: string }> = safeParse(
        TestSchema,
        { value: 'test' },
        'Test'
      );

      if (result.success) {
        // TypeScript should know result.data exists
        const value: string = result.data.value;
        expect(value).toBe('test');
      }
    });

    it('narrows to failure type correctly', () => {
      const result: ParseResult<{ value: string }> = safeParse(
        TestSchema,
        { value: 123 },
        'Test'
      );

      if (!result.success) {
        // TypeScript should know these exist
        const error: string = result.error;
        const raw: unknown = result.raw;
        expect(error).toContain('Invalid');
        expect(raw).toEqual({ value: 123 });
      }
    });
  });
});
