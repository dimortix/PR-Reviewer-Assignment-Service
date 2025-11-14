package service

import (
	"testing"

	"pr-reviewer-service/internal/models"
)

func TestSelectRandomReviewers(t *testing.T) {
	tests := []struct {
		name       string
		candidates []models.User
		maxCount   int
		wantCount  int
	}{
		{
			name:       "пустой список кандидатов",
			candidates: []models.User{},
			maxCount:   2,
			wantCount:  0,
		},
		{
			name: "кандидатов меньше чем maxCount",
			candidates: []models.User{
				{UserID: "u1", Username: "User1", TeamName: "backend", IsActive: true},
			},
			maxCount:  2,
			wantCount: 1,
		},
		{
			name: "кандидатов больше чем maxCount",
			candidates: []models.User{
				{UserID: "u1", Username: "User1", TeamName: "backend", IsActive: true},
				{UserID: "u2", Username: "User2", TeamName: "backend", IsActive: true},
				{UserID: "u3", Username: "User3", TeamName: "backend", IsActive: true},
				{UserID: "u4", Username: "User4", TeamName: "backend", IsActive: true},
			},
			maxCount:  2,
			wantCount: 2,
		},
		{
			name: "кандидатов ровно maxCount",
			candidates: []models.User{
				{UserID: "u1", Username: "User1", TeamName: "backend", IsActive: true},
				{UserID: "u2", Username: "User2", TeamName: "backend", IsActive: true},
			},
			maxCount:  2,
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selectRandomReviewers(tt.candidates, tt.maxCount)

			if len(result) != tt.wantCount {
				t.Errorf("selectRandomReviewers() returned %d reviewers, want %d", len(result), tt.wantCount)
			}

			// проверяем что все выбранные ревьюверы были в списке кандидатов
			candidateMap := make(map[string]bool)
			for _, c := range tt.candidates {
				candidateMap[c.UserID] = true
			}

			for _, r := range result {
				if !candidateMap[r.UserID] {
					t.Errorf("selectRandomReviewers() returned user %s who was not in candidates", r.UserID)
				}
			}

			// проверяем что нет дубликатов
			seen := make(map[string]bool)
			for _, r := range result {
				if seen[r.UserID] {
					t.Errorf("selectRandomReviewers() returned duplicate user %s", r.UserID)
				}
				seen[r.UserID] = true
			}
		})
	}
}

func TestSelectRandomReviewers_Randomness(t *testing.T) {
	// проверяем что функция действительно перемешивает кандидатов
	candidates := []models.User{
		{UserID: "u1", Username: "User1", TeamName: "backend", IsActive: true},
		{UserID: "u2", Username: "User2", TeamName: "backend", IsActive: true},
		{UserID: "u3", Username: "User3", TeamName: "backend", IsActive: true},
		{UserID: "u4", Username: "User4", TeamName: "backend", IsActive: true},
		{UserID: "u5", Username: "User5", TeamName: "backend", IsActive: true},
	}

	// запускаем несколько раз и проверяем что результаты разные
	results := make(map[string]int)
	iterations := 100

	for i := 0; i < iterations; i++ {
		reviewers := selectRandomReviewers(candidates, 2)
		if len(reviewers) != 2 {
			t.Fatalf("expected 2 reviewers, got %d", len(reviewers))
		}

		// создаем ключ из id выбранных ревьюверов
		key := reviewers[0].UserID + "," + reviewers[1].UserID
		results[key]++
	}

	// если функция действительно рандомная, должно быть больше одной уникальной комбинации
	if len(results) < 2 {
		t.Errorf("selectRandomReviewers() appears to not be random, got only %d unique combinations in %d iterations", len(results), iterations)
	}
}

func TestReassignmentExclusionLogic(t *testing.T) {
	// тест проверяет логику исключения при переназначении ревьювера
	// симулируем фильтрацию кандидатов как в ReassignReviewer

	candidates := []models.User{
		{UserID: "u1", Username: "User1", TeamName: "backend", IsActive: true},
		{UserID: "u2", Username: "User2", TeamName: "backend", IsActive: true},
		{UserID: "u3", Username: "User3", TeamName: "backend", IsActive: true},
		{UserID: "u4", Username: "User4", TeamName: "backend", IsActive: true},
		{UserID: "u5", Username: "User5", TeamName: "backend", IsActive: true},
	}

	oldReviewerID := "u2"
	authorID := "u1"
	currentReviewers := []string{"u2", "u3"}

	// создаем карту исключений
	excludeMap := make(map[string]bool)
	excludeMap[oldReviewerID] = true
	excludeMap[authorID] = true
	for _, reviewerID := range currentReviewers {
		excludeMap[reviewerID] = true
	}

	// фильтруем кандидатов
	filtered := []models.User{}
	for _, candidate := range candidates {
		if !excludeMap[candidate.UserID] {
			filtered = append(filtered, candidate)
		}
	}

	// проверяем что остались только u4 и u5
	if len(filtered) != 2 {
		t.Errorf("expected 2 candidates after exclusion, got %d", len(filtered))
	}

	expectedIDs := map[string]bool{"u4": true, "u5": true}
	for _, candidate := range filtered {
		if !expectedIDs[candidate.UserID] {
			t.Errorf("unexpected candidate %s after exclusion", candidate.UserID)
		}
	}

	// проверяем что исключенные пользователи не попали в список
	for _, candidate := range filtered {
		if candidate.UserID == oldReviewerID {
			t.Error("old reviewer should be excluded")
		}
		if candidate.UserID == authorID {
			t.Error("author should be excluded")
		}
		for _, reviewerID := range currentReviewers {
			if candidate.UserID == reviewerID {
				t.Errorf("current reviewer %s should be excluded", reviewerID)
			}
		}
	}
}
