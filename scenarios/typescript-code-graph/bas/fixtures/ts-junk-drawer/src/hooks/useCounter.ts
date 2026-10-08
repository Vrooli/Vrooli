export function useCounter(initial: number) {
  let count = initial;
  return {
    get: () => count,
    inc: () => {
      count += 1;
    },
  };
}
