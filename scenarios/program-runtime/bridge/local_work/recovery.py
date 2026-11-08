"""Reviewed local checkpoints and recovery; never a claim of stream completeness.

These methods have no network and do not grant authority. The management owner
must authenticate the operator and supply the exact action-time approval. They
are deliberately absent from the webhook/periodic worker operation allowlist.
"""
from dataclasses import dataclass
import json
from .adapter import Hold,canonical,digest,opaque,validated_snapshot
from .projection import project_v2


@dataclass(frozen=True)
class RecoveryApproval:
    reference:str
    source_instance:str
    source_epoch:str
    expected_state_digest:str
    not_before:int
    expires_at:int

    def validate(self,outbox,actual,now):
        opaque(self.reference,'recovery_reference')
        if (self.source_instance,self.source_epoch)!=(outbox.policy.source_instance,outbox.policy.source_epoch):
            raise Hold('recovery_source_mismatch')
        if self.expected_state_digest!=actual:raise Hold('recovery_state_changed')
        if type(self.not_before) is not int or type(self.expires_at) is not int or not 0<self.expires_at-self.not_before<=900 or not self.not_before<=now<self.expires_at:
            raise Hold('recovery_approval_not_current')


class Checkpoints:
    def __init__(self,outbox):
        self.outbox=outbox
        with outbox.transaction():
            outbox.db.execute('''CREATE TABLE IF NOT EXISTS recovery_checkpoints(
              reference TEXT PRIMARY KEY,kind TEXT NOT NULL,observed REAL NOT NULL,
              before_digest TEXT NOT NULL,evidence TEXT NOT NULL)''')

    def _view(self):
        db=self.outbox.db
        result={'source_instance':self.outbox.policy.source_instance,'source_epoch':self.outbox.policy.source_epoch,
                'historical_coverage':'unknown','checkpoint_kind':'local_durable_observation_only',
                'events':[dict(r) for r in db.execute('SELECT event_id,state,event_projection,batch_id FROM native_events ORDER BY event_id')],
                'batches':[dict(r) for r in db.execute('SELECT batch_id,state,attempts,due,lease_until,receipt_id FROM batches ORDER BY batch_id')],
                'holds':[r[0] for r in db.execute('SELECT kind FROM holds ORDER BY kind')],
                'coordination':{r['key']:r['value'] for r in db.execute("SELECT key,value FROM meta WHERE key IN ('collection_retry','collection_lease','active_collection_ids') ORDER BY key")}}
        # Retain identities/state only, not content digest projections in presentation.
        for event in result['events']:
            event['projection_digest']=digest(event.pop('event_projection'))
        return result

    def inspect(self):
        with self.outbox.transaction():
            view=self._view();view['state_digest']=digest(view)
            return view

    def record_baseline(self,approval,receipt,now):
        projection=validated_snapshot(receipt,now,self.outbox.policy.projection_policy())
        return self._apply('bounded_baseline',approval,now,{'snapshot':projection},False)

    def resume_same_collection(self,approval,now):
        """Reset local collection backoff only, preserving the original admission.

        No event/receipt/hold is removed; no failed execution is re-run. The owner
        client observes/resumes its same durable program, including an old failure.
        A new collector admission needs a separate future governed recovery design.
        """
        return self._apply('resume_same_collection',approval,now,{'execution_policy':'reuse_original_admission'},True)

    def _apply(self,kind,approval,now,evidence,resume):
        db=self.outbox.db
        with self.outbox.transaction():
            previous=db.execute('SELECT * FROM recovery_checkpoints WHERE reference=?',(approval.reference,)).fetchone()
            if previous:
                if previous['kind']!=kind or previous['before_digest']!=approval.expected_state_digest or previous['evidence']!=canonical(evidence):raise Hold('recovery_reference_conflict')
                # Still check scope/lifetime; a consumed request cannot expand authority.
                approval.validate(self.outbox,approval.expected_state_digest,now)
                return {'state':'duplicate','reference':approval.reference}
            view=self._view();before=digest(view);approval.validate(self.outbox,before,now)
            lease=db.execute("SELECT value FROM meta WHERE key='collection_lease'").fetchone()
            if lease and json.loads(lease['value'])['until']>now:raise Hold('recovery_collection_in_flight')
            if resume:
                retry=db.execute("SELECT value FROM meta WHERE key='collection_retry'").fetchone()
                if not retry or not json.loads(retry['value']).get('held'):raise Hold('recovery_requires_held_collection')
                if 'event_identity_conflict' in view['holds']:raise Hold('recovery_cannot_clear_identity_conflict')
                # Keep generation monotonic. Remove an expired lease; the next
                # collector gets a new generation so old callbacks remain fenced.
                db.execute("DELETE FROM meta WHERE key IN ('collection_retry','collection_lease')")
            if kind=='bounded_baseline' and evidence['snapshot']['permission_hold']:
                db.execute("INSERT OR IGNORE INTO holds VALUES ('permission_hold',?)",(now,))
            db.execute('INSERT INTO recovery_checkpoints VALUES (?,?,?,?,?)',(approval.reference,kind,now,before,canonical(evidence)))
            return {'state':'recorded','reference':approval.reference,'historical_coverage':'unknown',
                    'remaining_holds':[r[0] for r in db.execute('SELECT kind FROM holds ORDER BY kind')],'new_execution_authorized':False}
