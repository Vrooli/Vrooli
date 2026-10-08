import glob, json, collections

root = '/home/matthalloran8/Vrooli/scenarios/react-component-library/library'; latest = {}
for path in glob.glob(root + '/**/story.json', recursive=True):
    asset = path.replace(root + '/', '').split('/versions/')[0]; version = path.split('/versions/')[1].split('/')[0]
    if asset not in latest or version > latest[asset][0]: latest[asset] = (version, path)
single = []; noargs = []; nofields = []; total = 0; distribution = collections.Counter()
for asset, (_, path) in sorted(latest.items()):
    try: data = json.load(open(path))
    except Exception: continue
    stories = data.get('stories', [])
    if not stories: continue
    total += 1; distribution[min(len(stories), 8)] += 1
    if not (data.get('args', {}).get('fields') or []): nofields.append(asset)
    if len(stories) == 1: single.append((asset, stories[0].get('id'), bool(stories[0].get('args'))))
    if all(not (story.get('args') or {}) for story in stories): noargs.append(asset)
print('assets (latest version) with a story contract:', total)
print('\n=== how many stories does an asset have? ===')
for count in sorted(distribution): print('  %-3s stories : %3d assets  (%.1f%%)' % ('8+' if count == 8 else count, distribution[count], 100 * distribution[count] / total))
print('\nassets with EXACTLY ONE story:', len(single), '(%.1f%%)' % (100 * len(single) / total))
print('assets where EVERY story passes zero args:', len(noargs), '(%.1f%%)' % (100 * len(noargs) / total))
print('assets declaring NO args.fields:', len(nofields), '(%.1f%%)' % (100 * len(nofields) / total))
print('\n=== single-story assets (first 45) ===')
for asset, story_id, has_args in single[:45]: print('  %-44s story=%-22s args=%s' % (asset, story_id, 'yes' if has_args else 'NONE'))
