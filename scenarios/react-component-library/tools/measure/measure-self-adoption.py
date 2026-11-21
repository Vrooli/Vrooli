import os, re, collections

root = '/home/matthalloran8/Vrooli/scenarios/react-component-library/ui/src'; imports = collections.Counter(); using = set(); total = set()
for directory, _, files in os.walk(root):
    if 'node_modules' in directory: continue
    for filename in files:
        if not filename.endswith(('.tsx', '.ts')) or '.test.' in filename: continue
        path = os.path.join(directory, filename); total.add(path); source = open(path, errors='ignore').read()
        for module in re.findall(r'from ["\'](@vrooli/react-component-library[^"\']*)["\']', source): imports[module] += 1; using.add(path)
print('non-test UI files:', len(total)); print('files importing the published library:', len(using), '(%.1f%%)' % (100 * len(using) / max(1, len(total))))
print('\n=== library entry points imported ===')
for module, count in imports.most_common(60): print('  %3d  %s' % (count, module))
