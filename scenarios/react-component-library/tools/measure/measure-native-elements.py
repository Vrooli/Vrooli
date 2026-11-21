import os, re

root = '/home/matthalloran8/Vrooli/scenarios/react-component-library/ui/src'; native_select = []; raw_button = []
for directory, _, files in os.walk(root):
    if 'node_modules' in directory: continue
    for filename in files:
        if not filename.endswith('.tsx') or '.test.' in filename: continue
        path = os.path.join(directory, filename); source = open(path, errors='ignore').read()
        source = re.sub(r'/\*.*?\*/|//[^\n]*', '', source, flags=re.S)
        native_select += [path] * len(re.findall(r'<select[\s>]', source)); raw_button += [path] * len(re.findall(r'<button[\s>]', source))
print('files using a NATIVE <select>:', len(set(native_select)), 'occurrences:', len(native_select))
for path in sorted(set(native_select)): print('   ', path.replace(root + '/', ''))
print('\nfiles using a raw <button>:', len(set(raw_button)), 'occurrences:', len(raw_button))
for path in sorted(set(raw_button) )[:25]: print('   ', path.replace(root + '/', ''))
