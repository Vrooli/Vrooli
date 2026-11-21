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
    def test_cli_requires_explicit_historical_read_for_retained_formats(self):
        import contextlib
        import io
        plan = self.root/'production.json'
        plan.write_text(json.dumps(self.plan))
        for flags, expected in (([], True), (['--historical'], False)):
            with self.subTest(flags=flags), patch('sys.argv', ['production.py', 'validate', str(plan), *flags]):
                output = io.StringIO()
                with contextlib.redirect_stdout(output):
                    self.assertEqual(expected, p.main())
                result = json.loads(output.getvalue())
                self.assertEqual('fail' if expected else 'pass', result['status'])

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
    def test_faithful_state_hold_is_valid_product_evidence(self):
        self.plan['scenes'][0]['product_motion']=False
        self.assertEqual([],self.errors())
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
        meta={'streams':[{'codec_type':'video','width':1920,'height':1080,'avg_frame_rate':'30/1','duration':'4'}, {'codec_type':'audio','duration':'4'}],'format':{'duration':'4'}}
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

    def test_story_specific_balance_has_no_universal_footage_quota(self):
        self.plan['duration']=20
        import copy
        title=copy.deepcopy(self.plan['scenes'][0]);title.update(id='title',start=4,duration=16)
        title['class']='explanatory'
        self.plan['scenes'].append(title)
        self.assertEqual([],self.errors())

    def test_generated_music_cannot_be_replaced_with_unrequested_original_score(self):
        self.plan['audio'].update(mode='music',source_kind='original-score')
        self.assertIn('original score requires an explicit operator request',self.errors())

    def test_fixture_capture_cannot_prove_backend_claims(self):
        self.plan['scenes'][0]['capture_mode']='ui-fixture'
        self.plan['claims'][0]['scope']='backend'
        self.assertIn('scene proof: fixture capture cannot prove backend claim c1',self.errors())

    def test_requested_music_variety_needs_an_account_beyond_seed_or_genre_count(self):
        self.current_plan()
        self.plan['audio'].update(mode='music',source_kind='licensed-reuse',variety_requested=True,
                                  candidate_styles=['drum and bass','hardgroove'],source_id='test-source')
        self.assertIn('requested music variety needs an evidence-based rationale', self.errors())

    def test_same_family_variety_does_not_force_a_lower_energy_genre(self):
        self.current_plan()
        self.plan['audio'].update(mode='music',source_kind='licensed-reuse',variety_requested=True,
                                  candidate_styles=['drum and bass','drum and bass'],
                                  source_id='test-source', variety_rationale='Two retained takes and different arrangements; '
                                  'rhythm/timbre contrast is still unreviewed pending actual audition.')
        self.assertEqual([], self.errors())

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
        meta={'streams':[{'codec_type':'video','width':1920,'height':1080,'avg_frame_rate':'30/1','duration':'4'}, {'codec_type':'audio','duration':'4'}],
              'format':{'duration':'4'}}
        with patch.object(p,'probe',return_value=meta):
            self.assertEqual([],self.errors('deliver'))

    def test_audio_coverage_uses_streams_not_container(self):
        meta = {'format': {'duration': '33'}, 'streams': [
            {'codec_type': 'video', 'duration': '33', 'start_time': '0'},
            {'codec_type': 'audio', 'duration': '23.115', 'start_time': '0'}]}
        self.assertFalse(p.audio_covers_video(meta))
        meta['streams'][1]['duration'] = '33'
        self.assertTrue(p.audio_covers_video(meta))
        meta['streams'][1]['start_time'] = '1'
        self.assertFalse(p.audio_covers_video(meta))

    def test_unknown_audio_stream_duration_cannot_pass(self):
        self.assertFalse(p.audio_covers_video({'streams': [
            {'codec_type': 'video', 'duration': '33'}, {'codec_type': 'audio'}]}))

    def probe_packets(self, packets, duration='1', ending=None, format_start='0'):
        from types import SimpleNamespace
        meta = {'format': {'duration': duration, 'start_time': format_start}, 'streams': [
            {'index': 0, 'codec_type': 'video', 'tags': {'DURATION': '00:01:39'}},
            {'index': 1, 'codec_type': 'audio', 'sample_rate': '48000', 'tags': {'DURATION': '00:01:39'}}]}
        with patch.object(p, 'run', side_effect=[SimpleNamespace(stdout=json.dumps(meta)),
                                               SimpleNamespace(stdout=json.dumps({'packets': packets})),
                                               SimpleNamespace(stdout=json.dumps({'packets': packets if ending is None else ending}))]):
            return p.probe(self.root/'fallback.webm')

    def test_webm_missing_stream_duration_uses_observed_packet_boundaries(self):
        meta = self.probe_packets([
            {'stream_index': 0, 'pts_time': '0', 'duration_time': '0.04'},
            {'stream_index': 0, 'pts_time': '.96', 'duration_time': '.04'},
            {'stream_index': 1, 'pts_time': '-.007', 'duration_time': '.02'},
            {'stream_index': 1, 'pts_time': '.994', 'duration_time': '.02',
             'side_data_list': [{'discard_padding': 136}]}])
        self.assertTrue(p.audio_covers_video(meta))
        self.assertAlmostEqual(p.stream_span(meta['streams'][1])[1], 1.014-136/48000)

    def test_packet_coverage_refuses_short_audio_despite_duration_tags(self):
        meta = self.probe_packets([
            {'stream_index': 0, 'pts_time': '0', 'duration_time': '1'},
            {'stream_index': 1, 'pts_time': '0', 'duration_time': '.5'}])
        self.assertFalse(p.audio_covers_video(meta))

    def test_packet_coverage_accounts_for_audio_skip_and_discard(self):
        for field, samples in [('skip_samples', 6000), ('discard_padding', 6000)]:
            with self.subTest(field=field):
                meta = self.probe_packets([
                    {'stream_index': 0, 'pts_time': '0', 'duration_time': '1'},
                    {'stream_index': 1, 'pts_time': '0', 'duration_time': '1',
                     'side_data_list': [{field: samples}]}])
                self.assertFalse(p.audio_covers_video(meta))

    def test_packet_coverage_keeps_missing_or_nonfinite_observations_unknown(self):
        for packet in [{}, {'stream_index': 1, 'pts_time': 'NaN', 'duration_time': '1'},
                       {'stream_index': 1, 'pts_time': '0'},
                       {'stream_index': 1, 'pts_time': '0', 'duration_time': '-1'}]:
            with self.subTest(packet=packet):
                meta = self.probe_packets([{'stream_index': 0, 'pts_time': '0', 'duration_time': '1'}, packet])
                self.assertFalse(p.audio_covers_video(meta))

    def test_packet_coverage_refuses_missing_eof_witness_for_both_streams(self):
        packets = [{'stream_index': 0, 'pts_time': '0', 'duration_time': '1'},
                   {'stream_index': 1, 'pts_time': '0', 'duration_time': '1'}]
        for ending in [[], packets]:
            with self.subTest(ending=ending):
                meta = self.probe_packets(packets, duration='36', ending=ending)
                self.assertFalse(p.audio_covers_video(meta))

    def test_rejected_continuation_sources_stay_rejected_after_arrangement(self):
        self.current_plan()
        for source in ['efc8c831-527d-4398-80c6-6065dc79c719-take-02',
                       'efc8c831-527d-4398-80c6-6065dc79c719-take-01',
                       '4d0f02e4-2305-46f7-8f2d-6bc430549052-take-01',
                       'd09c9c44-cc3a-484a-b948-d55f665f4af2-take-01',
                       '8bb80015-1891-4f39-8342-50f27dca1f1e-take-01',
                       'ec08ddd9-857e-4d1b-861f-ef04db66dcbc-take-01',
                       '2448720b-acc3-4a40-826b-0432b96c4062-take-01']:
            with self.subTest(source=source):
                self.plan['audio'].update(mode='music', source_kind='governed-generation',
                                          source_id=source+';arranged-sha256:fixture')
                self.assertIn('selected audio source was explicitly rejected', self.errors())

    def test_packet_coverage_refuses_nonfinite_container_timeline(self):
        packets = [{'stream_index': 0, 'pts_time': '0', 'duration_time': '1'},
                   {'stream_index': 1, 'pts_time': '0', 'duration_time': '1'}]
        for start in ['NaN', 'Inf', '-Inf']:
            with self.subTest(start=start):
                meta = self.probe_packets(packets, duration='36', format_start=start)
                self.assertFalse(p.audio_covers_video(meta))

    def current_plan(self):
        self.plan.update(version=2, review_status='draft', cta={
            'url':'https://vrooli.com/apps/example', 'action':'Explore Example',
            'expected_content':'Example product page', 'availability':'Preview; learn more'})
        self.plan['story'].update(viewing_context='muted landing page', category='work organizer',
            setup='Find decisions in project history', next_action='Explore Example',
            footage_balance='A short decisive interaction supports a longer conceptual introduction')
        self.plan['scenes'][0].update(visual_intention='Calm readable decision detail',
            proof_kind='state', product_motion=False, frame='thin browser', cursor='none: static result',
            settle_evidence='diagram fully loaded', trim='0–4 seconds',
            meaning={'feature':'Message search','sample_input':'offline',
                     'observed_result':'Matching messages','viewer_inference':'Search past messages'})

    def test_current_plan_allows_static_proof_and_context_disclosure(self):
        self.current_plan()
        self.assertEqual([], self.errors())

    def test_query_cannot_be_promoted_to_feature_headline(self):
        self.current_plan()
        self.plan['scenes'][0]['headline']='Offline'
        self.assertTrue(any('sample input cannot be' in e for e in self.errors()))

    def test_internal_route_is_not_usable_cta(self):
        self.current_plan()
        self.plan['cta']['url']='/apps/example'
        self.assertIn('CTA requires a complete public HTTPS URL', self.errors())

    def test_interaction_claim_needs_observed_action(self):
        self.current_plan()
        self.plan['scenes'][0]['proof_kind']='interaction'
        self.assertTrue(any('interaction proof requires' in e for e in self.errors()))

    def test_setup_and_viewing_context_are_required(self):
        self.current_plan()
        self.plan['story']['setup']=''
        self.plan['story']['viewing_context']=''
        self.assertIn('story.setup required', self.errors())
        self.assertIn('story.viewing_context required', self.errors())

    def delivery_fixture(self):
        self.current_plan()
        for name in ('video.mp4', 'poster.jpg', 'page.html'):
            (self.root/name).write_text('unit fixture only; not production evidence')
        current = p.digest(self.root/'video.mp4')
        self.review = {'sha256': current, 'reviewer': 'unit fixture', 'method': 'fixture',
            'visual_observations': 'fixture', 'audio_observations': 'silent fixture',
            'audio_status': 'not-applicable', 'verdict': 'pass', 'defects': [],
            'review_kind': 'self-review', 'opening_comprehension': 'fixture setup',
            'campaign_variety': 'fixture direction comparison',
            'viewer_comprehension': {'verdict': 'pass', 'method': 'fixture',
                'scenes': [{'id': 'proof', 'observation': 'fixture meaning finding', 'verdict': 'pass'}]},
            'craft': {'verdict': 'pass', 'method': 'fixture',
                'scenes': [{'id': 'proof', 'observation': 'fixture craft finding', 'verdict': 'pass'}]}}
        (self.root/'review.json').write_text(json.dumps(self.review))
        (self.root/'technical.json').write_text(json.dumps({'sha256': current, 'decode_ok': True, 'inputs_stable': True}))
        (self.root/'destination.json').write_text(json.dumps({'url': self.plan['cta']['url'],
            'rendered_product_match': True, 'checked_at': '2026-10-16',
            'observation': 'fixture product identity', 'evidence': 'page.html'}))
        self.plan['cta']['destination_report'] = 'destination.json'
        self.plan.update(video='video.mp4', poster='poster.jpg', technical_report='technical.json', review='review.json')
        self.meta = {'streams': [{'codec_type': 'video', 'width': 1920, 'height': 1080,
            'avg_frame_rate': '30/1', 'duration': '4'}], 'format': {'duration': '4'}}

    def test_current_delivery_requires_scene_meaning_and_craft_findings(self):
        self.delivery_fixture()
        with patch.object(p, 'probe', return_value=self.meta):
            self.assertEqual([], self.errors('deliver'))
            self.review['viewer_comprehension']['scenes'] = []
            (self.root/'review.json').write_text(json.dumps(self.review))
            self.assertTrue(any('requires findings for every scene' in e for e in self.errors('deliver')))

    def test_source_only_or_partial_qr_receipt_does_not_approve_final_video(self):
        self.delivery_fixture()
        self.plan['cta'].update(qr=True, qr_report='qr.json')
        receipt = {'sha256': p.digest(self.root/'video.mp4'), 'url': self.plan['cta']['url'],
            'status': 'pass', 'inputs_stable': True, 'times': [2, 3], 'widths': [1920, 390],
            'checks': [{'time': 2, 'width': 1920, 'passed': True, 'decoded': self.plan['cta']['url']}]}
        (self.root/'qr.json').write_text(json.dumps(receipt))
        with patch.object(p, 'probe', return_value=self.meta):
            self.assertTrue(any('missing, failed or mismatched' in e for e in self.errors('deliver')))

    def test_example_retrieval_does_not_match_research_directory_as_search_intent(self):
        rows = [{'id': 'metrics', 'evidence': '/product-video-research/metrics.json'},
                {'id': 'message-search', 'purpose': ['search'], 'evidence': '/product-video-research/search.json'}]
        self.assertEqual(['message-search'], [r['id'] for r in p.retrieve(rows, 'search', 4)])

    def test_duplicate_qr_samples_do_not_establish_multiple_playback_sizes(self):
        self.delivery_fixture()
        self.plan['cta'].update(qr=True, qr_report='qr.json')
        receipt = {'sha256': p.digest(self.root/'video.mp4'), 'url': self.plan['cta']['url'],
            'status': 'pass', 'inputs_stable': True, 'times': [2, 2], 'widths': [390, 390],
            'checks': [{'time': 2, 'width': 390, 'passed': True, 'decoded': self.plan['cta']['url']}]}
        (self.root/'qr.json').write_text(json.dumps(receipt))
        with patch.object(p, 'probe', return_value=self.meta):
            self.assertTrue(any('multiple times and sizes' in e for e in self.errors('deliver')))

    def test_rejected_audio_take_cannot_silently_return(self):
        self.current_plan()
        self.plan['audio'].update(mode='music', source_kind='governed-generation', source_id='25d4cdfc-08aa-447b-a081-baf110c4c0fd-take-02')
        self.assertIn('selected audio source was explicitly rejected', self.errors())

    def test_faithful_image_hold_passes_polish_without_a_video_duration(self):
        self.current_plan()
        (self.root/'still.png').write_text('image fixture')
        self.plan['scenes'][0].update(media='still.png', source_sha256=p.digest(self.root/'still.png'))
        meta = {'streams': [{'codec_type': 'video', 'codec_name': 'png', 'width': 1920, 'height': 1080}], 'format': {'format_name': 'png_pipe'}}
        with patch.object(p, 'probe', return_value=meta):
            self.assertEqual([], self.errors('polish'))
            self.plan['scenes'][0].update(proof_kind='interaction', product_motion=True)
            self.assertTrue(any('still image cannot prove an interaction' in e for e in self.errors('polish')))


class ProductBriefTests(unittest.TestCase):
    """Protect offer eligibility and evidence relationships, not slogan wording."""

    def test_malformed_brief_reference_returns_a_finding(self):
        self.plan['product_brief'] = ['not-an-object']
        self.assertIn('product brief: object reference required', self.errors())

    def setUp(self):
        ProductionTests.setUp(self)
        self.current_plan()
        self.plan.update(version=3, context={
            'offer_ref':'desktop-preview', 'offer_revision':'offer-r1',
            'product_revision':'build-r1', 'targets':['desktop']})
        (self.root/'source.json').write_text('{"observed":"desktop search"}')
        self.brief = {
            'version':1, 'product':'demo', 'context':dict(self.plan['context']),
            'sources':[{'id':'product', 'kind':'product', 'file':'source.json',
                        'sha256':p.digest(self.root/'source.json'), 'read_at':'2026-10-18T00:00:00Z'},
                       {'id':'package', 'kind':'package', 'file':'source.json',
                        'sha256':p.digest(self.root/'source.json'), 'read_at':'2026-10-18T00:00:00Z'}],
            'candidates':[
                {'id':'search', 'job':'Recover a prior decision', 'alternative':'Manually reread transcripts',
                 'pain':'Too many conversations to check', 'benefit':'Find the relevant decision',
                 'value_basis':'observed', 'evidence':['product'], 'eligibility_sources':['package'],
                 'eligibility':'supported', 'reliability':'Observed on the declared desktop build',
                 'setup':'Normal desktop installation', 'dependencies':'Packaged local store',
                 'reason':'Directly useful recurring task', 'proof_scope':'ui', 'claim_ids':['c1']},
                {'id':'mobile', 'job':'Continue away from desk', 'alternative':'Return to desk',
                 'pain':'Desk-bound work', 'benefit':'Direct agents from a phone', 'value_basis':'operator',
                 'evidence':['product'], 'eligibility_sources':['package'], 'eligibility':'excluded',
                 'reliability':'Not qualified for this offer', 'setup':'Secure tunnel and home-screen setup',
                 'dependencies':'External secure access', 'reason':'Outside desktop offer',
                 'proof_scope':'runtime', 'claim_ids':[]}],
            'selected':['search'], 'selection_rationale':'Search proves a useful desktop workflow',
            'story_review':{'status':'pass', 'method':'Bounded evidence review',
                            'customer_outcome':'Recover a decision without rereading everything',
                            'setup_accuracy':'Desktop only; mobile excluded',
                            'limits':'Synthetic test, not audience evidence'}}
        self.plan['claims'][0].update(scope='ui', conditions=[])
        self.plan['story']['benefits']=['search']
        self.plan['scenes'][0].update(benefits=['search'], proof_scope='ui', depicted_targets=['desktop'])
        self.plan['render_inputs']=['product-brief.json','source.json','clip.mp4']
        self.write_brief()

    current_plan = ProductionTests.current_plan
    errors = ProductionTests.errors

    def write_brief(self):
        path=self.root/'product-brief.json'
        path.write_text(json.dumps(self.brief))
        self.plan['product_brief']={'file':path.name,'sha256':p.digest(path)}

    def test_requested_study_can_preserve_unknown_value_without_product_proof(self):
        self.plan.update(kind='study', study_requested=True, request_source='operator requested concept study')
        self.plan['safety']['status']='capture-blocked'
        self.plan['scenes'][0].update(**{'class':'explanatory'}, benefits=[])
        self.plan['story']['benefits']=[]
        self.brief['selected']=[]
        self.brief['candidates'][0]['eligibility']='unknown'
        self.write_brief()
        self.assertEqual([], self.errors('polish'))
        self.plan['kind']='launch'
        self.assertTrue(any('selected benefits must reference candidates' in e for e in self.errors()))

    def test_empty_study_cannot_smuggle_an_unselected_scene_benefit(self):
        self.plan['kind']='study'
        self.plan['story']['benefits']=[]
        self.brief['selected']=[]
        self.write_brief()
        self.assertTrue(any('unknown or unselected benefit' in e for e in self.errors()))

    def test_supported_benefit_has_offer_and_scene_evidence(self):
        self.assertEqual([],self.errors())

    def test_excluded_or_unknown_benefit_cannot_be_selected(self):
        for eligibility in ('excluded','unknown'):
            with self.subTest(eligibility=eligibility):
                self.brief['candidates'][0]['eligibility']=eligibility
                self.write_brief()
                self.assertTrue(any('selected benefit search is not eligible' in e for e in self.errors()))

    def test_changed_offer_or_product_requires_reassessment(self):
        for key in ('offer_revision','product_revision','targets'):
            with self.subTest(key=key):
                original=self.plan['context'][key]
                self.plan['context'][key]=['phone'] if key=='targets' else 'changed'
                self.assertTrue(any('brief context differs' in e for e in self.errors()))
                self.plan['context'][key]=original

    def test_changed_evidence_bytes_are_not_current(self):
        (self.root/'source.json').write_text('new build evidence')
        self.assertTrue(any('source product digest mismatch' in e for e in self.errors()))

    def test_missing_brief_or_stale_brief_is_rejected(self):
        self.plan['product_brief']['sha256']='stale'
        self.assertTrue(any('product brief digest mismatch' in e for e in self.errors()))
        self.plan.pop('product_brief')
        self.assertTrue(any('product brief' in e for e in self.errors()))

    def test_selected_value_needs_product_and_package_evidence(self):
        self.brief['candidates'][0]['eligibility_sources']=['product']
        self.write_brief()
        self.assertTrue(any('eligibility evidence' in e for e in self.errors()))

    def test_proof_must_cover_selected_benefits(self):
        self.plan['scenes'][0]['benefits']=[]
        self.assertTrue(any('no product proof for selected benefit search' in e for e in self.errors()))

    def test_phone_player_does_not_imply_phone_product_support(self):
        self.plan['story']['viewing_context']='phone-sized social player'
        self.assertEqual([],self.errors())
        self.plan['scenes'][0]['depicted_targets']=['phone']
        self.assertTrue(any('depicts an unsupported product target' in e for e in self.errors()))

    def test_conditional_value_needs_explicit_claim_conditions(self):
        self.brief['candidates'][0].update(eligibility='conditional',conditions=['Configured speech provider'])
        self.write_brief()
        self.assertTrue(any('claim c1 omits benefit conditions' in e for e in self.errors()))
        self.plan['claims'][0]['conditions']=['Configured speech provider']
        self.assertEqual([],self.errors())

    def test_runtime_benefit_cannot_use_ui_only_proof(self):
        self.brief['candidates'][0]['proof_scope']='runtime'
        self.write_brief()
        self.assertTrue(any('proof scope does not cover benefit search' in e for e in self.errors()))

    def test_brief_sources_must_be_in_frozen_inputs(self):
        self.plan['render_inputs'].remove('source.json')
        self.assertTrue(any('source product is not a declared render input' in e for e in self.errors()))

    def test_open_story_review_can_plan_but_cannot_polish(self):
        self.brief['story_review']['status']='open'
        self.write_brief()
        self.assertEqual([],self.errors())
        self.assertTrue(any('story review must pass before polish' in e for e in self.errors('polish')))



    def test_current_cli_rejects_legacy_brief_even_with_current_plan(self):
        self.brief['version'] = 1
        self.write_brief()
        import contextlib
        import io
        plan = self.root/'production.json'
        plan.write_text(json.dumps(self.plan))
        for flags, expected in (([], True), (['--historical'], False)):
            with self.subTest(flags=flags), patch('sys.argv', ['production.py', 'validate', str(plan), *flags]):
                with contextlib.redirect_stdout(io.StringIO()):
                    self.assertEqual(expected, p.main())


class QualifiedBriefTests(ProductBriefTests):
    def setUp(self):
        super().setUp()
        self.brief['version'] = 2
        (self.root/'script.md').write_text('A workspace for decisions. Search prior conversations, then explore the preview.')
        self.plan['render_inputs'].append('script.md')
        self.brief['story_review'].update(
            reviewer_context='author', prior_context='Knows the product and intended story',
            audience_match=False, evidence_class='author-review',
            stimulus={'file':'script.md','sha256':p.digest(self.root/'script.md')},
            responses={key:{'answer':'Specific observed meaning', 'finding':'clear',
                            'evidence':'Opening and search-result scene'}
                       for key in ('category','job','benefit','proof','next_action')})
        self.brief['candidates'][0]['availability_claim']='ui'
        self.write_brief()

    def test_v2_author_diagnostic_can_pass_without_claiming_customer_research(self):
        self.assertEqual([], self.errors())

    def test_commercial_intent_cannot_qualify_runtime(self):
        self.brief['candidates'][0].update(proof_scope='runtime', availability_claim='runtime')
        self.brief['sources'][1]['kind']='offer'
        self.plan['scenes'][0]['proof_scope']='runtime'
        self.write_brief()
        self.assertTrue(any('target-bound observed runtime evidence' in e for e in self.errors()))

    def test_runtime_evidence_is_bound_to_build_target_and_benefit(self):
        self.brief['candidates'][0].update(proof_scope='runtime',availability_claim='runtime')
        self.plan['scenes'][0]['proof_scope']='runtime'
        source=self.brief['sources'][1]
        source.update(kind='runtime', qualification={'status':'observed','scope':'runtime',
                      'product_revision':'build-r1','targets':['desktop'],'benefit_ids':['search']})
        self.brief['sources'].append({**source, 'id':'offer', 'kind':'offer'})
        self.brief['candidates'][0]['eligibility_sources'].append('offer')
        self.write_brief()
        self.assertEqual([], self.errors())
        for field,value in [('product_revision','old-build'),('targets',['mobile']),('benefit_ids',['unrelated']),('status','unverified')]:
            with self.subTest(field=field):
                original=source['qualification'][field]
                source['qualification'][field]=value
                self.write_brief()
                self.assertTrue(any('target-bound observed runtime evidence' in e for e in self.errors()))
                source['qualification'][field]=original

    def test_installation_requires_package_evidence(self):
        self.brief['candidates'][0]['availability_claim']='installation'
        self.write_brief()
        self.assertTrue(any('target-bound observed package evidence' in e for e in self.errors()))

    def test_author_cannot_label_review_as_customer_evidence(self):
        self.brief['story_review']['evidence_class']='customer-comprehension'
        self.write_brief()
        self.assertTrue(any('review evidence class' in e for e in self.errors()))

    def test_changed_script_invalidates_review(self):
        (self.root/'script.md').write_text('Different claims and setup')
        self.assertTrue(any('review stimulus digest' in e for e in self.errors()))

    def test_review_requires_observed_answers_before_polish(self):
        self.brief['story_review']['responses'].pop('category')
        self.write_brief()
        self.assertTrue(any('comprehension category' in e for e in self.errors('polish')))

    def test_known_misunderstanding_cannot_pass_review(self):
        self.brief['story_review']['responses']['proof']['finding']='misleading'
        self.write_brief()
        self.assertTrue(any('comprehension proof remains unresolved' in e for e in self.errors()))

if __name__ == '__main__':
    unittest.main()
