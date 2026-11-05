"""Bounded, file-only product eligibility and comprehension evidence checks.

Declarations preserve provenance; they cannot establish that a source is true.
"""
import hashlib
import json
from pathlib import Path

def digest(path):
    with Path(path).open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def qualifies(source, context, benefit, scope):
    witness = source.get('qualification', {})
    if not isinstance(witness, dict):
        return False
    targets, benefits = witness.get('targets'), witness.get('benefit_ids')
    return (source.get('kind') in ('runtime', 'package')
            and (scope != 'package' or source.get('kind') == 'package')
            and witness.get('status') == 'observed' and witness.get('scope') == scope
            and witness.get('product_revision') == context.get('product_revision')
            and isinstance(targets, list) and all(isinstance(t, str) for t in targets)
            and set(context.get('targets', [])) <= set(targets)
            and isinstance(benefits, list) and benefit in benefits)


def require_current_brief(plan, root, historical=False):
    if historical:
        return []
    if plan.get('version') != 3:
        return ['new productions require version 3; use --historical only for retained productions']
    try:
        ref = plan['product_brief']
        path = root / ref['file']
        if path.is_symlink() or not path.resolve().is_relative_to(root.resolve()) or path.stat().st_size > 2 * 1024**2:
            return ['invalid product brief reference']
        if json.loads(path.read_text()).get('version') != 2:
            return ['new productions require product brief version 2; use --historical only for retained productions']
    except (OSError, ValueError, KeyError, TypeError, AttributeError):
        return ['current product brief could not be read']
    return []

def validate_product_brief(plan, root, stage):
    """Check offer/evidence joins. Does not grade benefit quality or verify declarations."""
    errors = []
    def need(condition, message):
        if not condition:
            errors.append(message)
    def text(value):
        return isinstance(value, str) and bool(value.strip())
    def names(value):
        return isinstance(value, list) and all(text(v) for v in value) and len(value) == len(set(value))
    def local_file(value, label):
        if not text(value):
            errors.append(f'{label}: file reference required')
            return None
        relative = Path(value)
        path = root / relative
        if (relative.is_absolute() or '..' in relative.parts or path.is_symlink()
                or not path.resolve().is_relative_to(root.resolve()) or not path.is_file()):
            errors.append(f'{label}: expected a regular relative file')
            return None
        if not 0 < path.stat().st_size <= 2 * 1024**2:
            errors.append(f'{label}: evidence file must be nonempty and at most 2 MiB')
            return None
        return path

    ref = plan.get('product_brief', {})
    if not isinstance(ref, dict):
        return ['product brief: object reference required']
    path = local_file(ref.get('file'), 'product brief')
    if path is None:
        return errors
    need(digest(path) == ref.get('sha256'), 'product brief digest mismatch')
    try:
        brief = json.loads(path.read_text())
        need(brief.get('version') in (1, 2), 'product brief version must be 1 or 2')
        need(brief.get('product') == plan.get('product'), 'product brief belongs to another product')
        context = plan.get('context', {})
        for key in ('offer_ref', 'offer_revision', 'product_revision'):
            need(text(context.get(key)), f'context.{key} required')
        targets = context.get('targets', [])
        need(names(targets) and bool(targets), 'context.targets must name product deployment targets')
        need(brief.get('context') == context, 'brief context differs; reassess offer/target/build eligibility')
        inputs = plan.get('render_inputs', [])
        need(ref.get('file') in inputs, 'product brief is not a declared render input')
        sources = brief.get('sources', [])
        need(isinstance(sources, list) and 1 <= len(sources) <= 128, 'brief requires 1–128 sources')
        by_source = {s['id']: s for s in sources}
        need(len(by_source) == len(sources), 'brief source IDs must be unique')
        for source in sources:
            label = f'source {source["id"]}'
            need(text(source.get('id')) and text(source.get('kind')) and text(source.get('read_at')),
                 f'{label}: kind and read_at required')
            asset = local_file(source.get('file'), label)
            if asset:
                need(digest(asset) == source.get('sha256'), f'{label} digest mismatch')
            need(source.get('file') in inputs, f'{label} is not a declared render input')
        candidates = brief.get('candidates', [])
        need(isinstance(candidates, list) and 1 <= len(candidates) <= 128, 'brief requires 1–128 candidate benefits')
        by_benefit = {b['id']: b for b in candidates}
        need(len(by_benefit) == len(candidates), 'benefit IDs must be unique')
        for benefit in candidates:
            label = f'benefit {benefit["id"]}'
            for key in ('id', 'job', 'alternative', 'pain', 'benefit', 'reliability',
                        'setup', 'dependencies', 'reason'):
                need(text(benefit.get(key)), f'{label}: {key} required')
            need(benefit.get('value_basis') in ('observed', 'operator', 'inferred'), f'{label}: invalid value basis')
            need(benefit.get('eligibility') in ('supported', 'conditional', 'excluded', 'unknown'),
                 f'{label}: invalid eligibility')
            for key in ('evidence', 'eligibility_sources'):
                ids = benefit.get(key, [])
                need(names(ids) and bool(ids) and all(i in by_source for i in ids), f'{label}: {key} must reference sources')
        selected = brief.get('selected', [])
        need(names(selected) and (bool(selected) or plan.get('kind') == 'study') and all(i in by_benefit for i in selected),
             'brief selected benefits must reference candidates')
        need(text(brief.get('selection_rationale')), 'benefit selection rationale required')
        story_ids = plan.get('story', {}).get('benefits', [])
        need(names(story_ids) and set(story_ids) == set(selected), 'story benefits must match selected benefits')
        by_claim = {c['id']: c for c in plan.get('claims', [])}
        for scene in plan.get('scenes', []):
            scene_benefits = scene.get('benefits', [])
            need(names(scene_benefits) and all(i in selected for i in scene_benefits),
                 f'scene {scene.get("id")}: unknown or unselected benefit')
        for identifier in selected:
            benefit = by_benefit.get(identifier)
            if benefit is None:
                continue
            need(benefit.get('eligibility') in ('supported', 'conditional'),
                 f'selected benefit {identifier} is not eligible for this offer')
            product_evidence = [by_source[i].get('kind') for i in benefit.get('evidence', []) if i in by_source]
            eligibility_evidence = [by_source[i].get('kind') for i in benefit.get('eligibility_sources', []) if i in by_source]
            need('product' in product_evidence, f'benefit {identifier}: product evidence required')
            need(any(k in ('package', 'offer', 'operator-direction') for k in eligibility_evidence),
                 f'benefit {identifier}: offer/package eligibility evidence required')
            if brief.get('version') == 2:
                availability = benefit.get('availability_claim')
                need(availability in ('ui', 'runtime', 'installation', 'integration'),
                     f'benefit {identifier}: availability_claim required')
                required_scope = 'package' if availability == 'installation' else 'runtime'
                if availability != 'ui' or benefit.get('proof_scope') == 'runtime':
                    witnesses = [by_source[i] for i in benefit.get('eligibility_sources', []) if i in by_source]
                    need(any(qualifies(source, context, identifier, required_scope) for source in witnesses),
                         f'benefit {identifier}: target-bound observed {required_scope} evidence required')
                if availability != 'ui':
                    need(benefit.get('proof_scope') == 'runtime',
                         f'benefit {identifier}: availability claim requires runtime proof')
            claim_ids = benefit.get('claim_ids', [])
            need(names(claim_ids) and bool(claim_ids) and all(i in by_claim for i in claim_ids),
                 f'benefit {identifier}: qualified claim references required')
            conditions = benefit.get('conditions', [])
            if benefit.get('eligibility') == 'conditional':
                need(names(conditions) and bool(conditions), f'benefit {identifier}: explicit conditions required')
                for claim_id in claim_ids:
                    need(set(conditions) <= set(by_claim.get(claim_id, {}).get('conditions', [])),
                         f'claim {claim_id} omits benefit conditions')
            scope = benefit.get('proof_scope')
            need(scope in ('ui', 'runtime'), f'benefit {identifier}: proof_scope must be ui or runtime')
            proof = []
            for scene in plan.get('scenes', []):
                scene_benefits = scene.get('benefits', [])
                if identifier not in scene_benefits or scene.get('class') not in ('evidence', 'presentation'):
                    continue
                depicted = scene.get('depicted_targets', [])
                need(names(depicted) and bool(depicted) and set(depicted) <= set(targets),
                     f'scene {scene.get("id")} depicts an unsupported product target')
                need(scene.get('proof_scope') in ({'ui', 'runtime'} if scope == 'ui' else {'runtime'}),
                     f'scene {scene.get("id")}: proof scope does not cover benefit {identifier}')
                need(bool(set(claim_ids) & set(scene.get('claims', []))),
                     f'scene {scene.get("id")}: benefit {identifier} lacks its claim reference')
                if scene.get('capture_mode') == 'ui-fixture':
                    need(scope == 'ui', f'scene {scene.get("id")}: fixture cannot prove runtime benefit {identifier}')
                proof.append(scene['id'])
            need(bool(proof), f'no product proof for selected benefit {identifier}')
        review = brief.get('story_review', {})
        need(review.get('status') in ('open', 'pass'), 'brief story review status required')
        for key in ('method', 'customer_outcome', 'setup_accuracy', 'limits'):
            need(text(review.get(key)), f'brief story review {key} required')
        if stage != 'plan':
            need(review.get('status') == 'pass', 'story review must pass before polish')
        if brief.get('version') == 2 and (review.get('status') == 'pass' or stage != 'plan'):
            evidence_class = {'author':'author-review', 'fresh-agent':'agent-diagnostic',
                              'operator':'operator-feedback', 'unfamiliar-target-user':'customer-comprehension'}
            context_type = review.get('reviewer_context')
            need(context_type in evidence_class, 'story review reviewer_context required')
            need(review.get('evidence_class') == evidence_class.get(context_type), 'story review evidence class contradicts reviewer context')
            need(text(review.get('prior_context')), 'story review prior_context required')
            need(isinstance(review.get('audience_match'), bool), 'story review audience_match required')
            if context_type == 'unfamiliar-target-user':
                need(review.get('audience_match') is True, 'customer comprehension requires a matching target audience')
            stimulus = review.get('stimulus', {})
            asset = local_file(stimulus.get('file'), 'story review stimulus')
            if asset:
                need(digest(asset) == stimulus.get('sha256'), 'story review stimulus digest mismatch')
            need(stimulus.get('file') in inputs, 'story review stimulus is not a declared render input')
            for question in ('category', 'job', 'benefit', 'proof', 'next_action'):
                response = review.get('responses', {}).get(question, {})
                need(text(response.get('answer')) and text(response.get('evidence')),
                     f'comprehension {question}: observed answer and stimulus evidence required')
                need(response.get('finding') in ('clear', 'unclear', 'misleading'),
                     f'comprehension {question}: finding required')
                if review.get('status') == 'pass':
                    need(response.get('finding') == 'clear', f'comprehension {question} remains unresolved')

    except (ValueError, KeyError, TypeError, AttributeError) as error:
        errors.append(f'invalid product brief: {error}')
    return errors

