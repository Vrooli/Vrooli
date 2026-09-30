import glob, json, collections

root = '/home/matthalloran8/Vrooli/scenarios/react-component-library/library'
files = glob.glob(root + '/**/story.json', recursive=True)
roles = collections.Counter(); assets = {}; no_boundary = []; enum_boundaries = []; total = 0
for path in files:
    try: data = json.load(open(path))
    except Exception: continue
    asset_root = path.split('/versions/')[0]
    version = path.split('/versions/')[1].split('/')[0] if '/versions/' in path else ''
    try:
        manifest = json.load(open(asset_root + '/component.json'))
    except Exception:
        continue
    if version != manifest.get('latest'):
        continue
    stories = data.get('stories', [])
    if not stories: continue
    asset = asset_root.replace(root + '/', '')
    counts = collections.Counter(story.get('role') for story in stories); assets[path] = counts
    for story in stories: total += 1; roles[story.get('role')] += 1
    if not counts.get('boundary', 0): no_boundary.append(asset)
    enums = {field['path'] for field in (data.get('args', {}).get('fields') or []) if field.get('kind') == 'enum'}
    axes = [story for story in stories if story.get('role') == 'axis']
    for story in stories:
        args = story.get('args') or {}
        if story.get('role') != 'boundary' or not args or not set(args).issubset(enums):
            continue
        # A boundary may carry an enum value while exercising a lifecycle or
        # interaction state. Those stories are not replacements for the axis;
        # keep this measurement aligned with the blocking story-grammar gate.
        if story.get('states') or story.get('interactions'):
            continue
        if any(
            field in args and args[field] in (axis.get('covers') or {}).get(field, [])
            for axis in axes
            for field in (axis.get('covers') or {})
        ):
            enum_boundaries.append((asset, story.get('id')))
print('story contracts:', len(files), ' stories:', total)
print('\n=== role distribution (the formalized taxonomy) ===')
for key, value in roles.most_common(): print('  %5d  %-12s %.1f%%' % (value, key, 100 * value / total))
print('\nassets with NO boundary story:', len(set(no_boundary)), 'of', len({path.replace(root + '/', '').split('/versions/')[0] for path in assets}))
print("\n'boundary' stories that duplicate an enum axis:", len(enum_boundaries))
for asset, story_id in enum_boundaries[:20]: print('   %-42s %s' % (asset, story_id))
