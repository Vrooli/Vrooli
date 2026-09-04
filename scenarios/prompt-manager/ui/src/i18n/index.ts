import messages from './locales/en.json'

type Options = { defaultValue?: string }

function lookup(key: string): string | undefined {
  let value: unknown = messages
  for (const segment of key.split('.')) {
    if (!value || typeof value !== 'object' || !(segment in value)) return undefined
    value = (value as Record<string, unknown>)[segment]
  }
  return typeof value === 'string' ? value : undefined
}

export const i18n = {
  t(key: string, options: Options = {}): string {
    return lookup(key) ?? options.defaultValue ?? key
  },
}
