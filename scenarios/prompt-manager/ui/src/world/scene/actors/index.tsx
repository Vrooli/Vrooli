import type { QualityProfile } from '../../config'
import { ActorExtras } from './Extras'
import { Faces } from './Faces'
import { ActorShadows } from './Shadows'
import { Slimes } from './Slimes'

/** The actor layer: contact discs, bodies, faces and extras; four instanced draws for any roster. */
export function Actors({ profile, onSelect, onHover }: { profile: QualityProfile; onSelect?: (id: string | null) => void; onHover?: (id: string | null) => void }) {
  return (
    <group name="actors">
      <ActorShadows />
      <Slimes profile={profile} onSelect={onSelect} onHover={onHover} />
      <Faces />
      <ActorExtras />
    </group>
  )
}
