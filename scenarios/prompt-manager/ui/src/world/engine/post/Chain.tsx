import { Bloom, EffectComposer, EffectComposerContext, N8AO } from '@react-three/postprocessing'
import { useThree } from '@react-three/fiber'
import { useContext, useEffect, useRef } from 'react'

/** The slice of n8ao's pass we touch; the package ships no types. */
interface AoPass {
  configuration: { autoDetectTransparency: boolean }
}
import { AgXToneMapping } from 'three'
import type { PostTuning, QualityProfile } from '../../config'
import { passTimerFor } from '../diagnostics/passTimer'

interface PostChainProps {
  profile: QualityProfile
  settings: PostTuning
  diagnosticsEnabled?: boolean
}

function TimedPostPasses() {
  const gl = useThree((state) => state.gl)
  const { composer } = useContext(EffectComposerContext)
  useEffect(() => {
    const timer = passTimerFor(gl, gl.getContext() as WebGL2RenderingContext)
    const restores = composer.passes
      .filter((pass) => pass.constructor.name !== 'RenderPass')
      .map((pass) => {
        const original = pass.render.bind(pass)
        pass.render = function (...args: Parameters<typeof original>) {
          timer.begin('post')
          try {
            return original(...args)
          } finally {
            timer.end('post')
          }
        }
        return () => { pass.render = original }
      })
    return () => restores.forEach((restore) => restore())
  }, [composer, gl])
  return null
}

/**
 * N8AO (half resolution) and selective bloom on a multisampled composer
 * target (profile.msaa; the canvas itself has no antialiasing, so the low
 * profile, which mounts no composer, is the one tier without it). Tone
 * mapping stays on the
 * renderer (AgX) in every profile: materials tone-map in their own shaders,
 * so the HDRI sky and the scene agree, and only `toneMapped=false` emissives
 * (marks, lamps, the campfire) stay above the bloom threshold. The
 * composer's own ToneMapping pass was dropped because it rendered the sky
 * background flat grey.
 */
export function PostChain({ profile, settings, diagnosticsEnabled = false }: PostChainProps) {
  const gl = useThree((s) => s.gl)
  const enabled = profile.ao || profile.bloom
  const ao = useRef<AoPass | null>(null)
  const aoRenderQuality = profile.aoQuality === 'medium' ? 'low' : 'performance'

  // autoDetectTransparency traverses the scene and reads every material getter,
  // which trips troika's lazy Text material; labels are opaque anyway.
  useEffect(() => {
    if (ao.current) ao.current.configuration.autoDetectTransparency = false
  }, [profile.ao])

  useEffect(() => {
    gl.toneMapping = AgXToneMapping
  }, [gl])

  if (!enabled) return null

  return (
    <EffectComposer multisampling={profile.msaa} enableNormalPass={false}>
      {profile.ao ? (
        <N8AO ref={ao as never} halfRes aoRadius={settings.aoRadius} intensity={settings.aoIntensity} distanceFalloff={settings.aoFalloff} quality={aoRenderQuality} />
      ) : <></>}
      {profile.bloom ? (
        <Bloom mipmapBlur luminanceThreshold={settings.bloomThreshold} intensity={settings.bloomIntensity} radius={settings.bloomRadius} />
      ) : <></>}
      {diagnosticsEnabled ? <TimedPostPasses /> : null}
    </EffectComposer>
  )
}
