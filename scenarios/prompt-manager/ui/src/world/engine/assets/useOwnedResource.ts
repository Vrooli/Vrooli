import { useLayoutEffect, useState } from 'react'

/** Allocate only after commit; each effect instance owns its cleanup, including
 * StrictMode replay. A resource from a superseded factory is never returned.
 */
export function useOwnedResource<T extends { dispose(): void }>(factory: () => T | null): T | null {
  const [owned, setOwned] = useState<{ factory: () => T | null; resource: T | null } | null>(null)
  useLayoutEffect(() => {
    const resource = factory()
    setOwned({ factory, resource })
    return () => resource?.dispose()
  }, [factory])
  return owned?.factory === factory ? owned.resource : null
}
