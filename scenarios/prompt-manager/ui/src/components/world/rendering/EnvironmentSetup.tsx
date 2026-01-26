/**
 * EnvironmentSetup - HDR environment and lighting configuration.
 * Provides ambient lighting, reflections, and contact shadows.
 */

import { Environment, ContactShadows } from '@react-three/drei'
import { useResolvedTheme } from '@/hooks/use-theme'
import { useGraphicsStore } from '@/stores/graphicsStore'
import { useEnvironmentStore } from '@/stores/environmentStore'
import { THEME_TO_DREI_PRESET } from '@/config/environments'

interface EnvironmentSetupProps {
  /** Override drei environment preset */
  preset?: Parameters<typeof Environment>[0]['preset']
  /** Override background visibility */
  showBackground?: boolean
}

/**
 * Sets up HDR environment lighting and contact shadows.
 * Automatically syncs with theme when syncWithTheme is enabled.
 */
export function EnvironmentSetup({ preset, showBackground = false }: EnvironmentSetupProps) {
  const theme = useResolvedTheme()
  const config = useGraphicsStore((state) => state.config)
  const dreiPreset = useEnvironmentStore((state) => state.dreiPreset)
  const syncWithTheme = useEnvironmentStore((state) => state.syncWithTheme)

  // Determine which preset to use
  const effectivePreset = preset ?? (syncWithTheme ? THEME_TO_DREI_PRESET[theme] : dreiPreset)

  return (
    <>
      {/* HDR Environment for reflections and ambient lighting */}
      {config.envMap && (
        <Environment
          preset={effectivePreset}
          background={showBackground}
        />
      )}

      {/* Contact shadows for grounded feel */}
      {config.contactShadows && (
        <ContactShadows
          position={[0, 0.01, 0]}
          opacity={0.4}
          scale={20}
          blur={2.5}
          far={4}
          resolution={config.shadowMapSize > 1024 ? 512 : 256}
        />
      )}
    </>
  )
}
