export type ListSummary = {
  id: string;
  name: string;
  item_count: number;
  description?: string;
};

export type RankedItem = {
  item: unknown;
  elo_rating: number;
  confidence: number;
};

export type RankingsResponse = {
  rankings: RankedItem[];
};

export type ListDetail = ListSummary & {
  items?: unknown[];
  created_at?: string;
};

export type ComparisonCandidate = {
  id: string;
  content: unknown;
};

export type ComparisonPayload = {
  item_a: ComparisonCandidate;
  item_b: ComparisonCandidate;
  progress?: {
    completed: number;
    total: number;
  };
};

export type ComparisonResult = {
  id: string;
  winner_id: string;
  loser_id: string;
};

export type CreateListPayload = {
  name: string;
  description?: string;
  items: { content: string }[];
};
