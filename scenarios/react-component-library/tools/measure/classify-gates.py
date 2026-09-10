import json

gates = json.load(open('/home/matthalloran8/Vrooli/scenarios/react-component-library/catalog/config.json'))['gates']
perceptual = {'visual', 'accessibility', 'responsive', 'rtl', 'reduced-motion', 'composition', 'surface-discipline', 'interaction', 'examples', 'documentation', 'performance', 'console-clean'}
perceptual_gates = [gate for gate in gates if gate['id'] in perceptual]; bookkeeping = [gate for gate in gates if gate['id'] not in perceptual]
print('total gates:', len(gates))
print('bookkeeping gates:', len(bookkeeping), '-- blocking:', sum(gate.get('blocking', False) for gate in bookkeeping))
print('perceptual gates:', len(perceptual_gates), '-- blocking:', sum(gate.get('blocking', False) for gate in perceptual_gates))
print('\nperceptual gate detail:')
for gate in sorted(perceptual_gates, key=lambda item: item['id']): print('  %-20s blocking=%-5s rung=%s' % (gate['id'], gate.get('blocking'), gate.get('rung')))
print("\nappliesTo of the 'visual' gate:")
for gate in gates:
    if gate['id'] == 'visual': print(json.dumps(gate, indent=2))
