let generation = 0;

export function currentSessionGeneration() {
  return generation;
}

export function advanceSessionGeneration() {
  generation += 1;
  return generation;
}
