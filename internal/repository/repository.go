package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pr-reviewer-service/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) CreateTeam(ctx context.Context, teamName string, members []models.TeamMember) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// проверяем что команды не существует
	var exists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)", teamName).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("%s: team_name already exists", models.ErrCodeTeamExists)
	}

	// создаем команду
	_, err = tx.Exec(ctx, "INSERT INTO teams (team_name) VALUES ($1)", teamName)
	if err != nil {
		return err
	}

	// добавляем участников
	for _, member := range members {
		_, err = tx.Exec(ctx, `
			INSERT INTO users (user_id, username, team_name, is_active)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) DO UPDATE
			SET username = EXCLUDED.username,
			    team_name = EXCLUDED.team_name,
			    is_active = EXCLUDED.is_active,
			    updated_at = CURRENT_TIMESTAMP
		`, member.UserID, member.Username, teamName, member.IsActive)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) UpdateTeam(ctx context.Context, teamName string, members []models.TeamMember) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// проверяем что команда существует
	var exists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)", teamName).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%s: team not found", models.ErrCodeNotFound)
	}

	// обновляем участников команды
	for _, member := range members {
		_, err = tx.Exec(ctx, `
			INSERT INTO users (user_id, username, team_name, is_active)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (user_id) DO UPDATE
			SET username = EXCLUDED.username,
			    team_name = EXCLUDED.team_name,
			    is_active = EXCLUDED.is_active,
			    updated_at = CURRENT_TIMESTAMP
		`, member.UserID, member.Username, teamName, member.IsActive)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetTeam(ctx context.Context, teamName string) (*models.Team, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)", teamName).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	// получаем список участников команды
	rows, err := r.pool.Query(ctx, `
		SELECT user_id, username, is_active
		FROM users
		WHERE team_name = $1
		ORDER BY user_id
	`, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []models.TeamMember{}
	for rows.Next() {
		var member models.TeamMember
		if err := rows.Scan(&member.UserID, &member.Username, &member.IsActive); err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return &models.Team{
		TeamName: teamName,
		Members:  members,
	}, nil
}

func (r *Repository) GetUser(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE user_id = $1
	`, userID).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (r *Repository) SetUserActive(ctx context.Context, userID string, isActive bool) (*models.User, error) {
	var user models.User
	err := r.pool.QueryRow(ctx, `
		UPDATE users
		SET is_active = $1, updated_at = CURRENT_TIMESTAMP
		WHERE user_id = $2
		RETURNING user_id, username, team_name, is_active
	`, isActive, userID).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (r *Repository) GetActiveTeamMembers(ctx context.Context, teamName string, excludeUserID string) ([]models.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE team_name = $1 AND is_active = true AND user_id != $2
	`, teamName, excludeUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []models.User{}
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (r *Repository) CreatePR(ctx context.Context, pr *models.PullRequest, reviewers []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// проверяем что PR с таким id еще не существует
	var exists bool
	err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id = $1)", pr.PullRequestID).Scan(&exists)
	if err != nil {
		return err
	}
	if exists {
		return ErrAlreadyExists
	}

	now := time.Now()
	_, err = tx.Exec(ctx, `
		INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, pr.PullRequestID, pr.PullRequestName, pr.AuthorID, models.StatusOpen, now)
	if err != nil {
		return err
	}

	// назначаем ревьюверов
	for _, reviewerID := range reviewers {
		_, err = tx.Exec(ctx, `
			INSERT INTO pull_request_reviewers (pull_request_id, user_id)
			VALUES ($1, $2)
		`, pr.PullRequestID, reviewerID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetPR(ctx context.Context, prID string) (*models.PullRequest, error) {
	var pr models.PullRequest
	var createdAt, mergedAt *time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests
		WHERE pull_request_id = $1
	`, prID).Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &createdAt, &mergedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	pr.CreatedAt = createdAt
	pr.MergedAt = mergedAt

	// Get reviewers
	rows, err := r.pool.Query(ctx, `
		SELECT user_id FROM pull_request_reviewers WHERE pull_request_id = $1
	`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviewers := []string{}
	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, err
		}
		reviewers = append(reviewers, reviewerID)
	}
	pr.AssignedReviewers = reviewers

	return &pr, nil
}

func (r *Repository) MergePR(ctx context.Context, prID string) (*models.PullRequest, error) {
	now := time.Now()
	var pr models.PullRequest
	var createdAt, mergedAt *time.Time

	err := r.pool.QueryRow(ctx, `
		UPDATE pull_requests
		SET status = $1, merged_at = $2
		WHERE pull_request_id = $3
		RETURNING pull_request_id, pull_request_name, author_id, status, created_at, merged_at
	`, models.StatusMerged, now, prID).Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &createdAt, &mergedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	pr.CreatedAt = createdAt
	pr.MergedAt = mergedAt

	// Get reviewers
	rows, err := r.pool.Query(ctx, `
		SELECT user_id FROM pull_request_reviewers WHERE pull_request_id = $1
	`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviewers := []string{}
	for rows.Next() {
		var reviewerID string
		if err := rows.Scan(&reviewerID); err != nil {
			return nil, err
		}
		reviewers = append(reviewers, reviewerID)
	}
	pr.AssignedReviewers = reviewers

	return &pr, nil
}

func (r *Repository) IsReviewerAssigned(ctx context.Context, prID, userID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM pull_request_reviewers
			WHERE pull_request_id = $1 AND user_id = $2
		)
	`, prID, userID).Scan(&exists)
	return exists, err
}

func (r *Repository) ReassignReviewer(ctx context.Context, prID, oldUserID, newUserID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Remove old reviewer
	result, err := tx.Exec(ctx, `
		DELETE FROM pull_request_reviewers
		WHERE pull_request_id = $1 AND user_id = $2
	`, prID, oldUserID)
	if err != nil {
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("reviewer not found in PR")
	}

	// Add new reviewer
	_, err = tx.Exec(ctx, `
		INSERT INTO pull_request_reviewers (pull_request_id, user_id)
		VALUES ($1, $2)
	`, prID, newUserID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetPRsByReviewer(ctx context.Context, userID string) ([]models.PullRequestShort, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		INNER JOIN pull_request_reviewers prr ON pr.pull_request_id = prr.pull_request_id
		WHERE prr.user_id = $1
		ORDER BY pr.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prs := []models.PullRequestShort{}
	for rows.Next() {
		var pr models.PullRequestShort
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status); err != nil {
			return nil, err
		}
		prs = append(prs, pr)
	}

	return prs, nil
}

// получение статистики по пользователям
func (r *Repository) GetUserStats(ctx context.Context) ([]models.UserStats, error) {
	query := `
		SELECT
			u.user_id,
			u.username,
			u.team_name,
			u.is_active,
			COUNT(DISTINCT pr_authored.pull_request_id) as total_prs_authored,
			COUNT(DISTINCT prr.pull_request_id) as total_reviews,
			COUNT(DISTINCT CASE WHEN pr_review.status = 'OPEN' THEN prr.pull_request_id END) as active_reviews
		FROM users u
		LEFT JOIN pull_requests pr_authored ON u.user_id = pr_authored.author_id
		LEFT JOIN pull_request_reviewers prr ON u.user_id = prr.user_id
		LEFT JOIN pull_requests pr_review ON prr.pull_request_id = pr_review.pull_request_id
		GROUP BY u.user_id, u.username, u.team_name, u.is_active
		ORDER BY total_reviews DESC, total_prs_authored DESC
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := []models.UserStats{}
	for rows.Next() {
		var s models.UserStats
		if err := rows.Scan(&s.UserID, &s.Username, &s.TeamName, &s.IsActive, &s.TotalPRsAuthored, &s.TotalReviews, &s.ActiveReviews); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}

	return stats, nil
}
