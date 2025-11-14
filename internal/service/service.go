package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"

	"pr-reviewer-service/internal/logger"
	"pr-reviewer-service/internal/models"
	"pr-reviewer-service/internal/repository"
)

type Service struct {
	repo   *repository.Repository
	logger *logger.Logger
}

func New(repo *repository.Repository, log *logger.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: log,
	}
}

func (s *Service) CreateTeam(ctx context.Context, req models.CreateTeamRequest) (*models.Team, error) {
	if err := s.repo.CreateTeam(ctx, req.TeamName, req.Members); err != nil {
		return nil, err
	}

	return s.repo.GetTeam(ctx, req.TeamName)
}

func (s *Service) UpdateTeam(ctx context.Context, req models.CreateTeamRequest) (*models.Team, error) {
	if err := s.repo.UpdateTeam(ctx, req.TeamName, req.Members); err != nil {
		return nil, err
	}

	return s.repo.GetTeam(ctx, req.TeamName)
}

func (s *Service) GetTeam(ctx context.Context, teamName string) (*models.Team, error) {
	team, err := s.repo.GetTeam(ctx, teamName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("%s: %w", models.ErrCodeNotFound, err)
		}
		return nil, err
	}
	return team, nil
}

func (s *Service) SetUserActive(ctx context.Context, req models.SetIsActiveRequest) (*models.User, error) {
	user, err := s.repo.SetUserActive(ctx, req.UserID, req.IsActive)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("%s: %w", models.ErrCodeNotFound, err)
		}
		return nil, err
	}
	return user, nil
}

func (s *Service) GetUserReviews(ctx context.Context, userID string) ([]models.PullRequestShort, error) {
	prs, err := s.repo.GetPRsByReviewer(ctx, userID)
	if err != nil {
		return nil, err
	}
	return prs, nil
}

func (s *Service) CreatePR(ctx context.Context, req models.CreatePRRequest) (*models.PullRequest, error) {
	s.logger.Debug("Creating PR: %s by author: %s", req.PullRequestID, req.AuthorID)

	// получаем автора чтобы узнать его команду
	author, err := s.repo.GetUser(ctx, req.AuthorID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.Warn("Author not found: %s", req.AuthorID)
			return nil, fmt.Errorf("%s: author not found: %w", models.ErrCodeNotFound, err)
		}
		return nil, err
	}

	// получаем активных участников команды кроме автора
	candidates, err := s.repo.GetActiveTeamMembers(ctx, author.TeamName, author.UserID)
	if err != nil {
		return nil, err
	}

	// выбираем до 2 случайных ревьюверов
	reviewers := selectRandomReviewers(candidates, 2)
	reviewerIDs := make([]string, len(reviewers))
	for i, r := range reviewers {
		reviewerIDs[i] = r.UserID
	}

	s.logger.Info("Assigned %d reviewers to PR %s: %v", len(reviewerIDs), req.PullRequestID, reviewerIDs)

	pr := &models.PullRequest{
		PullRequestID:     req.PullRequestID,
		PullRequestName:   req.PullRequestName,
		AuthorID:          req.AuthorID,
		Status:            models.StatusOpen,
		AssignedReviewers: reviewerIDs,
	}

	if err := s.repo.CreatePR(ctx, pr, reviewerIDs); err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return nil, fmt.Errorf("%s: %w", models.ErrCodePRExists, err)
		}
		return nil, err
	}

	return s.repo.GetPR(ctx, req.PullRequestID)
}

func (s *Service) MergePR(ctx context.Context, req models.MergePRRequest) (*models.PullRequest, error) {
	// проверяем существует ли PR и его текущий статус
	pr, err := s.repo.GetPR(ctx, req.PullRequestID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, fmt.Errorf("%s: %w", models.ErrCodeNotFound, err)
		}
		return nil, err
	}

	// если уже merged, возвращаем текущее состояние (идемпотентность)
	if pr.Status == models.StatusMerged {
		return pr, nil
	}

	return s.repo.MergePR(ctx, req.PullRequestID)
}

func (s *Service) ReassignReviewer(ctx context.Context, req models.ReassignRequest) (*models.PullRequest, string, error) {
	pr, err := s.repo.GetPR(ctx, req.PullRequestID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, "", fmt.Errorf("%s: PR not found: %w", models.ErrCodeNotFound, err)
		}
		return nil, "", err
	}

	// нельзя переназначать на merged PR
	if pr.Status == models.StatusMerged {
		return nil, "", fmt.Errorf("%s: cannot reassign on merged PR", models.ErrCodePRMerged)
	}

	// проверяем что пользователь действительно назначен ревьювером
	isAssigned, err := s.repo.IsReviewerAssigned(ctx, req.PullRequestID, req.OldUserID)
	if err != nil {
		return nil, "", err
	}
	if !isAssigned {
		return nil, "", fmt.Errorf("%s: reviewer is not assigned to this PR", models.ErrCodeNotAssigned)
	}

	// получаем старого ревьювера чтобы узнать его команду
	oldReviewer, err := s.repo.GetUser(ctx, req.OldUserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, "", fmt.Errorf("%s: old reviewer not found: %w", models.ErrCodeNotFound, err)
		}
		return nil, "", err
	}

	// берем активных участников из команды старого ревьювера
	candidates, err := s.repo.GetActiveTeamMembers(ctx, oldReviewer.TeamName, "")
	if err != nil {
		return nil, "", err
	}

	// исключаем: старого ревьювера, текущих ревьюверов PR, автора PR
	excludeMap := make(map[string]bool)
	excludeMap[req.OldUserID] = true
	excludeMap[pr.AuthorID] = true
	for _, reviewerID := range pr.AssignedReviewers {
		excludeMap[reviewerID] = true
	}

	filtered := []models.User{}
	for _, candidate := range candidates {
		if !excludeMap[candidate.UserID] {
			filtered = append(filtered, candidate)
		}
	}

	if len(filtered) == 0 {
		return nil, "", fmt.Errorf("%s: no active replacement candidate in team", models.ErrCodeNoCandidate)
	}

	// выбираем случайного кандидата
	newReviewer := filtered[rand.Intn(len(filtered))]

	if err := s.repo.ReassignReviewer(ctx, req.PullRequestID, req.OldUserID, newReviewer.UserID); err != nil {
		return nil, "", err
	}

	updatedPR, err := s.repo.GetPR(ctx, req.PullRequestID)
	if err != nil {
		return nil, "", err
	}

	return updatedPR, newReviewer.UserID, nil
}

func (s *Service) HealthCheck(ctx context.Context) error {
	// простая проверка - пытаемся выполнить запрос к базе
	_, _ = s.repo.GetActiveTeamMembers(ctx, "__healthcheck__", "__healthcheck__")
	return nil
}

func (s *Service) GetStats(ctx context.Context) ([]models.UserStats, error) {
	return s.repo.GetUserStats(ctx)
}

func selectRandomReviewers(candidates []models.User, maxCount int) []models.User {
	if len(candidates) == 0 {
		return []models.User{}
	}

	count := maxCount
	if len(candidates) < count {
		count = len(candidates)
	}

	// перемешиваем кандидатов и берем первые count
	shuffled := make([]models.User, len(candidates))
	copy(shuffled, candidates)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled[:count]
}
