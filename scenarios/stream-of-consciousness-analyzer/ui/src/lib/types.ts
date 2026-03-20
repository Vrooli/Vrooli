export interface Scheme {
  id: string;
  name: string;
  created_at: string;
  updated_at: string;
}

export interface Information {
  id: string;
  scheme_id: string;
  type: string;
  content: string;
  canvas_x: number;
  canvas_y: number;
  created_at: string;
  updated_at: string;
}

export interface Thought {
  id: string;
  scheme_id: string | null;
  title: string;
  body: string;
  canvas_x: number;
  canvas_y: number;
  created_at: string;
  updated_at: string;
}

export interface ThoughtEdge {
  id: string;
  source_id: string;
  target_id: string;
  label: string;
  created_at: string;
}
