import { journeyResponsivenessBudget } from '../../journeys/support';

describe('MB3 UI journey responsiveness budgets', () => {
  it('defines positive budgets for every measured dimension in J26-J30', () => {
    const journeys = ['J26', 'J27', 'J28', 'J29', 'J30'] as const;
    const metrics = [
      'setup_to_ready_ms',
      'click_to_paint_ms',
      'tab_switch_to_first_frame_ms',
      'event_fanout_ms',
      'event_bytes',
    ] as const;
    for (const journey of journeys) {
      for (const metric of metrics) {
        expect(journeyResponsivenessBudget(journey, metric)).toBeGreaterThan(0);
      }
    }
  });
});
