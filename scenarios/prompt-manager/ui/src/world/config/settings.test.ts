import { describe, expect, it } from 'vitest'
import { numericOverrides, parseNumericOverride, choiceSettings, parseChoiceSetting, type ChoiceSetting, integerSettings, parseIntegerSetting } from './settings'

describe.each(Object.values(integerSettings))('$id integer setting', descriptor => {
  it('accepts both inclusive boundaries and defaults only when absent', () => {
    for (const value of [descriptor.minimum, descriptor.maximum]) expect(parseIntegerSetting(descriptor, String(value))).toEqual({ value, error: null })
    expect(parseIntegerSetting(descriptor, null)).toEqual({ value: descriptor.defaultValue, error: null })
  })
  it.each(['12junk', '1.5', '1e3', '0xff', '', ' ', '-1', 'Infinity', 'NaN', '9007199254740993'])('rejects malformed or unsafe %j without partial parsing', raw => {
    expect(parseIntegerSetting(descriptor, raw)).toMatchObject({ value: descriptor.defaultValue, error: expect.any(String) })
  })
  it('rejects values above its declared range', () => {
    expect(parseIntegerSetting(descriptor, String(descriptor.maximum + 1)).error).not.toBeNull()
  })
})

for (const descriptor of Object.values(choiceSettings)) {
  const setting: ChoiceSetting<string> = descriptor
  describe(`${setting.id} choice setting`, () => {
    it.each(setting.choices)('accepts the displayed $id choice', choice => {
      expect(parseChoiceSetting(setting, choice.id)).toEqual({ value: choice.id, error: null })
      expect(choice.label.trim()).not.toBe('')
    })
    it('has unique choices including its default', () => {
      expect(new Set(setting.choices.map(choice => choice.id)).size).toBe(setting.choices.length)
      expect(setting.choices.some(choice => choice.id === setting.defaultValue)).toBe(true)
      expect(parseChoiceSetting(setting, null)).toEqual({ value: setting.defaultValue, error: null })
    })
    it.each(['', 'unknown', ' DAY ', 'RAIN'])('rejects %j with an explicit fallback', raw => {
      expect(parseChoiceSetting(setting, raw)).toEqual({ value: setting.defaultValue, error: expect.any(String) })
    })
  })
}

for (const descriptor of Object.values(numericOverrides)) {
  describe(`${descriptor.id} numeric override`, () => {
    it('uses active defaults for missing and invalid values', () => {
      expect(parseNumericOverride(descriptor, null, null)).toEqual({ value: null, error: null })
      expect(parseNumericOverride(descriptor, 'bad', .75)).toEqual({ value: .75, error: expect.any(String) })
    })
    it.each(['', ' ', '1junk', '1e2', '0xff', 'NaN', 'Infinity', '-1'])('rejects malformed %j without coercion', raw => {
      expect(parseNumericOverride(descriptor, raw, null)).toEqual({ value: null, error: expect.any(String) })
    })
    it('accepts schema boundaries and rejects values beyond them', () => {
      const minimum = descriptor.schema.minValue, maximum = descriptor.schema.maxValue
      if (minimum === null || maximum === null) throw new Error('Numeric override must be bounded')
      for (const value of [minimum, maximum]) expect(parseNumericOverride(descriptor, String(value), null)).toEqual({ value, error: null })
      expect(parseNumericOverride(descriptor, String(maximum + 1), null).value).toBeNull()
    })
    it('keeps fractional values only for decimal settings', () => {
      const result = parseNumericOverride(descriptor, '.75', null)
      expect(result.value).toBe((descriptor.schema.format === 'safeint') ? null : .75)
    })
  })
}
