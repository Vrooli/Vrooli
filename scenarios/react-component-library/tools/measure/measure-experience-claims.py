import glob, json, collections

root = '/home/matthalloran8/Vrooli/scenarios/react-component-library/library'
contracts = glob.glob(root + '/**/experience-contract.json', recursive=True)
page_contracts = glob.glob('/home/matthalloran8/Vrooli/scenarios/react-component-library/experience/pages/*.json')
assets = {path.split('/versions/')[0] for path in glob.glob(root + '/**/versions', recursive=True)}
print('experience-contract.json files:', len(contracts)); print('asset directories:', len(assets))
with_contract = {path.split('/versions/')[0] for path in contracts}
print('assets with ANY experience contract:', len(with_contract), '(%.1f%%)' % (100 * len(with_contract) / max(1, len(assets))))
types = collections.Counter(); total = 0; empty = 0; visual_assets = set()
visual_types = {'single-dominant-action', 'visible-without-scroll', 'spacing', 'state-contrast', 'size-parity'}
for path in contracts:
    try: data = json.load(open(path))
    except Exception: continue
    claims = data.get('claims') or data.get('experienceClaims') or []
    if not claims: empty += 1
    for claim in claims:
        total += 1; kind = claim.get('type') or claim.get('claimType') or claim.get('kind'); types[kind] += 1
        if kind in visual_types: visual_assets.add(path.split('/versions/')[0])
print('\ntotal claims:', total, ' contracts with zero claims:', empty)
print('assets declaring a claim the *visual* gate can check:', len(visual_assets), '(%.1f%% of all assets)' % (100 * len(visual_assets) / max(1, len(assets))))
print('\nclaim types:')
for kind, count in types.most_common(30): print('  %4d  %s' % (count, kind))

page_claims = 0
page_states = 0
uncovered_states = []
for path in page_contracts:
    try:
        data = json.load(open(path))
    except Exception:
        continue
    claims = data.get('claims') or []
    states = {state.get('id') for state in data.get('states') or [] if state.get('id')}
    covered = {state for claim in claims for state in claim.get('states') or []}
    page_claims += len(claims)
    page_states += len(states)
    if states - covered:
        uncovered_states.append('%s: %s' % (path.rsplit('/', 1)[-1], ','.join(sorted(states - covered))))
print('\npage experience contracts:', len(page_contracts))
print('page claims:', page_claims, 'declared states:', page_states)
print('page states without a claim:', len(uncovered_states))
for item in uncovered_states: print('  ', item)
