export function RawHeavy() {
  return <section>{Array.from({ length: 40 }, (_, index) => <div key={index}>raw {index}</div>)}</section>;
}
