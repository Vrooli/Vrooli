interface InstructionPanelProps {
  onStart: () => void;
  disabled?: boolean;
}

export const InstructionPanel = ({ onStart, disabled }: InstructionPanelProps) => {
  return (
    <section className="instruction panel">
      <header>
        <p className="instruction__eyebrow">Welcome to the ranking studio</p>
        <h2>Design your priority stack.</h2>
        <p className="instruction__lede">
          Choose a list, then make confident calls by picking the stronger option in each head-to-head.
          Elo handles the math and keeps the order adaptive.
        </p>
      </header>
      <div className="instruction__steps grid-3">
        <div className="instruction__step">
          <span className="instruction__step-index">1</span>
          <div>
            <h3>Select a list</h3>
            <p>Import product ideas, creative drafts, or backlog items that need a real-time power ranking.</p>
          </div>
        </div>
        <div className="instruction__step">
          <span className="instruction__step-index">2</span>
          <div>
            <h3>Make fast calls</h3>
            <p>Use click, tap, arrow keys, A/D, or swipe gestures to crown a winner. Skip ties instantly.</p>
          </div>
        </div>
        <div className="instruction__step">
          <span className="instruction__step-index">3</span>
          <div>
            <h3>Stop at confidence</h3>
            <p>Watch signal quality rise and wrap once the ranking feels locked.</p>
          </div>
        </div>
      </div>
      <button type="button" className="btn btn-primary instruction__cta" onClick={onStart} disabled={disabled}>
        Start comparing
      </button>
    </section>
  );
};
