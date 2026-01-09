export type MealType = 'breakfast' | 'lunch' | 'dinner' | 'snack';

export interface NutritionGoals {
  calories: number;
  protein: number;
  carbs: number;
  fat: number;
}

export interface NutritionTotals extends NutritionGoals {}

export interface Meal {
  id?: string;
  user_id?: string;
  meal_date?: string;
  food_description?: string;
  name?: string;
  meal_type: MealType;
  calories: number;
  protein?: number;
  carbs?: number;
  fat?: number;
  fiber?: number;
  sugar?: number;
  sodium?: number;
  created_at?: string;
}

export interface Suggestion {
  meal_name: string;
  calories?: number;
  total_calories?: number;
  protein?: number;
  protein_grams?: number;
  recommendation_reason?: string;
}

export interface AnalyzerResponse {
  status: 'success' | 'error';
  total_calories: number;
  protein_grams: number;
  carbs_grams: number;
  fat_grams: number;
}

export interface SuggestionsResponse {
  suggestions: Suggestion[];
}
