import { describe, it, expect, vi } from 'vitest'
import { z } from 'zod'
import {
  safeParse,
  parseOrThrow,
  parseOrNull,
  parseArrayFiltered,
  ValidationError,
} from '../safeParse'

const TestSchema = z.object({
  id: z.string(),
  name: z.string(),
  count: z.number(),
})

describe('safeParse', () => {
  it('should return success with valid data', () => {
    const data = { id: '1', name: 'Test', count: 5 }
    const result = safeParse(TestSchema, data, 'test-context')

    expect(result.success).toBe(true)
    if (result.success) {
      expect(result.data).toEqual(data)
    }
  })

  it('should return error with invalid data', () => {
    const data = { id: '1', name: 'Test' } // missing count
    const result = safeParse(TestSchema, data, 'test-context')

    expect(result.success).toBe(false)
    if (!result.success) {
      expect(result.error).toBeInstanceOf(ValidationError)
      expect(result.error.context).toBe('test-context')
      expect(result.error.issues.length).toBeGreaterThan(0)
    }
  })

  it('should include field path in error messages', () => {
    const data = { id: 123, name: 'Test', count: 5 } // id should be string
    const result = safeParse(TestSchema, data, '/api/test')

    expect(result.success).toBe(false)
    if (!result.success) {
      expect(result.error.issues.some((i) => i.includes('id'))).toBe(true)
    }
  })
})

describe('parseOrThrow', () => {
  it('should return data for valid input', () => {
    const data = { id: '1', name: 'Test', count: 5 }
    const result = parseOrThrow(TestSchema, data, 'test')

    expect(result).toEqual(data)
  })

  it('should throw ValidationError for invalid input', () => {
    const data = { id: '1' } // missing required fields

    expect(() => parseOrThrow(TestSchema, data, '/api/test')).toThrow(
      ValidationError
    )
  })

  it('should include context in thrown error', () => {
    const data = { id: '1' }

    try {
      parseOrThrow(TestSchema, data, '/api/items/123')
      expect.fail('Should have thrown')
    } catch (error) {
      expect(error).toBeInstanceOf(ValidationError)
      if (error instanceof ValidationError) {
        expect(error.context).toBe('/api/items/123')
        expect(error.message).toContain('/api/items/123')
      }
    }
  })
})

describe('parseOrNull', () => {
  it('should return data for valid input', () => {
    const data = { id: '1', name: 'Test', count: 5 }
    const result = parseOrNull(TestSchema, data, 'test')

    expect(result).toEqual(data)
  })

  it('should return null for invalid input', () => {
    const data = { id: '1' }
    const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    const result = parseOrNull(TestSchema, data, 'test')

    expect(result).toBeNull()
    expect(consoleSpy).toHaveBeenCalled()

    consoleSpy.mockRestore()
  })

  it('should log warning with context', () => {
    const data = { invalid: 'data' }
    const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    parseOrNull(TestSchema, data, '/cache/item')

    // parseOrNull logs a single string message that includes the context
    expect(consoleSpy).toHaveBeenCalledWith(
      expect.stringContaining('[parseOrNull]')
    )
    expect(consoleSpy).toHaveBeenCalledWith(
      expect.stringContaining('/cache/item')
    )

    consoleSpy.mockRestore()
  })
})

describe('parseArrayFiltered', () => {
  it('should return all valid items', () => {
    const data = [
      { id: '1', name: 'One', count: 1 },
      { id: '2', name: 'Two', count: 2 },
      { id: '3', name: 'Three', count: 3 },
    ]

    const result = parseArrayFiltered(TestSchema, data, '/api/items')

    expect(result).toHaveLength(3)
    expect(result[0]?.id).toBe('1')
    expect(result[2]?.id).toBe('3')
  })

  it('should filter out invalid items', () => {
    const data = [
      { id: '1', name: 'One', count: 1 },
      { id: '2', name: 'Two' }, // missing count
      { id: '3', name: 'Three', count: 3 },
    ]
    const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    const result = parseArrayFiltered(TestSchema, data, '/api/items')

    expect(result).toHaveLength(2)
    expect(result[0]?.id).toBe('1')
    expect(result[1]?.id).toBe('3')

    consoleSpy.mockRestore()
  })

  it('should log warnings for filtered items', () => {
    const data = [
      { id: '1', name: 'One', count: 1 },
      { invalid: true }, // completely invalid
      { id: '3' }, // missing fields
    ]
    const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    parseArrayFiltered(TestSchema, data, '/api/items')

    // Should log for each invalid item plus summary
    expect(consoleSpy).toHaveBeenCalledTimes(3) // 2 invalid items + 1 summary

    consoleSpy.mockRestore()
  })

  it('should return empty array for empty input', () => {
    const result = parseArrayFiltered(TestSchema, [], '/api/items')
    expect(result).toEqual([])
  })

  it('should return empty array when all items are invalid', () => {
    const data = [{ bad: 'data' }, { also: 'bad' }]
    const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})

    const result = parseArrayFiltered(TestSchema, data, '/api/items')

    expect(result).toEqual([])

    consoleSpy.mockRestore()
  })
})

describe('ValidationError', () => {
  it('should have correct name', () => {
    // Trigger a validation error to get a real ZodError
    const result = TestSchema.safeParse({})
    if (!result.success) {
      const error = new ValidationError(result.error, 'test')
      expect(error.name).toBe('ValidationError')
    }
  })

  it('should be instance of Error', () => {
    const result = TestSchema.safeParse({})
    if (!result.success) {
      const error = new ValidationError(result.error, 'test')
      expect(error).toBeInstanceOf(Error)
      expect(error).toBeInstanceOf(ValidationError)
    }
  })

  it('should format issues from ZodError', () => {
    // Parse invalid data to get real ZodError with issues
    const result = TestSchema.safeParse({ id: 123, name: 'Test' }) // id wrong type, missing count

    expect(result.success).toBe(false)
    if (!result.success) {
      const error = new ValidationError(result.error, '/api/test')

      expect(error.issues.length).toBeGreaterThan(0)
      expect(error.context).toBe('/api/test')
      expect(error.originalError).toBe(result.error)
      expect(error.message).toContain('/api/test')
    }
  })

  it('should include field paths in issues', () => {
    const result = TestSchema.safeParse({ id: 123 }) // wrong type and missing fields

    expect(result.success).toBe(false)
    if (!result.success) {
      const error = new ValidationError(result.error, 'test')
      // Should have paths in at least some issues
      const hasPathInfo = error.issues.some((issue) => issue.includes(':'))
      expect(hasPathInfo).toBe(true)
    }
  })
})
