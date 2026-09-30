import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch
import production as p


class ProductionTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        for name in ['safety.md','audio.md','privacy.md','clip.mp4']:
            (self.root/name).write_text('fixture')
        self.plan = {
            'version':1,'kind':'launch','product':'demo','profile':'teaser','duration':4,
            'format':{'width':1920,'height':1080,'fps':30},
            'safety':{'status':'verified','report':'safety.md','protected_mutations':[]},
            'story':{**{key:'specific value' for key in p.STORY},'proof_scene':'proof'},
            'claims':[{'id':'c1','source':'official source v2','read_at':'2026-09-21','approved':'View signals',
                       'wording':'View signals','qualifiers':'none','availability':'available','verified':True,'exact_copy':True}],
            'audio':{'mode':'silent','direction':'deliberate silent demonstration','selection':'audio.md'},
            'scenes':[{'id':'proof','start':0,'duration':4,'class':'evidence','purpose':'prove claim','adds':'actual result',
                       'claims':['c1'],'focal_point':'result','motion_reason':'reveal consequence','start_state':'ready',
                       'action':'click','end_state':'result','text':[{'value':'View signals','hold':1}],
                       'media':'clip.mp4','source_sha256':p.digest(self.root/'clip.mp4'),'capture_instance':'demo@presentation',
                       'capture_date':'2026-09-21','privacy_review':'privacy.md','product_motion':True,
                       'crop':[0,0,1920,1080],'max_display':[1920,1080]}]}
    def errors(self,stage='plan'):
        return p.validate(self.plan,self.root,stage)
    def test_teaser_does_not_require_flagship_cost(self):
        self.assertEqual([],self.errors())
    def test_missing_promise_fails(self):
        self.plan['story']['promise']=''
        self.assertIn('story.promise required',self.errors())
    def test_unknown_claim_and_locked_rewrite_fail(self):
        self.plan['claims'][0]['wording']='Automatically fixes signals'
        self.plan['scenes'][0]['claims']=['invented']
        self.assertTrue(any('locked copy changed' in e for e in self.errors()))
        self.assertTrue(any('unknown claim' in e for e in self.errors()))
    def test_source_pixels_include_peak_zoom(self):
        self.plan['scenes'][0]['max_display']=[2000,1100]
        self.assertTrue(any('insufficient source pixels' in e for e in self.errors()))
    def test_crop_must_fit_actual_encoded_source(self):
        meta={'streams':[{'codec_type':'video','width':1280,'height':720}],'format':{'duration':'4'}}
        with patch.object(p,'probe',return_value=meta):
            self.assertTrue(any('crop outside' in e for e in self.errors('polish')))
    def test_changed_footage_invalidates_provenance(self):
        (self.root/'clip.mp4').write_text('replacement')
        self.assertTrue(any('source digest mismatch' in e for e in self.errors()))
    def test_camera_motion_alone_cannot_prove_product(self):
        self.plan['scenes'][0]['product_motion']=False
        self.assertTrue(any('real product motion required' in e for e in self.errors()))
    def test_short_text_hold_fails(self):
        self.plan['scenes'][0]['text']=[{'value':'A full sentence that needs time','hold':.5}]
        self.assertTrue(any('reading hold' in e for e in self.errors()))
    def test_missing_safety_and_protected_mutation_fail(self):
        self.plan['safety']={'report':'missing','protected_mutations':['restart critical service']}
        errors=self.errors()
        self.assertTrue(any('missing or empty' in e for e in errors))
        self.assertIn('protected mutations must be empty',errors)
    def test_flagship_rejects_palette_only_variants(self):
        self.plan['profile']='flagship'
        self.plan['directions']=[{'id':i,'structure':'same','shot_language':'same','audio':'same','rationale':'blue'} for i in ['a','b','c']]
        self.plan['chosen_direction']='a'
        self.assertIn('directions must differ beyond palette/title',self.errors())
    def test_nan_timing_fails(self):
        self.plan['duration']=float('nan')
        self.assertTrue(self.errors())
    def test_study_can_be_planned_but_not_delivered_as_launch(self):
        self.plan['kind']='study';self.plan['safety']['status']='capture-blocked'
        self.plan['scenes'][0]['class']='explanatory'
        self.assertEqual([],self.errors())
        self.assertIn('study is not a completed launch demonstration',self.errors('deliver'))
        self.assertIn('missing real product proof',self.errors('deliver'))
    def test_review_must_bind_final_file_and_resolve_major_defects(self):
        for name in ['video.mp4','poster.jpg']:(self.root/name).write_text('asset')
        report={'sha256':'stale','decode_ok':True}
        (self.root/'inspection.json').write_text(json.dumps(report))
        review={'sha256':'stale','reviewer':'critic','method':'export playback','visual_observations':'readable',
                'audio_observations':'silent by design','audio_status':'unreviewed','verdict':'pass','defects':[{'timestamp':1,'severity':'major',
                'category':'typography','observation':'too small','consequence':'unreadable','recommendation':'enlarge','disposition':'open'}]}
        (self.root/'review.json').write_text(json.dumps(review))
        self.plan['audio']['mode']='music'
        self.plan.update(video='video.mp4',poster='poster.jpg',technical_report='inspection.json',review='review.json')
        meta={'streams':[{'codec_type':'video','width':1920,'height':1080,'avg_frame_rate':'30/1'}],'format':{'duration':'4'}}
        with patch.object(p,'probe',return_value=meta):
            errors=self.errors('deliver')
        self.assertTrue(any('review does not match' in e for e in errors))
        self.assertTrue(any('unresolved major' in e for e in errors))
        self.assertTrue(any('technical inspection is stale' in e for e in errors))
        self.assertTrue(any('exported audio must be auditioned' in e for e in errors))

    def test_unrequested_study_stops_before_polish(self):
        self.plan['kind']='study'
        self.plan['safety']['status']='capture-blocked'
        self.plan['scenes'][0]['class']='explanatory'
        self.assertIn('unrequested study cannot proceed to polish',self.errors('polish'))

    def test_token_product_insert_does_not_replace_product_led_film(self):
        self.plan['duration']=20
        import copy
        title=copy.deepcopy(self.plan['scenes'][0]);title.update(id='title',start=4,duration=16)
        title['class']='explanatory'
        self.plan['scenes'].append(title)
        self.assertIn('product captures must occupy at least half the film',self.errors())

    def test_generated_music_cannot_be_replaced_with_unrequested_original_score(self):
        self.plan['audio'].update(mode='music',source_kind='original-score')
        self.assertIn('original score requires an explicit operator request',self.errors())

    def test_fixture_capture_cannot_prove_backend_claims(self):
        self.plan['scenes'][0]['capture_mode']='ui-fixture'
        self.plan['claims'][0]['scope']='backend'
        self.assertIn('scene proof: fixture capture cannot prove backend claim c1',self.errors())

    def test_requested_music_variety_is_not_three_seeds_of_one_style(self):
        self.plan['audio'].update(mode='music',source_kind='licensed-reuse',variety_requested=True,
                                  candidate_styles=['trap','trap','trap'])
        self.assertIn('requested music variety needs distinct styles',self.errors())

    def test_current_review_and_complete_teaser_can_pass_delivery(self):
        for name in ['video.mp4','poster.jpg']:
            (self.root/name).write_text('test fixture only')
        current=p.digest(self.root/'video.mp4')
        (self.root/'inspection.json').write_text(json.dumps({'sha256':current,'decode_ok':True}))
        (self.root/'review.json').write_text(json.dumps({
            'sha256':current,'reviewer':'test fixture','method':'fixture',
            'visual_observations':'fixture inspected','audio_observations':'silent by design',
            'audio_status':'not-applicable','verdict':'pass','defects':[]}))
        self.plan.update(video='video.mp4',poster='poster.jpg',technical_report='inspection.json',review='review.json')
        meta={'streams':[{'codec_type':'video','width':1920,'height':1080,'avg_frame_rate':'30/1'}],
              'format':{'duration':'4'}}
        with patch.object(p,'probe',return_value=meta):
            self.assertEqual([],self.errors('deliver'))


if __name__=='__main__':unittest.main()
