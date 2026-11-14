import { vi, type Mock } from 'vitest'

export interface StorageMock extends Storage {
  getItem: Mock<[key: string], string | null>
  setItem: Mock<[key: string, value: string], void>
  removeItem: Mock<[key: string], void>
  clear: Mock<[], void>
}

export function createStorageMock(initialValues: Record<string, string> = {}): StorageMock {
  const data = new Map<string, string>(Object.entries(initialValues))

  const storage = {
    get length() {
      return data.size
    },
    clear: vi.fn(() => {
      data.clear()
    }),
    getItem: vi.fn((key: string) => data.get(key) ?? null),
    key: vi.fn((index: number) => Array.from(data.keys())[index] ?? null),
    removeItem: vi.fn((key: string) => {
      data.delete(key)
    }),
    setItem: vi.fn((key: string, value: string) => {
      data.set(key, value)
    }),
  }

  return storage as StorageMock
}

export function installStorageMock(
  target: Pick<typeof globalThis, 'localStorage'> = window,
  initialValues: Record<string, string> = {}
): StorageMock {
  const storage = createStorageMock(initialValues)
  Object.defineProperty(target, 'localStorage', {
    configurable: true,
    writable: true,
    value: storage,
  })
  return storage
}

export function resetStorageMock(storage: StorageMock = window.localStorage as StorageMock) {
  storage.clear()
  storage.getItem.mockClear()
  storage.setItem.mockClear()
  storage.removeItem.mockClear()
  storage.clear.mockClear()
}
