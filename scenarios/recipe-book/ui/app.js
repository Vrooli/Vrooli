// Recipe Book Application JavaScript

const API_BASE = '/api/v1';
let currentUser = 'user-' + Math.random().toString(36).substr(2, 9);
let currentView = 'browse';
let recipes = [];
let favorites = [];

// Initialize the app
document.addEventListener('DOMContentLoaded', () => {
    initializeApp();
    setupEventListeners();
    loadRecipes();
});

function initializeApp() {
    // Set initial view
    showView('browse');
    
    // Load user preferences
    loadUserPreferences();
}

function setupEventListeners() {
    // Navigation
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.addEventListener('click', (e) => {
            const view = e.target.dataset.view;
            showView(view);
        });
    });
    
    // Search
    document.getElementById('search-btn').addEventListener('click', performSearch);
    document.getElementById('search-input').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') performSearch();
    });
    
    // Suggestion chips
    document.querySelectorAll('.suggestion-chip').forEach(chip => {
        chip.addEventListener('click', (e) => {
            const query = e.target.textContent.replace(/[^a-zA-Z\s]/g, '').trim();
            document.getElementById('search-input').value = query;
            performSearch();
        });
    });
    
    // Create recipe options
    document.getElementById('manual-create').addEventListener('click', showManualForm);
    document.getElementById('ai-create').addEventListener('click', showAIForm);
    
    // AI ingredients toggle
    document.getElementById('use-ingredients').addEventListener('change', (e) => {
        const ingredientsInput = document.getElementById('specific-ingredients');
        ingredientsInput.classList.toggle('hidden', !e.target.checked);
    });
    
    // Generate recipe
    document.getElementById('generate-recipe').addEventListener('click', generateRecipeWithAI);
    
    // Add ingredient/instruction buttons
    document.querySelector('.add-ingredient-btn')?.addEventListener('click', addIngredientRow);
    document.querySelector('.add-instruction-btn')?.addEventListener('click', addInstructionRow);
    
    // Save recipe
    document.querySelector('.save-recipe-btn')?.addEventListener('click', saveRecipe);
    
    // Modal close
    document.querySelector('.modal-close').addEventListener('click', closeModal);
    
    // Filter
    document.querySelector('.filter-select').addEventListener('change', filterRecipes);
}

function showView(viewName) {
    // Hide all views
    document.querySelectorAll('.view').forEach(view => {
        view.classList.remove('active');
    });
    
    // Show selected view
    document.getElementById(`${viewName}-view`).classList.add('active');
    
    // Update nav buttons
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.classList.remove('active');
        if (btn.dataset.view === viewName) {
            btn.classList.add('active');
        }
    });
    
    currentView = viewName;
    
    // Load view-specific data
    switch(viewName) {
        case 'browse':
            loadRecipes();
            break;
        case 'favorites':
            loadFavorites();
            break;
        case 'meal-plan':
            loadMealPlan();
            break;
    }
}

async function loadRecipes() {
    showLoading();
    
    try {
        const response = await fetch(`${API_BASE}/recipes?limit=20`);
        const data = await response.json();
        recipes = data.recipes || [];
        displayRecipes(recipes, 'recipe-grid');
    } catch (error) {
        showToast('Failed to load recipes');
        console.error('Error loading recipes:', error);
    } finally {
        hideLoading();
    }
}

function displayRecipes(recipeList, containerId) {
    const container = document.getElementById(containerId);
    if (!container) return;
    
    container.innerHTML = '';
    
    if (recipeList.length === 0) {
        container.innerHTML = '<p class="no-recipes">No recipes found. Create your first recipe!</p>';
        return;
    }
    
    recipeList.forEach(recipe => {
        const card = createRecipeCard(recipe);
        container.appendChild(card);
    });
}

function createRecipeCard(recipe) {
    const card = document.createElement('div');
    card.className = 'recipe-card';
    card.onclick = () => showRecipeDetails(recipe);
    
    const emoji = getRecipeEmoji(recipe);
    const rating = recipe.rating || 4.5;
    const totalTime = (recipe.prep_time || 0) + (recipe.cook_time || 0);
    
    card.innerHTML = `
        <div class="recipe-card-image">${emoji}</div>
        ${rating >= 4 ? `<div class="recipe-rating">⭐ ${rating}</div>` : ''}
        <div class="recipe-card-content">
            <h3 class="recipe-card-title">${recipe.title || 'Untitled Recipe'}</h3>
            <p class="recipe-card-description">${recipe.description || 'A delicious recipe'}</p>
            <div class="recipe-card-meta">
                <span class="recipe-meta-item">⏱️ ${totalTime} min</span>
                <span class="recipe-meta-item">🍽️ ${recipe.servings || 4} servings</span>
                ${recipe.cuisine ? `<span class="recipe-meta-item">🌍 ${recipe.cuisine}</span>` : ''}
            </div>
        </div>
    `;
    
    return card;
}

function getRecipeEmoji(recipe) {
    const title = (recipe.title || '').toLowerCase();
    const tags = recipe.tags || [];
    const cuisine = (recipe.cuisine || '').toLowerCase();
    
    // Check for specific keywords
    if (title.includes('cookie') || tags.includes('cookies')) return '🍪';
    if (title.includes('cake') || tags.includes('cake')) return '🍰';
    if (title.includes('soup')) return '🍲';
    if (title.includes('salad')) return '🥗';
    if (title.includes('pasta') || cuisine === 'italian') return '🍝';
    if (title.includes('pizza')) return '🍕';
    if (title.includes('taco') || cuisine === 'mexican') return '🌮';
    if (title.includes('burger')) return '🍔';
    if (title.includes('chicken')) return '🍗';
    if (title.includes('stir-fry') || cuisine === 'asian') return '🥘';
    if (tags.includes('dessert')) return '🍮';
    if (tags.includes('breakfast')) return '🥞';
    
    // Default emojis
    const defaultEmojis = ['🍳', '🥘', '🍲', '🥗', '🍛'];
    return defaultEmojis[Math.floor(Math.random() * defaultEmojis.length)];
}

function showRecipeDetails(recipe) {
    const modal = document.getElementById('recipe-modal');
    const content = document.getElementById('modal-recipe-content');
    
    const totalTime = (recipe.prep_time || 0) + (recipe.cook_time || 0);
    const emoji = getRecipeEmoji(recipe);
    
    content.innerHTML = `
        <div class="recipe-detail">
            <div class="recipe-detail-header">
                <div class="recipe-detail-image">${emoji}</div>
                <div class="recipe-detail-info">
                    <h2 class="recipe-detail-title">${recipe.title}</h2>
                    <p class="recipe-detail-description">${recipe.description || ''}</p>
                    <div class="recipe-detail-meta">
                        <span>⏱️ Prep: ${recipe.prep_time || 0} min</span>
                        <span>🔥 Cook: ${recipe.cook_time || 0} min</span>
                        <span>🍽️ Serves: ${recipe.servings || 4}</span>
                    </div>
                    ${recipe.dietary_info && recipe.dietary_info.length > 0 ? 
                        `<div class="dietary-badges">
                            ${recipe.dietary_info.map(d => `<span class="dietary-badge">${d}</span>`).join('')}
                        </div>` : ''}
                </div>
            </div>
            
            <div class="recipe-detail-content">
                <div class="ingredients-section">
                    <h3>Ingredients</h3>
                    <ul class="ingredients-list">
                        ${(recipe.ingredients || []).map(ing => `
                            <li>${ing.amount} ${ing.unit} ${ing.name} ${ing.notes ? `(${ing.notes})` : ''}</li>
                        `).join('')}
                    </ul>
                </div>
                
                <div class="instructions-section">
                    <h3>Instructions</h3>
                    <ol class="instructions-list">
                        ${(recipe.instructions || []).map(inst => `
                            <li>${inst}</li>
                        `).join('')}
                    </ol>
                </div>
                
                ${recipe.nutrition ? `
                <div class="nutrition-section">
                    <h3>Nutrition (per serving)</h3>
                    <div class="nutrition-grid">
                        <div class="nutrition-item">
                            <span class="nutrition-value">${recipe.nutrition.calories || 0}</span>
                            <span class="nutrition-label">Calories</span>
                        </div>
                        <div class="nutrition-item">
                            <span class="nutrition-value">${recipe.nutrition.protein || 0}g</span>
                            <span class="nutrition-label">Protein</span>
                        </div>
                        <div class="nutrition-item">
                            <span class="nutrition-value">${recipe.nutrition.carbs || 0}g</span>
                            <span class="nutrition-label">Carbs</span>
                        </div>
                        <div class="nutrition-item">
                            <span class="nutrition-value">${recipe.nutrition.fat || 0}g</span>
                            <span class="nutrition-label">Fat</span>
                        </div>
                    </div>
                </div>
                ` : ''}
            </div>
            
            <div class="recipe-detail-actions">
                <button class="action-btn favorite-btn" onclick="toggleFavorite('${recipe.id}')">
                    ${favorites.includes(recipe.id) ? '❤️ Favorited' : '🤍 Add to Favorites'}
                </button>
                <button class="action-btn" onclick="markAsCooked('${recipe.id}')">✅ Mark as Cooked</button>
                <button class="action-btn" onclick="shareRecipe('${recipe.id}')">📤 Share</button>
                <button class="action-btn" onclick="modifyRecipe('${recipe.id}')">✨ Modify</button>
            </div>
        </div>
    `;
    
    modal.classList.remove('hidden');
}

function closeModal() {
    document.getElementById('recipe-modal').classList.add('hidden');
}

async function performSearch() {
    const query = document.getElementById('search-input').value;
    if (!query) return;
    
    showLoading();
    
    try {
        const response = await fetch(`${API_BASE}/recipes/search`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                query: query,
                user_id: currentUser,
                limit: 20
            })
        });
        
        const data = await response.json();
        const results = data.results || [];
        
        displayRecipes(results.map(r => r.recipe), 'search-results');
        showView('search');
    } catch (error) {
        showToast('Search failed. Please try again.');
        console.error('Search error:', error);
    } finally {
        hideLoading();
    }
}

function filterRecipes() {
    const filter = document.querySelector('.filter-select').value;
    let filtered = [...recipes];
    
    switch(filter) {
        case 'Quick & Easy':
            filtered = recipes.filter(r => (r.prep_time + r.cook_time) <= 30);
            break;
        case 'Vegetarian':
            filtered = recipes.filter(r => r.dietary_info?.includes('vegetarian'));
            break;
        case 'Desserts':
            filtered = recipes.filter(r => r.tags?.includes('dessert'));
            break;
        case 'Comfort Food':
            filtered = recipes.filter(r => r.tags?.includes('comfort-food'));
            break;
    }
    
    displayRecipes(filtered, 'recipe-grid');
}

function showManualForm() {
    document.getElementById('recipe-form').classList.remove('hidden');
    document.getElementById('ai-generate-form').classList.add('hidden');
}

function showAIForm() {
    document.getElementById('recipe-form').classList.add('hidden');
    document.getElementById('ai-generate-form').classList.remove('hidden');
}

function addIngredientRow() {
    const list = document.getElementById('ingredients-list');
    const row = document.createElement('div');
    row.className = 'ingredient-row';
    row.innerHTML = `
        <input type="text" class="ingredient-amount form-input" placeholder="Amount">
        <input type="text" class="ingredient-unit form-input" placeholder="Unit">
        <input type="text" class="ingredient-name form-input" placeholder="Ingredient">
        <button class="remove-btn" onclick="this.parentElement.remove()">×</button>
    `;
    list.appendChild(row);
}

function addInstructionRow() {
    const list = document.getElementById('instructions-list');
    const stepNumber = list.children.length + 1;
    const row = document.createElement('div');
    row.className = 'instruction-row';
    row.innerHTML = `
        <span class="step-number">${stepNumber}.</span>
        <textarea class="instruction-text" placeholder="Enter step..."></textarea>
    `;
    list.appendChild(row);
}

async function saveRecipe() {
    const recipe = {
        title: document.getElementById('recipe-title').value,
        description: document.getElementById('recipe-description').value,
        ingredients: collectIngredients(),
        instructions: collectInstructions(),
        prep_time: parseInt(document.getElementById('prep-time').value) || 0,
        cook_time: parseInt(document.getElementById('cook-time').value) || 0,
        servings: parseInt(document.getElementById('servings').value) || 4,
        created_by: currentUser
    };
    
    if (!recipe.title) {
        showToast('Please enter a recipe title');
        return;
    }
    
    showLoading();
    
    try {
        const response = await fetch(`${API_BASE}/recipes`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(recipe)
        });
        
        if (response.ok) {
            showToast('Recipe saved successfully!');
            showView('browse');
            loadRecipes();
        } else {
            showToast('Failed to save recipe');
        }
    } catch (error) {
        showToast('Error saving recipe');
        console.error('Save error:', error);
    } finally {
        hideLoading();
    }
}

function collectIngredients() {
    const ingredients = [];
    document.querySelectorAll('.ingredient-row').forEach(row => {
        const amount = row.querySelector('.ingredient-amount').value;
        const unit = row.querySelector('.ingredient-unit').value;
        const name = row.querySelector('.ingredient-name').value;
        
        if (name) {
            ingredients.push({
                amount: parseFloat(amount) || 0,
                unit: unit,
                name: name
            });
        }
    });
    return ingredients;
}

function collectInstructions() {
    const instructions = [];
    document.querySelectorAll('.instruction-text').forEach(textarea => {
        if (textarea.value) {
            instructions.push(textarea.value);
        }
    });
    return instructions;
}

async function generateRecipeWithAI() {
    const prompt = document.getElementById('ai-prompt').value;
    if (!prompt) {
        showToast('Please describe the recipe you want');
        return;
    }
    
    const useIngredients = document.getElementById('use-ingredients').checked;
    const ingredients = useIngredients ? 
        document.getElementById('specific-ingredients').value.split(',').map(i => i.trim()) : [];
    
    showLoading();
    
    try {
        const response = await fetch(`${API_BASE}/recipes/generate`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                prompt: prompt,
                user_id: currentUser,
                available_ingredients: ingredients
            })
        });
        
        const data = await response.json();
        if (data.recipe) {
            showToast('Recipe generated successfully!');
            showRecipeDetails(data.recipe);
        } else {
            showToast('Failed to generate recipe');
        }
    } catch (error) {
        showToast('Error generating recipe');
        console.error('Generation error:', error);
    } finally {
        hideLoading();
    }
}

function toggleFavorite(recipeId) {
    const index = favorites.indexOf(recipeId);
    if (index > -1) {
        favorites.splice(index, 1);
        showToast('Removed from favorites');
    } else {
        favorites.push(recipeId);
        showToast('Added to favorites');
    }
    
    // Update button if modal is open
    const btn = document.querySelector('.favorite-btn');
    if (btn) {
        btn.innerHTML = favorites.includes(recipeId) ? '❤️ Favorited' : '🤍 Add to Favorites';
    }
    
    saveFavorites();
}

async function markAsCooked(recipeId) {
    const rating = prompt('Rate this recipe (1-5 stars):', '5');
    if (!rating) return;
    
    try {
        await fetch(`${API_BASE}/recipes/${recipeId}/cook`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                user_id: currentUser,
                rating: parseInt(rating),
                notes: ''
            })
        });
        
        showToast('Recipe marked as cooked!');
    } catch (error) {
        showToast('Failed to mark as cooked');
    }
}

function shareRecipe(recipeId) {
    // Simplified sharing - just copy link
    const url = `${window.location.origin}/recipe/${recipeId}`;
    navigator.clipboard.writeText(url);
    showToast('Recipe link copied to clipboard!');
}

async function modifyRecipe(recipeId) {
    const modifications = ['vegan', 'gluten-free', 'keto', 'dairy-free', 'instant-pot'];
    const choice = prompt(`How would you like to modify this recipe?\n${modifications.join(', ')}:`);
    
    if (!choice || !modifications.includes(choice)) return;
    
    showLoading();
    
    try {
        const response = await fetch(`${API_BASE}/recipes/${recipeId}/modify`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                modification_type: choice,
                user_id: currentUser
            })
        });
        
        const data = await response.json();
        if (data.modified_recipe) {
            showToast(`Recipe modified to be ${choice}!`);
            showRecipeDetails(data.modified_recipe);
        }
    } catch (error) {
        showToast('Failed to modify recipe');
    } finally {
        hideLoading();
    }
}

function loadFavorites() {
    const favoriteRecipes = recipes.filter(r => favorites.includes(r.id));
    displayRecipes(favoriteRecipes, 'favorites-grid');
}

function loadMealPlan() {
    // Simplified meal plan view
    const calendar = document.querySelector('.meal-calendar');
    if (!calendar) return;
    
    const days = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday'];
    calendar.innerHTML = days.map(day => `
        <div class="meal-day">
            <h3>${day}</h3>
            <div class="meal-slot" data-day="${day}">
                <button class="add-meal-btn">+ Add Meal</button>
            </div>
        </div>
    `).join('');
}

function loadUserPreferences() {
    // Load from localStorage
    const saved = localStorage.getItem('recipeBookPrefs');
    if (saved) {
        const prefs = JSON.parse(saved);
        favorites = prefs.favorites || [];
    }
}

function saveFavorites() {
    localStorage.setItem('recipeBookPrefs', JSON.stringify({ favorites }));
}

// Utility functions
function showLoading() {
    document.getElementById('loading').classList.remove('hidden');
}

function hideLoading() {
    document.getElementById('loading').classList.add('hidden');
}

function showToast(message) {
    const toast = document.getElementById('toast');
    const toastMessage = document.getElementById('toast-message');
    
    toastMessage.textContent = message;
    toast.classList.remove('hidden');
    
    setTimeout(() => {
        toast.classList.add('hidden');
    }, 3000);
}

// Add recipe detail styles
const styleSheet = document.createElement('style');
styleSheet.textContent = `
    .recipe-detail {
        padding: 2rem;
    }
    
    .recipe-detail-header {
        display: flex;
        gap: 2rem;
        margin-bottom: 2rem;
        padding-bottom: 2rem;
        border-bottom: 2px solid var(--sage-green);
    }
    
    .recipe-detail-image {
        width: 150px;
        height: 150px;
        background: linear-gradient(135deg, var(--sage-green) 0%, var(--olive) 100%);
        border-radius: 1rem;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 5rem;
    }
    
    .recipe-detail-info {
        flex: 1;
    }
    
    .recipe-detail-title {
        font-family: var(--font-handwritten);
        font-size: 2.5rem;
        color: var(--warm-terracotta);
        margin-bottom: 0.5rem;
    }
    
    .recipe-detail-description {
        font-size: 1.1rem;
        color: var(--soft-gray);
        margin-bottom: 1rem;
    }
    
    .recipe-detail-meta {
        display: flex;
        gap: 2rem;
        font-family: var(--font-sans);
        color: var(--olive);
        margin-bottom: 1rem;
    }
    
    .dietary-badges {
        display: flex;
        gap: 0.5rem;
    }
    
    .dietary-badge {
        padding: 0.25rem 0.75rem;
        background: var(--sage-green);
        color: white;
        border-radius: 2rem;
        font-family: var(--font-sans);
        font-size: 0.85rem;
    }
    
    .recipe-detail-content {
        display: grid;
        grid-template-columns: 1fr 1fr;
        gap: 2rem;
        margin-bottom: 2rem;
    }
    
    .ingredients-section h3,
    .instructions-section h3,
    .nutrition-section h3 {
        font-family: var(--font-handwritten);
        font-size: 1.75rem;
        color: var(--warm-terracotta);
        margin-bottom: 1rem;
    }
    
    .ingredients-list {
        list-style: none;
        padding: 0;
    }
    
    .ingredients-list li {
        padding: 0.5rem 0;
        border-bottom: 1px solid #eee;
    }
    
    .instructions-list {
        padding-left: 1.5rem;
    }
    
    .instructions-list li {
        margin-bottom: 1rem;
        line-height: 1.6;
    }
    
    .nutrition-section {
        grid-column: 1 / -1;
    }
    
    .nutrition-grid {
        display: flex;
        gap: 2rem;
        justify-content: center;
    }
    
    .nutrition-item {
        text-align: center;
    }
    
    .nutrition-value {
        display: block;
        font-size: 1.5rem;
        font-weight: bold;
        color: var(--warm-terracotta);
    }
    
    .nutrition-label {
        font-family: var(--font-sans);
        font-size: 0.9rem;
        color: var(--soft-gray);
    }
    
    .recipe-detail-actions {
        display: flex;
        gap: 1rem;
        padding-top: 2rem;
        border-top: 2px solid var(--sage-green);
    }
    
    .action-btn {
        padding: 0.75rem 1.5rem;
        background: white;
        border: 2px solid var(--sage-green);
        border-radius: 2rem;
        font-family: var(--font-sans);
        font-weight: 600;
        cursor: pointer;
        transition: all 0.3s ease;
    }
    
    .action-btn:hover {
        background: var(--sage-green);
        color: white;
        transform: translateY(-2px);
    }
    
    .favorite-btn {
        background: var(--warm-terracotta);
        color: white;
        border-color: var(--warm-terracotta);
    }
    
    .favorite-btn:hover {
        background: var(--deep-burgundy);
        border-color: var(--deep-burgundy);
    }
    
    .meal-calendar {
        display: grid;
        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
        gap: 1rem;
        margin-top: 2rem;
    }
    
    .meal-day {
        background: white;
        padding: 1.5rem;
        border-radius: 0.75rem;
        box-shadow: var(--card-shadow);
    }
    
    .meal-day h3 {
        font-family: var(--font-handwritten);
        font-size: 1.5rem;
        color: var(--warm-terracotta);
        margin-bottom: 1rem;
    }
    
    .meal-slot {
        min-height: 100px;
        border: 2px dashed var(--sage-green);
        border-radius: 0.5rem;
        display: flex;
        align-items: center;
        justify-content: center;
    }
    
    .add-meal-btn {
        padding: 0.5rem 1rem;
        background: transparent;
        border: none;
        color: var(--sage-green);
        font-family: var(--font-sans);
        font-weight: 600;
        cursor: pointer;
    }
    
    .no-recipes {
        text-align: center;
        padding: 3rem;
        font-family: var(--font-handwritten);
        font-size: 1.5rem;
        color: var(--soft-gray);
    }
`;
document.head.appendChild(styleSheet);